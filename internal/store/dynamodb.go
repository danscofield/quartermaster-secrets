package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/qm-secrets/qm-secrets/internal/model"
)

type DynamoDBStore struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBStore(client *dynamodb.Client, tableName string) *DynamoDBStore {
	return &DynamoDBStore{client: client, tableName: tableName}
}

func (s *DynamoDBStore) Put(ctx context.Context, meta model.SecretMetadata) error {
	item, err := attributevalue.MarshalMap(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) Get(ctx context.Context, name string) (*model.SecretMetadata, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"name": &types.AttributeValueMemberS{Value: name},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}

	var meta model.SecretMetadata
	if err := attributevalue.UnmarshalMap(out.Item, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return &meta, nil
}

func (s *DynamoDBStore) Delete(ctx context.Context, name string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"name": &types.AttributeValueMemberS{Value: name},
		},
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) ListAll(ctx context.Context) ([]model.SecretMetadata, error) {
	var items []model.SecretMetadata
	var lastKey map[string]types.AttributeValue

	for {
		out, err := s.client.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(s.tableName),
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		var page []model.SecretMetadata
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
			return nil, fmt.Errorf("unmarshal scan: %w", err)
		}
		items = append(items, page...)

		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	return items, nil
}

func (s *DynamoDBStore) UpdateMetadata(ctx context.Context, name string, owners, retrievers []string) error {
	ownersAV, err := attributevalue.Marshal(owners)
	if err != nil {
		return fmt.Errorf("marshal owners: %w", err)
	}
	retrieversAV, err := attributevalue.Marshal(retrievers)
	if err != nil {
		return fmt.Errorf("marshal retrievers: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"name": &types.AttributeValueMemberS{Value: name},
		},
		UpdateExpression: aws.String("SET owners = :owners, retrievers = :retrievers"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":owners":     ownersAV,
			":retrievers": retrieversAV,
		},
		ConditionExpression: aws.String("attribute_exists(#n)"),
		ExpressionAttributeNames: map[string]string{
			"#n": "name",
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("update metadata: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) TouchLastUpdated(ctx context.Context, name string, t time.Time) error {
	tsAV, err := attributevalue.Marshal(t.UTC())
	if err != nil {
		return fmt.Errorf("marshal timestamp: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"name": &types.AttributeValueMemberS{Value: name},
		},
		UpdateExpression: aws.String("SET last_updated = :ts"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ts": tsAV,
		},
		ConditionExpression: aws.String("attribute_exists(#n)"),
		ExpressionAttributeNames: map[string]string{
			"#n": "name",
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrNotFound
		}
		return fmt.Errorf("touch last_updated: %w", err)
	}
	return nil
}

func isConditionalCheckFailed(err error) bool {
	var condErr *types.ConditionalCheckFailedException
	return errors.As(err, &condErr)
}
