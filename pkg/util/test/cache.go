package test

import (
	"context"
	"sync"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/cache"
)

type mockClient struct {
	sync.Mutex
	cache map[string][]byte
}

func (m *mockClient) Store(_ context.Context, keys []string, bufs [][]byte) {
	m.Lock()
	defer m.Unlock()
	for i := range keys {
		m.cache[keys[i]] = bufs[i]
	}
}

func (m *mockClient) Fetch(_ context.Context, keys []string) (found []string, bufs [][]byte, missing []string) {
	m.Lock()
	defer m.Unlock()
	for _, key := range keys {
		buf, ok := m.cache[key]
		if ok {
			found = append(found, key)
			bufs = append(bufs, buf)
		} else {
			missing = append(missing, key)
		}
	}
	return
}

func (m *mockClient) FetchKey(_ context.Context, key string) (buf []byte, found bool) {
	m.Lock()
	defer m.Unlock()
	buf, ok := m.cache[key]
	if ok {
		return buf, true
	}
	return buf, false
}

func (m *mockClient) MaxItemSize() int {
	return 0
}

func (m *mockClient) Stop() {
}

// NewMockClient makes a new mockClient.
func NewMockClient() cache.Cache {
	return &mockClient{
		cache: map[string][]byte{},
	}
}

func NewMockProvider() cache.Provider {
	return &mockProvider{
		c: NewMockClient(),
	}
}

type mockProvider struct {
	services.Service

	c cache.Cache
}

func (p *mockProvider) CacheFor(_ cache.Role) cache.Cache {
	return p.c
}

func (p *mockProvider) AddCache(_ cache.Role, c cache.Cache) error {
	p.c = c
	return nil
}
