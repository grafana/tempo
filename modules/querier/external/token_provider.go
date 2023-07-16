package external

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

type tokenProvider interface {
	// If this returns nil, the request will be unauthenticated.
	getToken(ctx context.Context, endpoint string) (*oauth2.Token, error)

	// Used by the cachedTokenProvider to manage TokenSources internally.
	getTokenSource(ctx context.Context, endpoint string) (oauth2.TokenSource, error)
}

// Caches an oauth2.TokenSource to enable efficient auth on each of our external
// endpoints.
type cachedTokenProvider struct {
	tokenSources map[string]oauth2.TokenSource
}

func newCachedTokenProvider(
	ctx context.Context,
	endpoints []string,
	provider tokenProvider,
) (*cachedTokenProvider, error) {
	sources := make(map[string]oauth2.TokenSource, len(endpoints))

	var mtx sync.Mutex
	var wg sync.WaitGroup

	var tsErr error
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()
			ts, err := provider.getTokenSource(ctx, ep)

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

// Returns an oauth2 token, leveraging a cache unless the token is expired.
// If expired, the token is renewed and added to the cache.
func (t *cachedTokenProvider) getToken(_ context.Context, endpoint string) (*oauth2.Token, error) {
	if src, containsKey := t.tokenSources[endpoint]; containsKey {
		return src.Token()
	}
	return nil, fmt.Errorf("endpoint is not configured: %s", endpoint)
}

func (t *cachedTokenProvider) getTokenSource(_ context.Context, _ string) (oauth2.TokenSource, error) {
	return nil, fmt.Errorf("getting token source from cached provider is unimplemented")
}

type googleTokenProvider struct{}

// Ideally this function shouldn't be called - we should instead call it through
// the cached token provider instead.
func (t *googleTokenProvider) getToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	s, err := idtoken.NewTokenSource(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to create the token source")
	}
	return s.Token()
}

func (t *googleTokenProvider) getTokenSource(ctx context.Context, endpoint string) (oauth2.TokenSource, error) {
	return idtoken.NewTokenSource(ctx, endpoint)
}

type nilTokenProvider struct{}

func (t *nilTokenProvider) getToken(_ context.Context, _ string) (*oauth2.Token, error) {
	// no-op
	return nil, nil
}

func (t *nilTokenProvider) getTokenSource(_ context.Context, _ string) (oauth2.TokenSource, error) {
	return nil, fmt.Errorf("getting token source from nilTokenProvider is unimplemented")
}
