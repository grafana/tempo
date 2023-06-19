package external

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// Return Oauth2 token according to our google application credentials:
// https://cloud.google.com/run/docs/authenticating/service-to-service#acquire-token
func googleToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	ts, err := idtoken.NewTokenSource(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("cloud-run authenticator failed to create token source: %w", err)
	}
	return ts.Token()
}
