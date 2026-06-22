package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	appconfig "github.com/qm-secrets/qm-secrets/internal/config"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (default: QM_SECRETS_CONFIG, config.yaml, qm-secrets.yaml)")
	flag.Parse()

	if err := run(*configPath); err != nil {
		log.Fatal(err)
	}
}

func run(configPath string) error {
	cfg, err := appconfig.LoadBootstrap(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWSRegion))
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsCfg)

	exists, status, err := tableStatus(ctx, client, cfg.DynamoDBTable)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("table %q already exists (status=%s)", cfg.DynamoDBTable, status)
		if status == types.TableStatusActive {
			return nil
		}
		return waitForActive(ctx, client, cfg.DynamoDBTable)
	}

	log.Printf("creating table %q in %s", cfg.DynamoDBTable, cfg.AWSRegion)
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String(cfg.DynamoDBTable),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("name"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("name"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})
	if err != nil {
		var inUse *types.ResourceInUseException
		if errors.As(err, &inUse) {
			log.Printf("table %q is already being created", cfg.DynamoDBTable)
		} else {
			return fmt.Errorf("create table: %w", err)
		}
	}

	if err := waitForActive(ctx, client, cfg.DynamoDBTable); err != nil {
		return err
	}

	log.Printf("table %q is ready", cfg.DynamoDBTable)
	return nil
}

func tableStatus(ctx context.Context, client *dynamodb.Client, tableName string) (exists bool, status types.TableStatus, err error) {
	out, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("describe table: %w", err)
	}
	return true, out.Table.TableStatus, nil
}

func waitForActive(ctx context.Context, client *dynamodb.Client, tableName string) error {
	waiter := dynamodb.NewTableExistsWaiter(client)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, 2*time.Minute); err != nil {
		return fmt.Errorf("wait for table: %w", err)
	}
	return nil
}
