// Package pubsub contains utilities for handling Google Cloud Pub/Sub events.
package pubsub

import (
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/internal/fftypes"
)

const (
	pubsubEventType   = "google.pubsub.topic.publish"
	pubsubMessageType = "type.googleapis.com/google.pubusb.v1.PubsubMessage"
	pubsubService     = "pubsub.googleapis.com"
)

// LegacyPushSubscriptionEvent is the event payload for legacy Cloud Pub/Sub
// push subscription triggers (https://cloud.google.com/functions/docs/calling/pubsub#legacy_cloud_pubsub_triggers).
// This matched the event payload that is sent by Pub/Sub to HTTP push
// subscription endpoints (https://cloud.google.com/pubsub/docs/push#receiving_messages).
type LegacyPushSubscriptionEvent struct {
	Subscription string `json:"subscription"`
	Message      `json:"message"`
}

// Message is a pubsub.Message but with the correct JSON tag for the
// message ID field that matches https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type Message struct {
	pubsub.Message
	// The pubsub libary's Message.Id field (https://pkg.go.dev/cloud.google.com/go/internal/pubsub#Message)
	// doesn't have the correct JSON tag (it serializes to "id" instead of
	// "messageId"), so use this field to capture the JSON field with key
	// "messageId".
	IdFromJSON string `json:"messageId"`
}

// ExtractTopicFromRequestPath extracts a Pub/Sub topic from a URL request path.
func ExtractTopicFromRequestPath(path string) (string, error) {
	re := regexp.MustCompile(`(projects\/[^/?]+\/topics\/[^/?]+)/*`)
	matches := re.FindStringSubmatch(path)
	if matches == nil {
		return "", fmt.Errorf("failed to extract Pub/Sub topic name from the URL request path: %q, configure your subscription's push endpoint to use the following path pattern: 'projects/PROJECT_NAME/topics/TOPIC_NAME'",
			path)
	}

	// Index 0 is the entire input string matched, index 1 is the first submatch
	return matches[1], nil
}

// ToBackgroundEvent converts the event to the standard BackgroundEvent format
// for Background Functions.
func (e *LegacyPushSubscriptionEvent) ToBackgroundEvent(topic string) *fftypes.BackgroundEvent {
	timestamp := e.Message.PublishTime
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return &fftypes.BackgroundEvent{
		Metadata: &metadata.Metadata{
			EventID:   e.IdFromJSON,
			Timestamp: timestamp,
			EventType: pubsubEventType,
			Resource: &metadata.Resource{
				Name:    topic,
				Type:    pubsubMessageType,
				Service: pubsubService,
			},
		},
		Data: map[string]interface{}{
			"@type":      pubsubMessageType,
			"data":       e.Message.Data,
			"attributes": e.Message.Attributes,
		},
	}
}
