package secrets

import (
	"encoding/base64"
	"net/url"
	"strings"
)

// Candidate constraints informed by Titus v1.2.9 (LICENSE.titus) and
// TruffleHog v3.97.4 (AGPL-3.0); provider-specific carriers are implemented natively.
var auditedProviderRules4 = []catalogRuleSpec{
	{
		ID:              "holisticdev-api-key",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:HOLISTICDEV_API_KEY)"|\x27(?i:HOLISTICDEV_API_KEY)\x27|(?i:HOLISTICDEV_API_KEY))[ \t]*[:=][ \t]*(?:"([a-f0-9]{64})"|\x27([a-f0-9]{64})\x27|([a-f0-9]{64}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"HOLISTICDEV_API_KEY"},
		Source:          "https://docs.holistic.dev/",
		Description:     "Documented HOLISTICDEV_API_KEY environment literal; account API authentication, not nearby digest-like data.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "honeycomb-api-key-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:X-Honeycomb-Team)"|\x27(?i:X-Honeycomb-Team)\x27|(?i:X-Honeycomb-Team))[ \t]*[:=][ \t]*(?:"((?:[a-f0-9]{32}|[A-Za-z0-9]{22}))"|\x27((?:[a-f0-9]{32}|[A-Za-z0-9]{22}))\x27|((?:[a-f0-9]{32}|[A-Za-z0-9]{22})))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"X-Honeycomb-Team"},
		Source:          honeycombAuthSource,
		Description:     "Complete Honeycomb X-Honeycomb-Team authentication header for opaque legacy and configuration tokens. Current prefixed ingest and management credentials have separate rules; public key IDs are not credentials.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "honeycomb-ingest-key",
		Regex:           `\b(hc[a-z]i[ck]_[a-z0-9]{58})` + catalogRightBoundary,
		Keywords:        []string{"hc"},
		SecretGroup:     1,
		Source:          honeycombAuthSource,
		Description:     "Honeycomb prefixed ingest credentials concatenate the key ID and secret without a separator. Includes environment and Classic ingest keys with the SDK-supported variable regional prefix and 64-character total width, not the shorter public key ID. Lowercase alphanumeric candidate constraints follow the SDK; no issuance or live validity is inferred.",
		ValidateContext: validateHoneycombCredentialContext,
	},
	{
		ID:              "honeycomb-management-key",
		Regex:           `\b(hc[a-z]mk_[a-z0-9]{26}:[a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"hc"},
		SecretGroup:     1,
		Source:          honeycombAuthSource,
		Description:     "Honeycomb management credential combines the public key ID and confidential secret with a colon. Public IDs alone are excluded. Component widths follow the documented complete credential; lowercase alphanumeric bodies are bounded candidates, not claims of issuance, permissions or live validity.",
		ValidateContext: validateHoneycombCredentialContext,
	},
	{
		ID:              "jupiterone-api-token",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:J1_API_TOKEN)"|\x27(?i:J1_API_TOKEN)\x27|(?i:J1_API_TOKEN))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{76})"|\x27([A-Za-z0-9]{76})\x27|([A-Za-z0-9]{76}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"J1_API_TOKEN"},
		Source:          "https://raw.githubusercontent.com/JupiterOne/jupiterone-client-nodejs/main/README.md",
		Description:     "Documented J1_API_TOKEN environment literal supplies the JupiterOne CLI API access token; account IDs alone are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "keen-master-key",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:KEEN_MASTER_KEY)"|\x27(?i:KEEN_MASTER_KEY)\x27|(?i:KEEN_MASTER_KEY))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{64})"|\x27([A-Za-z0-9]{64})\x27|([A-Za-z0-9]{64}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"KEEN_MASTER_KEY"},
		Source:          "https://raw.githubusercontent.com/keenlabs/KeenClient-Python/master/README.rst",
		Description:     "Only the documented KEEN_MASTER_KEY administrator credential is accepted. Public project IDs and read/write keys intended for embedded clients are not inferred to be confidential.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "knapsack-pro-test-suite-token",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_RSPEC)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_RSPEC)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_RSPEC))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_MINITEST)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_MINITEST)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_MINITEST))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_TEST_UNIT)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_TEST_UNIT)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_TEST_UNIT))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_CUCUMBER)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_CUCUMBER)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_CUCUMBER))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_SPINACH)"|\x27(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_SPINACH)\x27|(?i:KNAPSACK_PRO_TEST_SUITE_TOKEN_SPINACH))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])|(?:^|[^A-Za-z0-9_.-])(?:"(?i:KNAPSACK-PRO-TEST-SUITE-TOKEN)"|\x27(?i:KNAPSACK-PRO-TEST-SUITE-TOKEN)\x27|(?i:KNAPSACK-PRO-TEST-SUITE-TOKEN))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"KNAPSACK_PRO_TEST_SUITE_TOKEN", "KNAPSACK_PRO_TEST_SUITE_TOKEN_RSPEC", "KNAPSACK_PRO_TEST_SUITE_TOKEN_MINITEST", "KNAPSACK_PRO_TEST_SUITE_TOKEN_TEST_UNIT", "KNAPSACK_PRO_TEST_SUITE_TOKEN_CUCUMBER", "KNAPSACK_PRO_TEST_SUITE_TOKEN_SPINACH", "KNAPSACK-PRO-TEST-SUITE-TOKEN"},
		Source:          "https://raw.githubusercontent.com/KnapsackPro/knapsack_pro-ruby/master/lib/knapsack_pro/config/env.rb",
		Description:     "Documented Knapsack Pro test-suite-token environment variables and API authentication header; runner-specific suffixes come from the SDK, not arbitrary brand-derived names.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "line-channel-access-token",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:LINE_CHANNEL_ACCESS_TOKEN)"|\x27(?i:LINE_CHANNEL_ACCESS_TOKEN)\x27|(?i:LINE_CHANNEL_ACCESS_TOKEN))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{171}=?)"|\x27([A-Za-z0-9+/]{171}=?)\x27|([A-Za-z0-9+/]{171}=?))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"LINE_CHANNEL_ACCESS_TOKEN"},
		Source:          "https://raw.githubusercontent.com/line/line-bot-sdk-python/master/examples/flask-echo/app.py",
		Description:     "Documented LINE_CHANNEL_ACCESS_TOKEN server SDK environment literal; the historical 171-character Base64-like body comes from Titus, not an exhaustive token grammar.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "loadmill-api-token",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:LOADMILL_API_TOKEN)"|\x27(?i:LOADMILL_API_TOKEN)\x27|(?i:LOADMILL_API_TOKEN))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|\x27([A-Za-z0-9]{40})\x27|([A-Za-z0-9]{40}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"LOADMILL_API_TOKEN"},
		Source:          "https://raw.githubusercontent.com/loadmill/loadmill-node/master/README.md",
		Description:     "Documented LOADMILL_API_TOKEN environment literal; possession permits creating and running load tests in the account.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "lookersdk-client-secret",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:LOOKERSDK_CLIENT_SECRET)"|\x27(?i:LOOKERSDK_CLIENT_SECRET)\x27|(?i:LOOKERSDK_CLIENT_SECRET))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{24})"|\x27([A-Za-z0-9]{24})\x27|([A-Za-z0-9]{24}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"LOOKERSDK_CLIENT_SECRET"},
		Source:          "https://raw.githubusercontent.com/looker-open-source/sdk-codegen/main/python/README.rst",
		Description:     "Documented LOOKERSDK_CLIENT_SECRET environment literal; the SDK authenticates using this confidential secret and a separate public client ID.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "klipfolio-api-key-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:kf-api-key)"|\x27(?i:kf-api-key)\x27|(?i:kf-api-key))[ \t]*[:=][ \t]*(?:"([a-f0-9]{40})"|\x27([a-f0-9]{40})\x27|([a-f0-9]{40}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"kf-api-key"},
		Source:          "https://apidocs.klipfolio.com/reference/getting-started",
		Description:     "Klipfolio kf-api-key authentication header, carrying the account API key rather than an arbitrary SHA-1-like value.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "mailerlite-api-key-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:X-MailerLite-ApiKey)"|\x27(?i:X-MailerLite-ApiKey)\x27|(?i:X-MailerLite-ApiKey))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\x27([a-z0-9]{32})\x27|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"X-MailerLite-ApiKey"},
		Source:          "https://developers-classic.mailerlite.com/docs/authentication",
		Description:     "MailerLite Classic X-MailerLite-ApiKey authentication header. The provider explicitly requires server-side storage and forbids public client-side exposure.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "linkpreview-api-key-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:X-Linkpreview-Api-Key)"|\x27(?i:X-Linkpreview-Api-Key)\x27|(?i:X-Linkpreview-Api-Key))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{32})"|\x27([A-Za-z0-9]{32})\x27|([A-Za-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{"X-Linkpreview-Api-Key"},
		Source:          "https://docs.linkpreview.net/",
		Description:     "Documented X-Linkpreview-Api-Key authentication header; provider recommends server-side use to safeguard API keys from public exposure.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "gyazo-access-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.gyazo\.com)/api/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_token=([A-Za-z0-9-]{43})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.gyazo.com"},
		Source:      "https://gyazo.com/api/docs/image",
		Description: providerAccessTokenURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "harvest-access-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.harvestapp\.com)/v2/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_token=([A-Za-z0-9._]{97})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.harvestapp.com"},
		Source:      "https://help.getharvest.com/api-v2/authentication-api/authentication/authentication/",
		Description: providerAccessTokenURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "holidayapi-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://holidayapi\.com)/v1/[A-Za-z0-9_/-]{1,64}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}key=([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"holidayapi.com"},
		Source:      "https://holidayapi.com/docs",
		Description: "Provider-authentication URL with exact host/path and documented key credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "humanity-access-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://www\.humanity\.com)/api/v2/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_token=([a-z0-9]{40})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"www.humanity.com"},
		Source:      "https://platform.humanity.com/docs/getting-started-with-authentication",
		Description: providerAccessTokenURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "hunter-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.hunter\.io)/v2/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}api_key=([a-z0-9_-]{40})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.hunter.io"},
		Source:      "https://hunter.io/api-documentation/v2",
		Description: providerAPIKeyURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "hybiscus-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.hybiscus\.dev)/api/v1/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}api_key=([A-Za-z0-9_-]{43})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.hybiscus.dev"},
		Source:      "https://hybiscus.dev/docs/api/authentication",
		Description: providerAPIKeyURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "intrinio-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api-v2\.intrinio\.com)/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}api_key=([A-Za-z0-9]{44})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api-v2.intrinio.com"},
		Source:      "https://raw.githubusercontent.com/intrinio/javascript-sdk/master/src/ApiClient.js",
		Description: providerAPIKeyURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "invoiceocean-api-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.invoiceocean\.com)/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}api_token=([A-Za-z0-9]{20})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"invoiceocean.com"},
		Source:      "https://github.com/invoiceocean/API",
		Description: "Provider-authentication URL with exact host/path and documented api_token credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "jotform-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://(?:api|eu-api|hipaa-api)\.jotform\.com)/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}apiKey=([A-Za-z0-9]{32})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.jotform.com"},
		Source:      "https://api.jotform.com/docs/",
		Description: "Provider-authentication URL with exact host/path and documented apiKey credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "kanbantool-access-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.kanbantool\.com)/api/v3/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_token=([A-Z0-9]{12})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"kanbantool.com"},
		Source:      "https://kanbantool.com/developer/api-v3",
		Description: providerAccessTokenURLDescription,
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "karmacrm-api-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://app\.karmacrm\.com)/api/v[23]/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}api_token=([A-Za-z0-9]{20})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"app.karmacrm.com"},
		Source:      "https://docs.karmacrm.com/",
		Description: "Provider-authentication URL with exact host/path and documented api_token credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "kickbox-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.kickbox\.com)/v2/verify\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}apikey=([A-Za-z0-9_]{1,128}[A-Za-z0-9]{64})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.kickbox.com"},
		Source:      "https://docs.kickbox.com/docs/single-verification-api",
		Description: "Provider-authentication URL with exact host/path and documented apikey credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "languagelayer-access-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.languagelayer\.com)/(?:detect|languages)\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_key=([a-z0-9]{32})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.languagelayer.com"},
		Source:      "https://languagelayer.com/documentation",
		Description: "Provider-authentication URL with exact host/path and documented access_key credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "linkpreview-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.linkpreview\.net)/\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}key=([A-Za-z0-9]{32})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.linkpreview.net"},
		Source:      "https://docs.linkpreview.net/",
		Description: "Provider-authentication URL with exact host/path and documented key credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "mailboxlayer-access-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://apilayer\.net)/api/check\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}access_key=([a-z0-9]{32})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{apilayerHost},
		Source:      "https://mailboxlayer.com/documentation",
		Description: "Provider-authentication URL with exact host/path and documented access_key credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:          "kagi-session-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://kagi\.com)/search\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}token=([A-Za-z0-9_-]{11}\.[A-Za-z0-9_-]{43})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"kagi.com"},
		Source:      "https://help.kagi.com/kagi/privacy/private-browser-sessions.html",
		Description: "Provider-authentication URL with exact host/path and documented token credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:              "holisticdev-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([a-f0-9]{64})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.holistic\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.holistic\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([a-f0-9]{64})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.holistic\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([a-f0-9]{64})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.holistic\.dev)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([a-f0-9]{64})\r?(?:\n|$))`,
		Keywords:        []string{"api.holistic.dev"},
		Source:          "https://docs.holistic.dev/",
		Description:     providerAPIKeyRequestDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.holistic\.dev`, `/api/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderLower, pattern: newLazyRegexp(`^[a-f0-9]{64}$`)}}, nil, nil, false)),
	},
	{
		ID:              "html2pdf-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([A-Za-z0-9]{64})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.html2pdf\.app)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.html2pdf\.app)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([A-Za-z0-9]{64})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.html2pdf\.app)/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([A-Za-z0-9]{64})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.html2pdf\.app)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([A-Za-z0-9]{64})\r?(?:\n|$))`,
		Keywords:        []string{"api.html2pdf.app"},
		Source:          "https://html2pdf.app/documentation/",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented X-API-Key account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.html2pdf\.app`, `/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderMixed, pattern: newLazyRegexp(`^[A-Za-z0-9]{64}$`)}}, nil, nil, false)),
	},
	{
		ID:              "hybiscus-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-KEY):[ \t]*([A-Za-z0-9_-]{43})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.hybiscus\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.hybiscus\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-KEY):[ \t]*([A-Za-z0-9_-]{43})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.hybiscus\.dev)/api/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-KEY):[ \t]*([A-Za-z0-9_-]{43})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.hybiscus\.dev)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-KEY):[ \t]*([A-Za-z0-9_-]{43})\r?(?:\n|$))`,
		Keywords:        []string{"api.hybiscus.dev"},
		Source:          "https://hybiscus.dev/docs/api/authentication",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented X-API-KEY account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.hybiscus\.dev`, `/api/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderUpper, pattern: newLazyRegexp(`^[A-Za-z0-9_-]{43}$`)}}, nil, nil, false)),
	},
	{
		ID:              "impala-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9_]{46})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://(?:sandbox|api)\.impala\.travel)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://(?:sandbox|api)\.impala\.travel)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9_]{46})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://(?:sandbox|api)\.impala\.travel)/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9_]{46})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:(?:sandbox|api)\.impala\.travel)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9_]{46})\r?(?:\n|$))`,
		Keywords:        []string{"impala.travel"},
		Source:          "https://api.apis.guru/v2/specs/impala.travel/hotels/1.003/openapi.yaml",
		Description:     providerAPIKeyRequestDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`(?:sandbox|api)\.impala\.travel`, `/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderLower, pattern: newLazyRegexp(`^[A-Za-z0-9_]{46}$`)}}, nil, nil, false)),
	},
	{
		ID:              "interseller-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://(?:api\.)?interseller\.io)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://(?:api\.)?interseller\.io)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://(?:api\.)?interseller\.io)/api/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:(?:api\.)?interseller\.io)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})\r?(?:\n|$))`,
		Keywords:        []string{"interseller.io"},
		Source:          "https://interseller.readme.io/reference/authentication",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented X-API-Key account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`(?:api\.)?interseller\.io`, `/api/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderMixed, pattern: newLazyRegexp(`^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$`)}}, nil, nil, false)),
	},
	{
		ID:              "jumpcloud-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9]{40})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://console\.jumpcloud\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://console\.jumpcloud\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9]{40})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://console\.jumpcloud\.com)/api/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9]{40})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:console\.jumpcloud\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9]{40})\r?(?:\n|$))`,
		Keywords:        []string{"console.jumpcloud.com"},
		Source:          "https://raw.githubusercontent.com/TheJumpCloud/jcapi-python/master/README.md",
		Description:     providerAPIKeyRequestDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`console\.jumpcloud\.com`, `/api/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderLower, pattern: newLazyRegexp(`^[A-Za-z0-9]{40}$`)}}, nil, nil, false)),
	},
	{
		ID:              "juro-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9]{40})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.juro\.com)/v3/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.juro\.com)/v3/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:x-api-key):[ \t]*([A-Za-z0-9]{40})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.juro\.com)/v3/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9]{40})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v3/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.juro\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:x-api-key):[ \t]*([A-Za-z0-9]{40})\r?(?:\n|$))`,
		Keywords:        []string{"api.juro.com"},
		Source:          "https://api-docs.juro.com/",
		Description:     providerAPIKeyRequestDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.juro\.com`, `/v3/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderLower, pattern: newLazyRegexp(`^[A-Za-z0-9]{40}$`)}}, nil, nil, false)),
	},
	{
		ID:              "kylas-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:api-key):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.kylas\.io)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.kylas\.io)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:api-key):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.kylas\.io)/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:api-key):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.kylas\.io)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:api-key):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})\r?(?:\n|$))`,
		Keywords:        []string{"api.kylas.io"},
		Source:          "https://support.kylas.io/portal/en/kb/articles/how-to-use-api-key-in-to-access-kylas-apis",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented api-key account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.kylas\.io`, `/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyFieldHyphen, pattern: newLazyRegexp(`^[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12}$`)}}, nil, nil, false)),
	},
	{
		ID:              "liveagent-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:apikey):[ \t]*([A-Za-z0-9]{32})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.ladesk\.com)/api/v3/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.ladesk\.com)/api/v3/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:apikey):[ \t]*([A-Za-z0-9]{32})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.ladesk\.com)/api/v3/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:apikey):[ \t]*([A-Za-z0-9]{32})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/v3/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.ladesk\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:apikey):[ \t]*([A-Za-z0-9]{32})\r?(?:\n|$))`,
		Keywords:        []string{"ladesk.com"},
		Source:          "https://support.liveagent.com/741982-API-key",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented apikey account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.ladesk\.com`, `/api/v3/[^?#]*`, []auditedProviderHeader2{{name: apiKeyFieldLower, pattern: newLazyRegexp(`^[A-Za-z0-9]{32}$`)}}, nil, nil, false)),
	},
	{
		ID:              "iterable-server-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Api-Key):[ \t]*([A-Za-z0-9]{32,36})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api(?:\.eu)?\.iterable\.com)/api/campaigns(?:\?[^\s\x22\x27\x60<>]{0,256})?(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api(?:\.eu)?\.iterable\.com)/api/campaigns(?:\?[^\s\x22\x27\x60<>]{0,256})?[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Api-Key):[ \t]*([A-Za-z0-9]{32,36})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api(?:\.eu)?\.iterable\.com)/api/campaigns(?:\?[^\s\x22\x27\x60<>]{0,256})? HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Api-Key):[ \t]*([A-Za-z0-9]{32,36})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/campaigns(?:\?[^\s]{0,256})? HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api(?:\.eu)?\.iterable\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Api-Key):[ \t]*([A-Za-z0-9]{32,36})\r?(?:\n|$))`,
		Keywords:        []string{"api.iterable.com"},
		Source:          "https://support.iterable.com/hc/en-us/articles/360043464871-API-Keys",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented Api-Key account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api(?:\.eu)?\.iterable\.com`, `/api/campaigns`, []auditedProviderHeader2{{name: "Api-Key", pattern: newLazyRegexp(`^[A-Za-z0-9]{32,36}$`)}}, nil, nil, false)),
	},
	{
		ID:              "lessannoyingcrm-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*([A-Za-z0-9-]{57})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.lessannoyingcrm\.com)/v2/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.lessannoyingcrm\.com)/v2/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*([A-Za-z0-9-]{57})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.lessannoyingcrm\.com)/v2/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*([A-Za-z0-9-]{57})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v2/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.lessannoyingcrm\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*([A-Za-z0-9-]{57})\r?(?:\n|$))`,
		Keywords:        []string{"api.lessannoyingcrm.com"},
		Source:          "https://account.lessannoyingcrm.com/api_docs/v2/Getting_Started/Connect",
		Description:     "Complete provider-bound curl or HTTP request carrying the documented Authorization account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.lessannoyingcrm\.com`, `/v2/[^?#]*`, []auditedProviderHeader2{{name: authorizationHeader, pattern: newLazyRegexp(`^[A-Za-z0-9-]{57}$`)}}, nil, nil, false)),
	},
	{
		ID:              "leadfeeder-token-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:Authorization)"|\x27(?i:Authorization)\x27|(?i:Authorization))[ \t]*[:=][ \t]*(?:"((?i:Token)[ \t]+token=[A-Za-z0-9-]{43})"|\x27((?i:Token)[ \t]+token=[A-Za-z0-9-]{43})\x27|((?i:Token)[ \t]+token=[A-Za-z0-9-]{43}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{authorizationHeader},
		Source:          "https://docs.leadfeeder.com/api/",
		Description:     "Complete Authorization: Token token= carrier documented by Leadfeeder. The credential is confidential despite the legacy endpoint status; no bare 43-character token attribution.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "kagi-api-token-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:Authorization)"|\x27(?i:Authorization)\x27|(?i:Authorization))[ \t]*[:=][ \t]*(?:"((?i:Bot)[ \t]+[A-Za-z0-9_-]{11}\.[A-Za-z0-9_-]{43})"|\x27((?i:Bot)[ \t]+[A-Za-z0-9_-]{11}\.[A-Za-z0-9_-]{43})\x27|((?i:Bot)[ \t]+[A-Za-z0-9_-]{11}\.[A-Za-z0-9_-]{43}))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{authorizationHeader},
		Source:          "https://help.kagi.com/kagi/api/search.html",
		Description:     "Complete Authorization: Bot credential in the pinned Kagi dotted-token candidate shape; the common bearer rule does not cover this authentication scheme.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "instabot-master-api-key-header",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?:"(?i:Authorization)"|\x27(?i:Authorization)\x27|(?i:Authorization))[ \t]*[:=][ \t]*(?:"((?i:X-Instabot-Master-Api-Key)[ \t]+[A-Za-z0-9+/]{43}=?)"|\x27((?i:X-Instabot-Master-Api-Key)[ \t]+[A-Za-z0-9+/]{43}=?)\x27|((?i:X-Instabot-Master-Api-Key)[ \t]+[A-Za-z0-9+/]{43}=?))(?:$|[\s\x22\x27\x60,;}\]]))`,
		Keywords:        []string{authorizationHeader},
		Source:          "https://docs.instabot.io/docs/serverapi",
		Description:     "Only the explicit X-Instabot-Master-Api-Key authorization scheme establishes administrative secrecy. X-Instabot-Api-Key merely identifies an application and is deliberately excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "ipqualityscore-api-key-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://(?:www\.)?ipqualityscore\.com)/api/(?:json|xml)/(?:ip|email|phone|url|account)/([A-Za-z0-9]{32})(?:$|[/\s\x22\x27\x60?#])`,
		Keywords:    []string{"ipqualityscore.com"},
		Source:      "https://www.ipqualityscore.com/documentation/proxy-detection-api/overview",
		Description: "Complete IPQualityScore API URL with the key in its documented path segment, not an arbitrary 32-character value near the provider name.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:              "gtmetrix-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:gtmetrix\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([a-f0-9]{32}:?)@(?i:gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{32}:)[\x22\x27]|([a-f0-9]{32}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://gtmetrix\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{32}:)[\x22\x27]|([a-f0-9]{32}:)(?:[ \t]|$)))`,
		Keywords:        []string{"gtmetrix.com"},
		Source:          "https://gtmetrix.com/api/docs/2.0/",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4BasicHex32,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`gtmetrix\.com`, `/api/[^?#]*`, validProvider4BasicHex32)),
	},
	{
		ID:              "hellosign-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v3/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.hellosign\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([A-Za-z0-9+]{64}:?)@(?i:api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9+/]{64}:)[\x22\x27]|([A-Za-z0-9+/]{64}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.hellosign\.com)/v3/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9+/]{64}:)[\x22\x27]|([A-Za-z0-9+/]{64}:)(?:[ \t]|$)))`,
		Keywords:        []string{"api.hellosign.com"},
		Source:          "https://developers.hellosign.com/api/reference/authentication/",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4Basic64,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`api\.hellosign\.com`, `/v3/[^?#]*`, validProvider4Basic64)),
	},
	{
		ID:              "hiveage-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([A-Za-z0-9_-]{20}:?)@(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9_-]{20}:)[\x22\x27]|([A-Za-z0-9_-]{20}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9_-]{20}:)[\x22\x27]|([A-Za-z0-9_-]{20}:)(?:[ \t]|$)))`,
		Keywords:        []string{"hiveage.com"},
		Source:          "https://www.hiveage.com/api/",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4Basic20,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.hiveage\.com`, `/api/[^?#]*`, validProvider4Basic20)),
	},
	{
		ID:              "insightly-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v3\.1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api(?:\.[a-z]{2}[0-9])?\.insightly\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}:?)@(?i:api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}:)[\x22\x27]|([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api(?:\.[a-z]{2}[0-9])?\.insightly\.com)/v3\.1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}:)[\x22\x27]|([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}:)(?:[ \t]|$)))`,
		Keywords:        []string{"insightly.com"},
		Source:          "https://api.na1.insightly.com/v3.1/Help/_OverviewResource",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4BasicUUID,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`api(?:\.[a-z]{2}[0-9])?\.insightly\.com`, `/v3\.1/[^?#]*`, validProvider4BasicUUID)),
	},
	{
		ID:              "madkudu-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.madkudu\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([a-f0-9]{32}:?)@(?i:api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{32}:)[\x22\x27]|([a-f0-9]{32}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([a-f0-9]{32}:)[\x22\x27]|([a-f0-9]{32}:)(?:[ \t]|$)))`,
		Keywords:        []string{"api.madkudu.com"},
		Source:          "https://developers.madkudu.com/getting-started/quickstart",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4BasicHex32,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`api\.madkudu\.com`, `/v1/[^?#]*`, validProvider4BasicHex32)),
	},
	{
		ID:              "loadmill-api-basic-credential",
		Regex:           `(?:(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:app\.loadmill\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Authorization):[ \t]*((?i:Basic)[ \t]+[A-Za-z0-9+/]{4,256}={0,2})\r?(?:\n|$))|(?:^|[^A-Za-z0-9_.:/-])(?i:https://)([A-Za-z0-9]{40}:?)@(?i:app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256}|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9]{40}:)[\x22\x27]|([A-Za-z0-9]{40}:)(?:[ \t]|$))(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://app\.loadmill\.com)/api/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-u|--user)[ \t]+(?:[\x22\x27]([A-Za-z0-9]{40}:)[\x22\x27]|([A-Za-z0-9]{40}:)(?:[ \t]|$)))`,
		Keywords:        []string{"app.loadmill.com"},
		Source:          "https://docs.loadmill.com/administration-and-deployment/api-tokens.md",
		Description:     providerBasicUsernameDescription,
		Validate:        validProvider4Basic40,
		ValidateContext: withAuditedAssignmentContext(provider4BasicRequestContext(`app\.loadmill\.com`, `/api/[^?#]*`, validProvider4Basic40)),
	},
	{
		ID:              "mandrill-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://mandrillapp\.com)/api/1\.[03]/[A-Za-z0-9_/-]{1,128}(?:\.json)?[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27\{[^\r\n\x27]{0,512}?"key"[ \t]*:[ \t]*"([A-Za-z0-9_-]{22})"[^\r\n\x27]{0,512}?\}\x27|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27\{[^\r\n\x27]{0,512}?"key"[ \t]*:[ \t]*"([A-Za-z0-9_-]{22})"[^\r\n\x27]{0,512}?\}\x27(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://mandrillapp\.com)/api/1\.[03]/[A-Za-z0-9_/-]{1,128}(?:\.json)?(?:$|[\s\x22\x27])|\bPOST (?i:https://mandrillapp\.com)/api/1\.[03]/[A-Za-z0-9_/-]{1,128}(?:\.json)? HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}\r?\n\{[^\r\n\x27]{0,512}?"key"[ \t]*:[ \t]*"([A-Za-z0-9_-]{22})"[^\r\n\x27]{0,512}?\}(?:\r?$|\r?\n))`,
		Keywords:        []string{"mandrillapp.com"},
		Source:          "https://raw.githubusercontent.com/mailchimp/mailchimp-transactional-node/master/src/ApiClient.js",
		Description:     "Complete Mandrill/Mailchimp Transactional request with confidential key in the JSON body. The official API client sets body.key; generic key assignments and public IDs are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`mandrillapp\.com`, `/api/1\.[03]/[A-Za-z0-9_/-]{1,128}(?:\.json)?`, nil, []auditedProviderField2{{name: keyField, pattern: newLazyRegexp(`^[A-Za-z0-9_-]{22}$`)}}, nil, false)),
	},
	{
		ID:              "instamojo-oauth-client-secret",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://(?:api|www)\.instamojo\.com)/oauth2/token/[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27((?:[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}&){0,8}client_secret=(?:[A-Za-z0-9]{128})(?:&[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}){0,8})\x27|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27((?:[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}&){0,8}client_secret=(?:[A-Za-z0-9]{128})(?:&[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}){0,8})\x27(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://(?:api|www)\.instamojo\.com)/oauth2/token/(?:$|[\s\x22\x27])|\bPOST (?i:https://(?:api|www)\.instamojo\.com)/oauth2/token/ HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}\r?\n((?:[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}&){0,8}client_secret=(?:[A-Za-z0-9]{128})(?:&[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}){0,8})(?:\r?$|\r?\n))`,
		Keywords:        []string{"instamojo.com"},
		Source:          "https://docs.instamojo.com/v2-mdp/reference/authentication-flow",
		Description:     "Complete Instamojo OAuth token request carrying the confidential client_secret form field; the separate public client_id is never classified alone.",
		Validate:        validProvider4InstamojoForm,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`(?:api|www)\.instamojo\.com`, `/oauth2/token/`, nil, []auditedProviderField2{{name: clientSecretField, pattern: newLazyRegexp(`^[A-Za-z0-9]{128}$`)}}, nil, false)),
	},
	{
		ID:              "ibm-cloud-cli-api-key",
		Regex:           `\bibmcloud[ \t]+login[ \t]+(?:-[A-Za-z-]+[ \t]+[A-Za-z0-9_.:/-]+[ \t]+){0,8}--apikey(?:[ \t]+|=)(?:"([A-Za-z0-9_-]{42,44})"|\x27([A-Za-z0-9_-]{42,44})\x27|([A-Za-z0-9_-]{42,44}))(?:$|[\s;])`,
		Keywords:        []string{"ibmcloud"},
		Source:          "https://raw.githubusercontent.com/ibm-cloud-docs/cli/master/reference/ibmcloud/bx_cli.md",
		Description:     "Documented ibmcloud login --apikey literal credential. IBM API keys are confidential IAM authentication credentials; the pinned 42-44 character opaque shape is not standalone attribution.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider4LiteralContext),
	},
	{
		ID:              "madkudu-api-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([a-f0-9]{32})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-Key):[ \t]*([a-f0-9]{32})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.madkudu\.com)/v1/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([a-f0-9]{32})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v1/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.madkudu\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-Key):[ \t]*([a-f0-9]{32})\r?(?:\n|$))`,
		Keywords:        []string{"api.madkudu.com"},
		Source:          "https://developers.madkudu.com/getting-started/quickstart",
		Description:     "Complete MadKudu API request with the documented X-API-Key header. Provider explicitly prohibits exposure in public repositories and client-side code.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.madkudu\.com`, `/v1/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderMixed, pattern: newLazyRegexp(`^[a-f0-9]{32}$`)}}, nil, nil, false)),
	},
	{
		ID:          "lessannoyingcrm-api-token-url",
		Regex:       `(?:^|[^A-Za-z0-9_.:/-])(?i:https?://api\.lessannoyingcrm\.com)/?\?(?:[A-Za-z0-9_.~-]+(?:=[^&\s\x22\x27\x60#<>]{0,64})?&){0,12}APIToken=([A-Za-z0-9-]{57})(?:$|[&#\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.lessannoyingcrm.com"},
		Source:      "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/lessannoyingcrm",
		Description: "Historical Less Annoying CRM API URL carrying APIToken; retains a clearly confidential historical carrier rather than rejecting it because the current API uses Authorization headers.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:              "hunter-api-key-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-KEY):[ \t]*([a-z0-9_-]{40})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.hunter\.io)/v2/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.hunter\.io)/v2/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-KEY):[ \t]*([a-z0-9_-]{40})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.hunter\.io)/v2/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-KEY):[ \t]*([a-z0-9_-]{40})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v2/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.hunter\.io)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-KEY):[ \t]*([a-z0-9_-]{40})\r?(?:\n|$))`,
		Keywords:        []string{"api.hunter.io"},
		Source:          "https://hunter.io/api-documentation/v2",
		Description:     "Complete Hunter API request with the documented X-API-KEY alternative to Bearer and api_key query authentication.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.hunter\.io`, `/v2/[^?#]*`, []auditedProviderHeader2{{name: apiKeyHeaderUpper, pattern: newLazyRegexp(`^[a-z0-9_-]{40}$`)}}, nil, nil, false)),
	},
	{
		ID:              "invoiceocean-api-token-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.invoiceocean\.com)/[A-Za-z0-9_./-]{1,128}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27\{[^\r\n\x27]{0,512}?"api_token"[ \t]*:[ \t]*"([A-Za-z0-9]{20})"[^\r\n\x27]{0,512}?\}\x27|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27\{[^\r\n\x27]{0,512}?"api_token"[ \t]*:[ \t]*"([A-Za-z0-9]{20})"[^\r\n\x27]{0,512}?\}\x27(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.invoiceocean\.com)/[A-Za-z0-9_./-]{1,128}(?:$|[\s\x22\x27])|\bPOST (?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.invoiceocean\.com)/[A-Za-z0-9_./-]{1,128} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}\r?\n\{[^\r\n\x27]{0,512}?"api_token"[ \t]*:[ \t]*"([A-Za-z0-9]{20})"[^\r\n\x27]{0,512}?\}(?:\r?$|\r?\n))`,
		Keywords:        []string{"invoiceocean.com"},
		Source:          "https://github.com/invoiceocean/API",
		Description:     "Complete InvoiceOcean JSON API request with confidential api_token body field, separately from the documented URL query carrier.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.invoiceocean\.com`, `/[A-Za-z0-9_./-]{1,128}`, nil, []auditedProviderField2{{name: apiTokenField, pattern: newLazyRegexp(`^[A-Za-z0-9]{20}$`)}}, nil, false)),
	},
	{
		ID:              "ibm-cloud-iam-api-key",
		Regex:           `(?:^|[\s\x22\x27])((?:grant_type=urn(?::|%3[Aa])ibm(?::|%3[Aa])params(?::|%3[Aa])oauth(?::|%3[Aa])grant-type(?::|%3[Aa])apikey&apikey=[A-Za-z0-9_-]{42,44}|apikey=[A-Za-z0-9_-]{42,44}&grant_type=urn(?::|%3[Aa])ibm(?::|%3[Aa])params(?::|%3[Aa])oauth(?::|%3[Aa])grant-type(?::|%3[Aa])apikey)(?:&response_type=cloud_iam)?)(?:$|[\s\x22\x27;])`,
		Keywords:        []string{"grant-type"},
		Source:          "https://raw.githubusercontent.com/IBM/python-sdk-core/master/README.md",
		Description:     "Complete IBM IAM apikey grant form: the issuer-specific grant_type and apikey field establish private role together in one value. No arbitrary IBM-proximity or generic API key assignment.",
		Validate:        validProvider4IBMForm,
		ValidateContext: withAuditedAssignmentContext(validateProvider4LiteralContext),
	},
	{
		ID:              "logzio-api-token-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-TOKEN):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api(?:-[a-z]{2,8})?\.logz\.io)/v2/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api(?:-[a-z]{2,8})?\.logz\.io)/v2/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-API-TOKEN):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api(?:-[a-z]{2,8})?\.logz\.io)/v2/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-TOKEN):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /v2/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api(?:-[a-z]{2,8})?\.logz\.io)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-API-TOKEN):[ \t]*([a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})\r?(?:\n|$))`,
		Keywords:        []string{"logz.io"},
		Source:          "https://docs.logz.io/api/",
		Description:     "Complete Logz.io API request with confidential X-API-TOKEN authentication. The token may change accounts and users; UUIDs and unbound generic headers are not standalone credentials.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api(?:-[a-z]{2,8})?\.logz\.io`, `/v2/[^?#]*`, []auditedProviderHeader2{{name: "X-API-TOKEN", pattern: newLazyRegexp(`^[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12}$`)}}, nil, nil, false)),
	},
	{
		ID:              "lokalise-api-token-request",
		Regex:           `(?:\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-Api-Token):[ \t]*([a-z0-9]{40})[\x22\x27](?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[ \t][\x22\x27]?(?i:https://api\.lokalise\.com)/api2/[^\s\x22\x27\x60<>]{0,256}(?:$|[\s\x22\x27\x60])|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://api\.lokalise\.com)/api2/[^\s\x22\x27\x60<>]{0,256}[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-H|--header)[ \t]+[\x22\x27](?i:X-Api-Token):[ \t]*([a-z0-9]{40})[\x22\x27]|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) (?i:https://api\.lokalise\.com)/api2/[^\s\x22\x27\x60<>]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-Api-Token):[ \t]*([a-z0-9]{40})\r?(?:\n|$)|\b(?:GET|POST|PUT|DELETE|PATCH|HEAD) /api2/[^\s]{0,256} HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:Host):[ \t]*(?i:api\.lokalise\.com)\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}(?i:X-Api-Token):[ \t]*([a-z0-9]{40})\r?(?:\n|$))`,
		Keywords:        []string{"api.lokalise.com"},
		Source:          "https://developers.lokalise.com/reference/api-authentication",
		Description:     "Complete Lokalise API request with personal X-Api-Token authentication; the provider says this token must not be put into source control. Bare digests and unbound generic headers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`api\.lokalise\.com`, `/api2/[^?#]*`, []auditedProviderHeader2{{name: "X-Api-Token", pattern: newLazyRegexp(`^[a-z0-9]{40}$`)}}, nil, nil, false)),
	},
	{
		ID:              "jamf-oauth-client-secret",
		Regex:           `(?:\bPOST (?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.jamfcloud\.com)/api/v1/oauth/token HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]+:[^\r\n]{0,96}\r?\n){0,8}\r?\n|\bcurl[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?[\x22\x27]?(?i:https://[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.jamfcloud\.com)/api/v1/oauth/token[\x22\x27]?[ \t]+(?:[^\r\n;&|\x60]|\\\r?\n){0,512}?(?:-d|--data|--data-raw|--data-binary)[ \t]+\x27)((?:[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}&){0,8}client_secret=[A-Za-z0-9_-]{30,256}(?:&[A-Za-z_]+=[A-Za-z0-9%_.:/-]{0,96}){0,8})(?:\r?$|\x27|\r?\n)`,
		Keywords:        []string{"jamfcloud.com"},
		Source:          "https://developer.jamf.com/jamf-pro/reference/postoauthtoken",
		Description:     "Complete Jamf Cloud OAuth token request with client_credentials grant, client ID and write-only client_secret form field. Public client IDs and opaque provider-proximity candidates are excluded. The 30-character lower bound is the pinned scanner subset; 256 is a local cap.",
		Validate:        validProvider4JamfForm,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2(`[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.jamfcloud\.com`, `/api/v1/oauth/token`, nil, []auditedProviderField2{{name: clientSecretField, pattern: newLazyRegexp(`^[A-Za-z0-9_-]{30,256}$`)}, {name: clientIDField, pattern: newLazyRegexp(`^[A-Za-z0-9_-]+$`)}, {name: "grant_type", pattern: newLazyRegexp(`^client_credentials$`)}}, nil, false)),
	},
}

// validateHoneycombCredentialContext rejects fragments of longer credentials.
// A bare issuer-prefixed credential needs no assignment or HTTP-header context.
func validateHoneycombCredentialContext(value string, start, _ int, secret string) contextValidation {
	if start > 0 && (isASCIIWordByte(value[start-1]) || strings.ContainsRune("+/.-\\", rune(value[start-1])) || value[start-1] >= 0x80) {
		return contextValidation{}
	}
	end := start + len(secret)
	if end < len(value) && (value[end] == ':' || value[end] >= 0x80) {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

// HTTP Basic is the only encoding decoded here. These providers put their
// confidential API key in the username and document an empty password.
func validProvider4Basic(s string, size int, alphabet string) bool {
	if len(s) > 6 && strings.EqualFold(s[:5], "Basic") && (s[5] == ' ' || s[5] == '\t') {
		encoded := strings.TrimLeft(s[5:], " \t")
		if len(encoded) > 256 {
			return false
		}
		var decoded [192]byte
		n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(encoded))
		if err != nil {
			return false
		}
		if n == size+1 {
			if decoded[n-1] != ':' {
				return false
			}
		} else if n != size || size != 36 {
			return false
		}
		for i, b := range decoded[:size] {
			if !validProvider4BasicByte(b, i, size, alphabet) {
				return false
			}
		}
		return true
	}
	key := strings.TrimSuffix(s, ":")
	if len(key) != size || !validAuditedCarrierLiteral2(key) {
		return false
	}
	for i := range key {
		if !validProvider4BasicByte(key[i], i, size, alphabet) {
			return false
		}
	}
	return true
}

func validProvider4BasicByte(b byte, i, size int, alphabet string) bool {
	if size == 36 {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			return b == '-'
		}
	}
	return strings.IndexByte(alphabet, b) >= 0
}

func validProvider4BasicHex32(s string) bool {
	return validProvider4Basic(s, 32, "0123456789abcdef")
}

func validProvider4Basic64(s string) bool {
	return validProvider4Basic(s, 64, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
}

func validProvider4Basic20(s string) bool {
	return validProvider4Basic(s, 20, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-")
}

func validProvider4BasicUUID(s string) bool {
	return validProvider4Basic(s, 36, "0123456789abcdef")
}

func validProvider4Basic40(s string) bool {
	return validProvider4Basic(s, 40, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
}

func validProvider4InstamojoForm(s string) bool {
	values, err := url.ParseQuery(s)
	if err != nil || len(values[clientSecretField]) != 1 {
		return false
	}
	secret := values.Get(clientSecretField)
	if len(secret) != 128 || !validAuditedCarrierLiteral2(secret) {
		return false
	}
	for i := range secret {
		b := secret[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') {
			return false
		}
	}
	return true
}

// CLI flags and whole-form captures may have no assignment marker before the
// capture, so check their literal's continuation even when the common guard exits.
func validateProvider4LiteralContext(value string, start, end int, secret string) contextValidation {
	relative := strings.LastIndex(value[start:end], secret)
	if relative < 0 || secret == "" {
		return contextValidation{}
	}
	secretStart := start + relative
	secretEnd := secretStart + len(secret)
	if secretStart > start && (value[secretStart-1] == '\'' || value[secretStart-1] == '"') {
		return contextValidation{accepted: secretEnd < len(value) && value[secretEnd] == value[secretStart-1]}
	}
	for secretEnd < len(value) && (value[secretEnd] == ' ' || value[secretEnd] == '\t') {
		secretEnd++
	}
	return contextValidation{accepted: secretEnd == len(value) || !strings.ContainsRune("([{.$+-*/%?:=!<>|&\\", rune(value[secretEnd]))}
}

func validProvider4IBMForm(s string) bool {
	values, err := url.ParseQuery(s)
	if err != nil || len(values[apiKeyFieldLower]) != 1 || len(values["grant_type"]) != 1 || values.Get("grant_type") != "urn:ibm:params:oauth:grant-type:apikey" {
		return false
	}
	key := values.Get(apiKeyFieldLower)
	if len(key) < 42 || len(key) > 44 || !validAuditedCarrierLiteral2(key) {
		return false
	}
	for i := range key {
		b := key[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '_' && b != '-' {
			return false
		}
	}
	return true
}

func validProvider4JamfForm(s string) bool {
	values, err := url.ParseQuery(s)
	if err != nil || len(values[clientSecretField]) != 1 || len(values[clientIDField]) != 1 || values.Get(clientIDField) == "" || len(values["grant_type"]) != 1 || values.Get("grant_type") != "client_credentials" {
		return false
	}
	secret := values.Get(clientSecretField)
	if len(secret) < 30 || len(secret) > 256 || !validAuditedCarrierLiteral2(secret) {
		return false
	}
	for i := range secret {
		b := secret[i]
		if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && b != '_' && b != '-' {
			return false
		}
	}
	return true
}

// Basic API-user credentials differ from password credentials: validate the
// actual request destination and authentication operand, never a nearby URL.
func provider4BasicRequestContext(host, path string, valid func(string) bool) func(string, int, int, string) contextValidation {
	hostPattern := newLazyRegexp("^(?:" + host + ")$")
	pathPattern := newLazyRegexp("^(?:" + path + ")$")
	return func(value string, start, end int, _ string) contextValidation {
		request, ok := parseAuditedProviderRequestContext2(value, start, end)
		if !ok {
			// A URL-only candidate can also occur inside a curl option. Parse
			// its enclosing command before considering the standalone case.
			raw := value[start:end]
			offset := indexFoldedASCII(raw, "https://")
			if offset < 0 || offset > 1 {
				return contextValidation{}
			}
			uriStart := start + offset
			lineStart := strings.LastIndexByte(value[:uriStart], '\n') + 1
			for lineStart > 0 {
				previousEnd := lineStart - 1
				if previousEnd > 0 && value[previousEnd-1] == '\r' {
					previousEnd--
				}
				if previousEnd == 0 || value[previousEnd-1] != '\\' {
					break
				}
				lineStart = strings.LastIndexByte(value[:previousEnd], '\n') + 1
			}
			commandStart := lineStart
			for commandStart < uriStart && (value[commandStart] == ' ' || value[commandStart] == '\t') {
				commandStart++
			}
			if commandStart < uriStart && value[commandStart] == '$' {
				commandStart++
				for commandStart < uriStart && (value[commandStart] == ' ' || value[commandStart] == '\t') {
					commandStart++
				}
			}
			if strings.HasPrefix(value[commandStart:], "curl ") || strings.HasPrefix(value[commandStart:], "curl\t") {
				request, ok = parseAuditedProviderRequestContext2(value, commandStart, end)
				if !ok {
					return contextValidation{}
				}
			} else {
				if strings.Trim(value[lineStart:uriStart], " \t\"'") != "" {
					return contextValidation{}
				}
				request.url = raw[offset:]
			}
		}
		u, err := url.Parse(request.url)
		if err != nil || !strings.EqualFold(u.Scheme, httpsScheme) && !strings.EqualFold(u.Scheme, httpScheme) || u.Opaque != "" || u.Fragment != "" || u.RawPath != "" || !hostPattern.MatchString(strings.ToLower(u.Host)) || !pathPattern.MatchString(u.Path) {
			return contextValidation{}
		}
		matches, hosts := 0, 0
		if u.User != nil {
			password, _ := u.User.Password()
			if password != "" || !valid(u.User.Username()) {
				return contextValidation{}
			}
			matches++
		}
		if request.user != "" {
			if !valid(request.user) {
				return contextValidation{}
			}
			matches++
		}
		for _, header := range request.headers[:request.headerCount] {
			name, credential, found := strings.Cut(header, ":")
			if !found {
				return contextValidation{}
			}
			credential = strings.TrimSpace(credential)
			if strings.EqualFold(name, "Host") {
				hosts++
				if hosts != 1 || !strings.EqualFold(credential, u.Host) {
					return contextValidation{}
				}
			}
			if strings.EqualFold(name, authorizationHeader) {
				if !valid(credential) {
					return contextValidation{}
				}
				matches++
			}
		}
		return contextValidation{accepted: matches == 1}
	}
}
