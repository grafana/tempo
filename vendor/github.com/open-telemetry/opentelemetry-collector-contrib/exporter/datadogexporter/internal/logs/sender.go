// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/logs"

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/clientutil"
)

// Sender submits logs to Datadog intake
type Sender struct {
	logger  *zap.Logger
	api     *datadogV2.LogsApi
	verbose bool // reports whether payload contents should be dumped when logging at debug level
}

// logsV2 is the key in datadog ServerConfiguration
// It is being used to customize the endpoint for datdog intake based on exporter configuration
// https://github.com/DataDog/datadog-api-client-go/blob/be7e034424012c7ee559a2153802a45df73232ea/api/datadog/configuration.go#L308
const logsV2 = "v2.LogsApi.SubmitLog"

// NewSender creates a new Sender
func NewSender(endpoint string, logger *zap.Logger, s exporterhelper.TimeoutSettings, insecureSkipVerify, verbose bool, apiKey string) *Sender {
	cfg := datadog.NewConfiguration()
	logger.Info("Logs sender initialized", zap.String("endpoint", endpoint))
	cfg.OperationServers[logsV2] = datadog.ServerConfigurations{
		datadog.ServerConfiguration{
			URL: endpoint,
		},
	}
	cfg.HTTPClient = clientutil.NewHTTPClient(s, insecureSkipVerify)
	cfg.AddDefaultHeader("DD-API-KEY", apiKey)
	apiClient := datadog.NewAPIClient(cfg)
	return &Sender{
		api:     datadogV2.NewLogsApi(apiClient),
		logger:  logger,
		verbose: verbose,
	}
}

// SubmitLogs submits the logs contained in payload to the Datadog intake
func (s *Sender) SubmitLogs(ctx context.Context, payload []datadogV2.HTTPLogItem) error {
	if s.verbose {
		s.logger.Debug("Submitting logs", zap.Any("payload", payload))
	}
	var (
		tags, prevtags string                  // keeps track of the ddtags of log items for grouping purposes
		batch          []datadogV2.HTTPLogItem // stores consecutive log items to be submitted together
	)
	// Correctly sets apiSubmitLogRequest ddtags field based on tags from translator Transform method
	for i, p := range payload {
		tags = p.GetDdtags()
		if prevtags == tags || i == 0 {
			// Batches consecutive log items with the same tags to be submitted together
			batch = append(batch, p)
			prevtags = tags
			continue
		}
		if err := s.handleSubmitLog(ctx, batch, prevtags); err != nil {
			return err
		}
		batch = []datadogV2.HTTPLogItem{p}
		prevtags = tags
	}
	if err := s.handleSubmitLog(ctx, batch, tags); err != nil {
		return err
	}
	return nil
}

func (s *Sender) handleSubmitLog(ctx context.Context, batch []datadogV2.HTTPLogItem, tags string) error {
	opts := *datadogV2.NewSubmitLogOptionalParameters().
		WithContentEncoding(datadogV2.CONTENTENCODING_GZIP).
		WithDdtags(tags)
	_, r, err := s.api.SubmitLog(ctx, batch, opts)
	if err != nil {
		if r != nil {
			b := make([]byte, 1024) // 1KB message max
			n, _ := r.Body.Read(b)  // ignore any error
			s.logger.Error("Failed to send logs", zap.Error(err), zap.String("msg", string(b[:n])), zap.String("status_code", r.Status))
			return err
		}
		// If response is nil assume permanent error.
		// The error will be logged by the exporter helper.
		return consumererror.NewPermanent(err)
	}
	return nil
}
