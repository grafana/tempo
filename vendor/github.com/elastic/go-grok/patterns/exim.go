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

var Exim map[string]string = map[string]string{
	"EXIM_MSGID":           `[0-9A-Za-z]{6}-[0-9A-Za-z]{6}-[0-9A-Za-z]{2}`,
	"EXIM_FLAGS":           `(?:<=|=>|->|\*>|\*\*|==|<>|>>)`,
	"EXIM_DATE":            `(:?%{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{TIME})`,
	"EXIM_PID":             `\[%{POSINT:process.pid:int}\]`,
	"EXIM_QT":              `((\d+y)?(\d+w)?(\d+d)?(\d+h)?(\d+m)?(\d+s)?)`,
	"EXIM_EXCLUDE_TERMS":   `(Message is frozen|(Start|End) queue run| Warning: | retry time not reached | no (IP address|host name) found for (IP address|host) | unexpected disconnection while reading SMTP command | no immediate delivery: |another process is handling this message)`,
	"EXIM_REMOTE_HOST":     `(H=(\(%{NOTSPACE:source.host.name}\) )?(\(%{NOTSPACE:exim.log.remote_address}\) )?\[%{IP:source.address}\](?::%{POSINT:source.port:int})?)`,
	"EXIM_INTERFACE":       `(I=\[%{IP:destination.address}\](?::%{NUMBER:destination.port:int}))`,
	"EXIM_PROTOCOL":        `(P=%{NOTSPACE:network.protocol.name})`,
	"EXIM_MSG_SIZE":        `(S=%{NUMBER:exim.log.message.body.size:int})`,
	"EXIM_HEADER_ID":       `(id=%{NOTSPACE:exim.log.header_id})`,
	"EXIM_QUOTED_CONTENT":  `(?:\\.|[^\\"])*`,
	"EXIM_SUBJECT":         `(T="%{EXIM_QUOTED_CONTENT:exim.log.message.subject}")`,
	"EXIM_UNKNOWN_FIELD":   `(?:[A-Za-z0-9]{1,4}=(?:%{QUOTEDSTRING}|%{NOTSPACE}))`,
	"EXIM_NAMED_FIELDS":    `(?: (?:%{EXIM_REMOTE_HOST}|%{EXIM_INTERFACE}|%{EXIM_PROTOCOL}|%{EXIM_MSG_SIZE}|%{EXIM_HEADER_ID}|%{EXIM_SUBJECT}|%{EXIM_UNKNOWN_FIELD}))*`,
	"EXIM_MESSAGE_ARRIVAL": `%{EXIM_DATE:timestamp} (?:%{EXIM_PID} )?%{EXIM_MSGID:exim.log.message.id} (?P<exim___log___flags>\<\=) ((?P<exim___log___status>[a-z:]) )?%{EMAILADDRESS:exim.log.sender.email}%{EXIM_NAMED_FIELDS}(?:(?: from \<?%{DATA:exim.log.sender.original}\>?)? for %{EMAILADDRESS:exim.log.recipient.email})?`,
	"EXIM":                 `%{EXIM_MESSAGE_ARRIVAL}`,
}
