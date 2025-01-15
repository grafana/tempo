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

var Firewalls map[string]string = map[string]string{

	// NetScreen firewall logs
	"NETSCREENSESSIONLOG": `%{SYSLOGTIMESTAMP:timestamp} %{IPORHOST:observer.hostname} %{NOTSPACE:observer.name}\: (?P<observer___product>NetScreen) device_id=%{WORD:netscreen.device_id} .*?(system-(\w+)-(%{NONNEGINT:event.code})\((%{WORD:netscreen.session.type})\))?\: start_time="%{DATA:netscreen.session.start_time}" duration=%{INT:netscreen.session.duration:int} policy_id=%{INT:netscreen.policy_id} service=%{DATA:netscreen.service} proto=%{INT:netscreen.protocol_number:int} src zone=%{WORD:observer.ingress.zone} dst zone=%{WORD:observer.egress.zone} action=%{WORD:event.action} sent=%{INT:source.bytes:long} rcvd=%{INT:destination.bytes:long} src=%{IPORHOST:source.address} dst=%{IPORHOST:destination.address}(?: src_port=%{INT:source.port:int} dst_port=%{INT:destination.port:int})?(?: src-xlated ip=%{IP:source.nat.ip} port=%{INT:source.nat.port:int} dst-xlated ip=%{IP:destination.nat.ip} port=%{INT:destination.nat.port:int})?(?: session_id=%{INT:netscreen.session.id} reason=%{GREEDYDATA:netscreen.session.reason})?`,

	// == Cisco ASA ==
	"CISCO_TAGGED_SYSLOG": `^<%{POSINT:log.syslog.priority:int}>%{CISCOTIMESTAMP:timestamp}( %{SYSLOGHOST:host.name})? ?: %%{CISCOTAG:cisco.asa.tag}:`,
	"CISCOTIMESTAMP":      `%{MONTH} +%{MONTHDAY}(?: %{YEAR})? %{TIME}`,
	"CISCOTAG":            `[A-Z0-9]+-%{INT}-(?:[A-Z0-9_]+)`,

	// Common Particles
	"CISCO_ACTION":     `Built|Teardown|Deny|Denied|denied by ACL|requested|permitted|denied|discarded|est-allowed|Dropping|created|deleted`,
	"CISCO_REASON":     `Duplicate TCP SYN|Failed to locate egress interface|Invalid transport field|No matching connection|DNS Response|DNS Query|(?:%{WORD}\s*)*`,
	"CISCO_DIRECTION":  `Inbound|inbound|Outbound|outbound`,
	"CISCO_INTERVAL":   `first hit|%{INT}-second interval`,
	"CISCO_XLATE_TYPE": `static|dynamic`,

	// helpers
	"CISCO_HITCOUNT_INTERVAL":     `hit-cnt %{INT:cisco.asa.hit_count:int} (?:first hit|%{INT:cisco.asa.interval:int}-second interval)`,
	"CISCO_SRC_IP_USER":           `%{NOTSPACE:observer.ingress.interface.name}:%{IP:source.address}(?:\(%{DATA:source.user.name}\))?`,
	"CISCO_DST_IP_USER":           `%{NOTSPACE:observer.egress.interface.name}:%{IP:destination.address}(?:\(%{DATA:destination.user.name}\))?`,
	"CISCO_SRC_HOST_PORT_USER":    `%{NOTSPACE:observer.ingress.interface.name}:(?:(?:%{IP:source.address})|(?:%{HOSTNAME:source.address}))(?:/%{INT:source.port:int})?(?:\(%{DATA:source.user.name}\))?`,
	"CISCO_DST_HOST_PORT_USER":    `%{NOTSPACE:observer.egress.interface.name}:(?:(?:%{IP:destination.address})|(?:%{HOSTNAME:destination.address}))(?:/%{INT:destination.port:int})?(?:\(%{DATA:destination.user.name}\))?`,
	"CISCOFW104001":               `\((?:Primary|Secondary)\) Switching to ACTIVE - %{GREEDYDATA:event.reason}`,
	"CISCOFW104002":               `\((?:Primary|Secondary)\) Switching to STANDBY - %{GREEDYDATA:event.reason}`,
	"CISCOFW104003":               `\((?:Primary|Secondary)\) Switching to FAILED\.`,
	"CISCOFW104004":               `\((?:Primary|Secondary)\) Switching to OK\.`,
	"CISCOFW105003":               `\((?:Primary|Secondary)\) Monitoring on [Ii]nterface %{NOTSPACE:network.interface.name} waiting`,
	"CISCOFW105004":               `\((?:Primary|Secondary)\) Monitoring on [Ii]nterface %{NOTSPACE:network.interface.name} normal`,
	"CISCOFW105005":               `\((?:Primary|Secondary)\) Lost Failover communications with mate on [Ii]nterface %{NOTSPACE:network.interface.name}`,
	"CISCOFW105008":               `\((?:Primary|Secondary)\) Testing [Ii]nterface %{NOTSPACE:network.interface.name}`,
	"CISCOFW105009":               `\((?:Primary|Secondary)\) Testing on [Ii]nterface %{NOTSPACE:network.interface.name} (?:Passed|Failed)`,
	"CISCOFW106001":               `%{CISCO_DIRECTION:cisco.asa.network.direction} %{WORD:cisco.asa.network.transport} connection %{CISCO_ACTION:cisco.asa.outcome} from %{IP:source.address}/%{INT:source.port:int} to %{IP:destination.address}/%{INT:destination.port:int} flags %{DATA:cisco.asa.tcp_flags} on interface %{NOTSPACE:observer.egress.interface.name}`,
	"CISCOFW106006_106007_106010": `%{CISCO_ACTION:cisco.asa.outcome} %{CISCO_DIRECTION:cisco.asa.network.direction} %{WORD:cisco.asa.network.transport} (?:from|src) %{IP:source.address}/%{INT:source.port:int}(?:\(%{DATA:source.user.name}\))? (?:to|dst) %{IP:destination.address}/%{INT:destination.port:int}(?:\(%{DATA:destination.user.name}\))? (?:(?:on interface %{NOTSPACE:observer.egress.interface.name})|(?:due to %{CISCO_REASON:event.reason}))`,
	"CISCOFW106014":               `%{CISCO_ACTION:cisco.asa.outcome} %{CISCO_DIRECTION:cisco.asa.network.direction} %{WORD:cisco.asa.network.transport} src %{CISCO_SRC_IP_USER} dst %{CISCO_DST_IP_USER}\s?\(type %{INT:cisco.asa.icmp_type:int}, code %{INT:cisco.asa.icmp_code:int}\)`,
	"CISCOFW106015":               `%{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} \(%{DATA:cisco.asa.rule_name}\) from %{IP:source.address}/%{INT:source.port:int} to %{IP:destination.address}/%{INT:destination.port:int} flags %{DATA:cisco.asa.tcp_flags} on interface %{NOTSPACE:observer.egress.interface.name}`,
	"CISCOFW106021":               `%{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} reverse path check from %{IP:source.address} to %{IP:destination.address} on interface %{NOTSPACE:observer.egress.interface.name}`,
	"CISCOFW106023":               `%{CISCO_ACTION:action}( protocol)? %{WORD:network.protocol.name} src %{DATA:source.interface}:%{DATA:source.address}(/%{INT:source.port})?(\(%{DATA:source.fwuser}\))? dst %{DATA:destination.interface}:%{DATA:destination.address}(/%{INT:destination.port})?(\(%{DATA:destination.fwuser}\))?( \(type %{INT:icmp_type}, code %{INT:icmp_code}\))? by access-group "?%{DATA:policy_id}"? \[%{DATA:hashcode1}, %{DATA:hashcode2}\]`,
	"CISCOFW106100_2_3":           `access-list %{NOTSPACE:cisco.asa.rule_name} %{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} for user '%{DATA:user.name}' %{DATA:observer.ingress.interface.name}\/%{IP:source.address}\(%{INT:source.port:int}\) -> %{DATA:observer.egress.interface.name}\/%{IP:destination.address}\(%{INT:destination.port:int}\) %{CISCO_HITCOUNT_INTERVAL} \[%{DATA:metadata.cisco.asa.hashcode1}\, %{DATA:metadata.cisco.asa.hashcode2}\]`,

	"CISCOFW106100":                             `access-list %{NOTSPACE:cisco.asa.rule_name} %{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} %{DATA:observer.ingress.interface.name}/%{IP:source.address}\(%{INT:source.port:int}\)(?:\(%{DATA:source.user.name}\))? -> %{DATA:observer.egress.interface.name}/%{IP:destination.address}\(%{INT:destination.port:int}\)(?:\(%{DATA:source.user.name}\))? hit-cnt %{INT:cisco.asa.hit_count:int} %{CISCO_INTERVAL} \[%{DATA:metadata.cisco.asa.hashcode1}\, %{DATA:metadata.cisco.asa.hashcode2}\]`,
	"CISCOFW304001":                             `%{IP:source.address}(?:\(%{DATA:source.user.name}\))? Accessed URL %{IP:destination.address}:%{GREEDYDATA:url.original}`,
	"CISCOFW110002":                             `%{CISCO_REASON:event.reason} for %{WORD:cisco.asa.network.transport} from %{DATA:observer.ingress.interface.name}:%{IP:source.address}/%{INT:source.port:int} to %{IP:destination.address}/%{INT:destination.port:int}`,
	"CISCOFW302010":                             `%{INT:cisco.asa.connections.in_use:int} in use, %{INT:cisco.asa.connections.most_used:int} most used`,
	"CISCOFW302013_302014_302015_302016":        `%{CISCO_ACTION:cisco.asa.outcome}(?: %{CISCO_DIRECTION:cisco.asa.network.direction})? %{WORD:cisco.asa.network.transport} connection %{INT:cisco.asa.connection_id} for %{NOTSPACE:observer.ingress.interface.name}:%{IP:source.address}/%{INT:source.port:int}(?: \(%{IP:source.nat.ip}/%{INT:source.nat.port:int}\))?(?:\(%{DATA:source.user.name?}\))? to %{NOTSPACE:observer.egress.interface.name}:%{IP:destination.address}/%{INT:destination.port:int}( \(%{IP:destination.nat.ip}/%{INT:destination.nat.port:int}\))?(?:\(%{DATA:destination.user.name}\))?( duration %{TIME:cisco.asa.duration} bytes %{INT:network.bytes:long})?(?: %{CISCO_REASON:event.reason})?(?: \(%{DATA:user.name}\))?`,
	"CISCOFW302020_302021":                      `%{CISCO_ACTION:cisco.asa.outcome}(?: %{CISCO_DIRECTION:cisco.asa.network.direction})? %{WORD:cisco.asa.network.transport} connection for faddr %{IP:destination.address}/%{INT:cisco.asa.icmp_seq:int}(?:\(%{DATA:destination.user.name}\))? gaddr %{IP:source.nat.ip}/%{INT:cisco.asa.icmp_type:int} laddr %{IP:source.address}/%{INT}(?: \(%{DATA:source.user.name}\))?`,
	"CISCOFW305011":                             `%{CISCO_ACTION:cisco.asa.outcome} %{CISCO_XLATE_TYPE} %{WORD:cisco.asa.network.transport} translation from %{DATA:observer.ingress.interface.name}:%{IP:source.address}(/%{INT:source.port:int})?(?:\(%{DATA:source.user.name}\))? to %{DATA:observer.egress.interface.name}:%{IP:destination.address}/%{INT:destination.port:int}`,
	"CISCOFW313001_313004_313008":               `%{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} type=%{INT:cisco.asa.icmp_type:int}, code=%{INT:cisco.asa.icmp_code:int} from %{IP:source.address} on interface %{NOTSPACE:observer.egress.interface.name}(?: to %{IP:destination.address})?`,
	"CISCOFW313005":                             `%{CISCO_REASON:event.reason} for %{WORD:cisco.asa.network.transport} error message: %{WORD} src %{CISCO_SRC_IP_USER} dst %{CISCO_DST_IP_USER} \(type %{INT:cisco.asa.icmp_type:int}, code %{INT:cisco.asa.icmp_code:int}\) on %{NOTSPACE} interface\.\s+Original IP payload: %{WORD:cisco.asa.original_ip_payload.network.transport} src %{IP:cisco.asa.original_ip_payload.source.address}/%{INT:cisco.asa.original_ip_payload.source.port:int}(?:\(%{DATA:cisco.asa.original_ip_payload.source.user.name}\))? dst %{IP:cisco.asa.original_ip_payload.destination.address}/%{INT:cisco.asa.original_ip_payload.destination.port:int}(?:\(%{DATA:cisco.asa.original_ip_payload.destination.user.name}\))?`,
	"CISCOFW321001":                             `Resource '%{DATA:cisco.asa.resource.name}' limit of %{POSINT:cisco.asa.resource.limit:int} reached for system`,
	"CISCOFW402117":                             `%{WORD:cisco.asa.network.type}: Received a non-IPSec packet \(protocol=\s?%{WORD:cisco.asa.network.transport}\) from %{IP:source.address} to %{IP:destination.address}\.?`,
	"CISCOFW402119":                             `%{WORD:cisco.asa.network.type}: Received an %{WORD:cisco.asa.ipsec.protocol} packet \(SPI=\s?%{DATA:cisco.asa.ipsec.spi}, sequence number=\s?%{DATA:cisco.asa.ipsec.seq_num}\) from %{IP:source.address} \(user=\s?%{DATA:source.user.name}\) to %{IP:destination.address} that failed anti-replay checking\.?`,
	"CISCOFW419001":                             `%{CISCO_ACTION:cisco.asa.outcome} %{WORD:cisco.asa.network.transport} packet from %{NOTSPACE:observer.ingress.interface.name}:%{IP:source.address}/%{INT:source.port:int} to %{NOTSPACE:observer.egress.interface.name}:%{IP:destination.address}/%{INT:destination.port:int}, reason: %{GREEDYDATA:event.reason}`,
	"CISCOFW419002":                             `%{CISCO_REASON:event.reason} from %{DATA:observer.ingress.interface.name}:%{IP:source.address}/%{INT:source.port:int} to %{DATA:observer.egress.interface.name}:%{IP:destination.address}/%{INT:destination.port:int} with different initial sequence number`,
	"CISCOFW500004":                             `%{CISCO_REASON:event.reason} for protocol=%{WORD:cisco.asa.network.transport}, from %{IP:source.address}/%{INT:source.port:int} to %{IP:destination.address}/%{INT:destination.port:int}`,
	"CISCOFW602303_602304":                      `%{WORD:cisco.asa.network.type}: An %{CISCO_DIRECTION:cisco.asa.network.direction} %{DATA:cisco.asa.ipsec.tunnel_type} SA \(SPI=%{DATA:cisco.asa.ipsec.spi}\) between %{IP:source.address} and %{IP:destination.address} \(user=%{DATA:source.user.name}\) has been %{CISCO_ACTION:cisco.asa.outcome}`,
	"CISCOFW710001_710002_710003_710005_710006": `%{WORD:cisco.asa.network.transport} (?:request|access) %{CISCO_ACTION:cisco.asa.outcome} from %{IP:source.address}/%{INT:source.port:int} to %{DATA:observer.egress.interface.name}:%{IP:destination.address}/%{INT:destination.port:int}`,
	"CISCOFW713172":                             `Group = %{DATA:cisco.asa.source.group}, IP = %{IP:source.address}, Automatic NAT Detection Status:\s+Remote end\s*%{DATA:metadata.cisco.asa.remote_nat}\s*behind a NAT device\s+This\s+end\s*%{DATA:metadata.cisco.asa.local_nat}\s*behind a NAT device`,
	"CISCOFW733100":                             `\\s*%{DATA:[cisco.asa.burst.object}\s*\] drop %{DATA:cisco.asa.burst.id} exceeded. Current burst rate is %{INT:cisco.asa.burst.current_rate:int} per second, max configured rate is %{INT:cisco.asa.burst.configured_rate:int}; Current average rate is %{INT:cisco.asa.burst.avg_rate:int} per second, max configured rate is %{INT:cisco.asa.burst.configured_avg_rate:int}; Cumulative total count is %{INT:cisco.asa.burst.cumulative_count:int}`,

	"IPTABLES_TCP_FLAGS": `(CWR |ECE |URG |ACK |PSH |RST |SYN |FIN )*`,
	"IPTABLES_TCP_PART":  `(?:SEQ=%{INT:iptables.tcp.seq:int}\s+)?(?:ACK=%{INT:iptables.tcp.ack:int}\s+)?WINDOW=%{INT:iptables.tcp.window:int}\s+RES=0x%{BASE16NUM:iptables.tcp_reserved_bits}\s+%{IPTABLES_TCP_FLAGS:iptables.tcp.flags}`,

	"IPTABLES4_FRAG": `((\s)?(CE|DF|MF))*`,
	"IPTABLES4_PART": `SRC=%{IPV4:source.address}\s+DST=%{IPV4:destination.address}\s+LEN=(?:%{INT:iptables.length:int})?\s+TOS=(?:0|0x%{BASE16NUM:iptables.tos})?\s+PREC=(?:0x%{BASE16NUM:iptables.precedence_bits})?\s+TTL=(?:%{INT:iptables.ttl:int})?\s+ID=(?:%{INT:iptables.id})?\s+(?:%{IPTABLES4_FRAG:iptables.fragment_flags})?(?:\s+FRAG: %{INT:iptables.fragment_offset:int})?`,
	"IPTABLES6_PART": `SRC=%{IPV6:source.address}\s+DST=%{IPV6:destination.address}\s+LEN=(?:%{INT:iptables.length:int})?\s+TC=(?:0|0x%{BASE16NUM:iptables.tos})?\s+HOPLIMIT=(?:%{INT:iptables.ttl:int})?\s+FLOWLBL=(?:%{INT:iptables.flow_label})?`,

	"IPTABLES": `IN=(?:%{NOTSPACE:observer.ingress.interface.name})?\s+OUT=(?:%{NOTSPACE:observer.egress.interface.name})?\s+(?:MAC=(?:%{COMMONMAC:destination.mac})?(?::%{COMMONMAC:source.mac})?(?::A-Fa-f0-9{2}:A-Fa-f0-9{2})?\s+)?(:?%{IPTABLES4_PART}|%{IPTABLES6_PART}).*?PROTO=(?:%{WORD:network.transport})?\s+SPT=(?:%{INT:source.port:int})?\s+DPT=(?:%{INT:destination.port:int})?\s+(?:%{IPTABLES_TCP_PART})?`,

	// Shorewall firewall logs
	"SHOREWALL": `(?:%{SYSLOGTIMESTAMP:timestamp}) (?:%{WORD:observer.hostname}) .*Shorewall:(?:%{WORD:shorewall.firewall.type})?:(?:%{WORD:shorewall.firewall.action})?.*%{IPTABLES}`,

	// == SuSE Firewall 2 ==
	"SFW2_LOG_PREFIX": `SFW2\-INext\-%{NOTSPACE:suse.firewall.action}`,
	"SFW2":            `((?:%{SYSLOGTIMESTAMP:timestamp})|(?:%{TIMESTAMP_ISO8601:timestamp}))\s*%{HOSTNAME:observer.hostname}.*?%{SFW2_LOG_PREFIX:suse.firewall.log_prefix}\s*%{IPTABLES}`,
}
