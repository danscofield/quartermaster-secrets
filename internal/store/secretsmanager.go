package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/qm-secrets/qm-secrets/internal/model"
)

type SecretsManagerStore struct {
	client *secretsmanager.Client
	prefix string
}

func NewSecretsManagerStore(client *secretsmanager.Client, prefix string) *SecretsManagerStore {
	return &SecretsManagerStore{client: client, prefix: prefix}
}

func (s *SecretsManagerStore) asmName(secretName string) string {
	return fmt.Sprintf("%s/%s", s.prefix, secretName)
}

func (s *SecretsManagerStore) Create(ctx context.Context, secretName string, values model.SecretValues) (string, error) {
	payload, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal values: %w", err)
	}

	name := s.asmName(secretName)
	out, err := s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(string(payload)),
	})
	if err != nil {
		return "", fmt.Errorf("create secret: %w", err)
	}
	return aws.ToString(out.ARN), nil
}

func (s *SecretsManagerStore) Get(ctx context.Context, asmSecretARN string) (model.SecretValues, error) {
	out, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(asmSecretARN),
	})
	if err != nil {
		return model.SecretValues{}, fmt.Errorf("get secret value: %w", err)
	}

	var values model.SecretValues
	if err := json.Unmarshal([]byte(aws.ToString(out.SecretString)), &values); err != nil {
		return model.SecretValues{}, fmt.Errorf("unmarshal values: %w", err)
	}
	return values, nil
}

func (s *SecretsManagerStore) Update(ctx context.Context, asmSecretARN string, values model.SecretValues) error {
	payload, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}

	_, err = s.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(asmSecretARN),
		SecretString: aws.String(string(payload)),
	})
	if err != nil {
		return fmt.Errorf("put secret value: %w", err)
	}
	return nil
}

func (s *SecretsManagerStore) Delete(ctx context.Context, asmSecretARN string) error {
	_, err := s.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(asmSecretARN),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}
