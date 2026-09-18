package secrets

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Exact-role carrier subsets audited against Titus v1.2.9 and TruffleHog
// v3.97.4. Scanner-derived candidate constraints are covered by LICENSE.titus,
// NOTICE.titus and the repository AGPL; protocol parsing is implemented natively.
var auditedProviderRules6 = []catalogRuleSpec{
	{
		ID:              "positionstack-access-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.positionstack\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.positionstack.com"},
		SecretGroup:     1,
		Source:          "https://docs.apilayer.com/positionstack/docs/getting-started",
		Description:     "positionstack credential in a complete HTTP(S) API URL with the documented access_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.positionstack\.com)$`, `^(?:/v1/(?:forward|reverse))$`, accessKeyField, `^(?:[A-Za-z0-9_]{32})$`),
	},
	{
		ID:              "postageapp-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.postageapp\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.postageapp.com"},
		SecretGroup:     1,
		Source:          postageAppSource,
		Description:     "postageapp credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.postageapp\.com)$`, `^(?:/v\.1\.0/[A-Za-z_]+\.json)$`, apiKeyFieldSnake, `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "postageapp-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:POSTAGEAPP_API_KEY)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Za-z0-9]{32})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"postageapp"},
		SecretGroup:     1,
		Source:          postageAppSource,
		Description:     "postageapp credential in the exact documented POSTAGEAPP_API_KEY assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "proxycrawl-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.(?:proxycrawl|crawlbase)\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"proxycrawl.com", "crawlbase.com"},
		SecretGroup:     1,
		Source:          "https://crawlbase.com/docs/crawling-api/",
		Description:     "proxycrawl credential in a complete HTTP(S) API URL with the documented token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.(?:proxycrawl|crawlbase)\.com)$`, `^(?:/[A-Za-z0-9/_-]*)$`, tokenField, `^(?:[A-Za-z0-9_]{22})$`),
	},
	{
		ID:              "qase-testops-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:QASE_TESTOPS_API_TOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Za-z0-9]{40})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"qase"},
		SecretGroup:     1,
		Source:          "https://developers.qase.io/docs/connecting-to-qase",
		Description:     "qase credential in the exact documented QASE_TESTOPS_API_TOKEN assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rawg-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.rawg\.io)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.rawg.io"},
		SecretGroup:     1,
		Source:          "https://rawg.io/apidocs",
		Description:     "rawg credential in a complete HTTP(S) API URL with the documented key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.rawg\.io)$`, `^(?:/api/[A-Za-z0-9/_-]+)$`, keyField, `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "rebrandly-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.rebrandly\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{rebrandlyHost},
		SecretGroup:     1,
		Source:          rebrandlyAuthSource,
		Description:     "rebrandly credential in a complete HTTP(S) API URL with the documented apikey authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.rebrandly\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, apiKeyFieldLower, `^(?:[A-Za-z0-9_]{32})$`),
	},
	{
		ID:              "redhat-pyxis-api-token-option",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:--pyxis-api-token)["\x27]?[ \t]*[:=][ \t]*["\x27]?([a-z0-9]{32})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"--pyxis-api-token"},
		SecretGroup:     1,
		Source:          "https://github.com/redhat-openshift-ecosystem/openshift-preflight",
		Description:     "redhat-pyxis credential in the exact documented --pyxis-api-token assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "refiner-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.refiner\.io)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{refinerHost},
		SecretGroup:     1,
		Source:          refinerAPISource,
		Description:     "refiner credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.refiner\.io)$`, `^(?:/v1/[A-Za-z0-9/_-]*)$`, apiKeyFieldSnake, `^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`),
	},
	{
		ID:              "restpack-access-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:restpack\.io)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"restpack.io"},
		SecretGroup:     1,
		Source:          "https://restpack.io/html2pdf/docs",
		Description:     "restpack credential in a complete HTTP(S) API URL with the documented access_token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:restpack\.io)$`, `^(?:/api/(?:html2pdf|screenshot)/[A-Za-z0-9/_-]+)$`, "access_token", `^(?:[A-Za-z0-9]{48})$`),
	},
	{
		ID:              "ritekit-private-client-id-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.ritekit\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.ritekit.com"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/RiteKit/api-docs/master/apiary.apib",
		Description:     "ritekit credential in a complete HTTP(S) API URL with the documented client_id authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. The provider explicitly restricts direct client_id authentication to non-public clients; no standalone client identifier is classified. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.ritekit\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, clientIDField, `^(?:[a-f0-9]{26}|[a-f0-9]{44})$`),
	},
	{
		ID:              "route4me-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.route4me\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.route4me.com"},
		SecretGroup:     1,
		Source:          "https://route4me.io/docs/",
		Description:     "route4me credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. The public all-ones demo key is excluded. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.route4me\.com)$`, `^(?:/api\.v[45]/[A-Za-z0-9/_.-]+)$`, apiKeyFieldSnake, `^(?:[A-Z0-9]{32})$`),
	},
	{
		ID:              "scrapeowl-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.scrapeowl\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.scrapeowl.com"},
		SecretGroup:     1,
		Source:          "https://scrapeowl.com/docs/",
		Description:     "scrapeowl credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. Both the historical scanner width and the currently documented 80-character key form are supported. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.scrapeowl\.com)$`, `^(?:/v1/scrape)$`, apiKeyFieldSnake, `^(?:[a-z0-9]{30}|[A-Za-z0-9]{80})$`),
	},
	{
		ID:              "scraperapi-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.scraperapi\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.scraperapi.com"},
		SecretGroup:     1,
		Source:          "https://docs.scraperapi.com/",
		Description:     "scraperapi credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.scraperapi\.com)$`, `^(?:(?:/[A-Za-z0-9/_-]*)?)$`, apiKeyFieldSnake, `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "scraperbox-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.scraperbox\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.scraperbox.com"},
		SecretGroup:     1,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/scraperbox",
		Description:     "scraperbox credential in a complete HTTP(S) API URL with the documented token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. The historical provider documentation is unavailable; the exact authentication carrier is grounded in the pinned verifier, not inferred from a brand keyword. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.scraperbox\.com)$`, `^(?:/scrape)$`, tokenField, `^(?:[A-Z0-9]{32})$`),
	},
	{
		ID:              "scrapestack-access-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.scrapestack\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.scrapestack.com"},
		SecretGroup:     1,
		Source:          "https://scrapestack.com/documentation",
		Description:     "scrapestack credential in a complete HTTP(S) API URL with the documented access_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.scrapestack\.com)$`, `^(?:/scrape)$`, accessKeyField, `^(?:[a-z0-9]{32})$`),
	},
	{
		ID:              "scrapingant-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.scrapingant\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.scrapingant.com"},
		SecretGroup:     1,
		Source:          "https://docs.scrapingant.com/request-response-format",
		Description:     "scrapingant credential in a complete HTTP(S) API URL with the documented x-api-key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.scrapingant\.com)$`, `^(?:/v[12]/general)$`, apiKeyHeaderLower, `^(?:[a-z0-9]{32})$`),
	},
	{
		ID:              "scrapingbee-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:app\.scrapingbee\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"app.scrapingbee.com"},
		SecretGroup:     1,
		Source:          "https://www.scrapingbee.com/documentation/",
		Description:     "scrapingbee credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:app\.scrapingbee\.com)$`, `^(?:/api/v1/?)$`, apiKeyFieldSnake, `^(?:[A-Z0-9]{80})$`),
	},
	{
		ID:              "screenshotapi-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:shot\.screenshotapi\.net)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"shot.screenshotapi.net"},
		SecretGroup:     1,
		Source:          "https://docs.screenshotapi.net/",
		Description:     "screenshotapi credential in a complete HTTP(S) API URL with the documented token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:shot\.screenshotapi\.net)$`, `^(?:/(?:v3/)?screenshot)$`, tokenField, `^(?:[A-Z0-9]{7}(?:-[A-Z0-9]{7}){3})$`),
	},
	{
		ID:              "screenshotlayer-access-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.screenshotlayer\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.screenshotlayer.com"},
		SecretGroup:     1,
		Source:          "https://github.com/apilayer/screenshotlayer-API/blob/master/docs/specifications.md",
		Description:     "screenshotlayer credential in a complete HTTP(S) API URL with the documented access_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.screenshotlayer\.com)$`, `^(?:/api/capture)$`, accessKeyField, `^(?:[A-Za-z0-9_]{32})$`),
	},
	{
		ID:              "scrutinizer-access-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:scrutinizer-ci\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"scrutinizer-ci.com"},
		SecretGroup:     1,
		Source:          "https://scrutinizer-ci.com/docs/api/",
		Description:     "scrutinizer-ci credential in a complete HTTP(S) API URL with the documented access_token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:scrutinizer-ci\.com)$`, `^(?:/api/[A-Za-z0-9/_.-]+)$`, "access_token", `^(?:[a-z0-9]{64})$`),
	},
	{
		ID:              "selectpdf-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:selectpdf\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"selectpdf.com"},
		SecretGroup:     1,
		Source:          "https://selectpdf.com/selectpdf-online-rest-api-python-client-library/",
		Description:     "selectpdf credential in a complete HTTP(S) API URL with the documented key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:selectpdf\.com)$`, `^(?:/api2/(?:convert|usage)/?)$`, keyField, `^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`),
	},
	{
		ID:              "semaphore-sms-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:(?:api\.)?semaphore\.co)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"semaphore.co"},
		SecretGroup:     1,
		Source:          "https://semaphore.co/docs",
		Description:     "semaphore credential in a complete HTTP(S) API URL with the documented apikey authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. This is the Semaphore SMS API, not the CI product named in the upstream description. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:(?:api\.)?semaphore\.co)$`, `^(?:/api/v4/[A-Za-z0-9/_-]+)$`, apiKeyFieldLower, `^(?:[a-z0-9]{32})$`),
	},
	{
		ID:              "serpstack-access-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.serpstack\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.serpstack.com"},
		SecretGroup:     1,
		Source:          "https://serpstack.com/documentation",
		Description:     "serpstack credential in a complete HTTP(S) API URL with the documented access_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.serpstack\.com)$`, `^(?:/search)$`, accessKeyField, `^(?:[a-z0-9]{32})$`),
	},
	{
		ID:              "shodan-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.shodan\.io)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.shodan.io"},
		SecretGroup:     1,
		Source:          "https://developer.shodan.io/api",
		Description:     "shodan credential in a complete HTTP(S) API URL with the documented key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.shodan\.io)$`, `^(?:/[A-Za-z0-9/_.-]+)$`, keyField, `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "shortcut-api-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.app\.shortcut\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.app.shortcut.com"},
		SecretGroup:     1,
		Source:          "https://developer.shortcut.com/api/rest/v3/",
		Description:     "shortcut credential in a complete HTTP(S) API URL with the documented token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. The deprecated query carrier remains a confidential historical credential. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.app\.shortcut\.com)$`, `^(?:/api/v3/[A-Za-z0-9/_-]+)$`, tokenField, `^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`),
	},
	{
		ID:              "shortcut-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SHORTCUT_API_TOKEN|Shortcut-Token)["\x27]?[ \t]*[:=][ \t]*["\x27]?([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"shortcut"},
		SecretGroup:     1,
		Source:          "https://developer.shortcut.com/api/rest/v3/",
		Description:     "shortcut credential in the exact documented SHORTCUT_API_TOKEN|Shortcut-Token assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sentry-legacy-auth-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SENTRY_AUTH_TOKEN|SENTRY_TOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([a-f0-9]{64})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"sentry"},
		SecretGroup:     1,
		Source:          "https://cli.sentry.dev/configuration/",
		Description:     "sentry credential in the exact documented SENTRY_AUTH_TOKEN|SENTRY_TOKEN assignment/header role. Public DSNs and OAuth client IDs are excluded; prefixed tokens remain under existing typed rules. Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "signupgenius-user-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.signupgenius\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.signupgenius.com"},
		SecretGroup:     1,
		Source:          "https://developer.signupgenius.com/developer/keybaseddocs",
		Description:     "signupgenius credential in a complete HTTP(S) API URL with the documented user_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.signupgenius\.com)$`, `^(?:/v2/k/[A-Za-z0-9/_-]+/?)$`, "user_key", `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "signalwire-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SIGNALWIRE_API_TOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Za-z0-9]{50})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"signalwire"},
		SecretGroup:     1,
		Source:          "https://signalwire.com/docs/server-sdks/reference/python/rest/client",
		Description:     "signalwire credential in the exact documented SIGNALWIRE_API_TOKEN assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sigopt-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SIGOPT_API_TOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Z0-9]{48})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"sigopt"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/sigopt/sigopt-python/master/README.md",
		Description:     "sigopt credential in the exact documented SIGOPT_API_TOKEN assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "simfin-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:(?:www\.)?simfin\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"simfin.com"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/SimFin/simfin/master/README.md",
		Description:     "simfin credential in a complete HTTP(S) API URL with the documented api-key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:(?:www\.)?simfin\.com)$`, `^(?:/api/v[12]/[A-Za-z0-9/_-]+)$`, apiKeyFieldHyphen, `^(?:[A-Za-z0-9]{32})$`),
	},
	{
		ID:              "simplesat-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:X-Simplesat-Token)["\x27]?[ \t]*[:=][ \t]*["\x27]?([a-z0-9]{40})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"x-simplesat-token"},
		SecretGroup:     1,
		Source:          "https://developer.simplesat.io/api/Simplesat%20API%20(v1)%20OpenAPI.yaml",
		Description:     "simplesat credential in the exact documented X-Simplesat-Token assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "siteleaf-api-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SITELEAF_API_SECRET|Siteleaf\.api_secret)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Za-z0-9]{32})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"siteleaf"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/siteleaf/siteleaf-gem/master/README.md",
		Description:     "siteleaf credential in the exact documented SITELEAF_API_SECRET|Siteleaf\\.api_secret assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "skybiometry-api-secret-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.skybiometry\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.skybiometry.com"},
		SecretGroup:     1,
		Source:          "https://classic.skybiometry.com/documentation/",
		Description:     "skybiometry credential in a complete HTTP(S) API URL with the documented api_secret authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. Domain-authenticated public api_key-only requests are excluded. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.skybiometry\.com)$`, `^(?:/fc/[A-Za-z0-9/_.-]+)$`, "api_secret", `^(?:[a-z0-9]{25,26})$`),
	},
	{
		ID:              "smartsheet-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SMARTSHEET_ACCESS_TOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([A-Za-z0-9]{26}|[A-Za-z0-9]{37})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"smartsheet"},
		SecretGroup:     1,
		Source:          "https://developers.smartsheet.com/api/smartsheet/guides/basics/authentication",
		Description:     "smartsheet credential in the exact documented SMARTSHEET_ACCESS_TOKEN assignment/header role.  Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "smarty-secret-auth-token-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:[a-z0-9-]+\.api\.(?:smartystreets|smarty)\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"smartystreets.com", "smarty.com"},
		SecretGroup:     1,
		Source:          "https://www.smarty.com/docs/account/authentication",
		Description:     "smartystreets credential in a complete HTTP(S) API URL with the documented auth-token authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. Embedded browser keys and standalone auth-id identifiers are public and excluded. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:[a-z0-9-]+\.api\.(?:smartystreets|smarty)\.com)$`, `^(?:/[A-Za-z0-9/_-]*)$`, "auth-token", `^(?:[A-Za-z0-9]{20})$`),
	},
	{
		ID:              "purestake-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{".api.purestake.io"},
		SecretGroup:     1,
		Source:          "https://developer.algorand.org/tutorials/creating-javascript-transaction-purestake-api/",
		Description:     "purestake credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:(?:testnet|mainnet|betanet)-algorand\.api\.purestake\.io)$`, `^(?:/(?:ps[12]|idx2)/[A-Za-z0-9/_-]+)$`, apiKeyHeaderMixed, "", "", false, `^(?:[A-Za-z0-9]{40})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "qase-api-token-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.qase.io"},
		SecretGroup:     1,
		Source:          "https://developers.qase.io/docs/connecting-to-qase",
		Description:     "qase credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.qase\.io)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, "Token", "", "", false, `^(?:[A-Za-z0-9]{40})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "qubole-api-token-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"qubole.com"},
		SecretGroup:     1,
		Source:          "https://docs.qubole.com/en/latest/rest-api/api_overview.html",
		Description:     "qubole credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:(?:us|api|eu|in)\.qubole\.com)$`, `^(?:/api/v[0-9.]+/[A-Za-z0-9/_-]+)$`, "X-AUTH-TOKEN", "", "", false, `^(?:[a-z0-9]{64})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rebrandly-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{rebrandlyHost},
		SecretGroup:     1,
		Source:          rebrandlyAuthSource,
		Description:     "rebrandly credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.rebrandly\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, apiKeyFieldLower, "", "", false, `^(?:[A-Za-z0-9_]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "redhat-pyxis-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"catalog.redhat.com"},
		SecretGroup:     1,
		Source:          "https://github.com/redhat-openshift-ecosystem/openshift-preflight",
		Description:     "redhat-pyxis credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:catalog\.redhat\.com)$`, `^(?:/api/containers/v1/[A-Za-z0-9/_-]+)$`, apiKeyHeaderUpper, "", "", false, `^(?:[a-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "replyio-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.reply.io"},
		SecretGroup:     1,
		Source:          "https://apidocs.reply.io/",
		Description:     "replyio credential in a complete literal curl request to its exact provider host and API path. Includes the current hyphenated primary-doc example form alongside the historical 24-alphanumeric scanner subset. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.reply\.io)$`, `^(?:/v[123]/[A-Za-z0-9/_-]+)$`, apiKeyHeaderTitle, "", "", false, `^(?:[A-Za-z0-9_-]{22,24})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "restpack-access-token-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"restpack.io"},
		SecretGroup:     1,
		Source:          "https://restpack.io/html2pdf/docs",
		Description:     "restpack credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:restpack\.io)$`, `^(?:/api/(?:html2pdf|screenshot)/[A-Za-z0-9/_-]+)$`, "X-Access-Token", "", "", false, `^(?:[A-Za-z0-9]{48})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rocketreach-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.rocketreach.co"},
		SecretGroup:     1,
		Source:          "https://docs.rocketreach.co/reference/account",
		Description:     "rocketreach credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.rocketreach\.co)$`, `^(?:/v[12]/api/[A-Za-z0-9/_-]+)$`, "Api-Key", "", "", false, `^(?:[a-z0-9-]{39})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rocketreach-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.rocketreach\.co)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.rocketreach.co"},
		SecretGroup:     1,
		Source:          "https://docs.rocketreach.co/reference/account",
		Description:     "rocketreach credential in a complete HTTP(S) API URL with the documented api_key authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify. Supports the documented deprecated query form without treating retirement as loss of confidentiality. Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.rocketreach\.co)$`, `^(?:/v[12]/api/[A-Za-z0-9/_-]+)$`, apiKeyFieldSnake, `^(?:[a-z0-9-]{39})$`),
	},
	{
		ID:              "salescookie-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"salescookie.com"},
		SecretGroup:     1,
		Source:          "https://support2.salescookie.com/portal/en/kb/articles/kb-how-can-i-use-the-simplified-transaction-import-rest-api",
		Description:     "salescookie credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:salescookie\.com)$`, `^(?:/app/Api/[A-Za-z]+)$`, "X-ApiKey", "", "", false, `^(?:[A-Za-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "securitytrails-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.securitytrails.com"},
		SecretGroup:     1,
		Source:          "https://docs.securitytrails.com/reference/ping-old-1",
		Description:     "securitytrails credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.securitytrails\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, "APIKEY", "", "", false, `^(?:[A-Za-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sendbird-api-token-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{".sendbird.com"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/sendbird/sendbird-platform-sdk-python/main/README.md",
		Description:     "sendbird credential in a complete literal curl request to its exact provider host and API path. The separate app UUID is public; the required master Api-Token is confidential. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api-[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}\.sendbird\.com)$`, `^(?:/v3/[A-Za-z0-9/_-]+)$`, "Api-Token", "", "", false, `^(?:[a-f0-9]{40})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sendbird-organization-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:SENDBIRDORGANIZATIONAPITOKEN)["\x27]?[ \t]*[:=][ \t]*["\x27]?([a-f0-9]{24})(?:$|[\s"\x27\x60,;}&\]])`,
		Keywords:        []string{"sendbirdorganizationapitoken"},
		SecretGroup:     1,
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/sendbirdorganizationapi",
		Description:     "sendbird credential in the exact documented SENDBIRDORGANIZATIONAPITOKEN assignment/header role. Retains the exact historical organization authentication header from the pinned verifier rather than loose Sendbird context. Literal boundaries and source call/index/operator continuations are checked; candidate widths are scanner/example constraints, not issuance guarantees.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "shipday-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.shipday.com"},
		SecretGroup:     1,
		Source:          "https://docs.shipday.com/reference/authentication",
		Description:     "shipday credential in a complete literal curl request to its exact provider host and API path. Shipday documents a raw dotted token after Basic, not RFC7617 Base64 username/password; the common Basic validator intentionally does not cover it. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.shipday\.com)$`, `^(?:/[A-Za-z0-9/_-]+)$`, authorizationHeader, "Basic ", "", false, `^(?:[A-Za-z0-9.]{11}[A-Za-z0-9]{20})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "shotstack-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.shotstack.io"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/shotstack/shotstack-sdk-node/master/README.md",
		Description:     "shotstack credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.shotstack\.io)$`, `^(?:/(?:stage|v1|edit/(?:stage|v1))/[A-Za-z0-9/_-]+)$`, apiKeyHeaderLower, "", "", false, `^(?:[A-Za-z0-9]{40})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "skrapp-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.skrapp.io"},
		SecretGroup:     1,
		Source:          "https://skrapp.io/api",
		Description:     "skrapp credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.skrapp\.io)$`, `^(?:/(?:api/v2|v3)/[A-Za-z0-9/_-]+)$`, "X-Access-Key", "", "", false, `^(?:[A-Za-z0-9]{42})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "signable-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.signable.co.uk"},
		SecretGroup:     1,
		Source:          "https://developers.signable.app/authentication",
		Description:     "signable credential in a complete literal curl request to its exact provider host and API path. The key is the Basic username; the password is not checked by the service. The historical empty-password form is not covered by common Basic validation. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.signable\.co\.uk)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, authorizationHeader, "", "", true, `^(?:[A-Za-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sigopt-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.sigopt.com"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/sigopt/sigopt-python/master/README.md",
		Description:     "sigopt credential in a complete literal curl request to its exact provider host and API path. The historical API-key-as-Basic-username form has an empty password. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.sigopt\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, authorizationHeader, "", "", true, `^(?:[A-Z0-9]{48})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "refiner-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{refinerHost},
		SecretGroup:     1,
		Source:          refinerAPISource,
		Description:     "refiner credential in a complete literal curl request to its exact provider host and API path. Covers the explicitly documented Basic username with an empty password, distinct from existing Bearer coverage. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.refiner\.io)$`, `^(?:/v1/[A-Za-z0-9/_-]*)$`, authorizationHeader, "", "", true, `^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "sirv-client-secret-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.sirv.com"},
		SecretGroup:     1,
		Source:          "https://sirv.com/help/articles/sirv-rest-api/",
		Description:     "sirv credential in a complete literal curl request to its exact provider host and API path. The clientSecret string is read only from a literal JSON authentication body. Client IDs and public delivery URLs are excluded. The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.sirv\.com)$`, `^(?:/v2/token)$`, "", "", "clientSecret", false, `^(?:[A-Za-z0-9+/]{86}==)$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "postageapp-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.postageapp.com"},
		SecretGroup:     1,
		Source:          postageAppSource,
		Description:     "postageapp credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.postageapp\.com)$`, `^(?:/v\.1\.0/[A-Za-z_]+\.json)$`, "", "", apiKeyFieldSnake, false, `^(?:[A-Za-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "scrapeowl-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"api.scrapeowl.com"},
		SecretGroup:     1,
		Source:          "https://scrapeowl.com/docs/",
		Description:     "scrapeowl credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.scrapeowl\.com)$`, `^(?:/v1/scrape)$`, "", "", apiKeyFieldSnake, false, `^(?:[a-z0-9]{30}|[A-Za-z0-9]{80})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rebrandly-body-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{rebrandlyHost},
		SecretGroup:     1,
		Source:          rebrandlyAuthSource,
		Description:     "rebrandly credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.rebrandly\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, "", "", apiKeyFieldLower, false, `^(?:[A-Za-z0-9_]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "refiner-body-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{refinerHost},
		SecretGroup:     1,
		Source:          refinerAPISource,
		Description:     "refiner credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:api\.refiner\.io)$`, `^(?:/v1/[A-Za-z0-9/_-]*)$`, "", "", apiKeyFieldSnake, false, `^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "semaphore-sms-api-key-request",
		Regex:           providerQuotedCurlPattern,
		Keywords:        []string{"semaphore.co"},
		SecretGroup:     1,
		Source:          "https://semaphore.co/docs",
		Description:     "semaphore credential in a complete literal curl request to its exact provider host and API path.  The documented header/body/user role, not brand proximity, establishes secrecy. Parses literal arguments in either order, rejects expressions and multi-request commands; parsing is bounded to 4096 bytes and 128 words.",
		Validate:        provider6CurlCredential(`^(?:(?:api\.)?semaphore\.co)$`, `^(?:/api/v4/[A-Za-z0-9/_-]+)$`, "", "", apiKeyFieldLower, false, `^(?:[a-z0-9]{32})$`),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "powrbot-secret-key-header",
		Regex:           `\b(?i:Authorization)["\x27]?[ \t]*:[ \t]*["\x27]?(?i:secret-key)[ \t]+([A-Za-z0-9]{40})(?:$|[\s"\x27,;}])`,
		Keywords:        []string{"secret-key"},
		SecretGroup:     1,
		Source:          "https://powrbot.com/cpages/docs/",
		Description:     "Exact secret-key HTTP Authorization scheme documented by powrbot. The required authentication carrier establishes the private key role; bare UUIDs/alphanumerics and provider-name proximity are excluded. Candidate widths follow the pinned scanner and do not prove issuance.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "privacy-api-key-header",
		Regex:           `\b(?i:Authorization)["\x27]?[ \t]*:[ \t]*["\x27]?(?i:api-key)[ \t]+([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})(?:$|[\s"\x27,;}])`,
		Keywords:        []string{apiKeyFieldHyphen},
		SecretGroup:     1,
		Source:          "https://developers.privacy.com/reference/get_cards-1",
		Description:     "Exact api-key HTTP Authorization scheme documented by privacy. The required authentication carrier establishes the private key role; bare UUIDs/alphanumerics and provider-name proximity are excluded. Candidate widths follow the pinned scanner and do not prove issuance.",
		Validate:        validProvider6Opaque,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "securitytrails-api-key-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://(?i:api\.securitytrails\.com)[^\s"\x27<>\x60]+)`,
		Keywords:        []string{"api.securitytrails.com"},
		SecretGroup:     1,
		Source:          "https://docs.securitytrails.com/reference/ping-old-1",
		Description:     "securitytrails credential in a complete HTTP(S) API URL with the documented apikey authentication parameter and exact provider host/path. Public IDs and loose provider proximity do not qualify.  Candidate widths follow pinned scanners/examples, not issuer verification; URI parsing is capped at 16 KiB.",
		ValidateContext: provider6URLContext(`^(?:api\.securitytrails\.com)$`, `^(?:/v1/[A-Za-z0-9/_-]+)$`, apiKeyFieldLower, `^(?:[A-Za-z0-9]{32})$`),
	},
}

// Candidate widths are scanner/example bounds. Only complete literal protocol
// carriers and provider-documented private roles supply confidentiality.
func validProvider6Opaque(s string) bool {
	if !validCarrier3Literal(s) || strings.ContainsAny(s, " \t\r\n$`{}()[]\\") {
		return false
	}
	if strings.HasPrefix(s, "your_") || strings.HasPrefix(s, "YOUR_") || strings.HasPrefix(s, "replace_") {
		return false
	}
	var first byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '_' || c == '=' || c == '.' {
			continue
		}
		if first == 0 {
			first = c
		} else if c != first {
			return true
		}
	}
	return false
}

func provider6URLContext(hostPattern, pathPattern, key, tokenPattern string) func(string, int, int, string) contextValidation {
	host := newLazyRegexp("(?i)" + hostPattern)
	path := newLazyRegexp(pathPattern)
	token := newLazyRegexp(tokenPattern)
	return func(value string, start, end int, secret string) contextValidation {
		if start > 0 && value[start-1] == ':' {
			return contextValidation{}
		}
		candidate, ok := frameConnectionURI(value, start, end, secret)
		if !ok || len(candidate) > maxStructuredCredentialBytes {
			return contextValidation{}
		}
		u, err := url.Parse(candidate)
		if err != nil || u.User != nil || u.Fragment != "" || u.Opaque != "" ||
			!host.MatchString(u.Host) || !path.MatchString(u.Path) {
			return contextValidation{}
		}
		// Query parsing is protocol-specific, not arbitrary recursive decoding.
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return contextValidation{}
		}
		for _, credential := range query[key] {
			if token.MatchString(credential) && validProvider6Opaque(credential) {
				return contextValidation{accepted: true}
			}
		}
		return contextValidation{}
	}
}

// The existing command lexer handles balanced quoted words without executing
// source. Fold only shell continuations between words; quoted newlines remain
// literal body data. Mid-word escapes/concatenations are outside this subset.
func provider6CurlCommand(s string) (string, bool) {
	if !strings.Contains(s, "\n") {
		return s, true
	}
	var folded strings.Builder
	var quote byte
	last := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == '\\' {
			next := i + 1
			if next < len(s) && s[next] == '\r' {
				next++
			}
			if next < len(s) && s[next] == '\n' {
				if i > 0 && s[i-1] != ' ' && s[i-1] != '\t' {
					return "", false
				}
				folded.WriteString(s[last:i])
				folded.WriteByte(' ')
				i = next
				last = next + 1
			}
		}
	}
	if last == 0 {
		return s, true
	}
	folded.WriteString(s[last:])
	return folded.String(), true
}

func provider6CurlWord(raw string) (string, bool) {
	if len(raw) >= 2 && (raw[0] == '\'' || raw[0] == '"') {
		quote := raw[0]
		if raw[len(raw)-1] != quote || strings.ContainsRune(raw[1:len(raw)-1], rune(quote)) {
			return "", false
		}
		raw = raw[1 : len(raw)-1]
	} else if strings.ContainsAny(raw, "\"' \t\r\n\\()[]{}") {
		return "", false
	}
	return raw, raw != "" && !strings.ContainsAny(raw, "$`\x00")
}

func provider6BodyCredential(body, key string) (string, bool) {
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		var fields map[string]json.RawMessage
		if json.Unmarshal([]byte(body), &fields) != nil {
			return "", false
		}
		var secret string
		if json.Unmarshal(fields[key], &secret) != nil {
			return "", false
		}
		return secret, true
	}
	fields, err := url.ParseQuery(body)
	if err != nil || len(fields[key]) != 1 {
		return "", false
	}
	return fields[key][0], true
}

func provider6CurlCredential(hostPattern, pathPattern, header, prefix, bodyKey string, basicUser bool, tokenPattern string) func(string) bool {
	host := newLazyRegexp("(?i)" + hostPattern)
	path := newLazyRegexp(pathPattern)
	token := newLazyRegexp(tokenPattern)
	return func(secret string) bool {
		if len(secret) > 4096 {
			return false
		}
		command, ok := provider6CurlCommand(secret)
		if !ok {
			return false
		}
		var words [128]string
		n, ok := carrier3CommandWords(command, &words)
		if !ok || n < 3 || words[0] != curlCommand {
			return false
		}
		remaining := words[1:n]
		var endpoint, credential, body string
		seenCredential := false
		for len(remaining) > 0 {
			option := remaining[0]
			remaining = remaining[1:]
			switch option {
			case "-s", "-S", curlSilentShowErrorOption, curlSilentOption, curlShowErrorOption, curlCompressedOption, "-L", curlLocationOption, "-G", curlGetOption, "--globoff":
				continue
			case "-H", curlHeaderOption, "-u", curlUserOption, "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, curlDataURLEncodeOption, curlURLOption, "-X", curlRequestOption:
				if len(remaining) == 0 {
					return false
				}
				argument, literal := provider6CurlWord(remaining[0])
				remaining = remaining[1:]
				if !literal {
					return false
				}
				switch option {
				case curlURLOption:
					if endpoint != "" {
						return false
					}
					endpoint = argument
				case "-H", curlHeaderOption:
					name, value, present := strings.Cut(argument, ":")
					if !present {
						return false
					}
					if strings.EqualFold(name, header) {
						if seenCredential {
							return false
						}
						seenCredential = true
						value = strings.TrimSpace(value)
						if basicUser {
							if len(value) < len("Basic ") || !strings.EqualFold(value[:len("Basic ")], "Basic ") || len(value) > 680 {
								return false
							}
							var decoded [512]byte
							n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(value[len("Basic "):]))
							if err != nil || n < 2 {
								return false
							}
							var present bool
							credential, _, present = strings.Cut(string(decoded[:n]), ":")
							if !present {
								return false
							}
						} else {
							if len(value) < len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
								return false
							}
							credential = value[len(prefix):]
						}
						if !validateAuditedAssignmentContext(argument, 0, len(argument), value).accepted {
							return false
						}
					}
				case "-u", curlUserOption:
					if !basicUser || seenCredential {
						return false
					}
					seenCredential = true
					var present bool
					credential, _, present = strings.Cut(argument, ":")
					if !present {
						return false
					}
				case "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, curlDataURLEncodeOption:
					if body != "" || strings.HasPrefix(argument, "@") {
						return false
					}
					body = argument
				}
			default:
				if strings.HasPrefix(option, "-") || endpoint != "" {
					return false
				}
				endpoint, ok = provider6CurlWord(option)
				if !ok {
					return false
				}
			}
		}
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != httpsScheme && u.Scheme != httpScheme) || u.User != nil || u.Fragment != "" ||
			!host.MatchString(u.Host) || !path.MatchString(u.Path) {
			return false
		}
		if bodyKey != "" {
			credential, ok = provider6BodyCredential(body, bodyKey)
			if !ok {
				return false
			}
		}
		return token.MatchString(credential) && validProvider6Opaque(credential)
	}
}
