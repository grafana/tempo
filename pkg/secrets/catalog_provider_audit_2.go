package secrets

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

// These structural markers cover every request-regex alternative. Provider
// hostnames are checked by the parser, not required by candidate extraction.
var auditedProviderRequestKeywords2 = []string{curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions}

// Carrier constraints adapted from TruffleHog v3.97.4.
// See LICENSE.titus / NOTICE.titus and the repository AGPL license.
var auditedProviderRules2 = []catalogRuleSpec{
	{
		ID:              "buttercms-write-token",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Bb][Uu][Tt][Tt][Ee][Rr]_[Ww][Rr][Ii][Tt][Ee]_[Tt][Oo][Kk][Ee][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{40})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"BUTTER_WRITE_TOKEN"},
		SecretGroup:     1,
		Source:          "https://buttercms.com/docs/api/concepts/authentication-api-tokens",
		Description:     "Documented BUTTER_WRITE_TOKEN assignment; public read tokens and auth_token read URLs are excluded. The 40-character body is the pinned scanner subset, not an issuer guarantee.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "censys-api-secret",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Ee][Nn][Ss][Yy][Ss]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt][\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CENSYS_API_SECRET"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/censys/censys-python/main/censys/search/v1/api.py",
		Description:     "Documented CENSYS_API_SECRET from the official SDK; public CENSYS_API_ID is excluded. The 32-character body is the audited scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "cexio-api-secret",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Ee][Xx][Ii][Oo]_[Ss][Ee][Cc][Rr][Ee][Tt][\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{24,27})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CEXIO_SECRET"},
		SecretGroup:     1,
		Source:          "https://github.com/sdd/cex-as-promised",
		Description:     "CEXIO_SECRET environment variable consumed by the cex-as-promised SDK, carrying the signing secret documented by CEX.IO; public CEXIO_KEY and transmitted signatures are excluded. Body lengths are pinned scanner constraints.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "chartmogul-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Hh][Aa][Rr][Tt][Mm][Oo][Gg][Uu][Ll]_[Aa][Pp][Ii]_[Kk][Ee][Yy][\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CHARTMOGUL_API_KEY"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/chartmogul/chartmogul-node/master/README.md",
		Description:     "CHARTMOGUL_API_KEY environment assignment in the official SDK configuration. Its Basic-auth username role is secret even with an empty password; body width is the pinned scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "checkly-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Hh][Ee][Cc][Kk][Ll][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy][\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CHECKLY_API_KEY"},
		SecretGroup:     1,
		Source:          "https://www.checklyhq.com/docs/cli/authentication.md",
		Description:     "Exact CHECKLY_API_KEY assignment documented for CLI authentication, not the public CHECKLY_ACCOUNT_ID. Historical 32-character scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "cloudsmith-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(csa_[a-f0-9]{30}[A-Za-z0-9]{6})|(?i:CLOUDSMITH_API_KEY)["\']?[ \t]*[:=][ \t]*["\']?([a-f0-9]{40}))` + catalogRightBoundary,
		Keywords:        []string{"CLOUDSMITH_API_KEY", "csa_"},
		SecretGroup:     0,
		Source:          "https://docs.cloudsmith.com/accounts-and-teams/api-key",
		Description:     "Cloudsmith API keys grant confidential read/write access. Preserve the exact CLOUDSMITH_API_KEY 40-hex assignment and its validators; additionally recognize the self-identifying csa_ scanner subset with 30 lowercase hex and six alphanumeric characters. Bare prefixed values do not require assignment context; public repository identifiers are not credentials.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterleaksCloudsmithContext1,
	},
	{
		ID:              "codemagic-api-token",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Mm]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9_]{43})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CM_API_TOKEN"},
		SecretGroup:     1,
		Source:          "https://docs.codemagic.io/rest-api/codemagic-rest-api/",
		Description:     "Exact CM_API_TOKEN assignment recommended in Codemagic REST authentication documentation. Personal token permissions follow the user role; 43 characters is an upstream scanner constraint.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "commercejs-secret-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Hh][Ee][Cc]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Kk][Ee][Yy][\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9_]{48})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CHEC_SECRET_KEY"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/chec/commercejs-nextjs-demo-store/master/.env.example",
		Description:     "CHEC_SECRET_KEY used by the official Commerce.js demo seeder, not NEXT_PUBLIC_CHEC_PUBLIC_KEY. The opaque 48-character historical scanner subset is classified only in the documented secret role.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "crowdin-personal-token",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Cc][Rr][Oo][Ww][Dd][Ii][Nn]_[Pp][Ee][Rr][Ss][Oo][Nn][Aa][Ll]_[Tt][Oo][Kk][Ee][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{80})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"CROWDIN_PERSONAL_TOKEN"},
		SecretGroup:     1,
		Source:          "https://developer.crowdin.com/configuration-file/",
		Description:     "CROWDIN_PERSONAL_TOKEN in the official configuration example. A personal access token grants manager API operations; public project IDs and api_token_env references are excluded. Width is the audited scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "coinapi-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Xx]-[Cc][Oo][Ii][Nn][Aa][Pp][Ii]-[Kk][Ee][Yy][\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Fa-f0-9]{8}(?:-[A-Fa-f0-9]{4}){3}-[A-Fa-f0-9]{12})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"X-CoinAPI-Key"},
		SecretGroup:     1,
		Source:          "https://docs.coinapi.io/products/market-data-api/docs/rest-api",
		Description:     "Exact provider-owned X-CoinAPI-Key header, not standalone UUIDs. UUID shape is the audited scanner subset; authenticity is not verified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "copper-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Xx]-[Pp][Ww]-[Aa][Cc][Cc][Ee][Ss][Ss][Tt][Oo][Kk][Ee][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"X-PW-AccessToken"},
		SecretGroup:     1,
		Source:          "https://developer.copper.com/introduction/requests.html",
		Description:     "Documented Copper X-PW-AccessToken API-key header. The companion user email is public and is not itself a credential; width is the audited scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "campayn-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?(TRUEREST apikey=[a-z0-9]{64})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{authorizationHeader},
		SecretGroup:     1,
		Source:          "https://github.com/campayn/campayn-api",
		Description:     "Complete Campayn TRUEREST apikey authorization header. The provider explicitly says this key exposes reports and lists; 64 characters is the pinned scanner subset.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "cloudelements-credential",
		Regex:           "(?:^|[^A-Za-z0-9_.-])[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn][\"']?[ \\t]*[:=][ \\t]*[\"']?(User [A-Za-z0-9]{43},[ \\t]*Organization [a-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{authorizationHeader},
		SecretGroup:     1,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/cloudelements",
		Description:     "Complete Cloud Elements User and Organization authorization carrier from the pinned request. Neither opaque component is matched standalone.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "calendarific-api-url",
		Regex:           "\\bhttps?://(?:calendarific\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"calendarific.com"},
		Source:          "https://calendarific.com/api-documentation",
		Description:     "Calendarific authenticated API URL; the provider says to keep the quota-spending key private. The pinned 32-character subset is not a standalone key grammar.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("calendarific\\.com", "/api/v2/[^?#]+", []auditedProviderField2{{name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[A-Za-z0-9]{32})$")}}),
	},
	{
		ID:              "checkvist-remote-key-url",
		Regex:           "\\bhttps?://(?:checkvist\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"checkvist.com"},
		Source:          "https://checkvist.com/auth/api",
		Description:     "Checkvist login URL carrying the documented username and remote_key pair. The remote key is a password substitute; public lists and bare email addresses are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("checkvist\\.com", "/auth/login\\.json", []auditedProviderField2{{name: usernameField, pattern: newLazyRegexp("^(?:[^@\\s]+@[^@\\s]+)$")}, {name: "remote_key", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{14})$")}}),
	},
	{
		ID:              "cicero-api-url",
		Regex:           "\\bhttps?://(?:cicero\\.azavea\\.com|app\\.cicerodata\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"cicero.azavea.com", "app.cicerodata.com"},
		Source:          "https://cicero.azavea.com/docs/",
		Description:     "Cicero API-key query credential at documented current or historical API origin. The API key spends account credits; width is the scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("cicero\\.azavea\\.com|app\\.cicerodata\\.com", "/v3\\.1/[^?#]+", []auditedProviderField2{{name: keyField, pattern: newLazyRegexp("^(?:[a-z0-9]{40})$")}}),
	},
	{
		ID:              "cliengo-api-url",
		Regex:           "\\bhttps?://(?:api\\.cliengo\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.cliengo.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/cliengo",
		Description:     "Historical Cliengo authenticated account API URL from the pinned request; the UUID is secret only in this authentication parameter and origin, not as a resource ID.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.cliengo\\.com", "/1\\.0/[^?#]+", []auditedProviderField2{{name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})$")}}),
	},
	{
		ID:              "cloze-api-url",
		Regex:           "\\bhttps?://(?:api\\.cloze\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.cloze.com"},
		Source:          "https://api.cloze.com/api-docs/cloze-openapi.json",
		Description:     "Cloze account API URL carrying the API key and user pair from the pinned request. User email alone and arbitrary 32-hex values are not secrets.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.cloze\\.com", "/v1/[^?#]+", []auditedProviderField2{{name: userField, pattern: newLazyRegexp("^(?:[^@\\s]+@[^@\\s]+)$")}, {name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[a-f0-9]{32})$")}}),
	},
	{
		ID:              "clustdoc-api-url",
		Regex:           "\\bhttps?://(?:app\\.clustdoc\\.com|sandbox\\.clustdoc\\.com|clustdoc\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"app.clustdoc.com", "sandbox.clustdoc.com", "clustdoc.com"},
		Source:          "https://app.clustdoc.com/api-documentation/clustdoc-openapi.yaml",
		Description:     "Documented Clustdoc api_token query-string alternative to Bearer authentication, including sandbox. Body width is the pinned historical subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("app\\.clustdoc\\.com|sandbox\\.clustdoc\\.com|clustdoc\\.com", "/api/[^?#]+", []auditedProviderField2{{name: apiTokenField, pattern: newLazyRegexp("^(?:[A-Za-z0-9]{60})$")}}),
	},
	{
		ID:              "coinlayer-api-url",
		Regex:           "\\bhttps?://(?:api\\.coinlayer\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.coinlayer.com"},
		Source:          "https://coinlayer.com/documentation",
		Description:     "Coinlayer authenticated API URL with the documented access_key role; not generic loose provider proximity. Width is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.coinlayer\\.com", "/(?:api/)?[^?#]+", []auditedProviderField2{{name: accessKeyField, pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}),
	},
	{
		ID:              "coinlib-api-url",
		Regex:           "\\bhttps?://(?:coinlib\\.io)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"coinlib.io"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/coinlib",
		Description:     "Coinlib API URL with the exact key parameter used by the pinned request. The short opaque key is never classified outside the provider authentication URL.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("coinlib\\.io", "/api/v1/[^?#]+", []auditedProviderField2{{name: keyField, pattern: newLazyRegexp("^(?:[a-z0-9]{16})$")}}),
	},
	{
		ID:              "commodities-api-url",
		Regex:           "\\bhttps?://(?:commodities-api\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"commodities-api.com"},
		Source:          "https://commodities-api.com/documentation",
		Description:     "Commodities-API account key in its documented access_key query carrier; width is the scanner subset, not provider issuance grammar.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("commodities-api\\.com", "/api/[^?#]+", []auditedProviderField2{{name: accessKeyField, pattern: newLazyRegexp("^(?:[A-Za-z0-9]{60})$")}}),
	},
	{
		ID:              "convertkit-secret-api-url",
		Regex:           "\\bhttps?://(?:api\\.convertkit\\.com|api\\.kit\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.convertkit.com", "api.kit.com"},
		Source:          "https://developers.kit.com/api-reference/v3/authentication",
		Description:     "Kit/ConvertKit V3 api_secret query carrier; provider documents this as password-equivalent and forbids client-side use. Ordinary api_key and public integration keys are excluded. The opaque literal has the full URI structural cap rather than reusing the public-key detector width.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.convertkit\\.com|api\\.kit\\.com", "/v3/[^?#]+", []auditedProviderField2{{name: "api_secret", pattern: newLazyRegexp("^(?:[A-Za-z0-9_-]+)$")}}),
	},
	{
		ID:              "countrylayer-api-url",
		Regex:           "\\bhttps?://(?:api\\.countrylayer\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.countrylayer.com"},
		Source:          "https://countrylayer.com/documentation/",
		Description:     "CountryLayer API origin and access_key authentication parameter from the pinned request, not arbitrary country IDs. Body width is scanner-only.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.countrylayer\\.com", "/v2/[^?#]+", []auditedProviderField2{{name: accessKeyField, pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}),
	},
	{
		ID:              "cryptocompare-api-url",
		Regex:           "\\bhttps?://(?:min-api\\.cryptocompare\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"min-api.cryptocompare.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/cryptocompare",
		Description:     "CryptoCompare API-key authentication URL from the pinned request. The 64-character value is only classified as the exact api_key query value at this API origin.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("min-api\\.cryptocompare\\.com", "/data/[^?#]+", []auditedProviderField2{{name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[a-z0-9-]{64})$")}}),
	},
	{
		ID:              "currencyfreaks-api-url",
		Regex:           "\\bhttps?://(?:api\\.currencyfreaks\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.currencyfreaks.com"},
		Source:          "https://currencyfreaks.com/documentation.html",
		Description:     "CurrencyFreaks API-key URL; provider explicitly discourages client-side exposure. The pinned 32-character subset is only accepted as apikey at the API origin.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.currencyfreaks\\.com", "/(?:v2\\.0/)?[^?#]+", []auditedProviderField2{{name: apiKeyFieldLower, pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}),
	},
	{
		ID:              "currencylayer-api-url",
		Regex:           "\\bhttps?://(?:api\\.currencylayer\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.currencylayer.com"},
		Source:          "https://currencylayer.com/documentation",
		Description:     "Currencylayer account API URL with access_key authentication, not standalone 32-character values. Width is a pinned scanner constraint.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.currencylayer\\.com", "/[^?#]+", []auditedProviderField2{{name: accessKeyField, pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}),
	},
	{
		ID:              "currencyscoop-api-url",
		Regex:           "\\bhttps?://(?:api\\.currencyscoop\\.com|api\\.currencybeacon\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.currencyscoop.com", "api.currencybeacon.com"},
		Source:          "https://currencybeacon.com/api-documentation",
		Description:     "CurrencyScoop/CurrencyBeacon API key in its documented private query carrier. Both historical and renamed origins are accepted; the width is the pinned subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.currencyscoop\\.com|api\\.currencybeacon\\.com", "/v1/[^?#]+", []auditedProviderField2{{name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}),
	},
	{
		ID:              "customerguru-api-url",
		Regex:           "\\bhttps?://(?:customer\\.guru)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"customer.guru"},
		Source:          "https://customer.guru/api/documentation/v2",
		Description:     "Customer.guru complete export credential URL containing both api_token and api_secret. The roles and example widths are documented; public survey IDs are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("customer\\.guru", "/export/(?:customers|ratings)", []auditedProviderField2{{name: apiTokenField, pattern: newLazyRegexp("^(?:[A-Za-z0-9]{30})$")}, {name: "api_secret", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{50})$")}}),
	},
	{
		ID:              "dandelion-api-url",
		Regex:           "\\bhttps?://(?:api\\.dandelion\\.eu)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.dandelion.eu"},
		Source:          "https://dandelion.eu/docs/api/",
		Description:     "Dandelion authentication URL. The provider specifies a 32-hex profile key in token; arbitrary UUIDs and the legacy public app ID are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.dandelion\\.eu", "/datatxt/[^?#]+", []auditedProviderField2{{name: tokenField, pattern: newLazyRegexp("^(?:[a-f0-9]{32})$")}}),
	},
	{
		ID:              calorieNinjasRuleID,
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://calorieninjas.com/api",
		Description:     "Complete CalorieNinjas HTTP or curl request with its API-key header and exact API origin. The 40-character body is the pinned scanner subset; standalone generic X-Api-Key fields are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.calorieninjas\\.com", "/v1/[^?#]+", []auditedProviderHeader2{{name: apiKeyHeaderTitle, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{40})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              cannyRuleID,
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://developers.canny.io/api-reference",
		Description:     "Canny API request with the documented apiKey form/JSON field at the API endpoint, not a standalone board UUID. The UUID shape is the audited scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("canny\\.io", "/api/v1/[^?#]+", []auditedProviderHeader2{}, []auditedProviderField2{{name: apiKeyFieldCamel, pattern: newLazyRegexp("^(?:[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})$")}}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "captaindata-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://support.captaindata.co/hc/en-us/articles/11250700817181-How-to-Use-the-Captain-Data-API-v3-A-Concise-Guide",
		Description:     "Captain Data v1/v2/v3 request with historical X-Api-Key or current Authorization x-api-key carrier. Provider API origin binds the confidential key role; public project UUID alone is excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.captaindata\\.co|api\\.captaindata\\.com", "/v[123]/[^?#]+", []auditedProviderHeader2{{name: apiKeyHeaderTitle, prefix: "", pattern: newLazyRegexp("^(?:[a-f0-9]{64})$")}, {name: authorizationHeader, prefix: "x-api-key ", pattern: newLazyRegexp("^(?:[a-f0-9]{64})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "caspio-client-secret-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://howto.caspio.com/integrate-your-apps/web-services-api/authenticating-rest/",
		Description:     "Caspio OAuth token request containing literal client_secret and client_id form fields. The identical public client ID is not classified alone. Shapes are the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("[a-z0-9]{8}\\.caspio\\.com", "/oauth/token", []auditedProviderHeader2{}, []auditedProviderField2{{name: clientSecretField, pattern: newLazyRegexp("^(?:[a-z0-9]{50})$")}, {name: clientIDField, pattern: newLazyRegexp("^(?:[a-z0-9]{50})$")}}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "centralstationcrm-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/centralstationcrm",
		Description:     "CentralStationCRM API request with the exact X-apikey carrier used by the pinned authenticated users request. Loose provider words or a 30-character ID are insufficient.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.centralstationcrm\\.net", "/api/[^?#]+", []auditedProviderHeader2{{name: "X-apikey", prefix: "", pattern: newLazyRegexp("^(?:[a-z0-9]{30})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "clockify-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://docs.clockify.me/",
		Description:     "Clockify API origin and X-Api-Key request header together; the 48-character historical body alone is not a credential signature.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.clockify\\.me", "/api/v1/[^?#]+", []auditedProviderHeader2{{name: apiKeyHeaderTitle, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{48})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "textanywhere-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/clockworksms",
		Description:     "Clockwork/TextAnywhere complete request carrying both access_token and user_key headers at the documented scanner API origin. Neither component is detected independently.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.textanywhere\\.com", "/API/v1\\.0/REST/[^?#]+", []auditedProviderHeader2{{name: "access_token", prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{24})$")}}, []auditedProviderField2{}, []auditedProviderField2{{name: "user_key", pattern: newLazyRegexp("^(?:[0-9]{5})$")}}, false)),
	},
	{
		ID:              "cloudimage-invalidation-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://docs.cloudimage.io/caching-and-acceleration/invalidation-api.md",
		Description:     "Cloudimage private invalidation API key carried by X-Client-Key at the invalidation endpoint. Public image-delivery tokens and URLs are not classified.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.cloudimage\\.com", "/invalidate", []auditedProviderHeader2{{name: "X-Client-Key", prefix: "", pattern: newLazyRegexp("^(?:[a-z0-9_]{30})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "cloudmersive-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://api.cloudmersive.com/docs/validate.asp",
		Description:     "Cloudmersive Apikey authentication header inside a request to its exact API origin. Public UUIDs and generic Apikey assignments elsewhere are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.cloudmersive\\.com", "/[^?#]+", []auditedProviderHeader2{{name: "Apikey", prefix: "", pattern: newLazyRegexp("^(?:[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "cloudplan-session-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://docs.cloudplan.net/quickstart.html",
		Description:     "Cloudplan authenticated session_id header paired with the exact API request origin. Ordinary session/resource IDs are not classified standalone; width is scanner-only.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.cloudplan\\.biz", "/api/[^?#]+", []auditedProviderHeader2{{name: "session_id", prefix: "", pattern: newLazyRegexp("^(?:[A-Z0-9-]{32})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "codequiry-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://codequiry.com/usage/api/auth",
		Description:     "Codequiry documented 64-character private apikey header in a complete request to the API origin. Provider says never expose the key in public/client-side code.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("codequiry\\.com", "/api/v1/[^?#]+", []auditedProviderHeader2{{name: apiKeyFieldLower, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9-]{64})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "companyhub-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/companyhub",
		Description:     "CompanyHub complete Authorization account-and-key carrier at its API origin as used by the pinned request; public company IDs alone are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.companyhub\\.com", "/v1/[^?#]+", []auditedProviderHeader2{{name: authorizationHeader, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9$%^=-]{4,32} [A-Za-z0-9]{20})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "convier-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/convier",
		Description:     "Historical Convier complete API Authorization Bearer request, including its pipe-delimited credential shape outside RFC6750 b64token. No bare provider-proximity or invented environment names.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("convier\\.me", "/api/event", []auditedProviderHeader2{{name: authorizationHeader, prefix: bearerPrefix, pattern: newLazyRegexp("^(?:[0-9]{2}\\|[A-Za-z0-9]{40})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "craftmypdf-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://craftmypdf.com/docs/",
		Description:     "CraftMyPDF documented secret X-API-KEY header inside a complete API request. The 35-character body is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.craftmypdf\\.com", "/v1/[^?#]+", []auditedProviderHeader2{{name: apiKeyHeaderUpper, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{35})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "currencycloud-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://developer.currencycloud.com/api-reference/",
		Description:     "Currencycloud login request with literal api_key and login_id fields. The secret key is not confused with a public login ID; key width is the scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("devapi\\.currencycloud\\.com|api\\.currencycloud\\.com", "/v2/authenticate/api", []auditedProviderHeader2{}, []auditedProviderField2{{name: apiKeyFieldSnake, pattern: newLazyRegexp("^(?:[a-z0-9]{64})$")}, {name: "login_id", pattern: newLazyRegexp("^(?:[^@\\s]+@[^@\\s]+)$")}}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "currentsapi-auth-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://currentsapi.services/json/swagger.json",
		Description:     "CurrentsAPI raw Authorization credential inside a complete provider API request; unlike Bearer this carrier has no scheme prefix. Width is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.currentsapi\\.services", "/v1/[^?#]+", []auditedProviderHeader2{{name: authorizationHeader, prefix: "", pattern: newLazyRegexp("^(?:[A-Za-z0-9_-]{48})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "dareboost-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/DareBoost/dareboost-api-client",
		Description:     "DareBoost request with its token in a literal JSON/form body at the API origin. Retired API versions remain confidential; width is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.dareboost\\.com", "/0\\.[0-9]+/[^?#]+", []auditedProviderHeader2{}, []auditedProviderField2{{name: tokenField, pattern: newLazyRegexp("^(?:[A-Za-z0-9]{60})$")}}, []auditedProviderField2{}, false)),
	},
	{
		ID:              "databox-push-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://github.com/databox/databox-python",
		Description:     "Databox Push API token in the HTTP Basic username with an empty password, inside a complete request to the Push API origin. The key is a write credential; width is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("push\\.databox\\.com", "/[^?#]*", []auditedProviderHeader2{{name: authorizationHeader, prefix: "Basic ", pattern: newLazyRegexp("^(?:[A-Za-z0-9]{21})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, true)),
	},
	{
		ID:              "chartmogul-api-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://raw.githubusercontent.com/chartmogul/chartmogul-node/master/README.md",
		Description:     "ChartMogul API key as the HTTP Basic username with an empty password at its API origin; this private carrier is not covered by nonempty-password generic Basic detection.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.chartmogul\\.com", "/v1/[^?#]+", []auditedProviderHeader2{{name: authorizationHeader, prefix: "Basic ", pattern: newLazyRegexp("^(?:[a-z0-9]{32})$")}}, []auditedProviderField2{}, []auditedProviderField2{}, true)),
	},
	{
		ID:              "collect2-datarecord-url",
		Regex:           "\\bhttps?://(?:collect2\\.com)/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"collect2.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/collect2",
		Description:     "Historical Collect2 complete datarecord capability URL used as the sole credential in the pinned request. The UUID is not recognized outside this exact authenticated API path.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("collect2\\.com", "/api/([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})/datarecord/", []auditedProviderField2{}),
	},
	{
		ID:              "capsulecrm-client-secret-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://developer.capsulecrm.com/v2/overview/authentication",
		Description:     "Capsule OAuth client_secret in a complete token-exchange request. Provider requires server-side secrecy; the 50-character lowercase-alphanumeric subset is from its published example, not an issuance guarantee.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.capsulecrm\\.com", "/oauth/token", nil, []auditedProviderField2{{name: clientSecretField, pattern: newLazyRegexp("^[a-z0-9]{50}$")}}, nil, false)),
	},
	{
		ID:              "capsulecrm-refresh-token-request",
		Regex:           auditedProviderRequestPattern2,
		Keywords:        auditedProviderRequestKeywords2,
		Source:          "https://developer.capsulecrm.com/v2/overview/authentication",
		Description:     "Capsule confidential refresh_token in a complete OAuth token-exchange request. The 50-character subset follows the provider example; client IDs and authorization redirect state are excluded.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: withAuditedAssignmentContext(auditedProviderRequestValidator2("api\\.capsulecrm\\.com", "/oauth/token", nil, []auditedProviderField2{{name: "refresh_token", pattern: newLazyRegexp("^[a-z0-9]{50}$")}}, nil, false)),
	},
	{
		ID:              "chatfuel-broadcast-url",
		Regex:           "\\bhttps?://api\\.chatfuel\\.com/[^\\s\"'<>\\x60]*",
		Keywords:        []string{"api.chatfuel.com"},
		Source:          "https://docs.chatfuel.com/en/articles/790461-broadcasting-api",
		Description:     "Chatfuel Broadcasting API URL with its documented chatfuel_token authentication parameter. Bot/user/block IDs are public and insufficient; the 128-character body is the pinned scanner subset.",
		Validate:        validAuditedProviderCandidate2,
		ValidateContext: auditedProviderURLValidator2("api\\.chatfuel\\.com", "/bots/[A-Za-z0-9_-]+/users/[A-Za-z0-9_-]+/send", []auditedProviderField2{{name: "chatfuel_token", pattern: newLazyRegexp("^[A-Za-z0-9]{128}$")}}),
	},
}

// Shapes are pinned scanner subsets unless the fixture evidence says otherwise.
// Parsing is local and bounded; no network verification is performed.
type auditedProviderField2 struct {
	name    string
	pattern *lazyRegexp
}

type auditedProviderHeader2 struct {
	name    string
	prefix  string
	pattern *lazyRegexp
}

func validAuditedProviderCandidate2(s string) bool {
	return s != "" && len(s) <= maxStructuredCredentialBytes
}

func auditedProviderURLValidator2(host, path string, fields []auditedProviderField2) func(string, int, int, string) contextValidation {
	hostPattern := newLazyRegexp("^(?:" + host + ")$")
	pathPattern := newLazyRegexp("^(?:" + path + ")$")
	return func(value string, start, end int, secret string) contextValidation {
		candidate, ok := frameConnectionURI(value, start, end, secret)
		if !ok || !validAuditedProviderCandidate2(candidate) || strings.ContainsAny(candidate, "$\\") {
			return contextValidation{}
		}
		u, ok := auditedProviderURL2(candidate, hostPattern, pathPattern)
		if !ok {
			return contextValidation{}
		}
		if len(fields) == 0 {
			return contextValidation{accepted: u.RawQuery == ""}
		}
		q, err := url.ParseQuery(u.RawQuery)
		return contextValidation{accepted: err == nil && auditedProviderFields2(q, fields)}
	}
}

func auditedProviderURL2(s string, host, path *lazyRegexp) (*url.URL, bool) {
	if !validAuditedProviderCandidate2(s) || strings.ContainsAny(s, "\r\n\x00") {
		return nil, false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != httpsScheme && u.Scheme != httpScheme) || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || strings.Count(u.RawQuery, "&") > 32 {
		return nil, false
	}
	for i := range len(u.Host) {
		if u.Host[i] >= 0x80 {
			return nil, false
		}
	}
	p := u.Path
	if p == "" {
		p = "/"
	}
	return u, host.MatchString(strings.ToLower(u.Host)) && path.MatchString(p)
}

func auditedProviderFields2(values url.Values, fields []auditedProviderField2) bool {
	for _, field := range fields {
		v := values[field.name]
		if len(v) != 1 || !field.pattern.MatchString(v[0]) || !validAuditedCarrierLiteral2(v[0]) {
			return false
		}
	}
	return true
}

// Regex candidates only nominate a command/request start. The context parser
// reads the complete logical command or header block from the original value.
const auditedProviderRequestPattern2 = `\bcurl[ \t]+(?:[^\r\n\\]|\\[^\r\n]|\\\r?\n)+|\b(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)[ \t]+(?:https?://[^\s]+|/[^\s]*)[ \t]*(?:HTTP/1\.[01])?\r?\n[^\r\n]+`

type auditedProviderRequest2 struct {
	url         string
	method      string
	body        string
	user        string
	headers     [32]string
	headerCount int
}

func auditedProviderRequestValidator2(host, path string, headers []auditedProviderHeader2, fields, required []auditedProviderField2, basicUser bool) func(string, int, int, string) contextValidation {
	hostPattern := newLazyRegexp("^(?:" + host + ")$")
	pathPattern := newLazyRegexp("^(?:" + path + ")$")
	return func(value string, start, end int, secret string) contextValidation {
		if start > 0 && strings.ContainsRune(".$/-", rune(value[start-1])) || !validAuditedProviderCandidate2(secret) {
			return contextValidation{}
		}
		r, ok := parseAuditedProviderRequestContext2(value, start, end)
		if !ok {
			return contextValidation{}
		}
		u, ok := auditedProviderURL2(r.url, hostPattern, pathPattern)
		if !ok {
			return contextValidation{}
		}
		hostHeaders := 0
		for _, line := range r.headers[:r.headerCount] {
			name, host, _ := strings.Cut(line, ":")
			if strings.EqualFold(name, "Host") {
				hostHeaders++
				if hostHeaders > 1 || !strings.EqualFold(strings.TrimSpace(host), u.Host) {
					return contextValidation{}
				}
			}
		}
		for _, field := range required {
			matches := 0
			for _, line := range r.headers[:r.headerCount] {
				name, credential, _ := strings.Cut(line, ":")
				if strings.EqualFold(name, field.name) && field.pattern.MatchString(strings.TrimSpace(credential)) {
					matches++
				}
			}
			if matches != 1 {
				return contextValidation{}
			}
		}
		if len(fields) > 0 {
			return contextValidation{accepted: r.method == httpMethodPost && auditedProviderBody2(r.body, fields)}
		}
		matches := 0
		for _, auth := range headers {
			if basicUser && r.user != "" && auditedProviderBasicUser2(r.user, auth.pattern) {
				matches++
			}
			for _, line := range r.headers[:r.headerCount] {
				name, credential, found := strings.Cut(line, ":")
				if !found || !strings.EqualFold(name, auth.name) {
					continue
				}
				credential = strings.TrimSpace(credential)
				if len(credential) < len(auth.prefix) || !strings.EqualFold(credential[:len(auth.prefix)], auth.prefix) {
					return contextValidation{}
				}
				credential = credential[len(auth.prefix):]
				if basicUser {
					decoded, err := base64.StdEncoding.Strict().DecodeString(credential)
					if err != nil || !auditedProviderBasicUser2(string(decoded), auth.pattern) {
						return contextValidation{}
					}
				} else if !auth.pattern.MatchString(credential) || !validAuditedCarrierLiteral2(credential) || !validateAuditedAssignmentContext(line, 0, len(line), credential).accepted {
					return contextValidation{}
				}
				matches++
			}
		}
		return contextValidation{accepted: matches == 1}
	}
}

func auditedProviderBasicUser2(s string, pattern *lazyRegexp) bool {
	user, password, found := strings.Cut(s, ":")
	return found && password == "" && pattern.MatchString(user) && validAuditedCarrierLiteral2(user)
}

// Regex matches only nominate a start. In particular, a matched URL may be a
// referer, or the match may stop before later options change the destination.
// Parse the entire bounded command/request from the original value instead.
func parseAuditedProviderRequestContext2(value string, start, end int) (auditedProviderRequest2, bool) {
	var empty auditedProviderRequest2
	if start < 0 || end <= start || end > len(value) {
		return empty, false
	}
	isCurl := strings.HasPrefix(value[start:], "curl ") || strings.HasPrefix(value[start:], "curl\t")
	if isCurl {
		lineStart := strings.LastIndexByte(value[:start], '\n') + 1
		prefix := strings.TrimSpace(value[lineStart:start])
		if prefix != "" && prefix != "$" {
			return empty, false
		}
		var quote byte
		commandEnd := len(value)
		for i := start; i < len(value); i++ {
			c := value[i]
			if quote == 0 && (c == '\n' || c == '\r' && i+1 < len(value) && value[i+1] == '\n') {
				commandEnd = i
				break
			}
			if i-start >= maxStructuredCredentialBytes {
				return empty, false
			}
			if c == '\\' && quote != '\'' && i+1 < len(value) {
				i++
				if value[i] == '\r' && i+1 < len(value) && value[i+1] == '\n' {
					i++
				}
				continue
			}
			if c == '\'' || c == '"' {
				switch quote {
				case 0:
					quote = c
				case c:
					quote = 0
				}
			}
		}
		if quote != 0 || commandEnd-start > maxStructuredCredentialBytes {
			return empty, false
		}
		return parseAuditedProviderRequest2(value[start:commandEnd])
	}
	// HTTP request lines start at the beginning of a physical line, not after
	// an X-Note/header prefix or inside an indented continuation field.
	if start > 0 && value[start-1] != '\n' {
		return empty, false
	}
	limit := min(len(value), start+maxStructuredCredentialBytes+1)
	candidate := value[start:limit]
	firstEnd := strings.IndexByte(candidate, '\n')
	if firstEnd < 0 {
		return empty, false
	}
	if _, _, ok := auditedProviderRequestLine2(strings.TrimSuffix(candidate[:firstEnd], "\r")); !ok {
		return empty, false
	}
	contentLength := -1
	transferEncoded := false
	for pos := firstEnd + 1; pos < len(candidate); {
		lineEnd := strings.IndexByte(candidate[pos:], '\n')
		next := len(candidate)
		if lineEnd < 0 {
			if limit != len(value) {
				return empty, false
			}
			lineEnd = len(candidate)
		} else {
			lineEnd += pos
			next = lineEnd + 1
		}
		line := strings.TrimSuffix(candidate[pos:lineEnd], "\r")
		if line == "" {
			requestEnd := len(value)
			if contentLength >= 0 {
				requestEnd = start + next + contentLength
			}
			if transferEncoded {
				// Header credentials remain literal even when the body uses a
				// transfer coding. Do not expose undecoded chunks as form/JSON.
				requestEnd = start + next
			}
			if requestEnd > len(value) || requestEnd-start > maxStructuredCredentialBytes {
				return empty, false
			}
			return parseAuditedProviderRequest2(value[start:requestEnd])
		}
		name, field, found := strings.Cut(line, ":")
		if !found || !auditedProviderHTTPFieldName2(name) || strings.ContainsAny(field, "\r\x00") {
			return empty, false
		}
		if strings.EqualFold(name, "Transfer-Encoding") {
			transferEncoded = true
		}
		if strings.EqualFold(name, "Content-Length") {
			field = strings.TrimSpace(field)
			if contentLength >= 0 || field == "" || len(field) > 5 {
				return empty, false
			}
			contentLength = 0
			for i := range len(field) {
				if field[i] < '0' || field[i] > '9' {
					return empty, false
				}
				contentLength = contentLength*10 + int(field[i]-'0')
				if contentLength > maxStructuredCredentialBytes {
					return empty, false
				}
			}
		}
		pos = next
	}
	// Request logs commonly end after the last complete header field, with
	// neither an empty separator line nor a body.
	if limit != len(value) || len(candidate) > maxStructuredCredentialBytes {
		return empty, false
	}
	return parseAuditedProviderRequest2(candidate)
}

func auditedProviderHTTPFieldName2(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			continue
		}
		return false
	}
	return true
}

func auditedProviderRequestLine2(line string) (string, string, bool) {
	parts := strings.Fields(line)
	switch len(parts) {
	case 2:
		// Logging may omit the protocol version, but must retain an absolute
		// destination URL; arbitrary "GET path" prose is not a request line.
		if !strings.HasPrefix(parts[1], "https://") && !strings.HasPrefix(parts[1], "http://") {
			return "", "", false
		}
	case 3:
		if parts[2] != "HTTP/1.0" && parts[2] != "HTTP/1.1" {
			return "", "", false
		}
	default:
		return "", "", false
	}
	switch parts[0] {
	case httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions:
		return parts[0], parts[1], true
	}
	return "", "", false
}

// Preserve quoted literals and shell escapes, but reject unquoted source-call,
// index and object operators which are not literal shell words.
func nextAuditedProviderCurlWord2(s string) (string, string, bool) {
	word, rest, ok := nextCurlLiteralWord1(s)
	if !ok {
		return "", "", false
	}
	var quote byte
	for i := 0; i < len(s)-len(rest); i++ {
		c := s[i]
		if c == '\\' && quote != '\'' {
			i++
			continue
		}
		if c == '\'' || c == '"' {
			switch quote {
			case 0:
				quote = c
			case c:
				quote = 0
			}
		} else if quote == 0 && strings.ContainsRune("()[]{}", rune(c)) {
			return "", "", false
		}
	}
	return word, rest, true
}

func parseAuditedProviderRequest2(s string) (auditedProviderRequest2, bool) {
	var r auditedProviderRequest2
	if !validAuditedProviderCandidate2(s) {
		return r, false
	}
	if strings.HasPrefix(s, "curl ") || strings.HasPrefix(s, "curl\t") {
		s = providerAudit3CurlLines(s)
		_, rest, ok := nextAuditedProviderCurlWord2(s)
		if !ok {
			return r, false
		}
		r.method = httpMethodGet
		explicitMethod, positional := false, false
		for range 64 {
			if strings.TrimSpace(rest) == "" {
				return r, r.url != ""
			}
			var word string
			word, rest, ok = nextAuditedProviderCurlWord2(rest)
			if !ok {
				return r, false
			}
			if word == "--" && !positional {
				positional = true
				continue
			}
			option, argument := word, ""
			hasArgument := false
			switch {
			case strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://"):
				option, argument, hasArgument = curlURLOption, word, true
			case positional:
				return r, false
			case strings.HasPrefix(word, "--"):
				option, argument, hasArgument = strings.Cut(word, "=")
			case len(word) > 2:
				switch word[:2] {
				case "-H", "-d", "-u", "-X", "-e", "-A", "-o", "-x", "-b", "-c", "-F":
					option, argument, hasArgument = word[:2], word[2:], true
				}
			}
			switch option {
			case "-s", "-S", curlSilentShowErrorOption, "-sSL", "-fsS", "-fsSL", "-L", "-f", "-i", "-k", "-v", "-g",
				curlSilentOption, curlShowErrorOption, curlLocationOption, curlFailOption, curlInsecureOption, "--verbose",
				curlCompressedOption, "--globoff", "--http1.1", "--http2", "--fail-with-body":
				if hasArgument {
					return r, false
				}
				continue
			case "-I", "--head", "-G", curlGetOption:
				if hasArgument || explicitMethod {
					return r, false
				}
				r.method, explicitMethod = httpMethodGet, true
				if option == "-I" || option == "--head" {
					r.method = httpMethodHead
				}
				continue
			case "-H", curlHeaderOption, "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, curlDataURLEncodeOption,
				"-u", curlUserOption, curlURLOption, "-X", curlRequestOption, "-e", "--referer", "-A", "--user-agent",
				"-o", "--output", "-x", "--proxy", "-b", "--cookie", "-c", "--cookie-jar", "-F", curlFormOption,
				"--connect-timeout", "--max-time", "--cacert", "--cert", "--key", "--retry", "--retry-delay":
				if !hasArgument {
					argument, rest, ok = nextAuditedProviderCurlWord2(rest)
					if !ok {
						return r, false
					}
				}
			default:
				return r, false
			}
			switch option {
			case "-H", curlHeaderOption:
				name, _, found := strings.Cut(argument, ":")
				if r.headerCount == len(r.headers) || strings.ContainsAny(argument, "\r\n\x00") || !found || !auditedProviderHTTPFieldName2(name) {
					return r, false
				}
				r.headers[r.headerCount] = argument
				r.headerCount++
			case "-u", curlUserOption:
				if r.user != "" {
					return r, false
				}
				r.user = argument
			case curlURLOption:
				if r.url != "" {
					return r, false
				}
				r.url = argument
			case "-X", curlRequestOption:
				if explicitMethod {
					return r, false
				}
				r.method, explicitMethod = argument, true
			case "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, curlDataURLEncodeOption:
				if strings.HasPrefix(argument, "@") {
					return r, false
				}
				if r.body != "" {
					r.body += "&"
				}
				r.body += argument
				if !explicitMethod {
					r.method = httpMethodPost
				}
				// Known non-destination operands are deliberately consumed above.
				// In particular, referer URLs must never become request URLs.
			}
		}
		return r, false
	}
	if strings.Contains(s, "\r\n") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	line, rest, found := strings.Cut(s, "\n")
	method, target, ok := auditedProviderRequestLine2(line)
	if !found || !ok {
		return r, false
	}
	r.method, r.url = method, target
	var host string
	for rest != "" {
		line, rest, _ = strings.Cut(rest, "\n")
		if line == "" {
			r.body = rest
			break
		}
		name, header, found := strings.Cut(line, ":")
		if !found || name == "" || strings.ContainsAny(name, " \t") || r.headerCount == len(r.headers) {
			return r, false
		}
		if strings.EqualFold(name, "Host") {
			if host != "" {
				return r, false
			}
			host = strings.TrimSpace(header)
		}
		r.headers[r.headerCount] = line
		r.headerCount++
	}
	if strings.HasPrefix(r.url, "/") {
		if host == "" {
			return r, false
		}
		r.url = "https://" + host + r.url
	} else if host != "" {
		u, err := url.Parse(r.url)
		if err != nil || !strings.EqualFold(u.Host, host) {
			return r, false
		}
	}
	return r, true
}

func auditedProviderBody2(s string, fields []auditedProviderField2) bool {
	if strings.HasPrefix(strings.TrimSpace(s), "{") {
		decoder := json.NewDecoder(strings.NewReader(s))
		opening, err := decoder.Token()
		if err != nil || opening != json.Delim('{') {
			return false
		}
		values := make(url.Values, len(fields))
		for count := 0; decoder.More(); count++ {
			if count == 64 {
				return false
			}
			key, err := decoder.Token()
			if err != nil {
				return false
			}
			var raw json.RawMessage
			if decoder.Decode(&raw) != nil {
				return false
			}
			for _, field := range fields {
				if key == field.name {
					var credential string
					if json.Unmarshal(raw, &credential) != nil {
						return false
					}
					values[field.name] = append(values[field.name], credential)
				}
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return false
		}
		var trailing json.RawMessage
		if decoder.Decode(&trailing) != io.EOF {
			return false
		}
		return auditedProviderFields2(values, fields)
	}
	if strings.Count(s, "&") > 32 {
		return false
	}
	values, err := url.ParseQuery(s)
	return err == nil && auditedProviderFields2(values, fields)
}
