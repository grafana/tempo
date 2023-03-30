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

package clientutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/clientutil"

import (
	"context"
	"errors"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
	zorkian "gopkg.in/zorkian/go-datadog-api.v2"
)

// GZipSubmitMetricsOptionalParameters is used to enable gzip compression for metric payloads submitted by native datadog client
var GZipSubmitMetricsOptionalParameters = datadogV2.NewSubmitMetricsOptionalParameters().WithContentEncoding(datadogV2.METRICCONTENTENCODING_GZIP)

// CreateAPIClient creates a new Datadog API client
func CreateAPIClient(buildInfo component.BuildInfo, endpoint string, settings exporterhelper.TimeoutSettings, insecureSkipVerify bool) *datadog.APIClient {
	configuration := datadog.NewConfiguration()
	configuration.UserAgent = UserAgent(buildInfo)
	configuration.HTTPClient = NewHTTPClient(settings, insecureSkipVerify)
	configuration.Compress = true
	configuration.Servers = datadog.ServerConfigurations{
		{
			URL:         "{site}",
			Description: "No description provided",
			Variables:   map[string]datadog.ServerVariable{"site": {DefaultValue: endpoint}},
		},
	}
	return datadog.NewAPIClient(configuration)
}

// ValidateAPIKey checks if the API key (not the APP key) is valid
func ValidateAPIKey(ctx context.Context, apiKey string, logger *zap.Logger, apiClient *datadog.APIClient) error {
	logger.Info("Validating API key.")
	authAPI := datadogV1.NewAuthenticationApi(apiClient)
	resp, httpresp, err := authAPI.Validate(GetRequestContext(ctx, apiKey))
	if err == nil && resp.Valid != nil && *resp.Valid {
		logger.Info("API key validation successful.")
		return nil
	}
	if err != nil {
		logger.Warn("Error while validating API key", zap.Error(err))
		return nil
	}
	logger.Warn(ErrInvalidAPI.Error())
	return WrapError(ErrInvalidAPI, httpresp)
}

// GetRequestContext creates a new context with API key for DatadogV2 requests
func GetRequestContext(ctx context.Context, apiKey string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{"apiKeyAuth": {Key: apiKey}},
	)
}

// CreateZorkianClient creates a new Zorkian Datadog client
// Deprecated: CreateZorkianClient returns a Zorkian Datadog client and Zorkian is deprecated. Use CreateAPIClient instead.
func CreateZorkianClient(apiKey string, endpoint string) *zorkian.Client {
	client := zorkian.NewClient(apiKey, "")
	client.SetBaseUrl(endpoint)

	return client
}

var ErrInvalidAPI = errors.New("API Key validation failed")

// ValidateAPIKeyZorkian checks that the provided client was given a correct API key.
// Deprecated: ValidateAPIKeyZorkian uses the deprecated Zorkian client. Use ValidateAPIKey instead.
func ValidateAPIKeyZorkian(logger *zap.Logger, client *zorkian.Client) error {
	logger.Info("Validating API key.")
	valid, err := client.Validate()
	if err == nil && valid {
		logger.Info("API key validation successful.")
		return nil
	}
	if err != nil {
		logger.Warn("Error while validating API key", zap.Error(err))
		return nil
	}
	logger.Warn(ErrInvalidAPI.Error())
	return ErrInvalidAPI
}
