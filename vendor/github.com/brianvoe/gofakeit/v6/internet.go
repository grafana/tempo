package gofakeit

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v6/data"
)

// DomainName will generate a random url domain name
func DomainName() string { return domainName(globalFaker.Rand) }

// DomainName will generate a random url domain name
func (f *Faker) DomainName() string { return domainName(f.Rand) }

func domainName(r *rand.Rand) string {
	name := strings.Replace(strings.ToLower(jobDescriptor(r)+bs(r)), " ", "", -1)

	return fmt.Sprintf("%s.%s", name, domainSuffix(r))
}

// DomainSuffix will generate a random domain suffix
func DomainSuffix() string { return domainSuffix(globalFaker.Rand) }

// DomainSuffix will generate a random domain suffix
func (f *Faker) DomainSuffix() string { return domainSuffix(f.Rand) }

func domainSuffix(r *rand.Rand) string {
	return getRandValue(r, []string{"internet", "domain_suffix"})
}

// URL will generate a random url string
func URL() string { return url(globalFaker.Rand) }

// URL will generate a random url string
func (f *Faker) URL() string { return url(f.Rand) }

func url(r *rand.Rand) string {
	// Slugs
	num := number(r, 1, 4)
	slug := make([]string, num)
	for i := 0; i < num; i++ {
		slug[i] = bs(r)
	}

	scheme := randomString(r, []string{"https", "http"})
	path := strings.ToLower(strings.Join(slug, "/"))

	url := fmt.Sprintf("%s://www.%s/%s", scheme, domainName(r), path)
	url = strings.Replace(url, " ", "", -1)

	return url
}

// HTTPMethod will generate a random http method
func HTTPMethod() string { return httpMethod(globalFaker.Rand) }

// HTTPMethod will generate a random http method
func (f *Faker) HTTPMethod() string { return httpMethod(f.Rand) }

func httpMethod(r *rand.Rand) string {
	return getRandValue(r, []string{"internet", "http_method"})
}

// IPv4Address will generate a random version 4 ip address
func IPv4Address() string { return ipv4Address(globalFaker.Rand) }

// IPv4Address will generate a random version 4 ip address
func (f *Faker) IPv4Address() string { return ipv4Address(f.Rand) }

func ipv4Address(r *rand.Rand) string {
	num := func() int { return r.Intn(256) }

	return fmt.Sprintf("%d.%d.%d.%d", num(), num(), num(), num())
}

// IPv6Address will generate a random version 6 ip address
func IPv6Address() string { return ipv6Address(globalFaker.Rand) }

// IPv6Address will generate a random version 6 ip address
func (f *Faker) IPv6Address() string { return ipv6Address(f.Rand) }

func ipv6Address(r *rand.Rand) string {
	num := func() int { return r.Intn(65536) }

	return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x", num(), num(), num(), num(), num(), num(), num(), num())
}

// MacAddress will generate a random mac address
func MacAddress() string { return macAddress(globalFaker.Rand) }

// MacAddress will generate a random mac address
func (f *Faker) MacAddress() string { return macAddress(f.Rand) }

func macAddress(r *rand.Rand) string {
	num := 255

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", r.Intn(num), r.Intn(num), r.Intn(num), r.Intn(num), r.Intn(num), r.Intn(num))
}

// HTTPStatusCode will generate a random status code
func HTTPStatusCode() int { return httpStatusCode(globalFaker.Rand) }

// HTTPStatusCode will generate a random status code
func (f *Faker) HTTPStatusCode() int { return httpStatusCode(f.Rand) }

func httpStatusCode(r *rand.Rand) int {
	randInt, _ := strconv.Atoi(getRandValue(r, []string{"internet", "http_status_general"}))
	return randInt
}

// HTTPStatusCodeSimple will generate a random simple status code
func HTTPStatusCodeSimple() int { return httpStatusCodeSimple(globalFaker.Rand) }

// HTTPStatusCodeSimple will generate a random simple status code
func (f *Faker) HTTPStatusCodeSimple() int { return httpStatusCodeSimple(f.Rand) }

func httpStatusCodeSimple(r *rand.Rand) int {
	randInt, _ := strconv.Atoi(getRandValue(r, []string{"internet", "http_status_simple"}))
	return randInt
}

// LogLevel will generate a random log level
// See data/LogLevels for list of available levels
func LogLevel(logType string) string { return logLevel(globalFaker.Rand, logType) }

// LogLevel will generate a random log level
// See data/LogLevels for list of available levels
func (f *Faker) LogLevel(logType string) string { return logLevel(f.Rand, logType) }

func logLevel(r *rand.Rand, logType string) string {
	if _, ok := data.LogLevels[logType]; ok {
		return getRandValue(r, []string{"log_level", logType})
	}

	return getRandValue(r, []string{"log_level", "general"})
}

// UserAgent will generate a random broswer user agent
func UserAgent() string { return userAgent(globalFaker.Rand) }

// UserAgent will generate a random broswer user agent
func (f *Faker) UserAgent() string { return userAgent(f.Rand) }

func userAgent(r *rand.Rand) string {
	randNum := randIntRange(r, 0, 4)
	switch randNum {
	case 0:
		return chromeUserAgent(r)
	case 1:
		return firefoxUserAgent(r)
	case 2:
		return safariUserAgent(r)
	case 3:
		return operaUserAgent(r)
	default:
		return chromeUserAgent(r)
	}
}

// ChromeUserAgent will generate a random chrome browser user agent string
func ChromeUserAgent() string { return chromeUserAgent(globalFaker.Rand) }

// ChromeUserAgent will generate a random chrome browser user agent string
func (f *Faker) ChromeUserAgent() string { return chromeUserAgent(f.Rand) }

func chromeUserAgent(r *rand.Rand) string {
	randNum1 := strconv.Itoa(randIntRange(r, 531, 536)) + strconv.Itoa(randIntRange(r, 0, 2))
	randNum2 := strconv.Itoa(randIntRange(r, 36, 40))
	randNum3 := strconv.Itoa(randIntRange(r, 800, 899))
	return "Mozilla/5.0 " + "(" + randomPlatform(r) + ") AppleWebKit/" + randNum1 + " (KHTML, like Gecko) Chrome/" + randNum2 + ".0." + randNum3 + ".0 Mobile Safari/" + randNum1
}

// FirefoxUserAgent will generate a random firefox broswer user agent string
func FirefoxUserAgent() string { return firefoxUserAgent(globalFaker.Rand) }

// FirefoxUserAgent will generate a random firefox broswer user agent string
func (f *Faker) FirefoxUserAgent() string { return firefoxUserAgent(f.Rand) }

func firefoxUserAgent(r *rand.Rand) string {
	ver := "Gecko/" + date(r).Format("2006-01-02") + " Firefox/" + strconv.Itoa(randIntRange(r, 35, 37)) + ".0"
	platforms := []string{
		"(" + windowsPlatformToken(r) + "; " + "en-US" + "; rv:1.9." + strconv.Itoa(randIntRange(r, 0, 3)) + ".20) " + ver,
		"(" + linuxPlatformToken(r) + "; rv:" + strconv.Itoa(randIntRange(r, 5, 8)) + ".0) " + ver,
		"(" + macPlatformToken(r) + " rv:" + strconv.Itoa(randIntRange(r, 2, 7)) + ".0) " + ver,
	}

	return "Mozilla/5.0 " + randomString(r, platforms)
}

// SafariUserAgent will generate a random safari browser user agent string
func SafariUserAgent() string { return safariUserAgent(globalFaker.Rand) }

// SafariUserAgent will generate a random safari browser user agent string
func (f *Faker) SafariUserAgent() string { return safariUserAgent(f.Rand) }

func safariUserAgent(r *rand.Rand) string {
	randNum := strconv.Itoa(randIntRange(r, 531, 536)) + "." + strconv.Itoa(randIntRange(r, 1, 51)) + "." + strconv.Itoa(randIntRange(r, 1, 8))
	ver := strconv.Itoa(randIntRange(r, 4, 6)) + "." + strconv.Itoa(randIntRange(r, 0, 2))

	mobileDevices := []string{
		"iPhone; CPU iPhone OS",
		"iPad; CPU OS",
	}

	platforms := []string{
		"(Windows; U; " + windowsPlatformToken(r) + ") AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + ver + " Safari/" + randNum,
		"(" + macPlatformToken(r) + " rv:" + strconv.Itoa(randIntRange(r, 4, 7)) + ".0; en-US) AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + ver + " Safari/" + randNum,
		"(" + randomString(r, mobileDevices) + " " + strconv.Itoa(randIntRange(r, 7, 9)) + "_" + strconv.Itoa(randIntRange(r, 0, 3)) + "_" + strconv.Itoa(randIntRange(r, 1, 3)) + " like Mac OS X; " + "en-US" + ") AppleWebKit/" + randNum + " (KHTML, like Gecko) Version/" + strconv.Itoa(randIntRange(r, 3, 5)) + ".0.5 Mobile/8B" + strconv.Itoa(randIntRange(r, 111, 120)) + " Safari/6" + randNum,
	}

	return "Mozilla/5.0 " + randomString(r, platforms)
}

// OperaUserAgent will generate a random opera browser user agent string
func OperaUserAgent() string { return operaUserAgent(globalFaker.Rand) }

// OperaUserAgent will generate a random opera browser user agent string
func (f *Faker) OperaUserAgent() string { return operaUserAgent(f.Rand) }

func operaUserAgent(r *rand.Rand) string {
	platform := "(" + randomPlatform(r) + "; en-US) Presto/2." + strconv.Itoa(randIntRange(r, 8, 13)) + "." + strconv.Itoa(randIntRange(r, 160, 355)) + " Version/" + strconv.Itoa(randIntRange(r, 10, 13)) + ".00"

	return "Opera/" + strconv.Itoa(randIntRange(r, 8, 10)) + "." + strconv.Itoa(randIntRange(r, 10, 99)) + " " + platform
}

// linuxPlatformToken will generate a random linux platform
func linuxPlatformToken(r *rand.Rand) string {
	return "X11; Linux " + getRandValue(r, []string{"computer", "linux_processor"})
}

// macPlatformToken will generate a random mac platform
func macPlatformToken(r *rand.Rand) string {
	return "Macintosh; " + getRandValue(r, []string{"computer", "mac_processor"}) + " Mac OS X 10_" + strconv.Itoa(randIntRange(r, 5, 9)) + "_" + strconv.Itoa(randIntRange(r, 0, 10))
}

// windowsPlatformToken will generate a random windows platform
func windowsPlatformToken(r *rand.Rand) string {
	return getRandValue(r, []string{"computer", "windows_platform"})
}

// randomPlatform will generate a random platform
func randomPlatform(r *rand.Rand) string {
	platforms := []string{
		linuxPlatformToken(r),
		macPlatformToken(r),
		windowsPlatformToken(r),
	}

	return randomString(r, platforms)
}

// HTTPVersion will generate a random http version
func HTTPVersion() string { return httpVersion(globalFaker.Rand) }

// HTTPVersion will generate a random http version
func (f *Faker) HTTPVersion() string { return httpVersion(f.Rand) }

func httpVersion(r *rand.Rand) string {
	return getRandValue(r, []string{"internet", "http_version"})
}

func addInternetLookup() {
	AddFuncLookup("url", Info{
		Display:     "URL",
		Category:    "internet",
		Description: "Web address that specifies the location of a resource on the internet",
		Example:     "http://www.principalproductize.biz/target",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return url(r), nil
		},
	})

	AddFuncLookup("domainname", Info{
		Display:     "Domain Name",
		Category:    "internet",
		Description: "Human-readable web address used to identify websites on the internet",
		Example:     "centraltarget.biz",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return domainName(r), nil
		},
	})

	AddFuncLookup("domainsuffix", Info{
		Display:     "Domain Suffix",
		Category:    "internet",
		Description: "The part of a domain name that comes after the last dot, indicating its type or purpose",
		Example:     "org",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return domainSuffix(r), nil
		},
	})

	AddFuncLookup("ipv4address", Info{
		Display:     "IPv4 Address",
		Category:    "internet",
		Description: "Numerical label assigned to devices on a network for identification and communication",
		Example:     "222.83.191.222",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return ipv4Address(r), nil
		},
	})

	AddFuncLookup("ipv6address", Info{
		Display:     "IPv6 Address",
		Category:    "internet",
		Description: "Numerical label assigned to devices on a network, providing a larger address space than IPv4 for internet communication",
		Example:     "2001:cafe:8898:ee17:bc35:9064:5866:d019",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return ipv6Address(r), nil
		},
	})

	AddFuncLookup("httpmethod", Info{
		Display:     "HTTP Method",
		Category:    "internet",
		Description: "Verb used in HTTP requests to specify the desired action to be performed on a resource",
		Example:     "HEAD",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return httpMethod(r), nil
		},
	})

	AddFuncLookup("loglevel", Info{
		Display:     "Log Level",
		Category:    "internet",
		Description: "Classification used in logging to indicate the severity or priority of a log entry",
		Example:     "error",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return logLevel(r, ""), nil
		},
	})

	AddFuncLookup("useragent", Info{
		Display:     "User Agent",
		Category:    "internet",
		Description: "String sent by a web browser to identify itself when requesting web content",
		Example:     "Mozilla/5.0 (Windows NT 5.0) AppleWebKit/5362 (KHTML, like Gecko) Chrome/37.0.834.0 Mobile Safari/5362",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return userAgent(r), nil
		},
	})

	AddFuncLookup("chromeuseragent", Info{
		Display:     "Chrome User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Google Chrome web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (X11; Linux i686) AppleWebKit/5312 (KHTML, like Gecko) Chrome/39.0.836.0 Mobile Safari/5312",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return chromeUserAgent(r), nil
		},
	})

	AddFuncLookup("firefoxuseragent", Info{
		Display:     "Firefox User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Firefox web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (Macintosh; U; PPC Mac OS X 10_8_3 rv:7.0) Gecko/1900-07-01 Firefox/37.0",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return firefoxUserAgent(r), nil
		},
	})

	AddFuncLookup("operauseragent", Info{
		Display:     "Opera User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Opera web browser when making requests on the internet",
		Example:     "Opera/8.39 (Macintosh; U; PPC Mac OS X 10_8_7; en-US) Presto/2.9.335 Version/10.00",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return operaUserAgent(r), nil
		},
	})

	AddFuncLookup("safariuseragent", Info{
		Display:     "Safari User Agent",
		Category:    "internet",
		Description: "The specific identification string sent by the Safari web browser when making requests on the internet",
		Example:     "Mozilla/5.0 (iPad; CPU OS 8_3_2 like Mac OS X; en-US) AppleWebKit/531.15.6 (KHTML, like Gecko) Version/4.0.5 Mobile/8B120 Safari/6531.15.6",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return safariUserAgent(r), nil
		},
	})

	AddFuncLookup("httpstatuscode", Info{
		Display:     "HTTP Status Code",
		Category:    "internet",
		Description: "Random http status code",
		Example:     "200",
		Output:      "int",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return httpStatusCode(r), nil
		},
	})

	AddFuncLookup("httpstatuscodesimple", Info{
		Display:     "HTTP Status Code Simple",
		Category:    "internet",
		Description: "Three-digit number returned by a web server to indicate the outcome of an HTTP request",
		Example:     "404",
		Output:      "int",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return httpStatusCodeSimple(r), nil
		},
	})

	AddFuncLookup("httpversion", Info{
		Display:     "HTTP Version",
		Category:    "internet",
		Description: "Number indicating the version of the HTTP protocol used for communication between a client and a server",
		Example:     "HTTP/1.1",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return httpVersion(r), nil
		},
	})

	AddFuncLookup("macaddress", Info{
		Display:     "MAC Address",
		Category:    "internet",
		Description: "Unique identifier assigned to network interfaces, often used in Ethernet networks",
		Example:     "cb:ce:06:94:22:e9",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return macAddress(r), nil
		},
	})
}
