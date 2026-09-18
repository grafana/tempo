package secrets

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash/crc32"
	"net/url"
	"strings"
)

const (
	slackFormatSource       = "https://docs.slack.dev/authentication/tokens/"
	sourcegraphFormatSource = "https://github.com/sourcegraph/openapi/blob/main/openapi.SourcegraphInternal.Latest.yaml"
)

// serviceRuleSpecs combines the pinned Gitleaks v8.30.1 matching baseline with
// explicitly documented provider extensions and native source-backed subsets.
// Adapted Gitleaks portions are covered by LICENSE.gitleaks. Literal-example
// allowlists and inline ignore directives are intentionally not imported.
// Constraints establish detection candidates, not issuer validity or complete grammar.
// Modern Adafruit and Contentful constraints are adapted from Titus v1.2.9
// (LICENSE.titus and NOTICE.titus). PostHog uses provider generator facts with
// a conservative lower candidate bound from TruffleHog v3.97.4.
var serviceRuleSpecs = []catalogRuleSpec{
	{
		ID:          "airtable-personal-access-token",
		Regex:       "\\b(pat[[:alnum:]]{14}\\.[a-f0-9]{64})\\b",
		SecretGroup: 1,
		Keywords:    []string{"airtable"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for airtable-personnal-access-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "adafruit-api-key",
		Regex:       "(?i:[\\w.-]{0,50}?(?:adafruit|(?-i:[Xx]-[Aa][Ii][Oo]-[Kk][Ee][Yy]))(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}([a-z0-9_-]{32})(?:[\\x60'\"\\s;&#]|\\\\[nr]|$))|\\b(aio_[A-Za-z0-9]{28})" + catalogRightBoundary,
		SecretGroup: 0, // Select the participating legacy or modern capture.
		Keywords:    []string{"adafruit", "x-aio-key", "aio_"},
		Source:      gitleaksBaselineSource,
		Description: "Legacy keys retain the pinned Gitleaks v8.30.1 assignment and provider-documented X-AIO-Key header/query acceptance semantics, including ASCII-only header-name case equivalence. Modern case-sensitive aio_ keys are recognized without assignment context using the 28-character ASCII-alphanumeric body from Titus v1.2.9. Body constraints are scanner assumptions, not an exhaustive provider grammar or issuer validation; no external attribute names are used.",
	},
	{
		ID:          "azure-logic-apps-webhook",
		Regex:       `\b(https://(?:[a-z0-9-]+\.[a-z0-9-]+\.logic\.azure\.com/workflows/[A-Za-z0-9_-]+/triggers/[A-Za-z0-9_-]+/paths/invoke/?|[a-z0-9-]+\.azurewebsites\.net(?::443)?/api/[A-Za-z0-9_-]+/triggers/[A-Za-z0-9_-]+/invoke)\?[A-Za-z0-9%_/.=&+\-]{16,1000})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"logic.azure.com", "azurewebsites.net"},
		Source:      "https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-http-endpoint",
		Description: "Azure Logic Apps Standard/Consumption signed callback URL, also used by some Teams Workflows. Requires an actual sig parameter and the documented sp/sv authentication fields, not just a workflow URL. Candidate query length and signature body length are defensive bounds; Power Platform hosts and OAuth-only callbacks are not claimed.",
		Validate:    validLogicAppsWebhook,
	},
	{
		ID:          "confluent-secret-key",
		Regex:       `\b(cflt[A-Za-z0-9+/]{60})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"cflt"},
		Source:      "https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/service-accounts/api-keys/overview.html",
		Description: "Confluent API secret issued in the documented July 2025 format: cflt, 54 Base64-alphabet characters, then the six-character Base64 encoding of the little-endian CRC32 checksum. API key IDs and unprefixed legacy secrets are excluded.",
		Validate:    validConfluentSecret,
	},
	{
		ID:          "contentful-personal-access-token",
		Regex:       `\b(CFPAT-[A-Za-z0-9_-]{43})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"CFPAT-"},
		Source:      "https://www.contentful.com/developers/docs/references/authentication/",
		Description: "Contentful management personal-access-token candidate with the CFPAT- prefix. The 43-character URL-safe body is a Titus v1.2.9 scanner constraint, not a complete issuer grammar inferred from examples. Unprefixed delivery/preview tokens, OAuth values and public token IDs are excluded; a match does not authenticate scope or issuer.",
	},
	{
		ID:          "defined-networking-api-token",
		Regex:       "(?i)[\\w.-]{0,50}?(?:dnkey)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}(dnkey-[a-z0-9=_\\-]{26}-[a-z0-9=_\\-]{52})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"dnkey"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for defined-networking-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "duffel-api-token",
		Regex:       "duffel_(?:test|live)_(?i)[a-z0-9_\\-=]{43}",
		SecretGroup: 0,
		Keywords:    []string{"duffel_"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for duffel-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "dynatrace-api-token",
		Regex:       "dt0c01\\.(?i)[a-z0-9]{24}\\.[a-z0-9]{64}",
		SecretGroup: 0,
		Keywords:    []string{"dt0c01."},
		Entropy:     4,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for dynatrace-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "flutterwave-secret-key",
		Regex:       "FLWSECK_TEST-(?i)[a-h0-9]{32}-X",
		SecretGroup: 0,
		Keywords:    []string{"flwseck_test"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for flutterwave-secret-key. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "grafana-api-key",
		Regex:       `\b(eyJ[A-Za-z0-9+/]{29,997}={0,2})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"eyJ"},
		Source:      "https://github.com/grafana/grafana/blob/main/pkg/components/apikeygen/apikeygen.go",
		Description: "Legacy Grafana API key: padded or unpadded standard-Base64 JSON containing a complete 32-character alphanumeric k secret, n name and numeric id. Missing padding carries no secret bits; both encodings retain strict decoding and structural checks. Encoded candidate size remains defensively capped at 1002 characters.",
		Validate:    validGrafanaLegacyKey,
	},
	{
		ID:          "grafana-cloud-api-token",
		Regex:       "(?i)\\b(glc_[A-Za-z0-9+/]{32,400}={0,3})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"glc_"},
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for grafana-cloud-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "grafana-service-account-token",
		Regex:       `\b(glsa_[A-Za-z0-9]{32}_[a-f0-9]{8})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"glsa_"},
		Source:      "https://github.com/grafana/grafana/blob/main/pkg/components/satokengen/tokengen.go",
		Description: "Grafana service-account token with the vendor-generated 32-character alphanumeric secret and eight-hex little-endian CRC32 checksum over glsa_secret. The checksum is validated locally; malformed or truncated prefixes are not credentials.",
		Validate:    validGrafanaServiceAccountToken,
	},
	{
		ID:          "lob-api-key",
		Regex:       "(?i)[\\w.-]{0,50}?(?:lob)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}((live|test)_[a-f0-9]{35})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"test_", "live_"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for lob-api-key. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "mailchimp-api-key",
		Regex:       "(?i)[\\w.-]{0,50}?(?:MailchimpSDK.initialize|mailchimp)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}([a-f0-9]{32}-us\\d{1,2})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"mailchimp"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for mailchimp-api-key. Retains the documented single-digit data-center suffix alongside the upstream two-digit form, without dropping upstream provider assignment context; the body alphabet and width remain upstream assumptions. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "mapbox-secret-access-token",
		Regex:       `\b(sk\.[A-Za-z0-9_-]{8,1000}\.[A-Za-z0-9_-]{22,172})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sk."},
		Source:      "https://docs.mapbox.com/api/accounts/tokens/",
		Description: "Mapbox confidential sk token with decoded nonempty Base64URL JSON payload and signature bytes. Public pk and temporary tk tokens are excluded. Payload and signature limits (1000 encoded characters and 16-128 decoded signature bytes) are defensive, not a promise of issuer lengths or cryptographic verification.",
		Validate:    validMapboxSecretToken,
	},
	{
		ID:          "maxmind-license-key",
		Regex:       `\b([A-Za-z0-9_]{36}_mmk)` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"_mmk"},
		Source:      "https://dev.maxmind.com/minfraud/release-notes/2023/",
		Description: "MaxMind March 2023 40-character license-key format using alphanumerics/underscore and _mmk suffix. Does not infer an undocumented separator after the six-character prefix or cover older unsuffixed license keys.",
	},
	{
		ID:          "new-relic-user-api-key",
		Regex:       "(?i:[\\w.-]{0,50}?(?:new-relic|newrelic|new_relic)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}(NRAK-[a-z0-9]{27})(?:[\\x60'\"\\s;]|\\\\[nr]|$))|\\b(NRAK-[A-Za-z0-9]{27})" + catalogRightBoundary,
		SecretGroup: 0,
		Keywords:    []string{"nrak"},
		Source:      gitleaksBaselineSource,
		Description: "New Relic confidential user-key candidates: unchanged legacy provider-assignment matching plus standalone case-sensitive NRAK- with a 27-character alphanumeric body from pinned Titus/TruffleHog scanner constraints. Browser/mobile keys and public account IDs are not this family; no issuer validity is inferred.",
	},
	{
		ID:          "plaid-api-token",
		Regex:       "(?i)[\\w.-]{0,50}?(?:plaid)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}(access-(?:sandbox|development|production)-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"plaid"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for plaid-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "posthog-personal-api-key",
		Regex:       `\b(phx_[A-Za-z0-9]{43,49})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"phx_"},
		Source:      "https://github.com/PostHog/posthog/blob/8257ed67904271e9dc7879d3a51723242bceba01/posthog/models/utils.py",
		Description: "PostHog confidential personal API key using phx_ and the ASCII-alphanumeric alphabet of pinned base62/base57 generators. The 43-character lower bound is a conservative TruffleHog v3.97.4 scanner constraint; the 49-character upper bound accommodates current 35-byte base57 generation, not an exhaustive issuance grammar. Public phc_ project tokens and distinct phs_/pha_/phr_ credentials are excluded. No issuer or authorization validation.",
	},
	{
		ID:          "resend-api-key",
		Regex:       `\b(re_[A-Za-z0-9]{8}_[A-Za-z0-9]{24})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"re_"},
		Source:      "https://resend.com/docs/api-reference/api-keys/create-api-key",
		Description: "Resend confidential API key with the documented re_ prefix and two components shown in the create-key response. The eight- and 24-character alphanumeric component widths are conservative Titus v1.2.9 scanner constraints corroborated by that example, not a complete issuance grammar. Public API-key UUIDs and partial prefixes are excluded.",
	},
	{
		ID:          "sendgrid-api-token",
		Regex:       "\\b(SG\\.(?i)[a-z0-9=_\\-\\.]{66})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"sg."},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for sendgrid-api-token. Retains a local check for the documented two nonempty dot-separated components and total length; component widths are no longer independently guessed. The candidate alphabet remains the upstream assumption. Literal-example allowlists and inline ignore directives are not imported.",
		Validate:    validSendGridKey,
	},
	{
		ID:          "sendinblue-api-token",
		Regex:       "\\b(xkeysib-[a-f0-9]{64}\\-(?i)[a-z0-9]{16})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"xkeysib-"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for sendinblue-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "sentry-org-token",
		Regex:       `\b(sntrys_[A-Za-z0-9+/]{16,1000}={0,2}_[A-Za-z0-9+/]{43})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sntrys_"},
		Source:      "https://github.com/getsentry/sentry/blob/master/src/sentry/utils/security/orgauthtoken_token.py",
		Description: "Sentry structural organization token: sntrys_, standard-Base64 JSON facts including issuance time/organization/region URL, and a 32-byte secret encoded without padding. Follows the vendor generator's serialization; decoded fields follow the generator types without inferring organization contents or URL schemes; encoded facts are defensively bounded to 1002 characters.",
		Validate:    validSentryOrgToken,
	},
	{
		ID:          "sentry-user-token",
		Regex:       "\\b(sntryu_[a-f0-9]{64})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"sntryu_"},
		Entropy:     3.5,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for sentry-user-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "shippo-api-token",
		Regex:       "\\b(shippo_(?:live|test)_[a-fA-F0-9]{40})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"shippo_"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for shippo-api-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "shopify-access-token",
		Regex:       "shpat_[a-fA-F0-9]{32}",
		SecretGroup: 0,
		Keywords:    []string{"shpat_"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for shopify-access-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "shopify-private-app-access-token",
		Regex:       "shppa_[a-fA-F0-9]{32}",
		SecretGroup: 0,
		Keywords:    []string{"shppa_"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for shopify-private-app-access-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "sidekiq-sensitive-url",
		Regex:       `\b([Hh][Tt][Tt][Pp][Ss]://[^\s/:@?#"\x27<>]{1,128}:[^\s/@?#"\x27<>]{1,128}@(?i:(?:gems|enterprise)\.contribsys\.com))(?:$|[/:?#\s"\x27<>])`,
		SecretGroup: 1,
		Keywords:    []string{"contribsys.com"},
		Source:      "https://github.com/sidekiq/sidekiq/wiki/Commercial-FAQ",
		Description: "Sidekiq Pro/Enterprise gem-server URL containing both a username and a nonempty password. The URI scheme uses ASCII case equivalence, excluding invalid Unicode scheme aliases; literal Contribsys hostnames compare case-insensitively. Userinfo is not case-folded. URL authority boundaries prevent lookalike-host attribution; userinfo is defensively bounded to 128 characters per component.",
	},
	{
		ID:          "slack-app-token",
		Regex:       `(?i:xapp-\d-[A-Z0-9]+-\d+-[a-z0-9]+)|\bxapp-[0-9]{12}-[A-Za-z0-9/+]{24}` + catalogRightBoundary,
		SecretGroup: 0,
		Keywords:    []string{"xapp"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Slack app-level token candidates retain the original Gitleaks v8.30.1 matching and entropy branch and add the bounded historical xapp-12digits-24character form from Titus v1.2.9. New historical candidates use case-sensitive prefixes and native right boundaries; scanner widths do not establish issuance or active authorization.",
	},
	{
		ID:          "slack-bot-token",
		Regex:       `xoxb-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*|\bxoxb-[0-9]{10,13}-[A-Za-z0-9]{24}` + catalogRightBoundary,
		SecretGroup: 0,
		Keywords:    []string{"xoxb"},
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Slack bot-token candidates retain the original Gitleaks v8.30.1 matching and entropy branch and add the historical one-numeric-component xoxb form with a 24-character alphanumeric secret from Titus v1.2.9. Historical retirement is not treated as nonconfidentiality; no issuer or live-token verification is performed.",
	},
	{
		ID:          "slack-workflow-token",
		Regex:       `\b(xwfp-[A-Za-z0-9_-]{16,1000})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"xwfp-"},
		Source:      slackFormatSource,
		Description: "Slack workflow authentication token with the documented xwfp- prefix. Its short lifetime does not make exposure harmless. URL-safe 16-1000-character body bounds are defensive candidate assumptions, not a published internal grammar; no expiry or API validation is performed.",
	},
	{
		ID:          "slack-refresh-token",
		Regex:       "(?i)xoxe-\\d-[A-Z0-9]{146}",
		SecretGroup: 0,
		Keywords:    []string{"xoxe-"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for slack-config-refresh-token. Preserves upstream context, boundaries, capture and entropy constraints; these are detector constraints, not a complete provider grammar or issuer verification. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "slack-rotating-access-token",
		Regex:       "(?i)xoxe\\.xox[bp]-\\d-[A-Z0-9]{163,166}",
		SecretGroup: 0,
		Keywords:    []string{"xoxe.xoxb-", "xoxe.xoxp-"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for slack-config-access-token. Uses the documented literal rotation separator rather than upstream wildcard punctuation. Upstream configuration-token naming does not establish the installation grant or privileges. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "slack-user-token",
		Regex:       "xox[pe](?:-[0-9]{10,13}){3}-(?:[a-zA-Z0-9-]{6}|[a-zA-Z0-9-]{10}|[a-zA-Z0-9-]{28,34})",
		SecretGroup: 0,
		Keywords:    []string{"xoxp-", "xoxe-"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for slack-user-token. Retains the explicitly documented six- and ten-character pre-2016 final secret sections while keeping upstream identity constraints and alphabet. The alphabet is a detector assumption, not issuer proof. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "slack-webhook-url",
		Regex:       "(?:https?://)?hooks\\.(?:slack\\.com|slack-gov\\.com)/(?:(?:services|workflows|triggers)/)?[A-Za-z0-9+/]{43,56}",
		SecretGroup: 0,
		Keywords:    []string{"hooks.slack.com"},
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for slack-webhook-url. Uses literal standard and GovSlack hosts and the documented legacy path without the services segment. The remaining candidate body length/alphabet are upstream assumptions, not a complete URL grammar. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "sourcegraph-access-token",
		Regex:       `\b(sgp_(?:[a-fA-F0-9]{16}_|local_)?[a-fA-F0-9]{40})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sgp_"},
		Source:      sourcegraphFormatSource,
		Description: "Sourcegraph documented v2/v3 access tokens, requiring sgp_ and the complete 40-hex secret, optionally after the v3 instance ID or local marker. Unprefixed v1 hashes are deliberately excluded.",
	},
	{
		ID:          "sourcegraph-cody-token",
		Regex:       `\b(sgd_[a-fA-F0-9]{64})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sgd_"},
		Source:      sourcegraphFormatSource,
		Description: "Sourcegraph documented backend dotcom user gateway credential granting access to Cody; requires sgd_ and its complete 64-hex secret.",
	},
	{
		ID:          "sourcegraph-license-token",
		Regex:       `\b(slk_[a-fA-F0-9]{64})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"slk_"},
		Source:      sourcegraphFormatSource,
		Description: "Sourcegraph documented backend subscription authentication token derived from a license key; requires slk_ and its complete 64-hex value. This is not a detector for arbitrary license identifiers.",
	},
	{
		ID:          "sourcegraph-subscription-token",
		Regex:       `\b(sgs_[a-fA-F0-9]{64})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sgs_"},
		Source:      sourcegraphFormatSource,
		Description: "Sourcegraph documented Enterprise product-subscription authentication token; requires sgs_ and its complete 64-hex secret.",
	},
	{
		ID:          "stripe-access-token",
		Regex:       "\\b((?:(?:sk|rk)_(?:test|live|prod)_[a-zA-Z0-9]{10,99}|sk_org_[a-zA-Z0-9]{10,99}))(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 1,
		Keywords:    []string{"k_"},
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Gitleaks v8.30.1 matching baseline for stripe-access-token. Adds the documented confidential organization-key prefix with the same upstream body constraint; does not invent a separate organization-key width or broaden restricted keys to organization mode. Literal-example allowlists and inline ignore directives are not imported.",
	},
	{
		ID:          "stripe-webhook-secret",
		Regex:       `\b(whsec_[A-Za-z0-9]{16,256})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"whsec_"},
		Source:      "https://docs.stripe.com/webhooks/signature",
		Description: "Stripe webhook endpoint signing secret, including CLI-issued secrets. The documented whsec_ prefix is required; the 16-256 alphanumeric body bound is defensive, not a promised secret length.",
	},
	{
		ID:          "supabase-personal-access-token",
		Regex:       `\b(sbp_[a-f0-9]{40})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sbp_"},
		Source:      "https://raw.githubusercontent.com/supabase/cli/3db642adde91f7f784437dd54af863791375411e/internal/utils/access_token.go",
		Description: "Supabase confidential Management API personal access token using the pinned provider CLI's sbp_ plus 40 lowercase hexadecimal acceptance constraint. This is client-side format evidence, not proof of issuer validity. OAuth-prefixed access tokens, project URLs, publishable keys and legacy JWTs are outside this PAT rule.",
	},
	{
		ID:          "supabase-secret-key",
		Regex:       `\b(sb_secret_[A-Za-z0-9_-]{31})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sb_secret_"},
		Source:      "https://supabase.com/docs/guides/getting-started/api-keys",
		Description: "Supabase elevated project secret key with the documented sb_secret_ prefix, distinct from public sb_publishable_ keys and project URLs. The 31-character URL-safe body is a conservative Titus v1.2.9 scanner constraint corroborated by a pinned provider CLI default example, not a published hosted-key issuance grammar. No project URL or external attribute context is required.",
	},
	{
		ID:          "tailscale-secret-key",
		Regex:       `\b(tskey-(?:api|auth|client|scim|webhook)-[A-Za-z0-9_]+-[A-Za-z0-9_]+)` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"tskey-"},
		Source:      "https://tailscale.com/docs/reference/key-prefixes",
		Description: "Tailscale case-sensitive API, auth, OAuth-client, SCIM and webhook secrets, the five types documented for secret scanning. Requires both nonempty ID and secret components; ID-only values and arbitrary tskey types are excluded. The alphanumeric/underscore two-component shape is a conservative TruffleHog v3.97.4 scanner constraint, not a promised width or complete issuance grammar. Legacy untyped keys are outside this rule.",
	},
	{
		ID:          "telegram-bot-api-token",
		Regex:       `\b(?:https://api\.telegram\.org/bot)?([0-9]{1,17}:[A-Za-z0-9_-]{22,78}={0,2})(?:$|[/?&\s"\x27<>])`,
		SecretGroup: 1,
		Keywords:    []string{":"},
		Source:      "https://raw.githubusercontent.com/tdlib/telegram-bot-api/e3e9dd8e5b3d7ab8537cd5a10dc31d5ffa8f82d1/telegram-bot-api/ClientManager.cpp",
		Description: "Telegram reference-server structural token subset, bare or in the official HTTPS Bot API carrier: a positive decimal bot ID below 2^54 without a leading zero, at most 80 total characters, and at least 24 characters of canonical Base64url secret. Preserves unpadded and correctly padded encodings accepted by pinned TDLib. No BotFather issuance, secret-byte semantics, deployment routing or authorization validation.",
		Validate:    validTelegramBotToken,
	},
	{
		ID:          "twilio-secret",
		Regex:       `\bTWILIO_(?:AUTH_TOKEN|API_SECRET)[\t ]*[:=][\t ]*["\x27]?([A-Za-z0-9]{16,128})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"TWILIO_AUTH_TOKEN", "TWILIO_API_SECRET"},
		Source:      "https://www.twilio.com/docs/usage/requests-to-twilio",
		Description: "Confidential Twilio Auth Token or API secret in an exact SDK-documented TWILIO_AUTH_TOKEN/TWILIO_API_SECRET assignment. The 16-128 alphanumeric bound is defensive; SK resource SIDs and AC account SIDs are identifiers and are not detected.",
	},
	{
		ID:          "typeform-api-token",
		Regex:       `\b(tfp_[A-Za-z0-9_]{40,59})` + catalogRightBoundary + "|" + "(?i)[\\w.-]{0,50}?(?:typeform)(?:[ \\t\\w.-]{0,20})[\\s'\"]{0,3}(?:=|>|:{1,3}=|\\|\\||:|=>|\\?=|,)[\\x60'\"\\s=]{0,5}(tfp_[a-z0-9\\-_\\.=]{59})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		SecretGroup: 0,
		Keywords:    []string{"tfp_"},
		Source:      "https://www.typeform.com/developers/get-started/personal-access-token/",
		Description: "Typeform personal tokens authorize reading, updating and deleting forms and data. Recognizes case-sensitive tfp_ tokens with 40-59 alphanumeric/underscore body characters directly, in assignments or Bearer headers: the bounded TruffleHog v3.97.4 v2 scanner subset includes the official illustrative 40-hex body. Preserves the Gitleaks v8.30.1 case-insensitive 59-character punctuation-bearing subset only in its existing provider-assignment context. Neither scanner specifies the complete issuer grammar; legacy unprefixed tokens and online verification are outside this rule.",
	},
}

var (
	serviceBase64       = base64.StdEncoding.Strict()
	serviceRawBase64    = base64.RawStdEncoding.Strict()
	serviceRawBase64URL = base64.RawURLEncoding.Strict()
)

func serviceJSON(encoded string, encoding *base64.Encoding, target any) bool {
	if len(encoded) > 1024 {
		return false
	}
	var decoded [768]byte
	n, err := encoding.Decode(decoded[:], []byte(encoded))
	return err == nil && json.Unmarshal(decoded[:n], target) == nil
}

func serviceAlphanumeric(value string) bool {
	for i := range len(value) {
		c := value[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

func validConfluentSecret(value string) bool {
	if len(value) != 64 || !strings.HasPrefix(value, "cflt") {
		return false
	}
	var checksum [4]byte
	n, err := serviceRawBase64.Decode(checksum[:], []byte(value[58:]))
	return err == nil && n == len(checksum) && binary.LittleEndian.Uint32(checksum[:]) == crc32.ChecksumIEEE([]byte(value[4:58]))
}

func validGrafanaLegacyKey(value string) bool {
	var key struct {
		Key   string  `json:"k"`
		Name  *string `json:"n"`
		OrgID *int64  `json:"id"`
	}
	encoding := serviceBase64
	if !strings.HasSuffix(value, "=") {
		encoding = serviceRawBase64
	}
	return serviceJSON(value, encoding, &key) &&
		len(key.Key) == 32 && serviceAlphanumeric(key.Key) && key.Name != nil && key.OrgID != nil
}

func validGrafanaServiceAccountToken(value string) bool {
	if len(value) != 46 || !strings.HasPrefix(value, "glsa_") || value[37] != '_' {
		return false
	}
	var checksum [4]byte
	_, err := hex.Decode(checksum[:], []byte(value[38:]))
	return err == nil && binary.LittleEndian.Uint32(checksum[:]) == crc32.ChecksumIEEE([]byte(value[:37]))
}

func validMapboxSecretToken(value string) bool {
	if !strings.HasPrefix(value, "sk.") {
		return false
	}
	payload, signature, ok := strings.Cut(value[3:], ".")
	if !ok || len(signature) > 172 {
		return false
	}
	var claims map[string]json.RawMessage
	if !serviceJSON(payload, serviceRawBase64URL, &claims) || len(claims) == 0 {
		return false
	}
	var decoded [129]byte
	n, err := serviceRawBase64URL.Decode(decoded[:], []byte(signature))
	return err == nil && n >= 16 && n <= 128
}

func validSendGridKey(value string) bool {
	if len(value) != 69 || !strings.HasPrefix(value, "SG.") {
		return false
	}
	identifier, secret, found := strings.Cut(value[3:], ".")
	return found && identifier != "" && secret != "" && !strings.Contains(secret, ".")
}

// ClientManager::send and Client::start_up bound the ID and encoded secret.
// TDLib bc9c263e2bfee06aaab41e82db51a103376030bc accepts optional correct padding
// but rejects nonzero unused terminal bits. The rule excludes CR/LF, which Go's
// strict Base64 decoder would otherwise ignore. No decoded secret is retained.
func validTelegramBotToken(value string) bool {
	if len(value) == 0 || len(value) > 80 || value[0] == '0' {
		return false
	}
	id, secret, found := strings.Cut(value, ":")
	if !found || len(id) == 0 || len(id) > 17 || len(secret) < 24 {
		return false
	}
	var botID uint64
	for i := range len(id) {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
		botID = botID*10 + uint64(id[i]-'0')
		if botID >= 1<<54 {
			return false
		}
	}
	if botID == 0 {
		return false
	}
	unpadded := strings.TrimRight(secret, "=")
	padding := len(secret) - len(unpadded)
	if padding > 2 || (padding > 0 && len(secret)%4 != 0) {
		return false
	}
	var decoded [60]byte
	_, err := serviceRawBase64URL.Decode(decoded[:], []byte(unpadded))
	return err == nil
}

func validSentryOrgToken(value string) bool {
	if !strings.HasPrefix(value, "sntrys_") {
		return false
	}
	facts, secret, ok := strings.Cut(value[7:], "_")
	if !ok || len(secret) != 43 {
		return false
	}
	var claims struct {
		IssuedAt  *float64 `json:"iat"`
		Org       *string  `json:"org"`
		RegionURL *string  `json:"region_url"`
	}
	if !serviceJSON(facts, serviceBase64, &claims) || claims.IssuedAt == nil || claims.Org == nil || claims.RegionURL == nil {
		return false
	}
	var decoded [32]byte
	n, err := serviceRawBase64.Decode(decoded[:], []byte(secret))
	return err == nil && n == len(decoded)
}

func validLogicAppsWebhook(value string) bool {
	endpoint, err := url.Parse(value)
	if err != nil {
		return false
	}
	query, err := url.ParseQuery(endpoint.RawQuery)
	if err != nil || len(query["sig"]) != 1 || len(query["sp"]) != 1 || len(query["sv"]) != 1 || query.Get("sv") != "1.0" {
		return false
	}
	signature := query.Get("sig")
	if len(signature) < 20 || len(signature) > 256 {
		return false
	}
	for i := range len(signature) {
		c := signature[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' && c != '_' {
			return false
		}
	}
	scope := query.Get("sp")
	return strings.HasPrefix(scope, "/triggers/") && strings.HasSuffix(scope, "/run") && len(scope) > len("/triggers//run")
}
