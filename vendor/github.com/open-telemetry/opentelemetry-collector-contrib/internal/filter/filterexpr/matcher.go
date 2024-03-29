// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterexpr // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterexpr"

import (
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var vmPool = sync.Pool{
	New: func() any {
		return &vm.VM{}
	},
}

type Matcher struct {
	program *vm.Program
}

type env struct {
	MetricName string
	MetricType string
	attributes pcommon.Map
}

func (e *env) HasLabel(key string) bool {
	_, ok := e.attributes.Get(key)
	return ok
}

func (e *env) Label(key string) string {
	v, _ := e.attributes.Get(key)
	return v.Str()
}

func NewMatcher(expression string) (*Matcher, error) {
	program, err := expr.Compile(expression)
	if err != nil {
		return nil, err
	}
	return &Matcher{program: program}, nil
}

func (m *Matcher) MatchMetric(metric pmetric.Metric) (bool, error) {
	metricName := metric.Name()
	vm := vmPool.Get().(*vm.VM)
	defer vmPool.Put(vm)
	//exhaustive:enforce
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return m.matchGauge(metricName, metric.Gauge(), vm)
	case pmetric.MetricTypeSum:
		return m.matchSum(metricName, metric.Sum(), vm)
	case pmetric.MetricTypeHistogram:
		return m.matchHistogram(metricName, metric.Histogram(), vm)
	case pmetric.MetricTypeExponentialHistogram:
		return m.matchExponentialHistogram(metricName, metric.ExponentialHistogram(), vm)
	case pmetric.MetricTypeSummary:
		return m.matchSummary(metricName, metric.Summary(), vm)
	default:
		return false, nil
	}
}

func (m *Matcher) matchGauge(metricName string, gauge pmetric.Gauge, vm *vm.VM) (bool, error) {
	pts := gauge.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		matched, err := m.matchEnv(metricName, pmetric.MetricTypeGauge, pts.At(i).Attributes(), vm)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchSum(metricName string, sum pmetric.Sum, vm *vm.VM) (bool, error) {
	pts := sum.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		matched, err := m.matchEnv(metricName, pmetric.MetricTypeSum, pts.At(i).Attributes(), vm)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchHistogram(metricName string, histogram pmetric.Histogram, vm *vm.VM) (bool, error) {
	pts := histogram.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		matched, err := m.matchEnv(metricName, pmetric.MetricTypeHistogram, pts.At(i).Attributes(), vm)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchExponentialHistogram(metricName string, eh pmetric.ExponentialHistogram, vm *vm.VM) (bool, error) {
	pts := eh.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		matched, err := m.matchEnv(metricName, pmetric.MetricTypeExponentialHistogram, pts.At(i).Attributes(), vm)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchSummary(metricName string, summary pmetric.Summary, vm *vm.VM) (bool, error) {
	pts := summary.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		matched, err := m.matchEnv(metricName, pmetric.MetricTypeSummary, pts.At(i).Attributes(), vm)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchEnv(metricName string, metricType pmetric.MetricType, attributes pcommon.Map, vm *vm.VM) (bool, error) {
	return m.match(env{
		MetricName: metricName,
		MetricType: metricType.String(),
		attributes: attributes,
	}, vm)
}

func (m *Matcher) match(env env, vm *vm.VM) (bool, error) {
	result, err := vm.Run(m.program, &env)
	if err != nil {
		return false, err
	}

	v, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("filter returned non-boolean value type=%T result=%v metric=%s, attributes=%v",
			result, result, env.MetricName, env.attributes.AsRaw())
	}

	return v, nil
}
