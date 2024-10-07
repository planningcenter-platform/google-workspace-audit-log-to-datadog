package secret

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/log"
)

var client *secretsmanager.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Error(context.Background(), "Failed to load configuration", "error", err)
		panic("Can't continue without AWS config")
	}
	client = secretsmanager.NewFromConfig(cfg)
}

func GetSecretValue(ctx context.Context, secretArn string) (*string, error) {
	if isLocal() {
		return GetLocalSecretValue(ctx, secretArn)
	}

	output, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretArn),
	})

	if err != nil {
		log.Error(ctx, "Error getting Secret", "secretArn", secretArn, err)
		return nil, err
	}

	return output.SecretString, nil
}
