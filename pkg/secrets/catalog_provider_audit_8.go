package secrets

import (
	"encoding/json"
	"net/url"
	"strings"
)

// Provider applicability closure against TruffleHog v3.97.4 and Titus v1.2.9.
// Titus candidate constraints are covered by LICENSE.titus and NOTICE.titus.
// TruffleHog-derived candidate constraints remain under the repository AGPL.
var auditedProviderRules8 = []catalogRuleSpec{
	{
		ID:              "upcdatabase-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.upcdatabase\\.org))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.upcdatabase.org"},
		Source:          "https://upcdatabase.org/api-auth",
		Description:     "upcdatabase credential in a complete provider API URL with the documented apikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.upcdatabase.org", "/product/", apiKeyFieldLower, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 32, 32),
	},
	{
		ID:              "uptimerobot-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.uptimerobot\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{uptimeRobotHost},
		Source:          "https://uptimerobot.com/api/",
		Description:     "uptimerobot credential in a complete provider API URL with the documented api_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(uptimeRobotHost, "/v2/getMonitors", apiKeyFieldSnake, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-", 34, 34),
	},
	{
		ID:              "userstack-access-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.userstack\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.userstack.com"},
		Source:          "https://docs.apilayer.com/userstack/docs/getting-started",
		Description:     "userstack credential in a complete provider API URL with the documented access_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.userstack.com", "/detect", accessKeyField, "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "vatlayer-access-key-url",
		Regex:           "\\b(?i:https?://(?:www\\.apilayer\\.net|apilayer\\.net))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"www.apilayer.net", apilayerHost},
		Source:          "https://vatlayer.com/documentation",
		Description:     "vatlayer credential in a complete provider API URL with the documented access_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("www.apilayer.net|apilayer.net", "/api/validate", accessKeyField, "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "vbout-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.vbout\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.vbout.com"},
		Source:          "https://developers.vbout.com/quickstart",
		Description:     "vbout credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.vbout.com", "/1/", keyField, "0123456789", 25, 25),
	},
	{
		ID:              "verifier-api-token-url",
		Regex:           "\\b(?i:https?://(?:verifier\\.meetchopra\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"verifier.meetchopra.com"},
		Source:          "https://verifier.meetchopra.com/docs",
		Description:     "verifier credential in a complete provider API URL with the documented token authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("verifier.meetchopra.com", "/verify/", tokenField, "abcdefghijklmnopqrstuvwxyz0123456789", 96, 96),
	},
	{
		ID:              "verimail-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.verimail\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.verimail.io"},
		Source:          "https://verimail.io/docs/v3",
		Description:     "verimail credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.verimail.io", "/v3/verify", keyField, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 32, 32),
	},
	{
		ID:              "veriphone-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.veriphone\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.veriphone.io"},
		Source:          "https://veriphone.io/docs/v2",
		Description:     "veriphone credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.veriphone.io", "/v2/", keyField, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 32, 32),
	},
	{
		ID:              "visualcrossing-api-key-url",
		Regex:           "\\b(?i:https?://(?:weather\\.visualcrossing\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"weather.visualcrossing.com"},
		Source:          "https://www.visualcrossing.com/resources/documentation/weather-api/timeline-weather-api/",
		Description:     "visualcrossing credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("weather.visualcrossing.com", "/VisualCrossingWebServices/rest/services/timeline/", keyField, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 25, 25),
	},
	{
		ID:              "vpnapi-api-key-url",
		Regex:           "\\b(?i:https?://(?:vpnapi\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"vpnapi.io"},
		Source:          "https://vpnapi.io/api-documentation",
		Description:     "vpnapi credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("vpnapi.io", apiPath, keyField, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 32, 32),
	},
	{
		ID:              "vyte-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.vyte\\.in))/[^\\s\"<>\\x60]+",
		Keywords:        []string{vyteHost},
		Source:          "https://api-doc.vyte.in/",
		Description:     "vyte credential in a complete provider API URL with the documented api_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(vyteHost, "/v2/", apiKeyFieldSnake, "abcdefghijklmnopqrstuvwxyz0123456789", 50, 50),
	},
	{
		ID:              "walkscore-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.walkscore\\.com|transit\\.walkscore\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.walkscore.com", "transit.walkscore.com"},
		Source:          "https://www.walkscore.com/professional/api.php",
		Description:     "walkscore credential in a complete provider API URL with the documented wsapikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.walkscore.com|transit.walkscore.com", "/", "wsapikey", "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "weatherbit-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.weatherbit\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.weatherbit.io"},
		Source:          "https://www.weatherbit.io/api/weather-current",
		Description:     "weatherbit credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.weatherbit.io", "/v2.0/", keyField, "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "weatherstack-access-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.weatherstack\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.weatherstack.com"},
		Source:          "https://docs.apilayer.com/weatherstack/docs/getting-started",
		Description:     "weatherstack credential in a complete provider API URL with the documented access_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.weatherstack.com", "/current", accessKeyField, "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "webscraper-api-token-url",
		Regex:           "\\b(?i:https?://(?:api\\.webscraper\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.webscraper.io"},
		Source:          "https://webscraper.io/documentation/web-scraper-cloud/api",
		Description:     "webscraper credential in a complete provider API URL with the documented api_token authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.webscraper.io", apiV1Path, apiTokenField, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 60, 60),
	},
	{
		ID:              "webscrapingapi-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.webscrapingapi\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.webscrapingapi.com"},
		Source:          "https://docs.webscrapingapi.com/webscrapingapi/getting-started",
		Description:     "webscraping credential in a complete provider API URL with the documented api_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.webscrapingapi.com", "/v1", apiKeyFieldSnake, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 32, 32),
	},
	{
		ID:              "websitepulse-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.websitepulse\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.websitepulse.com"},
		Source:          "https://www.websitepulse.com/blog/exporting-data-from-websitepulses-api-to-google-drive",
		Description:     "websitepulse credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.websitepulse.com", "/textserver.php", keyField, "0123456789abcdef", 32, 32),
	},
	{
		ID:              "whoxy-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.whoxy\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.whoxy.com"},
		Source:          "https://www.whoxy.com/account_balance.php",
		Description:     "whoxy credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.whoxy.com", "/", keyField, "abcdefghijklmnopqrstuvwxyz0123456789", 33, 33),
	},
	{
		ID:              "wistia-access-token-url",
		Regex:           "\\b(?i:https?://(?:api\\.wistia\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.wistia.com"},
		Source:          "https://docs.wistia.com/docs/authenticating-with-oauth2",
		Description:     "wistia credential in a complete provider API URL with the documented access_token/bearer_token/api_password authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.wistia.com", "/v1/", "access_token|bearer_token|api_password", "abcdefghijklmnopqrstuvwxyz0123456789", 64, 64),
	},
	{
		ID:              "workstack-api-token-url",
		Regex:           "\\b(?i:https?://(?:app\\.workstack\\.io))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"app.workstack.io"},
		Source:          "https://support.workstack.io/en/articles/443335-the-api",
		Description:     "workstack credential in a complete provider API URL with the documented api_token authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("app.workstack.io", apiPath, apiTokenField, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 60, 60),
	},
	{
		ID:              "worldcoinindex-api-key-url",
		Regex:           "\\b(?i:https?://(?:www\\.worldcoinindex\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"www.worldcoinindex.com"},
		Source:          "https://www.worldcoinindex.com/apiservice",
		Description:     "worldcoinindex credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("www.worldcoinindex.com", "/apiservice/", keyField, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 35, 35),
	},
	{
		ID:              "worldweather-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.worldweatheronline\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.worldweatheronline.com"},
		Source:          "https://raw.githubusercontent.com/WorldWeatherOnline/weather-api-docs/main/README.md",
		Description:     "worldweather credential in a complete provider API URL with the documented key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.worldweatheronline.com", "/premium/v1/", keyField, "abcdefghijklmnopqrstuvwxyz0123456789", 31, 31),
	},
	{
		ID:              "zenrows-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.zenrows\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.zenrows.com"},
		Source:          "https://docs.zenrows.com/universal-scraper-api/api-reference",
		Description:     "zenrows credential in a complete provider API URL with the documented apikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.zenrows.com", "/v1", apiKeyFieldLower, "0123456789abcdef", 40, 40),
	},
	{
		ID:              "zenscrape-api-key-url",
		Regex:           "\\b(?i:https?://(?:app\\.zenscrape\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{zenscrapeHost},
		Source:          "https://app.zenscrape.com/documentation",
		Description:     "zenscrape credential in a complete provider API URL with the documented apikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(zenscrapeHost, apiV1Path, apiKeyFieldLower, "0123456789abcdef-", 36, 36),
	},
	{
		ID:              "zenserp-api-key-url",
		Regex:           "\\b(?i:https?://(?:app\\.zenserp\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{zenserpHost},
		Source:          "https://github.com/saasindustries/zenserp",
		Description:     "zenserp credential in a complete provider API URL with the documented apikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(zenserpHost, "/api/v2/", apiKeyFieldLower, "0123456789abcdef-", 36, 36),
	},
	{
		ID:              "zipcodebase-api-key-url",
		Regex:           "\\b(?i:https?://(?:app\\.zipcodebase\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{zipcodebaseHost},
		Source:          "https://github.com/saasindustries/zipcodebase",
		Description:     "zipcodebase credential in a complete provider API URL with the documented apikey authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(zipcodebaseHost, apiV1Path, apiKeyFieldLower, "0123456789abcdef-", 36, 36),
	},
	{
		ID:              "vastai-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:VAST_API_KEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-f0-9]{64})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"vast_api_key"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/vast-ai/vast-python/master/README.md",
		Description:     "vastai credential in the explicitly documented VAST_API_KEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "vercel-access-token",
		Regex:           `(?:(?:^|[^A-Za-z0-9_.-])(?i:VERCEL_TOKEN|VERCEL_ACCESS_TOKEN)["']?[ \t]*[:=][ \t]*["']?((?:[A-Za-z0-9]{24}|vcp_[A-Za-z0-9]{40,128}|vc[piark]_[A-Za-z0-9_-]{56}))(?:$|[\s"'\x60,;}&\]])|\b(vc[piark]_[A-Za-z0-9_-]{56})` + catalogRightBoundary + ")",
		Keywords:        []string{"vercel_token", "vercel_access_token", "vcp_", "vci_", "vca_", "vcr_", "vck_"},
		SecretGroup:     0,
		Source:          "https://vercel.com/changelog/new-token-formats-and-secret-scanning",
		Description:     "Vercel confidential credentials: preserves the existing VERCEL_TOKEN/VERCEL_ACCESS_TOKEN legacy assignment scope and adds complete bare personal, integration, app access/refresh and API-key prefixes. The case-sensitive 56-character URL-safe body is a Betterleaks scanner subset, not a published issuer grammar. Public client IDs and incomplete prefixes are excluded; no BPE heuristic or issuer verification.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "versioneye-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:VERSIONEYE_API_KEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9-]{40})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"versioneye_api_key"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/versioneye/versioneye_maven_plugin/master/README.md",
		Description:     "versioneye credential in the explicitly documented VERSIONEYE_API_KEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "virustotal-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:VTCLI_APIKEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-f0-9]{64})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"vtcli_apikey"},
		SecretGroup:     1,
		Source:          "https://virustotal.github.io/vt-cli/",
		Description:     "virustotal credential in the explicitly documented VTCLI_APIKEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "vultr-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:VULTR_API_KEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Z0-9]{36})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"vultr_api_key"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/vultr/vultr-cli/master/README.md",
		Description:     "vultr credential in the explicitly documented VULTR_API_KEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "wit-server-access-token",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:WIT_TOKEN)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Z0-9]{32})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"wit_token"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/wit-ai/node-wit/main/README.md",
		Description:     "wit-ai credential in the explicitly documented WIT_TOKEN assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "wiz-client-secret",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:WIZ_AUTH_CLIENT_SECRET)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{64})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"wiz_auth_client_secret"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/queeno/terraform-provider-wiz/main/docs/index.md",
		Description:     "wiz credential in the explicitly documented WIZ_AUTH_CLIENT_SECRET assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "worldweather-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:WWO_API_KEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{31})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"wwo_api_key"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/WorldWeatherOnline/weather-api-docs/main/README.md",
		Description:     "worldweather credential in the explicitly documented WWO_API_KEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "wpengine-api-password",
		Regex:           "(?:^|[^A-Za-z0-9_.-])WPE_API_PASSWORD[\"']?[ \\t]*[:=][ \\t]*((?:\"[^\"\\\\\\r\\n]{1,128}\"|'[^'\\\\\\r\\n]{1,128}'|[0-9][A-Za-z0-9]{23,63}))(?:$|[\\s,;}&\\]])",
		Keywords:        []string{"wpe_api_password"},
		SecretGroup:     1,
		Source:          "https://developers.wpengine.com/docs/managed-hosting-platform/api/tutorial/",
		Description:     "Documented WPE_API_PASSWORD literal. Quoted passwords use binder-aware noninterpolating literal checks; the bare 24-64 alphanumeric subset must start with a digit so source identifiers are not mistaken for broad passwords. Public API user IDs are excluded.",
		Validate:        validQuotedOrBareCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider8Password),
	},
	{
		ID:              "zulip-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:ZULIP_API_KEY)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{32})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"zulip_api_key"},
		SecretGroup:     1,
		Source:          "https://zulip.com/api/api-keys",
		Description:     "zulip credential in the explicitly documented ZULIP_API_KEY assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "vault-approle-secret-id",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:role_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}[\"']?[ \\t\\r\\n]*,?[ \\t\\r\\n]*[\"']?(?i:secret_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"role_id"},
		SecretGroup:     1,
		Source:          "https://developer.hashicorp.com/vault/docs/auth/approle",
		Description:     "Vault AppRole adjacent role_id and secret_id UUID literals in JSON/config or a Vault CLI login. The SecretID is explicitly always secret; standalone UUIDs, role IDs and secret accessors are excluded. Only the documented random-UUID form is supported, not arbitrary custom SecretIDs.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "zenkit-api-key",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:Zenkit-API-Key)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{8}-[A-Za-z0-9]{32})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"zenkit-api-key"},
		SecretGroup:     1,
		Source:          "https://base.zenkit.com/docs/api/overview/authentication",
		Description:     "zenkit credential in the explicitly documented Zenkit-API-Key assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "zonka-feedback-api-token",
		Regex:           "(?:^|[^A-Za-z0-9_.-])(?i:Z-API-TOKEN)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{36})(?:$|[\\s\"'\\x60,;}&\\]])",
		Keywords:        []string{"z-api-token"},
		SecretGroup:     1,
		Source:          "https://apidocs.zonkafeedback.com/",
		Description:     "zonka-feedback credential in the explicitly documented Z-API-TOKEN assignment in the scanned value. The credential role, not provider proximity or a bare opaque body, establishes confidentiality; bounds are a supported scanner subset.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "tyntec-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([A-Za-z0-9]{32})[\"'][^\\r\\n]{0,1000}(?i:https?://api\\.tyntec\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://api\\.tyntec\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([A-Za-z0-9]{32})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"api.tyntec.com"},
		SecretGroup:     0,
		Source:          "https://api.tyntec.com/reference/sms/current.html",
		Description:     "tyntec private apikey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "api.tyntec.com", header: apiKeyFieldLower})),
	},

	{
		ID:              "unifyid-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:X-API-Key)[ \\t]*:[ \\t]*([A-Za-z0-9_=-]{44})[\"'][^\\r\\n]{0,1000}(?i:https?://api\\.unify\\.id)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://api\\.unify\\.id)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:X-API-Key)[ \\t]*:[ \\t]*([A-Za-z0-9_=-]{44})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"api.unify.id"},
		SecretGroup:     0,
		Source:          "https://developer.unify.id/docs/api/",
		Description:     "unifyid private X-API-Key header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "api.unify.id", header: apiKeyHeaderMixed})),
	},

	{
		ID:              "unplugg-api-token-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:x-access-token)[ \\t]*:[ \\t]*([a-z0-9]{64})[\"'][^\\r\\n]{0,1000}(?i:https?://api\\.unplu\\.gg)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://api\\.unplu\\.gg)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:x-access-token)[ \\t]*:[ \\t]*([a-z0-9]{64})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"api.unplu.gg"},
		SecretGroup:     0,
		Source:          "https://unplu.gg/test_api.html",
		Description:     "unplugg private x-access-token header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "api.unplu.gg", header: "x-access-token"})),
	},

	{
		ID:              "uplead-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:Authorization)[ \\t]*:[ \\t]*([a-z0-9-]{32})[\"'][^\\r\\n]{0,1000}(?i:https?://api\\.uplead\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://api\\.uplead\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:Authorization)[ \\t]*:[ \\t]*([a-z0-9-]{32})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"api.uplead.com"},
		SecretGroup:     0,
		Source:          "https://docs.uplead.com/",
		Description:     "uplead private Authorization header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "api.uplead.com", header: authorizationHeader})),
	},

	{
		ID:              "urlscan-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:API-Key)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000}(?i:https?://urlscan\\.io)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://urlscan\\.io)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:API-Key)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"urlscan.io"},
		SecretGroup:     0,
		Source:          "https://urlscan.io/docs/api/",
		Description:     "urlscan private API-Key header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "urlscan.io", header: "API-Key"})),
	},

	{
		ID:              "virustotal-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:x-apikey)[ \\t]*:[ \\t]*([a-f0-9]{64})[\"'][^\\r\\n]{0,1000}(?i:https?://www\\.virustotal\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://www\\.virustotal\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:x-apikey)[ \\t]*:[ \\t]*([a-f0-9]{64})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"www.virustotal.com"},
		SecretGroup:     0,
		Source:          "https://docs.virustotal.com/reference/authentication",
		Description:     "virustotal private x-apikey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "www.virustotal.com", header: "x-apikey"})),
	},

	{
		ID:              "versioneye-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:apiKey)[ \\t]*:[ \\t]*([A-Za-z0-9-]{40})[\"'][^\\r\\n]{0,1000}(?i:https?://www\\.versioneye\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://www\\.versioneye\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:apiKey)[ \\t]*:[ \\t]*([A-Za-z0-9-]{40})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{"www.versioneye.com"},
		SecretGroup:     0,
		Source:          "https://raw.githubusercontent.com/versioneye/versioneye_maven_plugin/master/README.md",
		Description:     "versioneye private apiKey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: "www.versioneye.com", header: apiKeyFieldCamel})),
	},

	{
		ID:              "vyte-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:Authorization)[ \\t]*:[ \\t]*([a-z0-9]{50})[\"'][^\\r\\n]{0,1000}(?i:https?://api\\.vyte\\.in)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://api\\.vyte\\.in)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:Authorization)[ \\t]*:[ \\t]*([a-z0-9]{50})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{vyteHost},
		SecretGroup:     0,
		Source:          "https://api-doc.vyte.in/",
		Description:     "vyte private Authorization header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: vyteHost, header: authorizationHeader})),
	},

	{
		ID:              "zenscrape-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000}(?i:https?://app\\.zenscrape\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://app\\.zenscrape\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{zenscrapeHost},
		SecretGroup:     0,
		Source:          "https://app.zenscrape.com/documentation",
		Description:     "zenscrape private apikey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: zenscrapeHost, header: apiKeyFieldLower})),
	},

	{
		ID:              "zenserp-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000}(?i:https?://app\\.zenserp\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://app\\.zenserp\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{zenserpHost},
		SecretGroup:     0,
		Source:          "https://github.com/saasindustries/zenserp",
		Description:     "zenserp private apikey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: zenserpHost, header: apiKeyFieldLower})),
	},

	{
		ID:              "zipcodebase-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?:(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000}(?i:https?://app\\.zipcodebase\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}|(?i:https?://app\\.zipcodebase\\.com)/[^\\s\"']*[^\\r\\n]{0,1000}(?:-H|--header)[ \\t]+[\"'](?i:apikey)[ \\t]*:[ \\t]*([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[\"'][^\\r\\n]{0,1000})",
		Keywords:        []string{zipcodebaseHost},
		SecretGroup:     0,
		Source:          "https://github.com/saasindustries/zipcodebase",
		Description:     "zipcodebase private apikey header in a complete single-line curl request to the exact provider API authority. Shell arguments and matching literal quotes are parsed; bare keys, generic headers without the provider URL, variable expansion and source expressions are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(provider8CurlCredential(provider8Request{host: zipcodebaseHost, header: apiKeyFieldLower})),
	},

	{
		ID:              "typetalk-client-secret",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?i:https://typetalk\\.com)/oauth2/access_token[^\\r\\n]{0,1000}",
		Keywords:        []string{typetalkHost},
		Source:          "https://developer.nulab.com/docs/typetalk/auth/",
		Description:     "Typetalk OAuth client_secret in a complete single-line curl form POST to its token endpoint. Client IDs and API URLs without the private form value are excluded. The 64-alphanumeric body is the pinned scanner subset.",
		ValidateContext: provider8CurlCredential(provider8Request{host: typetalkHost, path: "/oauth2/access_token", formKey: clientSecretField, alphabet: provider8Alnum, min: 64, max: 64}),
	},
	{
		ID:              "unsplash-application-secret",
		Regex:           "\\bUnsplash\\.configure[ \\t]+do[ \\t]+\\|config\\|[ \\t\\r\\n]{0,20}(?:#[^\\r\\n]{0,80}[\\r\\n][ \\t\\r\\n]{0,20}){0,3}(?:config\\.(?:application_access_key|application_redirect_uri|utm_source)[ \\t]*=[ \\t]*(?:\"[^\"\\\\\\r\\n]{0,256}\"|'[^'\\\\\\r\\n]{0,256}')[ \\t\\r\\n]{0,20}(?:#[^\\r\\n]{0,80}[\\r\\n][ \\t\\r\\n]{0,20}){0,3}){0,3}config\\.application_secret[ \\t]*=[ \\t]*[\"']([A-Za-z0-9_-]{43})[\"']",
		Keywords:        []string{"Unsplash.configure"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/unsplash/unsplash_rb/master/README.md",
		Description:     "Documented Unsplash Ruby configuration application_secret literal inside Unsplash.configure. Public access keys used as client_id or Authorization: Client-ID are deliberately excluded; only the separate confidential OAuth secret role is classified.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "zipcodeapi-server-key-url",
		Regex:           "\\b(?i:https?://www\\.zipcodeapi\\.com)/rest/[^\\s\"<>\\x60]+",
		Keywords:        []string{"www.zipcodeapi.com/rest/"},
		Source:          "https://www.zipcodeapi.com/API",
		Description:     "ZipCodeAPI private server API key in the complete /rest/{key}/ API URL. Documented js- client keys intended for browser embedding are excluded; malformed authorities and source concatenations are rejected.",
		ValidateContext: validateProvider8ZipcodeURL,
	},
	{
		ID:              "zipbooks-login-credentials",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?i:https://api\\.zipbooks\\.com)/v2/auth/login[^\\r\\n]{0,1000}",
		Keywords:        []string{"api.zipbooks.com"},
		Source:          "https://developer.zipbooks.com/",
		Description:     "Complete ZipBooks login curl request containing a literal JSON email/password pair. JSON string semantics preserve literal dollar characters inside single-quoted shell data; interpolation, bare password references, public account identifiers and malformed JSON are rejected.",
		ValidateContext: provider8CurlCredential(provider8Request{host: "api.zipbooks.com", path: "/v2/auth/login", jsonLogin: true}),
	},
	{
		ID:              "zerobounce-api-key-url",
		Regex:           "\\b(?i:https?://(?:api\\.zerobounce\\.net))/[^\\s\"<>\\x60]+",
		Keywords:        []string{"api.zerobounce.net"},
		Source:          "https://www.zerobounce.net/docs/email-validation-api-quickstart/",
		Description:     "zerobounce credential in a complete provider API URL with the documented api_key authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential("api.zerobounce.net", "/v2/", apiKeyFieldSnake, "abcdefghijklmnopqrstuvwxyz0123456789", 32, 32),
	},
	{
		ID:              "zendesk-api-token",
		Regex:           "\\bZendeskAPI::Client\\.new[ \\t]+do[ \\t]+\\|config\\|[ \\t\\r\\n]{0,20}(?:#[^\\r\\n]{0,80}[\\r\\n][ \\t\\r\\n]{0,20}){0,3}(?:config\\.(?:url|username)[ \\t]*=[ \\t]*(?:\"[^\"\\\\\\r\\n]{0,256}\"|'[^'\\\\\\r\\n]{0,256}')[ \\t\\r\\n]{0,20}(?:#[^\\r\\n]{0,80}[\\r\\n][ \\t\\r\\n]{0,20}){0,3}){0,2}config\\.token[ \\t]*=[ \\t]*[\"']([A-Za-z0-9_-]{40})[\"']",
		Keywords:        []string{"ZendeskAPI::Client.new"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/zendesk/zendesk_api_client_rb/master/README.md",
		Description:     "Zendesk Ruby SDK config.token literal inside the documented client initialization block. Only literal URL/username assignments and comments may precede it; arbitrary source proximity cannot supply provider context. Basic authentication and OAuth bearer requests retain the common rules.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "typetalk-client-secret-url",
		Regex:           "\\b(?i:https?://(?:typetalk\\.com))/[^\\s\"<>\\x60]+",
		Keywords:        []string{typetalkHost},
		Source:          "https://developer.nulab.com/docs/typetalk/auth/",
		Description:     "typetalk credential in a complete provider API URL with the documented client_secret authentication parameter. Exact parsed authority/path, one credential parameter, literal token and a 4096-byte URL cap are required; public IDs and opaque bare strings are not classified.",
		ValidateContext: provider8QueryCredential(typetalkHost, "/oauth2/access_token", clientSecretField, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 64, 64),
	},
	{
		ID:              "uptimerobot-api-key-request",
		Regex:           "\\bcurl[ \\t]+[^\\r\\n]{0,1000}(?i:https://api\\.uptimerobot\\.com)/v2/getMonitors[^\\r\\n]{0,1000}",
		Keywords:        []string{uptimeRobotHost},
		Source:          "https://uptimerobot.com/api/",
		Description:     "UptimeRobot API key in a complete curl form POST to the documented getMonitors endpoint. The scanner 9-alphanumeric hyphen 24-alphanumeric body is structurally checked; variables, public monitor identifiers and malformed shell arguments are excluded.",
		ValidateContext: provider8CurlCredential(provider8Request{host: uptimeRobotHost, path: "/v2/getMonitors", formKey: apiKeyFieldSnake, alphabet: provider8Alnum + "-", min: 34, max: 34}),
	},
}

const provider8Alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// provider8Token retains candidate bounds without attributing opaque bare values.
func provider8Token(s, alphabet string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen || !validCarrier3Literal(s) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(alphabet, rune(s[i])) {
			return false
		}
	}
	if minLen == 36 && maxLen == 36 && alphabet == "0123456789abcdef-" {
		for i := 0; i < len(s); i++ {
			dash := i == 8 || i == 13 || i == 18 || i == 23
			if (s[i] == '-') != dash {
				return false
			}
		}
	}
	if minLen == 34 && maxLen == 34 && alphabet == provider8Alnum+"-" {
		for i := 0; i < len(s); i++ {
			if (s[i] == '-') != (i == 9) {
				return false
			}
		}
	}
	return true
}

func provider8QueryCredential(hosts, path, key, alphabet string, minLen, maxLen int) func(string, int, int, string) contextValidation {
	allowedHosts := strings.Split(hosts, "|")
	allowedKeys := strings.Split(key, "|")
	return func(value string, start, end int, secret string) contextValidation {
		s, ok := frameConnectionURI(value, start, end, secret)
		if !ok || len(s) > 4096 || start > 0 && value[start-1] == ':' {
			return contextValidation{}
		}
		u, err := url.Parse(s)
		if err != nil || u.User != nil || u.Fragment != "" || (!strings.EqualFold(u.Scheme, httpScheme) && !strings.EqualFold(u.Scheme, httpsScheme)) {
			return contextValidation{}
		}
		allowed := false
		for _, host := range allowedHosts {
			if strings.EqualFold(u.Host, host) {
				allowed = true
				break
			}
		}
		if !allowed || !provider8APIPath(u.Path, path) {
			return contextValidation{}
		}
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return contextValidation{}
		}
		var credential string
		for _, name := range allowedKeys {
			if values := query[name]; len(values) != 0 {
				if len(values) != 1 || credential != "" {
					return contextValidation{}
				}
				credential = values[0]
			}
		}
		return contextValidation{accepted: provider8Token(credential, alphabet, minLen, maxLen)}
	}
}

func provider8APIPath(got, prefix string) bool {
	if prefix == "/" {
		return strings.HasPrefix(got, "/")
	}
	if strings.HasSuffix(prefix, "/") {
		return len(got) > len(prefix) && strings.HasPrefix(got, prefix)
	}
	return got == prefix || strings.HasPrefix(got, prefix+"/")
}

func validateProvider8ZipcodeURL(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > 4096 || start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || !strings.EqualFold(u.Host, "www.zipcodeapi.com") || u.User != nil || u.Fragment != "" || !strings.HasPrefix(u.Path, "/rest/") {
		return contextValidation{}
	}
	key, endpoint, ok := strings.Cut(strings.TrimPrefix(u.Path, "/rest/"), "/")
	return contextValidation{accepted: ok && endpoint != "" && provider8Token(key, provider8Alnum, 64, 64)}
}

// provider8Request is the bounded, literal curl subset needed by this slice.
// It parses arguments rather than joining a header to nearby provider prose.
type provider8Request struct {
	host      string
	header    string
	path      string
	formKey   string
	alphabet  string
	min       int
	max       int
	jsonLogin bool
}

func provider8CurlCredential(spec provider8Request) func(string, int, int, string) contextValidation {
	return func(value string, start, end int, secret string) contextValidation {
		if end-start > 4096 || end < len(value) && value[end] != '\r' && value[end] != '\n' {
			return contextValidation{}
		}
		command := value[start:end]
		first, offset, ok := provider8CurlWord(command, 0)
		if !ok || first != curlCommand {
			return contextValidation{}
		}
		foundURL, foundCredential := false, false
		for offset < len(command) {
			arg, next, valid := provider8CurlWord(command, offset)
			if !valid {
				return contextValidation{}
			}
			offset = next
			if arg == "" {
				break
			}
			switch arg {
			case "-H", curlHeaderOption, "-d", curlDataOption, curlDataRawOption, "-X", curlRequestOption, curlURLOption:
				option := arg
				arg, offset, valid = provider8CurlWord(command, offset)
				if !valid || arg == "" {
					return contextValidation{}
				}
				switch option {
				case "-H", curlHeaderOption:
					name, content, present := strings.Cut(arg, ":")
					if !present {
						return contextValidation{}
					}
					if spec.header != "" && strings.EqualFold(strings.TrimSpace(name), spec.header) {
						content = strings.TrimSpace(content)
						if content != secret || !validCarrier3Literal(content) || foundCredential {
							return contextValidation{}
						}
						foundCredential = true
					}
					continue
				case "-d", curlDataOption, curlDataRawOption:
					if spec.jsonLogin {
						var login struct {
							Email    string
							Password string
						}
						if json.Unmarshal([]byte(arg), &login) != nil || !strings.Contains(login.Email, "@") || !validCarrier3Literal(login.Password) || foundCredential {
							return contextValidation{}
						}
						if !validateAuditedAssignmentContext(value, start, end, login.Password).accepted {
							return contextValidation{}
						}
						foundCredential = true
					} else if spec.formKey != "" {
						form, err := url.ParseQuery(arg)
						if err != nil || len(form[spec.formKey]) > 1 {
							return contextValidation{}
						}
						if values := form[spec.formKey]; len(values) == 1 {
							if foundCredential || !provider8Token(values[0], spec.alphabet, spec.min, spec.max) || !validateProvider8FormAssignment(arg, spec.formKey, values[0]).accepted {
								return contextValidation{}
							}
							foundCredential = true
						}
					}
					continue
				case "-X", curlRequestOption:
					if (spec.jsonLogin || spec.formKey != "") && arg != httpMethodPost {
						return contextValidation{}
					}
					continue
				}
			case "-s", curlSilentOption, "-S", curlShowErrorOption, "-L", curlLocationOption, curlCompressedOption, "-i", "--include", "-f", curlFailOption:
				continue
			}
			u, err := url.Parse(arg)
			if err != nil || (!strings.EqualFold(u.Scheme, httpsScheme) && !strings.EqualFold(u.Scheme, httpScheme)) || !strings.EqualFold(u.Host, spec.host) || u.User != nil || u.Fragment != "" || foundURL || u.Path == "" || spec.path != "" && u.Path != spec.path {
				return contextValidation{}
			}
			foundURL = true
		}
		return contextValidation{accepted: foundURL && foundCredential}
	}
}

// Shell words and form fields already have protocol framing here: a following
// -d option or &field is not a source-expression continuation of this value.
func validateProvider8FormAssignment(form, key, secret string) contextValidation {
	for form != "" {
		field, rest, more := strings.Cut(form, "&")
		name, _, assigned := strings.Cut(field, "=")
		if assigned {
			decodedName, err := url.QueryUnescape(name)
			if err == nil && decodedName == key {
				return validateAuditedAssignmentContext(field, 0, len(field), secret)
			}
		}
		if !more {
			break
		}
		form = rest
	}
	return contextValidation{}
}

// No evaluation, escapes, concatenation, substitutions or shell operators.
// Single-quoted data is literal; double-quoted dollar/backtick expansion is not.
func provider8CurlWord(s string, offset int) (string, int, bool) {
	for offset < len(s) && (s[offset] == ' ' || s[offset] == '\t') {
		offset++
	}
	if offset == len(s) {
		return "", offset, true
	}
	start := offset
	quote := byte(0)
	if s[offset] == '\'' || s[offset] == '"' {
		quote = s[offset]
		offset++
		start = offset
	}
	for offset < len(s) {
		c := s[offset]
		if quote != 0 && c == quote {
			if offset+1 < len(s) && s[offset+1] != ' ' && s[offset+1] != '\t' {
				return "", offset, false
			}
			return s[start:offset], offset + 1, true
		}
		if quote == 0 && (c == ' ' || c == '\t') {
			return s[start:offset], offset, true
		}
		if c == '\\' || c == '\r' || c == '\n' || c == 0 || quote == '"' && (c == '$' || c == 96) || quote == 0 && (c == 96 || strings.ContainsRune("\"'$;&|<>(){}[]", rune(c))) {
			return "", offset, false
		}
		offset++
	}
	return s[start:offset], offset, quote == 0
}

func validateProvider8Password(value string, start, end int, secret string) contextValidation {
	if len(secret) != 0 && (secret[0] == '\'' || secret[0] == '"') {
		relative := strings.LastIndex(value[start:end], secret)
		if relative < 0 {
			return contextValidation{}
		}
		prefix := strings.TrimRight(value[start:start+relative], " \t")
		if strings.HasSuffix(prefix, ":") {
			return contextValidation{accepted: validQuotedAuditedPassword2(secret)}
		}
		return contextValidation{accepted: validExpansionFreeQuotedPassword2(secret, true)}
	}
	return contextValidation{accepted: len(secret) >= 24 && len(secret) <= 64 && secret[0] >= '0' && secret[0] <= '9' && validAuditedCarrierLiteral2(secret)}
}
