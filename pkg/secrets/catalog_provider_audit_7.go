package secrets

import (
	"encoding/base64"
	"strings"
)

// Candidate constraints are informed by Titus v1.2.9 (LICENSE.titus) and
// TruffleHog v3.97.4 (AGPL-3.0). Provider authentication roles are documented
// per rule; opaque bodies are never attributed by nearby provider prose.
var auditedProviderRules7 = []catalogRuleSpec{
	{
		ID:              "snipcart-secret-basic",
		Regex:           `(?:(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{102}={2})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{102}={2})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{102}={2})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{102}={2})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{102}={2})')[ \t\\\r\n]{1,16}(?:"(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-u|--user)[ \t]+(?:"([A-Za-z0-9_]{75}:)"|'([A-Za-z0-9_]{75}:)'|([A-Za-z0-9_]{75}:))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-u|--user)[ \t]+(?:"([A-Za-z0-9_]{75}:)"|'([A-Za-z0-9_]{75}:)'|([A-Za-z0-9_]{75}:))[ \t\\\r\n]{1,16}(?:"(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://app\.snipcart\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"app.snipcart.com"},
		Source:          "https://docs.snipcart.com/v3/api-reference/authentication",
		Description:     providerBasicEmptyPasswordDescription,
		Validate:        validProvider7UsernameBasic,
		ValidateContext: provider7RequestContext(`app\.snipcart\.com`, authorizationHeader, true),
	},
	{
		ID:              "snyk-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Nn][Yy][Kk]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Nn][Yy][Kk]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Nn][Yy][Kk]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})'|([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SNYK_TOKEN"},
		Source:          "https://docs.snyk.io/snyk-cli/commands/auth",
		Description:     "SNYK_TOKEN is the documented CI/CD private API-token environment variable. UUID-shaped bodies are limited to the pinned legacy candidate form; unrelated UUIDs and source references are not credentials.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "snyk-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api\.)?snyk\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"snyk.io"},
		Source:          "https://snyk.docs.apiary.io/#reference/users/my-user-details/get-my-details",
		Description:     "Complete Snyk API request using its legacy Authorization: token private API-token carrier; opaque UUIDs away from this carrier are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:api\.)?snyk\.io`, authorizationHeader, false),
	},
	{
		ID:              "solarwinds-apm-service-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Ww]_[Aa][Pp][Mm]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Kk][Ee][Yy]"|'[Ss][Ww]_[Aa][Pp][Mm]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Kk][Ee][Yy]'|[Ss][Ww]_[Aa][Pp][Mm]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9_-]{71}:[A-Za-z0-9_.-]{1,128})"|'([A-Za-z0-9_-]{71}:[A-Za-z0-9_.-]{1,128})'|([A-Za-z0-9_-]{71}:[A-Za-z0-9_.-]{1,128})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SW_APM_SERVICE_KEY"},
		Source:          "https://documentation.solarwinds.com/en/success_center/observability/content/configure/services/java/configure.htm",
		Description:     "SW_APM_SERVICE_KEY contains the documented private ingestion API token followed by a colon and service name. The 71-character token comes from the pinned detector; the service-name cap is local.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "speechtext-api-url",
		Regex:       `\bhttps?://api\.speechtext\.ai/(?:recognize|results)\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}key=([a-fA-F0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.speechtext.ai"},
		SecretGroup: 1,
		Source:      "https://speechtext.ai/speech-api-docs",
		Description: "Complete SpeechText recognize/results URL with its documented secret key query parameter. Public task IDs and bare hashes are not matched.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "splunk-observability-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Xx]-[Ss][Ff]-[Tt][Oo][Kk][Ee][Nn]"|'[Xx]-[Ss][Ff]-[Tt][Oo][Kk][Ee][Nn]'|[Xx]-[Ss][Ff]-[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{22})"|'([A-Za-z0-9]{22})'|([A-Za-z0-9]{22})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"X-Sf-Token"},
		Source:          "https://dev.splunk.com/observability/reference/api/org_tokens/latest",
		Description:     "Literal X-Sf-Token header used to authorize Splunk Observability/SignalFx requests. Header role is private; 22-alphanumeric body is the pinned TruffleHog candidate constraint.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "spoonacular-api-url",
		Regex:       `\bhttps?://api\.spoonacular\.com/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}apiKey=([a-z0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.spoonacular.com"},
		SecretGroup: 1,
		Source:      "https://spoonacular.com/food-api/docs",
		Description: "Complete Spoonacular API URL carrying an API key in its authentication query parameter; recipe IDs and opaque bare strings are excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:          "sportmonks-api-url",
		Regex:       `\bhttps?://(?:api|soccer)\.sportmonks\.com/api/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}api_token=([A-Za-z0-9]{60})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"sportmonks.com"},
		SecretGroup: 1,
		Source:      "https://docs.sportmonks.com/v3/welcome/authentication",
		Description: "Complete Sportmonks API URL with its confidential api_token query parameter. Both current api and historical soccer hosts are retained; tokens must not be exposed in browser code.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "sportmonks-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{60})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{60})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{60})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{60})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{60})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api|soccer)\.sportmonks\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"sportmonks.com"},
		Source:          "https://docs.sportmonks.com/v3/welcome/authentication",
		Description:     "Complete Sportmonks request using the documented raw-token Authorization header, distinct from generic Bearer.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:api|soccer)\.sportmonks\.com`, authorizationHeader, false),
	},
	{
		ID:              "sslmate-secret-basic",
		Regex:           `(?:(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{50}={2})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{50}={2})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{50}={2})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{50}={2})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{50}={2})')[ \t\\\r\n]{1,16}(?:"(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-u|--user)[ \t]+(?:"([A-Za-z0-9_]{36}:)"|'([A-Za-z0-9_]{36}:)'|([A-Za-z0-9_]{36}:))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-u|--user)[ \t]+(?:"([A-Za-z0-9_]{36}:)"|'([A-Za-z0-9_]{36}:)'|([A-Za-z0-9_]{36}:))[ \t\\\r\n]{1,16}(?:"(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://sslmate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"sslmate.com"},
		Source:          "https://sslmate.com/help/reference/apiv3",
		Description:     providerBasicEmptyPasswordDescription,
		Validate:        validProvider7UsernameBasic,
		ValidateContext: provider7RequestContext(`sslmate\.com`, authorizationHeader, true),
	},
	{
		ID:              "stability-sdk-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Tt][Aa][Bb][Ii][Ll][Ii][Tt][Yy]_[Kk][Ee][Yy]"|'[Ss][Tt][Aa][Bb][Ii][Ll][Ii][Tt][Yy]_[Kk][Ee][Yy]'|[Ss][Tt][Aa][Bb][Ii][Ll][Ii][Tt][Yy]_[Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"(sk-[A-Za-z0-9]{48})"|'(sk-[A-Za-z0-9]{48})'|(sk-[A-Za-z0-9]{48})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"STABILITY_KEY"},
		Source:          "https://raw.githubusercontent.com/Stability-AI/stability-sdk/main/README.md",
		Description:     "STABILITY_KEY is the official SDK environment variable for the private Stability API key. Shared sk- prefix alone does not identify this provider; the 48-character suffix is the pinned Titus constraint.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "statuscake-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Tt][Aa][Tt][Uu][Ss][Cc][Aa][Kk][Ee]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Tt][Aa][Tt][Uu][Ss][Cc][Aa][Kk][Ee]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Tt][Aa][Tt][Uu][Ss][Cc][Aa][Kk][Ee]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{20})"|'([A-Za-z0-9]{20})'|([A-Za-z0-9]{20})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"STATUSCAKE_API_TOKEN"},
		Source:          "https://raw.githubusercontent.com/StatusCakeDev/terraform-provider-statuscake/master/docs/index.md",
		Description:     "STATUSCAKE_API_TOKEN is the provider-supported environment variable for authenticated API operations. The 20-character body is a historical scanner constraint, not a provider issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "statuspage-api-url",
		Regex:       `\bhttps?://api\.statuspage\.io/v1/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}api_key=((?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.statuspage.io"},
		SecretGroup: 1,
		Source:      "https://developer.statuspage.io/",
		Description: "Complete Statuspage API URL with documented api_key authentication. Modern 64-hex examples and pinned legacy UUID candidates are confined to the private carrier.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "statuspage-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:[Oo][Aa][Uu][Tt][Hh] )?(?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:[Oo][Aa][Uu][Tt][Hh] )?(?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:[Oo][Aa][Uu][Tt][Hh] )?(?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:[Oo][Aa][Uu][Tt][Hh] )?(?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:[Oo][Aa][Uu][Tt][Hh] )?(?:[a-f0-9]{64}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))')[ \t\\\r\n]{1,16}(?:"(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.statuspage\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.statuspage.io"},
		Source:          "https://developer.statuspage.io/",
		Description:     "Complete Statuspage API request with documented OAuth API-key authorization; the pinned historical bare Authorization UUID form is also retained.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.statuspage\.io`, authorizationHeader, false),
	},
	{
		ID:              "statuspal-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{32})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{32})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9]{32})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:www\.)?statuspal\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"statuspal.io"},
		Source:          "https://www.statuspal.io/api-docs",
		Description:     "Complete Statuspal request with its private raw Authorization API key; public status-page IDs and arbitrary nearby values are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:www\.)?statuspal\.io`, authorizationHeader, false),
	},
	{
		ID:          "stockdata-api-url",
		Regex:       `\bhttps?://api\.stockdata\.org/v1/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}api_token=([A-Za-z0-9]{40})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.stockdata.org"},
		SecretGroup: 1,
		Source:      "https://www.stockdata.org/documentation",
		Description: "Complete StockData URL with the documented account API-token authentication parameter; stock symbols and other public query fields are excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "stormboard-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{40})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{40})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{40})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{40})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{40})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.stormboard\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.stormboard.com"},
		Source:          "https://api.stormboard.com/docs/auth",
		Description:     "Complete Stormboard API request with the documented private X-API-Key header. The 40-hex shape is only a scanner constraint inside the authenticated request.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.stormboard\.com`, apiKeyHeaderMixed, false),
	},
	{
		ID:              "stormglass-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9-]{73})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9-]{73})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9-]{73})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9-]{73})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([A-Za-z0-9-]{73})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.stormglass\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.stormglass.io"},
		Source:          "https://docs.stormglass.io/",
		Description:     "Complete Stormglass request with its raw Authorization API key. This is not a Bearer value, so it needs an explicit provider-bound carrier.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.stormglass\.io`, authorizationHeader, false),
	},
	{
		ID:          "strava-oauth-secret-url",
		Regex:       `\bhttps?://(?:www\.)?strava\.com/oauth/token\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}(?:client_secret|refresh_token)=([a-z0-9]{40})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"strava.com"},
		SecretGroup: 1,
		Source:      "https://developers.strava.com/docs/authentication/",
		Description: "Complete Strava token-exchange URL carrying a private client_secret or refresh_token in its documented query parameter; client IDs and authorize URLs do not match.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "streak-secret-basic",
		Regex:           `(?:(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{44}={0})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{44}={0})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{44}={0})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{44}={0})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [A-Za-z0-9+/]{44}={0})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-u|--user)[ \t]+(?:"([A-Fa-f0-9]{32}:)"|'([A-Fa-f0-9]{32}:)'|([A-Fa-f0-9]{32}:))|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-u|--user)[ \t]+(?:"([A-Fa-f0-9]{32}:)"|'([A-Fa-f0-9]{32}:)'|([A-Fa-f0-9]{32}:))[ \t\\\r\n]{1,16}(?:"(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:(?:www|api)\.)?streak\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"streak.com"},
		Source:          "https://streak.readme.io/docs/authentication",
		Description:     providerBasicEmptyPasswordDescription,
		Validate:        validProvider7UsernameBasic,
		ValidateContext: provider7RequestContext(`(?:(?:www|api)\.)?streak\.com`, authorizationHeader, true),
	},
	{
		ID:          "stytch-secret-key",
		Regex:       `\b(secret-(?:live|test)-[A-Za-z0-9_-]{35}=)(?:$|[\s\x22\x27\x60,;])`,
		Keywords:    []string{"secret-live-", "secret-test-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/stytchauth/stytch-node/main/README.md",
		Description: "Stytch explicitly labels secret-live-/secret-test- values as confidential API secrets, distinct from project IDs and public tokens. The 35-character URL-safe suffix and trailing equals follow the official SDK example and pinned 48-character candidate, not a complete issuer grammar.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:          "sugester-api-url",
		Regex:       `\bhttps?://[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.sugester\.(?:com|pl)/app(?:/[A-Za-z0-9_/-]{1,128})?.json\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}api_token=([A-Za-z0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"sugester.com", "sugester.pl"},
		SecretGroup: 1,
		Source:      "https://github.com/sugester/API",
		Description: "Complete Sugester tenant API URL with its documented authorization token. DNS label bounds replace the unsafe upstream tenant punctuation; public tenant names alone are excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "sumologic-access-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Uu][Mm][Oo][Ll][Oo][Gg][Ii][Cc]_[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy]"|'[Ss][Uu][Mm][Oo][Ll][Oo][Gg][Ii][Cc]_[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy]'|[Ss][Uu][Mm][Oo][Ll][Oo][Gg][Ii][Cc]_[Aa][Cc][Cc][Ee][Ss][Ss][Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{64})"|'([A-Za-z0-9]{64})'|([A-Za-z0-9]{64})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SUMOLOGIC_ACCESSKEY"},
		Source:          "https://raw.githubusercontent.com/SumoLogic/terraform-provider-sumologic/master/website/docs/index.html.markdown",
		Description:     "SUMOLOGIC_ACCESSKEY is the documented provider environment variable for the confidential access key. Public access IDs and bare 64-character values are not matched.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "supernotes-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9_-]{43})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9_-]{43})"|'[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9_-]{43})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9_-]{43})"|'[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9_-]{43})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.supernotes\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.supernotes.app"},
		Source:          "https://developer.supernotes.app/api-reference/introduction",
		Description:     "Complete Supernotes API request with the private Api-Key header. Provider docs warn this key grants full access to create, edit and delete account cards; whitespace is not part of the token.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.supernotes\.app`, "Api-Key", false),
	},
	{
		ID:              "surveyanyplace-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:API )?[A-Za-z0-9]{32})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:API )?[A-Za-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:API )?[A-Za-z0-9]{32})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:API )?[A-Za-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}((?:API )?[A-Za-z0-9]{32})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.(?:surveyanyplace|pointerpro)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.surveyanyplace.com", "api.pointerpro.com"},
		Source:          "https://help.pointerpro.com/en/support/solutions/articles/35000148495-survey-anyplace-api",
		Description:     "Complete Survey Anyplace / Pointerpro API request. The current provider documents a raw Authorization key; the pinned historical API-prefixed scheme is retained. Survey UUIDs are not confidential by themselves.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.(?:surveyanyplace|pointerpro)\.com`, authorizationHeader, false),
	},
	{
		ID:              "surveybot-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9-]{80})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9-]{80})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9-]{80})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9-]{80})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([A-Za-z0-9-]{80})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:app\.)?surveybot\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"surveybot.io"},
		Source:          "https://apps.make.com/surveybot",
		Description:     "Complete historical Surveybot request with its account Data API private key in X-Api-Key. Header and 80-character candidate form follow the pinned detector; account-key provisioning is documented by its integration guide.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:app\.)?surveybot\.io`, apiKeyHeaderTitle, false),
	},
	{
		ID:              "survicate-data-api-key",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [a-z0-9]{32})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [a-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [a-z0-9]{32})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [a-z0-9]{32})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Aa][Ss][Ii][Cc] [a-z0-9]{32})')[ \t\\\r\n]{1,16}(?:"(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://data-api\.survicate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"data-api.survicate.com"},
		Source:          "https://developers.survicate.com/data-export/setup/",
		Description:     "Complete Survicate data-export request with documented Authorization: Basic API-key text. This is an opaque API key, not Base64(username:password), and is intentionally distinct from generic Basic validation.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`data-api\.survicate\.com`, authorizationHeader, false),
	},
	{
		ID:              "swiftype-private-auth-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{6}_[A-Za-z0-9]{6}-[A-Za-z0-9]{6})"|'([A-Za-z0-9]{6}_[A-Za-z0-9]{6}-[A-Za-z0-9]{6})'|([A-Za-z0-9]{6}_[A-Za-z0-9]{6}-[A-Za-z0-9]{6})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"auth_token"},
		Source:          "https://swiftype.com/documentation/site-search/overview",
		Description:     "Swiftype documents auth_token as its private read/write API key, in contrast to the public engine_key. Only the exact private parameter assignment with the pinned 6_6-6 form is matched; generic Swiftype-near strings and public search keys are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "tatum-api-key-url",
		Regex:       `\bhttps?://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_/-]{0,128})?\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}xApiKey=([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"tatum.io"},
		SecretGroup: 1,
		Source:      tatumAuthSource,
		Description: "Complete Tatum URL with the documented backward-compatible xApiKey query carrier; provider warns these URLs carry private API credentials. Legacy UUID body is retained only inside this carrier.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "tatum-api-key-header",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Xx]-[Aa][Pp][Ii]-[Kk][Ee][Yy]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"tatum.io"},
		Source:          tatumAuthSource,
		Description:     "Complete Tatum API or blockchain-gateway request with its documented private x-api-key header; generic UUIDs and public wallet addresses are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:api(?:-[a-z0-9]{2,8})?|[a-z0-9-]{1,63}\.gateway)\.tatum\.io`, apiKeyHeaderLower, false),
	},
	{
		ID:          "tatum-gateway-key-path",
		Regex:       `\bhttps://[a-z0-9-]{1,63}\.gateway\.tatum\.io/([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"gateway.tatum.io"},
		SecretGroup: 1,
		Source:      tatumAuthSource,
		Description: "Tatum documents an API key in the first and only gateway URL path segment for backward-compatible endpoints. Only a complete gateway URL with the pinned legacy UUID candidate is accepted.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "taxjar-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Tt][Aa][Xx][Jj][Aa][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Tt][Aa][Xx][Jj][Aa][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Tt][Aa][Xx][Jj][Aa][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-z0-9]{32})"|'([a-z0-9]{32})'|([a-z0-9]{32})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"TAXJAR_API_KEY"},
		Source:          "https://raw.githubusercontent.com/taxjar/taxjar-node/master/README.md",
		Description:     "The official TaxJar SDK documents TAXJAR_API_KEY and warns never to expose this token in client-side code. Matching requires its exact named literal carrier, not a bare hash.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "teamgate-user-auth-token",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{40})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{40})"|'[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{40})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{40})"|'[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{40})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.teamgate\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.teamgate.com"},
		Source:          "https://developers.teamgate.com/",
		Description:     "Complete Teamgate API request containing the private per-user X-Auth-Token. The application X-App-Key component by itself identifies an app and is not mistaken for the user credential.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.teamgate\.com`, "X-Auth-Token", false),
	},
	{
		ID:          "technical-analysis-api-url",
		Regex:       `\bhttps?://technical-analysis-api\.com/api/v1/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}apiKey=([A-Z0-9]{48})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"technical-analysis-api.com"},
		SecretGroup: 1,
		Source:      "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/technicalanalysisapi",
		Description: "Complete historical Technical Analysis API authenticated request URL. The pinned verifier supplies apiKey here; both this carrier and the 48-uppercase/digit constraint are scanner evidence, as the provider documentation endpoint currently fails TLS certificate validation.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "tefter-api-token",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9]{20})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9]{20})"|'[Xx]-[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9]{20})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9]{20})"|'[Xx]-[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9]{20})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:www\.)?tefter\.io(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"tefter.io"},
		Source:          "https://www.tefter.io/docs/api",
		Description:     "Complete Tefter API request carrying the documented private X-User-Token; bookmark URLs and public account metadata do not qualify.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:www\.)?tefter\.io`, "X-User-Token", false),
	},
	{
		ID:              "teletype-api-token",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9-]{64})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9-]{64})"|'[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9-]{64})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9-]{64})"|'[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([A-Za-z0-9-]{64})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.teletype\.app(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.teletype.app"},
		Source:          "https://teletype.app/help/api",
		Description:     "Complete Teletype Public API request with the documented per-project private X-Auth-Token. Public API names do not make authentication tokens public.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.teletype\.app`, "X-Auth-Token", false),
	},
	{
		ID:          "teletype-api-token-url",
		Regex:       `\bhttps?://api\.teletype\.app/public/api/v1/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}token=([A-Za-z0-9-]{64})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.teletype.app"},
		SecretGroup: 1,
		Source:      "https://teletype.app/help/api",
		Description: "Complete Teletype API URL using the documented alternative token query authentication carrier.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "telnyx-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Tt][Ee][Ll][Nn][Yy][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Tt][Ee][Ll][Nn][Yy][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Tt][Ee][Ll][Nn][Yy][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"(KEY[A-Za-z0-9_-]{55})"|'(KEY[A-Za-z0-9_-]{55})'|(KEY[A-Za-z0-9_-]{55})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"TELNYX_API_KEY"},
		Source:          "https://raw.githubusercontent.com/team-telnyx/telnyx-node/master/README.md",
		Description:     "TELNYX_API_KEY is the official server-side SDK environment variable. Its pinned KEY-prefixed token form is kept distinct from documented TELNYX_PUBLIC_KEY webhook verification material.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "testingbot-api-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Tt][Ee][Ss][Tt][Ii][Nn][Gg][Bb][Oo][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Tt][Ee][Ss][Tt][Ii][Nn][Gg][Bb][Oo][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Tt][Ee][Ss][Tt][Ii][Nn][Gg][Bb][Oo][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-z0-9]{32})"|'([a-z0-9]{32})'|([a-z0-9]{32})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"TESTINGBOT_SECRET"},
		Source:          "https://raw.githubusercontent.com/testingbot/testingbot-api/master/README.md",
		Description:     "TESTINGBOT_SECRET is the official SDK environment variable for the private API secret; TESTINGBOT_KEY identifies the companion account key and is not matched by this rule.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "the-odds-api-url",
		Regex:       `\bhttps?://api\.the-odds-api\.com/v4/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}apiKey=([a-f0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.the-odds-api.com"},
		SecretGroup: 1,
		Source:      "https://the-odds-api.com/liveapi/guides/v4/",
		Description: "Complete The Odds API URL with its subscription-authenticating apiKey query parameter. Public event/sport identifiers and bare hashes remain excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "thinkific-api-key-header",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Aa][Pp][Ii]-[Kk][Ee][Yy]"|'[Xx]-[Aa][Uu][Tt][Hh]-[Aa][Pp][Ii]-[Kk][Ee][Yy]'|[Xx]-[Aa][Uu][Tt][Hh]-[Aa][Pp][Ii]-[Kk][Ee][Yy])[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-f0-9]{32})"|'([a-f0-9]{32})'|([a-f0-9]{32})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"X-Auth-API-Key"},
		Source:          "https://developers.thinkific.com/api/api-key-auth",
		Description:     "Thinkific documents X-Auth-API-Key as the secret API-key header for private or one-off apps. X-Auth-Subdomain is public routing metadata and does not qualify on its own.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "tiingo-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-z0-9]{40})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-z0-9]{40})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-z0-9]{40})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-z0-9]{40})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [a-z0-9]{40})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.tiingo\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.tiingo.com"},
		Source:          "https://www.tiingo.com/documentation/general/connecting",
		Description:     "Complete Tiingo API request carrying the private Authorization: Token credential. Provider documentation says the token replaces username/password; this non-Bearer scheme needs distinct coverage.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.tiingo\.com`, authorizationHeader, false),
	},
	{
		ID:          "tiingo-api-token-url",
		Regex:       `\bhttps?://api\.tiingo\.com/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}token=([a-z0-9]{40})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.tiingo.com"},
		SecretGroup: 1,
		Source:      "https://www.tiingo.com/documentation/general/connecting",
		Description: "Complete Tiingo URL using the documented token query authentication alternative. Account tokens must be kept private, unlike stock symbols or instrument metadata.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "timecamp-api-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-z0-9]{26})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-z0-9]{26})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-z0-9]{26})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-z0-9]{26})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-z0-9]{26})')[ \t\\\r\n]{1,16}(?:"(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://(?:app|www)\.timecamp\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"timecamp.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/timecamp",
		Description:     "Complete TimeCamp historical API request with its raw Authorization token as used by the pinned detector. Current Bearer authorization is already covered by the common rule; the raw 26-character legacy carrier is kept explicitly.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`(?:app|www)\.timecamp\.com`, authorizationHeader, false),
	},
	{
		ID:          "timezoneapi-token-url",
		Regex:       `\bhttps?://timezoneapi\.io/api/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}token=([A-Za-z]{20})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"timezoneapi.io"},
		SecretGroup: 1,
		Source:      "https://timezoneapi.io/developers/timezone",
		Description: "Complete historical TimezoneAPI request using its account service-token query parameter. The pinned 20-letter body is not detected standalone; the old provider site is now DNS-unavailable, but historical production credentials are not excluded merely for retirement.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "todoist-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Tt][Oo][Dd][Oo][Ii][Ss][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Tt][Oo][Dd][Oo][Ii][Ss][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Tt][Oo][Dd][Oo][Ii][Ss][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"([a-z0-9]{40})"|'([a-z0-9]{40})'|([a-z0-9]{40})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"TODOIST_API_TOKEN"},
		Source:          "https://raw.githubusercontent.com/Doist/todoist-api-typescript/main/README.md",
		Description:     "TODOIST_API_TOKEN is the official SDK repository environment variable for authenticated API requests. The old 40-character candidate constraint is preserved only in this exact secret assignment.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "tokeet-api-token",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})')[ \t\\\r\n]{1,16}(?:"(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://capi\.tokeet\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"capi.tokeet.com"},
		Source:          "https://apidocs.tokeet.com/",
		Description:     "Complete Tokeet Client API request carrying the documented raw Authorization API token. The account identifier is public routing metadata; an unrelated UUID is never treated as a secret.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`capi\.tokeet\.com`, authorizationHeader, false),
	},
	{
		ID:          "tomorrowio-api-key-url",
		Regex:       `\bhttps?://api\.tomorrow\.io/v4/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}apikey=([A-Za-z0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.tomorrow.io"},
		SecretGroup: 1,
		Source:      "https://docs.tomorrow.io/reference/welcome",
		Description: "Complete Tomorrow.io API URL containing the account apikey parameter used by the pinned alerts endpoint verifier. The 32-character body is scanner-derived; public location values and bare strings are excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:          "travelpayouts-api-token-url",
		Regex:       `\bhttps?://api\.travelpayouts\.com/[A-Za-z0-9_./-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}token=([a-z0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.travelpayouts.com"},
		SecretGroup: 1,
		Source:      "https://travelpayouts-data-api.readthedocs.io/",
		Description: "Complete Travelpayouts API URL with the documented token authentication parameter. Affiliate markers and public routing identifiers are not matched.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "travelpayouts-api-token-header",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Xx]-[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{32})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{32})"|'[Xx]-[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{32})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Xx]-[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{32})"|'[Xx]-[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]:[ \t]{0,16}([a-z0-9]{32})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.travelpayouts\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.travelpayouts.com"},
		Source:          "https://travelpayouts-data-api.readthedocs.io/",
		Description:     "Complete Travelpayouts API request with the documented X-Access-Token private authentication carrier, distinct from common Bearer.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.travelpayouts\.com`, "X-Access-Token", false),
	},
	{
		ID:              "travis-api-token",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [A-Za-z0-9_]{22})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [A-Za-z0-9_]{22})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [A-Za-z0-9_]{22})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [A-Za-z0-9_]{22})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Tt][Oo][Kk][Ee][Nn] [A-Za-z0-9_]{22})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.travis-ci\.(?:com|org)(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.travis-ci.com", "api.travis-ci.org"},
		Source:          "https://developer.travis-ci.com/authentication",
		Description:     "Complete Travis CI API request using documented Authorization: token on the current or historical public API host. The 22-character constraint comes from the pinned detectors; encrypted Travis secure blobs are not plaintext credentials.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.travis-ci\.(?:com|org)`, authorizationHeader, false),
	},
	{
		ID:          "trello-api-token-url",
		Regex:       `\bhttps?://api\.trello\.com/1/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}token=([A-Za-z0-9]{64})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.trello.com"},
		SecretGroup: 1,
		Source:      "https://developer.atlassian.com/cloud/trello/guides/rest-api/authorization/",
		Description: "Complete Trello API URL with a private user token. Trello explicitly permits public API keys but says user tokens must never be public; a key-only authorization URL is excluded.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:          "twelvedata-api-key-url",
		Regex:       `\bhttps?://api\.twelvedata\.com/[A-Za-z0-9_/-]{1,128}\?(?:[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}&){0,7}apikey=([a-z0-9]{32})(?:&[A-Za-z0-9_.~%-]{1,64}=[A-Za-z0-9_.~%+,:/@-]{0,128}){0,7}(?:$|[\s\x22\x27\x60<>])`,
		Keywords:    []string{"api.twelvedata.com"},
		SecretGroup: 1,
		Source:      "https://twelvedata.com/docs",
		Description: "Complete Twelve Data authenticated API URL with its account apikey query parameter. The scanner candidate is not detected in symbols, interval names or bare values.",
		Validate:    validCarrier3Literal,
	},
	{
		ID:              "twist-oauth2-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] oauth2:[a-f0-9]{40})[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] oauth2:[a-f0-9]{40})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] oauth2:[a-f0-9]{40})')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] oauth2:[a-f0-9]{40})"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] oauth2:[a-f0-9]{40})')[ \t\\\r\n]{1,16}(?:"(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.twist\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.twist.com"},
		Source:          "https://developer.twist.com/v3/",
		Description:     "Complete Twist API request with the historical Bearer oauth2:token form from the pinned verifier. Its colon falls outside generic RFC6750 b64token matching; ordinary Bearer tokens remain covered by the common rule.",
		Validate:        validCarrier3Literal,
		ValidateContext: provider7RequestContext(`api\.twist\.com`, authorizationHeader, false),
	},
	{
		ID:              "twitter-bearer-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Bb][Ee][Aa][Rr][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn]"|'[Bb][Ee][Aa][Rr][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn]'|[Bb][Ee][Aa][Rr][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"((?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))"|'((?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))'|((?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"BEARER_TOKEN"},
		Source:          "https://raw.githubusercontent.com/xdevplatform/Twitter-API-v2-sample-code/main/README.md",
		Description:     "The official X sample repository documents BEARER_TOKEN for private app-only access. Both the pinned AAAA raw and percent-escaped candidate families are confined to this explicit secret carrier; percent escapes are checked without decoding.",
		Validate:        validProvider7TwitterBearer,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "twitter-escaped-bearer-authorization",
		Regex:           `(?:\b(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?(?:[ \t]+HTTP/1\.[01])?\r?\n(?:[A-Za-z][A-Za-z0-9-]{0,63}:[^\r\n]{0,120}\r?\n){0,8}[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] (?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))[ \t]*(?:\r?\n|$)|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:"(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?))[ \t\\\r\n]{1,16}(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] (?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] (?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))')|\bcurl[ \t]+(?:(?:-X|--request)[ \t]+(?:GET|POST|PUT|PATCH|DELETE|HEAD)[ \t]+)?(?:-H|--header)[ \t]+(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] (?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]{0,16}([Bb][Ee][Aa][Rr][Ee][Rr] (?:AAAA[A-Za-z0-9]{50,500}|[A-Za-z0-9]{20,59}(?:%[0-9A-Fa-f]{2}[A-Za-z0-9]{0,100}){1,6}[A-Za-z0-9]{20,100}))')[ \t\\\r\n]{1,16}(?:"(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)"|'(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)'|(?:https://api\.(?:twitter|x)\.com(?:/[A-Za-z0-9_./?=&%+,:@~-]{0,256})?)))`,
		Keywords:        []string{"api.twitter.com", "api.x.com"},
		Source:          "https://docs.x.com/fundamentals/authentication/overview",
		Description:     "Complete X/Twitter API request with a percent-escaped Bearer candidate. Percent escapes are outside generic RFC6750 token matching; the provider request supplies a private authentication role without relying on nearby brand prose.",
		Validate:        validProvider7TwitterBearer,
		ValidateContext: provider7RequestContext(`api\.(?:twitter|x)\.com`, authorizationHeader, false),
	},
	{
		ID:              "twitter-consumer-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Cc][Oo][Nn][Ss][Uu][Mm][Ee][Rr]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Cc][Oo][Nn][Ss][Uu][Mm][Ee][Rr]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Cc][Oo][Nn][Ss][Uu][Mm][Ee][Rr]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{35,50})"|'([A-Za-z0-9]{35,50})'|([A-Za-z0-9]{35,50})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"CONSUMER_SECRET"},
		Source:          "https://raw.githubusercontent.com/xdevplatform/Twitter-API-v2-sample-code/main/README.md",
		Description:     "CONSUMER_SECRET is the private OAuth1 environment variable documented by the official X sample repository. The 35-50-character legacy union is confined to the explicit secret assignment; public CONSUMER_KEY values are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "twitter-oauth1-token-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"(?:[0-9]{15,20}-[A-Za-z0-9]{30,45})"|'(?:[0-9]{15,20}-[A-Za-z0-9]{30,45})'|(?:[0-9]{15,20}-[A-Za-z0-9]{30,45}))(?:[ \t\r\n,;]{1,16}|&)(?:"[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{35,50})"|'([A-Za-z0-9]{35,50})'|([A-Za-z0-9]{35,50}))|(?:"[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]{0,16}[:=][ \t]{0,16}(?:"([A-Za-z0-9]{35,50})"|'([A-Za-z0-9]{35,50})'|([A-Za-z0-9]{35,50}))(?:[ \t\r\n,;]{1,16}|&)(?:"[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Oo][Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]{0,16}[:=][ \t]{0,16}(?:"(?:[0-9]{15,20}-[A-Za-z0-9]{30,45})"|'(?:[0-9]{15,20}-[A-Za-z0-9]{30,45})'|(?:[0-9]{15,20}-[A-Za-z0-9]{30,45})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"oauth_token_secret"},
		Source:          "https://docs.x.com/fundamentals/authentication/oauth-1-0a/obtaining-user-access-tokens",
		Description:     "Complete OAuth1 token and token-secret pair using X numeric-user-ID token candidate syntax in either field order. The public/token identifier alone is excluded; the confidential oauth_token_secret is required and reported. Pair components must be adjacent fields in one value.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
}

var provider7RequestPath = newLazyRegexp(`^/[^\r\n]*$`)

// The regex locates a candidate, but only a complete parsed command or logged
// request establishes its destination and credential role. In particular,
// quoted-word suffixes, separate commands and header text are not destinations.
func provider7RequestContext(host, header string, basicUser bool) func(string, int, int, string) contextValidation {
	hostPattern := newLazyRegexp("^(?:" + host + ")$")
	return withAuditedAssignmentContext(func(value string, start, end int, secret string) contextValidation {
		r, ok := parseAuditedProviderRequestContext2(value, start, end)
		if !ok {
			return contextValidation{}
		}
		u, ok := auditedProviderURL2(r.url, hostPattern, provider7RequestPath)
		if !ok {
			return contextValidation{}
		}
		matches, hostHeaders := 0, 0
		if basicUser && r.user != "" {
			if r.user != secret {
				return contextValidation{}
			}
			matches++
		}
		for _, line := range r.headers[:r.headerCount] {
			name, credential, found := strings.Cut(line, ":")
			if !found {
				return contextValidation{}
			}
			credential = strings.TrimSpace(credential)
			if strings.EqualFold(name, "Host") {
				hostHeaders++
				if hostHeaders > 1 || !strings.EqualFold(credential, u.Host) {
					return contextValidation{}
				}
			}
			if strings.EqualFold(name, header) {
				if credential != secret {
					return contextValidation{}
				}
				matches++
			}
		}
		return contextValidation{accepted: matches == 1}
	})
}

// Snipcart, Streak and SSLMate explicitly put the confidential API key in the
// Basic username with an empty password. The provider-bound regex is required:
// username-only Basic is not a safe generic credential heuristic.
func validProvider7UsernameBasic(secret string) bool {
	if !validCarrier3Literal(secret) {
		return false
	}
	if len(secret) >= 6 && strings.EqualFold(secret[:6], "Basic ") {
		var decoded [80]byte
		n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(secret[6:]))
		if err != nil || (n != 33 && n != 37 && n != 76) || decoded[n-1] != ':' {
			return false
		}
		for _, c := range decoded[:n-1] {
			if !validProvider7UsernameByte(c, n) {
				return false
			}
		}
		return true
	}
	n := len(secret)
	if (n != 33 && n != 37 && n != 76) || secret[n-1] != ':' {
		return false
	}
	for i := 0; i < n-1; i++ {
		if !validProvider7UsernameByte(secret[i], n) {
			return false
		}
	}
	return true
}

func validProvider7UsernameByte(c byte, n int) bool {
	if n == 33 {
		return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
	}
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || n == 76 && c == '_'
}

// The legacy Twitter transport percent-escapes Base64 punctuation. Validate
// those literal escapes only; do not decode or infer a credential from AAAA.
func validProvider7TwitterBearer(secret string) bool {
	if len(secret) >= 7 && strings.EqualFold(secret[:7], bearerPrefix) {
		secret = secret[7:]
	}
	if len(secret) < 54 || len(secret) > 512 || !validCarrier3Literal(secret) {
		return false
	}
	for i := 0; i < len(secret); i++ {
		c := secret[i]
		if c == '%' {
			if i+2 >= len(secret) {
				return false
			}
			switch secret[i+1 : i+3] {
			case "2B", "2b", "2F", "2f", "3D", "3d":
			default:
				return false
			}
			i += 2
			continue
		}
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}
