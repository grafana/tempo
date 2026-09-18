package secrets

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
)

// Candidate bounds are adapted from Betterleaks commit 95237cf8eb4d (MIT;
// LICENSE.betterleaks). Provider sources establish the confidential roles.
// These offline recognizers neither execute upstream verifier expressions nor
// infer secrecy from public companion identifiers.
const betterCompositeEBayID3 = `(?:"(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])"|'(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])'|(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]))[ \t]{0,16}[:=][ \t]{0,16}(?:"[A-Za-z0-9_-]{1,64}-[A-Za-z0-9_-]{1,64}-PRD-[a-f0-9]{8,12}-[a-f0-9]{8,12}"|'[A-Za-z0-9_-]{1,64}-[A-Za-z0-9_-]{1,64}-PRD-[a-f0-9]{8,12}-[a-f0-9]{8,12}'|[A-Za-z0-9_-]{1,64}-[A-Za-z0-9_-]{1,64}-PRD-[a-f0-9]{8,12}-[a-f0-9]{8,12})`

// #nosec G101 -- Regular expression matching eBay secret assignments, not a credential value.
const betterCompositeEBaySecret3 = `(?:"(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])"|'(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])'|(?:[Ee][Bb][Aa][Yy]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]))[ \t]{0,16}[:=][ \t]{0,16}(?:"PRD-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4,12}"|'PRD-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4,12}'|PRD-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4,12})`

const betterCompositeExoscaleID3 = `(?:"(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|'(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy])'|(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]{0,16}[:=][ \t]{0,16}(?:"EXO[A-Za-z0-9]{24,30}"|'EXO[A-Za-z0-9]{24,30}'|EXO[A-Za-z0-9]{24,30})`

// #nosec G101 -- Regular expression matching Exoscale secret assignments, not a credential value.
const betterCompositeExoscaleSecret3 = `(?:"(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])"|'(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])'|(?:[Ee][Xx][Oo][Ss][Cc][Aa][Ll][Ee]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]))[ \t]{0,16}[:=][ \t]{0,16}(?:"[A-Za-z0-9_-]{40,60}"|'[A-Za-z0-9_-]{40,60}'|[A-Za-z0-9_-]{40,60})`

// Select one bounded field literal, leaving later permanent-secret candidates
// available if this is not STS. Validation consumes the entire value, never an
// assignment-shaped JSON prefix.
const betterCompositeAlibabaCandidate3 = `\b((?i:accesskeyid|accesskeysecret|securitytoken|(?:alibaba|aliyun)_sts_(?:access_key_id|access_key_secret|security_token))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?[A-Za-z0-9+/_=.-]{1,1000}[A-Za-z0-9+/_=.-]{0,28}["']?)`

const betterCompositeJoin3 = `[ \t\r\n,;]{1,64}`

var betterleaksCarriersRules3 = []catalogRuleSpec{
	{
		ID:              "ebay-client-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(` + betterCompositeEBayID3 + betterCompositeJoin3 + betterCompositeEBaySecret3 + `|` + betterCompositeEBaySecret3 + betterCompositeJoin3 + betterCompositeEBayID3 + `)[ \t]*(?:$|[\r\n,;}\]])`,
		Keywords:        []string{"EBAY_CLIENT_ID", "EBAY_CLIENT_SECRET"},
		Source:          "https://github.com/eBay/ebay-oauth-nodejs-client",
		Description:     "Complete named eBay production client-ID/client-secret pair in either order. The PRD secret is confidential; public client IDs alone are excluded. UUID-like component widths and the 64-byte assignment gap are supported scanner bounds, not issuance guarantees.",
		ValidateContext: validateBetterCompositePair3,
	},
	{
		ID:              "exoscale-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(` + betterCompositeExoscaleID3 + betterCompositeJoin3 + betterCompositeExoscaleSecret3 + `|` + betterCompositeExoscaleSecret3 + betterCompositeJoin3 + betterCompositeExoscaleID3 + `)[ \t]*(?:$|[\r\n,;}\]])`,
		Keywords:        []string{"EXOSCALE_API_KEY", "EXOSCALE_API_SECRET"},
		Source:          "https://raw.githubusercontent.com/exoscale/terraform-provider-exoscale/master/docs/index.md",
		Description:     "Complete documented EXOSCALE_API_KEY/EXOSCALE_API_SECRET literal pair in either order. EXO identifiers alone and request HMAC signatures are excluded. The 24-30-character ID suffix, 40-60-character secret and 64-byte/five-line pairing bounds are scanner constraints.",
		ValidateContext: validateBetterCompositePair3,
	},
	{
		ID:              "etsy-open-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Ee][Tt][Ss][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|'(?:[Ee][Tt][Ss][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy])'|(?:[Ee][Tt][Ss][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{24}:[A-Za-z0-9]{10,64})"|'([A-Za-z0-9]{24}:[A-Za-z0-9]{10,64})'|([A-Za-z0-9]{24}:[A-Za-z0-9]{10,64}))` + catalogRightBoundary + `|` + auditedProviderRequestPattern2,
		Keywords:        []string{"ETSY_API_KEY", curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions},
		Source:          "https://developers.etsy.com/documentation/essentials/authentication/",
		Description:     "Etsy keystring:shared-secret credential in an exact ETSY_API_KEY assignment or a complete api.etsy.com v3 request with x-api-key. The keystring alone is also the public OAuth client_id and is excluded. Widths are the pinned scanner subset, not a complete issuer grammar.",
		ValidateContext: validateBetterCompositeEtsy3,
	},
	{
		ID:              "fal-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Ff][Aa][Ll]_[Kk][Ee][Yy])"|'(?:[Ff][Aa][Ll]_[Kk][Ee][Yy])'|(?:[Ff][Aa][Ll]_[Kk][Ee][Yy]))[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}:[a-f0-9]{32})"|'([a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}:[a-f0-9]{32})'|([a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}:[a-f0-9]{32}))` + catalogRightBoundary + `|` + auditedProviderRequestPattern2,
		Keywords:        []string{"FAL_KEY", curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions},
		Source:          "https://fal.ai/docs/documentation/setting-up/authentication",
		Description:     "Complete fal key-ID:secret in the documented FAL_KEY assignment or Authorization: Key header on a complete fal.run/api.fal.ai request. UUID layout and the 32-hex secret are the supported scanner subset; arbitrary UUIDs and generic Key headers are excluded.",
		ValidateContext: validateBetterCompositeFal3,
	},
	{
		ID:              "polymarket-api-credentials",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?:[Cc][Ll][Oo][Bb]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt])"|'(?:[Cc][Ll][Oo][Bb]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt])'|(?:[Cc][Ll][Oo][Bb]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]|[Pp][Oo][Ll][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt]))[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9+/_-]{40,254}={0,2})"|'([A-Za-z0-9+/_-]{40,254}={0,2})'|([A-Za-z0-9+/_-]{40,254}={0,2}))|(?:^|[^A-Za-z0-9_.-])(?:"(?:[Cc][Ll][Oo][Bb]_[Pp][Aa][Ss][Ss]_[Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Bb][Uu][Ii][Ll][Dd][Ee][Rr]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee])"|'(?:[Cc][Ll][Oo][Bb]_[Pp][Aa][Ss][Ss]_[Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Bb][Uu][Ii][Ll][Dd][Ee][Rr]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee])'|(?:[Cc][Ll][Oo][Bb]_[Pp][Aa][Ss][Ss]_[Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy]_[Bb][Uu][Ii][Ll][Dd][Ee][Rr]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]|[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Aa][Pp][Ii]_[Pp][Aa][Ss][Ss][Pp][Hh][Rr][Aa][Ss][Ee]))[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9_]{8,128})"|'([A-Za-z0-9_]{8,128})'|([A-Za-z0-9_]{8,128})))` + catalogRightBoundary,
		Keywords:        []string{"CLOB_SECRET", "POLYMARKET_API_SECRET", "POLY_API_SECRET", "POLY_SECRET", "CLOB_PASS_PHRASE", "POLY_PASSPHRASE", "POLY_BUILDER_PASSPHRASE", "POLYMARKET_PASSPHRASE", "POLYMARKET_API_PASSPHRASE"},
		Source:          "https://docs.polymarket.com/getting-started/api#authentication",
		Description:     "Polymarket L2/Builder HMAC secret or private passphrase in an exact named carrier, including official CLOB SDK environment variables and POLY authentication headers. Complete key/secret/passphrase configurations are covered by their confidential members, never by the UUID key identifier. Secret Base64 is canonical; 40-256 encoded bytes and 8-128 passphrase characters are scanner bounds.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCompositePolymarket3,
	},
	{
		ID:              "polymarket-private-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]|[Pp][Oo][Ll][Yy]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy])"|'(?:[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]|[Pp][Oo][Ll][Yy]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy])'|(?:[Pp][Oo][Ll][Yy][Mm][Aa][Rr][Kk][Ee][Tt]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]|[Pp][Oo][Ll][Yy]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]))[ \t]{0,16}[:=][ \t]{0,16}(?:"(0x[a-fA-F0-9]{64})"|'(0x[a-fA-F0-9]{64})'|(0x[a-fA-F0-9]{64}))` + catalogRightBoundary,
		Keywords:        []string{"POLYMARKET_PRIVATE_KEY", "POLY_PRIVATE_KEY"},
		Source:          "https://docs.polymarket.com/getting-started/api#authentication",
		Description:     "Explicit Polymarket wallet-private-key assignment, not an unlabelled Ethereum hash or public wallet address. The literal is a 32-byte secp256k1 scalar in the range 1 <= key < curve order; wallet ownership and signatures are not verified.",
		Validate:        validBetterCompositePrivateKey3,
		ValidateContext: validateAuditedAssignmentContext,
	},
}

// All fields are complete literals and the intervening syntax is restricted to
// assignment separators: an expression cannot bridge the two credentials.
func validateBetterCompositePair3(_ string, _, _ int, secret string) contextValidation {
	return contextValidation{accepted: strings.Count(secret, "\n") <= 5}
}

var (
	betterCompositeEtsyBody3    = newLazyRegexp(`^[A-Za-z0-9]{24}:[A-Za-z0-9]{10,64}$`)
	betterCompositeEtsyRequest3 = auditedProviderRequestValidator2(`api\.etsy\.com`, `/v3/[^?#]+`, []auditedProviderHeader2{{name: apiKeyHeaderLower, pattern: betterCompositeEtsyBody3}}, nil, nil, false)
)

func validateBetterCompositeEtsy3(value string, start, end int, secret string) contextValidation {
	if betterCompositeEtsyBody3.MatchString(secret) {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	return betterCompositeEtsyRequest3(value, start, end, secret)
}

var (
	betterCompositeFalBody3    = newLazyRegexp(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}:[a-f0-9]{32}$`)
	betterCompositeFalRequest3 = auditedProviderRequestValidator2(`(?:api\.fal\.ai|fal\.run)`, `/[^?#]+`, []auditedProviderHeader2{{name: authorizationHeader, prefix: "Key ", pattern: betterCompositeFalBody3}}, nil, nil, false)
)

func validateBetterCompositeFal3(value string, start, end int, secret string) contextValidation {
	if betterCompositeFalBody3.MatchString(secret) {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	return betterCompositeFalRequest3(value, start, end, secret)
}

func validateBetterCompositePolymarket3(value string, start, end int, secret string) contextValidation {
	if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	prefixEnd := start + strings.LastIndex(value[start:end], secret)
	if indexFoldedASCII(value[start:prefixEnd], secretField) < 0 {
		return contextValidation{accepted: true}
	}
	// Official SDKs decode the signing key before HMAC; support canonical
	// standard and URL-safe Base64, not an arbitrary encoded-looking word.
	if len(secret) > 256 {
		return contextValidation{}
	}
	var decoded [192]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(secret))
	if err != nil {
		n, err = base64.URLEncoding.Strict().Decode(decoded[:], []byte(secret))
	}
	return contextValidation{accepted: err == nil && n > 0}
}

func validBetterCompositePrivateKey3(secret string) bool {
	if len(secret) != 66 || !strings.HasPrefix(secret, "0x") {
		return false
	}
	var key [32]byte
	if _, err := hex.Decode(key[:], []byte(secret[2:])); err != nil {
		return false
	}
	// SEC 2, section 2.4.1: secp256k1 subgroup order, not the field prime.
	order := [32]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe, 0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b, 0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41}
	nonzero := false
	comparison := 0
	for i, b := range key {
		nonzero = nonzero || b != 0
		if comparison == 0 {
			comparison = int(b) - int(order[i])
		}
	}
	return nonzero && comparison < 0
}

var betterCompositeAlibabaFields3 = []auditedProviderField2{
	{name: "AccessKeyId", pattern: newLazyRegexp(`^STS\.[A-Za-z0-9]{16,64}$`)},
	{name: "AccessKeySecret", pattern: newLazyRegexp(`^[A-Za-z0-9]{30,64}$`)},
	{name: "SecurityToken", pattern: newLazyRegexp(`^CAIS[A-Za-z0-9+/_=-]{20,1000}[A-Za-z0-9+/_=-]{0,24}$`)},
}

func validateBetterCompositeAlibaba3(value string, start, end int, secret string) contextValidation {
	// Existing permanent-secret branches capture only their alphanumeric
	// secret. STS candidates instead capture a field name and its binder.
	if !strings.ContainsAny(secret, ":=") {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	if len(value) > 2048 || strings.Count(value, "\n") > 10 {
		return contextValidation{}
	}
	carrier := strings.Trim(value, " \t\r\n")
	if strings.HasPrefix(carrier, "{") {
		decoder := json.NewDecoder(strings.NewReader(carrier))
		if !validBetterCompositeAlibabaJSON3(decoder, true) {
			return contextValidation{}
		}
		var trailing json.RawMessage
		return contextValidation{accepted: decoder.Decode(&trailing) == io.EOF}
	}
	return contextValidation{accepted: validBetterCompositeAlibabaAssignments3(carrier)}
}

// Accept a complete flat credential object or the documented Credentials
// envelope. Unlike the form/flat-body helper, this must validate the enclosing
// object too: a nested valid triple cannot conceal a duplicate or truncated root.
func validBetterCompositeAlibabaJSON3(decoder *json.Decoder, envelope bool) bool {
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	seen := make(map[string]bool)
	var mask uint8
	var credentials bool
	for decoder.More() {
		key, err := decoder.Token()
		name, ok := key.(string)
		if err != nil || !ok || seen[name] || len(seen) == 64 {
			return false
		}
		seen[name] = true
		if name == "Credentials" {
			if !envelope || !validBetterCompositeAlibabaJSON3(decoder, false) {
				return false
			}
			credentials = true
			continue
		}
		field := -1
		for i := range betterCompositeAlibabaFields3 {
			if name == betterCompositeAlibabaFields3[i].name {
				field = i
				break
			}
		}
		if field < 0 {
			var ignored json.RawMessage
			if decoder.Decode(&ignored) != nil {
				return false
			}
			continue
		}
		var literal string
		if decoder.Decode(&literal) != nil || !betterCompositeAlibabaFields3[field].pattern.MatchString(literal) || !validAuditedCarrierLiteral2(literal) {
			return false
		}
		mask |= 1 << field
	}
	closing, err := decoder.Token()
	return err == nil && closing == json.Delim('}') && (mask == 7 && !credentials || mask == 0 && credentials)
}

func validBetterCompositeAlibabaAssignments3(carrier string) bool {
	// Braces always denote an object, not plain assignments, even when the
	// object is malformed. Double-quoted JSON keys also require JSON braces.
	if carrier == "" || strings.ContainsAny(carrier, "{}[]") {
		return false
	}
	names := [3][3]string{
		{"accesskeyid", "alibaba_sts_access_key_id", "aliyun_sts_access_key_id"},
		{"accesskeysecret", "alibaba_sts_access_key_secret", "aliyun_sts_access_key_secret"},
		{"securitytoken", "alibaba_sts_security_token", "aliyun_sts_security_token"},
	}
	var mask uint8
	for carrier != "" {
		binder := strings.IndexAny(carrier, ":=")
		if binder < 0 {
			return false
		}
		name := strings.TrimRight(carrier[:binder], " \t")
		if name == "" || binder-len(name) > 16 {
			return false
		}
		if name[0] == '"' || name[0] == '\'' {
			if len(name) < 2 || name[len(name)-1] != name[0] || name[0] == '"' && carrier[binder] == ':' {
				return false
			}
			name = name[1 : len(name)-1]
		}
		field := -1
		for i, aliases := range names {
			for _, alias := range aliases {
				if len(name) == len(alias) && indexFoldedASCII(name, alias) == 0 {
					field = i
					break
				}
			}
		}
		if field < 0 || mask&(1<<field) != 0 {
			return false
		}
		body := carrier[binder+1:]
		carrier = strings.TrimLeft(body, " \t")
		if carrier == "" || len(body)-len(carrier) > 16 {
			return false
		}
		var literal string
		if carrier[0] == '"' || carrier[0] == '\'' {
			closing := strings.IndexByte(carrier[1:], carrier[0])
			if closing < 0 {
				return false
			}
			literal = carrier[1 : closing+1]
			carrier = carrier[closing+2:]
		} else {
			closing := strings.IndexAny(carrier, " \t\r\n,;")
			if closing < 0 {
				closing = len(carrier)
			}
			literal = carrier[:closing]
			carrier = carrier[closing:]
		}
		if !betterCompositeAlibabaFields3[field].pattern.MatchString(literal) || !validAuditedCarrierLiteral2(literal) {
			return false
		}
		mask |= 1 << field
		body = carrier
		carrier = strings.TrimLeft(carrier, " \t\r\n,;")
		gap := len(body) - len(carrier)
		if gap > 64 || carrier != "" && gap == 0 {
			return false
		}
	}
	return mask == 7
}
