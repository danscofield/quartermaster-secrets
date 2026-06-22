package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qm-secrets/qm-secrets/internal/auth"
	"github.com/qm-secrets/qm-secrets/internal/model"
	"github.com/qm-secrets/qm-secrets/internal/store"
)

type SecretService struct {
	meta   *store.DynamoDBStore
	values *store.SecretsManagerStore
}

func NewSecretService(meta *store.DynamoDBStore, values *store.SecretsManagerStore) *SecretService {
	return &SecretService{meta: meta, values: values}
}

func (s *SecretService) Create(ctx context.Context, callerBillets []string, req model.CreateSecretRequest) (*model.SecretSummary, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Owners) == 0 {
		return nil, fmt.Errorf("at least one owner billet is required")
	}
	if !auth.ValidateOwnerIntersection(callerBillets, req.Owners) {
		return nil, ErrForbidden
	}

	now := time.Now().UTC()
	secretValues := model.SecretValues{Value1: req.Value1, Value2: req.Value2}
	arn, err := s.values.Create(ctx, req.Name, secretValues)
	if err != nil {
		return nil, fmt.Errorf("create asm secret: %w", err)
	}

	meta := model.SecretMetadata{
		Name:         req.Name,
		Owners:       req.Owners,
		Retrievers:   req.Retrievers,
		ASMSecretARN: arn,
		LastUpdated:  now,
	}
	if err := s.meta.Put(ctx, meta); err != nil {
		_ = s.values.Delete(ctx, arn)
		return nil, fmt.Errorf("store metadata: %w", err)
	}

	return summaryFromMeta(meta, callerBillets), nil
}

func (s *SecretService) List(ctx context.Context, callerBillets []string) ([]model.SecretSummary, error) {
	all, err := s.meta.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []model.SecretSummary
	for _, meta := range all {
		if auth.CanRead(callerBillets, meta.Owners, meta.Retrievers) {
			result = append(result, *summaryFromMeta(meta, callerBillets))
		}
	}
	if result == nil {
		result = []model.SecretSummary{}
	}
	return result, nil
}

func (s *SecretService) Get(ctx context.Context, callerBillets []string, name string) (*model.Secret, error) {
	meta, err := s.meta.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if !auth.CanRead(callerBillets, meta.Owners, meta.Retrievers) {
		return nil, ErrForbidden
	}

	values, err := s.values.Get(ctx, meta.ASMSecretARN)
	if err != nil {
		return nil, fmt.Errorf("fetch values: %w", err)
	}

	return &model.Secret{
		Name:        meta.Name,
		Owners:      meta.Owners,
		Retrievers:  meta.Retrievers,
		Value1:      values.Value1,
		Value2:      values.Value2,
		LastUpdated: meta.LastUpdated,
	}, nil
}

func (s *SecretService) Update(ctx context.Context, callerBillets []string, name string, req model.UpdateSecretRequest) (*model.SecretSummary, error) {
	meta, err := s.meta.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	metadataChanged := req.Owners != nil || req.Retrievers != nil
	valuesChanged := req.Value1 != nil || req.Value2 != nil

	if metadataChanged {
		newOwners := meta.Owners
		if req.Owners != nil {
			newOwners = req.Owners
		}
		newRetrievers := meta.Retrievers
		if req.Retrievers != nil {
			newRetrievers = req.Retrievers
		}
		if len(newOwners) == 0 {
			return nil, fmt.Errorf("at least one owner billet is required")
		}
		if !auth.ValidateOwnerIntersection(callerBillets, newOwners) {
			return nil, ErrForbidden
		}
		if !auth.CanUpdate(callerBillets, meta.Owners) {
			return nil, ErrForbidden
		}
		meta.Owners = newOwners
		meta.Retrievers = newRetrievers
		if err := s.meta.UpdateMetadata(ctx, name, meta.Owners, meta.Retrievers); err != nil {
			return nil, err
		}
	}

	if valuesChanged {
		if !auth.CanUpdate(callerBillets, meta.Owners) {
			return nil, ErrForbidden
		}

		current, err := s.values.Get(ctx, meta.ASMSecretARN)
		if err != nil {
			return nil, fmt.Errorf("fetch current values: %w", err)
		}
		if req.Value1 != nil {
			current.Value1 = *req.Value1
		}
		if req.Value2 != nil {
			current.Value2 = *req.Value2
		}
		if err := s.values.Update(ctx, meta.ASMSecretARN, current); err != nil {
			return nil, fmt.Errorf("update values: %w", err)
		}

		now := time.Now().UTC()
		if err := s.meta.TouchLastUpdated(ctx, name, now); err != nil {
			return nil, err
		}
		meta.LastUpdated = now
	}

	// Re-fetch metadata if only values changed to get fresh last_updated
	if valuesChanged && !metadataChanged {
		meta, err = s.meta.Get(ctx, name)
		if err != nil {
			return nil, err
		}
	} else if metadataChanged && !valuesChanged {
		meta, err = s.meta.Get(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	return summaryFromMeta(*meta, callerBillets), nil
}

func (s *SecretService) Delete(ctx context.Context, callerBillets []string, name string) error {
	meta, err := s.meta.Get(ctx, name)
	if err != nil {
		return err
	}
	if !auth.CanUpdate(callerBillets, meta.Owners) {
		return ErrForbidden
	}

	if err := s.values.Delete(ctx, meta.ASMSecretARN); err != nil {
		return fmt.Errorf("delete asm secret: %w", err)
	}
	return s.meta.Delete(ctx, name)
}

func (s *SecretService) Poll(ctx context.Context, callerBillets []string, req model.PollRequest) (*model.PollResponse, error) {
	var updated []model.SecretSummary

	for _, entry := range req.Secrets {
		meta, err := s.meta.Get(ctx, entry.Name)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if !auth.CanRead(callerBillets, meta.Owners, meta.Retrievers) {
			continue
		}
		if meta.LastUpdated.After(entry.LastUpdated) {
			updated = append(updated, *summaryFromMeta(*meta, callerBillets))
		}
	}

	if updated == nil {
		updated = []model.SecretSummary{}
	}
	return &model.PollResponse{Updated: updated}, nil
}

func summaryFromMeta(meta model.SecretMetadata, callerBillets []string) *model.SecretSummary {
	return &model.SecretSummary{
		Name:        meta.Name,
		Owners:      meta.Owners,
		Retrievers:  meta.Retrievers,
		LastUpdated: meta.LastUpdated,
		CanUpdate:   auth.CanUpdate(callerBillets, meta.Owners),
	}
}

var ErrForbidden = errors.New("forbidden")
