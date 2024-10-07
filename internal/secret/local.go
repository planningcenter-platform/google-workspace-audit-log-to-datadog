package secret

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/log"
)

var localSecrets map[string]string

func init() {
	if isLocal() {
		readSecretsFile(context.Background())
	}
}

func isLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}

func readSecretsFile(ctx context.Context) {
	byte, err := os.ReadFile("secrets.json")
	if err != nil {
		log.Error(ctx, "failed to read secrets.json", err)
	}

	err = json.Unmarshal(byte, &localSecrets)
	if err != nil {
		log.Error(ctx, "failed to unmarshal secrets.json", err)
	}
}

func GetLocalSecretValue(ctx context.Context, secretArn string) (*string, error) {
	log.Debug(ctx, "GetLocalSecretValue", "secretArn", secretArn)

	secretValue, ok := localSecrets[secretArn]

	if !ok {
		return nil, errors.New("Secret not found")
	}

	return &secretValue, nil
}
