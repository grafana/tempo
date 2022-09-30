// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package basic // import "go.opentelemetry.io/otel/sdk/metric/controller/basic"

import (
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric/export"
	"go.opentelemetry.io/otel/sdk/resource"
)

// config contains configuration for a basic Controller.
type config struct {
	// Resource is the OpenTelemetry resource associated with all Meters
	// created by the Controller.
	Resource *resource.Resource

	// CollectPeriod is the interval between calls to Collect a
	// checkpoint.
	//
	// When pulling metrics and not exporting, this is the minimum
	// time between calls to Collect.  In a pull-only
	// configuration, collection is performed on demand; set
	// CollectPeriod to 0 always recompute the export record set.
	//
	// When exporting metrics, this must be > 0.
	//
	// Default value is 10s.
	CollectPeriod time.Duration

	// CollectTimeout is the timeout of the Context passed to
	// Collect() and subsequently to Observer instrument callbacks.
	//
	// Default value is 10s.  If zero, no Collect timeout is applied.
	CollectTimeout time.Duration

	// Exporter is used for exporting metric data.
	//
	// Note: Exporters such as Prometheus that pull data do not implement
	// export.Exporter.  These will directly call Collect() and ForEach().
	Exporter export.Exporter

	// PushTimeout is the timeout of the Context when a exporter is configured.
	//
	// Default value is 10s.  If zero, no Export timeout is applied.
	PushTimeout time.Duration
}

// Option is the interface that applies the value to a configuration option.
type Option interface {
	// apply sets the Option value of a Config.
	apply(config) config
}

// WithResource sets the Resource configuration option of a Config by merging it
// with the Resource configuration in the environment.
func WithResource(r *resource.Resource) Option {
	return resourceOption{r}
}

type resourceOption struct{ *resource.Resource }

func (o resourceOption) apply(cfg config) config {
	res, err := resource.Merge(cfg.Resource, o.Resource)
	if err != nil {
		otel.Handle(err)
	}
	cfg.Resource = res
	return cfg
}

// WithCollectPeriod sets the CollectPeriod configuration option of a Config.
func WithCollectPeriod(period time.Duration) Option {
	return collectPeriodOption(period)
}

type collectPeriodOption time.Duration

func (o collectPeriodOption) apply(cfg config) config {
	cfg.CollectPeriod = time.Duration(o)
	return cfg
}

// WithCollectTimeout sets the CollectTimeout configuration option of a Config.
func WithCollectTimeout(timeout time.Duration) Option {
	return collectTimeoutOption(timeout)
}

type collectTimeoutOption time.Duration

func (o collectTimeoutOption) apply(cfg config) config {
	cfg.CollectTimeout = time.Duration(o)
	return cfg
}

// WithExporter sets the exporter configuration option of a Config.
func WithExporter(exporter export.Exporter) Option {
	return exporterOption{exporter}
}

type exporterOption struct{ exporter export.Exporter }

func (o exporterOption) apply(cfg config) config {
	cfg.Exporter = o.exporter
	return cfg
}

// WithPushTimeout sets the PushTimeout configuration option of a Config.
func WithPushTimeout(timeout time.Duration) Option {
	return pushTimeoutOption(timeout)
}

type pushTimeoutOption time.Duration

func (o pushTimeoutOption) apply(cfg config) config {
	cfg.PushTimeout = time.Duration(o)
	return cfg
}
