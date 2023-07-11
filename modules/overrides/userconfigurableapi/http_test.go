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
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend/local"
)

func Test_UserConfigOverridesAPI_overridesHandlers(t *testing.T) {
	tenant := "single-tenant"

	overridesAPI, err := NewUserConfigOverridesAPI(&UserConfigurableOverridesClientConfig{
		Backend: "local",
		Local:   &local.Config{Path: t.TempDir()},
	})
	require.NoError(t, err)

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
			req:            httptest.NewRequest("GET", "/", nil),
			expResp:        "{\"forwarders\":[\"my-other-forwarder\"]}",
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "POST",
			handler:        overridesAPI.PostOverridesHandler,
			req:            httptest.NewRequest("POST", "/", bytes.NewReader(postJSON)),
			expResp:        "ok",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  200,
		},
		{
			name:           "POST - invalid JSON",
			handler:        overridesAPI.PostOverridesHandler,
			req:            httptest.NewRequest("POST", "/", bytes.NewReader([]byte("not a json"))),
			expResp:        "skipThreeBytes: expect ull, error found in #2 byte of ...|not a json|..., bigger context ...|not a json|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "POST - unknown field JSON",
			handler:        overridesAPI.PostOverridesHandler,
			req:            httptest.NewRequest("POST", "/", bytes.NewReader([]byte("{\"unknown\":true}"))),
			expResp:        "userconfigurableapi.UserConfigurableLimits.ReadObject: found unknown field: unknown, error found in #10 byte of ...|{\"unknown\":true}|..., bigger context ...|{\"unknown\":true}|...\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "DELETE",
			handler:        overridesAPI.DeleteOverridesHandler,
			req:            httptest.NewRequest("DELETE", "/", nil),
			expResp:        "ok",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// inject "single-tenant" in tc.req
			ctx := user.InjectOrgID(tc.req.Context(), util.FakeTenantID)
			tc.req = tc.req.WithContext(ctx)

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
