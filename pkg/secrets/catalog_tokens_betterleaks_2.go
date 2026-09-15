package secrets

import (
	"encoding/base64"
	"hash/crc32"
	"strings"
)

// Candidate constraints adapted from Betterleaks commit 95237cf8eb4d.
// See LICENSE.betterleaks; fixture evidence distinguishes provider contracts
// from bounded scanner subsets. All recognition is offline and value-only.
var betterleaksTokensRules2 = []catalogRuleSpec{
	{
		ID:          "onesignal-api-key",
		Regex:       `\b(os_v2_(?:app|org)_[a-z2-7]{103})` + catalogRightBoundary,
		Keywords:    []string{"os_v2_app_", "os_v2_org_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://documentation.onesignal.com/docs/en/keys-and-ids",
		Description: "OneSignal private App/Organization API-key candidates. Prefix aliases and the 103-character base32 body follow the pinned Betterleaks scanner; public IDs and legacy UUID-only values are outside this rule.",
	},
	{
		ID:          "onfido-api-token",
		Regex:       `\b(api_(?:live|sandbox)(?:_ca|_us)?\.[A-Za-z0-9_-]{20,80})` + catalogRightBoundary,
		Keywords:    []string{"api_live", "api_sandbox"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://documentation.identity.entrust.com/api/latest/",
		Description: "Onfido/Entrust confidential live and sandbox API-token candidates, combining EU, Canada and US spellings. Case-sensitive prefixes and complete 20-80-character URL-safe bodies are scanner bounds; SDK tokens and applicant IDs are excluded.",
	},
	{
		ID:          "paddle-api-key",
		Regex:       `\b(pdl_(?:live|sdbx)_apikey_[a-z0-9]{26}_[A-Za-z0-9]{22}_[A-Za-z0-9]{3})` + catalogRightBoundary,
		Keywords:    []string{"pdl_live_apikey_", "pdl_sdbx_apikey_"},
		SecretGroup: 1,
		Source:      "https://developer.paddle.com/api-reference/about/authentication",
		Description: "Paddle confidential live and sandbox API keys using the provider-published component grammar. Client-side tokens, malformed components and longer credentials are excluded.",
	},
	{
		ID:          "persona-api-key",
		Regex:       `\b(persona_(?:production|sandbox)_[a-z0-9_-]{20,80})` + catalogRightBoundary,
		Keywords:    []string{"persona_production_", "persona_sandbox_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://docs.withpersona.com/api-quickstart-tutorial",
		Description: "Persona confidential production/sandbox API-key candidates with documented environment prefixes and bounded scanner bodies; no client/public ID matching.",
	},
	{
		ID:          "pinterest-access-token",
		Regex:       `\b(pina_[A-Za-z0-9_-]{20,200})` + catalogRightBoundary,
		Keywords:    []string{"pina_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://developers.pinterest.com/docs/getting-started/set-up-authentication-and-authorization/",
		Description: "Pinterest confidential pina access-token candidates with complete bounded URL-safe secret bodies; refresh-token and app-ID roles are distinct.",
	},
	{
		ID:          "polar-access-token",
		Regex:       `\b(polar_(?:at|oat|pat)_[A-Za-z0-9_-]{20,100})` + catalogRightBoundary,
		Keywords:    []string{"polar_at_", "polar_oat_", "polar_pat_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://polar.sh/docs/integrate/oauth2/connect",
		Description: "Polar confidential OAuth, organization and personal access-token candidates combined under one family, preserving each case-sensitive role prefix.",
	},
	{
		ID:          "proof-full-access-api-key",
		Regex:       `\b(prf_(?:test_)?[A-Za-z0-9_-]{20,80})` + catalogRightBoundary,
		Keywords:    []string{"prf_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://dev.proof.com/docs/api-keys",
		Description: "Proof confidential full-access production and test API-key candidates. Explicitly excludes public client-only keys in both environments; scanner bounds do not prove issuance.",
		Validate:    validBetterleaksProofKey2,
	},
	{
		ID:          "render-api-key",
		Regex:       `\b(rnd_[A-Za-z0-9]{28})` + catalogRightBoundary,
		Keywords:    []string{"rnd_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://render.com/docs/cli",
		Description: "Render confidential rnd_ API-key candidate with the pinned scanner 28-alphanumeric body; no issuer validation.",
	},
	{
		ID:          "samsara-api-token",
		Regex:       `\b(samsara_api_[A-Za-z0-9]{26,32})` + catalogRightBoundary,
		Keywords:    []string{"samsara_api_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      "https://developers.samsara.com/docs/authentication",
		Description: "Samsara confidential prefixed API-token candidates with bounded 26-32-character scanner bodies; public organization identifiers are excluded.",
	},
	{
		ID:          "segment-public-api-token",
		Regex:       `\b(sgp_[A-Za-z0-9]{64})` + catalogRightBoundary,
		Keywords:    []string{"sgp_"},
		SecretGroup: 1,
		Entropy:     3.3,
		Source:      "https://www.twilio.com/docs/segment/api/public-api",
		Description: "Segment confidential Public API workspace-token candidates with complete 64-alphanumeric bodies; Public API describes the endpoint, not permission to expose the credential.",
	},
	{
		ID:          "settlemint-access-token",
		Regex:       `\b(sm_(?:aat|pat)_[A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:    []string{"sm_aat_", "sm_pat_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      "https://raw.githubusercontent.com/settlemint/sdk/main/sdk/mcp/README.md",
		Description: "SettleMint confidential personal/application access-token candidates combined under one family, with complete 16-alphanumeric scanner bodies.",
	},
	{
		ID:          "slack-legacy-token",
		Regex:       `\b(xox[os]-[0-9]{1,20}-[0-9]{1,20}-[0-9]{1,20}-[A-Fa-f0-9]{24,128})` + catalogRightBoundary,
		Keywords:    []string{"xoxo-", "xoxs-"},
		SecretGroup: 1,
		Entropy:     2,
		Source:      "https://docs.slack.dev/changelog/2016/05/19/authorship-changing-for-older-tokens",
		Description: "Slack confidential historical xoxo/xoxs user-token candidates with complete numeric components and a bounded nonempty secret tail; retirement does not erase historical confidentiality.",
	},
	{
		ID:          "slack-session-cookie",
		Regex:       `\b(xoxd-[A-Za-z0-9+/_-]{100,1000}={0,2})` + catalogRightBoundary,
		Keywords:    []string{"xoxd-"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://slack.engineering/proactive-measures-against-password-breaches-and-cookie-hijacking/",
		Description: "Slack confidential xoxd session-cookie candidates. Checks complete canonical Base64 representation after a selective prefix hit, with bounded storage; percent-encoded and mixed-alphabet copies are excluded.",
		Validate:    validBetterleaksSlackCookie2,
	},
	{
		ID:          "slack-session-token",
		Regex:       `\b(xoxc-[0-9]{9,15}-[0-9]{9,15}-[0-9]{9,15}-[a-f0-9]{64})` + catalogRightBoundary,
		Keywords:    []string{"xoxc-"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://slack.engineering/proactive-measures-against-password-breaches-and-cookie-hijacking/",
		Description: "Slack confidential xoxc user-session token component with all three IDs and the complete secret tail; session-cookie pairing or current validity is not asserted.",
	},
	{
		ID:          "unkey-root-key",
		Regex:       `\b(unkey_[A-Za-z0-9]{20,32})` + catalogRightBoundary,
		Keywords:    []string{"unkey_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://raw.githubusercontent.com/unkeyed/unkey/main/internal/services/keys/create.go",
		Description: "Unkey confidential administrative root-key candidates. The legacy generator Base58-encodes random bytes without a character-class quota. The unkey_ prefix, 20-32 alphanumeric body and entropy threshold are bounded scanner filters, not exhaustive issuer grammar; public API IDs are excluded.",
	},
	{
		ID:          "upcloud-api-token",
		Regex:       `\b(ucat_[A-Za-z0-9]{24,32})` + catalogRightBoundary,
		Keywords:    []string{"ucat_"},
		SecretGroup: 1,
		Source:      "https://developers.upcloud.com/1.3/24-api-tokens/",
		Description: "UpCloud confidential ucat_ Bearer-token candidates using complete bounded alphanumeric secret bodies.",
	},
	{
		ID:          "val-town-api-token",
		Regex:       `\b(vtwn_[A-Za-z0-9_-]{20,80})` + catalogRightBoundary,
		Keywords:    []string{"vtwn_"},
		SecretGroup: 1,
		Entropy:     3.5,
		Source:      "https://docs.val.town/api/authentication/",
		Description: "Val Town confidential prefixed API-token candidates. The vtwn_ prefix, 20-80 URL-safe body and entropy threshold are bounded scanner filters; the authentication documentation provides no digit-count guarantee or complete generator grammar. No live validity check.",
	},
	{
		ID:          "thunderstore-api-token",
		Regex:       `\b(tss_[A-Za-z0-9]{36})` + catalogRightBoundary,
		Keywords:    []string{"tss_"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/thunderstore-io/Thunderstore/84c5d445097c9e01fac6a839aba0f899f8d5ac20/django/thunderstore/account/tokens.py",
		Description: "Thunderstore confidential service-account tokens following the source-generated 30-character secret and six-character CRC32/base62 suffix. Malformed checksums and unsupported opaque bodies are rejected.",
		Validate:    validBetterleaksThunderstore2,
	},
	{
		ID:          "workato-developer-api-token",
		Regex:       `\b(wrka(?:eu|jp|sg|au|il|cn|kr)?-eyJ[A-Za-z0-9_-]{8,997}\.[A-Za-z0-9_-]{16,1000}\.[A-Za-z0-9_-]{64,1000})` + catalogRightBoundary,
		Keywords:    []string{"wrka"},
		SecretGroup: 1,
		Entropy:     4,
		Source:      "https://docs.workato.com/workato-api.html",
		Description: "Workato confidential region-prefixed Developer API-token candidates with a complete structurally valid JWT. Allowlisted region spellings and bounded segments prevent unbounded parsing or accepting arbitrary JWT-like text.",
		Validate:    validBetterleaksWorkato2,
	},
	{
		ID:          "xendit-secret-api-key",
		Regex:       `\b(xnd_(?:production|development)_[A-Za-z0-9]{56,72})` + catalogRightBoundary,
		Keywords:    []string{"xnd_production_", "xnd_development_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      "https://docs.xendit.co/docs/api-keys",
		Description: "Xendit confidential live/test secret API-key candidates, explicitly excluding publishable xnd_public keys. Complete case-sensitive environment prefixes and bounded secret bodies are required.",
	},
	{
		ID:              "zoho-zapi-key",
		Regex:           `\b(?i:zapikey)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?((?:1001\.[a-f0-9]{32}\.[a-f0-9]{32}(?:-d)?|1003\.[a-f0-9]{32,64}))` + catalogRightBoundary,
		Keywords:        []string{"zapikey"},
		SecretGroup:     1,
		Entropy:         3.5,
		Source:          "https://www.zoho.com/recruit/developer-console/widgets/API/get-zapi-key.html",
		Description:     "Zoho confidential ZAPI authentication key in an exact same-value zapikey assignment. Complete multi-component bodies and assignment guards reject public IDs, generic provider proximity and dynamic expressions.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
}

func validBetterleaksProofKey2(s string) bool {
	return !strings.HasPrefix(s, "prf_cli_")
}

func validBetterleaksSlackCookie2(s string) bool {
	if !strings.HasPrefix(s, "xoxd-") || len(s) > 1007 {
		return false
	}
	body := s[len("xoxd-"):]
	encoding := base64.StdEncoding.Strict()
	if strings.ContainsAny(body, "-_") {
		encoding = base64.URLEncoding.Strict()
	}
	if !strings.ContainsRune(body, '=') {
		encoding = encoding.WithPadding(base64.NoPadding)
	}
	var decoded [752]byte
	n, err := encoding.Decode(decoded[:], []byte(body))
	return err == nil && n >= 75 && n <= 750
}

func validBetterleaksThunderstore2(s string) bool {
	if len(s) != 40 || !strings.HasPrefix(s, "tss_") {
		return false
	}
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	checksum := crc32.ChecksumIEEE([]byte(s[4:34]))
	for i := 39; i >= 34; i-- {
		if s[i] != alphabet[checksum%62] {
			return false
		}
		checksum /= 62
	}
	return checksum == 0
}

func validBetterleaksWorkato2(s string) bool {
	_, token, ok := strings.Cut(s, "-")
	return ok && validJWT(token)
}
