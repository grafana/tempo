// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"errors"

	"go.uber.org/zap"

	"github.com/jaegertracing/jaeger/pkg/distributedlock"
	"github.com/jaegertracing/jaeger/pkg/metrics"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	metricsstore "github.com/jaegertracing/jaeger/storage/metricsstore"
	"github.com/jaegertracing/jaeger/storage/samplingstore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

// Factory defines an interface for a factory that can create implementations of different storage components.
// Implementations are also encouraged to implement plugin.Configurable interface.
//
// # See also
//
// plugin.Configurable
type Factory interface {
	// Initialize performs internal initialization of the factory, such as opening connections to the backend store.
	// It is called after all configuration of the factory itself has been done.
	Initialize(metricsFactory metrics.Factory, logger *zap.Logger) error

	// CreateSpanReader creates a spanstore.Reader.
	CreateSpanReader() (spanstore.Reader, error)

	// CreateSpanWriter creates a spanstore.Writer.
	CreateSpanWriter() (spanstore.Writer, error)

	// CreateDependencyReader creates a dependencystore.Reader.
	CreateDependencyReader() (dependencystore.Reader, error)
}

// SamplingStoreFactory defines an interface that is capable of returning the necessary backends for
// adaptive sampling.
type SamplingStoreFactory interface {
	// CreateLock creates a distributed lock.
	CreateLock() (distributedlock.Lock, error)
	// CreateSamplingStore creates a sampling store.
	CreateSamplingStore(maxBuckets int) (samplingstore.Store, error)
}

var (
	// ErrArchiveStorageNotConfigured can be returned by the ArchiveFactory when the archive storage is not configured.
	ErrArchiveStorageNotConfigured = errors.New("archive storage not configured")

	// ErrArchiveStorageNotSupported can be returned by the ArchiveFactory when the archive storage is not supported by the backend.
	ErrArchiveStorageNotSupported = errors.New("archive storage not supported")
)

// ArchiveFactory is an additional interface that can be implemented by a factory to support trace archiving.
type ArchiveFactory interface {
	// CreateArchiveSpanReader creates a spanstore.Reader.
	CreateArchiveSpanReader() (spanstore.Reader, error)

	// CreateArchiveSpanWriter creates a spanstore.Writer.
	CreateArchiveSpanWriter() (spanstore.Writer, error)
}

// MetricsFactory defines an interface for a factory that can create implementations of different metrics storage components.
// Implementations are also encouraged to implement plugin.Configurable interface.
//
// # See also
//
// plugin.Configurable
type MetricsFactory interface {
	// Initialize performs internal initialization of the factory, such as opening connections to the backend store.
	// It is called after all configuration of the factory itself has been done.
	Initialize(logger *zap.Logger) error

	// CreateMetricsReader creates a metricsstore.Reader.
	CreateMetricsReader() (metricsstore.Reader, error)
}
