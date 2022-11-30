// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package service handles the command-line, configuration, and runs the
// OpenTelemetry Collector.
package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/internal/grpclog"
)

// State defines Collector's state.
type State int

const (
	StateStarting State = iota
	StateRunning
	StateClosing
	StateClosed
)

const (
	// Deprecated: [v0.65.0] use StateStarting.
	Starting = StateStarting
	// Deprecated: [v0.65.0] use StateRunning.
	Running = StateRunning
	// Deprecated: [v0.65.0] use StateClosing.
	Closing = StateClosing
	// Deprecated: [v0.65.0] use StateClosed.
	Closed = StateClosed
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	}
	return "UNKNOWN"
}

// (Internal note) Collector Lifecycle:
// - New constructs a new Collector.
// - Run starts the collector.
// - Run calls setupConfigurationComponents to handle configuration.
//   If configuration parser fails, collector's config can be reloaded.
//   Collector can be shutdown if parser gets a shutdown error.
// - Run runs runAndWaitForShutdownEvent and waits for a shutdown event.
//   SIGINT and SIGTERM, errors, and (*Collector).Shutdown can trigger the shutdown events.
// - Upon shutdown, pipelines are notified, then pipelines and extensions are shut down.
// - Users can call (*Collector).Shutdown anytime to shut down the collector.

// Collector represents a server providing the OpenTelemetry Collector service.
type Collector struct {
	set CollectorSettings

	service *service
	state   *atomic.Int32

	// shutdownChan is used to terminate the collector.
	shutdownChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// New creates and returns a new instance of Collector.
func New(set CollectorSettings) (*Collector, error) {
	if set.ConfigProvider == nil {
		return nil, errors.New("invalid nil config provider")
	}

	if set.telemetry == nil {
		set.telemetry = newColTelemetry(featuregate.GetRegistry())
	}

	return &Collector{
		set:               set,
		state:             atomic.NewInt32(int32(StateStarting)),
		shutdownChan:      make(chan struct{}),
		signalsChannel:    make(chan os.Signal, 1),
		asyncErrorChannel: make(chan error),
	}, nil
}

// GetState returns current state of the collector server.
func (col *Collector) GetState() State {
	return State(col.state.Load())
}

// Shutdown shuts down the collector server.
func (col *Collector) Shutdown() {
	// Only shutdown if we're in a Running or Starting State else noop
	state := col.GetState()
	if state == StateRunning || state == StateStarting {
		defer func() {
			recover() // nolint:errcheck
		}()
		close(col.shutdownChan)
	}
}

// setupConfigurationComponents loads the config and starts the components. If all the steps succeeds it
// sets the col.service with the service currently running.
func (col *Collector) setupConfigurationComponents(ctx context.Context) error {
	col.setCollectorState(StateStarting)

	cfg, err := col.set.ConfigProvider.Get(ctx, col.set.Factories)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	col.service, err = newService(&settings{
		BuildInfo:         col.set.BuildInfo,
		Factories:         col.set.Factories,
		Config:            cfg,
		AsyncErrorChannel: col.asyncErrorChannel,
		LoggingOptions:    col.set.LoggingOptions,
		telemetry:         col.set.telemetry,
	})
	if err != nil {
		return err
	}

	if !col.set.SkipSettingGRPCLogger {
		grpclog.SetLogger(col.service.telemetrySettings.Logger, cfg.Service.Telemetry.Logs.Level)
	}

	if err = col.service.Start(ctx); err != nil {
		return multierr.Append(err, col.shutdownServiceAndTelemetry(ctx))
	}
	col.setCollectorState(StateRunning)
	return nil
}

func (col *Collector) reloadConfiguration(ctx context.Context) error {
	col.service.telemetrySettings.Logger.Warn("Config updated, restart service")
	col.setCollectorState(StateClosing)

	if err := col.service.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown the retiring config: %w", err)
	}

	if err := col.setupConfigurationComponents(ctx); err != nil {
		return fmt.Errorf("failed to setup configuration components: %w", err)
	}

	return nil
}

// Run starts the collector according to the given configuration, and waits for it to complete.
// Consecutive calls to Run are not allowed, Run shouldn't be called once a collector is shut down.
func (col *Collector) Run(ctx context.Context) error {
	if err := col.setupConfigurationComponents(ctx); err != nil {
		col.setCollectorState(StateClosed)
		return err
	}

	// Always notify with SIGHUP for configuration reloading.
	signal.Notify(col.signalsChannel, syscall.SIGHUP)

	// Only notify with SIGTERM and SIGINT if graceful shutdown is enabled.
	if !col.set.DisableGracefulShutdown {
		signal.Notify(col.signalsChannel, os.Interrupt, syscall.SIGTERM)
	}

LOOP:
	for {
		select {
		case err := <-col.set.ConfigProvider.Watch():
			if err != nil {
				col.service.telemetrySettings.Logger.Error("Config watch failed", zap.Error(err))
				break LOOP
			}

			if err = col.reloadConfiguration(ctx); err != nil {
				return err
			}
		case err := <-col.asyncErrorChannel:
			col.service.telemetrySettings.Logger.Error("Asynchronous error received, terminating process", zap.Error(err))
			break LOOP
		case s := <-col.signalsChannel:
			col.service.telemetrySettings.Logger.Info("Received signal from OS", zap.String("signal", s.String()))
			switch s {
			case syscall.SIGHUP:
				if err := col.reloadConfiguration(ctx); err != nil {
					return err
				}
			default:
				break LOOP
			}
		case <-col.shutdownChan:
			col.service.telemetrySettings.Logger.Info("Received shutdown request")
			break LOOP
		case <-ctx.Done():
			col.service.telemetrySettings.Logger.Info("Context done, terminating process", zap.Error(ctx.Err()))

			// Call shutdown with background context as the passed in context has been canceled
			return col.shutdown(context.Background())
		}
	}
	return col.shutdown(ctx)
}

func (col *Collector) shutdown(ctx context.Context) error {
	col.setCollectorState(StateClosing)

	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	if err := col.set.ConfigProvider.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown config provider: %w", err))
	}

	errs = multierr.Append(errs, col.shutdownServiceAndTelemetry(ctx))

	col.setCollectorState(StateClosed)

	return errs
}

// shutdownServiceAndTelemetry bundles shutting down the service and telemetryInitializer.
// Returned error will be in multierr form and wrapped.
func (col *Collector) shutdownServiceAndTelemetry(ctx context.Context) error {
	var errs error

	// shutdown service
	if err := col.service.Shutdown(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown service after error: %w", err))
	}

	// TODO: Move this as part of the service shutdown.
	// shutdown telemetryInitializer
	if err := col.service.telemetryInitializer.shutdown(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown collector telemetry: %w", err))
	}
	return errs
}

// setCollectorState provides current state of the collector
func (col *Collector) setCollectorState(state State) {
	col.state.Store(int32(state))
}

func getBallastSize(host component.Host) uint64 {
	var ballastSize uint64
	extensions := host.GetExtensions()
	for _, extension := range extensions {
		if ext, ok := extension.(interface{ GetBallastSize() uint64 }); ok {
			ballastSize = ext.GetBallastSize()
			break
		}
	}
	return ballastSize
}
