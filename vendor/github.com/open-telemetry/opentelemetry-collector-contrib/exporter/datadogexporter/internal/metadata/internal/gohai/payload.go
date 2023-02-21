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

package gohai // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata/internal/gohai"

import (
	"encoding/json"
)

type gohai struct {
	CPU        interface{} `json:"cpu"`
	FileSystem interface{} `json:"filesystem"`
	Memory     interface{} `json:"memory"`
	Network    interface{} `json:"network"`
	Platform   interface{} `json:"platform"`
}

// Payload handles the JSON unmarshalling of the metadata payload
// As weird as it sounds, in the v5 payload the value of the "gohai" field
// is a JSON-formatted string. So this struct contains a MarshaledGohaiPayload
// which will be marshaled as a JSON-formatted string.
type Payload struct {
	Gohai gohaiMarshaler `json:"gohai"`
}

// gohaiSerializer implements json.Marshaler and json.Unmarshaler on top of a gohai payload
type gohaiMarshaler struct {
	gohai *gohai
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the gohai struct twice (to a string) to comply with
// the v5 payload format
func (m gohaiMarshaler) MarshalJSON() ([]byte, error) {
	marshaledPayload, err := json.Marshal(m.gohai)
	if err != nil {
		return []byte(""), err
	}
	doubleMarshaledPayload, err := json.Marshal(string(marshaledPayload))
	if err != nil {
		return []byte(""), err
	}
	return doubleMarshaledPayload, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Unmarshals the passed bytes twice (first to a string, then to gohai.Gohai)
func (m *gohaiMarshaler) UnmarshalJSON(bytes []byte) error {
	firstUnmarshall := ""
	err := json.Unmarshal(bytes, &firstUnmarshall)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(firstUnmarshall), &(m.gohai))
	return err
}
