// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package patterns

var Rails map[string]string = map[string]string{
	"RUUID":       `\S{32}`,
	"RCONTROLLER": `(?P<rails___controller___class>[^#]+)#(?P<rails___controller___action>\w+)`,

	"RAILS3HEAD":    `(?m)Started %{WORD:http.request.method} "%{URIPATHPARAM:url.original}" for %{IPORHOST:source.address} at (?<timestamp>%{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND} %{ISO8601_TIMEZONE})`,
	"RPROCESSING":   `\W*Processing by %{RCONTROLLER} as (?P<rails___request___format>\S+)(?:\W*Parameters: {%{DATA:rails.request.params}}\W*)?`,
	"RAILS3FOOT":    `Completed %{POSINT:http.response.status_code:int}%{DATA} in %{NUMBER:rails.request.duration.total:float}ms %{RAILS3PROFILE}%{GREEDYDATA}`,
	"RAILS3PROFILE": `(?:\(Views: %{NUMBER:rails.request.duration.view:float}ms \| ActiveRecord: %{NUMBER:rails.request.duration.active_record:float}ms|\(ActiveRecord: %{NUMBER:rails.request.duration.active_record:float}ms)?`,

	"RAILS3": `%{RAILS3HEAD}(?:%{RPROCESSING})?(?P<rails___request___explain___original>(?:%{DATA}\n)*)(?:%{RAILS3FOOT})?`,
}
