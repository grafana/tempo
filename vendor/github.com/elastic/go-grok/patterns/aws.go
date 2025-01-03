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

var AWS map[string]string = map[string]string{
	"S3_REQUEST_LINE": `(?:%{WORD:http.request.method} %{NOTSPACE:url.original}(?: HTTP/%{NUMBER:http.version})?)`,
	"S3_ACCESS_LOG":   `%{WORD:aws.s3access.bucket_owner} %{NOTSPACE:aws.s3access.bucket} \[%{HTTPDATE:timestamp}\] (?:-|%{IP:client.address}) (?:-|%{NOTSPACE:client.user.id}) %{NOTSPACE:aws.s3access.request_id} %{NOTSPACE:aws.s3access.operation} (?:-|%{NOTSPACE:aws.s3access.key}) (?:-|"%{S3_REQUEST_LINE:aws.s3access.request_uri}") (?:-|%{INT:http.response.status_code:int}) (?:-|%{NOTSPACE:aws.s3access.error_code}) (?:-|%{INT:aws.s3access.bytes_sent:long}) (?:-|%{INT:aws.s3access.object_size:long}) (?:-|%{INT:aws.s3access.total_time:int}) (?:-|%{INT:aws.s3access.turn_around_time:int}) "(?:-|%{DATA:http.request.referrer})" "(?:-|%{DATA:user_agent.original})" (?:-|%{NOTSPACE:aws.s3access.version_id})(?: (?:-|%{NOTSPACE:aws.s3access.host_id}) (?:-|%{NOTSPACE:aws.s3access.signature_version}) (?:-|%{NOTSPACE:tls.cipher}) (?:-|%{NOTSPACE:aws.s3access.authentication_type}) (?:-|%{NOTSPACE:aws.s3access.host_header}) (?:-|%{NOTSPACE:aws.s3access.tls_version}))?`,

	"ELB_URIHOST":      `%{IPORHOST:url.domain}(?::%{POSINT:url.port:int})?`,
	"ELB_URIPATHQUERY": `%{URIPATH:url.path}(?:\?%{URIQUERY:url.query})?`,
	"ELB_URIPATHPARAM": `%{ELB_URIPATHQUERY}`,
	"ELB_URI":          `%{URIPROTO:url.scheme}://(?:%{USER:url.username}(?::[^@]*)?@)?(?:%{ELB_URIHOST})?(?:%{ELB_URIPATHQUERY})?`,
	"ELB_REQUEST_LINE": `(?:%{WORD:http.request.method} %{ELB_URI:url.original}(?: HTTP/%{NUMBER:http.version})?)`,
	"ELB_V1_HTTP_LOG":  `%{TIMESTAMP_ISO8601:timestamp} %{NOTSPACE:aws.elb.name} %{IP:source.address}:%{INT:source.port:int} (?:-|(?:%{IP:aws.elb.backend.ip}:%{INT:aws.elb.backend.port:int})) (?:-1|%{NUMBER:aws.elb.request_processing_time.sec:float}) (?:-1|%{NUMBER:aws.elb.backend_processing_time.sec:float}) (?:-1|%{NUMBER:aws.elb.response_processing_time.sec:float}) %{INT:http.response.status_code:int} (?:-|%{INT:aws.elb.backend.http.response.status_code:int}) %{INT:http.request.body.size:long} %{INT:http.response.body.size:long} "%{ELB_REQUEST_LINE}"(?: "(?:-|%{DATA:user_agent.original})" (?:-|%{NOTSPACE:tls.cipher}) (?:-|%{NOTSPACE:aws.elb.ssl_protocol}))?`,
	"ELB_ACCESS_LOG":   `%{ELB_V1_HTTP_LOG}`,

	"CLOUDFRONT_ACCESS_LOG": `(?<timestamp>%{YEAR}[-]%{MONTHNUM}[-]%{MONTHDAY}\t%{TIME})\t%{WORD:aws.cloudfront.x_edge_location}\t(?:-|%{INT:destination.bytes:long})\t%{IPORHOST:source.address}\t%{WORD:http.request.method}\t%{HOSTNAME:url.domain}\t%{NOTSPACE:url.path}\t(?:(?:000)|%{INT:http.response.status_code:int})\t(?:-|%{DATA:http.request.referrer})\t%{DATA:user_agent.original}\t(?:-|%{DATA:url.query})\t(?:-|%{DATA:aws.cloudfront.http.request.cookie})\t%{WORD:aws.cloudfront.x_edge_result_type}\t%{NOTSPACE:aws.cloudfront.x_edge_request_id}\t%{HOSTNAME:aws.cloudfront.http.request.host}\t%{URIPROTO:network.protocol.name}\t(?:-|%{INT:source.bytes:long})\t%{NUMBER:aws.cloudfront.time_taken:float}\t(?:-|%{IP:network.forwarded_ip})\t(?:-|%{DATA:aws.cloudfront.ssl_protocol})\t(?:-|%{NOTSPACE:tls.cipher})\t%{WORD:aws.cloudfront.x_edge_response_result_type}(?:\t(?:-|HTTP/%{NUMBER:http.version})\t(?:-|%{DATA:aws.cloudfront.fle_status})\t(?:-|%{DATA:aws.cloudfront.fle_encrypted_fields})\t%{INT:source.port:int}\t%{NUMBER:aws.cloudfront.time_to_first_byte:float}\t(?:-|%{DATA:aws.cloudfront.x_edge_detailed_result_type})\t(?:-|%{NOTSPACE:http.request.mime_type})\t(?:-|%{INT:aws.cloudfront.http.request.size:long})\t(?:-|%{INT:aws.cloudfront.http.request.range.start:long})\t(?:-|%{INT:aws.cloudfront.http.request.range.end:long}))?`,
}
