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

var Bro map[string]string = map[string]string{
	"BRO_BOOL":  `[TF]`,
	"BRO_DATA":  `[^\t]+`,
	"BRO_HTTP":  `%{NUMBER:timestamp}\t%{NOTSPACE:zeek.session_id}\t%{IP:source.address}\t%{INT:source.port:int}\t%{IP:destination.address}\t%{INT:destination.port:int}\t%{INT:zeek.http.trans_depth:int}\t(?:-|%{WORD:http.request.method})\t(?:-|%{BRO_DATA:url.domain})\t(?:-|%{BRO_DATA:url.original})\t(?:-|%{BRO_DATA:http.request.referrer})\t(?:-|%{BRO_DATA:user_agent.original})\t(?:-|%{NUMBER:http.request.body.size:long})\t(?:-|%{NUMBER:http.response.body.size:long})\t(?:-|%{POSINT:http.response.status_code:int})\t(?:-|%{DATA:zeek.http.status_msg})\t(?:-|%{POSINT:zeek.http.info_code:int})\t(?:-|%{DATA:zeek.http.info_msg})\t(?:-|%{BRO_DATA:zeek.http.filename})\t(?:\(empty\)|%{BRO_DATA:zeek.http.tags})\t(?:-|%{BRO_DATA:url.username})\t(?:-|%{BRO_DATA:url.password})\t(?:-|%{BRO_DATA:zeek.http.proxied})\t(?:-|%{BRO_DATA:zeek.http.orig_fuids})\t(?:-|%{BRO_DATA:http.request.mime_type})\t(?:-|%{BRO_DATA:zeek.http.resp_fuids})\t(?:-|%{BRO_DATA:http.response.mime_type})`,
	"BRO_DNS":   `%{NUMBER:timestamp}\t%{NOTSPACE:zeek.session_id}\t%{IP:source.address}\t%{INT:source.port:int}\t%{IP:destination.address}\t%{INT:destination.port:int}\t%{WORD:network.transport}\t(?:-|%{INT:dns.id:int})\t(?:-|%{BRO_DATA:dns.question.name})\t(?:-|%{INT:zeek.dns.qclass:int})\t(?:-|%{BRO_DATA:zeek.dns.qclass_name})\t(?:-|%{INT:zeek.dns.qtype:int})\t(?:-|%{BRO_DATA:dns.question.type})\t(?:-|%{INT:zeek.dns.rcode:int})\t(?:-|%{BRO_DATA:dns.response_code})\t(?:-|%{BRO_BOOL:zeek.dns.AA})\t(?:-|%{BRO_BOOL:zeek.dns.TC})\t(?:-|%{BRO_BOOL:zeek.dns.RD})\t(?:-|%{BRO_BOOL:zeek.dns.RA})\t(?:-|%{NONNEGINT:zeek.dns.Z:int})\t(?:-|%{BRO_DATA:zeek.dns.answers})\t(?:-|%{DATA:zeek.dns.TTLs})\t(?:-|%{BRO_BOOL:zeek.dns.rejected})`,
	"BRO_CONN":  `%{NUMBER:timestamp}\t%{NOTSPACE:zeek.session_id}\t%{IP:source.address}\t%{INT:source.port:int}\t%{IP:destination.address}\t%{INT:destination.port:int}\t%{WORD:network.transport}\t(?:-|%{BRO_DATA:network.protocol.name})\t(?:-|%{NUMBER:zeek.connection.duration:float})\t(?:-|%{INT:zeek.connection.orig_bytes:long})\t(?:-|%{INT:zeek.connection.resp_bytes:long})\t(?:-|%{BRO_DATA:zeek.connection.state})\t(?:-|%{BRO_BOOL:zeek.connection.local_orig})\t(?:(?:-|%{BRO_BOOL:zeek.connection.local_resp})\t)?(?:-|%{INT:zeek.connection.missed_bytes:long})\t(?:-|%{BRO_DATA:zeek.connection.history})\t(?:-|%{INT:source.packets:long})\t(?:-|%{INT:source.bytes:long})\t(?:-|%{INT:destination.packets:long})\t(?:-|%{INT:destination.bytes:long})\t(?:\(empty\)|%{BRO_DATA:zeek.connection.tunnel_parents})`,
	"BRO_FILES": `%{NUMBER:timestamp}\t%{NOTSPACE:zeek.files.fuid}\t(?:-|%{IP:server.address})\t(?:-|%{IP:client.address})\t(?:-|%{BRO_DATA:zeek.files.session_ids})\t(?:-|%{BRO_DATA:zeek.files.source})\t(?:-|%{INT:zeek.files.depth:int})\t(?:-|%{BRO_DATA:zeek.files.analyzers})\t(?:-|%{BRO_DATA:file.mime_type})\t(?:-|%{BRO_DATA:file.name})\t(?:-|%{NUMBER:zeek.files.duration:float})\t(?:-|%{BRO_DATA:zeek.files.local_orig})\t(?:-|%{BRO_BOOL:zeek.files.is_orig})\t(?:-|%{INT:zeek.files.seen_bytes:long})\t(?:-|%{INT:file.size:long})\t(?:-|%{INT:zeek.files.missing_bytes:long})\t(?:-|%{INT:zeek.files.overflow_bytes:long})\t(?:-|%{BRO_BOOL:zeek.files.timedout})\t(?:-|%{BRO_DATA:zeek.files.parent_fuid})\t(?:-|%{BRO_DATA:file.hash.md5})\t(?:-|%{BRO_DATA:file.hash.sha1})\t(?:-|%{BRO_DATA:file.hash.sha256})\t(?:-|%{BRO_DATA:zeek.files.extracted})`,
}
