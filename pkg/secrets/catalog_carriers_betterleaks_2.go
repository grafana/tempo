package secrets

import (
	"strings"
)

// Credential-role carriers audited against Betterleaks 95237cf8eb4d (MIT).
// See LICENSE.betterleaks. Provider sources establish confidentiality; scanner
// widths do not establish issuance, entropy or online credential validity.
var betterleaksCarriersRules2 = []catalogRuleSpec{
	{
		ID:              "lighton-paradigm-api-key",
		Regex:           `\b(?i:PARADIGM_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{40,80})` + catalogRightBoundary,
		Keywords:        []string{"paradigm_api_key"},
		SecretGroup:     1,
		Source:          "https://docs.lighton.ai/fr/developer-resources/api-fundamentals/proxy-configuration",
		Description:     "Documented PARADIGM_API_KEY assignment supplies the confidential LightOn Bearer credential. Opaque 40-80-character bounds are the Betterleaks scanner subset, not issuer grammar; loose LightOn/Paradigm proximity is excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "linear-client-secret",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		SecretGroup:     0,
		Source:          "https://linear.app/developers/oauth-2-0-authentication",
		Description:     "Linear OAuth client_secret in a complete parsed POST request to api.linear.app/oauth/token. The 32-hex body is a scanner subset; public client IDs and secrets merely near the word Linear are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderRequestValidator2(`api\.linear\.app`, `/oauth/token`, nil, []auditedProviderField2{{clientSecretField, newLazyRegexp(`^[a-fA-F0-9]{32}$`)}}, nil, false),
	},
	{
		ID:              "mailgun-webhook-signing-key",
		Regex:           `\b(?i:MAILGUN_(?:INGRESS_)?SIGNING_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-hA-H0-9]{32}-[a-hA-H0-9]{8}-[a-hA-H0-9]{8})` + catalogRightBoundary,
		Keywords:        []string{"mailgun"},
		SecretGroup:     1,
		Source:          "https://documentation.mailgun.com/docs/mailgun/user-manual/webhooks/securing-webhooks",
		Description:     "Mailgun webhook signing key in the documented mailgun_signing_key or MAILGUN_INGRESS_SIGNING_KEY literal role. This HMAC signing key is distinct from sending API credentials and transmitted webhook signatures; the 32-8-8 candidate shape is scanner-bounded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "miro-client-secret",
		Regex:           `\b(?i:MIRO_CLIENT_SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-zA-Z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"miro_client_secret"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/miroapp/api-clients/main/packages/miro-api/README.md",
		Description:     "Miro SDK MIRO_CLIENT_SECRET literal with a nearby literal MIRO_CLIENT_ID in the same scanned value. Public IDs are context only; the companion search is bounded to 512 bytes on each side and five intervening line breaks. Body bounds follow Betterleaks, not issuer guarantees.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierMiro2,
	},
	{
		ID:              "ovh-application-secret",
		Regex:           `\b(?i:OVH_APPLICATION_SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{32})` + catalogRightBoundary,
		Keywords:        []string{"ovh_application_secret"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/ovh/python-ovh/master/README.rst",
		Description:     "OVH application-secret in the SDK-documented OVH_APPLICATION_SECRET assignment. OVH explicitly classifies both application secret and consumer key as confidential; public application keys and transmitted signatures are not findings. The 32-byte candidate bound is scanner-supported.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "ovh-consumer-key",
		Regex:           `\b(?i:OVH_CONSUMER_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{32})` + catalogRightBoundary,
		Keywords:        []string{"ovh_consumer_key"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/ovh/python-ovh/master/README.rst",
		Description:     "OVH consumer-key in the SDK-documented OVH_CONSUMER_KEY assignment. OVH explicitly classifies both application secret and consumer key as confidential; public application keys and transmitted signatures are not findings. The 32-byte candidate bound is scanner-supported.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "rainforest-pay-api-key",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		SecretGroup:     0,
		Source:          "https://docs.rainforestpay.com/reference/authentication",
		Description:     "Rainforest Pay production and sandbox API credentials in complete host-bound Bearer requests. The shared apikey_/sbx_apikey_ prefixes alone do not establish provider attribution; public resource IDs and unrelated request destinations are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderRequestValidator2(`api\.(?:sandbox\.)?rainforestpay\.com`, `/v1/.*`, []auditedProviderHeader2{{authorizationHeader, bearerPrefix, newLazyRegexp(`^(?:sbx_)?apikey_[a-f0-9]{64}$`)}}, nil, nil, false),
	},
	{
		ID:              "squarespace-access-token",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		SecretGroup:     0,
		Source:          "https://developers.squarespace.com/commerce-apis/making-requests",
		Description:     "Squarespace UUID-shaped API key/OAuth access token in a complete api.squarespace.com Bearer request. Site/customer UUIDs and provider-proximate values are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderRequestValidator2(`api\.squarespace\.com`, `/[0-9]+\.[0-9]+/.*`, []auditedProviderHeader2{{authorizationHeader, bearerPrefix, newLazyRegexp(`^[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$`)}}, nil, nil, false),
	},
	{
		ID:              "upstage-api-key",
		Regex:           `\b(?i:UPSTAGE_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{40,50})` + catalogRightBoundary,
		Keywords:        []string{"upstage_api_key"},
		SecretGroup:     1,
		Source:          "https://console.upstage.ai/docs/guides/mcp-server.md",
		Description:     "Official Upstage MCP server UPSTAGE_API_KEY assignment. The opaque 40-50-alphanumeric candidate is scanner-bounded; public model IDs, body-only matches and brand proximity are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "woocommerce-consumer-secret",
		Regex:           `\bnew[ \t]+WooCommerce(?:RestApi|API)[ \t]*\([ \t\r\n]*\{[ \t\r\n]*(?:(?:url[ \t]*:[ \t]*(?:"https?://[^"\r\n{}\x60]{1,256}"|'https?://[^'\r\n{}\x60]{1,256}')|consumerKey[ \t]*:[ \t]*(?:"ck_[a-f0-9]{40}"|'ck_[a-f0-9]{40}'))[ \t\r\n]*,[ \t\r\n]*){0,2}consumerSecret[ \t]*:[ \t]*["'](cs_[a-f0-9]{40})["']`,
		Keywords:        []string{"woocommerce"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/woocommerce/woocommerce-rest-api-js-lib/master/README.md",
		Description:     "WooCommerce consumerSecret literal in the official WooCommerceRestApi/WooCommerceAPI SDK constructor, optionally following literal url/consumerKey fields. The issuer generates cs_ plus 20 random bytes rendered as 40 lowercase hex; public consumer keys and a standalone shared cs_ prefix are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "zai-api-key",
		Regex:           `\b(?i:ZAI_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{32}\.[A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:        []string{"zai_api_key"},
		SecretGroup:     1,
		Source:          "https://docs.z.ai/guides/develop/python/introduction",
		Description:     "Z.AI SDK-documented ZAI_API_KEY literal with the scanner-supported key-ID dot secret body. Generic GLM/zlm proximity, model names and bare dotted identifiers are not provider credentials.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "zoho-client-secret",
		Regex:           `\bhttps://accounts\.zoho\.com/oauth/v2/token\?[^\s"'<>\x60]+`,
		Keywords:        []string{"accounts.zoho.com"},
		SecretGroup:     0,
		Source:          "https://www.zoho.com/developer/oauth/web-server-apps/get-access-token.html",
		Description:     "Zoho OAuth client_secret in a complete accounts.zoho.com token URL paired with the 1000.-prefixed client_id. Public app IDs alone are not reported; duplicate parameters and nonliteral/host-spoofed carriers are rejected. Candidate widths are pinned scanner bounds.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2(`accounts\.zoho\.com`, `/oauth/v2/token`, []auditedProviderField2{{clientSecretField, newLazyRegexp(`^[a-fA-F0-9]{42}$`)}, {clientIDField, newLazyRegexp(`^1000\.[A-Za-z0-9]{30}$`)}}),
	},
	{
		ID:              "sidekiq-bundler-credential",
		Regex:           `\b(?i:BUNDLE_(?:ENTERPRISE|GEMS)__CONTRIBSYS__COM)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{8}:[a-fA-F0-9]{8})` + catalogRightBoundary,
		Keywords:        []string{"bundle_enterprise__contribsys__com", "bundle_gems__contribsys__com"},
		SecretGroup:     1,
		Source:          "https://github.com/sidekiq/sidekiq/wiki/Commercial-FAQ",
		Description:     "Sidekiq commercial gem-server credential in its exact Bundler environment variable. Both username and password remain in the value; the 8hex:8hex body is the Betterleaks subset, not a full issuer grammar. The separately named URL rule is unchanged.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "weatherstack-api-key-assignment",
		Regex:           `\b(?i:WEATHERSTACK_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"weatherstack_api_key"},
		SecretGroup:     1,
		Source:          "https://blog.weatherstack.com/blog/building-a-simple-javascript-weather-app-using-weatherstack/",
		Description:     "Weatherstack server-side WEATHERSTACK_API_KEY literal documented by the provider. The 32-lowercase-alphanumeric body is a scanner subset; bare opaque values and public location identifiers are excluded. The existing parsed URL rule is unchanged.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID:              "sslmate-api-key-config",
		Regex:           `(?m:^[ \t]*(?:api_endpoint[ \t]+https://sslmate\.com/api/v[23]/?[ \t]*\r?\n[ \t]*api_key[ \t]+([A-Za-z0-9]{36})[ \t]*(?:\r?\n|$)|api_key[ \t]+([A-Za-z0-9]{36})[ \t]*\r?\n[ \t]*api_endpoint[ \t]+https://sslmate\.com/api/v[23]/?[ \t]*(?:\r?\n|$)))`,
		Keywords:        []string{"sslmate.com"},
		SecretGroup:     0,
		Source:          "https://sslmate.com/help/reference/cli1",
		Description:     "SSLMate raw api_key in a complete two-line classic CLI configuration with an exact api_endpoint, in either field order. The opaque 36-alphanumeric body is a scanner subset; filename hints, arbitrary assignments and endpoint overrides are excluded. The existing Basic/request rule is unchanged.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierSSLMate2,
	},
}

// Assignment candidates are complete private-role literals, not a prefix of a
// concatenation or interpolated value. Reuse the audited checks, then inspect
// the text after a closing quote too; the common check stops at that quote.
func validateBetterCarrierAssignment2(value string, start, end int, secret string) contextValidation {
	if !validateAuditedAssignmentContext(value, start, end, secret).accepted || !validateAuditedHeaderLiteral2(value, start, end, secret).accepted {
		return contextValidation{}
	}
	if start > 0 && (isASCIIWordByte(value[start-1]) || strings.ContainsRune(".-", rune(value[start-1])) || value[start-1] >= 0x80) {
		return contextValidation{}
	}
	relative := strings.LastIndex(value[start:end], secret)
	if relative < 0 {
		return contextValidation{}
	}
	position := start + relative + len(secret)
	if position < len(value) && (isASCIIWordByte(value[position]) || strings.ContainsRune("_+/=.-", rune(value[position]))) {
		return contextValidation{}
	}
	quoted := start+relative > start && (value[start+relative-1] == '\'' || value[start+relative-1] == '"')
	if quoted {
		// The audited assignment validator has already checked quote pairing.
		position++
		if position < len(value) && isASCIIWordByte(value[position]) {
			return contextValidation{}
		}
	}
	limit := min(len(value), position+256)
	for position < limit && strings.ContainsRune(" \t\r\n", rune(value[position])) {
		position++
	}
	if position == limit && position < len(value) {
		return contextValidation{}
	}
	if position < len(value) && strings.ContainsRune("([{.$+-*/%?:=!<>|&\\\"'`", rune(value[position])) {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

var betterCarrierMiroID2 = newLazyRegexp(`\b(?i:MIRO_CLIENT_ID)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([0-9]{15,21})` + catalogRightBoundary)

func validateBetterCarrierMiro2(value string, start, end int, secret string) contextValidation {
	if !validateBetterCarrierAssignment2(value, start, end, secret).accepted {
		return contextValidation{}
	}
	low, high := max(0, start-512), min(len(value), end+512)
	for scan := low; scan < high; {
		indices := betterCarrierMiroID2.FindStringSubmatchIndex(value[scan:high])
		if indices == nil {
			break
		}
		idStart, idEnd := scan+indices[0], scan+indices[1]
		id := value[scan+indices[2] : scan+indices[3]]
		if strings.Count(value[min(start, idStart):max(start, idStart)], "\n") <= 5 && validateBetterCarrierAssignment2(value, idStart, idEnd, id).accepted {
			return contextValidation{accepted: true}
		}
		scan = idEnd
	}
	return contextValidation{}
}

// The existing Pinecone prefixed branch retains its interpretation; bare private
// prefixes must not acquire assignment-only context requirements.
func validateBetterCarrierPinecone2(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(secret, "pcsk_") {
		return contextValidation{accepted: true}
	}
	return validateBetterCarrierAssignment2(value, start, end, secret)
}

func validateBetterCarrierSSLMate2(value string, start, end int, _ string) contextValidation {
	// Raw keys have no issuer prefix. Require the complete two-line CLI config,
	// so a later api_endpoint override cannot turn an unrelated key into SSLMate.
	return contextValidation{accepted: len(value) <= maxStructuredCredentialBytes && strings.TrimSpace(value[:start]) == "" && strings.TrimSpace(value[end:]) == ""}
}
