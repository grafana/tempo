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
	m.tenantToQueueListMu.RLock()
	defer m.tenantToQueueListMu.RUnlock()

	ql, found := m.tenantToQueueList[tenantID]
	if !found {
		return nil
	}

	return ql.list
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

	queueListsToAdd := make(map[string]*queueList)

	for tenantID, forwarderNames := range tenantToForwarderNames {
		if _, found := m.tenantToQueueList[tenantID]; !found {
			queueListsToAdd[tenantID] = newQueueList(m.logger, tenantID, forwarderNames, m.forwarderNameToForwarder)
		}
	}

	for tenantID, ql := range m.tenantToQueueList {
		forwarderNames, found := tenantToForwarderNames[tenantID]
		if !found {
			if err := ql.shutdown(context.Background()); err != nil {
				_ = level.Warn(m.logger).Log("msg", "failed to shutdown queue list", "tenantID", tenantID)
			}

			delete(m.tenantToQueueList, tenantID)

			continue
		}

		if ql.shouldUpdate(forwarderNames) {
			if err := ql.shutdown(context.Background()); err != nil {
				_ = level.Warn(m.logger).Log("msg", "failed to shutdown queue list", "tenantID", tenantID)
			}

			delete(m.tenantToQueueList, tenantID)

			queueListsToAdd[tenantID] = newQueueList(m.logger, tenantID, forwarderNames, m.forwarderNameToForwarder)
		}
	}

	for tenantID, ql := range queueListsToAdd {
		m.tenantToQueueList[tenantID] = ql
	}
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

type queueList struct {
	logger               log.Logger
	tenantID             string
	forwarderNameToQueue map[string]*queue.Queue[ptrace.Traces]
	list                 List
}

func newQueueList(logger log.Logger, tenantID string, forwarderNames []string, forwarderNameToForwarder map[string]Forwarder) *queueList {
	forwarderNameToQueue := make(map[string]*queue.Queue[ptrace.Traces], len(forwarderNames))
	list := make(List, 0, len(forwarderNames))
	for _, forwarderName := range forwarderNames {
		forwarder, found := forwarderNameToForwarder[forwarderName]
		if !found {
			_ = level.Warn(logger).Log("msg", "failed to find forwarder by name", "forwarderName", forwarderName, "tenantID", tenantID)
			continue
		}

		queueCfg := queue.Config{
			Name:        forwarderName,
			TenantID:    tenantID,
			Size:        defaultQueueSize,
			WorkerCount: defaultWorkerCount,
		}

		processFunc := func(ctx context.Context, traces ptrace.Traces) {
			if err := forwarder.ForwardTraces(ctx, traces); err != nil {
				_ = level.Warn(logger).Log("msg", "failed to forward batches", "forwarderName", forwarderName, "tenantID", tenantID, "err", err)
			}
		}
		newQueue := queue.New(queueCfg, logger, processFunc)
		newQueue.StartWorkers()
		forwarderNameToQueue[forwarderName] = newQueue
		list = append(list, queueAdapter{queue: newQueue})
	}

	return &queueList{
		logger:               logger,
		tenantID:             tenantID,
		forwarderNameToQueue: forwarderNameToQueue,
		list:                 list,
	}
}

func (l *queueList) shouldUpdate(forwarderNames []string) bool {
	if len(forwarderNames) != len(l.forwarderNameToQueue) {
		return true
	}

	for _, forwarderName := range forwarderNames {
		if _, found := l.forwarderNameToQueue[forwarderName]; !found {
			return true
		}
	}

	return false
}

func (l *queueList) shutdown(ctx context.Context) error {
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

func (a queueAdapter) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	return a.queue.Push(ctx, traces)
}

// Shutdown does nothing. Queue lifecycle is handled by queueList.
func (a queueAdapter) Shutdown(_ context.Context) error {
	return nil
}
