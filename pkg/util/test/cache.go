package test

import (
	"context"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/cache"
)

type mockClient struct {
	client map[string][]byte
}

func (m *mockClient) Store(_ context.Context, key []string, val [][]byte) {
	m.client[key[0]] = val[0]
}

func (m *mockClient) Fetch(_ context.Context, key []string) (found []string, bufs [][]byte, missing []string) {
	val, ok := m.client[key[0]]
	if ok {
		found = append(found, key[0])
		bufs = append(bufs, val)
	} else {
		missing = append(missing, key[0])
	}
	return
}

func (m *mockClient) Stop() {
}

// NewMockClient makes a new mockClient.
func NewMockClient() cache.Cache {
	return &mockClient{
		client: map[string][]byte{},
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

func (p *mockProvider) AddCache(_ cache.Role, _ cache.Cache) error {
	return nil
}
