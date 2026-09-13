package secrets

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/netip"
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
)

// Portions adapted from Gitleaks v8.30.1; see LICENSE.gitleaks.
var structuredRuleSpecs = []catalogRuleSpec{
	{
		ID:          "1password-secret-key",
		Regex:       `(?:\bA3-[A-Z0-9]{6}-(?:(?:[A-Z0-9]{11})|(?:[A-Z0-9]{6}-[A-Z0-9]{5}))-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}\b|\bA3[A-Z0-9]{32}\b)`,
		Keywords:    []string{"A3"},
		SecretGroup: 0,
		Entropy:     3.8,
		Source:      "https://1passwordstatic.com/files/security/1password-white-paper.pdf",
		Description: "1Password A3 Secret Key candidate using Gitleaks v8.30.1 grouping, boundaries, alphabet and entropy, plus the unhyphenated representation: the whitepaper states that hyphens are not part of the key. The provider documents 34 alphanumeric characters but does not enumerate the 31-character generation alphabet; the upstream candidate alphabet is not issuer validation. Arbitrary separator placement and other versions are not inferred.",
	},
	{
		ID:          "1password-service-account-token",
		Regex:       `ops_eyJ[a-zA-Z0-9+/_-]{250,}={0,3}`,
		Keywords:    []string{"ops_"},
		SecretGroup: 0,
		Entropy:     4,
		Source:      "https://www.1password.dev/service-accounts/security",
		Description: "1Password service-account token candidate using Gitleaks v8.30.1 opening, minimum length and entropy, with the provider-documented Base64url transport alongside upstream Base64 and padding. Requires a decoded UTF-8 JSON object, but does not infer required SRPx fields, cryptographic component widths or their relationships from an example. The 16 KiB candidate cap is local. The documentation's JSON/JWT terminology is ambiguous; this recognizer covers the displayed serialized-object carrier, not issuer validity or authentication.",
		Validate:    validOnePasswordServiceToken,
	},
	{
		ID:          "age-secret-key",
		Regex:       `\b(AGE-SECRET-KEY-1[023456789ACDEFGHJKLMNPQRSTUVWXYZ]{58})` + catalogRightBoundary,
		Keywords:    []string{"AGE-SECRET-KEY-1"},
		SecretGroup: 1,
		Source:      "https://github.com/FiloSottile/age/blob/main/x25519.go",
		Description: "Canonical uppercase age X25519 private identity: exactly 32 secret bytes in Bech32, with checksum and zero padding verified. Public age1 recipients, plugin identities, and encrypted identity files are not this format.",
		Validate:    validAgeSecretKey,
	},
	{
		ID:          "age-secret-key-pq",
		Regex:       `\b(AGE-SECRET-KEY-PQ-1[023456789ACDEFGHJKLMNPQRSTUVWXYZ]{58})` + catalogRightBoundary,
		Keywords:    []string{"AGE-SECRET-KEY-PQ-1"},
		SecretGroup: 1,
		Source:      "https://github.com/FiloSottile/age/blob/main/pq.go",
		Description: "Canonical uppercase age MLKEM768-X25519 private identity: a 32-byte hybrid seed encoded with Bech32 checksum and canonical padding. Seed length follows the vendor's filippo.io/hpke hybridKEM.NewPrivateKey implementation; public age1pq1 recipients are excluded.",
		Validate:    validAgeSecretKey,
	},
	{
		ID:          "jwt",
		Regex:       `\b(?i:Bearer)[ \t]+([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)` + catalogRightBoundary,
		Keywords:    []string{"bearer"},
		SecretGroup: 1,
		Source:      "https://www.rfc-editor.org/rfc/rfc6750#section-2.1",
		Description: "Explicit Bearer authentication carrier containing a signed compact JWT (RFC 7519/7515). Strict base64url, UTF-8 JSON objects, and JWA/EdDSA signature sizes are checked locally. Bare JWT containers, unsecured alg=none tokens, JWE, and unknown algorithms are excluded. The 16 KiB candidate and 8192-bit RSA signature bounds are defensive limits, not protocol maxima. No signature, issuer, expiry, or live-token verification is performed.",
		Validate:    validJWT,
	},
	{
		ID:              "postgres-connection-uri",
		Regex:           `\b(postgres(?:ql)?://[^\s"<>\x60]+)`,
		Keywords:        []string{"postgres://", "postgresql://"},
		SecretGroup:     1,
		Source:          "https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING",
		Description:     "PostgreSQL connection URI with a nonempty authority or named-query password, including one-character credentials, optional/default/socket hosts, bracketed IPv6 and multihost lists. Components are parsed before percent decoding, with libpq query rather than HTML form semantics. Any literal password remains sensitive even if a later parameter overrides it. URI syntax, IPv6 structure, decimal ports and host/port list cardinality are checked locally; unknown option names are accepted and DNS, other option values, defaults and authentication are not checked. Whitespace and double-quote/backtick/angle-bracket delimiters bound maximal candidates; single-quote wrappers are recognized contextually. Raw fragments/non-URI characters and malformed suffixes are rejected, not truncated. The 16 KiB cap and decimal 1-65535 port constraint are conservative scanner limits, not full libpq equivalence.",
		ValidateContext: validatePostgresConnectionURI,
	},
	{
		ID:              "mysql-connection-uri",
		Regex:           `\b(mysql(?:x(?:\+srv)?)?://[^\s"<>\x60]+)`,
		Keywords:        []string{"mysql://", "mysqlx://", "mysqlx+srv://"},
		SecretGroup:     1,
		Source:          "https://dev.mysql.com/doc/refman/8.4/en/connecting-using-uri-or-key-value-pairs.html",
		Description:     "MySQL mysql://, mysqlx:// and mysqlx+srv:// URI-like strings containing a nonempty userinfo password, including one-character and percent-escaped credentials, IPv6 and encoded or parenthesized Unix sockets. SRV uses one host without a port. Bare scheme-less strings, JDBC URLs, host-property credentials and query-only passwords are not inferred. Complete candidates are bounded at 16 KiB and malformed suffixes are retained; DNS, port ranges, option values and connection success are not validated.",
		ValidateContext: validatePasswordConnectionURI,
	},
	{
		ID:              "mongodb-connection-uri",
		Regex:           `\b(mongodb(?:\+srv)?://[^\s"<>\x60]+)`,
		Keywords:        []string{"mongodb://", "mongodb+srv://"},
		SecretGroup:     1,
		Source:          "https://github.com/mongodb/specifications/blob/master/source/connection-string/connection-string-spec.md",
		Description:     "MongoDB mongodb:// and mongodb+srv:// connection strings containing a nonempty userinfo password, including one-character and percent-escaped credentials, IPv6, encoded socket hosts and standard multihost lists. SRV requires a single hostname without a port; queries require the separating slash. Authentication passwords are not inferred from arbitrary query options. Complete candidates are bounded at 16 KiB and malformed suffixes are retained; DNS, port ranges, option values and connection success are not validated.",
		ValidateContext: validatePasswordConnectionURI,
	},
	{
		ID:              "redis-connection-uri",
		Regex:           `\b([Rr][Ee][Dd][Ii][Ss][Ss]?://[^\s"<>\x60]+)`, // URI scheme folding is ASCII-only.
		Keywords:        []string{"redis://", "rediss://"},
		SecretGroup:     1,
		Source:          "https://www.iana.org/assignments/uri-schemes/prov/redis",
		Description:     "Redis redis:// and rediss:// URIs containing a nonempty userinfo or exact password query value. Covers username/password, empty/default usernames and redis-cli's password-only userinfo, one-character and percent-escaped credentials, optional hosts and IPv6. A lone userinfo value is sensitive because redis-cli treats it as a password, even where other clients treat it as a username. Complete candidates are bounded at 16 KiB and malformed suffixes are retained; DNS, port ranges, option values and connection success are not validated.",
		ValidateContext: validatePasswordConnectionURI,
	},
	{
		ID:              "amqp-connection-uri",
		Regex:           `\b([Aa][Mm][Qq][Pp][Ss]?://[^\s"<>\x60]+)`,
		Keywords:        []string{"amqp://", "amqps://"},
		SecretGroup:     1,
		Source:          "https://www.rabbitmq.com/docs/uri-spec",
		Description:     "RabbitMQ AMQP 0-9-1 amqp:// and amqps:// URIs containing a nonempty userinfo password, including one-character and percent-escaped credentials, optional or empty usernames/hosts, IPv6 and an optional single-segment virtual host. Passwordless addresses, implicit guest defaults and arbitrary query password text are not credentials. Complete candidates are bounded at 16 KiB and malformed suffixes are retained; DNS, port ranges, option values and connection success are not validated.",
		ValidateContext: validatePasswordConnectionURI,
	},
	{
		ID:          "private-key",
		Regex:       `(?m)^(-----BEGIN (?:RSA PRIVATE KEY|PRIVATE KEY|EC PRIVATE KEY|DSA PRIVATE KEY|OPENSSH PRIVATE KEY)-----[ \t]*(?:\r\n|[\r\n])[A-Za-z0-9+/=\r\n \t]+-----END (?:RSA PRIVATE KEY|PRIVATE KEY|EC PRIVATE KEY|DSA PRIVATE KEY|OPENSSH PRIVATE KEY)-----)[ \t]*\r?$`,
		Keywords:    []string{"-----BEGIN "},
		SecretGroup: 1,
		Source:      "https://github.com/golang/crypto/blob/v0.55.0/ssh/keys.go",
		Description: "Matching-label PEM armor containing parsed unencrypted PKCS#1 RSA, PKCS#8, SEC1 EC, legacy DSA, or OpenSSH RSA/EC/Ed25519 private keys. LF/CRLF armor and standalone CR-only armor are recognized; a header must start the value or follow LF. Public keys, arbitrary armor/prose, encrypted keys, PGP, and unsupported algorithms are excluded. The 16 KiB armor and 3072-bit legacy DSA bounds are local defensive limits. OpenSSH framing follows openssh-portable PROTOCOL.key; PEM framing follows RFC 7468.",
		Validate:    validPrivateKey,
	},
}

// These bounds apply only after a regex candidate has been extracted. They are
// work limits for structural recognition, not limits imposed by the formats.
const maxStructuredCredentialBytes = 16 * 1024

func validatePostgresConnectionURI(value string, matchStart, matchEnd int, secret string) contextValidation {
	secret, ok := frameConnectionURI(value, matchStart, matchEnd, secret)
	return contextValidation{accepted: ok && validPostgresConnectionURI(secret)}
}

// Keep candidate framing independent of each protocol's component semantics.
func frameConnectionURI(value string, matchStart, matchEnd int, secret string) (string, bool) {
	if matchStart > 0 {
		previous := value[matchStart-1]
		// A word boundary alone would accept the suffix of another URI scheme.
		if strings.ContainsRune(".+-/", rune(previous)) {
			return "", false
		}
		// Apostrophes are legal URI characters, unlike double quotes. Preserve
		// them inside components, but exclude a matching outer quote.
		if previous == '\'' && strings.HasSuffix(secret, "'") {
			secret = secret[:len(secret)-1]
		}
	}
	if matchEnd < len(value) {
		// A non-URI delimiter may close a wrapper, but must not make an
		// embedded invalid byte disappear from an otherwise malformed URI.
		var opening byte
		switch value[matchEnd] {
		case '"', '`':
			opening = value[matchEnd]
		case '>':
			opening = '<'
		case '<':
			return "", false
		}
		if opening != 0 && (matchStart == 0 || value[matchStart-1] != opening) {
			return "", false
		}
	}
	return secret, true
}

func validatePasswordConnectionURI(value string, matchStart, matchEnd int, secret string) contextValidation {
	// Do not recognize the mysql suffix inside a JDBC or another nested scheme.
	// PostgreSQL intentionally retains its existing framing behavior.
	if matchStart > 0 && value[matchStart-1] == ':' {
		return contextValidation{}
	}
	secret, ok := frameConnectionURI(value, matchStart, matchEnd, secret)
	if !ok || len(secret) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	scheme, rest, _ := strings.Cut(secret, "://")
	switch scheme {
	case mysqlScheme, "mysqlx", "mysqlx+srv":
		ok = validMySQLPasswordURI(rest, scheme == "mysqlx+srv")
	case "mongodb", "mongodb+srv":
		ok = validMongoDBPasswordURI(rest, scheme == "mongodb+srv")
	default:
		switch {
		case strings.EqualFold(scheme, "redis"), strings.EqualFold(scheme, "rediss"):
			ok = validRedisPasswordURI(rest)
		case strings.EqualFold(scheme, "amqp"), strings.EqualFold(scheme, "amqps"):
			ok = validAMQPPasswordURI(rest)
		default:
			ok = false
		}
	}
	return contextValidation{accepted: ok}
}

func validMySQLPasswordURI(rest string, srv bool) bool {
	hierarchy, query, _ := strings.Cut(rest, "?")
	hostPath, hasPassword, ok := connectionURIUserinfo(hierarchy, false)
	if !ok || !hasPassword {
		return false
	}
	host, database, _ := strings.Cut(hostPath, "/")
	switch {
	case strings.HasPrefix(hostPath, "("):
		// MySQL explicitly permits an unescaped Unix socket inside parentheses.
		// This is not Connector/J's parenthesized host-property grammar.
		end := strings.IndexByte(hostPath, ')')
		if srv || end < 2 {
			return false
		}
		socket := hostPath[1:end]
		if (!strings.HasPrefix(socket, "/") && !strings.HasPrefix(socket, ".")) ||
			strings.ContainsAny(socket, "()") || !connectionURIComponent(socket, ":@/") {
			return false
		}
		suffix := hostPath[end+1:]
		if suffix != "" && !strings.HasPrefix(suffix, "/") {
			return false
		}
		database = strings.TrimPrefix(suffix, "/")
	case srv:
		if !connectionURISRVHost(host) {
			return false
		}
	case !connectionURIHost(host):
		return false
	}
	// MySQL query values also permit parenthesized paths and bracketed arrays.
	return connectionURIComponent(database, ":@") && connectionURIComponent(query, ":@/?[]")
}

func validMongoDBPasswordURI(rest string, srv bool) bool {
	hierarchy, query, querySet := strings.Cut(rest, "?")
	hostPath, hasPassword, ok := connectionURIUserinfo(hierarchy, false)
	if !ok || !hasPassword {
		return false
	}
	hosts, database, hasPath := strings.Cut(hostPath, "/")
	if querySet && !hasPath {
		return false
	}
	if srv {
		if !connectionURISRVHost(hosts) {
			return false
		}
	} else {
		for host := range strings.SplitSeq(hosts, ",") {
			if host == "" || !connectionURIHost(host) {
				return false
			}
		}
	}
	return connectionURIComponent(database, ":@") && connectionURIComponent(query, ":@/?")
}

func validRedisPasswordURI(rest string) bool {
	hierarchy, query, _ := strings.Cut(rest, "?")
	hostPath, hasPassword, ok := connectionURIUserinfo(hierarchy, true)
	if !ok {
		return false
	}
	host, database, _ := strings.Cut(hostPath, "/")
	if !connectionURIHost(host) || !connectionURIComponent(database, "") ||
		!connectionURIComponent(query, ":@/?") {
		return false
	}
	// Decode only this framed path; encoded decimal database numbers are used
	// by clients too. No database range or server configuration is assumed.
	database, _ = url.PathUnescape(database)
	for i := range len(database) {
		if database[i] < '0' || database[i] > '9' {
			return false
		}
	}
	for parameter := range strings.SplitSeq(query, "&") {
		name, password, set := strings.Cut(parameter, "=")
		// The IANA registration names this exact option. Neither substrings
		// inside other values nor unknown options establish password evidence.
		name, _ = url.PathUnescape(name)
		if set && name == passwordField && password != "" {
			hasPassword = true
		}
	}
	return hasPassword
}

func validAMQPPasswordURI(rest string) bool {
	hierarchy, query, _ := strings.Cut(rest, "?")
	hostPath, hasPassword, ok := connectionURIUserinfo(hierarchy, false)
	if !ok || !hasPassword {
		return false
	}
	host, vhost, _ := strings.Cut(hostPath, "/")
	return connectionURIHost(host) && connectionURIComponent(vhost, ":@") &&
		connectionURIComponent(query, ":@/?")
}

// Components are framed before escapes are inspected. Redis additionally has
// redis-cli's password-only userinfo and RFC 3986 colons within the password.
func connectionURIUserinfo(hierarchy string, redis bool) (hostPath string, hasPassword, ok bool) {
	userinfo, hostPath, present := strings.Cut(hierarchy, "@")
	if !present {
		return hierarchy, false, true
	}
	user, password, set := strings.Cut(userinfo, ":")
	extra := ""
	if redis {
		extra = ":"
		if !set {
			password, user = user, ""
		}
	}
	return hostPath, password != "", connectionURIComponent(user, "") && connectionURIComponent(password, extra)
}

// Host checks preserve delimiters and escapes, not DNS or connection success.
// In particular, a decimal port outside 1-65535 does not hide a literal password.
func connectionURIHost(hostPort string) bool {
	var host, port string
	if strings.HasPrefix(hostPort, "[") {
		end := strings.IndexByte(hostPort, ']')
		if end <= 1 || !connectionURIComponent(hostPort[1:end], ":") {
			return false
		}
		suffix := hostPort[end+1:]
		if suffix != "" {
			var ok bool
			port, ok = strings.CutPrefix(suffix, ":")
			if !ok {
				return false
			}
		}
	} else {
		host, port, _ = strings.Cut(hostPort, ":")
		if !connectionURIComponent(host, "") {
			return false
		}
	}
	for i := range len(port) {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	return true
}

func connectionURISRVHost(host string) bool {
	if host == "" || !connectionURIComponent(host, "") {
		return false
	}
	// SRV uses a hostname, not the encoded socket paths accepted by the
	// ordinary schemes. This is component-local, nonrecursive decoding.
	host, _ = url.PathUnescape(host)
	return !strings.ContainsAny(host, ":,[]/@\\")
}

// Validate escapes without allocating decoded copies: any valid escape still
// represents one credential byte. Encoded NUL is not treated as an empty value.
// This intentionally does not change libpq's separate NUL-rejection semantics.
func connectionURIComponent(raw, extra string) bool {
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '%' {
			if i+2 >= len(raw) {
				return false
			}
			for j := i + 1; j <= i+2; j++ {
				digit := raw[j]
				if (digit < '0' || digit > '9') && (digit < 'a' || digit > 'f') && (digit < 'A' || digit > 'F') {
					return false
				}
			}
			i += 2
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.ContainsRune("-._~!$&'()*+,;=", rune(c)) || strings.ContainsRune(extra, rune(c)) {
			continue
		}
		return false
	}
	return true
}

// validPostgresConnectionURI follows libpq's component framing instead of
// net/url.Parse: Go's authority parser rejects libpq multihost and escaped
// socket paths, while ParseQuery changes literal '+' and accepts extra '='.
func validPostgresConnectionURI(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	rest, ok := strings.CutPrefix(s, "postgresql://")
	if !ok {
		rest, ok = strings.CutPrefix(s, "postgres://")
	}
	if !ok {
		return false
	}
	hasPassword := false
	// libpq looks for the first '@' before '/', including before a query
	// without a slash. A raw '@' in that query must therefore be escaped.
	if at := strings.IndexAny(rest, "@/"); at >= 0 && rest[at] == '@' {
		userinfo := rest[:at]
		rest = rest[at+1:]
		user, password, passwordSet := strings.Cut(userinfo, ":")
		if _, ok := postgresURIComponent(user, ""); !ok {
			return false
		}
		if passwordSet {
			decoded, ok := postgresURIComponent(password, ":")
			if !ok {
				return false
			}
			hasPassword = decoded != ""
		}
	}
	hierarchy, query, _ := strings.Cut(rest, "?")
	authority, database, _ := strings.Cut(hierarchy, "/")
	if _, ok := postgresURIComponent(database, ":@/"); !ok {
		return false
	}
	hostCount, portCount, ok := postgresAuthority(authority)
	if !ok {
		return false
	}
	hostaddrCount := 0
	for query != "" {
		parameter, remaining, _ := strings.Cut(query, "&")
		query = remaining
		name, rawValue, found := strings.Cut(parameter, "=")
		// libpq requires exactly one unescaped '=' per query parameter.
		if !found || strings.ContainsRune(rawValue, '=') {
			return false
		}
		name, ok = postgresURIComponent(name, "")
		if !ok || name == "" {
			return false
		}
		decoded, ok := postgresURIComponent(rawValue, ":@/?")
		if !ok {
			return false
		}
		switch name {
		case passwordField:
			hasPassword = hasPassword || decoded != ""
		case hostField:
			hostCount = 0
			if decoded != "" {
				hostCount = strings.Count(decoded, ",") + 1
			}
		case "hostaddr":
			hostaddrCount = 0
			if decoded != "" {
				for address := range strings.SplitSeq(decoded, ",") {
					if address != "" {
						if _, err := netip.ParseAddr(address); err != nil {
							return false
						}
					}
					hostaddrCount++
				}
			}
		case "port":
			portCount = 0
			for port := range strings.SplitSeq(decoded, ",") {
				if !validPostgresPort(port) {
					return false
				}
				portCount++
			}
		}
	}
	// Empty host lists select the local default. One port applies to all
	// hosts; otherwise libpq requires the specified lists to line up.
	if hostCount != 0 && hostaddrCount != 0 && hostCount != hostaddrCount {
		return false
	}
	hosts := max(1, hostCount, hostaddrCount)
	return hasPassword && (portCount == 1 || portCount == hosts)
}

func postgresAuthority(authority string) (hostCount, portCount int, ok bool) {
	hostSet := false
	for part := range strings.SplitSeq(authority, ",") {
		var host, port string
		if strings.HasPrefix(part, "[") {
			end := strings.IndexByte(part, ']')
			if end < 0 {
				return 0, 0, false
			}
			host, ok = postgresURIComponent(part[1:end], ":")
			if !ok {
				return 0, 0, false
			}
			address, err := netip.ParseAddr(host)
			if err != nil || !address.Is6() {
				return 0, 0, false
			}
			if suffix := part[end+1:]; suffix != "" {
				port, ok = strings.CutPrefix(suffix, ":")
				if !ok {
					return 0, 0, false
				}
			}
		} else {
			host, port, _ = strings.Cut(part, ":")
			host, ok = postgresURIComponent(host, "")
			if !ok {
				return 0, 0, false
			}
		}
		port, ok = postgresURIComponent(port, "")
		if !ok {
			return 0, 0, false
		}
		for item := range strings.SplitSeq(port, ",") {
			if !validPostgresPort(item) {
				return 0, 0, false
			}
			portCount++
		}
		hostSet = hostSet || host != ""
		hostCount += strings.Count(host, ",") + 1
	}
	if !hostSet && hostCount == 1 {
		hostCount = 0
	}
	return hostCount, portCount, true
}

func validPostgresPort(port string) bool {
	if port == "" {
		return true
	}
	n := 0
	for i := range len(port) {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
		n = n*10 + int(port[i]-'0')
		if n > 65535 {
			return false
		}
	}
	return n > 0
}

// Decode only already-framed URI components. PathUnescape, unlike
// QueryUnescape, preserves '+', and does not recursively decode escapes.
func postgresURIComponent(raw, extra string) (string, bool) {
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '%' {
			// PathUnescape below validates both hex digits.
			if i+2 >= len(raw) {
				return "", false
			}
			i += 2
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.ContainsRune("-._~!$&'()*+,;=", rune(c)) || strings.ContainsRune(extra, rune(c)) {
			continue
		}
		return "", false
	}
	decoded, err := url.PathUnescape(raw)
	return decoded, err == nil && !strings.ContainsRune(decoded, '\x00')
}

var credentialBase64URL = base64.RawURLEncoding.Strict()

func decodeCredentialBase64URL(s string) ([]byte, bool) {
	if len(s) == 0 || len(s) > maxStructuredCredentialBytes {
		return nil, false
	}
	// encoding/base64 deliberately ignores CR/LF even in Strict mode; JWT
	// components must not contain them or padding characters.
	for i := range len(s) {
		c := s[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' && c != '_' {
			return nil, false
		}
	}
	decoded, err := credentialBase64URL.DecodeString(s)
	return decoded, err == nil
}

func credentialJSONObject(b []byte) bool {
	b = bytes.TrimSpace(b)
	return len(b) >= 2 && b[0] == '{' && b[len(b)-1] == '}' && utf8.Valid(b) && json.Valid(b)
}

// validJWT recognizes signed JWT structure, not authenticity or authorization.
// The caller decides whether the carrier makes a JWT a confidential credential.
func validJWT(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	header, remainder, ok := strings.Cut(s, ".")
	if !ok {
		return false
	}
	payload, signature, ok := strings.Cut(remainder, ".")
	if !ok {
		return false
	}
	h, ok := decodeCredentialBase64URL(header)
	if !ok || !credentialJSONObject(h) {
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(h, &fields) != nil {
		return false
	}
	var alg string
	if json.Unmarshal(fields["alg"], &alg) != nil {
		return false
	}
	if encoded, present := fields["b64"]; present && !bytes.Equal(bytes.TrimSpace(encoded), []byte("true")) {
		return false // RFC 7797 unencoded payloads are outside this recognizer.
	}
	p, ok := decodeCredentialBase64URL(payload)
	if !ok || !credentialJSONObject(p) {
		return false
	}
	sig, ok := decodeCredentialBase64URL(signature)
	if !ok {
		return false
	}
	// RFC 7518 sections 3.2-3.5, RFC 8032, and RFC 9864. Length is a
	// structural check only; these bytes are never cryptographically verified.
	switch alg {
	case "HS256":
		return len(sig) == 32
	case "HS384":
		return len(sig) == 48
	case "HS512", "ES256", "ES256K", "Ed25519":
		return len(sig) == 64
	case "ES384":
		return len(sig) == 96
	case "ES512":
		return len(sig) == 132
	case "Ed448":
		return len(sig) == 114
	case "EdDSA":
		return len(sig) == 64 || len(sig) == 114
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		return len(sig) >= 256 && len(sig) <= 1024
	default:
		return false
	}
}

func validPrivateKey(s string) bool {
	const begin = "-----BEGIN "
	if len(s) > maxStructuredCredentialBytes || !strings.HasPrefix(s, begin) || strings.Contains(s[len(begin):], begin) {
		return false
	}
	encoded := []byte(s)
	// RFC 7468 allows CR, CRLF and LF line endings. Go's PEM decoder expects
	// LF/CRLF, so normalize CR within the already-owned candidate buffer.
	if bytes.IndexByte(encoded, '\r') >= 0 {
		written := 0
		for read := 0; read < len(encoded); read++ {
			c := encoded[read]
			if c == '\r' {
				c = '\n'
				if read+1 < len(encoded) && encoded[read+1] == '\n' {
					read++
				}
			}
			encoded[written] = c
			written++
		}
		encoded = encoded[:written]
	}
	block, rest := pem.Decode(encoded)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 || len(block.Headers) != 0 || !strings.HasPrefix(s, begin+block.Type+"-----") {
		return false
	}
	if block.Type == "OPENSSH PRIVATE KEY" {
		key, err := ssh.ParseRawPrivateKey(encoded)
		if err != nil {
			return false // PassphraseMissingError is not private-key validation.
		}
		// The SSH parser checks framing, checkints and padding, but its Ed25519
		// branch accepts any 64-byte private value. Check seed/public consistency.
		if ed, ok := key.(*ed25519.PrivateKey); ok {
			return len(*ed) == ed25519.PrivateKeySize && bytes.Equal(*ed, ed25519.NewKeyFromSeed((*ed)[:ed25519.SeedSize]))
		}
		return true
	}
	// Some x509 parsers tolerate trailing DER. Require one complete sequence
	// before handing off algorithm-specific private-key validation.
	var outer asn1.RawValue
	tail, err := asn1.Unmarshal(block.Bytes, &outer)
	if err != nil || len(tail) != 0 || outer.Class != asn1.ClassUniversal || outer.Tag != asn1.TagSequence || !outer.IsCompound {
		return false
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		_, err = x509.ParseECPrivateKey(block.Bytes)
	case "DSA PRIVATE KEY":
		key, parseErr := ssh.ParseDSAPrivateKey(block.Bytes)
		if parseErr != nil {
			return false
		}
		if key.P.Sign() <= 0 || key.P.BitLen() > 3072 || key.Q.Sign() <= 0 || key.Q.BitLen() > 256 || key.G.Sign() <= 0 || key.G.Cmp(key.P) >= 0 || key.Y.Sign() <= 0 || key.Y.Cmp(key.P) >= 0 || key.X.Sign() <= 0 || key.X.Cmp(key.Q) >= 0 {
			return false
		}
		return new(big.Int).Exp(key.G, key.X, key.P).Cmp(key.Y) == 0
	default:
		return false
	}
	return err == nil
}

func validOnePasswordServiceToken(s string) bool {
	if len(s) > maxStructuredCredentialBytes || !strings.HasPrefix(s, "ops_") {
		return false
	}
	encoded := s[4:]
	encoding := base64.RawStdEncoding
	if strings.ContainsAny(encoded, "-_") {
		encoding = base64.RawURLEncoding
	}
	if strings.HasSuffix(encoded, "=") {
		if encoding == base64.RawURLEncoding {
			encoding = base64.URLEncoding
		} else {
			encoding = base64.StdEncoding
		}
	}
	b, err := encoding.DecodeString(encoded)
	return err == nil && credentialJSONObject(b)
}

// The vendor encodes both X25519 scalars and MLKEM768-X25519 seeds as 32
// unrestricted bytes. Checking 52 Bech32 symbols, their four zero padding bits,
// and the six-symbol checksum establishes that structure without deriving keys.
// Sources: age/internal/bech32, age/x25519.go, age/pq.go, and
// https://github.com/FiloSottile/hpke/blob/main/pq.go (hybridKEM.NewPrivateKey).
func validAgeSecretKey(s string) bool {
	const alphabet = "QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L"
	hrp := "age-secret-key-"
	data, ok := strings.CutPrefix(s, "AGE-SECRET-KEY-1")
	if !ok {
		hrp = "age-secret-key-pq-"
		data, ok = strings.CutPrefix(s, "AGE-SECRET-KEY-PQ-1")
	}
	if !ok || len(data) != 58 {
		return false
	}
	checksum := uint32(1)
	for i := range len(hrp) {
		checksum = ageChecksumStep(checksum, uint32(hrp[i]>>5))
	}
	checksum = ageChecksumStep(checksum, 0)
	for i := range len(hrp) {
		checksum = ageChecksumStep(checksum, uint32(hrp[i]&31))
	}
	for i := range len(data) {
		symbol := strings.IndexByte(alphabet, data[i])
		if symbol < 0 || i == 51 && symbol&15 != 0 {
			return false
		}
		checksum = ageChecksumStep(checksum, uint32(symbol))
	}
	return checksum == 1
}

// BIP-173's Bech32 generator coefficients are format constants, not a hash or
// authenticity check. Updating the residue in place avoids decoding allocations.
func ageChecksumStep(checksum, symbol uint32) uint32 {
	generators := [...]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	top := checksum >> 25
	checksum = (checksum&0x1ffffff)<<5 ^ symbol
	for bit, generator := range generators {
		if top&(1<<bit) != 0 {
			checksum ^= generator
		}
	}
	return checksum
}
