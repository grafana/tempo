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

var MongoDB map[string]string = map[string]string{
	"MONGO_LOG":           `%{SYSLOGTIMESTAMP:timestamp} \[%{WORD:db.mongodb.component}\] %{GREEDYDATA:message}`,
	"MONGO_QUERY_CONTENT": `(.*?)`,
	"MONGO_QUERY":         `\{ %{MONGO_QUERY_CONTENT:MONGO_QUERY} \} ntoreturn:`,
	"MONGO_SLOWQUERY":     `%{WORD:db.mongodb.profile.op} %{MONGO_WORDDASH:db.mongodb.database}\.%{MONGO_WORDDASH:db.mongodb.collection} %{WORD}: \{ %{MONGO_QUERY_CONTENT:db.mongodb.query.original} \} ntoreturn:%{NONNEGINT:db.mongodb.profile.ntoreturn:int} ntoskip:%{NONNEGINT:db.mongodb.profile.ntoskip:int} nscanned:%{NONNEGINT:db.mongodb.profile.nscanned:int}.*? nreturned:%{NONNEGINT:db.mongodb.profile.nreturned:int}.*? %{INT:db.mongodb.profile.duration:int}ms`,
	"MONGO_WORDDASH":      `\b[\w-]+\b`,
	"MONGO3_SEVERITY":     `\w`,
	"MONGO3_COMPONENT":    `%{WORD}`,
	"MONGO3_LOG":          `%{TIMESTAMP_ISO8601:timestamp} %{MONGO3_SEVERITY:log.level} (?:-|%{MONGO3_COMPONENT:db.mongodb.component})%{SPACE}(?:\[%{DATA:db.mongodb.context}\])? %{GREEDYDATA:message}`,
}
