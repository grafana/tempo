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

var Junos map[string]string = map[string]string{
	"RT_FLOW_TAG":   `(?:RT_FLOW_SESSION_CREATE|RT_FLOW_SESSION_CLOSE|RT_FLOW_SESSION_DENY)`,
	"RT_FLOW_EVENT": `%{RT_FLOW_TAG}`,

	"RT_FLOW1": `%{RT_FLOW_TAG:juniper.srx.tag}: %{GREEDYDATA:juniper.srx.reason}: %{IP:source.address}/%{INT:source.port:int}->%{IP:destination.address}/%{INT:destination.port:int} %{DATA:juniper.srx.service_name} %{IP:source.nat.ip}/%{INT:source.nat.port:int}->%{IP:destination.nat.ip}/%{INT:destination.nat.port:int} (?:(?:None)|(?:%{DATA:juniper.srx.src_nat_rule_name})) (?:(?:None)|(?:%{DATA:juniper.srx.dst_nat_rule_name})) %{INT:network.iana_number} %{DATA:rule.name} %{DATA:observer.ingress.zone} %{DATA:observer.egress.zone} %{INT:juniper.srx.session_id} \d+\(%{INT:source.bytes:long}\) \d+\(%{INT:destination.bytes:long}\) %{INT:juniper.srx.elapsed_time:int} .*`,
	"RT_FLOW2": `%{RT_FLOW_TAG:juniper.srx.tag}: session created %{IP:source.address}/%{INT:source.port:int}->%{IP:destination.address}/%{INT:destination.port:int} %{DATA:juniper.srx.service_name} %{IP:source.nat.ip}/%{INT:source.nat.port:int}->%{IP:destination.nat.ip}/%{INT:destination.nat.port:int} (?:(?:None)|(?:%{DATA:juniper.srx.src_nat_rule_name})) (?:(?:None)|(?:%{DATA:juniper.srx.dst_nat_rule_name})) %{INT:network.iana_number} %{DATA:rule.name} %{DATA:observer.ingress.zone} %{DATA:observer.egress.zone} %{INT:juniper.srx.session_id} .*`,
	"RT_FLOW3": `%{RT_FLOW_TAG:juniper.srx.tag}: session denied %{IP:source.address}/%{INT:source.port:int}->%{IP:destination.address}/%{INT:destination.port:int} %{DATA:juniper.srx.service_name} %{INT:network.iana_number}\(\d\) %{DATA:rule.name} %{DATA:observer.ingress.zone} %{DATA:observer.egress.zone} (.*)?`,
}
