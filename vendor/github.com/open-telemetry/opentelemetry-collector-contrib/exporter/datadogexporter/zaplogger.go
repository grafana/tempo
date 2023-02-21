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

package datadogexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"

import (
	"fmt"

	"go.uber.org/zap"
)

// zaplogger implements the tracelog.Logger interface on top of a zap.Logger
type zaplogger struct{ logger *zap.Logger }

// Trace implements Logger.
func (z *zaplogger) Trace(v ...interface{}) { /* N/A */ }

// Tracef implements Logger.
func (z *zaplogger) Tracef(format string, params ...interface{}) { /* N/A */ }

// Debug implements Logger.
func (z *zaplogger) Debug(v ...interface{}) {
	z.logger.Debug(fmt.Sprint(v...))
}

// Debugf implements Logger.
func (z *zaplogger) Debugf(format string, params ...interface{}) {
	z.logger.Debug(fmt.Sprintf(format, params...))
}

// Info implements Logger.
func (z *zaplogger) Info(v ...interface{}) {
	z.logger.Info(fmt.Sprint(v...))
}

// Infof implements Logger.
func (z *zaplogger) Infof(format string, params ...interface{}) {
	z.logger.Info(fmt.Sprintf(format, params...))
}

// Warn implements Logger.
func (z *zaplogger) Warn(v ...interface{}) error {
	z.logger.Warn(fmt.Sprint(v...))
	return nil
}

// Warnf implements Logger.
func (z *zaplogger) Warnf(format string, params ...interface{}) error {
	z.logger.Warn(fmt.Sprintf(format, params...))
	return nil
}

// Error implements Logger.
func (z *zaplogger) Error(v ...interface{}) error {
	z.logger.Error(fmt.Sprint(v...))
	return nil
}

// Errorf implements Logger.
func (z *zaplogger) Errorf(format string, params ...interface{}) error {
	z.logger.Error(fmt.Sprintf(format, params...))
	return nil
}

// Critical implements Logger.
func (z *zaplogger) Critical(v ...interface{}) error {
	z.logger.Error(fmt.Sprint(v...), zap.Bool("critical", true))
	return nil
}

// Criticalf implements Logger.
func (z *zaplogger) Criticalf(format string, params ...interface{}) error {
	z.logger.Error(fmt.Sprintf(format, params...), zap.Bool("critical", true))
	return nil
}

// Flush implements Logger.
func (z *zaplogger) Flush() {
	_ = z.logger.Sync()
}
