package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/util/listtomap"
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

	postJSON, err := json.Marshal(&client.Limits{
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
			expResp:        `{"forwarders":["my-other-forwarder"],"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
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

func TestUserConfigOverridesAPISecretDetection(t *testing.T) {
	tenant := "my-tenant"
	cfg := client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &cfg, o, &mockValidator{})
	require.NoError(t, err)

	body := []byte(`{"metrics_generator":{"processor":{"secret_detection":{"disabled_rules":["generic-api-key"],"custom_rules":[{"id":"customer-token","regex":"CUSTOMER-[0-9]+"}]}}}}`)
	post := httptest.NewRecorder()
	overridesAPI.PostHandler(post, prepareRequest(tenant, http.MethodPost, body))
	require.Equal(t, http.StatusOK, post.Code)

	get := httptest.NewRecorder()
	overridesAPI.GetHandler(get, prepareRequest(tenant, http.MethodGet, nil))
	require.Equal(t, http.StatusOK, get.Code)

	var limits client.Limits
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &limits))
	policy, ok := limits.GetMetricsGenerator().GetProcessor().GetSecretDetection()
	require.True(t, ok)
	assert.Equal(t, &secrets.Policy{
		DisabledRules: []string{"generic-api-key"},
		CustomRules:   []secrets.CustomRule{{ID: "customer-token", Regex: `CUSTOMER-[0-9]+`}},
	}, policy)
}

func TestUserConfigOverridesAPIRejectsRemovedSecretsPolicyFields(t *testing.T) {
	cfg := client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &cfg, o, &mockValidator{})
	require.NoError(t, err)

	for _, body := range []string{
		`{"metrics_generator":{"processor":{"secret_detection":{"catalog_version":"tempo-secrets-v1"}}}}`,
		`{"metrics_generator":{"processor":{"secret_detection":{"revision":7}}}}`,
		`{"metrics_generator":{"processor":{"secret_detection":{"optional_rules":["generic-api-key"]}}}}`,
	} {
		response := httptest.NewRecorder()
		overridesAPI.PostHandler(response, prepareRequest("my-tenant", http.MethodPost, []byte(body)))
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
}

func TestUserConfigOverridesAPIPolicyConfidentiality(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	oldTracer := tracer
	tracer = provider.Tracer("policy-confidentiality")
	t.Cleanup(func() {
		tracer = oldTracer
		require.NoError(t, provider.Shutdown(context.Background()))
	})
	store, err := client.New(&client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}})
	require.NoError(t, err)
	t.Cleanup(store.Shutdown)
	var logs bytes.Buffer
	a := &UserConfigOverridesAPI{
		cfg: &overrides.UserConfigurableOverridesAPIConfig{}, client: store,
		validator: &mockValidator{}, logger: log.NewLogfmtLogger(&logs),
	}
	const private = "synthetic-private-policy-marker"
	checkSafe := func(t *testing.T, output string) {
		t.Helper()
		require.False(t, strings.Contains(output, private), "diagnostics disclosed private policy text")
		encoded := fmt.Sprintf("%v", []byte(private))
		require.False(t, strings.Contains(output, encoded[1:len(encoded)-1]), "diagnostics disclosed raw policy bytes")
	}
	body := []byte(`{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `"}]}}}}`)
	response := httptest.NewRecorder()
	a.PostHandler(response, prepareRequest("tenant", http.MethodPost, body))
	require.Equal(t, http.StatusOK, response.Code)
	checkSafe(t, response.Body.String())

	policy := &secrets.Policy{CustomRules: []secrets.CustomRule{{ID: "safe-rule", Regex: private}}}
	assertPolicy := func(t *testing.T) {
		t.Helper()
		response := httptest.NewRecorder()
		a.GetHandler(response, prepareRequest("tenant", http.MethodGet, nil))
		require.Equal(t, http.StatusOK, response.Code)
		var got client.Limits
		require.True(t, json.Unmarshal(response.Body.Bytes(), &got) == nil, "authorized response must be valid JSON")
		require.True(t, reflect.DeepEqual(policy, got.MetricsGenerator.Processor.SecretDetection), "authorized GET must preserve complete policy")
		persisted, _, err := store.Get(context.Background(), "tenant")
		require.True(t, err == nil, "persisted policy must remain readable")
		require.True(t, reflect.DeepEqual(policy, persisted.MetricsGenerator.Processor.SecretDetection), "persistence must preserve complete policy")
	}
	assertPolicy(t)
	response = httptest.NewRecorder()
	a.PatchHandler(response, prepareRequest("tenant", http.MethodPatch, body))
	require.Equal(t, http.StatusOK, response.Code)
	assertPolicy(t)

	for _, tc := range []struct{ name, body string }{
		{"unknown-field", `{"` + private + `":true}`},
		{"invalid-policy-type", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":"` + private + `"}}}}`},
		{"removed-description", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `","description":"` + private + `"}]}}}}`},
		{"removed-entropy", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"` + private + `","entropy":1.5}]}}}}`},
		{"removed-secret-group", `{"metrics_generator":{"processor":{"secret_detection":{"custom_rules":[{"id":"safe-rule","regex":"(` + private + `)","secret_group":1}]}}}}`},
		{"second-document", string(body) + `{}`},
		{"malformed-suffix", string(body) + private},
		{"malformed-json", `{"` + private},
	} {
		for _, method := range []string{http.MethodPost, http.MethodPatch} {
			t.Run(tc.name+"-"+method, func(t *testing.T) {
				response := httptest.NewRecorder()
				request := prepareRequest("tenant", method, []byte(tc.body))
				if method == http.MethodPost {
					a.PostHandler(response, request)
				} else {
					a.PatchHandler(response, request)
				}
				expectedStatus := http.StatusBadRequest
				if tc.name == "malformed-json" && method == http.MethodPatch {
					// A non-policy malformed patch retains the merge library's
					// existing server-error response; its diagnostics stay safe.
					expectedStatus = http.StatusInternalServerError
				}
				require.Equal(t, expectedStatus, response.Code)
				checkSafe(t, response.Body.String())
				assertPolicy(t)
			})
		}
	}
	checkSafe(t, logs.String())
	seenSet, seenUpdate := false, false
	for _, span := range recorder.Ended() {
		seenSet = seenSet || span.Name() == "UserConfigOverridesAPI.set"
		seenUpdate = seenUpdate || span.Name() == "UserConfigOverridesAPI.update"
		checkSafe(t, fmt.Sprint(span.Attributes(), span.Events(), span.Status()))
	}
	require.True(t, seenSet && seenUpdate, "write and patch spans must be inspected")
}

func TestUserConfigOverridesAPINonPolicyDocumentCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name     string
		method   string
		existing bool
		status   int
	}{
		{"post-first-document", http.MethodPost, false, http.StatusOK},
		{"patch-first-document", http.MethodPatch, false, http.StatusOK},
		{"existing-patch-rejects-suffix", http.MethodPatch, true, http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := client.New(&client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}})
			require.NoError(t, err)
			t.Cleanup(store.Shutdown)
			var logs bytes.Buffer
			a := &UserConfigOverridesAPI{
				cfg: &overrides.UserConfigurableOverridesAPIConfig{}, client: store,
				validator: &mockValidator{}, logger: log.NewLogfmtLogger(&logs),
			}
			if tc.existing {
				_, err := store.Set(context.Background(), "tenant", &client.Limits{Forwarders: &[]string{"previous"}}, backend.VersionNew)
				require.NoError(t, err)
			}
			request := prepareRequest("tenant", tc.method, []byte(`{"forwarders":["first"]} {"forwarders":["later"]}`))
			response := httptest.NewRecorder()
			if tc.method == http.MethodPost {
				a.PostHandler(response, request)
			} else {
				a.PatchHandler(response, request)
			}
			require.Equal(t, tc.status, response.Code)
			limits, _, err := store.Get(context.Background(), "tenant")
			require.NoError(t, err)
			want := []string{"first"}
			if tc.existing {
				want = []string{"previous"}
			}
			require.Equal(t, &want, limits.Forwarders)
			require.Contains(t, logs.String(), want[0], "ordinary override values remain available in diagnostics")
		})
	}
}

func TestUserConfigOverridesAPI_invalidOrgID(t *testing.T) {
	invalidTenant := "invalid/org"

	cfg := client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &cfg, o, &mockValidator{})
	require.NoError(t, err)

	expectedError := "tenant ID 'invalid/org' contains unsupported character '/'"

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		request func() *http.Request
	}{
		{
			name:    "GET",
			handler: overridesAPI.GetHandler,
			request: func() *http.Request {
				return prepareRequest(invalidTenant, http.MethodGet, nil)
			},
		},
		{
			name:    "POST",
			handler: overridesAPI.PostHandler,
			request: func() *http.Request {
				return prepareRequest(invalidTenant, http.MethodPost, []byte("{}"))
			},
		},
		{
			name:    "PATCH",
			handler: overridesAPI.PatchHandler,
			request: func() *http.Request {
				return prepareRequest(invalidTenant, http.MethodPatch, []byte("{}"))
			},
		},
		{
			name:    "DELETE",
			handler: overridesAPI.DeleteHandler,
			request: func() *http.Request {
				return prepareRequest(invalidTenant, http.MethodDelete, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := tc.request()

			tc.handler(w, req)

			res := w.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, expectedError+"\n", w.Body.String())
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
			expResp:        `{"forwarders":["my-other-forwarder"],"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - empty overrides are merged",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{}`,
			expResp:        `{"forwarders":["my-other-forwarder"],"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - overwrite",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{"forwarders":["previous-forwarder"]}`,
			expResp:        `{"forwarders":["my-other-forwarder"],"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - processors: non-empty list over existing empty list sets the field",
			current:        `{"metrics_generator":{"processors":[]}}`,
			patch:          `{"metrics_generator":{"processors":["span-metrics"]}}`,
			expResp:        `{"cost_attribution":{},"metrics_generator":{"processors":["span-metrics"],"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - processors: null over existing list unsets the field (RFC 7386)",
			current:        `{"metrics_generator":{"processors":["span-metrics"]}}`,
			patch:          `{"metrics_generator":{"processors":null}}`,
			expResp:        `{"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - processors: empty list over existing non-empty list disables all processors",
			current:        `{"metrics_generator":{"processors":["span-metrics"]}}`,
			patch:          `{"metrics_generator":{"processors":[]}}`,
			expResp:        `{"cost_attribution":{},"metrics_generator":{"processors":[],"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
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
	assert.Equal(t, `{"forwarders":["f"],"cost_attribution":{},"metrics_generator":{"processor":{"service_graphs":{},"span_metrics":{},"host_info":{}}}}`, data)

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
					Processors:         &listtomap.ListToMap{"service-graphs": {}},
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
					Processors:         &listtomap.ListToMap{"service-graphs": {}},
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

			json, err := json.Marshal(tc.request)
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
	err := json.Unmarshal([]byte(s), &limits)
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

func Test_UserConfigOverridesAPI_CostAttribution(t *testing.T) {
	tenant := "test-tenant"
	cfg := client.Config{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	overridesAPI, err := New(&overrides.UserConfigurableOverridesAPIConfig{}, &cfg, o, &mockValidator{})
	require.NoError(t, err)

	// POST - create cost attribution config
	payload := `{"cost_attribution":{"dimensions":{"k8s.namespace.name":"namespace","k8s.cluster":"cluster"}}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	req.Header.Set(headerIfMatch, string(backend.VersionNew))
	overridesAPI.PostHandler(w, req)
	assert.Equal(t, 200, w.Code)

	// GET - verify POST created the config correctly
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	overridesAPI.GetHandler(w, req)
	assert.Equal(t, 200, w.Code)
	var limits client.Limits
	err = json.Unmarshal(w.Body.Bytes(), &limits)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k8s.namespace.name": "namespace", "k8s.cluster": "cluster"}, *limits.CostAttribution.Dimensions)

	// PATCH - update the config
	patch := `{"cost_attribution":{"dimensions":{"namespace":""}}}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte(patch)))
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	overridesAPI.PatchHandler(w, req)
	assert.Equal(t, 200, w.Code)

	// Verify PATCH response body
	err = json.Unmarshal(w.Body.Bytes(), &limits)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k8s.namespace.name": "namespace", "k8s.cluster": "cluster", "namespace": ""}, *limits.CostAttribution.Dimensions)

	// PATCH - again update the config but with remapping
	// it's treated as a new key because we change the config keys
	patchUpdate := `{"cost_attribution":{"dimensions":{"k8s.namespace":"namespace"}}}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte(patchUpdate)))
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	overridesAPI.PatchHandler(w, req)
	assert.Equal(t, 200, w.Code)

	// Verify PATCH response body
	err = json.Unmarshal(w.Body.Bytes(), &limits)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k8s.namespace.name": "namespace", "k8s.cluster": "cluster", "namespace": "", "k8s.namespace": "namespace"}, *limits.CostAttribution.Dimensions)

	// GET - verify PATCH merged the config correctly
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	overridesAPI.GetHandler(w, req)
	assert.Equal(t, 200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &limits)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k8s.namespace.name": "namespace", "k8s.cluster": "cluster", "namespace": "", "k8s.namespace": "namespace"}, *limits.CostAttribution.Dimensions)
	etag := w.Header().Get(headerEtag)

	// DELETE - remove the config
	w = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", "/", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	req.Header.Set(headerIfMatch, etag)
	overridesAPI.DeleteHandler(w, req)
	assert.Equal(t, 200, w.Code)

	// GET - verify DELETE removed the config (404)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), tenant))
	overridesAPI.GetHandler(w, req)
	assert.Equal(t, 404, w.Code)
}
