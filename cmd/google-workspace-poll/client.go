package main

import (
	"context"
	"time"

	"github.com/planningcenter-platform/google-workspace-audit-log-to-datadog/internal/log"

	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/reports/v1"
	"google.golang.org/api/option"
)

// Client is the Interface for the Reports API Client
type GoogleAdminClient interface {
	ListActivities(context.Context, string, time.Duration) (error, []*Activity)
}

type googleAdminClient struct {
	service *admin.Service
}

func NewGoogleAdminClient(ctx context.Context, adminEmail string, serviceAccountKey []byte) (GoogleAdminClient, error) {
	config, err := google.JWTConfigFromJSON(serviceAccountKey, admin.AdminReportsAuditReadonlyScope)
	if err != nil {
		return nil, err
	}

	config.Subject = adminEmail
	tokenSource := config.TokenSource(ctx) // Sets the token source to JWT
	service, err := admin.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	return &googleAdminClient{
		service: service,
	}, nil
}

func (c *googleAdminClient) ListActivities(ctx context.Context, service string, maxAge time.Duration) (error, []*Activity) {
	var activities []*Activity

	log.Debug(ctx, "Listing activites")

	startTime := time.Now().Add(-maxAge).Format(time.RFC3339)

	log.Debug(ctx, "Filter", "startTime", startTime, "service", service)
	log.Debug(ctx, "Time", "now", time.Now().Format(time.RFC3339), "maxAge", maxAge)

	err := c.service.Activities.List("all", service).StartTime(startTime).Pages(ctx, func(response *admin.Activities) error {
		log.Debug(ctx, "Page of items found", "item_count", len(response.Items))

		for _, item := range response.Items {
			log.Debug(ctx, "Processing Item", "etag", item.Etag, "event_count", len(item.Events))
			activities = append(activities, NewActivitiesFromResponseItem(ctx, item)...)
		}

		return nil
	})
	if err != nil {
		log.Error(ctx, "Error retrieving pages of activities", err)
		return err, nil
	}

	log.Info(ctx, "Events fetched successfully", "event_count", len(activities))
	return nil, activities
}
