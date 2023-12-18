// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensusreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"

import (
	"go.opentelemetry.io/collector/config/configgrpc"
)

// ocOption interface defines for configuration settings to be applied to receivers.
//
// withReceiver applies the configuration to the given receiver.
type ocOption interface {
	withReceiver(*ocReceiver)
}

type corsOrigins struct {
	origins []string
}

var _ ocOption = (*corsOrigins)(nil)

func (co *corsOrigins) withReceiver(ocr *ocReceiver) {
	ocr.corsOrigins = co.origins
}

// withCorsOrigins is an option to specify the allowed origins to enable writing
// HTTP/JSON requests to the grpc-gateway adapter using CORS.
func withCorsOrigins(origins []string) ocOption {
	return &corsOrigins{origins: origins}
}

type grpcServerSettings configgrpc.GRPCServerSettings

func withGRPCServerSettings(settings configgrpc.GRPCServerSettings) ocOption {
	gsvOpts := grpcServerSettings(settings)
	return gsvOpts
}
func (gsvo grpcServerSettings) withReceiver(ocr *ocReceiver) {
	ocr.grpcServerSettings = configgrpc.GRPCServerSettings(gsvo)
}
