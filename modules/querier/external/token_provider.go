package external

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// Caches an oauth2.TokenSource to enable efficient auth on each of our external
// endpoints.
type cachedTokenProvider struct {
	tokenSources map[string]oauth2.TokenSource
}

func newTokenProvider(
	ctx context.Context,
	endpoints []string,
	getTokenSource func(ctx context.Context, endpoint string) (oauth2.TokenSource, error),
) (*cachedTokenProvider, error) {
	sources := make(map[string]oauth2.TokenSource, len(endpoints))

	var mtx sync.Mutex
	var wg sync.WaitGroup

	var tsErr error
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()
			ts, err := getTokenSource(ctx, ep)

			mtx.Lock()
			defer mtx.Unlock()

			if err != nil {
				tsErr = multierr.Combine(tsErr, err)
			}
			sources[ep] = ts
		}(endpoint)
	}
	wg.Wait()
	if tsErr != nil {
		return nil, fmt.Errorf("failed to create one or more token sources: %w", tsErr)
	}

	return &cachedTokenProvider{
		tokenSources: sources,
	}, nil
}

func (t *cachedTokenProvider) getToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	if src, containsKey := t.tokenSources[endpoint]; containsKey {
		return src.Token()
	}
	return nil, fmt.Errorf("endpoint is not configured: %s", endpoint)
}

func newGoogleProvider(ctx context.Context, endpoints []string) (*cachedTokenProvider, error) {
	return newTokenProvider(ctx, endpoints, func(ctx context.Context, endpoint string) (oauth2.TokenSource, error) {
		return idtoken.NewTokenSource(ctx, endpoint)
	})
}

type nilTokenProvider struct{}

func (t *nilTokenProvider) getToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	// no-op
	return nil, nil
}
