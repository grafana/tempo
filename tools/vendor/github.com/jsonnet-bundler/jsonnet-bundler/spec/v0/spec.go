// Copyright 2018 jsonnet-bundler authors
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

package spec

import (
	"encoding/json"
	"sort"

	"github.com/elliotchance/orderedmap/v2"
)

const Version = 0

// JsonnetFile is the structure of a `.json` file describing a set of jsonnet
// dependencies. It is used for both, the jsonnetFile and the lockFile.
type JsonnetFile struct {
	Dependencies *orderedmap.OrderedMap[string, Dependency]
}

// New returns a new JsonnetFile with the dependencies map initialized
func New() JsonnetFile {
	return JsonnetFile{
		Dependencies: orderedmap.NewOrderedMap[string, Dependency](),
	}
}

// jsonFile is the json representation of a JsonnetFile, which is different for
// compatibility reasons.
type jsonFile struct {
	Dependencies []Dependency `json:"dependencies"`
}

// UnmarshalJSON unmarshals a `jsonFile`'s json into a JsonnetFile
func (jf *JsonnetFile) UnmarshalJSON(data []byte) error {
	var s jsonFile
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	jf.Dependencies = orderedmap.NewOrderedMap[string, Dependency]()
	for _, d := range s.Dependencies {
		jf.Dependencies.Set(d.Name, d)
	}
	return nil
}

// MarshalJSON serializes a JsonnetFile into json of the format of a `jsonFile`
func (jf JsonnetFile) MarshalJSON() ([]byte, error) {
	var s jsonFile
	for _, k := range jf.Dependencies.Keys() {
		d, _ := jf.Dependencies.Get(k)
		s.Dependencies = append(s.Dependencies, d)
	}

	sort.SliceStable(s.Dependencies, func(i int, j int) bool {
		return s.Dependencies[i].Name < s.Dependencies[j].Name
	})

	return json.Marshal(s)
}

type Dependency struct {
	Name      string `json:"name"`
	Source    Source `json:"source"`
	Version   string `json:"version"`
	Sum       string `json:"sum,omitempty"`
	DepSource string `json:"-"`
}

type Source struct {
	GitSource   *GitSource   `json:"git,omitempty"`
	LocalSource *LocalSource `json:"local,omitempty"`
}

type GitSource struct {
	Remote string `json:"remote"`
	Subdir string `json:"subdir"`
}

type LocalSource struct {
	Directory string `json:"directory"`
}
