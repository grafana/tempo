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

var Default map[string]string = map[string]string{
	"WORD":     `\b\w+\b`,
	"NOTSPACE": `\S+`,
	"SPACE":    `\s*`,
	"DATA":     `.*?`,

	// types
	"INT":    `(?:[+-]?(?:[0-9]+))`,
	"NUMBER": `(?:%{BASE10NUM})`,
	"BOOL":   "true|false",

	"BASE10NUM":    `([+-]?(?:[0-9]+(?:\.[0-9]+)?)|\.[0-9]+)`,
	"BASE16NUM":    `[+-]?(?:0x)?[0-9A-Fa-f]+`,                    // Adjusted, removed lookbehind
	"BASE16FLOAT":  `[+-]?(?:0x)?[0-9A-Fa-f]+(?:\.[0-9A-Fa-f]*)?`, // Adjusted, removed lookbehind and word boundaries
	"POSINT":       `\b[1-9][0-9]*\b`,
	"NONNEGINT":    `\b[0-9]+\b`,
	"GREEDYDATA":   `.*`,
	"QUOTEDSTRING": `"([^"\\]*(\\.[^"\\]*)*)"|\'([^\'\\]*(\\.[^\'\\]*)*)\'`,
	"UUID":         `[A-Fa-f0-9]{8}-(?:[A-Fa-f0-9]{4}-){3}[A-Fa-f0-9]{12}`,
	"URN":          `urn:[0-9A-Za-z][0-9A-Za-z-]{0,31}:[0-9A-Za-z()+,.:=@;$_!*'/?#-]+`,

	// network
	"IP":   `(?:%{IPV6}|%{IPV4})`,
	"IPV6": `((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?`,
	"IPV4": `(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`,

	"IPORHOST":       `(?:%{IP}|%{HOSTNAME})`,
	"HOSTNAME":       `\b(?:[0-9A-Za-z][0-9A-Za-z-]{0,62})(?:\.(?:[0-9A-Za-z][0-9A-Za-z-]{0,62}))*(\.?|\b)`,
	"EMAILLOCALPART": `[a-zA-Z][a-zA-Z0-9_.+-=:]+`,
	"EMAILADDRESS":   `%{EMAILLOCALPART}@%{HOSTNAME}`,
	"USERNAME":       `[a-zA-Z0-9._-]+`,
	"USER":           `%{USERNAME}`,

	"MAC":        `(?:%{CISCOMAC}|%{WINDOWSMAC}|%{COMMONMAC})`,
	"CISCOMAC":   `(?:(?:[A-Fa-f0-9]{4}\.){2}[A-Fa-f0-9]{4})`,
	"WINDOWSMAC": `(?:(?:[A-Fa-f0-9]{2}-){5}[A-Fa-f0-9]{2})`,
	"COMMONMAC":  `(?:(?:[A-Fa-f0-9]{2}:){5}[A-Fa-f0-9]{2})`,
	"HOSTPORT":   `%{IPORHOST}:%{POSINT}`,

	// paths
	"UNIXPATH":     `(/[\w_%!$@:.,+~-]+)+`,
	"TTY":          `/dev/(pts|tty([pq])?)(\w+)?/?(?:[0-9]+)`,
	"WINPATH":      `[A-Za-z]+:(\\[^\\?*]+)+`,
	"URIPROTO":     `[A-Za-z][A-Za-z0-9+\.-]+`,
	"URIHOST":      `%{IPORHOST}(?::%{POSINT})?`,
	"URIPATH":      `(/[A-Za-z0-9$.+!*'(){},~:;=@#%&_\-]+)+`,
	"URIQUERY":     `[A-Za-z0-9$.+!*'|(){},~@#%&/=:;_?\-\[\]<>]*`,
	"URIPARAM":     `\?%{URIQUERY}`,
	"URIPATHPARAM": `%{URIPATH}(?:\?%{URIQUERY})?`,
	"URI":          `%{URIPROTO}://(?:%{USER}(?::[^@]*)?@)?%{URIHOST}(?:%{URIPATH}(?:\?%{URIQUERY})?)?`,
	"PATH":         `(?:%{UNIXPATH}|%{WINPATH})`,

	// dates
	"MONTH": `\b(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\b`,

	// Months: January, Feb, 3, 03, 12, December "MONTH": `\b(?:[Jj]an(?:uary|uar)?|[Ff]eb(?:ruary|ruar)?|[Mm](?:a|ä)?r(?:ch|z)?|[Aa]pr(?:il)?|[Mm]a(?:y|i)?|[Jj]un(?:e|i)?|[Jj]ul(?:y|i)?|[Aa]ug(?:ust)?|[Ss]ep(?:tember)?|[Oo](?:c|k)?t(?:ober)?|[Nn]ov(?:ember)?|[Dd]e(?:c|z)(?:ember)?)\b`,
	"MONTHNUM": `(?:0[1-9]|1[0-2])`,
	"MONTHDAY": `(?:(?:0[1-9])|(?:[12][0-9])|(?:3[01])|[1-9])`,

	// Days Monday, Tue, Thu, etc
	"DAY": `\b(?:Mon(?:day)?|Tue(?:sday)?|Wed(?:nesday)?|Thu(?:rsday)?|Fri(?:day)?|Sat(?:urday)?|Sun(?:day)?)\b`,

	// Years?
	"YEAR":   `(\d\d){1,2}`,
	"HOUR":   `(?:2[0123]|[01]?[0-9])`,
	"MINUTE": `(?:[0-5][0-9])`,

	// '60' is a leap second in most time standards and thus is valid.
	"SECOND": `(?:(?:[0-5][0-9]|60)(?:[:.,][0-9]+)?)`,
	"TIME":   `%{HOUR}:%{MINUTE}(?::%{SECOND})?`,

	// datestamp is YYYY/MM/DD-HH:MM:SS.UUUU (or something like it)
	"DATE_US":            `%{MONTHNUM}[/-]%{MONTHDAY}[/-]%{YEAR}`,
	"DATE_EU":            `%{MONTHDAY}[./-]%{MONTHNUM}[./-]%{YEAR}`,
	"ISO8601_TIMEZONE":   `(?:Z|[+-]%{HOUR}(?::?%{MINUTE}))`,
	"ISO8601_SECOND":     `%{SECOND}`,
	"TIMESTAMP_ISO8601":  `%{YEAR}-%{MONTHNUM}-%{MONTHDAY}[T ]%{HOUR}:?%{MINUTE}(?::?%{SECOND})?%{ISO8601_TIMEZONE}?`,
	"DATE":               `%{DATE_US}|%{DATE_EU}`,
	"DATESTAMP":          `%{DATE}[- ]%{TIME}`,
	"TZ":                 `(?:[PMACE][SED]T|UTC)`,
	"DATESTAMP_RFC822":   `%{DAY} %{MONTH} %{MONTHDAY} %{YEAR} %{TIME} %{TZ}`,
	"DATESTAMP_RFC2822":  `%{DAY}, %{MONTHDAY} %{MONTH} %{YEAR} %{TIME} %{ISO8601_TIMEZONE}`,
	"DATESTAMP_OTHER":    `%{DAY} %{MONTH} %{MONTHDAY} %{TIME} %{TZ} %{YEAR}`,
	"DATESTAMP_EVENTLOG": `%{YEAR}%{MONTHNUM}%{MONTHDAY}%{HOUR}%{MINUTE}%{SECOND}`,

	// Syslog Dates: Month Day HH:MM:SS	"MONTH":         `\b(?:Jan(?:uary|uar)?|Feb(?:ruary|ruar)?|Mar(?:ch|z)?|Apr(?:il)?|May|i|Jun(?:e|i)?|Jul(?:y|i)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\b`,
	"SYSLOGTIMESTAMP": `%{MONTH} +%{MONTHDAY} %{TIME}`,
	"PROG":            `[!-Z\\^-~]+`,         // Simplified range based on ASCII
	"SYSLOGPROG":      `%{PROG}(?:\[\d+\])?`, // Simplified, as type hints and named groups aren't supported in Go `regexp`
	"SYSLOGHOST":      `%{IPORHOST}`,
	"SYSLOGFACILITY":  `<%{NONNEGINT}.%{NONNEGINT}>`, // Simplified to remove type hints
	"HTTPDATE":        `%{MONTHDAY}/%{MONTH}/%{YEAR}:%{TIME} %{INT}`,

	// Shortcuts
	"QS": `%{QUOTEDSTRING}`,

	// Log formats
	"SYSLOGBASE": `%{SYSLOGTIMESTAMP:timestamp} (?:%{SYSLOGFACILITY} )?%{SYSLOGHOST:host.name} %{SYSLOGPROG}:`,

	//	Log Levels
	"LOGLEVEL": `(?i)(alert|trace|debug|notice|info(?:rmation)?|warn(?:ing)?|err(?:or)?|crit(?:ical)?|fatal|severe|emerg(?:ency)?)`,
}
