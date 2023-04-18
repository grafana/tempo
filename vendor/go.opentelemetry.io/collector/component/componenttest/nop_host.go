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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"go.opentelemetry.io/collector/component"
)

// nopHost mocks a receiver.ReceiverHost for test purposes.
type nopHost struct{}

// NewNopHost returns a new instance of nopHost with proper defaults for most tests.
func NewNopHost() component.Host {
	return &nopHost{}
}

func (nh *nopHost) ReportFatalError(_ error) {}

func (nh *nopHost) GetFactory(_ component.Kind, _ component.Type) component.Factory {
	return nil
}

func (nh *nopHost) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (nh *nopHost) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return nil
}
