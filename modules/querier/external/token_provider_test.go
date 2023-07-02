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
		getTokenSource := func(ctx context.Context, endpoint string) (oauth2.TokenSource, error) {
			return getDummyTokenSource(fmt.Sprintf("%s-token", endpoint)), nil
		}

		tp, err := newTokenProvider(context.Background(), tc.endpoints, getTokenSource)
		require.NoError(t, err)

		actual, err := tp.getToken(context.Background(), tc.testEndpoint)
		require.NoError(t, err)
		require.Equal(t, tc.expectedToken, actual)
	}
}

type dummyTokenSource struct {
	dummyToken string
}

func (t *dummyTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.dummyToken,
	}, nil
}

func getDummyTokenSource(dummyToken string) oauth2.TokenSource {
	return &dummyTokenSource{dummyToken: dummyToken}
}
