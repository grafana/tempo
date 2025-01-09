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

var Syslog map[string]string = map[string]string{
	"SYSLOG5424PRINTASCII": `[!-~]+`,

	"SYSLOGBASE2":      `(?:%{SYSLOGTIMESTAMP:timestamp}|%{TIMESTAMP_ISO8601:timestamp})(?: %{SYSLOGFACILITY})?(?: %{SYSLOGHOST:host.name})?(?: %{SYSLOGPROG}:)?`,
	"SYSLOGPAMSESSION": `%{SYSLOGBASE} (%{GREEDYDATA:message})%{WORD:system.auth.pam.module}\(%{DATA:system.auth.pam.origin}\): session %{WORD:system.auth.pam.session_state} for user %{USERNAME:user.name}(?: by %{GREEDYDATA})?`,

	"CRON_ACTION": `[A-Z ]+`,
	"CRONLOG":     `%{SYSLOGBASE} \(%{USER:user.name}\) %{CRON_ACTION:system.cron.action} \(%{DATA:message}\)`,

	"SYSLOGLINE": `%{SYSLOGBASE2} %{GREEDYDATA:message}`,

	"SYSLOG5424PRI":  `<%{NONNEGINT:log.syslog.priority:int}>`,
	"SYSLOG5424SD":   `\[%{DATA}\]+`,
	"SYSLOG5424BASE": `%{SYSLOG5424PRI}%{NONNEGINT:system.syslog.version} +(?:-|%{TIMESTAMP_ISO8601:timestamp}) +(?:-|%{IPORHOST:host.name}) +(?:-|%{SYSLOG5424PRINTASCII:process.command}) +(?:-|%{POSINT:process.pid:int}) +(?:-|%{SYSLOG5424PRINTASCII:event.code}) +(?:-|%{SYSLOG5424SD:system.syslog.structured_data})?`,

	"SYSLOG5424LINE": `%{SYSLOG5424BASE} +%{GREEDYDATA:message}`,
}
