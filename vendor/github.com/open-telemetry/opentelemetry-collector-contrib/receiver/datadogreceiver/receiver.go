// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadogreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/datadogreceiver"

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/receiver"
)

type datadogReceiver struct {
	config       *Config
	params       receiver.CreateSettings
	nextConsumer consumer.Traces
	server       *http.Server
	tReceiver    *obsreport.Receiver
}

func newDataDogReceiver(config *Config, nextConsumer consumer.Traces, params receiver.CreateSettings) (receiver.Traces, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	instance, err := obsreport.NewReceiver(obsreport.ReceiverSettings{LongLivedCtx: false, ReceiverID: params.ID, Transport: "http", ReceiverCreateSettings: params})
	if err != nil {
		return nil, err
	}
	return &datadogReceiver{
		params:       params,
		config:       config,
		nextConsumer: nextConsumer,
		server: &http.Server{
			ReadTimeout: config.ReadTimeout,
			Addr:        config.HTTPServerSettings.Endpoint,
		},
		tReceiver: instance,
	}, nil
}

func (ddr *datadogReceiver) Start(_ context.Context, host component.Host) error {
	go func() {
		ddmux := http.NewServeMux()
		ddmux.HandleFunc("/v0.3/traces", ddr.handleTraces)
		ddmux.HandleFunc("/v0.4/traces", ddr.handleTraces)
		ddmux.HandleFunc("/v0.5/traces", ddr.handleTraces)
		ddmux.HandleFunc("/v0.7/traces", ddr.handleTraces)
		ddr.server.Handler = ddmux
		if err := ddr.server.ListenAndServe(); err != http.ErrServerClosed {
			host.ReportFatalError(fmt.Errorf("error starting datadog receiver: %w", err))
		}
	}()
	return nil
}

func (ddr *datadogReceiver) Shutdown(ctx context.Context) (err error) {
	return ddr.server.Shutdown(ctx)
}

func (ddr *datadogReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	obsCtx := ddr.tReceiver.StartTracesOp(req.Context())
	var err error
	var spanCount int
	defer func(spanCount *int) {
		ddr.tReceiver.EndTracesOp(obsCtx, "datadog", *spanCount, err)
	}(&spanCount)
	var ddTraces *pb.TracerPayload

	ddTraces, err = handlePayload(req)
	if err != nil {
		http.Error(w, "Unable to unmarshal reqs", http.StatusInternalServerError)
		ddr.params.Logger.Error("Unable to unmarshal reqs")
	}

	otelTraces := toTraces(ddTraces, req)
	spanCount = otelTraces.SpanCount()
	err = ddr.nextConsumer.ConsumeTraces(obsCtx, otelTraces)
	if err != nil {
		http.Error(w, "Trace consumer errored out", http.StatusInternalServerError)
		ddr.params.Logger.Error("Trace consumer errored out")
	} else {
		_, _ = w.Write([]byte("OK"))
	}
}
