package forwarder

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/multierr"
	"golang.org/x/exp/constraints"

	"github.com/grafana/tempo/modules/distributor/queue"
)

const (
	defaultWorkerCount = 2
	defaultQueueSize   = 100
)

type Overrides interface {
	TenantIDs() []string
	Forwarders(tenantID string) []string
}

type Manager struct {
	services.Service
	logger    log.Logger
	overrides Overrides

	// forwarderNameToForwarder is static throughout lifecycle of the manager and read-only
	forwarderNameToForwarder map[string]Forwarder

	tenantToQueueList   map[string]*queueList
	tenantToQueueListMu *sync.RWMutex
}

func NewManager(cfgs ConfigList, logger log.Logger, overrides Overrides) (*Manager, error) {
	if err := cfgs.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config list: %w", err)
	}

	forwarderNameToForwarder := make(map[string]Forwarder, len(cfgs))
	for i, cfg := range cfgs {
		forwarder, err := New(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create forwarder for cfg at index=%d: %w", i, err)
		}

		forwarderNameToForwarder[cfg.Name] = forwarder
	}

	m := &Manager{
		logger:                   logger,
		overrides:                overrides,
		forwarderNameToForwarder: forwarderNameToForwarder,
		tenantToQueueList:        make(map[string]*queueList),
		tenantToQueueListMu:      &sync.RWMutex{},
	}

	m.Service = services.NewBasicService(m.start, m.run, m.stop)

	return m, nil
}

func (m *Manager) ForTenant(tenantID string) List {
	list, found := m.getQueueList(tenantID)
	if !found {
		return nil
	}

	return list.copy()
}

func (m *Manager) start(_ context.Context) error {
	tenantToForwarderNames := m.readOverrides()
	m.updateTenantToForwarderList(tenantToForwarderNames)

	return nil
}

func (m *Manager) run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := m.shutdown(); err != nil {
				return fmt.Errorf("failed to shutdown: %w", err)
			}

			return nil
		case <-ticker.C:
			m.handleTick()
		}
	}
}

func (m *Manager) stop(err error) error {
	if err != nil {
		_ = level.Warn(m.logger).Log("msg", "manager returned error from running state", "err", err)
	}

	return nil
}

func (m *Manager) handleTick() {
	tenantToForwarderNames := m.readOverrides()
	m.updateTenantToForwarderList(tenantToForwarderNames)
}

// readOverrides returns a mapping between tenant ID and a
// list of requested forwarders.
func (m *Manager) readOverrides() map[string][]string {
	result := make(map[string][]string)

	for _, tenantID := range m.overrides.TenantIDs() {
		names := m.overrides.Forwarders(tenantID)
		if len(names) < 1 {
			continue
		}

		result[tenantID] = names
	}

	return result
}

func (m *Manager) updateTenantToForwarderList(tenantToForwarderNames map[string][]string) {
	m.tenantToQueueListMu.Lock()
	defer m.tenantToQueueListMu.Unlock()

	oldTenantIDs := make([]string, 0, len(m.tenantToQueueList))
	for oldTenantID := range m.tenantToQueueList {
		oldTenantIDs = append(oldTenantIDs, oldTenantID)
	}

	newTenantIDs := make([]string, 0, len(tenantToForwarderNames))
	for newTenantID := range tenantToForwarderNames {
		newTenantIDs = append(newTenantIDs, newTenantID)
	}

	diff := diff(oldTenantIDs, newTenantIDs)

	for _, removedTenantID := range diff.removed {
		m.removeQueueListForTenantUnderLock(removedTenantID)
	}

	for _, addedTenantID := range diff.added {
		m.updateQueueListForTenantUnderLock(addedTenantID, tenantToForwarderNames[addedTenantID])
	}

	// Go through all tenants and try to update them.
	// If forwarder names for particular tenant changed, this will update them.
	// If names didn't change, this is a noop.
	for tenantID := range m.tenantToQueueList {
		m.updateQueueListForTenantUnderLock(tenantID, tenantToForwarderNames[tenantID])
	}
}

func (m *Manager) removeQueueListForTenantUnderLock(tenantID string) {
	ql, found := m.tenantToQueueList[tenantID]
	if !found {
		_ = level.Warn(m.logger).Log("msg", "queue list not found", "tenantID", tenantID)
		return
	}

	if err := ql.shutdown(context.TODO()); err != nil {
		_ = level.Warn(m.logger).Log("msg", "failed to shutdown queue list", "tenantID", tenantID)
	}

	delete(m.tenantToQueueList, tenantID)
}

// updateQueueListForTenantUnderLock should only be called under tenantToQueueListMu write lock.
func (m *Manager) updateQueueListForTenantUnderLock(tenantID string, forwarderNames []string) {
	queueList, found := m.tenantToQueueList[tenantID]
	if !found {
		queueList = newQueueList(m.logger, tenantID)
		m.tenantToQueueList[tenantID] = queueList
	}

	queueList.update(forwarderNames, m.forwarderNameToForwarder)
}

func (m *Manager) getQueueList(tenantID string) (*queueList, bool) {
	m.tenantToQueueListMu.RLock()
	defer m.tenantToQueueListMu.RUnlock()

	queueList, found := m.tenantToQueueList[tenantID]

	return queueList, found
}

func (m *Manager) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m.tenantToQueueListMu.Lock()
	defer m.tenantToQueueListMu.Unlock()

	var errs []error
	for tenantID, ql := range m.tenantToQueueList {
		if err := ql.shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown queuelist for tenantID=%s: %w", tenantID, err))
		}
	}

	m.tenantToQueueList = make(map[string]*queueList)

	for forwarderName, forwarder := range m.forwarderNameToForwarder {
		if err := forwarder.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown forwarder with name=%s: %w", forwarderName, err))
		}
	}

	m.forwarderNameToForwarder = make(map[string]Forwarder)

	return multierr.Combine(errs...)
}

type queueListDiff[T any] struct {
	added   []T
	removed []T
}

type queueList struct {
	logger               log.Logger
	tenantID             string
	forwarderNameToQueue map[string]*queue.Queue[ptrace.Traces]
	mu                   *sync.RWMutex
}

func newQueueList(logger log.Logger, tenantID string) *queueList {
	return &queueList{
		logger:               logger,
		tenantID:             tenantID,
		forwarderNameToQueue: make(map[string]*queue.Queue[ptrace.Traces]),
		mu:                   &sync.RWMutex{},
	}
}

func (l *queueList) copy() List {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(List, 0, len(l.forwarderNameToQueue))
	for _, q := range l.forwarderNameToQueue {
		result = append(result, &queueAdapter{queue: q})
	}

	return result
}

func (l *queueList) update(forwarderNames []string, forwarderNameToForwarder map[string]Forwarder) {
	l.mu.Lock()
	defer l.mu.Unlock()

	oldForwarderNames := make([]string, 0, len(l.forwarderNameToQueue))
	for oldForwarderName := range l.forwarderNameToQueue {
		oldForwarderNames = append(oldForwarderNames, oldForwarderName)
	}

	diff := diff(oldForwarderNames, forwarderNames)

	for _, removedForwarderName := range diff.removed {
		q, found := l.forwarderNameToQueue[removedForwarderName]
		if !found {
			_ = level.Warn(l.logger).Log("msg", "queue not found", "removedForwarderName", removedForwarderName)
			continue
		}

		// TODO: Consider making shutdown asynchronous not to block here
		if err := q.Shutdown(context.TODO()); err != nil {
			_ = level.Warn(l.logger).Log("msg", "failed to shutdown queue", "removedForwarderName", removedForwarderName, "tenantID", l.tenantID)
		}

		delete(l.forwarderNameToQueue, removedForwarderName)
	}

	for _, addedForwarderName := range diff.added {
		forwarder, found := forwarderNameToForwarder[addedForwarderName]
		if !found {
			_ = level.Warn(l.logger).Log("msg", "failed to find forwarder by name", "addedForwarderName", addedForwarderName, "tenantID", l.tenantID)
			continue
		}

		queueCfg := queue.Config{
			Name:        addedForwarderName,
			TenantID:    l.tenantID,
			Size:        defaultQueueSize,
			WorkerCount: defaultWorkerCount,
		}

		processFunc := func(ctx context.Context, traces ptrace.Traces) {
			if err := forwarder.ForwardTraces(ctx, traces); err != nil {
				_ = level.Warn(l.logger).Log("msg", "failed to forward batches", "forwarderName", addedForwarderName, "tenantID", l.tenantID, "err", err)
			}
		}
		newQueue := queue.New(queueCfg, l.logger, processFunc)
		newQueue.StartWorkers()
		l.forwarderNameToQueue[addedForwarderName] = newQueue
	}
}

func (l *queueList) shutdown(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for forwarderName, q := range l.forwarderNameToQueue {
		if err := q.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown queue for forwarder=%s: %w", forwarderName, err))
		}

		delete(l.forwarderNameToQueue, forwarderName)
	}

	return multierr.Combine(errs...)
}

type queueAdapter struct {
	queue *queue.Queue[ptrace.Traces]
}

func (a *queueAdapter) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	return a.queue.Push(ctx, traces)
}

// Shutdown does nothing. Queue lifecycle is handled by queueList.
func (a *queueAdapter) Shutdown(_ context.Context) error {
	return nil
}

// diff returns the difference between two lists.
func diff[T constraints.Ordered](oldList, newList []T) queueListDiff[T] {
	newSet := make(map[T]struct{}, len(newList))
	for _, n := range newList {
		newSet[n] = struct{}{}
	}

	oldSet := make(map[T]struct{}, len(oldList))
	for _, o := range oldList {
		oldSet[o] = struct{}{}
	}

	diff := queueListDiff[T]{}
	for _, o := range oldList {
		if _, found := newSet[o]; !found {
			diff.removed = append(diff.removed, o)
		}
	}

	for _, n := range newList {
		if _, found := oldSet[n]; !found {
			diff.added = append(diff.added, n)
		}
	}

	return diff
}
