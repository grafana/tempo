package secrets

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Carrier constraints adapted from Titus v1.2.9 and TruffleHog v3.97.4.
// See LICENSE.titus / NOTICE.titus and the repository AGPL license.
var auditedCarriersRules2 = []catalogRuleSpec{
	{
		ID:              "eraser-api-token",
		Regex:           "\\b(?i:eraser[_-](?:api[_-])?token)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{20})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"eraser"},
		SecretGroup:     1,
		Source:          "https://docs.eraser.io/reference/api-token.md",
		Description:     "Eraser API token in an exact provider-qualified token assignment; 20-character upstream subset only, not arbitrary Eraser proximity.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "eventbrite-private-token",
		Regex:           "\\b(?i:eventbrite[_-](?:private[_-]token|access[_-]token|oauth[_-]token))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Z0-9]{20})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"eventbrite"},
		SecretGroup:     1,
		Source:          "https://www.eventbrite.com/platform/docs/authentication",
		Description:     "Eventbrite private OAuth token in an exact token assignment; API keys and public client IDs are not matched.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "fastly-api-token",
		Regex:           "\\b(?i:fastly[_-](?:api[_-]?token|key))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9_-]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"fastly"},
		SecretGroup:     1,
		Source:          "https://www.fastly.com/documentation/reference/api/",
		Description:     "Fastly-Key header or FASTLY_API_TOKEN assignment containing the upstream 32-character API token subset.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "fileio-api-token",
		Regex:           "\\b(?i:fileio[_-](?:api[_-]?(?:key|token)|access[_-]token))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{20}\\.[A-Za-z0-9]{20})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"fileio"},
		SecretGroup:     1,
		Source:          "https://www.file.io/developers",
		Description:     "File.io API credential explicitly assigned to a provider API-key/token role, using Titus two-component scanner constraints; unqualified file keys/links are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "finicity-api-token",
		Regex:           `\b(?i:finicity[_-]app[_-]token)["']?[ \t]*[:=][ \t]*["']?([A-Fa-f0-9]{32}|[A-Za-z0-9]{20})` + catalogRightBoundary,
		Keywords:        []string{"finicity"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/Mastercard/finicity-openapi/main/bin/setup.sh",
		Description:     "Finicity-App-Token header or exact FINICITY_APP_TOKEN assignment. The 20-character provider-example and audited 32-hex candidate forms are scanner constraints, not issuer guarantees; app-key identifiers and partner IDs are not this token.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "finicity-partner-secret",
		Regex:           "\\b(?i:finicity[_-]partner[_-]secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{20})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"finicity"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/Mastercard/finicity-openapi/main/bin/setup.sh",
		Description:     "Finicity confidential partner secret in an exact provider-qualified partner-secret assignment; twenty-character upstream subset, not proximity to Finicity.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "flickr-oauth-secret",
		Regex:           "\\b(?i:flickr[_-](?:(?:api|consumer)[_-]secret|oauth[_-]token[_-]secret))[\"']?[ \\t]*[:=][ \\t]*[\"']?((?:[a-fA-F0-9]{16}|[a-fA-F0-9]{32}))(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"flickr"},
		SecretGroup:     1,
		Source:          "https://www.flickr.com/services/api/auth.oauth.html",
		Description:     "Flickr consumer or OAuth token signing secret in an exact secret role. Provider examples use 16 hex characters; the audited 32-hex variant is accepted only as an explicitly named secret, never as an API key or OAuth token ID.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "foursquare-client-secret",
		Regex:           "\\b(?i:foursquare[_-]client[_-]secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Z0-9]{48})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"foursquare"},
		SecretGroup:     1,
		Source:          "https://docs.foursquare.com/developer/reference/v2-authentication",
		Description:     "Foursquare client secret in an exact provider-qualified assignment. The identically shaped public client ID is not matched.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "freshdesk-api-key",
		Regex:           "\\b(?i:freshdesk[_-]api[_-]?key)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{20})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"freshdesk"},
		SecretGroup:     1,
		Source:          "https://developers.freshdesk.com/api/#authentication",
		Description:     "Freshdesk personal API key in an exact provider-qualified API-key assignment; public tenant domains alone are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "gocardless-access-token",
		Regex:           "\\b(?i:(?:gocardless|gc)[_-]access[_-]token)[\"']?[ \\t]*[:=][ \\t]*[\"']?(live_[A-Za-z0-9_=-]{40,42})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"gocardless", "gc"},
		SecretGroup:     1,
		Source:          "https://docs.gocardless.com/docs/getting-started/set-up",
		Description:     "GoCardless live access token in the explicitly named access-token carrier; the shared live_ prefix alone is not provider evidence.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "gumroad-access-token",
		Regex:           "\\b(?i:gumroad[_-](?:access[_-]token|oauth[_-]token))[\"']?[ \\t]*[:=][ \\t]*[\"']?((?:[A-Fa-f0-9]{64}|[A-Za-z0-9-]{43}))(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"gumroad"},
		SecretGroup:     1,
		Source:          "https://gumroad.com/api",
		Description:     "Gumroad access-token assignment in the two pinned historical scanner forms; bare values, application IDs and loose provider proximity are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "helpscout-client-secret",
		Regex:           "\\b(?i:help[_-]?scout[_-]client[_-]secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{20,64})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"scout"},
		SecretGroup:     1,
		Source:          "https://developer.helpscout.com/mailbox-api/overview/authentication/",
		Description:     "Help Scout OAuth client secret in an exact provider-qualified client-secret assignment. The upstream 20-64-character subset is not a provider length guarantee.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "hasura-admin-secret",
		Regex:           `\b(?i:(?:x-hasura-admin-secret|hasura_graphql_admin_secret))["']?[ \t]*[:=][ \t]*("[^"\r\n]+"|'[^'\r\n]+'|[^\s"'<>\x60{},;()\[\]]+)` + catalogRightBoundary,
		Keywords:        []string{"hasura"},
		SecretGroup:     1,
		Source:          "https://hasura.io/docs/2.0/auth/authentication/admin-secret-access/",
		Description:     "Exact X-Hasura-Admin-Secret header or HASURA_GRAPHQL_ADMIN_SECRET assignment. This privileged role alone establishes confidentiality, so a cloud endpoint is unnecessary; no guessed length restriction.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "mattermost-access-token",
		Regex:           "\\b(?i:mattermost[_-](?:(?:personal|bot|access)[_-])?token)[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{26})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"mattermost"},
		SecretGroup:     1,
		Source:          "https://developers.mattermost.com/integrate/reference/personal-access-token/",
		Description:     "Mattermost personal/bot access token in an exact provider-qualified token assignment, including the cloud configuration subset. Public user/token IDs and loose mm proximity are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "meraki-api-key",
		Regex:           "\\b(?i:(?:x-cisco-meraki-api-key|meraki[_-]api[_-]?key))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Fa-f0-9]{40})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"meraki"},
		SecretGroup:     1,
		Source:          "https://developer.cisco.com/meraki/api-v1/authorization/",
		Description:     "Cisco Meraki API key carried by its exact documented X-Cisco-Meraki-API-Key header or equivalent provider API-key assignment.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "metabase-session-token",
		Regex:           "\\b(?i:(?:x-metabase-session|metabase[_-]session))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Fa-f0-9]{8}(?:-[A-Fa-f0-9]{4}){3}-[A-Fa-f0-9]{12})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"metabase"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/metabase/metabase/master/src/metabase/server/middleware/session.clj",
		Description:     "Exact X-Metabase-Session header or named Metabase session assignment; the authentication role identifies the confidential session without a public base-URL helper.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "oanda-access-token",
		Regex:           "\\b(?i:oanda[_-](?:access[_-]token|api[_-]token))[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{24})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"oanda"},
		SecretGroup:     1,
		Source:          "https://developer.oanda.com/rest-live-v20/authentication/",
		Description:     "Explicit OANDA access-token assignment for the pinned 24-character historical subset; complete Bearer headers also use the protocol carrier, including modern hyphenated forms.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "droneci-access-token",
		Regex:           "\\b(?i:drone(?:ci)?[_-](?:access[_-])?token)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9_-]+(?:\\.[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+)?)(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"drone"},
		SecretGroup:     1,
		Source:          "https://docs.drone.io/api/overview/",
		Description:     "Explicit Drone token assignment containing a signed JWT or the pinned historical 32-64 hex / 32 alphanumeric token subset. JWT structure is validated; loose service-name proximity and public token IDs are not used.",
		Validate:        validDroneToken2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "endorlabs-api-secret",
		Regex:           "\\b(?i:endor_api_credentials_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?(endr\\+[A-Za-z0-9-]{16})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"endor"},
		SecretGroup:     1,
		Source:          "https://docs.endorlabs.com/developers-api/rest-api/authentication",
		Description:     "Exact Endor API secret assignment; the endr+ value in the public KEY role is deliberately not accepted.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "endorlabs-key-secret",
		Regex:       "\\{[^{}]*\"(?:key|secret)\"[^{}]*\\}",
		Keywords:    []string{"{"},
		SecretGroup: 0,
		Source:      "https://docs.endorlabs.com/developers-api/rest-api/authentication",
		Description: "Complete flat Endor key/secret JSON exchange, parsed and required to contain distinct endr+16 values in the documented roles. Bare prefixed identifiers are not secrets.",
		Validate:    validEndorPair2,
	},
	{
		ID:              "github-oauth-client-secret",
		Regex:           "\\b(?i:github[_-](?:oauth[_-])?client[_-]secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Fa-f0-9]{40})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"github"},
		SecretGroup:     1,
		Source:          "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps",
		Description:     "GitHub OAuth client secret in an exact provider-qualified assignment; prefixed access tokens and public application IDs are separate families.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "gitalk-oauth-client-secret",
		Regex:       "\\bnew[ \\t]+Gitalk[ \\t]*\\([ \\t\\r\\n]*\\{[^{}]{0,1000}\\}",
		Keywords:    []string{"Gitalk"},
		SecretGroup: 0,
		Source:      "https://github.com/gitalk/gitalk/blob/master/README.md",
		Description: "Gitalk configuration with literal clientID and clientSecret properties in either order; only its secret-bearing OAuth pair is reported.",
		Validate:    validGitalkPair2,
	},
	{
		ID:              "gemini-api-key-secret",
		Regex:           "\\b(?i:gemini_api_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?(?:master-|account-)[A-Za-z0-9]{20}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:gemini_api_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{27,28})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"gemini"},
		SecretGroup:     1,
		Source:          "https://developer.gemini.com/authentication/api-key",
		Description:     "Adjacent explicitly named Gemini API key and HMAC signing secret. The master-/account- API key alone is public; upstream widths constrain candidates, not issuance.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "klingai-access-key-secret",
		Regex:           "\\b(?i:kling_access_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9]{32}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:kling_secret_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"kling"},
		SecretGroup:     1,
		Source:          "https://docs.qingque.cn/d/home/eZQDkhg4h2Qg8SEVSUTBdzYeY",
		Description:     "Kling access-key and secret-key literals in adjacent explicitly named assignments; the access identifier alone is excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "kraken-api-key-secret",
		Regex:           "\\b(?i:kraken_api_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9+/=]{56}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:kraken_api_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9+/]{86}={0,2})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"kraken"},
		SecretGroup:     1,
		Source:          "https://docs.kraken.com/exchange/guides/rest/authentication",
		Description:     "Adjacent Kraken public API key and explicitly labeled Base64 private signing secret. API-Sign signatures are not accepted as private keys.",
		Validate:        validKrakenSecret2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "luno-api-key-secret",
		Regex:           "\\b(?i:luno_key_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[a-z0-9]{13}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:luno_key_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9_-]{43})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"luno"},
		SecretGroup:     1,
		Source:          "https://www.luno.com/en/developers/api",
		Description:     "Adjacent Luno key-ID and key-secret assignments; the pair is used as HTTP Basic username/password and neither bare component is classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "mux-token-secret",
		Regex:           "\\b(?i:mux_token_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:mux_token_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9+/]{75})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"mux"},
		SecretGroup:     1,
		Source:          "https://www.mux.com/docs/core/make-api-requests",
		Description:     "Adjacent Mux token ID and token secret assignments in the pinned UUID/75-character scanner subset. This is the Basic-auth password, not a public playback ID.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "neutrinoapi-key-secret",
		Regex:           "\\b(?i:neutrinoapi_user_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9]{6,24}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?\\b(?i:neutrinoapi_api_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{48})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"neutrinoapi"},
		SecretGroup:     1,
		Source:          "https://www.neutrinoapi.com/api/api-basics/",
		Description:     "Adjacent explicitly named Neutrino user ID and API-key assignments. A bare API key-shaped word or public user ID is excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "kucoin-api-credentials",
		Regex:           `\b(?i:kucoin_api_key)["']?[ \t]*[:=][ \t]*["']?[a-fA-F0-9]{24}["']?[ \t\r\n]*[,;]?[ \t\r\n]*["']?(?i:kucoin_api_secret)["']?[ \t]*[:=][ \t]*["']?[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}["']?[ \t\r\n]*[,;]?[ \t\r\n]*["']?(?i:kucoin_api_passphrase)["']?[ \t]*[:=][ \t]*("[^"\r\n]{7,32}"|'[^'\r\n]{7,32}'|[^\s"'<>\x60{},;()\[\]]{7,32})` + catalogRightBoundary,
		Keywords:        []string{"kucoin"},
		SecretGroup:     1,
		Source:          "https://www.kucoin.com/docs-new/authentication",
		Description:     "Adjacent explicitly named KuCoin key, signing secret and passphrase. Signed KC-API-SIGN/KC-API-PASSPHRASE headers are not treated as the plaintext signing secret.",
		Validate:        validQuotedOrBareCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "netsuite-tba-credentials",
		Regex:           "\\b(?i:netsuite_account_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9_-]{6,15}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?(?i:netsuite_consumer_key)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9]{64}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?(?i:netsuite_consumer_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9]{64}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?(?i:netsuite_token_id)[\"']?[ \\t]*[:=][ \\t]*[\"']?[A-Za-z0-9]{64}[\"']?[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?(?i:netsuite_token_secret)[\"']?[ \\t]*[:=][ \\t]*[\"']?([A-Za-z0-9]{64})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"netsuite"},
		SecretGroup:     1,
		Source:          "https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/section_4254801119.html",
		Description:     "Complete adjacent NetSuite account, consumer-key/secret and token-ID/secret assignments. All five roles are required; four opaque 64-character strings without their roles are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "mite-api-key",
		Regex:           "\\b(?i:(?:x-miteapikey|mite[_-]api[_-]?key))[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-z0-9]{16})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"mite"},
		SecretGroup:     1,
		Source:          "https://mite.de/en/api/",
		Description:     "Mite API key in the exact X-MiteApiKey header or provider-qualified key assignment. The documented role is sufficient without a public tenant-domain helper.",
		ValidateContext: withAuditedAssignmentContext(validateAuditedHeaderLiteral2),
	},
	{
		ID:              "loggly-api-token",
		Regex:           "\\b(?i:loggly[_-](?:api[_-]token|access[_-]token))[\"']?[ \\t]*[:=][ \\t]*[\"']?([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"loggly"},
		SecretGroup:     1,
		Source:          "https://documentation.solarwinds.com/en/success_center/loggly/content/admin/token-based-api-authentication.htm",
		Description:     "Loggly API access-token assignment, not a customer ingestion token. A bare UUID or customer-token label does not establish a confidential search/admin credential.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "http-basic-credential",
		Regex:           `\b[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]["']?[ \t]*:[ \t]*["']?[Bb][Aa][Ss][Ii][Cc][ \t]+([A-Za-z0-9+/]+={0,2})(?:$|[ \t\r\n"',;])`,
		Keywords:        []string{"authorization"},
		SecretGroup:     1,
		Source:          "https://www.rfc-editor.org/rfc/rfc7617",
		Description:     "Complete HTTP Authorization Basic credential, strictly Base64-decoded once and required to contain username:nonempty-password without control bytes or placeholders. Empty usernames are allowed by the protocol. The 16KiB cap is local.",
		ValidateContext: withAuditedAssignmentContext(validateHTTPBasic2),
	},
	{
		ID:              "http-bearer-credential",
		Regex:           `\b[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]["']?[ \t]*:[ \t]*["']?[Bb][Ee][Aa][Rr][Ee][Rr][ \t]+([A-Za-z0-9._~+/-]+=*)(?:$|[ \t\r\n"',;])`,
		Keywords:        []string{"authorization"},
		SecretGroup:     1,
		Source:          "https://www.rfc-editor.org/rfc/rfc6750#section-2.1",
		Description:     "Complete HTTP Authorization Bearer carrier for opaque credentials in the RFC6750 b64token alphabet. Valid signed JWTs remain in jwt rather than duplicate here. No arbitrary length minimum or entropy heuristic; only bounded plaintext non-placeholder tokens.",
		ValidateContext: withAuditedAssignmentContext(validateHTTPBearer2),
	},
	{
		ID:          "curl-token-credential",
		Regex:       `\bcurl[ \t]+[^\r\n]{0,1000}(?:-H|--header)[ \t]+["'][Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]:[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+([A-Za-z0-9+/_=-]+)["']`,
		Keywords:    []string{curlCommand},
		SecretGroup: 1,
		Source:      "https://curl.se/docs/manpage.html#-H",
		Description: "Opaque Token authentication in an explicit curl Authorization header, not arbitrary token assignments. Basic and Bearer headers use the protocol-specific rules.",
		Validate:    validAuditedCarrierLiteral2,
	},
	{
		ID:              "ftp-userinfo-credential",
		Regex:           `\b([Ff][Tt][Pp][Ss]?://[^\s"<>\x60]+)`,
		Keywords:        []string{"ftp://", "ftps://"},
		SecretGroup:     1,
		Source:          "https://www.rfc-editor.org/rfc/rfc1738#section-3.2",
		Description:     "Parsed FTP or FTPS URL with non-anonymous user and nonempty password. Userinfo escapes are decoded once after URI framing; anonymous FTP email-address passwords, placeholders and passwordless addresses are excluded. Local 16KiB work cap. FTPS adds the TLS scheme without changing the original FTP validator or scope.",
		ValidateContext: validateFTPURI2,
	},
	{
		ID:          "gcp-service-account-credential",
		Regex:       "\\{(?:[^{}\"\\\\]|\"(?:[^\"\\\\]|\\\\.)*\")*\\}",
		Keywords:    []string{"{"},
		SecretGroup: 0,
		Source:      "https://cloud.google.com/iam/docs/keys-create-delete",
		Description: "Flat Google service-account JSON with type=service_account, an iam.gserviceaccount.com client_email, and a structurally parsed unencrypted private_key. JSON escapes are decoded by encoding/json; public metadata-only objects and private_key_id are not credentials.",
		Validate:    validGCPServiceAccount2,
	},
	{
		ID:          "gcp-authorized-user-credential",
		Regex:       "\\{(?:[^{}\"\\\\]|\"(?:[^\"\\\\]|\\\\.)*\")*\\}",
		Keywords:    []string{"{"},
		SecretGroup: 0,
		Source:      "https://raw.githubusercontent.com/googleapis/google-auth-library-python/main/google/oauth2/credentials.py",
		Description: "Flat authorized_user ADC JSON containing an apps.googleusercontent.com client ID, client secret and nonempty refresh token. The refresh token is the confidential renewable credential; public desktop application client settings without it are excluded.",
		Validate:    validGCPAuthorizedUser2,
	},
	{
		ID:          "docker-registry-credential",
		Regex:       "\"auths\"[ \\t\\r\\n]*:[ \\t\\r\\n]*(\\{(?:[^{}\"\\\\]|\"(?:[^\"\\\\]|\\\\.)*\"|\\{(?:[^{}\"\\\\]|\"(?:[^\"\\\\]|\\\\.)*\")*\\})*\\})",
		Keywords:    []string{"auths"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/docker/cli/v27.5.1/cli/config/types/authconfig.go",
		Description: "Docker auths registry map with a valid registry authority and strictly decoded auth username:password or explicit username/password. JSON escaping is parsed once. Credential-helper-only maps and empty/passwordless auth are excluded; 16KiB local cap.",
		Validate:    validDockerAuths2,
	},
	{
		ID:          "filezilla-server-password",
		Regex:       "<Server\\b[^>]*>(?:[^<]|<[^>]*>)*?</Server>",
		Keywords:    []string{"Server"},
		SecretGroup: 0,
		Source:      "https://forum.filezilla-project.org/viewtopic.php?style=246&t=38820",
		Description: "Parsed FileZilla Server XML carrying Host, User and Pass. Pass supports plaintext or base64/radix64 reversible encoding; encrypted Pass and standalone ambiguous Pass result elements are excluded.",
		Validate:    validFileZillaServer2,
	},
	{
		ID:          "maven-settings-password",
		Regex:       "<(?:server|proxy)\\b[^>]*>(?:[^<]|<[^>]*>)*?</(?:server|proxy)>",
		Keywords:    []string{serverField, proxyField},
		SecretGroup: 0,
		Source:      "https://maven.apache.org/settings.html#servers",
		Description: "Parsed Maven server/proxy XML with nonempty username and plaintext password elements in either order. Interpolation and Maven encrypted brace-enclosed passwords are excluded.",
		Validate:    validMavenPassword2,
	},
	{
		ID:          "html-password-value",
		Regex:       "<[Ii][Nn][Pp][Uu][Tt]\\b(?:[^>\"\\x27]|\"[^\"]*\"|\\x27[^\\x27]*\\x27)*>",
		Keywords:    []string{inputField},
		SecretGroup: 0,
		Source:      "https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input/password",
		Description: "HTML input with type=password and a literal nonempty value, recognized with the HTML tokenizer so attribute order, entities and unquoted type obey HTML semantics. Templates, masks and placeholders are excluded.",
		Validate:    validHTMLPassword2,
	},
	{
		ID:          "literal-password-disclosure",
		Regex:       "(?i:\\bthe (?:default )?password (?:for (?:the )?(?:user|username) (?:'[^'\\r\\n]+'|\"[^\"\\r\\n]+\") )?is )[ \\t]*('[^'\\r\\n]{1,1000}'|\"[^\"\\r\\n]{1,1000}\")",
		Keywords:    []string{passwordField},
		SecretGroup: 1,
		Source:      "https://cwe.mitre.org/data/definitions/798.html",
		Description: "Explicit natural-language password disclosure with matching literal quotes, optionally naming its user. Symbols, templates, masks and public example placeholders are excluded; ordinary passwords need no artificial entropy floor.",
		Validate:    validQuotedAuditedPassword2,
	},
	{
		ID:              "literal-username-password",
		Regex:           "\\b(?i:username|user)[\"']?[ \\t]*[=:][ \\t]*(?:\"[^\"\\r\\n]+\"|'[^'\\r\\n]+')[ \\t\\r\\n]*[,;]?[ \\t\\r\\n]*[\"']?(?i:password|passwd|pass)[\"']?[ \\t]*[=:][ \\t]*(\"[^\"\\r\\n]{1,1000}\"|'[^'\\r\\n]{1,1000}')",
		Keywords:        []string{usernameField, userField},
		SecretGroup:     1,
		Source:          "https://cwe.mitre.org/data/definitions/798.html",
		Description:     "Adjacent quoted username/password assignments or object properties. Double-quoted equals assignments must be expansion-free; colon-bound JSON/object literals and single-quoted literal dollars retain their noninterpolating semantics.",
		ValidateContext: validateQuotedPasswordAssignment2,
	},
	{
		ID:          "networkcredential-password",
		Regex:       "\\bNetworkCredential[ \\t]*\\([ \\t]*\"[^\"\\r\\n]+\"[ \\t]*,[ \\t]*(\"[^\"\\r\\n]{1,1000}\")[ \\t]*(?:,[ \\t]*\"[^\"\\r\\n]+\"[ \\t]*)?\\)",
		Keywords:    []string{"NetworkCredential"},
		SecretGroup: 1,
		Source:      "https://learn.microsoft.com/en-us/dotnet/api/system.net.networkcredential",
		Description: "NetworkCredential constructor with literal username and password, with optional domain; variable arguments and placeholders are excluded.",
		Validate:    validQuotedAuditedPassword2,
	},
	{
		ID:          "gradle-credentials-password",
		Regex:       "\\bcredentials[ \\t]*\\{[^{}]{1,1000}\\}",
		Keywords:    []string{"credentials"},
		SecretGroup: 0,
		Source:      "https://docs.gradle.org/current/userguide/supported_repository_protocols.html",
		Description: "Gradle credentials block containing distinct literal username and password declarations in either order. Double-quoted Groovy/Kotlin interpolation, comments and variable expressions are excluded; single-quoted and escaped literal dollars are preserved.",
		Validate:    validGradleCredentials2,
	},
	{
		ID:          "jwt-signing-secret",
		Regex:       "(?mi:^[ \\t]*(?:jwt:[ \\t\\r\\n]+secret:|jwtsecret:)[ \\t]+)(\"[^\"\\r\\n]{1,1000}\"|'[^'\\r\\n]{1,1000}')",
		Keywords:    []string{"jwt"},
		SecretGroup: 1,
		Source:      "https://www.rfc-editor.org/rfc/rfc7518#section-3.2",
		Description: "Literal symmetric JWT signing secret in jwt: secret: or jwtsecret: configuration. This is a credential role, not a rule for arbitrary JWT containers or public verification keys.",
		Validate:    validQuotedAuditedPassword2,
	},
	{
		ID:          "jenkins-initial-admin-password",
		Regex:       "(?m)Please use the following password to proceed to installation:\\r?\\n\\r?\\n([a-f0-9]{30,36})\\r?$",
		Keywords:    []string{"Please use the following password"},
		SecretGroup: 1,
		Source:      "https://www.jenkins.io/doc/book/installing/linux/#setup-wizard",
		Description: "Exact multiline Jenkins initial setup password banner. The generated hexadecimal password is recognized only on its own immediately following blank line.",
	},
	{
		ID:          "kubernetes-bootstrap-token",
		Regex:       "\\b(?:(?i:kubeadm)[^\\r\\n]{0,1000}--token[ =]+|(?i:(?:kubernetes[_-])?bootstrap[_-]token)[\"']?[ \\t]*[:=][ \\t]*[\"']?)([a-z0-9]{6}\\.[a-z0-9]{16})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:    []string{"kubeadm", "bootstrap"},
		SecretGroup: 1,
		Source:      "https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/#token-format",
		Description: "Canonical Kubernetes six-character public ID and sixteen-character secret inside a kubeadm --token or explicit bootstrap-token assignment. Bare shape and public token IDs are not accepted.",
	},
	{
		ID:              "kubernetes-bootstrap-secret",
		Regex:           "\\btoken-id:[ \\t]+[\"\\x27]?[a-z0-9]{6}[\"\\x27]?[ \\t\\r\\n]+token-secret:[ \\t]+[\"\\x27]?([a-z0-9]{16})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:        []string{"token-id"},
		SecretGroup:     1,
		Source:          "https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/#token-format",
		Description:     "Adjacent Kubernetes token-id/token-secret fields; the ID alone is public, while the labeled secret reconstructs the bootstrap bearer credential.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "ldap-bind-credential",
		Regex:       "\\bOpenDSObject\\([ \\t]*\"[Ll][Dd][Aa][Pp][Ss]?://[^\"\\r\\n]+\"[ \\t]*,[ \\t]*\"[^\"\\r\\n]+\"[ \\t]*,[ \\t]*(\"[^\"\\r\\n]{1,1000}\")[ \\t]*,[ \\t]*[0-9]+[ \\t]*\\)",
		Keywords:    []string{"OpenDSObject"},
		SecretGroup: 1,
		Source:      "https://learn.microsoft.com/en-us/windows/win32/api/iads/nf-iads-iadsopendsobject-opendsobject",
		Description: "Literal LDAP OpenDSObject binding call containing endpoint, bind identity, password and authentication flags. Independent URI/user/password proximity is not treated as a bundle.",
		Validate:    validQuotedAuditedPassword2,
	},
	{
		ID:          "netrc-password",
		Regex:       "\\b(?:machine[ \\t]+[^\\s\"']+|default)[ \\t\\r\\n]+login[ \\t]+(?:\"(?:[^\"\\\\]|\\\\.)+\"|[^\\s\"']+)[ \\t\\r\\n]+password[ \\t]+(\"(?:[^\"\\\\]|\\\\.)+\"|[^\\s\"']+)",
		Keywords:    []string{"machine", "default"},
		SecretGroup: 1,
		Source:      "https://everything.curl.dev/usingcurl/netrc",
		Description: "Complete netrc machine/default login/password stanza, including quoted password with spaces. Only adjacent fields are associated; template/macro placeholders are excluded.",
		Validate:    validNetrcPassword2,
	},
	{
		ID:              "jdbc-password",
		Regex:           "\\b(jdbc:(?:[^\\s\"<>`{}]|\\{(?:[^}]|}})*\\})+)",
		Keywords:        []string{"jdbc:"},
		SecretGroup:     1,
		Source:          "https://jdbc.postgresql.org/documentation/use/",
		Description:     "Password-bearing PostgreSQL, MySQL/MariaDB, SQL Server, H2/Derby or Oracle thin JDBC carrier. Driver-specific query, semicolon/braced properties or Oracle user/password syntax is framed before decoding; passwordless driver URLs and unknown drivers are excluded. Local 16KiB cap.",
		ValidateContext: validateJDBCPassword2,
	},
	{
		ID:          "mattermost-webhook-url",
		Regex:       "\\b(https?://[A-Za-z0-9.-]+(?::[0-9]+)?/hooks/[a-z0-9]{26})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:    []string{"/hooks/"},
		SecretGroup: 1,
		Source:      "https://developers.mattermost.com/integrate/webhooks/incoming/",
		Description: "Mattermost incoming webhook URL with exact hooks path and the 26-character identifier shape. Host need not contain a guessed mattermost/mm substring because self-hosted deployments use arbitrary domains. Endpoint parsing rejects malformed hosts and trailing path components.",
		Validate:    validMattermostWebhook2,
	},
	{
		ID:          "microsoft-teams-webhook-url",
		Regex:       "\\b(https://(?:outlook\\.office\\.com/webhook|[A-Za-z0-9-]+\\.webhook\\.office\\.com/webhookb2)/[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}@[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}/IncomingWebhook/[a-fA-F0-9]{32}/[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})(?:$|[^A-Za-z0-9_+/=.\\-])",
		Keywords:    []string{"IncomingWebhook"},
		SecretGroup: 1,
		Source:      "https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook",
		Description: "Complete legacy Outlook or webhook.office.com Teams IncomingWebhook URL. Secret path segments authorize posting; tenant UUIDs alone and Logic Apps URLs are separate families.",
	},
	{
		ID:              "jenkins-api-token",
		Regex:           `\b(?i:jenkins[_-]api[_-]token)["']?[ \t]*[:=][ \t]*["']?([a-fA-F0-9]{32,36})` + catalogRightBoundary,
		Keywords:        []string{"jenkins"},
		SecretGroup:     1,
		Source:          "https://www.jenkins.io/blog/2018/07/02/new-api-token-system/",
		Description:     "Explicit Jenkins API-token assignment in the pinned hexadecimal subset. Crumb/session hashes, UUID identifiers and loose Jenkins proximity are not treated as authentication secrets.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "ldap-bind-config",
		Regex:           `\b(?i:ldap_(?:uri|url))["']?[ \t]*[:=][ \t]*["']?[Ll][Dd][Aa][Pp][Ss]?://[^\s"']+["']?[ \t\r\n]*[,;]?[ \t\r\n]*["']?(?i:ldap_bind_(?:dn|user))["']?[ \t]*[:=][ \t]*["'][^"'\r\n]+["'][ \t\r\n]*[,;]?[ \t\r\n]*["']?(?i:ldap_bind_password)["']?[ \t]*[:=][ \t]*("[^"\r\n]{1,1000}"|'[^'\r\n]{1,1000}')`,
		Keywords:        []string{"ldap_"},
		SecretGroup:     1,
		Source:          "https://www.openldap.org/doc/admin26/guide.html#Simple%20Authentication",
		Description:     "Adjacent LDAP URL, literal bind identity and literal bind-password assignments. Equals assignments reject double-quoted expansion syntax; colon-bound object literals retain noninterpolating semantics.",
		ValidateContext: validateQuotedPasswordAssignment2,
	},
}

// These predicates recognize confidential carriers, not live credential validity.
func validAuditedCarrierLiteral2(s string) bool {
	if s == "" || len(s) > maxStructuredCredentialBytes || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	s = strings.TrimSpace(s)
	for _, placeholder := range []string{"", passwordField, passwdField, secretField, tokenField, placeholderChangeMe, "change-me", redactedLiteral, exampleLiteral, placeholderLiteral, nullLiteral, noneLiteral, undefinedLiteral} {
		if strings.EqualFold(s, placeholder) {
			return false
		}
	}
	for _, prefix := range []string{"your_", "$2a$", "$2b$", "$2y$", "$argon2", "$scrypt$", "pbkdf2_"} {
		if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
			return false
		}
	}
	for _, marker := range []string{"${", "$(", "{{", "}}", "<", ">"} {
		if strings.Contains(s, marker) {
			return false
		}
	}
	return strings.Trim(s, "xX*.") != ""
}

func validQuotedAuditedPassword2(s string) bool {
	if len(s) < 2 || len(s) > maxStructuredCredentialBytes || s[0] != s[len(s)-1] {
		return false
	}
	if s[0] == '"' {
		decoded, err := strconv.Unquote(s)
		return err == nil && validAuditedCarrierLiteral2(decoded)
	}
	return s[0] == '\'' && validAuditedCarrierLiteral2(s[1:len(s)-1])
}

func validQuotedOrBareCarrierLiteral2(s string) bool {
	if s != "" && (s[0] == '"' || s[0] == '\'') {
		return validQuotedAuditedPassword2(s)
	}
	return validAuditedCarrierLiteral2(s)
}

func validateQuotedPasswordAssignment2(value string, start, end int, secret string) contextValidation {
	// These regexes end with the complete quoted secret. Its preceding binder
	// distinguishes JSON/object literals from potentially shell-like assignments.
	prefix := strings.TrimRight(value[start:end-len(secret)], " \t")
	if len(prefix) > 0 && prefix[len(prefix)-1] == ':' {
		return contextValidation{accepted: validQuotedAuditedPassword2(secret)}
	}
	return contextValidation{accepted: validExpansionFreeQuotedPassword2(secret, true)}
}

func validExpansionFreeQuotedPassword2(s string, shell bool) bool {
	if len(s) < 2 || s[0] != '"' {
		return validQuotedAuditedPassword2(s)
	}
	for i := 1; i < len(s)-1; i++ {
		switch s[i] {
		case '\\':
			i++ // An escaped dollar is literal; an escaped backslash is not.
		case '`':
			if shell {
				return false
			}
		case '$':
			if i+1 < len(s)-1 {
				next := s[i+1]
				// Groovy/Kotlin references include $name and ${expression}.
				// Shell assignments additionally expand positional/special
				// parameters and command substitutions.
				if next == '{' || next == '_' || next >= 'A' && next <= 'Z' || next >= 'a' && next <= 'z' || next >= 0x80 || shell && (next >= '0' && next <= '9' || strings.ContainsRune("@*#?$!-(", rune(next))) {
					return false
				}
			}
		}
	}
	// \$ is a real quoted-string escape in these carriers but not in Go's
	// strconv.Unquote. Normalize only that escape after checking references;
	// JSON and other noninterpolating callers never enter this path.
	return validQuotedAuditedPassword2(strings.ReplaceAll(s, `\$`, `$`))
}

func validDroneToken2(s string) bool {
	if strings.Contains(s, ".") {
		return validJWT(s)
	}
	if !validAuditedCarrierLiteral2(s) || len(s) < 32 || len(s) > 64 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			if len(s) != 32 || (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
				return false
			}
		}
	}
	return true
}

func validEndorPart2(s string) bool {
	if len(s) != 21 || !strings.HasPrefix(s, "endr+") {
		return false
	}
	for i := 5; i < len(s); i++ {
		c := s[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return validAuditedCarrierLiteral2(s[5:])
}

func validEndorPair2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var fields struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}
	return json.Unmarshal([]byte(s), &fields) == nil && fields.Key != fields.Secret && validEndorPart2(fields.Key) && validEndorPart2(fields.Secret)
}

var (
	gitalkClientID2     = newLazyRegexp(`\bclientID\s*:\s*(?:'[A-Za-z0-9]{20}'|"[A-Za-z0-9]{20}")`)
	gitalkClientSecret2 = newLazyRegexp(`\bclientSecret\s*:\s*('[a-fA-F0-9]{40}'|"[a-fA-F0-9]{40}")`)
)

func validGitalkPair2(s string) bool {
	if !gitalkClientID2.MatchString(s) {
		return false
	}
	m := gitalkClientSecret2.FindStringSubmatch(s)
	return len(m) == 2 && validQuotedAuditedPassword2(m[1])
}

func validKrakenSecret2(s string) bool {
	// The documented signing input is Base64-decoded. Do not infer a key-byte
	// width or cryptographic strength from the published example.
	_, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil {
		_, err = base64.RawStdEncoding.Strict().DecodeString(s)
	}
	return err == nil && validAuditedCarrierLiteral2(s)
}

func validBasicCredential2(s string, requireUser bool) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil || !utf8.Valid(b) {
		return false
	}
	user, password, ok := strings.Cut(string(b), ":")
	if !ok || requireUser && user == "" || !validAuditedCarrierLiteral2(password) {
		return false
	}
	for _, c := range user {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func validateHTTPBasic2(value string, start, _ int, secret string) contextValidation {
	return contextValidation{accepted: (start == 0 || value[start-1] != '-') && validBasicCredential2(secret, false)}
}

func validateHTTPBearer2(value string, start, _ int, secret string) contextValidation {
	return contextValidation{accepted: (start == 0 || value[start-1] != '-') && validAuditedCarrierLiteral2(secret) && !validJWT(secret)}
}

func validateFTPURI2(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes || start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.User == nil || u.Hostname() == "" || !connectionURIHost(u.Host) {
		return contextValidation{}
	}
	user := u.User.Username()
	password, set := u.User.Password()
	return contextValidation{accepted: set && user != "" && !strings.EqualFold(user, "anonymous") && !strings.EqualFold(user, "ftp") && validAuditedCarrierLiteral2(password)}
}

func validGCPServiceAccount2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var fields struct {
		Type        string `json:"type"`
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if json.Unmarshal([]byte(s), &fields) != nil || fields.Type != "service_account" || !strings.HasSuffix(fields.ClientEmail, ".iam.gserviceaccount.com") {
		return false
	}
	user, domain, ok := strings.Cut(fields.ClientEmail, "@")
	return ok && user != "" && domain != ".iam.gserviceaccount.com" && validPrivateKey(fields.PrivateKey)
}

func validGCPAuthorizedUser2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var fields struct {
		Type         string `json:"type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
	}
	return json.Unmarshal([]byte(s), &fields) == nil && fields.Type == "authorized_user" && len(fields.ClientID) > len(".apps.googleusercontent.com") && strings.HasSuffix(fields.ClientID, ".apps.googleusercontent.com") && validAuditedCarrierLiteral2(fields.ClientSecret) && validAuditedCarrierLiteral2(fields.RefreshToken)
}

func validDockerAuths2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var registries map[string]struct {
		Auth     string `json:"auth"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if json.Unmarshal([]byte(s), &registries) != nil {
		return false
	}
	for registry, auth := range registries {
		if !strings.Contains(registry, "://") {
			registry = "https://" + registry
		}
		u, err := url.Parse(registry)
		if err != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != httpScheme && u.Scheme != httpsScheme) || !connectionURIHost(u.Host) {
			continue
		}
		if auth.Username != "" && validAuditedCarrierLiteral2(auth.Password) || validBasicCredential2(auth.Auth, true) {
			return true
		}
	}
	return false
}

func validFileZillaServer2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var server struct {
		XMLName xml.Name `xml:"Server"`
		Host    string   `xml:"Host"`
		User    string   `xml:"User"`
		Pass    struct {
			Encoding string `xml:"encoding,attr"`
			Value    string `xml:",chardata"`
		} `xml:"Pass"`
	}
	if xml.Unmarshal([]byte(s), &server) != nil || server.Host == "" || server.User == "" {
		return false
	}
	switch server.Pass.Encoding {
	case "":
		return validAuditedCarrierLiteral2(server.Pass.Value)
	case "base64", "radix64":
		b, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(server.Pass.Value))
		return err == nil && validAuditedCarrierLiteral2(string(b))
	default:
		return false
	}
}

func validMavenPassword2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var fields struct {
		XMLName  xml.Name
		Username string `xml:"username"`
		Password string `xml:"password"`
	}
	return xml.Unmarshal([]byte(s), &fields) == nil && (fields.XMLName.Local == serverField || fields.XMLName.Local == proxyField) && fields.Username != "" && !strings.HasPrefix(strings.TrimSpace(fields.Password), "{") && validAuditedCarrierLiteral2(fields.Password)
}

func validHTMLPassword2(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	z := html.NewTokenizer(strings.NewReader(s))
	t := z.Next()
	if t != html.StartTagToken && t != html.SelfClosingTagToken {
		return false
	}
	token := z.Token()
	if token.Data != inputField {
		return false
	}
	var kind, value string
	var kindSet, valueSet bool
	for _, attr := range token.Attr {
		// HTML uses the first attribute when duplicate names occur.
		if attr.Key == "type" && !kindSet {
			kind, kindSet = attr.Val, true
		} else if attr.Key == valueField && !valueSet {
			value, valueSet = attr.Val, true
		}
	}
	if len(kind) != len(passwordField) {
		return false
	}
	for i := range len(kind) {
		c := kind[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != passwordField[i] {
			return false
		}
	}
	return validAuditedCarrierLiteral2(value)
}

var (
	gradleUsername2 = newLazyRegexp(`(?:^|[{;\r\n])[ \t]*username[ \t]*(?:=[ \t]*)?(?:'[^'\r\n]+'|"[^"\r\n]+")[ \t]*(?:$|[;\r\n}])`)
	gradlePassword2 = newLazyRegexp(`(?:^|[{;\r\n])[ \t]*password[ \t]*(?:=[ \t]*)?('[^'\r\n]+'|"[^"\r\n]+")[ \t]*(?:$|[;\r\n}])`)
)

func validGradleCredentials2(s string) bool {
	if !gradleUsername2.MatchString(s) {
		return false
	}
	m := gradlePassword2.FindStringSubmatch(s)
	return len(m) == 2 && validExpansionFreeQuotedPassword2(m[1], false)
}

func validNetrcPassword2(s string) bool {
	if strings.HasPrefix(s, "\"") {
		return validQuotedAuditedPassword2(s)
	}
	return validAuditedCarrierLiteral2(s)
}

func validateJDBCPassword2(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes || start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	s = strings.TrimPrefix(s, "jdbc:")
	driver, rest, ok := strings.Cut(s, ":")
	if !ok {
		return contextValidation{}
	}
	switch driver {
	case postgresqlScheme, mysqlScheme, "mariadb":
		base, query, _ := strings.Cut(rest, "?")
		if base == "" || strings.ContainsAny(base, "#{}") || driver != postgresqlScheme && !strings.HasPrefix(base, "//") {
			return contextValidation{}
		}
		if strings.HasPrefix(base, "//") {
			u, err := url.Parse(driver + ":" + base)
			if err != nil {
				return contextValidation{}
			}
			if u.User != nil && driver != postgresqlScheme {
				password, set := u.User.Password()
				if set && validAuditedCarrierLiteral2(password) {
					return contextValidation{accepted: true}
				}
			}
		}
		params, err := url.ParseQuery(query)
		if err != nil {
			return contextValidation{}
		}
		for _, password := range params[passwordField] {
			if validAuditedCarrierLiteral2(password) {
				return contextValidation{accepted: true}
			}
		}
	case "sqlserver", "h2", "derby":
		base, properties, ok := strings.Cut(rest, ";")
		if ok && base != "" && (driver != "sqlserver" || strings.HasPrefix(base, "//")) {
			return contextValidation{accepted: validJDBCProperties2(properties)}
		}
	case "oracle":
		credentials, host, hasHost := strings.Cut(rest, "@")
		credentials, thin := strings.CutPrefix(credentials, "thin:")
		user, password, hasPassword := strings.Cut(credentials, "/")
		return contextValidation{accepted: thin && hasHost && host != "" && user != "" && hasPassword && validAuditedCarrierLiteral2(password)}
	}
	return contextValidation{}
}

func validJDBCProperties2(s string) bool {
	found := false
	for s != "" {
		name, rest, ok := strings.Cut(s, "=")
		if !ok || strings.ContainsAny(name, ";{}") {
			return false
		}
		var v string
		if strings.HasPrefix(rest, "{") {
			end := 1
			for end < len(rest) {
				if rest[end] == '}' {
					if end+1 < len(rest) && rest[end+1] == '}' {
						end += 2
						continue
					}
					break
				}
				end++
			}
			if end == len(rest) || end+1 < len(rest) && rest[end+1] != ';' {
				return false
			}
			v = strings.ReplaceAll(rest[1:end], "}}", "}")
			s = strings.TrimPrefix(rest[end+1:], ";")
		} else {
			v, s, _ = strings.Cut(rest, ";")
			if strings.ContainsAny(v, "{}") {
				return false
			}
		}
		if strings.EqualFold(name, passwordField) && validAuditedCarrierLiteral2(v) {
			found = true
		}
	}
	return found
}

func validMattermostWebhook2(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.User == nil && u.Hostname() != "" && connectionURIHost(u.Host) && u.RawQuery == "" && u.Fragment == "" && strings.Count(u.Path, "/") == 2
}

func validateAuditedHeaderLiteral2(value string, start, end int, secret string) contextValidation {
	if start > 0 && value[start-1] == '-' {
		return contextValidation{}
	}
	// HTTP field names are ASCII even though RE2's case-folding also accepts
	// Unicode aliases. Do not let those aliases create an authentication role.
	for i := start; i < end; i++ {
		if value[i] == ':' || value[i] == '=' {
			if !validQuotedOrBareCarrierLiteral2(secret) {
				return contextValidation{}
			}
			if secret[0] != '"' && secret[0] != '\'' {
				// A bare regex candidate must not stop before a template
				// continuation: ${...} otherwise becomes the literal "$".
				relative := strings.LastIndex(value[start:end], secret)
				secretEnd := start + relative + len(secret)
				if relative < 0 || secretEnd < len(value) && (value[secretEnd] == '{' || value[secretEnd] == '$') {
					return contextValidation{}
				}
			}
			return contextValidation{accepted: true}
		}
		if value[i] >= 0x80 {
			return contextValidation{}
		}
	}
	return contextValidation{}
}
