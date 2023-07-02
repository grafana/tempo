package external

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"
)

func TestAuthHeader(t *testing.T) {
	authorizationHeader := atomic.NewString("")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		curr := authorizationHeader.Load()
		if curr != "" {
			http.Error(w, "authorization has already been set", http.StatusBadRequest)
			return
		}
		authHeader := r.Header.Get("authorization")
		authorizationHeader.Store(authHeader)

		// Create an instance of SearchResponse and populate its fields
		response := tempopb.SearchResponse{}

		// Marshal the SearchResponse struct into a JSON byte slice
		jsonData, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(jsonData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer srv.Close()

	tests := []struct {
		cfg                *commonConfig
		authHeaderExpected string
		options            []option
	}{
		{
			cfg: &commonConfig{
				endpoints: []string{srv.URL},
			},
			authHeaderExpected: "",
		},
		{
			cfg: &commonConfig{
				endpoints: []string{srv.URL},
			},
			options: []option{
				withTokenProvider(getDummyTokenProvider("dummytoken")),
			},

			authHeaderExpected: "Bearer dummytoken",
		},
	}

	ctx := user.InjectOrgID(context.Background(), "blerg")

	for _, tc := range tests {
		authorizationHeader.Store("")

		c, err := newClientWithOpts(tc.cfg, tc.options...)
		require.NoError(t, err)

		_, err = c.Search(ctx, 0, &tempopb.SearchBlockRequest{})
		require.NoError(t, err)

		require.Equal(t, tc.authHeaderExpected, authorizationHeader.Load())
	}
}

type dummyProvider struct {
	dummyToken string
}

func (t *dummyProvider) getToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.dummyToken,
	}, nil
}
func getDummyTokenProvider(dummyToken string) tokenProvider {
	return &dummyProvider{dummyToken: dummyToken}
}
