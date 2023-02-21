// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/clientutil"

import (
	"net/http"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

// WrapError wraps an error to a permanent consumer error that won't be retried if the http response code is non-retriable.
func WrapError(err error, resp *http.Response) error {
	if err == nil || resp == nil || !isNonRetriable(resp) {
		return err
	}
	return consumererror.NewPermanent(err)
}

func isNonRetriable(resp *http.Response) bool {
	return resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 413 || resp.StatusCode == 403
}
