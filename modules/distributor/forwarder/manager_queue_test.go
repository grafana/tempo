package forwarder

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestManager_QueueConfig(t *testing.T) {
	o := &mockWorkingOverrides{
		tenantIDToForwarders: map[string][]string{
			"default": {"first", "second"},
			"custom":  {"first", "second"},
		},
		tenantIDToQueueSize: map[string]int{"custom": 17},
		tenantIDToWorkers:   map[string]int{"custom": 3},
	}
	manager := newManagerWithForwarders(t, map[string]Forwarder{
		"first": &mockWorkingForwarder{}, "second": &mockWorkingForwarder{},
	}, log.NewNopLogger(), o)

	for _, tc := range []struct {
		tenant        string
		size, workers int
	}{
		{"default", 100, 2},
		{"custom", 17, 3},
	} {
		t.Run(tc.tenant, func(t *testing.T) {
			require.Len(t, manager.ForTenant(tc.tenant), 2)
			ql, found := manager.getQueueList(tc.tenant)
			require.True(t, found)
			for _, q := range ql.forwarderNameToQueue {
				require.Equal(t, tc.size, q.Size())
				require.Equal(t, tc.workers, q.WorkerCount())
			}
		})
	}
}

func TestManager_QueueConfigRefresh(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		initialSize, initialWorkers int
		updatedSize, updatedWorkers int
		wantSize, wantWorkers       int
		replace                     bool
	}{
		{"size only", 3, 1, 7, 1, 7, 1, true},
		{"workers only", 3, 1, 3, 2, 3, 2, true},
		{"unchanged", 3, 1, 3, 1, 3, 1, false},
		{"explicit defaults", 0, 0, 100, 2, 100, 2, false},
		{"reset defaults", 3, 1, 0, 0, 100, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &mockWorkingOverrides{
				tenantIDToForwarders: map[string][]string{
					"updated": {"external"}, "unchanged": {"external"},
				},
				tenantIDToQueueSize: map[string]int{"updated": tc.initialSize, "unchanged": 11},
				tenantIDToWorkers:   map[string]int{"updated": tc.initialWorkers, "unchanged": 1},
			}
			traces := make(chan ptrace.Traces, 1)
			manager := newManagerWithForwarders(t, map[string]Forwarder{
				"external": &mockChannelledInterceptorForwarder{next: &mockWorkingForwarder{}, traces: traces},
			}, log.NewNopLogger(), o)
			require.Len(t, manager.ForTenant("updated"), 1)
			require.Len(t, manager.ForTenant("unchanged"), 1)
			oldList, _ := manager.getQueueList("updated")
			unaffectedList, _ := manager.getQueueList("unchanged")

			o.tenantIDToQueueSize["updated"] = tc.updatedSize
			o.tenantIDToWorkers["updated"] = tc.updatedWorkers
			manager.updateQueueLists()

			newList, found := manager.getQueueList("updated")
			require.True(t, found)
			if tc.replace {
				require.NotSame(t, oldList, newList)
			} else {
				require.Same(t, oldList, newList)
			}
			require.Equal(t, tc.wantSize, newList.forwarderNameToQueue["external"].Size())
			require.Equal(t, tc.wantWorkers, newList.forwarderNameToQueue["external"].WorkerCount())
			gotUnaffected, _ := manager.getQueueList("unchanged")
			require.Same(t, unaffectedList, gotUnaffected)

			batch := ptrace.NewTraces()
			require.NoError(t, manager.ForTenant("updated").ForwardTraces(context.Background(), batch))
			select {
			case got := <-traces:
				require.Equal(t, batch, got)
			case <-time.After(5 * time.Second):
				t.Fatal("replacement queue did not forward the batch")
			}
		})
	}
}

type drainingTestForwarder struct {
	started chan struct{}
	release chan struct{}
	traces  chan ptrace.Traces
}

func (f *drainingTestForwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	if traces.ResourceSpans().At(0).SchemaUrl() == "blocked" {
		close(f.started)
		select {
		case <-f.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case f.traces <- traces:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (*drainingTestForwarder) Shutdown(context.Context) error { return nil }

func TestManager_QueueConfigRefreshDrainsOldQueue(t *testing.T) {
	o := &mockWorkingOverrides{
		tenantIDToForwarders: map[string][]string{"tenant": {"external"}},
		tenantIDToQueueSize:  map[string]int{"tenant": 2},
		tenantIDToWorkers:    map[string]int{"tenant": 1},
	}
	forwarder := &drainingTestForwarder{
		started: make(chan struct{}), release: make(chan struct{}), traces: make(chan ptrace.Traces, 3),
	}
	manager := newManagerWithForwarders(t, map[string]Forwarder{"external": forwarder}, log.NewNopLogger(), o)
	// Unblock the old worker before manager cleanup even if an assertion fails.
	defer close(forwarder.release)
	batch := func(name string) ptrace.Traces {
		traces := ptrace.NewTraces()
		traces.ResourceSpans().AppendEmpty().SetSchemaUrl(name)
		return traces
	}
	receive := func() string {
		t.Helper()
		select {
		case traces := <-forwarder.traces:
			return traces.ResourceSpans().At(0).SchemaUrl()
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for forwarded batch")
			return ""
		}
	}

	require.NoError(t, manager.ForTenant("tenant").ForwardTraces(context.Background(), batch("blocked")))
	select {
	case <-forwarder.started:
	case <-time.After(5 * time.Second):
		t.Fatal("old worker did not start")
	}
	require.NoError(t, manager.ForTenant("tenant").ForwardTraces(context.Background(), batch("queued")))

	o.tenantIDToQueueSize["tenant"] = 4
	manager.updateQueueLists()
	require.NoError(t, manager.ForTenant("tenant").ForwardTraces(context.Background(), batch("new")))
	require.Equal(t, "new", receive(), "new queue must forward while the old worker is blocked")

	forwarder.release <- struct{}{}
	require.Equal(t, "blocked", receive())
	require.Equal(t, "queued", receive(), "old queue must drain its buffered batch")
}
