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
//go:build !windows
// +build !windows

package system // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata/internal/system"

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// keep as var for testing
var hostnamePath = "/bin/hostname"

func getSystemFQDN() (string, error) {
	// Go does not provide a way to get the full hostname
	// so we make a best-effort by running the hostname binary
	// if available
	if _, err := os.Stat(hostnamePath); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()
		cmd := exec.CommandContext(ctx, hostnamePath, "-f")
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}

	// if stat failed for any reason, fail silently
	return "", nil
}
