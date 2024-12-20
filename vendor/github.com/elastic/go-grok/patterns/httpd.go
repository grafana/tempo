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

var Httpd map[string]string = map[string]string{
	"HTTPDUSER":       `%{EMAILADDRESS}|%{USER}`,
	"HTTPDERROR_DATE": `%{DAY} %{MONTH} %{MONTHDAY} %{TIME} %{YEAR}`,

	"HTTPD_COMMONLOG":   `%{IPORHOST:source.address} (?:-|%{HTTPDUSER:apache.access.user.identity}) (?:-|%{HTTPDUSER:user.name}) \[%{HTTPDATE:timestamp}\] "(?:%{WORD:http.request.method} %{NOTSPACE:url.original}(?: HTTP/%{NUMBER:http.version})?|%{DATA})" (?:-|%{INT:http.response.status_code:int}) (?:-|%{INT:http.response.body.size:long})`,
	"HTTPD_COMBINEDLOG": `%{HTTPD_COMMONLOG} "(?:-|%{DATA:http.request.referrer})" "(?:-|%{DATA:user_agent.original})"`,

	"HTTPD20_ERRORLOG": `\[%{HTTPDERROR_DATE:timestamp}\] \[%{LOGLEVEL:log.level}\] (?:\[client %{IPORHOST:source.address}\] )?%{GREEDYDATA:message}`,
	"HTTPD24_ERRORLOG": `\[%{HTTPDERROR_DATE:timestamp}\] \[(?:%{WORD:apache.error.module})?:%{LOGLEVEL:log.level}\] \[pid %{POSINT:process.pid:long}(:tid %{INT:process.thread.id:int})?\](?: \(%{POSINT:apache.error.proxy.error.code}\)?%{DATA:apache.error.proxy.error.message}:)?(?: \[client %{IPORHOST:source.address}(?::%{POSINT:source.port:int})?\])?(?: %{DATA:error.code}:)? %{GREEDYDATA:message}`,
	"HTTPD_ERRORLOG":   `%{HTTPD20_ERRORLOG}|%{HTTPD24_ERRORLOG}`,

	"COMMONAPACHELOG":   `%{HTTPD_COMMONLOG}`,
	"COMBINEDAPACHELOG": `%{HTTPD_COMBINEDLOG}`,
}
