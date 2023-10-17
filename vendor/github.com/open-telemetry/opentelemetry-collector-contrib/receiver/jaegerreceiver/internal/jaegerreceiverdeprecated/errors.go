// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiverdeprecated // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver/internal/jaegerreceiverdeprecated"

type httpError struct {
	msg        string
	statusCode int
}

func (h httpError) Error() string {
	return h.msg
}
