package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/log"
	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/secret"
)

var (
	googleCredentialsSecretArn string
	googleAdminUser            string
	googleClient               GoogleAdminClient
	s3Bucket                   string
	s3Client                   *s3.Client
)

type ListActivityEvent struct {
	ApplicationName   string `json:"application_name"`
	TimeWindowMinutes int16  `json:"time_window_minutes"`
}

func init() {
	ctx := context.Background()
	googleCredentialsSecretArn = os.Getenv("GOOGLE_CREDENTIALS_SECRET_ARN")
	googleAdminUser = os.Getenv("GOOGLE_ADMIN_USER")
	credentials, err := getGoogleCredentials(ctx)
	if err != nil {
		log.Error(ctx, "Error loading google credentials", err)
		panic("Failed to load google credentials")
	}

	googleClient, err = NewGoogleAdminClient(ctx, googleAdminUser, credentials)
	if err != nil {
		log.Error(ctx, "Create googleAdminClient error", err)
		panic("Failed to create google Admin Client")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Error(ctx, "Unable to load AWS SDK config", err)
		panic("Failed to load AWS SDK config")
	}

	s3Client = s3.NewFromConfig(cfg)
	s3Bucket = os.Getenv("S3_BUCKET")
}

func handler(ctx context.Context, event *ListActivityEvent) error {
	log.Info(ctx, "Fetching events", "application_name", event.ApplicationName, "time_window_minutes", event.TimeWindowMinutes)

	timeDuration := time.Duration(event.TimeWindowMinutes) * time.Minute
	err, events := googleClient.ListActivities(ctx, event.ApplicationName, timeDuration)
	if err != nil {
		log.Error(ctx, "Error Listing Activities", err, "AppplicationName", event.ApplicationName, "timeDuration", timeDuration)
	}

	err = storeEvents(ctx, event.ApplicationName, events)
	if err != nil {
		log.Error(ctx, "Error storing events on S3", err)
	}

	return nil
}

func isLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}

func getGoogleCredentials(ctx context.Context) ([]byte, error) {
	if isLocal() {
		log.Debug(ctx, "Loading credentials from file")
		byte, err := os.ReadFile("credentials.json")
		if err != nil {
			return nil, err
		}
		return byte, nil
	}

	secret, err := secret.GetSecretValue(ctx, googleCredentialsSecretArn)
	if err != nil {
		log.Error(ctx, "Unable to retrieve googleCredentialsSecret", err)
		return nil, err
	}

	return []byte(*secret), nil
}

func storeEvents(ctx context.Context, applicationName string, events []*Activity) error {
	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Error(ctx, "Error Marshaling JSON", err, "event", event)
			return err
		}

		hash := sha256.New()
		hash.Write(eventJSON)
		hashSum := hex.EncodeToString(hash.Sum(nil))

		objectKey := fmt.Sprintf("%s/%s.json", applicationName, hashSum)

		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s3Bucket),
			Key:         aws.String(objectKey),
			Body:        bytes.NewReader(eventJSON),
			ContentType: aws.String("application/json"),
			IfNoneMatch: aws.String("*"),
		})
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				// Check if it's a PreconditionFailed error (object already exists)
				if ae.ErrorCode() == "PreconditionFailed" {
					log.Debug(ctx, "Object already exists")
					continue
				} else {
					log.Error(ctx, "Unknown error", "ErrorCode", ae.ErrorCode(), "ErrorMessage", ae.ErrorMessage())
				}
			}
			log.Error(ctx, "Error uploading to S3", err)
			return err
		}
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
