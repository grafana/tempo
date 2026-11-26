package overrides

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/grafana/tempo/v3/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/v3/pkg/sharedconfig"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type presentationTestSource struct {
	Service
	current atomic.Pointer[Overrides]
}

func (s *presentationTestSource) serviceGraphPresentationState(string) (*Overrides, []string, []string) {
	snapshot := s.current.Load()
	cfg := snapshot.MetricsGenerator.Processor.ServiceGraphs
	return snapshot, cfg.BypassServices, cfg.BypassServiceRegex
}

func (s *presentationTestSource) MetricsGeneratorProcessorServiceGraphsPresentation(tenant string) ([]string, []string) {
	_, names, patterns := s.serviceGraphPresentationState(tenant)
	return names, patterns
}

func newPresentationTestManager(t *testing.T) (*userConfigurableOverridesManager, *presentationTestSource, *Overrides, *Overrides) {
	t.Helper()
	base, err := newRuntimeConfigOverrides(Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	empty := &Overrides{}
	withRegex := &Overrides{MetricsGenerator: MetricsGeneratorOverrides{Processor: ProcessorOverrides{ServiceGraphs: ServiceGraphsOverrides{BypassServiceRegex: []string{"proxy"}}}}}
	source := &presentationTestSource{Service: base}
	source.current.Store(empty)
	cfg := &UserConfigurableOverridesConfig{Client: client.Config{Backend: backend.Local, Local: &local.Config{Path: t.TempDir()}}}
	manager, err := newUserConfigOverrides(cfg, source)
	require.NoError(t, err)
	return manager, source, empty, withRegex
}

func presentationTestLimits(count int) *client.Limits {
	names := strings.Fields(strings.Repeat("gateway ", count))
	return &client.Limits{MetricsGenerator: client.LimitsMetricsGenerator{Processor: client.LimitsMetricsGeneratorProcessor{ServiceGraphs: client.LimitsMetricsGeneratorProcessorServiceGraphs{BypassServices: &names}}}}
}

// Reloads seed the last valid pair before queries arrive. Cache entries are
// bounded by saved tenants, and removing saved settings releases their snapshot.
func TestPresentationSnapshotRetentionAndBounds(t *testing.T) {
	manager, source, _, changed := newPresentationTestManager(t)
	manager.setTenantLimit("tenant", presentationTestLimits(128))
	require.Len(t, manager.presentation, 1)
	source.current.Store(changed)
	names, patterns := manager.MetricsGeneratorProcessorServiceGraphsPresentation("tenant")
	require.Len(t, names, 128)
	require.Empty(t, patterns)
	require.NoError(t, sharedconfig.ValidateServiceGraphPresentation(names, patterns))
	for i := 0; i < 1000; i++ {
		manager.MetricsGeneratorProcessorServiceGraphsPresentation(fmt.Sprint(i))
	}
	require.Len(t, manager.presentation, 1, "unknown tenants must not create unbounded query caches")
	manager.setTenantLimit("tenant", nil)
	require.Empty(t, manager.presentation)
	_, patterns = manager.MetricsGeneratorProcessorServiceGraphsPresentation("tenant")
	require.Equal(t, []string{"proxy"}, patterns)
}

// Readers must receive a complete valid pair while runtime and API snapshots
// change concurrently, including transient combinations exceeding the rule limit.
func TestPresentationSnapshotsConcurrentReload(t *testing.T) {
	manager, source, empty, withRegex := newPresentationTestManager(t)
	full, partial := presentationTestLimits(128), presentationTestLimits(127)
	manager.setTenantLimit("tenant", full)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				names, patterns := manager.MetricsGeneratorProcessorServiceGraphsPresentation("tenant")
				if err := sharedconfig.ValidateServiceGraphPresentation(names, patterns); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	for i := 0; i < 200; i++ {
		source.current.Store(withRegex)
		manager.setTenantLimit("tenant", partial)
		source.current.Store(empty)
		manager.setTenantLimit("tenant", full)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
