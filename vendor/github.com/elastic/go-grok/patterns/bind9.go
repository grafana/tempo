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

var Bind9 map[string]string = map[string]string{
	"BIND9_TIMESTAMP":    `%{MONTHDAY}[-]%{MONTH}[-]%{YEAR} %{TIME}`,
	"BIND9_DNSTYPE":      `(?:A|AAAA|CAA|CDNSKEY|CDS|CERT|CNAME|CSYNC|DLV|DNAME|DNSKEY|DS|HINFO|LOC|MX|NAPTR|NS|NSEC|NSEC3|OPENPGPKEY|PTR|RRSIG|RP|SIG|SMIMEA|SOA|SRV|TSIG|TXT|URI|IN)`,
	"BIND9_CATEGORY":     `(?:queries)`,
	"BIND9_QUERYLOGBASE": `client(:? @0x(?:[0-9A-Fa-f]+))? %{IP:client.address}#%{POSINT:client.port:int} \(%{GREEDYDATA:bind.log.question.name}\): query: %{GREEDYDATA:dns.question.name} (?P<dns___question___class>(?:IN)) %{BIND9_DNSTYPE:dns.question.type}(:? %{DATA:bind.log.question.flags})? \(%{IP:server.address}\)`,
	"BIND9_QUERYLOG":     `%{BIND9_TIMESTAMP:timestamp} %{BIND9_CATEGORY:bing.log.category}: %{LOGLEVEL:log.level}: %{BIND9_QUERYLOGBASE}`,
	"BIND9":              `%{BIND9_QUERYLOG}`,
}
