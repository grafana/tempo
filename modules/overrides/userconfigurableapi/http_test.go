package userconfigurableapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"golang.org/x/net/context"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

func Test_UserConfigOverridesAPI_overridesHandlers(t *testing.T) {
	tenant := "my-tenant"

	overridesAPI, err := NewUserConfigOverridesAPI(&UserConfigurableOverridesClientConfig{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	})
	require.NoError(t, err)

	// Provision some data
	require.NoError(t, overridesAPI.client.Set(context.Background(), tenant, &UserConfigurableLimits{
		Forwarders: &[]string{"my-other-forwarder"},
	}))

	postJSON, err := jsoniter.Marshal(&UserConfigurableLimits{
		Forwarders: &[]string{"my-updated-forwarder"},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		handler        http.HandlerFunc
		req            *http.Request
		expResp        string
		expContentType string
		expStatusCode  int
	}{
		{
			name:           "GET",
			handler:        overridesAPI.GetOverridesHandler,
			req:            prepareRequest(tenant, "GET", nil),
			expResp:        "{\"forwarders\":[\"my-other-forwarder\"]}",
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:          "GET - not found",
			handler:       overridesAPI.GetOverridesHandler,
			req:           prepareRequest("some-other-tenant", "GET", nil),
			expStatusCode: 404,
		},
		{
			name:          "POST",
			handler:       overridesAPI.PostOverridesHandler,
			req:           prepareRequest(tenant, "POST", postJSON),
			expStatusCode: 200,
		},
		{
			name:           "POST - invalid JSON",
			handler:        overridesAPI.PostOverridesHandler,
			req:            prepareRequest(tenant, "POST", []byte("not a json")),
			expResp:        "skipThreeBytes: expect ull, error found in #2 byte of ...|not a json|..., bigger context ...|not a json|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "POST - unknown field JSON",
			handler:        overridesAPI.PostOverridesHandler,
			req:            prepareRequest(tenant, "POST", []byte("{\"unknown\":true}")),
			expResp:        "userconfigurableapi.UserConfigurableLimits.ReadObject: found unknown field: unknown, error found in #10 byte of ...|{\"unknown\":true}|..., bigger context ...|{\"unknown\":true}|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:          "DELETE",
			handler:       overridesAPI.DeleteOverridesHandler,
			req:           prepareRequest(tenant, "DELETE", nil),
			expStatusCode: 200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, tc.req)

			data := w.Body.String()
			require.Equal(t, tc.expResp, data)

			res := w.Result()
			require.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			require.Equal(t, tc.expStatusCode, res.StatusCode)

			if tc.req.Method == http.MethodPost {
				limits, err := overridesAPI.client.Get(context.Background(), tenant)
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
			name:           "PATCH - no value set yet",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        ``,
			expResp:        `{"forwarders":["my-other-forwarder"]}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - empty overrides are merged",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{}`,
			expResp:        `{"forwarders":["my-other-forwarder"]}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "PATCH - overwrite",
			patch:          `{"forwarders":["my-other-forwarder"]}`,
			current:        `{"forwarders":["prior-forwarder"]}`,
			expResp:        `{"forwarders":["my-other-forwarder"]}`,
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overridesAPI, err := NewUserConfigOverridesAPI(&UserConfigurableOverridesClientConfig{
				Backend: backend.Local,
				Local:   &local.Config{Path: t.TempDir()},
			})
			require.NoError(t, err)

			if tc.current != "" {
				err := overridesAPI.client.Set(context.Background(), tenant, parseJson(t, tc.current))
				assert.NoError(t, err)
			}

			w := httptest.NewRecorder()

			r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(tc.patch)))
			ctx := user.InjectOrgID(r.Context(), tenant)
			r = r.WithContext(ctx)

			overridesAPI.PatchOverridesHandler(w, r)

			data := w.Body.String()
			require.Equal(t, tc.expResp, data)

			res := w.Result()
			require.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			require.Equal(t, tc.expStatusCode, res.StatusCode)
		})
	}
}

func prepareRequest(tenant, method string, payload []byte) *http.Request {
	r := httptest.NewRequest(method, "/", bytes.NewReader(payload))
	ctx := user.InjectOrgID(r.Context(), tenant)
	r = r.WithContext(ctx)
	return r
}

func parseJson(t *testing.T, s string) *UserConfigurableLimits {
	var limits UserConfigurableLimits
	err := jsoniter.Unmarshal([]byte(s), &limits)
	require.NoError(t, err)
	return &limits
}
