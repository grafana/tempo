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

var Java map[string]string = map[string]string{
	"JAVACLASS":          `(?:[a-zA-Z$_][a-zA-Z$_0-9]*\.)*[a-zA-Z$_][a-zA-Z$_0-9]*`,
	"JAVAFILE":           `(?:[a-zA-Z$_0-9. -]+)`,
	"JAVAMETHOD":         `(?:(<(?:cl)?init>)|[a-zA-Z$_][a-zA-Z$_0-9]*)`,
	"JAVASTACKTRACEPART": `%{SPACE}at %{JAVACLASS:java.log.origin.class.name}\.%{JAVAMETHOD:log.origin.function}\(%{JAVAFILE:log.origin.file.name}(?::%{INT:log.origin.file.line:int})?\)`,
	"JAVATHREAD":         `(?:[A-Z]{2}-Processor[\d]+)`,
	"JAVALOGMESSAGE":     `(?:.*)`,

	"CATALINA7_DATESTAMP": `%{MONTH} %{MONTHDAY}, %{YEAR} %{HOUR}:%{MINUTE}:%{SECOND} (?:AM|PM)`,
	"CATALINA7_LOG":       `%{CATALINA7_DATESTAMP:timestamp} %{JAVACLASS:java.log.origin.class.name}(?: %{JAVAMETHOD:log.origin.function})?\s*(?:%{LOGLEVEL:log.level}:)? %{JAVALOGMESSAGE:message}`,

	"CATALINA8_DATESTAMP": `%{MONTHDAY}-%{MONTH}-%{YEAR} %{HOUR}:%{MINUTE}:%{SECOND}`,
	"CATALINA8_LOG":       `%{CATALINA8_DATESTAMP:timestamp} %{LOGLEVEL:log.level} \[%{DATA:java.log.origin.thread.name}\] %{JAVACLASS:java.log.origin.class.name}\.(?:%{JAVAMETHOD:log.origin.function})? %{JAVALOGMESSAGE:message}`,

	"CATALINA_DATESTAMP": `(?:%{CATALINA8_DATESTAMP})|(?:%{CATALINA7_DATESTAMP})`,
	"CATALINALOG":        `(?:%{CATALINA8_LOG})|(?:%{CATALINA7_LOG})`,

	"TOMCAT7_LOG": `%{CATALINA7_LOG}`,
	"TOMCAT8_LOG": `%{CATALINA8_LOG}`,

	"TOMCATLEGACY_DATESTAMP": `%{YEAR}-%{MONTHNUM}-%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND}(?: %{ISO8601_TIMEZONE})?`,
	"TOMCATLEGACY_LOG":       `%{TOMCATLEGACY_DATESTAMP:timestamp} \| %{LOGLEVEL:log.level} \| %{JAVACLASS:java.log.origin.class.name} - %{JAVALOGMESSAGE:message}`,

	"TOMCAT_DATESTAMP": `(?:%{CATALINA8_DATESTAMP})|(?:%{CATALINA7_DATESTAMP})|(?:%{TOMCATLEGACY_DATESTAMP})`,

	"TOMCATLOG": `(?:%{TOMCAT8_LOG})|(?:%{TOMCAT7_LOG})|(?:%{TOMCATLEGACY_LOG})`,
}
