package secrets

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Provider carriers audited from Titus v1.2.9 and TruffleHog v3.97.4.
// See LICENSE.titus / NOTICE.titus and the repository AGPL license. Provider
// roles and scanner-only bounds are attributed separately in the fixtures.
var auditedProviderRules5 = []catalogRuleSpec{
	{
		ID:              "mistral-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Mm][Ii][Ss][Tt][Rr][Aa][Ll]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|\'(?:[Mm][Ii][Ss][Tt][Rr][Aa][Ll]_[Aa][Pp][Ii]_[Kk][Ee][Yy])\'|(?:[Mm][Ii][Ss][Tt][Rr][Aa][Ll]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{32})"|\'([A-Za-z0-9]{32})\'|([A-Za-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"MISTRAL_API_KEY"},
		Source:          "https://raw.githubusercontent.com/mistralai/client-python/main/README.md",
		Description:     "Documented MISTRAL_API_KEY confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "moralis-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Mm][Oo][Rr][Aa][Ll][Ii][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|\'(?:[Mm][Oo][Rr][Aa][Ll][Ii][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy])\'|(?:[Mm][Oo][Rr][Aa][Ll][Ii][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{64})"|\'([A-Za-z0-9]{64})\'|([A-Za-z0-9]{64}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"MORALIS_API_KEY"},
		Source:          "https://docs.moralis.io/web3-data-api/security-guidelines",
		Description:     "Documented MORALIS_API_KEY confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "mockaroo-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Mm][Oo][Cc][Kk][Aa][Rr][Oo][Oo]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|\'(?:[Mm][Oo][Cc][Kk][Aa][Rr][Oo][Oo]_[Aa][Pp][Ii]_[Kk][Ee][Yy])\'|(?:[Mm][Oo][Cc][Kk][Aa][Rr][Oo][Oo]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([a-z0-9]{8})"|\'([a-z0-9]{8})\'|([a-z0-9]{8}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"MOCKAROO_API_KEY"},
		Source:          mockarooAPISource,
		Description:     "Documented MOCKAROO_API_KEY confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "plaid-secret-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Pp][Ll][Aa][Ii][Dd]_[Ss][Ee][Cc][Rr][Ee][Tt])"|\'(?:[Pp][Ll][Aa][Ii][Dd]_[Ss][Ee][Cc][Rr][Ee][Tt])\'|(?:[Pp][Ll][Aa][Ii][Dd]_[Ss][Ee][Cc][Rr][Ee][Tt]))[ \t]*[:=][ \t]*(?:"([a-z0-9]{30})"|\'([a-z0-9]{30})\'|([a-z0-9]{30}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"PLAID_SECRET"},
		Source:          "https://raw.githubusercontent.com/plaid/quickstart/master/.env.example",
		Description:     "Documented PLAID_SECRET confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pastebin-user-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Aa][Pp][Ii]_[Uu][Ss][Ee][Rr]_[Kk][Ee][Yy])"|\'(?:[Aa][Pp][Ii]_[Uu][Ss][Ee][Rr]_[Kk][Ee][Yy])\'|(?:[Aa][Pp][Ii]_[Uu][Ss][Ee][Rr]_[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_]{32})"|\'([A-Za-z0-9_]{32})\'|([A-Za-z0-9_]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"api_user_key"},
		Source:          "https://pastebin.com/doc_api",
		Description:     "Documented api_user_key confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pendo-integration-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Xx]\-[Pp][Ee][Nn][Dd][Oo]\-[Ii][Nn][Tt][Ee][Gg][Rr][Aa][Tt][Ii][Oo][Nn]\-[Kk][Ee][Yy])"|\'(?:[Xx]\-[Pp][Ee][Nn][Dd][Oo]\-[Ii][Nn][Tt][Ee][Gg][Rr][Aa][Tt][Ii][Oo][Nn]\-[Kk][Ee][Yy])\'|(?:[Xx]\-[Pp][Ee][Nn][Dd][Oo]\-[Ii][Nn][Tt][Ee][Gg][Rr][Aa][Tt][Ii][Oo][Nn]\-[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{32,64})"|\'([A-Za-z0-9]{32,64})\'|([A-Za-z0-9]{32,64}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"x-pendo-integration-key"},
		Source:          "https://engageapi.pendo.io/",
		Description:     "Documented x-pendo-integration-key confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pivotaltracker-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Xx]\-[Tt][Rr][Aa][Cc][Kk][Ee][Rr][Tt][Oo][Kk][Ee][Nn])"|\'(?:[Xx]\-[Tt][Rr][Aa][Cc][Kk][Ee][Rr][Tt][Oo][Kk][Ee][Nn])\'|(?:[Xx]\-[Tt][Rr][Aa][Cc][Kk][Ee][Rr][Tt][Oo][Kk][Ee][Nn]))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|\'([a-z0-9]{32})\'|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"X-TrackerToken"},
		Source:          "https://www.pivotaltracker.com/help/api/rest/v5",
		Description:     "Documented X-TrackerToken confidential credential carrier; bare opaque values and unrelated public identifiers are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "marketstack-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.marketstack\.com/v[12]/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}access_key=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.marketstack.com", accessKeyField},
		Source:          "https://marketstack.com/documentation",
		Description:     "Complete documented marketstack API URL with its access_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "mediastack-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.mediastack\.com/v1/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}access_key=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.mediastack.com", accessKeyField},
		Source:          "https://mediastack.com/documentation",
		Description:     "Complete documented mediastack API URL with its access_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "mockaroo-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.mockaroo\.com/api/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}key=([a-z0-9]{8})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.mockaroo.com", keyField},
		Source:          mockarooAPISource,
		Description:     "Complete documented mockaroo API URL with its key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "moosend-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.moosend\.com/v3/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}apikey=([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.moosend.com", apiKeyFieldLower},
		Source:          "https://docs.moosend.com/",
		Description:     "Complete documented moosend API URL with its apikey authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "newsapi-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://newsapi\.org/v2/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}apiKey=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"newsapi.org", apiKeyFieldCamel},
		Source:          "https://newsapi.org/docs/authentication",
		Description:     "Complete documented newsapi API URL with its apiKey authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "numverify-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://apilayer\.net/api/validate\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}access_key=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{apilayerHost, accessKeyField},
		Source:          "https://numverify.com/documentation",
		Description:     "Complete documented numverify API URL with its access_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "nytimes-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.nytimes\.com/svc/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}api\-key=([A-Za-z0-9_=-]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.nytimes.com", apiKeyFieldHyphen},
		Source:          "https://developer.nytimes.com/get-started",
		Description:     "Complete documented nytimes API URL with its api-key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "opencage-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.opencagedata\.com/geocode/v1/(?:json|xml)\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}key=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.opencagedata.com", keyField},
		Source:          "https://opencagedata.com/api",
		Description:     "Complete documented opencagedata API URL with its key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "openweather-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://(?:api|pro|history)\.openweathermap\.org/(?:data|geo)/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}appid=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.openweathermap.org", "appid"},
		Source:          "https://openweathermap.org/faq",
		Description:     "Complete documented openweather API URL with its appid authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "pandascore-api-token-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.pandascore\.co/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}token=([A-Za-z0-9_-]{51})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.pandascore.co", tokenField},
		Source:          "https://developers.pandascore.co/docs/getting-started",
		Description:     "Complete documented pandascore API URL with its token authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "parsehub-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://(?:www\.)?parsehub\.com/api/v2/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}api_key=([A-Za-z0-9]{12})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"www.parsehub.com", apiKeyFieldSnake},
		Source:          "https://www.parsehub.com/docs/ref/api/v2/",
		Description:     "Complete documented parsehub API URL with its api_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "paydirt-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://paydirtapp\.com/api/v1/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}api_key=([a-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"paydirtapp.com", apiKeyFieldSnake},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/paydirtapp",
		Description:     "Complete documented paydirt API URL with its api_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "pipedrive-api-token-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}api_token=([A-Za-z0-9]{40})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"pipedrive.com", apiTokenField},
		Source:          "https://pipedrive.readme.io/docs/core-api-concepts-authentication",
		Description:     "Complete documented pipedrive API URL with its api_token authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "planyo-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://(?:www\.)?planyo\.com/rest/\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}api_key=([a-z0-9]{62})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"www.planyo.com", apiKeyFieldSnake},
		Source:          "https://www.planyo.com/api.php",
		Description:     "Complete documented planyo API URL with its api_key authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "polygon-api-key-url",
		Regex:           `\b[Hh][Tt][Tt][Pp][Ss]?://api\.(?:polygon\.io|massive\.com)/v[123]/[A-Za-z0-9_/.-]{1,128}\?(?:[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}&){0,6}apiKey=([A-Za-z0-9]{32})(?:&[A-Za-z0-9_~-]{1,64}=[A-Za-z0-9%._~,+:/=-]{0,128}){0,6}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{"api.polygon.io", apiKeyFieldCamel},
		Source:          "https://raw.githubusercontent.com/polygon-io/client-python/master/README.md",
		Description:     "Complete documented polygonio API URL with its apiKey authentication query parameter. The exact service authority and API path distinguish the credential from opaque IDs; candidate alphabet/width follow the pinned scanner, and surrounding query fields are bounded locally.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "mesibo-backend-token",
		Regex:           `(POST[ \t]+https://api\.mesibo\.com/backend[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}\r?\n[^\r\n]{2,1000})(?:$|\r?\n)`,
		Keywords:        []string{"api.mesibo.com"},
		Source:          "https://docs.mesibo.com/api/backend-api/",
		Description:     "Complete mesibo POST request carrying its documented private authentication fields; exact provider endpoint plus parsed body establishes the role, without loose brand proximity.",
		ValidateContext: provider5BodyContext("api\\.mesibo\\.com", "/backend", validProvider5MesiboBody),
	},
	{
		ID:              "meaningcloud-license-key",
		Regex:           `(POST[ \t]+https://api\.meaningcloud\.com/[A-Za-z0-9_./-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}\r?\n[^\r\n]{2,1000}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\')[ \t]+https://api\.meaningcloud\.com/[A-Za-z0-9_./-]{1,128}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?https://api\.meaningcloud\.com/[A-Za-z0-9_./-]{1,128}[ \t]+(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\'))(?:$|\r?\n)`,
		Keywords:        []string{"api.meaningcloud.com"},
		Source:          "https://raw.githubusercontent.com/MeaningCloud/meaningcloud-python/master/meaningcloud/Request.py",
		Description:     "Complete meaningcloud POST request carrying its documented private authentication fields; exact provider endpoint plus parsed body establishes the role, without loose brand proximity.",
		ValidateContext: provider5BodyContext("api\\.meaningcloud\\.com", "/[A-Za-z0-9_./-]{1,128}", validProvider5MeaningCloudBody),
	},
	{
		ID:              "paralleldots-api-key",
		Regex:           `(POST[ \t]+https://apis\.paralleldots\.com/v[34]/[A-Za-z0-9_/-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}\r?\n[^\r\n]{2,1000}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\')[ \t]+https://apis\.paralleldots\.com/v[34]/[A-Za-z0-9_/-]{1,128}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?https://apis\.paralleldots\.com/v[34]/[A-Za-z0-9_/-]{1,128}[ \t]+(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\'))(?:$|\r?\n)`,
		Keywords:        []string{"apis.paralleldots.com"},
		Source:          "https://raw.githubusercontent.com/ParallelDots/ParallelDots-Python-API/master/README.md",
		Description:     "Complete paralleldots POST request carrying its documented private authentication fields; exact provider endpoint plus parsed body establishes the role, without loose brand proximity.",
		ValidateContext: provider5BodyContext("apis\\.paralleldots\\.com", "/v[34]/[A-Za-z0-9_/-]{1,128}", validProvider5ParallelDotsBody),
	},
	{
		ID:              "mrticktock-login-credentials",
		Regex:           `(POST[ \t]+https://mrticktock\.com/app/api/[a-z_]{1,64}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}\r?\n[^\r\n]{2,1000}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\')[ \t]+https://mrticktock\.com/app/api/[a-z_]{1,64}|curl[ \t]+(?:-X[ \t]+POST[ \t]+)?https://mrticktock\.com/app/api/[a-z_]{1,64}[ \t]+(?:-d|--data)[ \t]+(?:"[^"\r\n]{2,1000}"|\'[^\'\r\n]{2,1000}\'))(?:$|\r?\n)`,
		Keywords:        []string{"mrticktock.com"},
		Source:          "https://mrticktock.com/api.html",
		Description:     "Complete mrticktock POST request carrying its documented private authentication fields; exact provider endpoint plus parsed body establishes the role, without loose brand proximity.",
		ValidateContext: provider5BodyContext("mrticktock\\.com", "/app/api/[a-z_]{1,64}", validProvider5MrTickTockBody),
	},
	{
		ID:              "onedesk-login-credentials",
		Regex:           `(POST[ \t]+https://app\.onedesk\.com/rest/2\.0/login/loginUser[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}\r?\n[^\r\n]{2,1000})(?:$|\r?\n)`,
		Keywords:        []string{"app.onedesk.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/onedesk",
		Description:     "Complete onedesk POST request carrying the documented private email/password pair. Native parsing requires a literal structured body and exact service endpoint; bare values, public fields, source expressions and interpolation are not credentials. The historical OneDesk login carrier is retained independently of newer API-key authentication.",
		ValidateContext: provider5BodyContext("app\\.onedesk\\.com", "/rest/2\\.0/login/loginUser", validProvider5OneDeskBody),
	},
	{
		ID:              "onedesk-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Oo][Dd]\-[Pp][Uu][Bb][Ll][Ii][Cc]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy])"|\'(?:[Oo][Dd]\-[Pp][Uu][Bb][Ll][Ii][Cc]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy])\'|(?:[Oo][Dd]\-[Pp][Uu][Bb][Ll][Ii][Cc]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]))[ \t]*:[ \t]*(?:"([A-Za-z0-9_-]{8,256})"|\'([A-Za-z0-9_-]{8,256})\'|([A-Za-z0-9_-]{8,256}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"od-public-api-key"},
		Source:          "https://kb.onedesk.com/app-od/o-onedesk/knowledge-base-1/article-1bebcbd3-e96f-4fe6-8363-aac06cbc6361",
		Description:     "Documented OD-Public-API-Key authentication header for the OneDesk public API. Public names the API surface, not a publishable credential.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "onfleet-api-key-basic-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*onfleet\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*onfleet\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{onfleetHost},
		Source:          onfleetAuthSource,
		Description:     providerBasicDummyPasswordDescription,
		ValidateContext: provider5HeaderContext("onfleet\\.com", "/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Basic", "[A-Za-z0-9+/]{12,800}={0,2}", validProvider5OnfleetBasic),
	},
	{
		ID:              "onfleet-api-key-userinfo-url",
		Regex:           `\bhttps://([a-z0-9]{32}):@onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{onfleetHost},
		Source:          onfleetAuthSource,
		Description:     providerAPIKeyUserinfoDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "packagecloud-api-key-basic-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*packagecloud\.io\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*packagecloud\.io(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{packagecloudHost},
		Source:          packagecloudAPISource,
		Description:     providerBasicDummyPasswordDescription,
		ValidateContext: provider5HeaderContext("packagecloud\\.io", "/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Basic", "[A-Za-z0-9+/]{12,800}={0,2}", validProvider5PackageCloudBasic),
	},
	{
		ID:              "packagecloud-api-key-userinfo-url",
		Regex:           `\bhttps://([a-f0-9]{48}):@packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{packagecloudHost},
		Source:          packagecloudAPISource,
		Description:     providerAPIKeyUserinfoDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "myintervals-api-key-basic-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.myintervals\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.myintervals\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{myIntervalsHost},
		Source:          myIntervalsAuthSource,
		Description:     providerBasicDummyPasswordDescription,
		ValidateContext: provider5HeaderContext("api\\.myintervals\\.com", "/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Basic", "[A-Za-z0-9+/]{12,800}={0,2}", validProvider5MyIntervalsBasic),
	},
	{
		ID:              "myintervals-api-key-userinfo-url",
		Regex:           `\bhttps://([A-Za-z0-9]{11}):X@api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{myIntervalsHost},
		Source:          myIntervalsAuthSource,
		Description:     "Exact service API URL with the documented confidential API-key username and dummy X password. Public user-only URLs are excluded; the bounded key subset is provider-scoped rather than an arbitrary username heuristic.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "paymo-api-key-basic-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*app\.paymoapp\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*app\.paymoapp\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{paymoHost},
		Source:          paymoAuthSource,
		Description:     providerBasicDummyPasswordDescription,
		ValidateContext: provider5HeaderContext("app\\.paymoapp\\.com", "/api/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Basic", "[A-Za-z0-9+/]{12,800}={0,2}", validProvider5PaymoBasic),
	},
	{
		ID:              "paymo-api-key-userinfo-url",
		Regex:           `\bhttps://([A-Za-z0-9]{8,256}):X@app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{paymoHost},
		Source:          paymoAuthSource,
		Description:     "Exact service API URL with the documented confidential API-key username and dummy X password. Public user-only URLs are excluded; the bounded key subset is provider-scoped rather than an arbitrary username heuristic.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "paymongo-api-key-basic-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.paymongo\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.paymongo\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]{12,800}={0,2})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{paymongoHost},
		Source:          paymongoAuthSource,
		Description:     providerBasicDummyPasswordDescription,
		ValidateContext: provider5HeaderContext("api\\.paymongo\\.com", "/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Basic", "[A-Za-z0-9+/]{12,800}={0,2}", validProvider5PaymongoBasic),
	},
	{
		ID:              "paymongo-api-key-userinfo-url",
		Regex:           `\bhttps://(sk_(?:test|live)_[A-Za-z0-9]{24}):@api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:        []string{paymongoHost},
		Source:          paymongoAuthSource,
		Description:     providerAPIKeyUserinfoDescription,
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateProvider5URLContext,
	},
	{
		ID:              "pepipost-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pepipost\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pepipost\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})"|\'(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})"|\'(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{32})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pepipost\.com/v5(?:\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.pepipost.com"},
		Source:          "https://raw.githubusercontent.com/pepipost/pepipost-sdk-python/master/README.md",
		Description:     "Complete official pepipost API request with its private api_key header. The generic field name alone and public API identifiers do not match.",
		ValidateContext: provider5HeaderContext("api\\.pepipost\\.com", "/v5(?:\\.1)?/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyFieldSnake, "", "", "[A-Za-z0-9-]{32}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "pollsapi-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pollsapi\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pollsapi\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})"|\'(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})"|\'(?:[Aa][Pp][Ii]_[Kk][Ee][Yy]|[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Z0-9]{28})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pollsapi\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.pollsapi.com"},
		Source:          "https://docs.pollsapi.com/",
		Description:     "Complete official pollsapi API request with its private api_key header. The generic field name alone and public API identifiers do not match.",
		ValidateContext: provider5HeaderContext("api\\.pollsapi\\.com", "/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyFieldSnake, apiKeyFieldHyphen, "", "[A-Z0-9]{28}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "newscatcher-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:"(?:[Nn][Ee][Ww][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy])"|\'(?:[Nn][Ee][Ww][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy])\'|(?:[Nn][Ee][Ww][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy]))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_]{43})"|\'([A-Za-z0-9_]{43})\'|([A-Za-z0-9_]{43}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"news_api_key"},
		Source:          "https://raw.githubusercontent.com/Newscatcher/news-mcp/main/README.md",
		Description:     "The official NewsCatcher MCP server NEWS_API_KEY environment role. Only literal assignments in the pinned 43-character candidate subset are recognized; the unqualified api_token tool parameter is not a default signature.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "messagebird-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*rest\.messagebird\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*rest\.messagebird\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy][ \t]+((?:test_)?[A-Za-z0-9_-]{25})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://rest\.messagebird\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"rest.messagebird.com"},
		Source:          "https://developers.messagebird.com/api/",
		Description:     "Complete request to the official messagebird API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("rest\\.messagebird\\.com", "/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "AccessKey", "(?:test_)?[A-Za-z0-9_-]{25}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "mixmax-api-token-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.mixmax\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.mixmax\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_-]{36})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.mixmax\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.mixmax.com"},
		Source:          "https://developer.mixmax.com/reference/authentication",
		Description:     "Complete request to the official mixmax API with its documented private X-API-Token credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.mixmax\\.com", "/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", "X-API-Token", "", "", "[A-Za-z0-9_-]{36}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "monday-api-token-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.monday\.com/v2[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v2[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.monday\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v2[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.monday\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.monday\.com/v2"|\'https://api\.monday\.com/v2\'|https://api\.monday\.com/v2)(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.monday\.com/v2"|\'https://api\.monday\.com/v2\'|https://api\.monday\.com/v2)(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.monday.com"},
		Source:          "https://developer.monday.com/api-reference/docs/authentication",
		Description:     "Complete request to the official monday API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.monday\\.com", "/v2", authorizationHeader, "", "", "[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+", validJWT),
	},
	{
		ID:              "moralis-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*deep-index\.moralis\.io\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*deep-index\.moralis\.io(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{64})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://deep-index\.moralis\.io/api/v2(?:\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"deep-index.moralis.io"},
		Source:          "https://docs.moralis.com/web3-data-api/evm/reference/Authentication",
		Description:     "Complete request to the official moralis API with its documented private X-API-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("deep-index\\.moralis\\.io", "/api/v2(?:\\.[0-9])?/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderMixed, "", "", "[A-Za-z0-9]{64}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "newsapi-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*newsapi\.org\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v2/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*newsapi\.org(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{32})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://newsapi\.org/v2/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"newsapi.org"},
		Source:          "https://newsapi.org/docs/authentication",
		Description:     "Complete request to the official newsapi API with its documented private X-Api-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("newsapi\\.org", "/v2/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderTitle, authorizationHeader, "", "[a-z0-9]{32}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "newscatcher-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*(?:api|v3-api)\.newscatcherapi\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*(?:api|v3-api)\.newscatcherapi\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]|[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9_]{43})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://(?:api|v3-api)\.newscatcherapi\.com/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.newscatcherapi.com"},
		Source:          "https://www.newscatcherapi.com/docs/news-api/api-reference/authentication",
		Description:     "Complete request to the official newscatcher API with its documented private X-Api-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("(?:api|v3-api)\\.newscatcherapi\\.com", "/(?:v2|api)/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderTitle, "X-Api-Token", "", "[A-Za-z0-9_]{43}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "nftport-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.nftport\.xyz\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.nftport\.xyz(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.nftport\.xyz/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.nftport.xyz"},
		Source:          "https://docs.nftport.xyz/reference/get_contracts",
		Description:     "Complete request to the official nftport API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.nftport\\.xyz", "/(?:me|v0)/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "", "[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "noticeable-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.noticeable\.io/graphql[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/graphql[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.noticeable\.io\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/graphql[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.noticeable\.io(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.noticeable\.io/graphql"|\'https://api\.noticeable\.io/graphql\'|https://api\.noticeable\.io/graphql)(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii][Kk][Ee][Yy][ \t]+([A-Za-z0-9]{20})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.noticeable\.io/graphql"|\'https://api\.noticeable\.io/graphql\'|https://api\.noticeable\.io/graphql)(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.noticeable.io"},
		Source:          "https://developer.noticeable.io/",
		Description:     "Complete request to the official noticeable API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.noticeable\\.io", "/graphql", authorizationHeader, "", "Apikey", "[A-Za-z0-9]{20}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "omnisend-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.omnisend\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.omnisend\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9-]{16,128})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.omnisend\.com/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.omnisend.com"},
		Source:          "https://api-docs.omnisend.com/reference/authentication",
		Description:     "Complete request to the official omnisend API with its documented private X-API-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.omnisend\\.com", "/v[345]/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderMixed, "", "", "[A-Za-z0-9-]{16,128}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "oopspam-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.oopspam\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.oopspam\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([A-Za-z0-9]{40})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.oopspam\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.oopspam.com"},
		Source:          "https://www.oopspam.com/docs/",
		Description:     "Complete request to the official oopspam API with its documented private X-Api-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.oopspam\\.com", "/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderTitle, "", "", "[A-Za-z0-9]{40}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "openuv-access-token-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.openuv\.io\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.openuv\.io(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})"|\'(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})"|\'(?:[Xx]\-[Aa][Cc][Cc][Ee][Ss][Ss]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([a-z0-9]{32})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.openuv\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.openuv.io"},
		Source:          "https://www.openuv.io/uvindex",
		Description:     "Complete request to the official openuv API with its documented private x-access-token credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.openuv\\.io", "/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", "x-access-token", "", "", "[a-z0-9]{32}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "overloop-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.overloop\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.overloop\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*([A-Za-z0-9_-]{50})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.overloop\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.overloop.com"},
		Source:          "https://apidoc.overloop.com/",
		Description:     "Complete request to the official overloop API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.overloop\\.com", "/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "", "[A-Za-z0-9_-]{50}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "pandadoc-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pandadoc\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.pandadoc\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Aa][Pp][Ii]\-[Kk][Ee][Yy][ \t]+([A-Za-z0-9]{40})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.pandadoc\.com/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.pandadoc.com"},
		Source:          "https://developers.pandadoc.com/reference/auth-overview",
		Description:     "Complete request to the official pandadoc API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.pandadoc\\.com", "/public/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "API-Key", "[A-Za-z0-9]{40}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "parsers-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.parsers\.dev\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.parsers\.dev(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.parsers\.dev/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.parsers.dev"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/parsers",
		Description:     "Complete request to the official parsers API with its documented private x-api-key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.parsers\\.dev", "/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderLower, "", "", "[a-z0-9]{64}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "peopledatalabs-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v5/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.peopledatalabs\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/v5/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.peopledatalabs\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{64})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.peopledatalabs\.com/v5/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.peopledatalabs.com"},
		Source:          "https://docs.peopledatalabs.com/docs/authentication",
		Description:     "Complete request to the official peopledatalabs API with its documented private X-Api-Key credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.peopledatalabs\\.com", "/v5/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderTitle, "", "", "[a-z0-9]{64}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "pipedrive-api-token-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Tt][Oo][Kk][Ee][Nn]):[ \t]*([A-Za-z0-9]{40})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://(?:api|[a-z0-9][a-z0-9-]{0,62})\.pipedrive\.com/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"pipedrive.com"},
		Source:          "https://pipedrive.readme.io/docs/core-api-concepts-authentication",
		Description:     "Complete request to the official pipedrive API with its documented private X-Api-Token credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("(?:api|[a-z0-9][a-z0-9-]{0,62})\\.pipedrive\\.com", "/(?:api/)?v[12]/[A-Za-z0-9_/?&=.,%-]{1,128}", "X-Api-Token", "", "", "[A-Za-z0-9]{40}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "podio-oauth-token-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.podio\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.podio\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Oo][Aa][Uu][Tt][Hh]2[ \t]+([a-z0-9]{32})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.podio\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.podio.com"},
		Source:          "https://developers.podio.com/authentication",
		Description:     "Complete request to the official podio API with its documented private Authorization credential carrier. Exact authority/path and HTTP framing prevent interpreting public/client keys from unrelated services as secrets.",
		ValidateContext: provider5HeaderContext("api\\.podio\\.com", "/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "OAuth2", "[a-z0-9]{32}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "moonclerk-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.moonclerk\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.moonclerk\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})"|\'(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]):[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn]=([a-z0-9]{32})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.moonclerk\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.moonclerk.com"},
		Source:          "https://raw.githubusercontent.com/moonclerk/developer/main/api/README.md",
		Description:     "Complete MoonClerk API request carrying its confidential Authorization: Token token= API key. MoonClerk explicitly says the key gives access to account data and must remain private; unrelated services and unscoped Token token= text do not match.",
		ValidateContext: provider5HeaderContext("api\\.moonclerk\\.com", "/[A-Za-z0-9_/?&=.,%-]{1,128}", authorizationHeader, "", "Token token=", "[a-z0-9]{32}", validAuditedCarrierLiteral2),
	},
	{
		ID:              "onfleet-api-key-curl",
		Regex:           `(?:\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([a-z0-9]{32}):"|\'([a-z0-9]{32}):\'|([a-z0-9]{32}):)(?:[ \t]|\\\r?\n)+(?:"https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128})|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:"https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://onfleet\.com/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([a-z0-9]{32}):"|\'([a-z0-9]{32}):\'|([a-z0-9]{32}):))(?:$|[ \t\r\n])`,
		Keywords:        []string{onfleetHost},
		Source:          onfleetAuthSource,
		Description:     "Documented literal curl -u/--user API-key authentication bound to the exact onfleet API endpoint. The credential username is confidential despite the empty/dummy password; arbitrary public username-only requests to other hosts are not detected.",
		ValidateContext: provider5CurlUserContext("onfleet\\.com", "/api/v2/[A-Za-z0-9_/?&=.,%-]{1,128}", "", "[a-z0-9]{32}"),
	},
	{
		ID:              "packagecloud-api-key-curl",
		Regex:           `(?:\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([a-f0-9]{48}):"|\'([a-f0-9]{48}):\'|([a-f0-9]{48}):)(?:[ \t]|\\\r?\n)+(?:"https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:"https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://packagecloud\.io/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([a-f0-9]{48}):"|\'([a-f0-9]{48}):\'|([a-f0-9]{48}):))(?:$|[ \t\r\n])`,
		Keywords:        []string{packagecloudHost},
		Source:          packagecloudAPISource,
		Description:     "Documented literal curl -u/--user API-key authentication bound to the exact packagecloud API endpoint. The credential username is confidential despite the empty/dummy password; arbitrary public username-only requests to other hosts are not detected.",
		ValidateContext: provider5CurlUserContext("packagecloud\\.io", "/api/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", "", "[a-f0-9]{48}"),
	},
	{
		ID:              "myintervals-api-key-curl",
		Regex:           `(?:\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([A-Za-z0-9]{11}):X"|\'([A-Za-z0-9]{11}):X\'|([A-Za-z0-9]{11}):X)(?:[ \t]|\\\r?\n)+(?:"https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128})|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:"https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.myintervals\.com/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([A-Za-z0-9]{11}):X"|\'([A-Za-z0-9]{11}):X\'|([A-Za-z0-9]{11}):X))(?:$|[ \t\r\n])`,
		Keywords:        []string{myIntervalsHost},
		Source:          myIntervalsAuthSource,
		Description:     "Documented literal curl -u/--user API-key authentication bound to the exact myintervals API endpoint. The credential username is confidential despite the empty/dummy password; arbitrary public username-only requests to other hosts are not detected.",
		ValidateContext: provider5CurlUserContext("api\\.myintervals\\.com", "/[A-Za-z0-9_/?&=.,%-]{1,128}", "X", "[A-Za-z0-9]{11}"),
	},
	{
		ID:              "paymo-api-key-curl",
		Regex:           `(?:\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([A-Za-z0-9]{8,256}):X"|\'([A-Za-z0-9]{8,256}):X\'|([A-Za-z0-9]{8,256}):X)(?:[ \t]|\\\r?\n)+(?:"https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:"https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://app\.paymoapp\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"([A-Za-z0-9]{8,256}):X"|\'([A-Za-z0-9]{8,256}):X\'|([A-Za-z0-9]{8,256}):X))(?:$|[ \t\r\n])`,
		Keywords:        []string{paymoHost},
		Source:          paymoAuthSource,
		Description:     "Documented literal curl -u/--user API-key authentication bound to the exact paymo API endpoint. The credential username is confidential despite the empty/dummy password; arbitrary public username-only requests to other hosts are not detected.",
		ValidateContext: provider5CurlUserContext("app\\.paymoapp\\.com", "/api/[A-Za-z0-9_/?&=.,%-]{1,128}", "X", "[A-Za-z0-9]{8,256}"),
	},
	{
		ID:              "paymongo-api-key-curl",
		Regex:           `(?:\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"(sk_(?:test|live)_[A-Za-z0-9]{24}):"|\'(sk_(?:test|live)_[A-Za-z0-9]{24}):\'|(sk_(?:test|live)_[A-Za-z0-9]{24}):)(?:[ \t]|\\\r?\n)+(?:"https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:"https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.paymongo\.com/v1/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:-u|--user)(?:[ \t]|\\\r?\n)+(?:"(sk_(?:test|live)_[A-Za-z0-9]{24}):"|\'(sk_(?:test|live)_[A-Za-z0-9]{24}):\'|(sk_(?:test|live)_[A-Za-z0-9]{24}):))(?:$|[ \t\r\n])`,
		Keywords:        []string{paymongoHost},
		Source:          paymongoAuthSource,
		Description:     "Documented literal curl -u/--user API-key authentication bound to the exact paymongo API endpoint. The credential username is confidential despite the empty/dummy password; arbitrary public username-only requests to other hosts are not detected.",
		ValidateContext: provider5CurlUserContext("api\\.paymongo\\.com", "/v1/[A-Za-z0-9_/?&=.,%-]{1,128}", "", "sk_(?:test|live)_[A-Za-z0-9]{24}"),
	},
	{
		ID:              "mockaroo-api-key-request",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.mockaroo\.com\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})(?:$|\r?\n)|\b(?:GET|POST|PUT|PATCH|DELETE)[ \t]+/api/[A-Za-z0-9_/?&=.,%-]{1,128}[ \t]+HTTP/1\.[01]\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})\r?\n(?:[A-Za-z0-9-]{1,64}:[^\r\n]{0,128}\r?\n){0,6}[Hh][Oo][Ss][Tt]:[ \t]*api\.mockaroo\.com(?:$|\r?\n)|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})\')(?:$|[ \t\r\n])|\bcurl(?:[ \t]|\\\r?\n)+(?:(?:-X|--request)(?:[ \t]|\\\r?\n)+(?:GET|POST|PUT|PATCH|DELETE)(?:[ \t]|\\\r?\n)+)?(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})"|\'(?:[Xx]\-[Aa][Pp][Ii]\-[Kk][Ee][Yy]):[ \t]*([a-z0-9]{8})\')(?:[ \t]|\\\r?\n)+(?:(?:-H|--header)(?:[ \t]|\\\r?\n)+(?:"[A-Za-z0-9-]{1,64}:[^"\r\n]{0,128}"|\'[A-Za-z0-9-]{1,64}:[^\'\r\n]{0,128}\')(?:[ \t]|\\\r?\n)+){0,3}(?:--url(?:[ \t]|\\\r?\n)+)?(?:"https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}"|\'https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128}\'|https://api\.mockaroo\.com/api/[A-Za-z0-9_/?&=.,%-]{1,128})(?:$|[ \t\r\n]))`,
		Keywords:        []string{"api.mockaroo.com"},
		Source:          mockarooAPISource,
		Description:     "Complete Mockaroo API request with its documented private X-API-Key header, including raw HTTP endpoint/Host and literal curl forms. Unscoped API-key headers and unrelated public/client keys remain excluded.",
		ValidateContext: provider5HeaderContext("api\\.mockaroo\\.com", "/api/[A-Za-z0-9_/?&=.,%-]{1,128}", apiKeyHeaderMixed, "", "", "[a-z0-9]{8}", validAuditedCarrierLiteral2),
	},
}

// URL matches consume their complete bounded query or userinfo carrier. Check
// continuations after that carrier, not after a key followed by valid '&' fields.
func validateProvider5URLContext(value string, start, end int, _ string) contextValidation {
	urlEnd := end
	if i := strings.IndexAny(value[start:end], " \t\r\n\"'`<>"); i >= 0 {
		urlEnd = start + i
	}
	if urlEnd < len(value) && (value[urlEnd] == '\'' || value[urlEnd] == '"' || value[urlEnd] == '`') {
		return contextValidation{accepted: true}
	}
	for urlEnd < len(value) && (value[urlEnd] == ' ' || value[urlEnd] == '\t') {
		urlEnd++
	}
	if urlEnd < len(value) && strings.ContainsRune("([{.$+-*/%?:=!<>|&\\", rune(value[urlEnd])) {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

func validProvider5AlphaNum(s string, size int, lowerOnly bool) bool {
	if len(s) != size || !validAuditedCarrierLiteral2(s) {
		return false
	}
	for i := range s {
		c := s[i]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || !lowerOnly && c >= 'A' && c <= 'Z' {
			continue
		}
		return false
	}
	return true
}

func validProvider5MesiboBody(body string) bool {
	op, token, ok := provider5JSONStringFields(body, "op", tokenField)
	return ok && op != "" && len(op) <= 64 && validProvider5AlphaNum(token, 64, false)
}

func validProvider5FormKey(body, name string, size int, lowerOnly bool) bool {
	fields, err := url.ParseQuery(body)
	values := fields[name]
	return err == nil && len(values) == 1 && validProvider5AlphaNum(values[0], size, lowerOnly)
}

func validProvider5MeaningCloudBody(body string) bool {
	return validProvider5FormKey(body, keyField, 32, true)
}

func validProvider5ParallelDotsBody(body string) bool {
	return validProvider5FormKey(body, apiKeyFieldSnake, 43, false)
}

func validProvider5Login(email, password string) bool {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || !strings.Contains(domain, ".") || strings.ContainsAny(email, "\r\n\t <>()") || strings.Contains(domain, "@") {
		return false
	}
	return len(email) <= 254 && len(password) <= 256 && validAuditedCarrierLiteral2(password)
}

func validProvider5MrTickTockBody(body string) bool {
	fields, err := url.ParseQuery(body)
	return err == nil && len(fields["email"]) == 1 && len(fields[passwordField]) == 1 && validProvider5Login(fields.Get("email"), fields.Get(passwordField))
}

func validProvider5OneDeskBody(body string) bool {
	email, password, ok := provider5JSONStringFields(body, "email", passwordField)
	return ok && validProvider5Login(email, password)
}

func provider5BasicParts(s string) (string, string, bool) {
	if len(s) > 800 {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil || !utf8.Valid(decoded) {
		return "", "", false
	}
	user, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}
	for i := range password {
		if password[i] < 0x20 || password[i] == 0x7f {
			return "", "", false
		}
	}
	return user, password, true
}

func validProvider5OnfleetBasic(s string) bool {
	user, password, ok := provider5BasicParts(s)
	return ok && password == "" && validProvider5AlphaNum(user, 32, true)
}

func validProvider5PackageCloudBasic(s string) bool {
	user, password, ok := provider5BasicParts(s)
	if !ok || password != "" || len(user) != 48 || !validAuditedCarrierLiteral2(user) {
		return false
	}
	for i := range user {
		if (user[i] < '0' || user[i] > '9') && (user[i] < 'a' || user[i] > 'f') {
			return false
		}
	}
	return true
}

func validProvider5MyIntervalsBasic(s string) bool {
	user, _, ok := provider5BasicParts(s)
	return ok && validProvider5AlphaNum(user, 11, false)
}

func validProvider5PaymoBasic(s string) bool {
	user, _, ok := provider5BasicParts(s)
	return ok && len(user) >= 8 && len(user) <= 256 && validProvider5AlphaNum(user, len(user), false)
}

func validProvider5PaymongoBasic(s string) bool {
	user, password, ok := provider5BasicParts(s)
	if !ok || password != "" || len(user) != 32 || !strings.HasPrefix(user, "sk_test_") && !strings.HasPrefix(user, "sk_live_") {
		return false
	}
	return validProvider5AlphaNum(user[8:], 24, false)
}

// Regexes nominate candidates; only the shared parser may establish the actual
// request target and top-level authentication carrier in the original value.
type provider5EndpointScope struct {
	host *lazyRegexp
	path *lazyRegexp
}

func newProvider5EndpointScope(host, path string) provider5EndpointScope {
	return provider5EndpointScope{
		host: newLazyRegexp("^(?:" + host + ")$"),
		path: newLazyRegexp("^(?:" + path + ")$"),
	}
}

func (scope provider5EndpointScope) matches(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != httpsScheme || u.User != nil || u.Fragment != "" || u.RawPath != "" || !scope.host.MatchString(u.Host) || !scope.path.MatchString(u.Path) {
		return false
	}
	for rest := u.Path; rest != ""; {
		var segment string
		segment, rest, _ = strings.Cut(rest, "/")
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func provider5HeaderContext(host, path, header, alternate, scheme, pattern string, validate func(string) bool) func(string, int, int, string) contextValidation {
	scope := newProvider5EndpointScope(host, path)
	credentialPattern := newLazyRegexp("^(?:" + pattern + ")$")
	schemeName, parameter, _ := strings.Cut(scheme, " ")
	return func(value string, start, end int, _ string) contextValidation {
		request, ok := parseAuditedProviderRequestContext2(value, start, end)
		if !ok || !scope.matches(request.url) {
			return contextValidation{}
		}
		matches := 0
		for _, line := range request.headers[:request.headerCount] {
			name, credential, found := strings.Cut(line, ":")
			if !found || !strings.EqualFold(name, header) && (alternate == "" || !strings.EqualFold(name, alternate)) {
				continue
			}
			matches++
			credential = strings.TrimSpace(credential)
			if schemeName != "" {
				space := strings.IndexAny(credential, " \t")
				if space < 0 || !strings.EqualFold(credential[:space], schemeName) {
					return contextValidation{}
				}
				credential = strings.TrimLeft(credential[space:], " \t")
				if parameter != "" {
					if len(credential) < len(parameter) || !strings.EqualFold(credential[:len(parameter)], parameter) {
						return contextValidation{}
					}
					credential = credential[len(parameter):]
				}
			}
			if !credentialPattern.MatchString(credential) || !validate(credential) || !validateAuditedAssignmentContext(line, 0, len(line), credential).accepted {
				return contextValidation{}
			}
		}
		return contextValidation{accepted: matches == 1}
	}
}

func provider5BodyContext(host, path string, validate func(string) bool) func(string, int, int, string) contextValidation {
	scope := newProvider5EndpointScope(host, path)
	return func(value string, start, end int, _ string) contextValidation {
		request, ok := parseAuditedProviderRequestContext2(value, start, end)
		return contextValidation{accepted: ok && request.method == httpMethodPost && scope.matches(request.url) && len(request.body) <= 1000 && validate(request.body)}
	}
}

func provider5CurlUserContext(host, path, password, pattern string) func(string, int, int, string) contextValidation {
	scope := newProvider5EndpointScope(host, path)
	credentialPattern := newLazyRegexp("^(?:" + pattern + ")$")
	return func(value string, start, end int, _ string) contextValidation {
		request, ok := parseAuditedProviderRequestContext2(value, start, end)
		if !ok || !scope.matches(request.url) {
			return contextValidation{}
		}
		user, actualPassword, found := strings.Cut(request.user, ":")
		return contextValidation{accepted: found && actualPassword == password && credentialPattern.MatchString(user) && validAuditedCarrierLiteral2(user)}
	}
}

// Exact case-sensitive top-level names are significant to provider APIs. Decode
// each required string once and reject duplicate fields rather than inheriting
// encoding/json's case-folding and last-key-wins behavior for Go structs.
func provider5JSONStringFields(body, first, second string) (string, string, bool) {
	decoder := json.NewDecoder(strings.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return "", "", false
	}
	var firstValue, secondValue string
	var firstSeen, secondSeen bool
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return "", "", false
		}
		name, ok := key.(string)
		if !ok {
			return "", "", false
		}
		switch name {
		case first:
			if firstSeen || decoder.Decode(&firstValue) != nil {
				return "", "", false
			}
			firstSeen = true
		case second:
			if secondSeen || decoder.Decode(&secondValue) != nil {
				return "", "", false
			}
			secondSeen = true
		default:
			var ignored json.RawMessage
			if decoder.Decode(&ignored) != nil {
				return "", "", false
			}
		}
	}
	closing, err := decoder.Token()
	return firstValue, secondValue, err == nil && closing == json.Delim('}') && firstSeen && secondSeen && strings.TrimSpace(body[decoder.InputOffset():]) == ""
}
