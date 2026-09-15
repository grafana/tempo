package secrets

import (
	"encoding/base64"
	"net/url"
	"strings"
)

// Audited provider carriers use documented private roles inside one value.
// Opaque body widths adapt TruffleHog v3.97.4; proximity-only forms are excluded.
var auditedProviderRules1 = []catalogRuleSpec{
	{
		ID:              "abyssale-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:ABYSSALE_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{40})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"ABYSSALE_API_KEY"},
		SecretGroup:     1,
		Source:          "https://developers.abyssale.com/rest-api/authentication",
		Description:     "Official Abyssale SDKs read ABYSSALE_API_KEY; the credential accesses workspace data and spends generation credits. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "agora-app-certificate",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:AGORA_APP_CERTIFICATE)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"AGORA_APP_CERTIFICATE"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/AgoraIO/Tools/master/DynamicKey/AgoraDynamicKey/go/sample/rtctokenbuilder2/sample.go",
		Description:     "Agora token generator reads AGORA_APP_CERTIFICATE for HMAC token signing, distinct from public AGORA_APP_ID; REST Customer ID/Secret Basic authentication remains covered separately. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "amadeus-client-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:AMADEUS_CLIENT_SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{16})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"AMADEUS_CLIENT_SECRET"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/amadeus4dev/amadeus-python/master/README.rst",
		Description:     "Amadeus official SDK reads AMADEUS_CLIENT_SECRET, distinct from its client ID. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "autodesk-client-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:APS_CLIENT_SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{16})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"APS_CLIENT_SECRET"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/autodesk-platform-services/aps-simple-viewer-nodejs/master/README.md",
		Description:     "Autodesk official sample configures APS_CLIENT_SECRET, distinct from APS_CLIENT_ID. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bitfinex-api-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BFX_API_SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{43})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"BFX_API_SECRET"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/bitfinexcom/bitfinex-api-py/master/README.md",
		Description:     "Bitfinex official client documents BFX_API_SECRET and requires keeping API secrets confidential; key identifiers and signatures alone are excluded. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "braintree-private-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BT_PRIVATE_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([0-9a-f]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"BT_PRIVATE_KEY"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/braintree/braintree_express_example/master/example.env",
		Description:     "Braintree official Express sample names BT_PRIVATE_KEY for its confidential server private key, not BT_PUBLIC_KEY or browser client tokens. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "besttime-private-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:api_key_private)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(pri_[0-9a-f]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"api_key_private"},
		SecretGroup:     1,
		Source:          "https://documentation.besttime.app/",
		Description:     "BestTime specifies api_key_private with pri_ examples as a secret forecast-creation/deletion credential; api_key_public/pub_ read-only keys are explicitly excluded. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "alienvault-otx-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:X-OTX-API-KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9]{64})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"X-OTX-API-KEY"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/AlienVault-OTX/OTX-Python-SDK/master/OTXv2.py",
		Description:     "AlienVault OTX SDK sends X-OTX-API-KEY to authenticate account and pulse operations. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "autopilot-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:autopilotapikey)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([0-9a-f]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"autopilotapikey"},
		SecretGroup:     1,
		Source:          "https://developers.autopilothq.com/",
		Description:     "Autopilot authenticates requests with the provider-specific autopilotapikey header, as confirmed by the pinned verifier. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "aylien-news-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:X-AYLIEN-NewsAPI-Application-Key)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"X-AYLIEN-NewsAPI-Application-Key"},
		SecretGroup:     1,
		Source:          "https://docs.aylien.com/newsapi/v5/",
		Description:     "AYLIEN authenticates its News API with the exact Application-Key header; the separate Application-ID alone is excluded. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "billomat-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:X-BillomatApiKey)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([0-9a-f]{32})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"X-BillomatApiKey"},
		SecretGroup:     1,
		Source:          "https://www.billomat.com/en/api/basics/authentication/",
		Description:     "Billomat calls its personal API key an API password and documents the X-BillomatApiKey header. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "abuseipdb-api-key-url",
		Regex:           `\b(?i:https?://api\.abuseipdb\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{abuseIPDBHost},
		SecretGroup:     0,
		Source:          "https://docs.abuseipdb.com/",
		Description:     "AbuseIPDB explicitly permits key as a query parameter, although it recommends the Key header. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "accuweather-api-key-url",
		Regex:           `\b(?i:https?://dataservice\.accuweather\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"dataservice.accuweather.com"},
		SecretGroup:     0,
		Source:          "https://developer.accuweather.com/documentation/authentication",
		Description:     "Historical AccuWeather apikey URL authentication from both pinned detector versions. Current AccuWeather uses common Authorization Bearer coverage and explicitly requires key confidentiality. Only complete official-host URLs are accepted; opaque values and public identifiers are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "adzuna-api-credentials-url",
		Regex:           `\b(?i:https?://api\.adzuna\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.adzuna.com"},
		SecretGroup:     0,
		Source:          "https://developer.adzuna.com/overview",
		Description:     "Adzuna requires the app_id/app_key pair in API requests; neither an app ID nor a provider-proximate opaque value is sufficient. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "airbrake-user-key-url",
		Regex:           `\b(?i:https?://api\.airbrake\.io)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.airbrake.io"},
		SecretGroup:     0,
		Source:          "https://docs.airbrake.io/docs/devops-tools/api/",
		Description:     "Airbrake user keys authenticate the account project API via key in the API URL; notifier project keys are not matched. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "airvisual-api-key-url",
		Regex:           `\b(?i:https?://api\.airvisual\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.airvisual.com"},
		SecretGroup:     0,
		Source:          "https://api-docs.iqair.com/",
		Description:     "IQAir/AirVisual API requests authenticate using the account key parameter. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "allsports-api-key-url",
		Regex:           `\b(?i:https?://apiv2\.allsportsapi\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"apiv2.allsportsapi.com"},
		SecretGroup:     0,
		Source:          "https://allsportsapi.com/soccer-football-api-documentation",
		Description:     "AllSports documents APIkey as the account authorization code in its API request URL. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "apacta-api-key-url",
		Regex:           `\b(?i:https?://app\.apacta\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"app.apacta.com"},
		SecretGroup:     0,
		Source:          "https://apis.guru/apis/apacta.com",
		Description:     "The published Apacta OpenAPI description specifies api_key URL authentication for private working-hour/material records; the original provider endpoint is no longer reachable. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "api2cart-api-key-url",
		Regex:           `\b(?i:https?://api\.api2cart\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.api2cart.com"},
		SecretGroup:     0,
		Source:          "https://api2cart.com/docs/how-to-start-your-integration-with-api2cart/",
		Description:     "API2Cart grants authenticated store-account API access through its personal api_key URL parameter. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "apiflash-access-key-url",
		Regex:           `\b(?i:https?://api\.apiflash\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.apiflash.com"},
		SecretGroup:     0,
		Source:          "https://apiflash.com/documentation",
		Description:     "ApiFlash explicitly requires its access_key in authenticated GET query strings. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "appsynergy-api-key-url",
		Regex:           `\b(?i:https?://www\.appsynergy\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"www.appsynergy.com"},
		SecretGroup:     0,
		Source:          "https://www.appsynergy.com/content/resources/docs/rest-api.jsp",
		Description:     "AppSynergy REST API uses apiKey with a security role for data/document operations; the documented query carrier is distinct from common Bearer headers. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "apptivo-api-credentials-url",
		Regex:           `\b(?i:https?://api\.apptivo\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.apptivo.com"},
		SecretGroup:     0,
		Source:          "https://www.apptivo.com/developer-api/getting-started/",
		Description:     "Apptivo requires apiKey and accessKey together in each request; a complete URL establishes their private authentication roles. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "artsy-client-secret-url",
		Regex:           `\b(?i:https?://api\.artsy\.net)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.artsy.net"},
		SecretGroup:     0,
		Source:          "https://developers.artsy.net/v2/docs/authentication",
		Description:     "Artsy exchanges client_id and client_secret for authentication tokens; only a complete token-endpoint credential pair is accepted, despite API retirement. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "audd-api-token-url",
		Regex:           `\b(?i:https?://api\.audd\.io)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.audd.io"},
		SecretGroup:     0,
		Source:          "https://docs.audd.io/",
		Description:     "AudD explicitly permits the account api_token in GET query strings and charges usage to the account. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "autoklose-api-token-url",
		Regex:           `\b(?i:https?://api\.autoklose\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.autoklose.com"},
		SecretGroup:     0,
		Source:          "https://api.aklab.xyz/",
		Description:     "Autoklose documents api_token URI authentication and explicitly instructs users to keep the token safe. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "aviationstack-access-key-url",
		Regex:           `\b(?i:https?://api\.aviationstack\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.aviationstack.com"},
		SecretGroup:     0,
		Source:          "https://aviationstack.com/documentation",
		Description:     "Aviationstack authenticates account-quota API requests with the access_key URL parameter. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "billomat-api-key-url",
		Regex:           `\b(?i:https?://[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.billomat\.net)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{".billomat.net"},
		SecretGroup:     0,
		Source:          "https://www.billomat.com/en/api/basics/authentication/",
		Description:     "Billomat explicitly documents api_key query authentication as an alternative to its private-key header. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "browshot-api-key-url",
		Regex:           `\b(?i:https?://api\.browshot\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.browshot.com"},
		SecretGroup:     0,
		Source:          "https://browshot.com/api/documentation",
		Description:     "Browshot requires its account API key as key in all screenshot/account API requests. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "bscscan-api-key-url",
		Regex:           `\b(?i:https?://api\.bscscan\.com)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"api.bscscan.com"},
		SecretGroup:     0,
		Source:          "https://docs.bscscan.com/",
		Description:     "Historical BscScan account API-key query requests are identifiable carriers; blockchain addresses and API responses are not credentials. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "bulbul-api-key-url",
		Regex:           `\b(?i:https?://prod-api\.bulbul\.io)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"prod-api.bulbul.io"},
		SecretGroup:     0,
		Source:          "https://docs.jungleworks.com/bulbul/bulbul-api-details",
		Description:     "Bulbul account API authenticates user-management requests with api_key in its production API URL. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "besttime-private-key-url",
		Regex:           `\b(?i:https?://besttime\.app)/[^\s"'<>\x60]{1,1000}`,
		Keywords:        []string{"besttime.app"},
		SecretGroup:     0,
		Source:          "https://documentation.besttime.app/",
		Description:     "BestTime private API keys are also carried in authenticated forecast URLs and the documented /api/v1/keys/ private-key path. Only the complete official-host URL is a carrier; unrelated URLs, public identifiers, fragments and loose provider proximity are excluded.",
		Validate:        validProvider1URL,
		ValidateContext: validateProvider1URLContext,
	},
	{
		ID:              "apimatic-auth-key",
		Regex:           `(?:\b(?i:Authorization)["']?[ \t]*:[ \t]*["']?(?i:X-Auth-Key)[ \t]+|\bapimatic[ \t]+auth[ \t]+login[ \t]+--auth-key[= \t]+["']?)([A-Za-z0-9_-]{64})(?:$|[ \t\r\n"',;])`,
		Keywords:        []string{"X-Auth-Key", "apimatic"},
		SecretGroup:     1,
		Source:          "https://docs.apimatic.io/account-management/obtaining-auth-keys/",
		Description:     "APIMatic authentication key in its complete Authorization X-Auth-Key header or documented apimatic auth login --auth-key command. Bare opaque values are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bitmex-client-credentials",
		Regex:           `\bbitmex\.bitmex\([ \t]*(?:test[ \t]*=[ \t]*(?:True|False)[ \t]*,[ \t]*)?api_key[ \t]*=[ \t]*(?:"[A-Za-z0-9_-]{24}"|'[A-Za-z0-9_-]{24}')[ \t]*,[ \t]*api_secret[ \t]*=[ \t]*["']([A-Za-z0-9_-]{48})["'][ \t]*\)`,
		Keywords:        []string{"bitmex"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/BitMEX/api-connectors/master/official-http/python-swaggerpy/README.md",
		Description:     "Official BitMEX Python constructor containing literal API key and signing secret in the same invocation. Generic API_SECRET and request signatures are not accepted without provider attribution.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID: "provider-authenticated-api-request", Regex: `\b(?:curl(?:[^\r\n]|\\\r?\n){1,1000}|(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)[ \t]+[^\r\n]+[ \t]+HTTP/1\.[01]\r?\n(?:[^\r\n]+\r?\n){1,16}\r?\n)`,
		Keywords:        []string{curlCommand, httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions},
		SecretGroup:     0,
		Source:          "https://curl.se/docs/manpage.html",
		Description:     "Complete literal curl or HTTP/1 API request for the enumerated provider hosts and private header roles. No provider-key proximity matching, generic X-API-Key heuristic, shell execution or online verification. BitBar is narrowly recognized because its confidential key is the Basic username with an empty password.",
		Validate:        validProvider1Request,
		ValidateContext: withAuditedAssignmentContext(validateProvider1RequestContext),
	},
	{
		ID:              "box-developer-token",
		Regex:           `\b(?:new[ \t]+)?BoxDeveloperTokenAuth\([ \t]*\{[ \t]*["']?token["']?[ \t]*:[ \t]*["']([A-Za-z0-9]{32})["'][ \t]*\}[ \t]*\)`,
		Keywords:        []string{"BoxDeveloperTokenAuth"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/box/box-node-sdk/main/README.md",
		Description:     "Box official SDK developer-token authentication constructor with a literal token property. Bare IDs, token variables and generic object properties are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bitcoinaverage-client-secret",
		Regex:           `(?m:^from[ \t]+bitcoinaverage[ \t]+import[ \t]+RestfulClient[ \t]*\r?\n[ \t]*[A-Za-z_][A-Za-z0-9_]*[ \t]*=[ \t]*RestfulClient\([ \t]*)("[^"\r\n]{1,1000}"|'[^'\r\n]{1,1000}')[ \t]*,[ \t]*(?:"[A-Za-z0-9]{43}"|'[A-Za-z0-9]{43}')[ \t]*\)`,
		Keywords:        []string{"bitcoinaverage"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/bitcoinaverage/api-integration-examples/master/README.md",
		Description:     "Official BitcoinAverage Python import immediately followed by a literal RestfulClient(secret, public) constructor. The x-ba-key/public-key candidate from the upstream detector is excluded; only the explicitly positioned confidential signing secret is captured. Generic imports, unbound classes and source expressions are not carriers.",
		Validate:        validQuotedAuditedPassword2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "betterstack-uptime-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BETTERUPTIME_API_TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{24})(?:$|[ \t\r\n"',;\]}])`,
		Keywords:        []string{"BETTERUPTIME_API_TOKEN"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/BetterStackHQ/terraform-provider-better-uptime/master/docs/index.md",
		Description:     "Official Better Stack Terraform provider documents BETTERUPTIME_API_TOKEN as Sensitive authentication input; generic Bearer coverage alone misses this carrier. Exact named literal carrier only; body widths are scanner constraints, not an issuer guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
}

var provider1URLRoles = map[string]provider1URLRole{
	abuseIPDBHost:                 {pathPrefix: "/api/v2/", key: keyField, body: newLazyRegexp("^(?:[a-z0-9]{80})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"dataservice.accuweather.com": {pathPrefix: "/", key: apiKeyFieldLower, body: newLazyRegexp("^(?:(?:[A-Za-z0-9]{32}|[A-Za-z0-9%]{35}))$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.adzuna.com":              {pathPrefix: "/v1/api/", key: "app_key", body: newLazyRegexp("^(?:[a-z0-9]{32})$"), pairedKey: "app_id", pairedBody: newLazyRegexp("^(?:[a-z0-9]{8})$"), pathSecretPrefix: ""},
	"api.airbrake.io":             {pathPrefix: "/api/v4/", key: keyField, body: newLazyRegexp("^(?:[A-Za-z0-9-]{40})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.airvisual.com":           {pathPrefix: "/v2/", key: keyField, body: newLazyRegexp("^(?:[a-z0-9-]{36})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"apiv2.allsportsapi.com":      {pathPrefix: "/", key: "APIkey", body: newLazyRegexp("^(?:[a-z0-9]{64})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"app.apacta.com":              {pathPrefix: apiV1Path, key: apiKeyFieldSnake, body: newLazyRegexp("^(?:[a-z0-9-]{36})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.api2cart.com":            {pathPrefix: "/v1.1/", key: apiKeyFieldSnake, body: newLazyRegexp("^(?:[0-9a-f]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.apiflash.com":            {pathPrefix: "/v1/urltoimage", key: accessKeyField, body: newLazyRegexp("^(?:[a-z0-9]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"www.appsynergy.com":          {pathPrefix: "/api", key: apiKeyFieldCamel, body: newLazyRegexp("^(?:[a-z0-9]{64})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.apptivo.com":             {pathPrefix: "/app/dao/", key: apiKeyFieldCamel, body: newLazyRegexp("^(?:[a-z0-9-]{36})$"), pairedKey: "accessKey", pairedBody: newLazyRegexp("^(?:[A-Za-z0-9-]{32})$"), pathSecretPrefix: ""},
	"api.artsy.net":               {pathPrefix: "/api/tokens/xapp_token", key: clientSecretField, body: newLazyRegexp("^(?:[A-Za-z0-9]{32})$"), pairedKey: clientIDField, pairedBody: newLazyRegexp("^(?:[A-Za-z0-9]{20})$"), pathSecretPrefix: ""},
	"api.audd.io":                 {pathPrefix: "/", key: apiTokenField, body: newLazyRegexp("^(?:[a-z0-9-]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.autoklose.com":           {pathPrefix: apiPath, key: apiTokenField, body: newLazyRegexp("^(?:[A-Za-z0-9-]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.aviationstack.com":       {pathPrefix: "/v1/", key: accessKeyField, body: newLazyRegexp("^(?:[a-z0-9]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"account.billomat.net":        {pathPrefix: apiPath, key: apiKeyFieldSnake, body: newLazyRegexp("^(?:[0-9a-f]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.browshot.com":            {pathPrefix: apiV1Path, key: keyField, body: newLazyRegexp("^(?:[A-Za-z0-9-]{28})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"api.bscscan.com":             {pathPrefix: "/api", key: apiKeyFieldLower, body: newLazyRegexp("^(?:[A-Z0-9]{34})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	"prod-api.bulbul.io":          {pathPrefix: "/", key: apiKeyFieldSnake, body: newLazyRegexp("^(?:[a-z0-9]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: ""},
	// #nosec G101 -- Provider query parameter and credential-matching pattern, not a credential value.
	"besttime.app": {pathPrefix: apiV1Path, key: "api_key_private", body: newLazyRegexp("^(?:pri_[0-9a-f]{32})$"), pairedKey: "", pairedBody: nil, pathSecretPrefix: "/api/v1/keys/"},
}

var provider1RequestRoles = map[string]provider1RequestRole{
	abuseIPDBHost:          {pathPrefix: "/api/v2/", header: "Key", body: newLazyRegexp("^(?:[a-z0-9]{80})$"), basicUsername: false},
	"api.abyssale.com":     {pathPrefix: "/", header: apiKeyHeaderLower, body: newLazyRegexp("^(?:[A-Za-z0-9]{40})$"), basicUsername: false},
	"api.aeroworkflow.com": {pathPrefix: "/", header: apiKeyFieldLower, body: newLazyRegexp("^(?:[A-Za-z0-9^!?#:*;]{20})$"), basicUsername: false},
	"api.aletheiaapi.com":  {pathPrefix: "/", header: "Key", body: newLazyRegexp("^(?:[A-Z0-9]{32})$"), basicUsername: false},
	"api.ambeedata.com":    {pathPrefix: "/", header: apiKeyHeaderLower, body: newLazyRegexp("^(?:[0-9a-f]{64})$"), basicUsername: false},
	"api.apilayer.com":     {pathPrefix: "/", header: apiKeyFieldLower, body: newLazyRegexp("^(?:[A-Za-z0-9]{32})$"), basicUsername: false},
	"api.apitemplate.io":   {pathPrefix: "/", header: apiKeyHeaderUpper, body: newLazyRegexp("^(?:[A-Za-z0-9]{29,64})$"), basicUsername: false},
	"api.appointedd.com":   {pathPrefix: "/", header: apiKeyHeaderUpper, body: newLazyRegexp("^(?:[A-Za-z0-9+/=]{88})$"), basicUsername: false},
	"app.atera.com":        {pathPrefix: apiPath, header: apiKeyHeaderUpper, body: newLazyRegexp("^(?:[a-z0-9]{32})$"), basicUsername: false},
	"axonaut.com":          {pathPrefix: apiPath, header: "userApiKey", body: newLazyRegexp("^(?:[a-z0-9]{32})$"), basicUsername: false},
	"app.beebole.com":      {pathPrefix: "/graphql", header: apiKeyFieldLower, body: newLazyRegexp("^(?:[a-z0-9]{40})$"), basicUsername: false},
	"blitapp.com":          {pathPrefix: apiPath, header: "API-Key", body: newLazyRegexp("^(?:[A-Za-z0-9_-]{39})$"), basicUsername: false},
	"cloud.bitbar.com":     {pathPrefix: apiPath, header: authorizationHeader, body: newLazyRegexp("^(?:[A-Za-z0-9]{32})$"), basicUsername: true},
}

// These roles are deliberately keyed by complete destinations, not nearby brands.
type provider1URLRole struct {
	pathPrefix       string
	key              string
	body             *lazyRegexp
	pairedKey        string
	pairedBody       *lazyRegexp
	pathSecretPrefix string
}

func provider1URLRoleFor(host string) (provider1URLRole, bool) {
	role, ok := provider1URLRoles[host]
	if ok {
		return role, true
	}
	if strings.HasSuffix(host, ".billomat.net") {
		label := strings.TrimSuffix(host, ".billomat.net")
		if label != "" && !strings.Contains(label, ".") {
			return provider1URLRoles["account.billomat.net"], true
		}
	}
	return provider1URLRole{}, false
}

func validProvider1URL(s string) bool {
	if len(s) > 1000 {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.User != nil || u.Fragment != "" || (!strings.EqualFold(u.Scheme, httpsScheme) && !strings.EqualFold(u.Scheme, httpScheme)) {
		return false
	}
	role, ok := provider1URLRoleFor(strings.ToLower(u.Host))
	if !ok || !provider1PathMatches(u.Path, role.pathPrefix) {
		return false
	}
	if role.pathSecretPrefix != "" && strings.HasPrefix(u.Path, role.pathSecretPrefix) {
		key := strings.TrimPrefix(u.Path, role.pathSecretPrefix)
		return u.RawQuery == "" && role.body.MatchString(key) && validCarrier3Literal(key)
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q[role.key]) != 1 {
		return false
	}
	key := q[role.key][0]
	if !role.body.MatchString(key) || !validCarrier3Literal(key) {
		return false
	}
	return role.pairedKey == "" || len(q[role.pairedKey]) == 1 && role.pairedBody.MatchString(q[role.pairedKey][0]) && validCarrier3Literal(q[role.pairedKey][0])
}

func validateProvider1URLContext(value string, start, end int, _ string) contextValidation {
	if end < len(value) && !strings.ContainsRune(" \t\r\n\"'<>"+string(rune(96)), rune(value[end])) {
		return contextValidation{}
	}
	if start > 0 && (value[start-1] == '"' || value[start-1] == '\'') {
		return contextValidation{accepted: end < len(value) && value[end] == value[start-1]}
	}
	return contextValidation{accepted: true}
}

type provider1RequestRole struct {
	pathPrefix    string
	header        string
	body          *lazyRegexp
	basicUsername bool
}

func provider1ASCII(s string) bool {
	for i := range s {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func validProvider1RequestAuth(target string, headers []string, user string) bool {
	u, err := url.Parse(target)
	if err != nil || u.User != nil || u.Fragment != "" || (!strings.EqualFold(u.Scheme, httpsScheme) && !strings.EqualFold(u.Scheme, httpScheme)) {
		return false
	}
	role, ok := provider1RequestRoles[strings.ToLower(u.Host)]
	if !ok || !provider1PathMatches(u.Path, role.pathPrefix) {
		return false
	}
	found := false
	if user != "" {
		if !role.basicUsername {
			return false
		}
		key, password, present := strings.Cut(user, ":")
		if !present || password != "" || !role.body.MatchString(key) || !validCarrier3Literal(key) {
			return false
		}
		found = true
	}
	for _, header := range headers {
		name, value, present := strings.Cut(header, ":")
		if !present || !provider1ASCII(name) {
			return false
		}
		if strings.EqualFold(name, "Host") && !strings.EqualFold(strings.TrimSpace(value), u.Host) {
			return false
		}
		if !strings.EqualFold(name, role.header) {
			continue
		}
		if found {
			return false
		}
		value = strings.TrimSpace(value)
		key := value
		if role.basicUsername {
			scheme, encoded, present := strings.Cut(value, " ")
			if !present || !strings.EqualFold(scheme, "Basic") {
				return false
			}
			decoded, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(encoded))
			if err != nil {
				return false
			}
			var password string
			key, password, present = strings.Cut(string(decoded), ":")
			if !present || password != "" {
				return false
			}
		}
		if !role.body.MatchString(key) || !validCarrier3Literal(key) {
			return false
		}
		if !role.basicUsername && !validateAuditedAssignmentContext(header, 0, len(header), key).accepted {
			return false
		}
		found = true
	}
	return found
}

// Shell words are inspected, never executed. Only whole quoted literals and
// expansion-free bare words are supported; concatenations and operators fail.
func provider1ShellWord(s string, pos int) (string, int, bool) {
	for pos < len(s) {
		if s[pos] == ' ' || s[pos] == '\t' {
			pos++
			continue
		}
		if strings.HasPrefix(s[pos:], "\\\r\n") {
			pos += 3
			continue
		}
		if strings.HasPrefix(s[pos:], "\\\n") {
			pos += 2
			continue
		}
		break
	}
	if pos == len(s) {
		return "", pos, true
	}
	start := pos
	quote := byte(0)
	if s[pos] == '\'' || s[pos] == '"' {
		quote = s[pos]
		pos++
		start = pos
	}
	for pos < len(s) {
		c := s[pos]
		if quote != 0 {
			if c == quote {
				word := s[start:pos]
				pos++
				if pos < len(s) && s[pos] != ' ' && s[pos] != '\t' && !strings.HasPrefix(s[pos:], "\\\n") && !strings.HasPrefix(s[pos:], "\\\r\n") {
					return "", pos, false
				}
				return word, pos, true
			}
			if c < 0x20 || c == 0x7f || quote == '"' && (c == '$' || c == 96 || c == '\\') {
				return "", pos, false
			}
		} else {
			if c == ' ' || c == '\t' {
				return s[start:pos], pos, true
			}
			if c < 0x20 || c == 0x7f || strings.ContainsRune("\"'\\$(){}[]|;&<>"+string(rune(96)), rune(c)) {
				return "", pos, false
			}
		}
		pos++
	}
	return s[start:pos], pos, quote == 0
}

func validProvider1Curl(s string) bool {
	var words [64]string
	n, pos := 0, 0
	for pos < len(s) {
		word, next, ok := provider1ShellWord(s, pos)
		if !ok || next <= pos || n == len(words) {
			return false
		}
		pos = next
		if word == "" && pos == len(s) {
			break
		}
		words[n] = word
		n++
	}
	if n < 2 || words[0] != curlCommand {
		return false
	}
	remaining := words[1:n]
	var headers [16]string
	headerCount := 0
	target, user := "", ""
	for len(remaining) > 0 {
		arg := remaining[0]
		remaining = remaining[1:]
		switch arg {
		case "-H", curlHeaderOption:
			if len(remaining) == 0 || headerCount == len(headers) {
				return false
			}
			headers[headerCount] = remaining[0]
			remaining = remaining[1:]
			headerCount++
		case "-u", curlUserOption:
			if len(remaining) == 0 || user != "" {
				return false
			}
			user = remaining[0]
			remaining = remaining[1:]
		case curlURLOption:
			if len(remaining) == 0 || target != "" {
				return false
			}
			target = remaining[0]
			remaining = remaining[1:]
		case "-X", curlRequestOption:
			if len(remaining) == 0 {
				return false
			}
			switch remaining[0] {
			case httpMethodGet, httpMethodPost, httpMethodPut, httpMethodPatch, httpMethodDelete, httpMethodHead, httpMethodOptions:
			default:
				return false
			}
			remaining = remaining[1:]
		case "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, curlDataURLEncodeOption, "-F", curlFormOption:
			if len(remaining) == 0 {
				return false
			}
			remaining = remaining[1:]
		case curlSilentOption, curlShowErrorOption, curlLocationOption, curlFailOption, "--include", curlGetOption, curlInsecureOption:
		default:
			if len(arg) > 1 && arg[0] == '-' && strings.Trim(arg[1:], "sSiLfGk") == "" {
				continue
			}
			if target != "" || !strings.HasPrefix(arg, "https://") && !strings.HasPrefix(arg, "http://") {
				return false
			}
			target = arg
		}
	}
	return target != "" && validProvider1RequestAuth(target, headers[:headerCount], user)
}

func validProvider1HTTP(s string) bool {
	// Native extraction trims final LF bytes after the regex has framed the header block.
	s = strings.TrimRight(s, "\r\n")
	first, rest, ok := strings.Cut(s, "\n")
	if !ok {
		return false
	}
	fields := strings.Fields(strings.TrimSuffix(first, "\r"))
	if len(fields) != 3 || (fields[2] != "HTTP/1.1" && fields[2] != "HTTP/1.0") {
		return false
	}
	var headers [16]string
	n := 0
	host := ""
	for {
		line, next, hasNext := strings.Cut(rest, "\n")
		rest = next
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			break
		}
		if n == len(headers) || line[0] == ' ' || line[0] == '\t' {
			return false
		}
		name, value, present := strings.Cut(line, ":")
		if !present || !provider1ASCII(name) {
			return false
		}
		if strings.EqualFold(name, "Host") {
			if host != "" {
				return false
			}
			host = strings.TrimSpace(value)
		}
		headers[n] = line
		n++
		if !hasNext {
			break
		}
	}
	if rest != "" || host == "" {
		return false
	}
	target := fields[1]
	if strings.HasPrefix(target, "/") {
		target = "https://" + host + target
	} else {
		u, err := url.Parse(target)
		if err != nil || !strings.EqualFold(u.Host, host) {
			return false
		}
	}
	return validProvider1RequestAuth(target, headers[:n], "")
}

func validProvider1Request(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	if strings.HasPrefix(s, "curl ") || strings.HasPrefix(s, "curl\t") {
		return validProvider1Curl(s)
	}
	return validProvider1HTTP(s)
}

func validateProvider1RequestContext(value string, start, end int, _ string) contextValidation {
	if start > 0 && strings.ContainsRune("_.-/\\", rune(value[start-1])) {
		return contextValidation{}
	}
	if strings.HasPrefix(value[start:end], curlCommand) && end < len(value) && value[end] != '\n' && value[end] != '\r' {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

func provider1PathMatches(path, prefix string) bool {
	if strings.HasSuffix(prefix, "/") {
		return strings.HasPrefix(path, prefix)
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func validateProvider1AssignmentContext(value string, start, end int, secret string) contextValidation {
	relative := strings.LastIndex(value[start:end], secret)
	return contextValidation{accepted: relative >= 0 && provider1ASCII(value[start:start+relative])}
}
