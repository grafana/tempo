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

package azure // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders/azure"

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Metadata(_ context.Context) (*ComputeMetadata, error) {
	args := m.MethodCalled("Metadata")
	arg := args.Get(0)
	var cm *ComputeMetadata
	if arg != nil {
		cm = arg.(*ComputeMetadata)
	}
	return cm, args.Error(1)
}
