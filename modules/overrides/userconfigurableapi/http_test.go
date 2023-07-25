package userconfigurableapi

import (
	"bytes"
	"errors"
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

	cfg := UserConfigurableOverridesClientConfig{
		Backend: backend.Local,
		Local:   &local.Config{Path: t.TempDir()},
	}
	validator := &mockValidator{}
	overridesAPI, err := NewUserConfigOverridesAPI(&cfg, validator)
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
		validatorErr   error
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
			name:           "POST - invalid overrides",
			handler:        overridesAPI.PostOverridesHandler,
			req:            prepareRequest(tenant, "POST", postJSON),
			validatorErr:   errors.New("these limits are invalid"),
			expResp:        "these limits are invalid\n",
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
			validator.err = tc.validatorErr

			w := httptest.NewRecorder()
			tc.handler(w, tc.req)

			data := w.Body.String()
			assert.Equal(t, tc.expResp, data)

			res := w.Result()
			assert.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			assert.Equal(t, tc.expStatusCode, res.StatusCode)

			if tc.req.Method == http.MethodPost {
				limits, err := overridesAPI.client.Get(context.Background(), tenant)
				assert.NoError(t, err)
				assert.NotNil(t, limits.Forwarders)
				assert.Equal(t, *limits.Forwarders, []string{"my-updated-forwarder"})
			}
		})
	}
}

func prepareRequest(tenant, method string, payload []byte) *http.Request {
	r := httptest.NewRequest(method, "/", bytes.NewReader(payload))
	ctx := user.InjectOrgID(r.Context(), tenant)
	r = r.WithContext(ctx)
	return r
}

type mockValidator struct {
	err error
}

func (m *mockValidator) Validate(_ *UserConfigurableLimits) error {
	return m.err
}
