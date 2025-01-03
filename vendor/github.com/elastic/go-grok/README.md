# Grok

Grok is grok parsing library based on `re2` regexp.

## Usage

#### Basic usage:
```go
g := grok.New()

// use custom patterns
patternDefinitions := map[string]string{
    // patterns can be nested
    "NGINX_HOST":         `(?:%{IP:destination.ip}|%{NGINX_NOTSEPARATOR:destination.domain})(:%{NUMBER:destination.port})?`,
    // NGINX_NOTSEPARATOR is used in NGINX_HOST. IP and NUMBER are part of default pattern set
    "NGINX_NOTSEPARATOR": `"[^\t ,:]+"`,
}
g.AddPatterns(patternDefinitions)

// compile grok before use, this will generate regex.Regex based on pattern and 
// subpatterns provided.
// this needs to be performed just once.
err := g.Compile("%{NGINX_HOST}", true)

res, err := g.ParseString("127.0.0.1:1234")
```

results in:
```go
map[string]string {
    "destination.ip": "127.0.0.1", 
    "destination.port": "1234", 
}
```


#### Unnamed usage:

In this case we changed
`err := g.Compile("%{NGINX_HOST}", false)` to
`err := g.Compile("%{NGINX_HOST}", true)` 
allowing unnamed return matches. In case of unnamed match, definition name is used. 

```go
g := grok.New()

// use custom patterns
patternDefinitions := map[string]string{
    // patterns can be nested
    "NGINX_HOST":         `(?:%{IP:destination.ip}|%{NGINX_NOTSEPARATOR:destination.domain})(:%{NUMBER:destination.port})?`,
    // NGINX_NOTSEPARATOR is used in NGINX_HOST. IP and NUMBER are part of default pattern set
    "NGINX_NOTSEPARATOR": `"[^\t ,:]+"`,
}
g.AddPatterns(patternDefinitions)

// compile grok before use, this will generate regex.Regex based on pattern and 
// subpatterns provided
err := g.Compile("%{NGINX_HOST}", false)

res, err := g.ParseString("127.0.0.1:1234")
```

results in:
```go
map[string]string {
    "NGINX_HOST": "127.0.0.1:1234", 
    "destination.ip": "127.0.0.1", 
    "IPV4": "127.0.0.1", 
    "destination.port": "1234", 
    "BASE10NUM": "1234", 
}
```


#### Typed arguments usage:

In this case we're marking `destination.port` as `int` using definition `%{NUMBER:destination.port:int}`.

```go
g := grok.New()

// use custom patterns
patternDefinitions := map[string]string{
    "NGINX_HOST":         `(?:%{IP:destination.ip}|%{NGINX_NOTSEPARATOR:destination.domain})(:%{NUMBER:destination.port:int})?`,
    "NGINX_NOTSEPARATOR": `"[^\t ,:]+"`,
}
g.AddPatterns(patternDefinitions)

// compile grok before use, this will generate regex.Regex based on pattern and 
// subpatterns provided
err := g.Compile("%{NGINX_HOST}", true)

res, err := g.ParseTypedString("127.0.0.1:1234")
```

See type changed from `map[string]string` to `map[string]interface{}` and `destination.port` is now a number:
```go
map[string]interface {} {
    "destination.ip": "127.0.0.1", 
    "destination.port": 1234, 
}
```

## Benchmarks

Comparing to [github.com/vjeantet/grok](https://github.com/vjeantet/grok) and more optimized version based on previous one [github.com/trivago/grok](https://github.com/trivago/grok)

```
BenchmarkParseString-10                 	   15466	     76811 ns/op	    4578 B/op	       5 allocs/op
BenchmarkParseStringRegexp-10           	   15351	     77109 ns/op	    3840 B/op	       3 allocs/op
BenchmarkParseStringTrivago-10          	   15868	     76416 ns/op	    4593 B/op	       5 allocs/op
BenchmarkParseStringVjeanet-10          	   15548	     77111 ns/op	    5897 B/op	       6 allocs/op

BenchmarkNestedParseString-10           	   42201	     28908 ns/op	    3463 B/op	       4 allocs/op
BenchmarkNestedParseStringTrivago-10    	   41937	     28836 ns/op	    3449 B/op	       4 allocs/op
BenchmarkNestedParseStringVjeanet-10    	   41080	     29174 ns/op	    4045 B/op	       5 allocs/op

BenchmarkTypedParseString-10            	   39934	     29707 ns/op	    3851 B/op	       9 allocs/op
BenchmarkTypedParseStringTrivago-10     	   40146	     29238 ns/op	    3475 B/op	       6 allocs/op
BenchmarkTypedParseStringVjeanet-10     	   39931	     30616 ns/op	    4196 B/op	      14 allocs/op
```


## Default set of patterns

This library comes with a default set of patterns defined in `patterns/default.go` file.
You can include more predefined patterns from `patterns/*.go` like so

```go
g := grok.New()
g.AddPatterns(patterns.Rails) // to include whole set
g.AddPattern(patterns.Ruby["RUBY_LOGLEVEL"]) // to include specific one
```

Default set consists of:

| Name | Example |
|-----|-----|
| WORD |  "hello", "world123", "test_data" |
| NOTSPACE | "example", "text-with-dashes", "12345" |
| SPACE | " ", "\t", "  " |
| INT | "123", "-456", "+789" |
| NUMBER | "123", "456.789", "-0.123" |
| BOOL |"true", "false", "true" |
| BASE10NUM | "123", "-123.456", "0.789" |
| BASE16NUM | "1a2b", "0x1A2B", "-0x1a2b3c" |
| BASE16FLOAT |  "0x1.a2b3", "-0x1A2B3C.D" |
| POSINT | "123", "456", "789" |
| NONNEGINT | "0", "123", "456" |
| GREEDYDATA |"anything goes", "literally anything", "123 #@!" |
| QUOTEDSTRING | "\"This is a quote\"", "'single quoted'" |
| UUID |"123e4567-e89b-12d3-a456-426614174000" |
| URN | "urn:isbn:0451450523", "urn:ietf:rfc:2648" |

#### Network patterns

| Name | Example |
|-----|-----|
| IP | "192.168.1.1", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"|
| IPV6 |  "2001:0db8:85a3:0000:0000:8a2e:0370:7334", " |:1", "fe80::1ff:fe23:4567:890a" |
| IPV4 |  "192.168.1.1", "10.0.0.1", "172.16.254.1" |
| IPORHOST | "example.com", "192.168.1.1", "fe80::1ff:fe23:4567:890a" |
| HOSTNAME | "example.com", "sub.domain.co.uk", "localhost" |
| EMAILLOCALPART | "john.doe", "alice123", "bob-smith" |
| EMAILADDRESS |"john.doe@example.com", "alice123@domain.co.uk" |
| USERNAME | "user1", "john.doe", "alice_123" |
| USER |  "user1", "john.doe", "alice_123" |
| MAC |"00:1A:2B:3C:4D:5E", "001A.2B3C.4D5E" |
| CISCOMAC | "001A.2B3C.4D5E", "001B.2C3D.4E5F", "001C.2D3E.4F5A" |
| WINDOWSMAC |  "00-1A-2B-3C-4D-5E", "00-1B-2C-3D-4E-5F" |
| COMMONMAC |"00:1A:2B:3C:4D:5E", "00:1B:2C:3D:4E:5F" |
| HOSTPORT | "example.com:80", "192.168.1.1:8080" |

#### Paths patterns

| Name | Example |
|-----|-----|
| UNIXPATH |  "/home/user", "/var/log/syslog", "/tmp/abc_123" |
| TTY | "/dev/pts/1", "/dev/tty0", "/dev/ttyS0" |
| WINPATH |"C:\\Program Files\\App", "D:\\Work\\project\\file.txt" |
| URIPROTO |  "http", "https", "ftp" |
| URIHOST |"example.com", "192.168.1.1:8080" |
| URIPATH |"/path/to/resource", "/another/path", "/root" |
| URIQUERY |  "key=value", "search=query&active=true" |
| URIPARAM |  "?key=value", "?search=query&active=true" |
| URIPATHPARAM | "/path?query=1", "/folder/path?valid=true" |
| PATH |"/home/user/documents", "C:\\Windows\\system32", "/var/log/syslog" |

#### Datetime patterns

| Name | Example |
|-----|-----|
| MONTH | "January", "Feb", "March", "Apr", "May", "Jun", "Jul", "August", "September", "October", "Nov", "December" |
| MONTHNUM | "01", "02", "03", ... "11", "12" |
| DAY | "Monday", "Tuesday", ... "Sunday" |
| YEAR |"1999", "2000", "2021" |
| HOUR |"00", "12", "23" |
| MINUTE | "00", "30", "59" |
| SECOND | "00", "30", "60" |
| TIME |"14:30", "23:59:59", "12:00:00", "12:00:60" |
| DATE_US |"04/21/2022", "12-25-2020", "07/04/1999" |
| DATE_EU |"21.04.2022", "25/12/2020", "04-07-1999" |
| ISO8601_TIMEZONE |"Z", "+02:00", "-05:00" |
| ISO8601_SECOND |  "59", "30", "60.123" |
| TIMESTAMP_ISO8601 |  "2022-04-21T14:30:00Z", "2020-12-25T23:59:59+02:00", "1999-07-04T12:00:00-05:00" |
| DATE |"04/21/2022", "21.04.2022", "12-25-2020" |
| DATESTAMP | "04/21/2022 14:30", "21.04.2022 23:59", "12-25-2020 12:00" |
| TZ |  "EST", "CET", "PDT" |
| DATESTAMP_RFC822 |"Wed Jan 12 2024 14:33 EST" |
| DATESTAMP_RFC2822 |  "Tue, 12 Jan 2022 14:30 +0200", "Fri, 25 Dec 2020 23:59 -0500", "Sun, 04 Jul 1999 12:00 Z" |
| DATESTAMP_OTHER | "Tue Jan 12 14:30 EST 2022", "Fri Dec 25 23:59 CET 2020", "Sun Jul 04 12:00 PDT 1999" |
| DATESTAMP_EVENTLOG | "20220421143000", "20201225235959", "19990704120000" |

#### Syslog patterns

| Name | Example |
|-----|-----|
| SYSLOGTIMESTAMP | "Jan  1 00:00:00", "Mar 15 12:34:56", "Dec 31 23:59:59" |
| PROG |"sshd", "kernel", "cron" |
| SYSLOGPROG |"sshd[1234]", "kernel", "cron[5678]" |
| SYSLOGHOST |"example.com", "192.168.1.1", "localhost" |
| SYSLOGFACILITY |  "<1.2>", "<12345.13456>" |
| HTTPDATE |  "25/Dec/2024:14:33 4" |