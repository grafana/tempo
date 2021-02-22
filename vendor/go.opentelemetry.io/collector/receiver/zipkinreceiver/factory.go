// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkinreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

// This file implements factory for Zipkin receiver.

const (
	// The value of "type" key in configuration.
	typeStr = "zipkin"

	defaultBindEndpoint = "0.0.0.0:9411"
)

// NewFactory creates a new Zipkin receiver factory
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTraceReceiver),
	)
}

// createDefaultConfig creates the default configuration for Zipkin receiver.
func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		HTTPServerSettings: confighttp.HTTPServerSettings{
			Endpoint: defaultBindEndpoint,
		},
		ParseStringTags: false,
	}
}

// createTraceReceiver creates a trace receiver based on provided config.
func createTraceReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TracesConsumer,
) (component.TracesReceiver, error) {
	rCfg := cfg.(*Config)
	return New(rCfg, nextConsumer)
}
