package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/log"
	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/secret"
)

var (
	s3Bucket      string
	s3Client      *s3.Client
	httpClient    *http.Client
	datadogApiKey *string
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Error(ctx, "Unable to load AWS SDK config", err)
		panic("Failed to load AWS SDK config")
	}

	s3Client = s3.NewFromConfig(cfg)
	s3Bucket = os.Getenv("S3_BUCKET")

	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	datadogApiKeySecretArn := os.Getenv("DATADOG_API_KEY_SECRET_ARN")

	datadogApiKey, err = secret.GetSecretValue(ctx, datadogApiKeySecretArn)
	if err != nil {
		log.Error(ctx, "Unable to retrieve datadogApiKeySecretArn", err)
		panic("Unable to retrieve Datadog API Key")
	}
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	log.Debug(ctx, "Records", s3Event.Records)

	err := processRecords(ctx, s3Event.Records)
	if err != nil {
		log.Error(ctx, "Error processing records", err)
		return err
	}

	return nil
}

func processRecords(ctx context.Context, records []events.S3EventRecord) error {
	for _, record := range records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.URLDecodedKey
		getOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		})
		if err != nil {
			log.Error(ctx, "Error getting object", "bucket", bucket, "key", key, "error", err)
			return err
		}

		err = sendToDatadog(ctx, getOutput.Body)
		if err != nil {
			log.Error(ctx, "Error sending to Datadog", "bucket", bucket, "key", key, "error", err)
			return err
		}
	}

	return nil
}

func sendToDatadog(ctx context.Context, body io.ReadCloser) error {
	defer body.Close()

	url := "https://http-intake.logs.datadoghq.com/api/v2/logs?ddsource=gsuite"

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Error(ctx, "Unable to make http request", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", *datadogApiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error(ctx, "HTTP request failed", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Error(ctx, "Received non 202 response", "statusCode", resp.StatusCode, "body", string(responseBody))
		return err
	}

	return nil
}

func isLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}

func main() {
	lambda.Start(handler)
}
