package secrets

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// Candidate subsets audited against Betterleaks 95237cf8eb4d; see
// LICENSE.betterleaks. Issuer facts and local scanner bounds are separated in
// the corresponding fixture evidence. No network verification is performed.
var betterleaksCarriersRules4 = []catalogRuleSpec{
	{
		ID:              "freemius-secret-key",
		Regex:           `(?:"secret_key"|'secret_key')[ \t]*=>[ \t]*("sk_[^"\\\s]{29}"|'sk_[^'\\\s]{29}')`,
		Keywords:        []string{"secret_key"},
		SecretGroup:     1,
		Source:          "https://github.com/Freemius/wordpress-sdk/blob/master/README.md",
		Description:     "Freemius secret_key PHP array entry, not public_key or a bare sk_ value. The 29-character body is the pinned scanner subset. Single-quoted dollar characters remain literal; double-quoted interpolation, references and expression continuations are excluded. PHP carrier syntax substitutes for unavailable file-path context.",
		Validate:        validBetterStructuredFreemius4,
		ValidateContext: validateBetterStructuredAssignment4,
	},
	{
		ID:              "service-userinfo-credential",
		Regex:           `\b((?i:mariadb|ldaps?|smtps?|ssh)://[^\s"<>\x60]+@[^\s"<>\x60]+)`,
		Keywords:        []string{"mariadb://", "ldap://", "ldaps://", "smtp://", "smtps://", "ssh://"},
		SecretGroup:     1,
		Source:          "https://www.rfc-editor.org/rfc/rfc3986#section-3.2.1",
		Description:     "Parsed MariaDB, LDAP(S), SMTP(S) or SSH URI userinfo with a nonempty password. This recognizes an explicit credential carrier, not client support or successful authentication. Requires a single well-framed authority, checks IPv6 syntax, decodes credentials once and excludes references/placeholders. Local 16 KiB bound; FTP(S) retains its existing native family.",
		ValidateContext: validateBetterStructuredServiceURI4,
	},
	{
		ID:              "terraform-administrator-login-password",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:administrator_login_password(?:_wo)?)[ \t]*=[ \t]*("(?:\\(?:["\\nrt]|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})|[^"\\\r\n])+")`,
		Keywords:        []string{"administrator_login_password"},
		SecretGroup:     1,
		Source:          "https://github.com/hashicorp/terraform-provider-azurerm/blob/main/website/docs/r/mssql_server.html.markdown",
		Description:     "Exact Terraform administrator_login_password or write-only administrator_login_password_wo assignment containing a quoted literal. Generic password fields and file-name-only proximity are not classified. Escapes are decoded locally; interpolation, expressions, empty values and placeholders are excluded. The 1000-byte candidate bound is local, not password policy.",
		Validate:        validBetterStructuredTerraform4,
		ValidateContext: validateBetterStructuredAssignment4,
	},
	{
		ID:          "nuget-cleartext-password",
		Regex:       `(?i:<packageSourceCredentials[ \t\r\n]*>[\s\S]*?</packageSourceCredentials[ \t\r\n]*>)`,
		Keywords:    []string{"packageSourceCredentials"},
		Source:      "https://learn.microsoft.com/en-us/nuget/reference/nuget-config-file#packagesourcecredentials",
		Description: "NuGet packageSourceCredentials XML containing a source child and an add element whose key is ClearTextPassword. XML framing, attribute uniqueness and entities are parsed locally with a 16 KiB cap. Encrypted Password entries, environment references, malformed XML and unrelated add elements are excluded; the section supplies value-only NuGet scope.",
		Validate:    validBetterStructuredNuget4,
	},
	{
		ID:              "snowflake-programmatic-access-token",
		Regex:           betterStructuredSnowflakePair4 + `|` + auditedProviderRequestPattern2,
		Keywords:        []string{"snowflake", curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions},
		Source:          "https://docs.snowflake.com/en/user-guide/programmatic-access-tokens",
		Description:     "Snowflake PAT-labelled token paired with an adjacent named account host in either order, or a complete HTTP(S) Snowflake request with Bearer and PROGRAMMATIC_ACCESS_TOKEN type headers. Public account hosts and arbitrary token proximity are not credentials. URL-safe 100-500-character bodies and adjacent-field limits are scanner bounds, not an issuer grammar; request parsing reuses the bounded native parser.",
		ValidateContext: validateBetterStructuredSnowflake4,
	},
	{
		ID:              "temporal-cloud-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:TEMPORAL_CLOUD_API_KEY)["']?[ \t]*[:=][ \t]*["']?(eyJ[A-Za-z0-9_-]{10,1000}\.[A-Za-z0-9_-]{20,1000}\.[A-Za-z0-9_-]{20,1000})` + catalogRightBoundary,
		Keywords:        []string{"TEMPORAL_CLOUD_API_KEY"},
		SecretGroup:     1,
		Source:          "https://github.com/temporalio/terraform-provider-temporalcloud/blob/main/docs/index.md",
		Description:     "Temporal Cloud's documented API-key assignment carrying a signed JWT with temporal.io issuer and nonempty account_id/key_id strings. Reuses strict compact-JWT and algorithm-specific signature-size validation. Claim markers and segment bounds are scanner recognition constraints, not issuer guarantees; bare JWTs are not attributed without provider carrier evidence.",
		Validate:        validBetterStructuredTemporal4,
		ValidateContext: validateBetterStructuredAssignment4,
	},
	{
		ID:              "upstash-redis-rest-token",
		Regex:           betterStructuredUpstashPair4 + `|` + auditedProviderRequestPattern2,
		Keywords:        []string{"upstash", curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions},
		Source:          "https://upstash.com/docs/redis/features/restapi",
		Description:     "Adjacent UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN literal assignments in either order, or a complete HTTP(S) Upstash request carrying a Bearer token. Includes confidential read-only tokens, not public endpoint URLs. Opaque 32-48-character alphanumeric and AYNgAS-prefixed canonical Base64 candidates are pinned scanner subsets, not issuance guarantees. Literal framing and complete request destination are checked offline.",
		ValidateContext: validateBetterStructuredUpstash4,
	},
}

const (
	// #nosec G101 -- Regular expression for Snowflake token shapes, not a credential value.
	betterStructuredSnowflakeTokenBody4 = `[A-Za-z0-9_-]{100,500}`
	betterStructuredSnowflakeHostBody4  = `(?:https://)?[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*\.snowflakecomputing\.com`
	// #nosec G101 -- Regular expression for Snowflake token field names, not a credential value.
	betterStructuredSnowflakeTokenName4  = `(?i:snowflake_(?:programmatic_)?(?:access_)?token|sf_token)`
	betterStructuredSnowflakeHostName4   = `(?i:snowflake_(?:account_host|host|url|account))`
	betterStructuredSnowflakeTokenField4 = `["']?` + betterStructuredSnowflakeTokenName4 + `["']?[ \t]*[:=][ \t]*(?:"` + betterStructuredSnowflakeTokenBody4 + `"|'` + betterStructuredSnowflakeTokenBody4 + `'|` + betterStructuredSnowflakeTokenBody4 + `)`
	betterStructuredSnowflakeHostField4  = `["']?` + betterStructuredSnowflakeHostName4 + `["']?[ \t]*[:=][ \t]*(?:"` + betterStructuredSnowflakeHostBody4 + `"|'` + betterStructuredSnowflakeHostBody4 + `'|` + betterStructuredSnowflakeHostBody4 + `)`
	betterStructuredSnowflakePair4       = `(?:^|[^A-Za-z0-9_.-])(?:` + betterStructuredSnowflakeTokenField4 + `[ \t\r\n,;]{1,32}` + betterStructuredSnowflakeHostField4 + `|` + betterStructuredSnowflakeHostField4 + `[ \t\r\n,;]{1,32}` + betterStructuredSnowflakeTokenField4 + `)` + catalogRightBoundary
	// #nosec G101 -- Regular expression for Upstash token shapes, not a credential value.
	betterStructuredUpstashTokenBody4 = `(?:[A-Za-z0-9]{32,48}|AYNgAS[A-Za-z0-9+/_-]{26,90}={0,2})`
	betterStructuredUpstashHostBody4  = `https://[A-Za-z0-9][A-Za-z0-9-]{2,63}\.upstash\.io`
	// #nosec G101 -- Regular expression for an Upstash token field name, not a credential value.
	betterStructuredUpstashTokenName4  = `(?i:UPSTASH_REDIS_REST_TOKEN)`
	betterStructuredUpstashHostName4   = `(?i:UPSTASH_REDIS_REST_URL)`
	betterStructuredUpstashTokenField4 = `["']?` + betterStructuredUpstashTokenName4 + `["']?[ \t]*[:=][ \t]*(?:"` + betterStructuredUpstashTokenBody4 + `"|'` + betterStructuredUpstashTokenBody4 + `'|` + betterStructuredUpstashTokenBody4 + `)`
	betterStructuredUpstashHostField4  = `["']?` + betterStructuredUpstashHostName4 + `["']?[ \t]*[:=][ \t]*(?:"` + betterStructuredUpstashHostBody4 + `"|'` + betterStructuredUpstashHostBody4 + `'|` + betterStructuredUpstashHostBody4 + `)`
	betterStructuredUpstashPair4       = `(?:^|[^A-Za-z0-9_.-])(?:` + betterStructuredUpstashTokenField4 + `[ \t\r\n,;]{1,32}` + betterStructuredUpstashHostField4 + `|` + betterStructuredUpstashHostField4 + `[ \t\r\n,;]{1,32}` + betterStructuredUpstashTokenField4 + `)` + catalogRightBoundary
)

var (
	betterStructuredSnowflakeToken4       = newLazyRegexp(betterStructuredSnowflakeTokenName4 + `["']?[ \t]*[:=][ \t]*["']?(` + betterStructuredSnowflakeTokenBody4 + `)`)
	betterStructuredSnowflakeHost4        = newLazyRegexp(betterStructuredSnowflakeHostName4 + `["']?[ \t]*[:=][ \t]*["']?(` + betterStructuredSnowflakeHostBody4 + `)`)
	betterStructuredUpstashToken4         = newLazyRegexp(betterStructuredUpstashTokenName4 + `["']?[ \t]*[:=][ \t]*["']?(` + betterStructuredUpstashTokenBody4 + `)`)
	betterStructuredUpstashHost4          = newLazyRegexp(betterStructuredUpstashHostName4 + `["']?[ \t]*[:=][ \t]*["']?(` + betterStructuredUpstashHostBody4 + `)`)
	betterStructuredSnowflakeHostPattern4 = newLazyRegexp(`^[a-z0-9_-]+(?:\.[a-z0-9_-]+)*\.snowflakecomputing\.com$`)
	betterStructuredUpstashHostPattern4   = newLazyRegexp(`^[a-z0-9][a-z0-9-]{2,63}\.upstash\.io$`)
	betterStructuredRequestPath4          = newLazyRegexp(`^/[^\s]*$`)
	betterStructuredSnowflakeBodyPattern4 = newLazyRegexp(`^` + betterStructuredSnowflakeTokenBody4 + `$`)
	betterStructuredUpstashBodyPattern4   = newLazyRegexp(`^` + betterStructuredUpstashTokenBody4 + `$`)
	betterStructuredNugetReference4       = newLazyRegexp(`%[A-Za-z_][A-Za-z0-9_()]*%`)
)

func validBetterStructuredFreemius4(s string) bool {
	if len(s) != 34 || s[0] != s[len(s)-1] || !strings.HasPrefix(s[1:], "sk_") {
		return false
	}
	return validAuditedCarrierLiteral2(s[1:len(s)-1]) && (s[0] == '\'' || !strings.ContainsAny(s[1:len(s)-1], "$`"))
}

func validBetterStructuredTerraform4(s string) bool {
	if len(s) > 1002 || strings.Contains(s, "${") || strings.Contains(s, "%{") {
		return false
	}
	decoded, err := strconv.Unquote(s)
	return err == nil && validAuditedCarrierLiteral2(decoded)
}

// Supplement the common assignment guard with post-quote continuation checks.
// Only literal assignments use this helper; bare prefixed tokens do not.
func validateBetterStructuredAssignment4(value string, start, end int, secret string) contextValidation {
	if !validateAuditedAssignmentContext(value, start, end, secret).accepted {
		return contextValidation{}
	}
	relative := strings.LastIndex(value[start:end], secret)
	if relative < 0 {
		return contextValidation{}
	}
	left := start + relative
	right := left + len(secret)
	quoted := len(secret) >= 2 && (secret[0] == '\'' || secret[0] == '"') && secret[len(secret)-1] == secret[0]
	if !quoted && left > start && (value[left-1] == '\'' || value[left-1] == '"') {
		if right >= len(value) || value[right] != value[left-1] {
			return contextValidation{}
		}
		right++
	}
	for right < len(value) && (value[right] == ' ' || value[right] == '\t') {
		right++
	}
	return contextValidation{accepted: right == len(value) || strings.ContainsRune("\r\n,;})]#", rune(value[right]))}
}

func validateBetterStructuredServiceURI4(value string, start, end int, secret string) contextValidation {
	if start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.User == nil || u.Hostname() == "" || !connectionURIHost(u.Host) || strings.ContainsAny(u.Hostname(), ",;\\") {
		return contextValidation{}
	}
	if strings.HasPrefix(u.Host, "[") {
		if address, err := netip.ParseAddr(u.Hostname()); err != nil || !address.Is6() {
			return contextValidation{}
		}
	}
	password, set := u.User.Password()
	if !set || !validAuditedCarrierLiteral2(password) || !validCarrier3Literal(password) {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

func validBetterStructuredNuget4(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	decoder := xml.NewDecoder(strings.NewReader(s))
	depth, roots := 0, 0
	found := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return found && roots == 1 && depth == 0
		}
		if err != nil {
			return false
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			if t.Name.Space != "" || depth > 3 {
				return false
			}
			switch depth {
			case 1:
				roots++
				if roots != 1 || !strings.EqualFold(t.Name.Local, "packageSourceCredentials") || len(t.Attr) != 0 {
					return false
				}
			case 2:
				if len(t.Attr) != 0 {
					return false
				}
			default:
				if !strings.EqualFold(t.Name.Local, "add") {
					return false
				}
				var key, value string
				keySet, valueSet := false, false
				for _, attr := range t.Attr {
					if attr.Name.Space != "" {
						return false
					}
					switch strings.ToLower(attr.Name.Local) {
					case keyField:
						if keySet {
							return false
						}
						key, keySet = attr.Value, true
					case valueField:
						if valueSet {
							return false
						}
						value, valueSet = attr.Value, true
					default:
						return false
					}
				}
				if !keySet || !valueSet {
					return false
				}
				if strings.EqualFold(key, "ClearTextPassword") && validAuditedCarrierLiteral2(value) && validCarrier3Literal(value) && !betterStructuredNugetReference4.MatchString(value) && value != "33f!!lloppa" && value != "hal+9ooo_da!sY" {
					found = true
				}
			}
		case xml.EndElement:
			depth--
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return false
			}
		case xml.Comment:
		default:
			return false
		}
	}
}

func validBetterStructuredTemporal4(s string) bool {
	if !validJWT(s) {
		return false
	}
	_, rest, _ := strings.Cut(s, ".")
	payload, _, _ := strings.Cut(rest, ".")
	decoded, ok := decodeCredentialBase64URL(payload)
	if !ok {
		return false
	}
	var fields struct {
		Issuer    string `json:"iss"`
		AccountID string `json:"account_id"`
		KeyID     string `json:"key_id"`
	}
	return json.Unmarshal(decoded, &fields) == nil && fields.Issuer == "temporal.io" && validAuditedCarrierLiteral2(fields.AccountID) && validAuditedCarrierLiteral2(fields.KeyID)
}

// Used by the existing-family Scalr proposal. Its old 136-byte paired scope is
// preserved separately; a new standalone carrier must contain a parsed JWT.
func validBetterStructuredScalrJWT4(s string) bool {
	if !validJWT(s) {
		return false
	}
	header, rest, _ := strings.Cut(s, ".")
	payload, _, _ := strings.Cut(rest, ".")
	h, ok := decodeCredentialBase64URL(header)
	if !ok {
		return false
	}
	p, ok := decodeCredentialBase64URL(payload)
	if !ok {
		return false
	}
	var algorithm struct {
		Alg  string `json:"alg"`
		Type string `json:"typ"`
	}
	var claims struct {
		Issuer string `json:"iss"`
		ID     string `json:"jti"`
	}
	if json.Unmarshal(h, &algorithm) != nil || json.Unmarshal(p, &claims) != nil || algorithm.Alg != "HS256" || algorithm.Type != "JWT" {
		return false
	}
	id, present := strings.CutPrefix(claims.ID, "at-")
	return claims.Issuer == userField && present && validAuditedCarrierLiteral2(id)
}

func validateBetterStructuredScalr4(value string, start, end int, secret string) contextValidation {
	if indexFoldedASCII(value[start:end], "scalr_hostname") >= 0 && len(secret) == 136 {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	return contextValidation{accepted: validBetterStructuredScalrJWT4(secret) && validateBetterStructuredAssignment4(value, start, end, secret).accepted}
}

func betterStructuredFieldValue4(value string, start, end int, field *lazyRegexp) (string, bool) {
	m := field.FindStringSubmatchIndex(value[start:end])
	if m == nil {
		return "", false
	}
	nameStart := start + m[0]
	binder := strings.IndexAny(value[nameStart:start+m[2]], ":=")
	if binder < 0 {
		return "", false
	}
	key := strings.TrimSpace(value[nameStart : nameStart+binder])
	keyQuote := byte(0)
	if strings.HasSuffix(key, "\"") || strings.HasSuffix(key, "'") {
		keyQuote = key[len(key)-1]
	}
	if keyQuote != 0 {
		if nameStart == 0 || value[nameStart-1] != keyQuote {
			return "", false
		}
	} else if nameStart > 0 && (value[nameStart-1] == '\'' || value[nameStart-1] == '"') {
		return "", false
	}
	secret := value[start+m[2] : start+m[3]]
	return secret, validateBetterStructuredAssignment4(value, start+m[0], start+m[1], secret).accepted
}

func validateBetterStructuredSnowflake4(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(value[start:end], curlCommand) || strings.HasPrefix(value[start:end], "GET ") || strings.HasPrefix(value[start:end], "POST ") || strings.HasPrefix(value[start:end], "PUT ") || strings.HasPrefix(value[start:end], "PATCH ") || strings.HasPrefix(value[start:end], "DELETE ") || strings.HasPrefix(value[start:end], "HEAD ") || strings.HasPrefix(value[start:end], "OPTIONS ") {
		return betterStructuredSnowflakeRequest4(value, start, end, secret)
	}
	token, ok := betterStructuredFieldValue4(value, start, end, betterStructuredSnowflakeToken4)
	if !ok || !validAuditedCarrierLiteral2(token) {
		return contextValidation{}
	}
	host, ok := betterStructuredFieldValue4(value, start, end, betterStructuredSnowflakeHost4)
	if !ok {
		return contextValidation{}
	}
	if !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	_, ok = auditedProviderURL2(host, betterStructuredSnowflakeHostPattern4, betterStructuredRequestPath4)
	return contextValidation{accepted: ok}
}

var betterStructuredSnowflakeRequest4 = auditedProviderRequestValidator2(
	`[a-z0-9_-]+(?:\.[a-z0-9_-]+)*\.snowflakecomputing\.com`, `/[^\s]*`,
	[]auditedProviderHeader2{{name: authorizationHeader, prefix: bearerPrefix, pattern: betterStructuredSnowflakeBodyPattern4}}, nil,
	[]auditedProviderField2{{name: "X-Snowflake-Authorization-Token-Type", pattern: newLazyRegexp(`^PROGRAMMATIC_ACCESS_TOKEN$`)}}, false,
)

func validBetterStructuredUpstashToken4(s string) bool {
	if !betterStructuredUpstashBodyPattern4.MatchString(s) || !validAuditedCarrierLiteral2(s) {
		return false
	}
	if !strings.HasPrefix(s, "AYNgAS") {
		return true
	}
	encoding := serviceRawBase64
	if strings.ContainsAny(s, "-_") {
		encoding = serviceRawBase64URL
		if strings.HasSuffix(s, "=") {
			raw := strings.TrimRight(s, "=")
			if len(s)%4 != 0 || len(s)-len(raw) > 2 {
				return false
			}
			s = raw
		}
	} else if strings.HasSuffix(s, "=") {
		encoding = serviceBase64
	}
	var decoded [74]byte
	n, err := encoding.Decode(decoded[:], []byte(s))
	return err == nil && n > 0
}

func validateBetterStructuredUpstash4(value string, start, end int, _ string) contextValidation {
	if indexFoldedASCII(value[start:end], "upstash_redis_rest_") >= 0 {
		token, ok := betterStructuredFieldValue4(value, start, end, betterStructuredUpstashToken4)
		if !ok || !validBetterStructuredUpstashToken4(token) {
			return contextValidation{}
		}
		host, ok := betterStructuredFieldValue4(value, start, end, betterStructuredUpstashHost4)
		if !ok {
			return contextValidation{}
		}
		_, ok = auditedProviderURL2(host, betterStructuredUpstashHostPattern4, betterStructuredRequestPath4)
		return contextValidation{accepted: ok}
	}
	r, ok := parseAuditedProviderRequestContext2(value, start, end)
	if !ok {
		return contextValidation{}
	}
	u, ok := auditedProviderURL2(r.url, betterStructuredUpstashHostPattern4, betterStructuredRequestPath4)
	if !ok {
		return contextValidation{}
	}
	authCount, hostCount := 0, 0
	for _, line := range r.headers[:r.headerCount] {
		name, content, found := strings.Cut(line, ":")
		if !found {
			return contextValidation{}
		}
		content = strings.TrimSpace(content)
		if strings.EqualFold(name, "Host") {
			hostCount++
			if hostCount > 1 || !strings.EqualFold(content, u.Host) {
				return contextValidation{}
			}
		}
		if strings.EqualFold(name, authorizationHeader) {
			authCount++
			if authCount > 1 || len(content) < len(bearerPrefix) || !strings.EqualFold(content[:len(bearerPrefix)], bearerPrefix) || !validBetterStructuredUpstashToken4(content[len(bearerPrefix):]) {
				return contextValidation{}
			}
		}
	}
	return contextValidation{accepted: authCount == 1}
}
