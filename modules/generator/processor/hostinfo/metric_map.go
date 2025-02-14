package hostinfo

import (
	"github.com/grafana/tempo/modules/generator/registry"
	"sync"
)

type metricMap struct {
	mutex  sync.RWMutex
	metric string
	values map[string]struct{}
}

func newMetricMap(metricName string) *metricMap {
	return &metricMap{
		metric: metricName,
		values: make(map[string]struct{}),
	}
}

func (m *metricMap) add(value string) {
	m.mutex.RLock()
	if _, ok := m.values[value]; !ok {
		m.mutex.RUnlock()
		m.mutex.Lock()
		defer m.mutex.Unlock()
		m.values[value] = struct{}{}
	} else {
		m.mutex.RUnlock()
	}
}

func (m *metricMap) reset() {
	if len(m.values) > 0 {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		m.values = make(map[string]struct{})
	}
}

func (m *metricMap) register(reg registry.Registry) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metric := reg.NewGauge(m.metric)
	count := len(m.values)
	if count > 0 {
		for k, _ := range m.values {
			labelValues := reg.NewLabelValueCombo([]string{hostIdentifierAttr}, []string{k})
			metric.Set(labelValues, 1)
		}
	}
}
