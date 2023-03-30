// Copyright 2020, OpenTelemetry Authors
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

package ecsutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil"

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/sanitize"
)

// Client defines the basic HTTP client interface with GET response validation and content parsing
type Client interface {
	Get(path string) ([]byte, error)
}

// NewClientProvider creates the default rest client provider
func NewClientProvider(baseURL url.URL, clientSettings confighttp.HTTPClientSettings, host component.Host, settings component.TelemetrySettings) ClientProvider {
	return &defaultClientProvider{
		baseURL:        baseURL,
		clientSettings: clientSettings,
		host:           host,
		settings:       settings,
	}
}

// ClientProvider defines
type ClientProvider interface {
	BuildClient() (Client, error)
}

type defaultClientProvider struct {
	baseURL        url.URL
	clientSettings confighttp.HTTPClientSettings
	host           component.Host
	settings       component.TelemetrySettings
}

func (dcp *defaultClientProvider) BuildClient() (Client, error) {
	return defaultClient(
		dcp.baseURL,
		dcp.clientSettings,
		dcp.host,
		dcp.settings,
	)
}

func defaultClient(
	baseURL url.URL,
	clientSettings confighttp.HTTPClientSettings,
	host component.Host,
	settings component.TelemetrySettings,
) (*clientImpl, error) {
	client, err := clientSettings.ToClient(host, settings)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("unexpected default client nil value")
	}
	return &clientImpl{
		baseURL:    baseURL,
		httpClient: *client,
		settings:   settings,
	}, nil
}

var _ Client = (*clientImpl)(nil)

type clientImpl struct {
	baseURL    url.URL
	httpClient http.Client
	settings   component.TelemetrySettings
}

func (c *clientImpl) Get(path string) ([]byte, error) {
	req, err := c.buildReq(path)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			c.settings.Logger.Warn("Failed to close response body", zap.Error(closeErr))
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request GET %s failed - %q", sanitize.URL(req.URL), resp.Status)
	}
	return body, nil
}

func (c *clientImpl) buildReq(path string) (*http.Request, error) {
	url := c.baseURL.String() + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
