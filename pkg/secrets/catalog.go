package secrets

import (
	"cmp"
	"slices"
)

// catalogRightBoundary prevents accepting a fixed-length prefix of a longer credential.
// It is consumed only on the right; rules use a zero-width boundary on the left
// so adjacent credentials separated by one delimiter remain independently visible.
const catalogRightBoundary = `(?:$|[^A-Za-z0-9_+/=.\-])`

const gitleaksBaselineSource = "https://github.com/gitleaks/gitleaks/blob/v8.30.1/config/gitleaks.toml"

const (
	curlUserOption                        = "--user"
	curlDataOption                        = "--data"
	curlFailOption                        = "--fail"
	unattendAdministratorPassword         = "AdministratorPassword"
	vaultTokenEnvironment                 = "VAULT_TOKEN"
	unattendPassword                      = "Password"
	httpMethodDelete                      = "DELETE"
	apiV1Path                             = "/api/v1/"
	apiPath                               = "/api/"
	apiKeyHeaderUpper                     = "X-API-KEY"
	curlHeaderOption                      = "--header"
	curlURLOption                         = "--url"
	curlRequestOption                     = "--request"
	authorizationHeader                   = "Authorization"
	apiKeyHeaderTitle                     = "X-Api-Key" // #nosec G101 -- Public HTTP header name, not a credential.
	bearerPrefix                          = "Bearer "
	curlSilentShowErrorOption             = "-sS"
	curlCompressedOption                  = "--compressed"
	curlGetOption                         = "--get"
	curlFormOption                        = "--form"
	providerAssignmentDescription         = "Documented provider SDK/environment secret assignment in the value. Exact private role, quoted/bare literals, full body boundary, and expression-continuation rejection are required."
	providerCurlLinePattern               = `\b(curl[ \t]+(?:\\\r?\n|\\[^\r\n]|[^\\\r\n]){1,1000})(?:$|[\r\n])`
	httpMethodPost                        = "POST"
	providerAccessTokenURLDescription     = "Provider-authentication URL with exact host/path and documented access_token credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded."
	providerAPIKeyURLDescription          = "Provider-authentication URL with exact host/path and documented api_key credential parameter; public IDs, arbitrary nearby values and fragment-only lookalikes are excluded."
	providerAPIKeyRequestDescription      = "Complete provider-bound curl or HTTP request carrying the documented x-api-key account API credential. Bare opaque values, unbound generic headers and public identifiers are excluded."
	apiKeyHeaderMixed                     = "X-API-Key" // #nosec G101 -- Public HTTP header name, not a credential.
	providerBasicUsernameDescription      = "Provider-bound HTTP Basic API credential where the private API key occupies the username and the password is empty. Generic Basic/userinfo intentionally rejects empty passwords, so exact API host and path supply the private role."
	providerBasicDummyPasswordDescription = "Provider-scoped complete HTTP Basic request with an API key in the username. This preserves the documented empty or dummy password forms that the common password-oriented Basic rule deliberately rejects."
	providerAPIKeyUserinfoDescription     = "Exact service API URL with the documented confidential API-key username and empty password. Public user-only URLs are excluded; the bounded key subset is provider-scoped rather than an arbitrary username heuristic."
	providerQuotedCurlPattern             = `(?m)(?:^|[^A-Za-z0-9_.-])(curl[ \t]+(?:\\\r?\n|"[^"]*"|\x27[^\x27]*\x27|[^\r\n]){1,1000})(?:\r?\n|$)`
	providerBasicEmptyPasswordDescription = "Provider-bound HTTP or curl Basic authentication with the private API key as username and an empty password. This form is intentionally not covered by generic Basic; strict single protocol Base64 decoding requires key: and the pinned candidate width."
)

const (
	catalogCIOKeyword        = "cio"
	catalogAzureKeyword      = "azure"
	blynkHost                = "blynk.cloud"
	curlCommand              = "curl"
	placeholderChangeMe      = "changeme"
	httpScheme               = "http"
	curlDataBinaryOption     = "--data-binary"
	curlInsecureOption       = "--insecure"
	proxyField               = "proxy"
	inputField               = "input"
	passwordField            = "password"
	userField                = "user"
	valueField               = "value"
	mysqlScheme              = "mysql"
	clientSecretField        = "client_secret"
	tokenField               = "token"
	cmdkeyCommand            = "cmdkey"
	serverField              = "server"
	hostField                = "host"
	dropboxKeyword           = "dropbox"
	clientIDField            = "client_id"
	httpMethodGet            = "GET"
	apiKeyHeaderLower        = "x-api-key"
	keyField                 = "key"
	awsSecretAccessKeyRuleID = "aws-secret-access-key" // #nosec G101 -- Public detection rule ID, not a credential.
	gcpAPIKeyRuleID          = "gcp-api-key"           // #nosec G101 -- Public detection rule ID, not a credential.
	yandexKeyword            = "yandex"
	githubPATRuleID          = "github-pat"
	linkedinKeyword          = "linkedin"
	genericAPIKeyRuleID      = "generic-api-key"
	abuseIPDBHost            = "api.abuseipdb.com"
	apiKeyFieldLower         = "apikey"
	apiKeyFieldSnake         = "api_key"
	accessKeyField           = "access_key"
	apiKeyFieldCamel         = "apiKey"
	apiTokenField            = "api_token"
	usernameField            = "username"
	calorieNinjasRuleID      = "calorieninjas-api-request"
	cannyRuleID              = "canny-api-request"
	nullLiteral              = "null"
	curlShowErrorOption      = "--show-error"
	basicAuthMode            = "basic"
	honeycombAuthSource      = "https://docs.honeycomb.io/api/authentication"
	apilayerHost             = "apilayer.net"
	apiKeyFieldHyphen        = "api-key"
	mockarooAPISource        = "https://www.mockaroo.com/api/docs"
	onfleetHost              = "onfleet.com"
	onfleetAuthSource        = "https://docs.onfleet.com/reference/authentication"
	packagecloudHost         = "packagecloud.io"
	packagecloudAPISource    = "https://packagecloud.io/docs/api"
	myIntervalsHost          = "api.myintervals.com"
	myIntervalsAuthSource    = "https://www.myintervals.com/api/authentication/"
	paymoHost                = "app.paymoapp.com"
	paymoAuthSource          = "https://raw.githubusercontent.com/paymoapp/api/master/sections/authentication.md"
	paymongoHost             = "api.paymongo.com"
	paymongoAuthSource       = "https://docs.paymongo.com/docs/developer-tools-webhook-setup-management"
	httpsScheme              = "https"
	postageAppSource         = "https://raw.githubusercontent.com/postageapp/postageapp-ruby/master/README.md"
	rebrandlyHost            = "api.rebrandly.com"
	rebrandlyAuthSource      = "https://developers.rebrandly.com/docs/api-key-authentication"
	refinerHost              = "api.refiner.io"
	refinerAPISource         = "https://refiner.io/docs/api/"
	curlLocationOption       = "--location"
	tatumAuthSource          = "https://docs.tatum.io/docs/authentication"
	uptimeRobotHost          = "api.uptimerobot.com"
	vyteHost                 = "api.vyte.in"
	zenscrapeHost            = "app.zenscrape.com"
	zenserpHost              = "app.zenserp.com"
	zipcodebaseHost          = "app.zipcodebase.com"
	typetalkHost             = "typetalk.com"
	curlDataRawOption        = "--data-raw"
	newRelicAPIKeysSource    = "https://docs.newrelic.com/docs/apis/intro-apis/new-relic-api-keys/" // #nosec G101 -- Public documentation URL.
)

const (
	passwdField             = "passwd"
	secretField             = "secret"
	redactedLiteral         = "redacted"
	placeholderLiteral      = "placeholder"
	placeholderYourPassword = "your_password"
	placeholderYourSecret   = "your_secret"
	placeholderYourToken    = "your_token"
	placeholderYourAPIKey   = "your_api_key"
	noneLiteral             = "none"
	curlSilentOption        = "--silent"
	exampleLiteral          = "example"
	undefinedLiteral        = "undefined"
	postgresqlScheme        = "postgresql"
	httpMethodPut           = "PUT"
	httpMethodPatch         = "PATCH"
	httpMethodHead          = "HEAD"
	httpMethodOptions       = "OPTIONS"
	curlDataURLEncodeOption = "--data-urlencode"
)

// nativeRuleSpecs is Tempo's versioned credential catalog, enabled in full by
// default. Provider evidence, rather than upstream inventory parity, determines
// scope. Execution remains native and value-only; generic candidate filters
// apply only to the generic heuristic.
var nativeRuleSpecs = func() []catalogRuleSpec {
	rules := slices.Concat(
		cloudRuleSpecs, developerRuleSpecs, serviceRuleSpecs, structuredRuleSpecs, genericRuleSpecs,
		auditedTokensRules1, auditedTokensRules2, auditedTokensRules3,
		auditedCarriersRules1, auditedCarriersRules2, auditedCarriersRules3,
		auditedEvidenceRules1, auditedEvidenceRules2,
		auditedProviderRules1, auditedProviderRules2, auditedProviderRules3, auditedProviderRules4,
		auditedProviderRules5, auditedProviderRules6, auditedProviderRules7, auditedProviderRules8,
		betterleaksTokensRules1, betterleaksTokensRules2,
		betterleaksCarriersRules1, betterleaksCarriersRules2, betterleaksCarriersRules3, betterleaksCarriersRules4,
		betterleaksEvidenceRules1,
	)
	slices.SortFunc(rules, func(a, b catalogRuleSpec) int { return cmp.Compare(a.ID, b.ID) })
	return rules
}()
