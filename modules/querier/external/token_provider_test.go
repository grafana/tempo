package external

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCachedTokenProvider(t *testing.T) {
	tests := []struct {
		endpoints     []string
		expectedToken *oauth2.Token
		testEndpoint  string
	}{
		{
			endpoints:     []string{"foo"},
			testEndpoint:  "foo",
			expectedToken: &oauth2.Token{AccessToken: "foo-token"},
		},
		{
			endpoints:     []string{"foo", "bar"},
			testEndpoint:  "foo",
			expectedToken: &oauth2.Token{AccessToken: "foo-token"},
		},
		{
			endpoints:     []string{"foo", "bar"},
			testEndpoint:  "bar",
			expectedToken: &oauth2.Token{AccessToken: "bar-token"},
		},
	}

	for _, tc := range tests {
		tp, err := newCachedTokenProvider(context.Background(), tc.endpoints, &stubbedTokenProvider{})
		require.NoError(t, err)

		actual, err := tp.getToken(context.Background(), tc.testEndpoint)
		require.NoError(t, err)
		require.Equal(t, tc.expectedToken, actual)
	}
}

type stubbedTokenSource struct {
	dummyToken string
}

func (t *stubbedTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.dummyToken,
	}, nil
}

type stubbedTokenProvider struct {
}

func (t *stubbedTokenProvider) getToken(_ context.Context, _ string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("this is a stubbed function")
}
func (t *stubbedTokenProvider) getTokenSource(_ context.Context, endpoint string) (oauth2.TokenSource, error) {
	return &stubbedTokenSource{
		dummyToken: fmt.Sprintf("%s-token", endpoint),
	}, nil
}
