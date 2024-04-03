package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

func Test_UserConfigOverridesAPI_overridesHandlers(t *testing.T) {
	tenant := "my-tenant"

	cfg := client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	validator := &mockValidator{}
	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &cfg, o, validator)
	require.NoError(t, err)

	// Provision some data
	_, err = overridesAPI.client.Set(context.Background(), tenant, &client.Limits{
		Forwarders: &[]string{"my-other-forwarder"},
	}, backend.VersionNew)
	require.NoError(t, err)

	postJSON, err := jsoniter.Marshal(&client.Limits{
		Forwarders: &[]string{"my-updated-forwarder"},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		handler        http.HandlerFunc
		req            *http.Request
		validatorErr   error
		expResp        string
		expContentType string
		expStatusCode  int
	}{
		{
			name:           "GET",
			handler:        overridesAPI.GetHandler,
			req:            prepareRequest(tenant, "GET", nil),
			expResp:        `{"forwarders":["my-other-forwarder"],"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:          "GET - not found",
			handler:       overridesAPI.GetHandler,
			req:           prepareRequest("some-other-tenant", "GET", nil),
			expStatusCode: 404,
		},
		{
			name:          "POST",
			handler:       overridesAPI.PostHandler,
			req:           prepareRequest(tenant, "POST", postJSON),
			expStatusCode: 200,
		},
		{
			name:           "POST - invalid JSON",
			handler:        overridesAPI.PostHandler,
			req:            prepareRequest(tenant, "POST", []byte("not a json")),
			expResp:        "skipThreeBytes: expect ull, error found in #2 byte of ...|not a json|..., bigger context ...|not a json|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "POST - unknown field JSON",
			handler:        overridesAPI.PostHandler,
			req:            prepareRequest(tenant, "POST", []byte("{\"unknown\":true}")),
			expResp:        "client.Limits.ReadObject: found unknown field: unknown, error found in #10 byte of ...|{\"unknown\":true}|..., bigger context ...|{\"unknown\":true}|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "POST - invalid overrides",
			handler:        overridesAPI.PostHandler,
			req:            prepareRequest(tenant, "POST", postJSON),
			validatorErr:   errors.New("these limits are invalid"),
			expResp:        "these limits are invalid\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:          "DELETE",
			handler:       overridesAPI.DeleteHandler,
			req:           prepareRequest(tenant, "DELETE", nil),
			expStatusCode: 200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validator.err = tc.validatorErr

			w := httptest.NewRecorder()
			tc.handler(w, tc.req)

			data := w.Body.String()
			assert.Equal(t, tc.expResp, data)

			res := w.Result()
			assert.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			assert.Equal(t, tc.expStatusCode, res.StatusCode)

			if tc.req.Method == http.MethodPost {
				limits, _, err := overridesAPI.client.Get(context.Background(), tenant)
				assert.NoError(t, err)
				assert.NotNil(t, limits.Forwarders)
				assert.Equal(t, *limits.Forwarders, []string{"my-updated-forwarder"})
			}
		})
	}
}

func Test_UserConfigOverridesAPI_patchOverridesHandlers(t *testing.T) {
	tenant := "my-tenant"

	tests := []struct {
		name           string
		patch          string
		current        string
		expResp        string
		expContentType string
		expStatusCode  int
	}{
		{
			name:           "PATCH - no values stored yet",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        ``,
			expResp:        `{"forwarders":["my-other-forwarder"],"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - empty overrides are merged",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{}`,
			expResp:        `{"forwarders":["my-other-forwarder"],"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - overwrite",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{"forwarders":["previous-forwarder"]}`,
			expResp:        `{"forwarders":["my-other-forwarder"],"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:          "PATCH - invalid patch",
			patch:         `{"newField":true}`,
			current:       `{"forwarders":["prior-forwarder"]}`,
			expResp:       "client.Limits.ReadObject: found unknown field: newField, error found in #10 byte of ...|\"newField\":true}|..., bigger context ...|\"service_graphs\":{},\"span_metrics\":{}}},\"newField\":true}|...\n",
			expStatusCode: 400,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
			assert.NoError(t, err)

			overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &client.Config{
				Backend: backend.Local,
				Local:   &local.Config{Path: t.TempDir()},
			}, o, &mockValidator{})
			require.NoError(t, err)

			if tc.current != "" {
				_, err := overridesAPI.client.Set(context.Background(), tenant, parseJSON(t, tc.current), backend.VersionNew)
				assert.NoError(t, err)
			}

			w := httptest.NewRecorder()

			r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(tc.patch)))
			ctx := user.InjectOrgID(r.Context(), tenant)
			r = r.WithContext(ctx)

			overridesAPI.PatchHandler(w, r)

			data := w.Body.String()
			require.Equal(t, tc.expResp, data)

			res := w.Result()
			if tc.expContentType != "" {
				require.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			}
			require.Equal(t, tc.expStatusCode, res.StatusCode)
		})
	}
}

func TestUserConfigOverridesAPI_patchOverridesHandler_noVersionConflict(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}, o, &mockValidator{})
	require.NoError(t, err)

	// inject our client
	testClient := &testClient{}
	overridesAPI.client = testClient

	testClient.get = func(ctx context.Context, userID string) (*client.Limits, backend.Version, error) {
		return &client.Limits{}, "1", nil
	}
	testClient.set = func(ctx context.Context, userID string, limits *client.Limits, version backend.Version) (backend.Version, error) {
		// Must pass in version from get
		assert.Equal(t, backend.Version("1"), version)
		assert.NotNil(t, limits)
		assert.Equal(t, client.Limits{Forwarders: &[]string{"f"}}, *limits)
		return "2", nil
	}

	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{"forwarders":["f"]}`)))
	ctx := user.InjectOrgID(r.Context(), "foo")
	r = r.WithContext(ctx)

	overridesAPI.PatchHandler(w, r)

	data := w.Body.String()
	assert.Equal(t, `{"forwarders":["f"],"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{}}}}`, data)

	res := w.Result()
	assert.Equal(t, "2", res.Header.Get(headerEtag))
	assert.Equal(t, 200, res.StatusCode)
}

func TestUserConfigOverridesAPI_patchOverridesHandler_versionConflict(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}, o, &mockValidator{})
	require.NoError(t, err)

	// inject our client
	testClient := &testClient{}
	overridesAPI.client = testClient

	testClient.get = func(ctx context.Context, userID string) (*client.Limits, backend.Version, error) {
		return &client.Limits{}, "1", nil
	}
	testClient.set = func(ctx context.Context, userID string, limits *client.Limits, version backend.Version) (backend.Version, error) {
		// Someone else changed the file!
		return "", backend.ErrVersionDoesNotMatch
	}

	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{"forwarders":["f"]}`)))
	ctx := user.InjectOrgID(r.Context(), "foo")
	r = r.WithContext(ctx)

	overridesAPI.PatchHandler(w, r)

	res := w.Result()
	assert.Equal(t, 500, res.StatusCode)

	data := w.Body.String()
	assert.Equal(t, "overrides have been modified during request processing, try again\n", data)
}

func TestUserConfigOverridesAPI_assertConflictingRuntimeOverrides(t *testing.T) {
	tenant := "foo"

	testCases := []struct {
		name                                string
		checkForConflictingRuntimeOverrides bool
		defaultOverrides                    overrides.Overrides
		userConfigOverrides                 *client.Limits
		request                             *client.Limits
		skipConflictingOverridesCheck       string
		expStatusCode                       int
		expResp                             string
	}{
		{
			name:                                "No conflicting runtime overrides",
			checkForConflictingRuntimeOverrides: true,
			defaultOverrides: overrides.Overrides{
				Ingestion: overrides.IngestionOverrides{
					RateStrategy: overrides.GlobalIngestionRateStrategy,
				},
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					// processors is ignored when checking for conflicting fields since we merge this field
					Processors: map[string]struct{}{"service-graphs": {}},
				},
			},
			userConfigOverrides: nil,
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			expStatusCode: 200,
			expResp:       "",
		},
		{
			name:                                "Conflicting runtime overrides",
			checkForConflictingRuntimeOverrides: true,
			defaultOverrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: 15 * time.Second,
				},
			},
			userConfigOverrides: nil,
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			expStatusCode: 400,
			expResp:       errConflictingRuntimeOverrides.Error() + "\n",
		},
		{
			name:                                "Conflicting runtime overrides but check disabled",
			checkForConflictingRuntimeOverrides: false,
			defaultOverrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					CollectionInterval: 15 * time.Second,
				},
			},
			userConfigOverrides: nil,
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			expStatusCode: 200,
			expResp:       "",
		},
		{
			name:                                "Conflicting runtime overrides but skip check",
			checkForConflictingRuntimeOverrides: true,
			defaultOverrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: 15 * time.Second,
				},
			},
			userConfigOverrides: nil,
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			skipConflictingOverridesCheck: "true",
			expStatusCode:                 200,
			expResp:                       "",
		},
		{
			name:                                "Conflicting runtime overrides but already has user-config overiddes",
			checkForConflictingRuntimeOverrides: true,
			defaultOverrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: 15 * time.Second,
				},
			},
			userConfigOverrides: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 30 * time.Second},
				},
			},
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			expStatusCode: 200,
			expResp:       "",
		},
		{
			name:                                "Invalid skip check parameter",
			checkForConflictingRuntimeOverrides: true,
			defaultOverrides: overrides.Overrides{
				MetricsGenerator: overrides.MetricsGeneratorOverrides{
					Processors:         map[string]struct{}{"service-graphs": {}},
					CollectionInterval: 15 * time.Second,
				},
			},
			userConfigOverrides: nil,
			request: &client.Limits{
				MetricsGenerator: client.LimitsMetricsGenerator{
					CollectionInterval: &client.Duration{Duration: 60 * time.Second},
				},
			},
			skipConflictingOverridesCheck: "yes",
			expStatusCode:                 400,
			expResp:                       "could not parse skip-conflicting-overrides-check, must be a boolean value\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := overrides.Config{
				Defaults: tc.defaultOverrides,
				UserConfigurableOverridesConfig: overrides.UserConfigurableOverridesConfig{
					Enabled: true,
					Client: client.Config{
						Backend: backend.Local,
						Local:   &local.Config{Path: t.TempDir()},
					},
					API: overrides.UserConfigurableOverridesAPIConfig{
						CheckForConflictingRuntimeOverrides: tc.checkForConflictingRuntimeOverrides,
					},
				},
				ConfigType: "",
			}
			o, err := overrides.NewOverrides(cfg, nil, prometheus.DefaultRegisterer)
			assert.NoError(t, err)

			overridesAPI, err := New(&cfg.UserConfigurableOverridesConfig.API, &cfg.UserConfigurableOverridesConfig.Client, o, &mockValidator{})
			require.NoError(t, err)

			version := backend.VersionNew
			if tc.userConfigOverrides != nil {
				_, err = overridesAPI.client.Set(context.Background(), tenant, tc.userConfigOverrides, backend.VersionNew)
				assert.NoError(t, err)
			}

			w := httptest.NewRecorder()

			json, err := jsoniter.Marshal(tc.request)
			assert.NoError(t, err)

			path := "/"
			if tc.skipConflictingOverridesCheck != "" {
				path = fmt.Sprintf("%s?%s=%s", path, queryParamSkipConflictingOverridesCheck, tc.skipConflictingOverridesCheck)
			}
			r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(json))
			r.Header.Set(headerIfMatch, string(version))
			ctx := user.InjectOrgID(r.Context(), tenant)
			r = r.WithContext(ctx)

			overridesAPI.PostHandler(w, r)

			res := w.Result()
			assert.Equal(t, tc.expStatusCode, res.StatusCode)

			data := w.Body.String()
			assert.Equal(t, tc.expResp, data)
		})
	}
}

func prepareRequest(tenant, method string, payload []byte) *http.Request {
	r := httptest.NewRequest(method, "/", bytes.NewReader(payload))
	ctx := user.InjectOrgID(r.Context(), tenant)
	r = r.WithContext(ctx)

	if method == "POST" || method == "DELETE" {
		r.Header.Set(headerIfMatch, string(backend.VersionNew))
	}

	return r
}

func parseJSON(t *testing.T, s string) *client.Limits {
	var limits client.Limits
	err := jsoniter.Unmarshal([]byte(s), &limits)
	require.NoError(t, err)
	return &limits
}

type testClient struct {
	get func(context.Context, string) (*client.Limits, backend.Version, error)
	set func(context.Context, string, *client.Limits, backend.Version) (backend.Version, error)
}

var _ client.Client = (*testClient)(nil)

func (t *testClient) List(_ context.Context) ([]string, error) {
	panic("implement me")
}

func (t *testClient) Get(ctx context.Context, userID string) (*client.Limits, backend.Version, error) {
	return t.get(ctx, userID)
}

func (t *testClient) Set(ctx context.Context, userID string, limits *client.Limits, version backend.Version) (backend.Version, error) {
	return t.set(ctx, userID, limits, version)
}

func (t *testClient) Delete(_ context.Context, _ string, _ backend.Version) error {
	panic("implement me")
}

func (t *testClient) Shutdown() {
}

type mockValidator struct {
	err error
}

func (m *mockValidator) Validate(_ *client.Limits) error {
	return m.err
}
