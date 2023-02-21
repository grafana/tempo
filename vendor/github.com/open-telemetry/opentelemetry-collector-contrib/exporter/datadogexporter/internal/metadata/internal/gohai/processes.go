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
//go:build linux || darwin
// +build linux darwin

package gohai // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata/internal/gohai"

import (
	"github.com/DataDog/gohai/processes"
	"go.uber.org/zap"
)

// NewProcessesPayload builds a payload of processes metadata collected from gohai.
func NewProcessesPayload(hostname string, logger *zap.Logger) *ProcessesPayload {
	// Get processes metadata from gohai
	proc, err := new(processes.Processes).Collect()
	if err != nil {
		logger.Warn("Failed to retrieve processes metadata", zap.Error(err))
		return nil
	}

	processesPayload := map[string]interface{}{
		"snaps": []interface{}{proc},
	}
	return &ProcessesPayload{
		Processes: processesPayload,
		Meta: map[string]string{
			"host": hostname,
		},
	}
}
