// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package topic // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/topic"

import (
	"context"
)

func WithTopic(ctx context.Context, topic string) context.Context {
	return context.WithValue(ctx, topicContextKey{}, topic)
}

func FromContext(ctx context.Context) (string, bool) {
	contextTopic, ok := ctx.Value(topicContextKey{}).(string)
	return contextTopic, ok
}

type topicContextKey struct{}
