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

package kafkaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"

import (
	"encoding/json"
	"errors"

	"github.com/Shopify/sarama"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

var errUnsupported = errors.New("unsupported serialization")

type rawMarshaler struct {
}

func newRawMarshaler() rawMarshaler {
	return rawMarshaler{}
}

func (r rawMarshaler) Marshal(logs plog.Logs, topic string) ([]*sarama.ProducerMessage, error) {
	var messages []*sarama.ProducerMessage
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		rl := logs.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				lr := sl.LogRecords().At(k)
				b, err := r.logBodyAsBytes(lr.Body())
				if err != nil {
					return nil, err
				}
				if len(b) == 0 {
					continue
				}

				messages = append(messages, &sarama.ProducerMessage{
					Topic: topic,
					Value: sarama.ByteEncoder(b),
				})
			}
		}
	}

	return messages, nil
}

func (r rawMarshaler) logBodyAsBytes(value pcommon.Value) ([]byte, error) {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		return r.interfaceAsBytes(value.Str())
	case pcommon.ValueTypeBytes:
		return value.Bytes().AsRaw(), nil
	case pcommon.ValueTypeBool:
		return r.interfaceAsBytes(value.Bool())
	case pcommon.ValueTypeDouble:
		return r.interfaceAsBytes(value.Double())
	case pcommon.ValueTypeInt:
		return r.interfaceAsBytes(value.Int())
	case pcommon.ValueTypeEmpty:
		return []byte{}, nil
	case pcommon.ValueTypeSlice:
		return r.interfaceAsBytes(value.Slice().AsRaw())
	case pcommon.ValueTypeMap:
		return r.interfaceAsBytes(value.Map().AsRaw())
	default:
		return nil, errUnsupported
	}
}

func (r rawMarshaler) interfaceAsBytes(value interface{}) ([]byte, error) {
	if value == nil {
		return []byte{}, nil
	}
	res, err := json.Marshal(value)
	return res, err
}

func (r rawMarshaler) Encoding() string {
	return "raw"
}
