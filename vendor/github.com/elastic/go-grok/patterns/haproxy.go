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

var HAProxy map[string]string = map[string]string{
	"HAPROXYTIME":                    `\b%{HOUR}:%{MINUTE}(:%{SECOND})?\b`,
	"HAPROXYDATE":                    `%{MONTHDAY}/%{MONTH}/%{YEAR}:%{HAPROXYTIME}.%{INT}`,
	"HAPROXYCAPTUREDREQUESTHEADERS":  `(?:-|%{DATA:haproxy.http.request.captured_headers})`,
	"HAPROXYCAPTUREDRESPONSEHEADERS": `(?:-|%{DATA:haproxy.http.response.captured_headers})`,
	"HAPROXYURI":                     `(?:%{URIPROTO:url.scheme}://)?(?:%{USER:url.username}(?::[^@]*)?@)?(?:%{IPORHOST:url.domain}(?::%{POSINT:url.port:int})?)?(?:%{URIPATH:url.path}(?:\?%{URIQUERY:url.query})?)?`,
	"HAPROXYHTTPREQUESTLINE":         `(?:<BADREQ>|(?:%{WORD:http.request.method} %{HAPROXYURI:url.original}(?: HTTP/%{NUMBER:http.version})?))`,
	"HAPROXYHTTPBASE":                `%{IP:source.address}:%{INT:source.port:int} \[%{HAPROXYDATE:haproxy.request_date}\] %{NOTSPACE:haproxy.frontend_name} %{NOTSPACE:haproxy.backend_name}/(?:<NOSRV>|%{NOTSPACE:haproxy.server_name}) (?:-1|%{INT:haproxy.http.request.time_wait_ms:int})/(?:-1|%{INT:haproxy.total_waiting_time_ms:int})/(?:-1|%{INT:haproxy.connection_wait_time_ms:int})/(?:-1|%{INT:haproxy.http.request.time_wait_without_data_ms:int})/%{NOTSPACE:haproxy.total_time_ms} %{INT:http.response.status_code:int} %{INT:source.bytes:long} (?:-|%{DATA:haproxy.http.request.captured_cookie}) (?:-|%{DATA:haproxy.http.response.captured_cookie}) %{NOTSPACE:haproxy.termination_state} %{INT:haproxy.connections.active:int}/%{INT:haproxy.connections.frontend:int}/%{INT:haproxy.connections.backend:int}/%{INT:haproxy.connections.server:int}/%{INT:haproxy.connections.retries:int} %{INT:haproxy.server_queue:int}/%{INT:haproxy.backend_queue:int}(?: \{%{HAPROXYCAPTUREDREQUESTHEADERS}\}(?: \{%{HAPROXYCAPTUREDRESPONSEHEADERS}\})?)?(?: "%{HAPROXYHTTPREQUESTLINE}"?)?`,
	"HAPROXYHTTP":                    `(?:%{SYSLOGTIMESTAMP:timestamp}|%{TIMESTAMP_ISO8601:timestamp}) %{IPORHOST:host.name} %{SYSLOGPROG}: %{HAPROXYHTTPBASE}`,
	"HAPROXYTCP":                     `(?:%{SYSLOGTIMESTAMP:timestamp}|%{TIMESTAMP_ISO8601:timestamp}) %{IPORHOST:host.name} %{SYSLOGPROG}: %{IP:source.address}:%{INT:source.port:int} \[%{HAPROXYDATE:haproxy.request_date}\] %{NOTSPACE:haproxy.frontend_name} %{NOTSPACE:haproxy.backend_name}/(?:<NOSRV>|%{NOTSPACE:haproxy.server_name}) (?:-1|%{INT:haproxy.total_waiting_time_ms:int})/(?:-1|%{INT:haproxy.connection_wait_time_ms:int})/%{NOTSPACE:haproxy.total_time_ms} %{INT:source.bytes:long} %{NOTSPACE:haproxy.termination_state} %{INT:haproxy.connections.active:int}/%{INT:haproxy.connections.frontend:int}/%{INT:haproxy.connections.backend:int}/%{INT:haproxy.connections.server:int}/%{INT:haproxy.connections.retries:int} %{INT:haproxy.server_queue:int}/%{INT:haproxy.backend_queue:int}`,
}
