package main

import (
	"context"

	admin "google.golang.org/api/admin/reports/v1"
)

type Activity struct {
	Actor       *admin.ActivityActor `json:"actor,omitempty"`
	Etag        string               `json:"etag,omitempty"`
	Id          *admin.ActivityId    `json:"id,omitempty"`
	IpAddress   string               `json:"ipAddress,omitempty"`
	Kind        string               `json:"kind,omitempty"`
	OwnerDomain string               `json:"ownerDomain,omitempty"`
	Event       *Event               `json:"event"`
}

type Event struct {
	Name       string                 `json:"name,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Type       string                 `json:"type,omitempty"`
}

func NewActivitiesFromResponseItem(ctx context.Context, googleActivity *admin.Activity) []*Activity {
	var activities []*Activity

	// The data we get back from Google's API has an _array_ of Events. We want
	// to split each of those out into their own individual event, but keep the
	// rest of the audit data (Actor, ID, IP, etc) associated with it. We also
	// use this opportunity to transform the data shape a bit as needed.
	for _, event := range googleActivity.Events {
		activity := &Activity{}
		activity.Actor = googleActivity.Actor
		activity.Etag = googleActivity.Etag
		activity.Id = googleActivity.Id
		activity.IpAddress = googleActivity.IpAddress
		activity.Kind = googleActivity.Kind
		activity.OwnerDomain = googleActivity.OwnerDomain
		activity.Event = NewEvent(ctx, event)
		activities = append(activities, activity)
	}

	return activities
}

func NewEvent(ctx context.Context, googleEvent *admin.ActivityEvents) *Event {
	return &Event{
		Name:       googleEvent.Name,
		Parameters: transformParameters(ctx, googleEvent.Parameters),
		Type:       googleEvent.Type,
	}
}

func transformParameters(ctx context.Context, parameters []*admin.ActivityEventsParameters) map[string]interface{} {
	transformedParams := make(map[string]interface{})

	for _, param := range parameters {
		if param.Value != "" {
			transformedParams[param.Name] = param.Value
		} else if len(param.MultiValue) > 0 {
			transformedParams[param.Name] = param.MultiValue
		} else if param.BoolValue {
			transformedParams[param.Name] = param.BoolValue
		} else if param.IntValue != 0 {
			transformedParams[param.Name] = param.IntValue
		} else if len(param.MultiIntValue) > 0 {
			transformedParams[param.Name] = param.MultiIntValue
		} else {
			// If no value is present, default to `false`
			transformedParams[param.Name] = false
		}
	}

	return transformedParams
}
