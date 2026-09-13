package secrets

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

// Candidate constraints informed by TruffleHog v3.97.4 (AGPL-3.0).
// Provider documentation defines credential roles; carrier parsers are native.
var auditedProviderRules3 = []catalogRuleSpec{
	{
		ID:              "demio-api-url",
		Regex:           `\b((?i:https?)://(?i:my\.demio\.com)/api/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"my.demio.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/demio",
		Description:     "Complete demio authenticated API URL with exact host/path and confidential api_secret plus api_key pair. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("api_secret", "[A-Za-z0-9]{10,20}", apiKeyFieldSnake, "[A-Za-z0-9]{32}"),
	},
	{
		ID:              "dnscheck-api-url",
		Regex:           `\b((?i:https?)://(?i:www\.dnscheck\.co)/api/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"www.dnscheck.co"},
		Source:          "https://www.dnscheck.co/api/dns-record-group-monitoring",
		Description:     "Complete dnscheck authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[A-Za-z0-9]{32}", "", ""),
	},
	{
		ID:              "docparser-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.docparser\.com)/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.docparser.com"},
		Source:          "https://dev.docparser.com/",
		Description:     "Complete docparser authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[a-f0-9]{40}", "", ""),
	},
	{
		ID:              "dronahq-api-url",
		Regex:           `\b((?i:https?)://(?i:plugin\.api\.dronahq\.com)/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"plugin.api.dronahq.com"},
		Source:          "https://sdk.dronahq.com/en/v6.3/developer/api-rest-auth.html",
		Description:     "Complete dronahq authenticated API URL with exact host/path and confidential tokenkey parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("tokenkey", "[a-z0-9]{50}", "", ""),
	},
	{
		ID:              "edamam-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.edamam\.com)/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.edamam.com"},
		Source:          "https://api.edamam.com/doc/open-api/nutrition-analysis-v1.yaml",
		Description:     "Complete edamam authenticated API URL with exact host/path and confidential app_key plus app_id pair. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("app_key", "[a-z0-9]{32}", "app_id", "[a-z0-9]{8}"),
	},
	{
		ID:              "elastic-email-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.elasticemail\.com)/v2/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.elasticemail.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/elasticemail",
		Description:     "Complete elastic-email authenticated API URL with exact host/path and confidential apikey parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldLower, "[A-Za-z0-9_-]{96}", "", ""),
	},
	{
		ID:              "exchange-rates-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.exchangeratesapi\.io)/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.exchangeratesapi.io"},
		Source:          "https://exchangeratesapi.io/documentation/",
		Description:     "Complete exchange-rates-api authenticated API URL with exact host/path and confidential access_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(accessKeyField, "[a-z0-9]{32}", "", ""),
	},
	{
		ID:              "extractor-api-auth-url",
		Regex:           `\b((?i:https?)://(?i:extractorapi\.com)/api/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"extractorapi.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/extractorapi",
		Description:     "Complete extractor-api authenticated API URL with exact host/path and confidential apikey parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldLower, "[A-Za-z0-9-]{40}", "", ""),
	},
	{
		ID:              "faceplusplus-api-url",
		Regex:           `\b((?i:https?)://(?i:api-us\.faceplusplus\.com)/facepp/v3/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api-us.faceplusplus.com"},
		Source:          "https://console.faceplusplus.com/documents/5679127",
		Description:     "Complete face-plus-plus authenticated API URL with exact host/path and confidential api_secret plus api_key pair. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("api_secret", "[A-Za-z0-9_-]{32}", apiKeyFieldSnake, "[A-Za-z0-9_-]{32}"),
	},
	{
		ID:              "fastforex-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.fastforex\.io)/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.fastforex.io"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/fastforex",
		Description:     "Complete fast-forex authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[a-z0-9-]{28}", "", ""),
	},
	{
		ID:              "fetchrss-api-url",
		Regex:           `\b((?i:https?)://(?i:fetchrss\.com)/api/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"fetchrss.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/fetchrss",
		Description:     "Complete fetchrss authenticated API URL with exact host/path and confidential auth parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("auth", "[A-Za-z0-9.]{40}", "", ""),
	},
	{
		ID:              "financial-modeling-prep-api-url",
		Regex:           `\b((?i:https?)://(?i:financialmodelingprep\.com)/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"financialmodelingprep.com"},
		Source:          "https://site.financialmodelingprep.com/developer/docs",
		Description:     "Complete financial-modeling-prep authenticated API URL with exact host/path and confidential apikey parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldLower, "[A-Za-z0-9]{32}", "", ""),
	},
	{
		ID:              "fixer-api-url",
		Regex:           `\b((?i:https?)://(?i:data\.fixer\.io)/api/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"data.fixer.io"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/fixerio",
		Description:     "Complete fixer-io authenticated API URL with exact host/path and confidential access_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(accessKeyField, "[A-Za-z0-9]{32}", "", ""),
	},
	{
		ID:              "flightstats-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.flightstats\.com)/flex/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.flightstats.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/flightstats",
		Description:     "Complete flightstats authenticated API URL with exact host/path and confidential appKey plus appId pair. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("appKey", "[a-z0-9]{32}", "appId", "[a-z0-9]{8}"),
	},
	{
		ID:              "flowlu-api-url",
		Regex:           `\b((?i:https?)://(?i:[a-z0-9][a-z0-9-]{0,62}\.flowlu\.com)/api/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"flowlu.com"},
		Source:          "https://www.flowlu.com/api/md/llms-full.md.txt",
		Description:     "Complete flowflu authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[A-Za-z0-9]{51}", "", ""),
	},
	{
		ID:              "fxmarket-api-url",
		Regex:           `\b((?i:https?)://(?i:fxmarketapi\.com)/api[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"fxmarketapi.com"},
		Source:          "https://fxmarketapi.com/documentation",
		Description:     "Complete fxmarket authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[A-Za-z0-9_=-]{20}", "", ""),
	},
	{
		ID:              "geocode-xyz-api-url",
		Regex:           `\b((?i:https?)://(?i:geocode\.xyz)/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"geocode.xyz"},
		Source:          "https://geocode.xyz/api",
		Description:     "Complete geocode authenticated API URL with exact host/path and confidential auth parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query("auth", "[a-z0-9]{28}", "", ""),
	},
	{
		ID:              "geocodify-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.geocodify\.com)/v2/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.geocodify.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/geocodify",
		Description:     "Complete geocodify authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[a-z0-9]{40}", "", ""),
	},
	{
		ID:              "geocodio-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.geocod\.io)/v[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.geocod.io"},
		Source:          "https://www.geocod.io/docs/",
		Description:     "Complete geocodio authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[a-z0-9]{39}", "", ""),
	},
	{
		ID:              "ipify-geolocation-api-url",
		Regex:           `\b((?i:https?)://(?i:geo\.ipify\.org)/api/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"geo.ipify.org"},
		Source:          "https://geo.ipify.org/docs",
		Description:     "Complete geoipifi authenticated API URL with exact host/path and confidential apiKey parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldCamel, "[A-Za-z0-9_]{32}", "", ""),
	},
	{
		ID:              "getgeoapi-auth-url",
		Regex:           `\b((?i:https?)://(?i:api\.getgeoapi\.com)/v2/currency/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.getgeoapi.com"},
		Source:          "https://currency.getgeoapi.com/documentation/",
		Description:     "Complete getgeoapi authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[a-z0-9]{40}", "", ""),
	},
	{
		ID:              "glassnode-api-url",
		Regex:           `\b((?i:https?)://(?i:api\.glassnode\.com)/v1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.glassnode.com"},
		Source:          "https://docs.glassnode.com/basic-api/api",
		Description:     "Complete glassnode authenticated API URL with exact host/path and confidential api_key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(apiKeyFieldSnake, "[A-Za-z0-9]{27}", "", ""),
	},
	{
		ID:              "graphhopper-api-url",
		Regex:           `\b((?i:https?)://(?i:graphhopper\.com)/api/1/[^\s\x22\x27<>\x60]{0,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"graphhopper.com"},
		Source:          "https://docs.graphhopper.com/",
		Description:     "Complete graphhopper authenticated API URL with exact host/path and confidential key parameter. Candidate alphabet/width follows pinned TruffleHog, not an issuance guarantee; complete URL, unique fields, no userinfo/fragment, and bounded query parsing are required.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Query(keyField, "[a-z0-9-]{36}", "", ""),
	},
	{
		ID:              "exchange-rate-api-path-key",
		Regex:           `\b((?i:https?)://(?i:v6\.exchangerate-api\.com)/v6/[^\s\x22\x27<>\x60]{1,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"v6.exchangerate-api.com"},
		Source:          "https://www.exchangerate-api.com/docs/authentication",
		Description:     "Complete provider API URL with the API key in its documented path slot. Bare path identifiers and public endpoints are excluded; candidate width is scanner-derived.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Path(2, "[a-f0-9]{24}"),
	},
	{
		ID:              "flightapi-path-key",
		Regex:           `\b((?i:https?)://(?i:api\.flightapi\.io)/iata/[^\s\x22\x27<>\x60]{1,1000})(?:$|[\s\x22\x27<>\x60])`,
		Keywords:        []string{"api.flightapi.io"},
		Source:          "https://docs.flightapi.io/",
		Description:     "Complete provider API URL with the API key in its documented path slot. Bare path identifiers and public endpoints are excluded; candidate width is scanner-derived.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: providerAudit3Path(2, "[a-z0-9]{24}"),
	},
	{
		ID:              "deputy-oauth-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"Authorization"|'Authorization'|Authorization))[ \t]*[:=][ \t]*(?:"((?i:OAuth)[ \t]+[a-z0-9]{32})"|'((?i:OAuth)[ \t]+[a-z0-9]{32})'|((?i:OAuth)[ \t]+[a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{authorizationHeader},
		Source:          "https://developer.deputy.com/docs/using-oauth-20",
		Description:     "Exact documented Authorization authentication carrier with OAuth scheme. Secret role comes from the header, never provider prose or a bare opaque value. Names/schemes use ASCII case equivalence; body bounds are scanner constraints.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "detectify-api-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"X-Detectify-Key"|'X-Detectify-Key'|X-Detectify-Key))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|'([a-z0-9]{32})'|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"X-Detectify-Key"},
		Source:          "https://developer.detectify.com/",
		Description:     "Exact documented X-Detectify-Key authentication carrier. Secret role comes from the header, never provider prose or a bare opaque value. Names/schemes use ASCII case equivalence; body bounds are scanner constraints.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "fulcrum-api-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"X-ApiToken"|'X-ApiToken'|X-ApiToken))[ \t]*[:=][ \t]*(?:"([a-z0-9]{80})"|'([a-z0-9]{80})'|([a-z0-9]{80}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"X-ApiToken"},
		Source:          "https://docs.fulcrumapp.com/",
		Description:     "Exact documented X-ApiToken authentication carrier. Secret role comes from the header, never provider prose or a bare opaque value. Names/schemes use ASCII case equivalence; body bounds are scanner constraints.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "getresponse-api-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"X-Auth-Token"|'X-Auth-Token'|X-Auth-Token))[ \t]*[:=][ \t]*(?:"((?i:api-key)[ \t]+[A-Za-z0-9]{20,64})"|'((?i:api-key)[ \t]+[A-Za-z0-9]{20,64})'|((?i:api-key)[ \t]+[A-Za-z0-9]{20,64}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"X-Auth-Token"},
		Source:          "https://apidocs.getresponse.com/v3/",
		Description:     "Exact documented X-Auth-Token authentication carrier. Secret role comes from the header, never provider prose or a bare opaque value. Names/schemes use ASCII case equivalence; body bounds are scanner constraints.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "goodday-api-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"gd-api-token"|'gd-api-token'|gd-api-token))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|'([a-z0-9]{32})'|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"gd-api-token"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/goodday",
		Description:     "Exact documented gd-api-token authentication carrier. Secret role comes from the header, never provider prose or a bare opaque value. Names/schemes use ASCII case equivalence; body bounds are scanner constraints.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "dovico-wrap-credential",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:Authorization)[ \t]*:[ \t]*(?i:WRAP)[ \t]+access_token=("client=[a-z0-9]{32}\.[a-z0-9]{1,64}&user_token=[a-z0-9]{32}\.[a-z0-9]{1,64}")(?:$|[\s\x27,;])`,
		Keywords:        []string{authorizationHeader, "WRAP"},
		Source:          "https://timesheet.dovico.com/developer/API_doc/API_Overview.html",
		Description:     "Complete Dovico OAuth WRAP client-secret and user-token header. The client/user distinction comes from exact WRAP fields, not chunk-wide pairing. Suffix cap is local.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "dwolla-app-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"DWOLLA_APP_SECRET"|'DWOLLA_APP_SECRET'|DWOLLA_APP_SECRET))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9-]{50})"|'([A-Za-z0-9-]{50})'|([A-Za-z0-9-]{50}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"DWOLLA_APP_SECRET"},
		Source:          "https://raw.githubusercontent.com/Dwolla/dwolla-v2-node/master/README.md",
		Description:     providerAssignmentDescription,
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "geocodio-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"GEOCODIO_API_KEY"|'GEOCODIO_API_KEY'|GEOCODIO_API_KEY))[ \t]*[:=][ \t]*(?:"([a-z0-9]{39})"|'([a-z0-9]{39})'|([a-z0-9]{39}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"GEOCODIO_API_KEY"},
		Source:          "https://www.geocod.io/docs/",
		Description:     providerAssignmentDescription,
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "detectlanguage-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"detectlanguage\.configuration\.api_key"|'detectlanguage\.configuration\.api_key'|detectlanguage\.configuration\.api_key)|(?:"DetectLanguage\.apiKey"|'DetectLanguage\.apiKey'|DetectLanguage\.apiKey))[ \t]*[:=][ \t]*(?:"([a-z0-9]{32})"|'([a-z0-9]{32})'|([a-z0-9]{32}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{"detectlanguage.configuration.api_key", "DetectLanguage.apiKey"},
		Source:          "https://detectlanguage.com/documentation",
		Description:     providerAssignmentDescription,
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "deepai-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.deepai.org"},
		Source:          "https://deepai.org/docs",
		Description:     "Complete literal curl API request to api.deepai.org/api/ using header:api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.deepai.org", apiPath, "header:api-key", "[a-z0-9-]{36}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "enigma-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.enigma.com"},
		Source:          "https://documentation.enigma.com/guides/graphql/api",
		Description:     "Complete literal curl API request to api.enigma.com/graphql using header:x-api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.enigma.com", "/graphql", "header:x-api-key", "[A-Za-z0-9]{40}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "envoy-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.envoy.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/envoyapikey",
		Description:     "Complete literal curl API request to api.envoy.com/v1/ using header:x-api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.envoy.com", "/v1/", "header:x-api-key", "[A-Za-z0-9]{220}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "findl-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.findl.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/findl",
		Description:     "Complete literal curl API request to api.findl.com/v1.0/ using header:x-api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.findl.com", "/v1.0/", "header:x-api-key", "[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "getsandbox-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"getsandbox.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/getsandbox",
		Description:     "Complete literal curl API request to getsandbox.com/api/1/ using header:api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("getsandbox.com", "/api/1/", "header:api-key", "[a-z0-9-]{40}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "dynalist-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"dynalist.io"},
		Source:          "https://apidocs.dynalist.io/",
		Description:     "Complete literal curl API request to dynalist.io/api/v1/ using json:token. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("dynalist.io", apiV1Path, "json:token", "[A-Za-z0-9_-]{128}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "eagleeye-login-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"login.eagleeyenetworks.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/eagleeyenetworks",
		Description:     "Complete literal curl API request to login.eagleeyenetworks.com/g/aaa/authenticate using json:password. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("login.eagleeyenetworks.com", "/g/aaa/authenticate", "json:password", "[A-Za-z0-9]{15}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "formcraft-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"formcrafts.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/formcraft",
		Description:     "Complete literal curl API request to formcrafts.com/api/v1/ using basic. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("formcrafts.com", apiV1Path, basicAuthMode, "[a-z0-9]{16}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "geckoboard-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.geckoboard.com"},
		Source:          "https://developer.geckoboard.com/",
		Description:     "Complete literal curl API request to api.geckoboard.com/ using basic. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.geckoboard.com", "/", basicAuthMode, "[A-Za-z0-9]{32,64}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "docparser-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.docparser.com"},
		Source:          "https://dev.docparser.com/",
		Description:     "Complete literal curl API request to api.docparser.com/v1/ using basic. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.docparser.com", "/v1/", basicAuthMode, "[a-f0-9]{40}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "fastforex-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.fastforex.io"},
		Source:          "https://fastforex.readme.io/reference/introduction",
		Description:     "Complete literal curl API request to api.fastforex.io/ using header:x-api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.fastforex.io", "/", "header:x-api-key", "[a-z0-9-]{28}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "fetchrss-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"fetchrss.com"},
		Source:          "https://fetchrss.com/developers",
		Description:     "Complete literal curl API request to fetchrss.com/api/v2/ using header:api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("fetchrss.com", "/api/v2/", "header:api-key", "[A-Za-z0-9.]{40}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "glassnode-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.glassnode.com"},
		Source:          "https://docs.glassnode.com/basic-api/api",
		Description:     "Complete literal curl API request to api.glassnode.com/v1/ using header:x-api-key. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.glassnode.com", "/v1/", "header:x-api-key", "[A-Za-z0-9]{27}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "financial-modeling-prep-api-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"financialmodelingprep.com"},
		Source:          "https://site.financialmodelingprep.com/developer/docs",
		Description:     "Complete literal curl API request to financialmodelingprep.com/ using header:apikey. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("financialmodelingprep.com", "/", "header:apikey", "[A-Za-z0-9]{32}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "elastic-email-api-header-request",
		Regex:           providerCurlLinePattern,
		Keywords:        []string{"api.elasticemail.com"},
		Source:          "https://raw.githubusercontent.com/ElasticEmail/elasticemail-go/master/README.md",
		Description:     "Complete literal curl API request to api.elasticemail.com/v4/ using header:x-elasticemail-apikey. The endpoint and actual parsed auth operand must occur in the same command, not unrelated provider proximity. No shell expansion, command separators, arbitrary decoding, or live verification.",
		Validate:        providerAudit3Curl("api.elasticemail.com", "/v4/", "header:x-elasticemail-apikey", "[A-Za-z0-9_-]{96}"),
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "godaddy-sso-key-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:"Authorization"|'Authorization'|Authorization))[ \t]*[:=][ \t]*(?:"((?i:sso-key)[ \t]+[A-Za-z0-9_]{35}(?:[A-Za-z0-9_]{2})?:[A-Za-z0-9]{22})"|'((?i:sso-key)[ \t]+[A-Za-z0-9_]{35}(?:[A-Za-z0-9_]{2})?:[A-Za-z0-9]{22})'|((?i:sso-key)[ \t]+[A-Za-z0-9_]{35}(?:[A-Za-z0-9_]{2})?:[A-Za-z0-9]{22}))(?:$|[\s\x22\x27\x60,;}\]])`,
		Keywords:        []string{authorizationHeader, "sso-key"},
		Source:          "https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/godaddy/v1/godaddy.go",
		Description:     "Complete legacy GoDaddy sso-key authentication header with key and secret. Production (35) and OTE (37) identifiers share one rule; neither public identifier alone nor a bare 22-character value qualifies.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "duo-integration-secret-pair",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?i:(?:"ikey"|'ikey'|ikey))[ \t]*[:=][ \t]*(?:"(?:DI[A-Z0-9]{18})"|'(?:DI[A-Z0-9]{18})'|(?:DI[A-Z0-9]{18}))[ \t\r\n,;]{1,16}(?i:(?:"skey"|'skey'|skey))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|'([A-Za-z0-9]{40})'|([A-Za-z0-9]{40}))|(?i:(?:"skey"|'skey'|skey))[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|'([A-Za-z0-9]{40})'|([A-Za-z0-9]{40}))[ \t\r\n,;]{1,16}(?i:(?:"ikey"|'ikey'|ikey))[ \t]*[:=][ \t]*(?:"(?:DI[A-Z0-9]{18})"|'(?:DI[A-Z0-9]{18})'|(?:DI[A-Z0-9]{18})))(?:$|[\s\x22\x27,;}\]])`,
		Keywords:        []string{"ikey", "skey"},
		Source:          "https://duo.com/docs/authapi",
		Description:     "Adjacent, explicitly named Duo ikey/skey credential fields in one value, in either order. The DI integration-key marker types the pair; the public integration key alone is not a secret. Both field literals must be complete, without loose provider proximity.",
		Validate:        validProviderAudit3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "gengo-private-key-call",
		Regex:           `\bGengo\([ \t\r\n]*public_key[ \t]*=[ \t]*(?:"[0-9A-Za-z\[\]\-(){}|_^@$=~]{64}"|'[0-9A-Za-z\[\]\-(){}|_^@$=~]{64}')[ \t\r\n]*,[ \t\r\n]*private_key[ \t]*=[ \t]*("[0-9A-Za-z\[\]\-(){}|_^@$=~]{64}"|'[0-9A-Za-z\[\]\-(){}|_^@$=~]{64}')[ \t\r\n]*(?:,[ \t\r\n]*sandbox[ \t]*=[ \t]*(?:True|False)[ \t\r\n]*)?\)`,
		Keywords:        []string{"Gengo", "private_key"},
		Source:          "https://raw.githubusercontent.com/gengo/gengo-python/master/README.rst",
		Description:     "Complete documented Gengo SDK construction with public_key and quoted private_key. Only the private half is captured. Quoted punctuation-rich legacy private keys are allowed; double-quoted interpolation and all bare source references are excluded.",
		Validate:        validProviderAudit3QuotedSecret,
		ValidateContext: validateAuditedAssignmentContext,
	},
}

// These validators operate only on this audit slice's complete carriers. Body
// bounds are offline candidate constraints, never claims of active issuance.
func validProviderAudit3Literal(s string) bool {
	if s == "" || len(s) > maxStructuredCredentialBytes || strings.ContainsAny(s, "\x00\r\n") {
		return false
	}
	for _, marker := range []string{"$" + "{", "$" + "(", "{{", "}}", "<", ">"} {
		if strings.Contains(s, marker) {
			return false
		}
	}
	for _, placeholder := range []string{passwordField, secretField, tokenField, redactedLiteral, placeholderLiteral, placeholderYourAPIKey, nullLiteral, undefinedLiteral} {
		if strings.EqualFold(strings.Trim(s, "\"'"), placeholder) {
			return false
		}
	}
	return strings.Trim(s, "xX* .\"'") != ""
}

func validProviderAudit3QuotedSecret(s string) bool {
	if len(s) < 2 || s[0] != s[len(s)-1] || s[0] != '\'' && s[0] != '"' {
		return false
	}
	if s[0] == '"' && (strings.ContainsRune(s, '$') || strings.ContainsRune(s, 96)) {
		return false
	}
	return validProviderAudit3Literal(s[1 : len(s)-1])
}

func providerAudit3URL(value string, start, end int, secret string) (*url.URL, bool) {
	relative := strings.Index(value[start:end], secret)
	if relative < 0 || len(secret) > maxStructuredCredentialBytes {
		return nil, false
	}
	start += relative
	end = start + len(secret)
	framed, ok := frameConnectionURI(value, start, end, secret)
	if !ok || !validProviderAudit3Literal(framed) {
		return nil, false
	}
	u, err := url.Parse(framed)
	if err != nil || !strings.EqualFold(u.Scheme, httpsScheme) && !strings.EqualFold(u.Scheme, httpScheme) || u.User != nil || u.Fragment != "" || u.Host == "" || u.Host != u.Hostname() || strings.Count(u.RawQuery, "&") > 32 {
		return nil, false
	}
	return u, true
}

func providerAudit3Query(key, pattern, companion, companionPattern string) func(string, int, int, string) contextValidation {
	credential := newLazyRegexp("^(?:" + pattern + ")$")
	var paired *lazyRegexp
	if companion != "" {
		paired = newLazyRegexp("^(?:" + companionPattern + ")$")
	}
	return func(value string, start, end int, secret string) contextValidation {
		u, ok := providerAudit3URL(value, start, end, secret)
		if !ok {
			return contextValidation{}
		}
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil || len(q[key]) != 1 || !credential.MatchString(q.Get(key)) || !validProviderAudit3Literal(q.Get(key)) {
			return contextValidation{}
		}
		if paired != nil && (len(q[companion]) != 1 || !paired.MatchString(q.Get(companion))) {
			return contextValidation{}
		}
		return contextValidation{accepted: true}
	}
}

func providerAudit3Path(index int, pattern string) func(string, int, int, string) contextValidation {
	credential := newLazyRegexp("^(?:" + pattern + ")$")
	return func(value string, start, end int, secret string) contextValidation {
		u, ok := providerAudit3URL(value, start, end, secret)
		if !ok {
			return contextValidation{}
		}
		rest := u.Path
		for range index {
			_, remainder, found := strings.Cut(rest, "/")
			if !found {
				return contextValidation{}
			}
			rest = remainder
		}
		part, _, found := strings.Cut(rest, "/")
		return contextValidation{accepted: found && credential.MatchString(part) && validProviderAudit3Literal(part)}
	}
}

// Line continuation removal follows shell quotation: inside single quotes the
// backslash and newline are literal, so the credential validator rejects them.
func providerAudit3CurlLines(s string) string {
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && quote != '\'' && i+1 < len(s) {
			if s[i+1] == '\n' {
				i++
				continue
			}
			if i+2 < len(s) && s[i+1] == '\r' && s[i+2] == '\n' {
				i += 2
				continue
			}
			b.WriteByte(c)
			i++
			b.WriteByte(s[i])
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
		b.WriteByte(c)
	}
	return b.String()
}

func providerAudit3JSONSecret(body, key string) (string, bool) {
	d := json.NewDecoder(strings.NewReader(body))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return "", false
	}
	seen := make(map[string]bool)
	var secret, username string
	for d.More() {
		token, err = d.Token()
		name, ok := token.(string)
		if err != nil || !ok || seen[name] || len(seen) >= 32 {
			return "", false
		}
		seen[name] = true
		var raw json.RawMessage
		if err = d.Decode(&raw); err != nil {
			return "", false
		}
		switch name {
		case key:
			if json.Unmarshal(raw, &secret) != nil {
				return "", false
			}
		case usernameField:
			if json.Unmarshal(raw, &username) != nil {
				return "", false
			}
		}
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return "", false
	}
	if _, err = d.Token(); err != io.EOF || key == passwordField && !validProviderAudit3Literal(username) {
		return "", false
	}
	return secret, validProviderAudit3Literal(secret)
}

func providerAudit3Curl(host, path, mode, pattern string) func(string) bool {
	credential := newLazyRegexp("^(?:" + pattern + ")$")
	return func(s string) bool {
		if len(s) > maxStructuredCredentialBytes {
			return false
		}
		s = providerAudit3CurlLines(s)
		word, rest, ok := nextCurlLiteralWord1(s)
		if !ok || word != curlCommand {
			return false
		}
		var target, secret, body, contentType, method string
		var foundCredential, foundBody bool
		for count := 0; strings.TrimSpace(rest) != ""; count++ {
			if count >= 64 {
				return false
			}
			word, rest, ok = nextCurlLiteralWord1(rest)
			if !ok {
				return false
			}
			option, operand, inline := word, "", false
			if strings.HasPrefix(word, "--") {
				option, operand, inline = strings.Cut(word, "=")
			}
			switch option {
			case "-s", curlSilentOption, "-S", curlShowErrorOption, "-f", curlFailOption:
				if inline {
					return false
				}
				continue
			case "-H", curlHeaderOption, "-u", curlUserOption, "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, "-X", curlRequestOption, curlURLOption:
				if !inline {
					operand, rest, ok = nextCurlLiteralWord1(rest)
					if !ok {
						return false
					}
				}
			default:
				if inline || target != "" || !strings.HasPrefix(word, "https://") && !strings.HasPrefix(word, "http://") {
					return false
				}
				target = word
				continue
			}
			switch option {
			case curlURLOption:
				if target != "" {
					return false
				}
				target = operand
			case "-X", curlRequestOption:
				if method != "" {
					return false
				}
				method = operand
			case "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption:
				if foundBody || strings.HasPrefix(operand, "@") {
					return false
				}
				body, foundBody = operand, true
			case "-u", curlUserOption:
				if mode != basicAuthMode || foundCredential {
					return false
				}
				user, password, present := strings.Cut(operand, ":")
				if !present || password != "" {
					return false
				}
				secret, foundCredential = user, true
			case "-H", curlHeaderOption:
				name, field, present := strings.Cut(operand, ":")
				if !present || strings.ContainsAny(operand, "\r\n") {
					return false
				}
				name, field = strings.TrimSpace(name), strings.TrimSpace(field)
				if strings.EqualFold(name, "Host") {
					return false // Do not attribute a request whose authority is overridden.
				}
				if strings.EqualFold(name, "Content-Type") {
					if contentType != "" {
						return false
					}
					contentType, _, _ = strings.Cut(field, ";")
				}
				if strings.HasPrefix(mode, "header:") && strings.EqualFold(name, strings.TrimPrefix(mode, "header:")) {
					if foundCredential {
						return false
					}
					secret, foundCredential = field, true
				} else if mode == basicAuthMode && strings.EqualFold(name, authorizationHeader) {
					if foundCredential {
						return false
					}
					scheme, encoded, present := strings.Cut(field, " ")
					if !present || !strings.EqualFold(scheme, "Basic") {
						return false
					}
					decoded, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(encoded))
					if err != nil {
						return false
					}
					user, password, present := strings.Cut(string(decoded), ":")
					if !present || password != "" {
						return false
					}
					secret, foundCredential = user, true
				}
			}
		}
		u, err := url.Parse(target)
		if err != nil || u.User != nil || u.Fragment != "" || !strings.EqualFold(u.Host, host) || !strings.HasPrefix(u.Path, path) || u.Scheme != httpsScheme && u.Scheme != httpScheme {
			return false
		}
		if strings.HasPrefix(mode, "json:") {
			if !foundBody || !strings.EqualFold(strings.TrimSpace(contentType), "application/json") || method != "" && method != httpMethodPost {
				return false
			}
			secret, foundCredential = providerAudit3JSONSecret(body, strings.TrimPrefix(mode, "json:"))
		}
		return foundCredential && credential.MatchString(secret) && validProviderAudit3Literal(secret)
	}
}
