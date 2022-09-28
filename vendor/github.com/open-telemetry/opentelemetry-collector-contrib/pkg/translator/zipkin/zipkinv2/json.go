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

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"encoding/json"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type jsonUnmarshaler struct {
	toTranslator ToTranslator
}

// UnmarshalTraces from JSON bytes.
func (j jsonUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(buf, &spans); err != nil {
		return ptrace.Traces{}, err
	}
	return j.toTranslator.ToTraces(spans)
}

// NewJSONTracesUnmarshaler returns an unmarshaler for JSON bytes.
func NewJSONTracesUnmarshaler(parseStringTags bool) ptrace.Unmarshaler {
	return jsonUnmarshaler{toTranslator: ToTranslator{ParseStringTags: parseStringTags}}
}

// NewJSONTracesMarshaler returns a marshaler to JSON bytes.
func NewJSONTracesMarshaler() ptrace.Marshaler {
	return marshaler{
		serializer: zipkinreporter.JSONSerializer{},
	}
}
