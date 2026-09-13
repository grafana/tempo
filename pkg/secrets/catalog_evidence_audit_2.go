package secrets

import (
	"net/url"
	"strings"
)

// Exact-role subsets audited against Titus v1.2.9 and TruffleHog v3.97.4.
// Titus-derived candidate constraints: see LICENSE.titus and NOTICE.titus.
// TruffleHog-derived candidate constraints remain under the repository AGPL.
var auditedEvidenceRules2 = []catalogRuleSpec{
	{
		ID:              "iex-secret-token",
		Regex:           `\b(?i:IEX(?:APIS|_?CLOUD)?_(?:API_)?(?:TOKEN|SECRET_KEY|KEY))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(sk_[a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"iex"},
		SecretGroup:     1,
		Source:          "https://iexcloud.io/documentation/administration/access-and-security.html",
		Description:     "IEX sk_ secret tokens can act on the account including billing; pk_ tokens are publishable. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "imagekit-private-key",
		Regex:           `\b(?i:IMAGEKIT_PRIVATE_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(private_[A-Za-z0-9_=-]{8,128})` + catalogRightBoundary,
		Keywords:        []string{"imagekit"},
		SecretGroup:     1,
		Source:          "https://imagekit.io/docs/api-keys",
		Description:     "ImageKit explicitly documents private_ keys as server-only confidential credentials and public_ keys as publishable. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "infura-api-key-secret",
		Regex:           `\b(?i:INFURA_(?:API_KEY|PROJECT)_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9_-]{32,128})` + catalogRightBoundary,
		Keywords:        []string{"infura"},
		SecretGroup:     1,
		Source:          "https://docs.metamask.io/developer-tools/dashboard/how-to/secure-an-api/api-key-secret/",
		Description:     "Infura separates the API key/username from the API key secret/password. Only the explicitly named secret is classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "instantly-api-key",
		Regex:           `\b(?i:INSTANTLY_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9+/]{66}==)` + catalogRightBoundary,
		Keywords:        []string{"instantly"},
		SecretGroup:     1,
		Source:          "https://developer.instantly.ai/",
		Description:     "Instantly API keys authorize workspace operations and must not be exposed in client-side code or source control. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "intercom-access-token",
		Regex:           `\b(?i:INTERCOM_(?:ACCESS_TOKEN|API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9+/]{59}=)` + catalogRightBoundary,
		Keywords:        []string{"intercom"},
		SecretGroup:     1,
		Source:          "https://developers.intercom.com/docs/build-an-integration/learn-more/authentication",
		Description:     "Intercom access tokens expose private workspace data and must be treated as passwords. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ipgeolocation-api-key",
		Regex:           `\b(?i:IPGEOLOCATION_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"ipgeolocation"},
		SecretGroup:     1,
		Source:          "https://ipgeolocation.io/documentation/api-authentication.html",
		Description:     "IPGeolocation says API keys must stay out of frontend JavaScript; request-origin authentication is the browser-safe alternative. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "lark-access-token",
		Regex:           `\b(?i:(?:LARK|LARKSUITE)_(?:TENANT|USER|APP)_ACCESS_TOKEN)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([tua]-[A-Za-z0-9_.]{14,50})` + catalogRightBoundary,
		Keywords:        []string{"lark"},
		SecretGroup:     1,
		Source:          "https://open.larksuite.com/document/server-docs/authentication-management/access-token/tenant_access_token_internal",
		Description:     "Lark tenant, app and user access tokens are server API authentication credentials. Their short prefixes are not attributed when bare. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "lark-app-secret",
		Regex:           `\b(?i:(?:LARK|LARKSUITE)_APP_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"lark"},
		SecretGroup:     1,
		Source:          "https://github.com/larksuite/node-sdk",
		Description:     "Lark appSecret is used with the separate appId to initialize the server SDK and obtain access tokens; cli_ app IDs alone are public. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "linkedin-client-secret",
		Regex:           `\b(?i:LINKEDIN_CLIENT_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:        []string{linkedinKeyword},
		SecretGroup:     1,
		Source:          "https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow",
		Description:     "LinkedIn explicitly prohibits sharing the OAuth client secret; the client ID is a public authorization identifier. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "linkedin-access-token",
		Regex:           `\b(?i:LINKEDIN_ACCESS_TOKEN)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(AQ[A-Za-z0-9_-]{200,1000})` + catalogRightBoundary,
		Keywords:        []string{linkedinKeyword},
		SecretGroup:     1,
		Source:          "https://learn.microsoft.com/en-us/linkedin/shared/authentication/client-credentials-flow",
		Description:     "LinkedIn access tokens authorize API calls; the generic AQ prefix is not treated as a standalone provider signature. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "mailgun-api-key",
		Regex:           `\b(?i:MAILGUN_(?:PRIVATE_)?API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?((?:key-[a-z0-9]{32}|[A-Za-z0-9-]{72}|[a-f0-9]{32}-[a-f0-9]{8}-[a-f0-9]{8}))` + catalogRightBoundary,
		Keywords:        []string{"mailgun"},
		SecretGroup:     1,
		Source:          "https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/messages/post-v3--domain-name--messages",
		Description:     "Mailgun sending APIs require Basic authentication using an API credential. Exact private/API key naming excludes public validation keys. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "mailjet-secret-key",
		Regex:           `\b(?i:(?:MAILJET_(?:API_)?SECRET_KEY|MJ_APIKEY_PRIVATE))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"mailjet", "mj_apikey_private"},
		SecretGroup:     1,
		Source:          "https://github.com/mailjet/mailjet-apiv3-nodejs",
		Description:     "Mailjet uses the API key and secret key as the username/password of Basic authentication; the secret-key role, not the public key, is classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "mailjet-sms-token",
		Regex:           `\b(?i:(?:MAILJET_SMS_(?:TOKEN|API_KEY)|MJ_API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"mailjet", "mj_api_token"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/mailjet/mailjet-apiv3-nodejs/master/README.md",
		Description:     "Mailjet SMS uses a separate bearer token rather than the public email API key. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "mailmodo-api-key",
		Regex:           `\b(?i:(?:MAILMODO_API_KEY|mmApiKey))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Z0-9]{7}(?:-[A-Z0-9]{7}){3})` + catalogRightBoundary,
		Keywords:        []string{"mailmodo", "mmapikey"},
		SecretGroup:     1,
		Source:          "https://support.mailmodo.com/articles/306530-trigger-mails-via-api",
		Description:     "Mailmodo documents the mmApiKey authentication header for scheduling email campaigns. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "mailsac-api-key",
		Regex:           `\b(?i:(?:MAILSAC_API_KEY|Mailsac-Key))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(k_[A-Za-z0-9]{36,128})` + catalogRightBoundary,
		Keywords:        []string{"mailsac"},
		SecretGroup:     1,
		Source:          "https://docs.mailsac.com/en/master/api_examples/getting_started/getting_started.html",
		Description:     "Mailsac documents Mailsac-Key as the authentication header for reading account email. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "nethunt-api-key",
		Regex:           `\b(?i:NETHUNT_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9-]{36})` + catalogRightBoundary,
		Keywords:        []string{"nethunt"},
		SecretGroup:     1,
		Source:          "https://nethunt.com/integration-api",
		Description:     "NetHunt documents the API key as the password in email:API-key Basic authentication. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ngrok-auth-token",
		Regex:           `\b(?i:NGROK_(?:AUTHTOKEN|API_KEY))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?((?:[A-Za-z0-9]{16,64}_[0-9][A-Za-z0-9]{19,64}|(?:cr|ak)_[A-Za-z0-9]{25,30}))` + catalogRightBoundary,
		Keywords:        []string{"ngrok"},
		SecretGroup:     1,
		Source:          "https://ngrok.com/docs/agent/config/v3/",
		Description:     "Ngrok distinguishes agent authtokens from API keys; both authenticate to ngrok and only their exact named assignments are classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "nozbe-api-token",
		Regex:           `\b(?i:NOZBE(?:_TEAMS)?_API_(?:KEY|TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{16}_[A-Za-z0-9_-]{64})` + catalogRightBoundary,
		Keywords:        []string{"nozbe"},
		SecretGroup:     1,
		Source:          "https://nozbe.help/advancedfeatures/api/",
		Description:     "Nozbe says its API token must remain private and disclosure is like giving away an email and password. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "octopus-deploy-api-key",
		Regex:           `\b(?i:(?:X-Octopus-ApiKey|OCTOPUS_API_KEY))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(API-(?:[A-Z0-9]{26}|[A-Z0-9]{29,34}))` + catalogRightBoundary,
		Keywords:        []string{"octopus"},
		SecretGroup:     1,
		Source:          "https://github.com/OctopusDeploy/OctopusDeploy-Api/wiki/Authentication",
		Description:     "Octopus confidential API key in the exact X-Octopus-ApiKey header or OCTOPUS_API_KEY assignment. Preserves the 29-34-character body scope and adds the pinned Betterleaks 26-character candidate; these are scanner bounds, not provider issuance promises. Bare API- strings remain excluded.",
		ValidateContext: validateBetterleaksEvidenceAssignment1,
	},
	{
		ID:              "ollama-api-key",
		Regex:           `\b(?i:OLLAMA_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Fa-f0-9]{32}\.[A-Za-z0-9_-]{24})` + catalogRightBoundary,
		Keywords:        []string{"ollama"},
		SecretGroup:     1,
		Source:          "https://docs.ollama.com/api/authentication",
		Description:     "Ollama requires API-key authentication for hosted cloud/private-model operations; the unauthenticated local compatibility placeholder is not this format. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "onesignal-user-auth-key",
		Regex:           `\b(?i:ONESIGNAL_(?:USER_AUTH_KEY|USER_AUTH_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})` + catalogRightBoundary,
		Keywords:        []string{"onesignal"},
		SecretGroup:     1,
		Source:          "https://documentation.onesignal.com/docs/accounts-and-keys",
		Description:     "Legacy OneSignal user auth keys authorize account operations. UUID App IDs are public and are deliberately excluded. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "optimizely-access-token",
		Regex:           `\b(?i:OPTIMIZELY_(?:PERSONAL_ACCESS_TOKEN|ACCESS_TOKEN|DATAFILE_ACCESS_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9:-]{54})` + catalogRightBoundary,
		Keywords:        []string{"optimizely"},
		SecretGroup:     1,
		Source:          "https://docs.developers.optimizely.com/web-experimentation/docs/personal-access-token",
		Description:     "Optimizely personal access tokens authorize REST API requests; SDK/environment identifiers without a confidential token role are excluded. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "lendflow-api-token",
		Regex:           `\b(?i:LENDFLOW_API_(?:KEY|TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"lendflow"},
		SecretGroup:     1,
		Source:          "https://app.lendflow.io/docs/",
		Description:     "Lendflow documents bearer API tokens as required authentication for its financial workflow API. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "paperform-api-token",
		Regex:           `\b(?i:PAPERFORM_API_(?:KEY|TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"paperform"},
		SecretGroup:     1,
		Source:          "https://paperform.readme.io/reference/getting-started-1",
		Description:     "Paperform documents an account-generated API key in a Bearer authorization header. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "really-simple-systems-access-token",
		Regex:           `\b(?i:REALLY_?SIMPLE_?SYSTEMS_ACCESS_TOKEN)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"really"},
		SecretGroup:     1,
		Source:          "https://support.reallysimplesystems.com/api-v4/",
		Description:     "Really Simple Systems V4 API requires an OAuth access token with the permissions of an account user. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "rentman-api-token",
		Regex:           `\b(?i:RENTMAN_API_TOKEN)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"rentman"},
		SecretGroup:     1,
		Source:          "https://support.rentman.io/hc/en-us/articles/360013767839-The-Rentman-API",
		Description:     "Rentman says its personalized API token is like a password and grants access to account data. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "simply-noted-api-token",
		Regex:           `\b(?i:SIMPLY_?NOTED_API_(?:KEY|TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"simply"},
		SecretGroup:     1,
		Source:          "https://simplynoted.com/pages/api-automation",
		Description:     "Simply Noted documents API keys as bearer authentication for API operations including orders. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "storychief-api-token",
		Regex:           `\b(?i:STORYCHIEF_API_(?:KEY|TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"storychief"},
		SecretGroup:     1,
		Source:          "https://developers.storychief.io/",
		Description:     "StoryChief REST API credentials authorize creating and modifying stories and account resources. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "tallyfy-access-token",
		Regex:           `\b(?i:TALLYFY_(?:ACCESS_TOKEN|API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"tallyfy"},
		SecretGroup:     1,
		Source:          "https://tallyfy.com/products/pro/integrations/open-api/",
		Description:     "Tallyfy documents OAuth access tokens in Bearer authorization headers for workflow operations. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "viewneo-access-token",
		Regex:           `\b(?i:VIEWNEO_(?:ACCESS_TOKEN|API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"viewneo"},
		SecretGroup:     1,
		Source:          "https://docs.viewneo.com/en/developers-guide/viewneo-api/authorization",
		Description:     "Viewneo explicitly says personal access tokens must be protected like passwords. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "wrike-access-token",
		Regex:           `\b(?i:WRIKE_(?:ACCESS_TOKEN|PERMANENT_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"wrike"},
		SecretGroup:     1,
		Source:          "https://developers.wrike.com/docs/oauth-20-authorization",
		Description:     "Wrike permanent access tokens grant access to account data and must be kept private. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "zeplin-access-token",
		Regex:           `\b(?i:ZEPLIN_ACCESS_TOKEN)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9._~+/-]{16,}={0,2})(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"zeplin"},
		SecretGroup:     1,
		Source:          "https://github.com/zeplin/connected-components-docs/blob/master/docs/AUTHENTICATION.md",
		Description:     "Zeplin documents ZEPLIN_ACCESS_TOKEN for noninteractive authenticated CLI/CI access. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "pusher-app-secret",
		Regex:           `\b(?i:PUSHER_(?:APP_)?SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-z0-9]{20})` + catalogRightBoundary,
		Keywords:        []string{"pusher"},
		SecretGroup:     1,
		Source:          "https://github.com/pusher/pusher-http-node",
		Description:     "Pusher server clients use the app secret for authenticating and signing operations. Public app IDs and client-visible app keys are not secrets. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "recaptcha-secret-key",
		Regex:           `\b(?i:RECAPTCHA_SECRET(?:_KEY)?)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(6L[A-Za-z0-9_-]{38})` + catalogRightBoundary,
		Keywords:        []string{"recaptcha"},
		SecretGroup:     1,
		Source:          "https://developers.google.com/recaptcha/intro",
		Description:     "reCAPTCHA distinguishes public site keys from backend secret keys; only explicit secret assignments are classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "request-finance-api-key",
		Regex:           `\b(?i:REQUEST_?FINANCE_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Z0-9]{7}(?:-[A-Z0-9]{7}){3})` + catalogRightBoundary,
		Keywords:        []string{"request"},
		SecretGroup:     1,
		Source:          "https://docs.request.finance/getting-started",
		Description:     "Request Finance documents API keys in the Authorization header; only a provider-qualified API key assignment is classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "retell-api-key",
		Regex:           `\b(?i:RETELL(?:AI)?_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(key_[a-f0-9]{28})` + catalogRightBoundary,
		Keywords:        []string{"retell"},
		SecretGroup:     1,
		Source:          "https://docs.retellai.com/accounts/api-keys-overview",
		Description:     "Retell says REST API and webhook keys must stay out of public repositories and client-side code. The generic key_ prefix is not matched bare. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "rev-api-key",
		Regex:           `(?:\b(?i:REV_(?:CLIENT|USER)_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?((?:[A-Za-z0-9-]{27}|[A-Za-z0-9+/]{27}=))|\b(?i:Authorization)[\t ]*:[\t ]*(?i:Rev)[\t ]+[A-Za-z0-9-]{27}:([A-Za-z0-9+/]{27}=))` + catalogRightBoundary,
		Keywords:        []string{"rev"},
		SecretGroup:     0,
		Source:          "https://www.rev.com/api/authentication",
		Description:     "Rev documents both Client API Key and User API Key as secret values used in the Rev authorization scheme. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation. Also recognizes the complete Rev client-key:user-key authorization header and reports its confidential user key.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "roaring-client-secret",
		Regex:           `\b(?i:ROARING_(?:CLIENT|CONSUMER)_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9_-]{28})` + catalogRightBoundary,
		Keywords:        []string{"roaring"},
		SecretGroup:     1,
		Source:          "https://developer.roaring.io/docs/guides/api-authorization-guide",
		Description:     "Roaring combines a Consumer Key and Consumer Secret in Basic authentication to obtain a bearer access token. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "runway-api-secret",
		Regex:           `\b(?i:RUNWAY(?:ML)?_API_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(key_[A-Fa-f0-9]{128})` + catalogRightBoundary,
		Keywords:        []string{"runway"},
		SecretGroup:     1,
		Source:          "https://github.com/runwayml/sdk-node",
		Description:     "Runway SDK reads RUNWAYML_API_SECRET for authenticating API requests; generic key_ values without that exact role are excluded. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "salesblink-api-key",
		Regex:           `\b(?i:SALESBLINK_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(key-[A-Za-z0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"salesblink"},
		SecretGroup:     1,
		Source:          "https://salesblink.io/api/",
		Description:     "SalesBlink documents API keys as Authorization credentials for its API endpoints. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "sanity-auth-token",
		Regex:           `\b(?i:SANITY_(?:AUTH_TOKEN|API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(sk[A-Za-z0-9]{79})` + catalogRightBoundary,
		Keywords:        []string{"sanity"},
		SecretGroup:     1,
		Source:          "https://www.sanity.io/docs/http-auth",
		Description:     "Sanity personal auth tokens give account access and the provider requires keeping them private. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "satismeter-access-token",
		Regex:           `\b(?i:SATISMETER_(?:ACCESS_TOKEN|API_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"satismeter"},
		SecretGroup:     1,
		Source:          "https://support.satismeter.com/hc/en-us/articles/6980481518227-Track-event-API",
		Description:     "SatisMeter API requests use a separate bearer authentication token. Client-publishable writeKey and projectId fields are not classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "shutterstock-client-secret",
		Regex:           `\b(?i:SHUTTERSTOCK_(?:CLIENT|CONSUMER)_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:        []string{"shutterstock"},
		SecretGroup:     1,
		Source:          "https://www.shutterstock.com/developers/documentation/authentication",
		Description:     "Shutterstock says not to share the consumer key and secret, which can be used to access the account. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "shutterstock-oauth-token",
		Regex:           `\b(?i:SHUTTERSTOCK_(?:ACCESS_TOKEN|OAUTH_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(v2/[A-Za-z0-9]{388})` + catalogRightBoundary,
		Keywords:        []string{"shutterstock"},
		SecretGroup:     1,
		Source:          "https://www.shutterstock.com/developers/documentation/authentication",
		Description:     "Shutterstock OAuth tokens authorize user operations including licensing and downloading media. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "tableau-personal-access-token",
		Regex:           `\b(?i:(?:TABLEAU_PERSONAL_ACCESS_TOKEN_SECRET|personalAccessTokenSecret))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9+/]{22}==:[A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"tableau", "personalaccesstokensecret"},
		SecretGroup:     1,
		Source:          "https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref_authentication.htm",
		Description:     "Tableau uses the personalAccessTokenSecret credential field for PAT sign-in. Token names/IDs alone are not credentials. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "uploadcare-secret-key",
		Regex:           `(?:\b(?i:UPLOADCARE_SECRET_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-z0-9]{20})|\b(?i:Authorization)[\t ]*:[\t ]*(?i:Uploadcare\.Simple)[\t ]+[a-z0-9]{20}:([a-z0-9]{20}))` + catalogRightBoundary,
		Keywords:        []string{"uploadcare"},
		SecretGroup:     0,
		Source:          "https://uploadcare.com/docs/api/rest/authentication.md",
		Description:     "Uploadcare documents the secret API key as plaintext in Uploadcare.Simple authentication, distinct from the public key and signed request signatures. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation. Also recognizes the documented Authorization: Uploadcare.Simple public-key:secret-key header and reports only its secret; signed Uploadcare requests are excluded.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "voyage-api-key",
		Regex:           `\b(?i:VOYAGE_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?(pa-[A-Za-z0-9_-]{43})` + catalogRightBoundary,
		Keywords:        []string{"voyage"},
		SecretGroup:     1,
		Source:          "https://docs.voyageai.com/docs/api-key-and-installation",
		Description:     "Voyage explicitly says API keys must remain secret and recommends the VOYAGE_API_KEY environment variable. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "webex-client-secret",
		Regex:           `\b(?i:WEBEX_CLIENT_SECRET)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Fa-f0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"webex"},
		SecretGroup:     1,
		Source:          "https://developer.webex.com/docs/integrations",
		Description:     "Webex OAuth client secrets authenticate the app during token exchange; client IDs are public identifiers. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "webex-bot-token",
		Regex:           `\b(?i:WEBEX_(?:BOT_TOKEN|ACCESS_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Za-z0-9]{64}_[A-Za-z0-9]{4}_[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})` + catalogRightBoundary,
		Keywords:        []string{"webex"},
		SecretGroup:     1,
		Source:          "https://developer.webex.com/docs/bots",
		Description:     "Webex bot tokens are shown only once and authorize acting as the bot; the provider recommends storing them safely. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "zhipu-api-key",
		Regex:           `\b(?i:(?:ZHIPU|ZHIPUAI|BIGMODEL)_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Fa-f0-9]{32}\.[A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:        []string{"zhipu", "bigmodel"},
		SecretGroup:     1,
		Source:          "https://docs.bigmodel.cn/cn/guide/develop/http/introduction",
		Description:     "BigModel documents the dot-separated API key ID and HMAC secret used for authentication. Provider-qualified assignments disambiguate the compound value. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ip2location-api-key",
		Regex:           `\b(?i:IP2LOCATION_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([A-Z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"ip2location"},
		SecretGroup:     1,
		Source:          "https://www.ip2location.com/web-service/ip2location",
		Description:     "IP2Location assigns a unique per-account API key to consume purchased lookup credits; exact API-key naming avoids arbitrary uppercase identifiers. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ipapi-access-key",
		Regex:           `\b(?i:IPAPI_(?:ACCESS_KEY|API_KEY))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"ipapi"},
		SecretGroup:     1,
		Source:          "https://ipapi.com/faq",
		Description:     "IPAPI access keys authorize account-metered lookups, including billable overages; the exact provider credential role is required. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ipinfo-access-token",
		Regex:           `\b(?i:IPINFO_(?:TOKEN|ACCESS_TOKEN))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-f0-9]{14})` + catalogRightBoundary,
		Keywords:        []string{"ipinfo"},
		SecretGroup:     1,
		Source:          "https://support.ipinfo.io/hc/en-us/articles/30792479662738-Securing-your-IPinfo-token-on-the-frontend",
		Description:     "IPinfo recommends protecting API access tokens on the backend and documents safeguards for exceptional frontend use. Exact token naming identifies the authentication role, not token liveness or restrictions. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ipinfodb-api-key",
		Regex:           `\b(?i:IPINFODB_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-z0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"ipinfodb"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/ip2location/ipinfodb-php/master/README.md",
		Description:     "IPInfoDB documents a registered API key for a server-to-server web service; only explicitly named authentication keys are classified. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "ipstack-access-key",
		Regex:           `\b(?i:IPSTACK_(?:ACCESS_KEY|API_KEY))[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-f0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"ipstack"},
		SecretGroup:     1,
		Source:          "https://ipstack.com/faq",
		Description:     "IPstack access keys authorize per-account lookup requests with billable overages; loose nearby hex and public geolocation data are excluded. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "meta-api-key",
		Regex:           `\b(?i:META_?API_API_KEY)[\t ]*['"]?[\t ]*(?:=|:)[\t ]*['"]?([a-f0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"meta"},
		SecretGroup:     1,
		Source:          "https://docs.meta-api.io/docs/security/meta-api-keys",
		Description:     "Meta-API documents API keys as protecting Spell execution so it cannot run without permission; Spell IDs are not credentials. Requires the exact in-value credential assignment/header, not loose provider proximity. Body constraints are a supported scanner subset, not an exhaustive issuance grammar or live validation.",
		ValidateContext: validateEvidence2Credential,
	},
	{
		ID:              "snowflake-connection-uri",
		Regex:           `\b([Ss][Nn][Oo][Ww][Ff][Ll][Aa][Kk][Ee]://[^\s"<>\x60]+)`,
		Keywords:        []string{"snowflake://"},
		SecretGroup:     1,
		Source:          "https://docs.snowflake.com/en/developer-guide/python-connector/sqlalchemy",
		Description:     "Snowflake documents the SQLAlchemy snowflake://user:password@account connection carrier. Requires parsed nonempty username/password and account, rejects templates, fragments and unsupported ports, and respects a 16 KiB bound. Percent-escaped userinfo is parsed once. Password-policy complexity, account existence and successful authentication are not inferred.",
		ValidateContext: validateEvidence2SnowflakeURI,
	},
}

// Exact credential roles supply confidentiality. These local limits and filters
// reject templates/masks; they do not establish issuance or token validity.
func validateEvidence2Credential(value string, start, end int, secret string) contextValidation {
	if len(secret) < 8 || len(secret) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	if start > 0 && (value[start-1] == '-' || value[start-1] == '.' || value[start-1] >= 0x80) {
		return contextValidation{}
	}
	prefixEnd := strings.Index(value[start:end], secret)
	if prefixEnd < 0 {
		return contextValidation{}
	}
	for i := start; i < start+prefixEnd; i++ {
		// Go's case folding also accepts Unicode lookalikes in ASCII names.
		if value[i] >= 0x80 {
			return contextValidation{}
		}
	}
	return contextValidation{accepted: !evidence2Placeholder(secret)}
}

func evidence2Placeholder(secret string) bool {
	for _, marker := range [...]string{redactedLiteral, placeholderChangeMe, placeholderYourAPIKey, placeholderYourToken, placeholderYourSecret, "your_access_token", "replace_me", exampleLiteral, placeholderLiteral, undefinedLiteral, nullLiteral, passwordField} {
		if strings.EqualFold(secret, marker) {
			return true
		}
	}
	for _, prefix := range [...]string{"process.env.", "os.environ", "env.", "settings.", "config.", "your_", "replace_"} {
		if len(secret) >= len(prefix) && strings.EqualFold(secret[:len(prefix)], prefix) {
			return true
		}
	}
	masked := len(secret) > 0
	for i := 0; i < len(secret); i++ {
		if secret[i] != 'x' && secret[i] != 'X' && secret[i] != '*' {
			masked = false
		}
		if secret[i] < 0x20 || secret[i] == 0x7f || strings.ContainsRune("$<>{}", rune(secret[i])) {
			return true
		}
	}
	return masked
}

func validateEvidence2SnowflakeURI(value string, start, end int, secret string) contextValidation {
	if start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	secret, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(secret) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(secret)
	if err != nil || u.Opaque != "" || u.User == nil || u.User.Username() == "" || u.Host == "" || u.Fragment != "" || strings.Contains(u.Host, ":") {
		return contextValidation{}
	}
	for label := range strings.SplitSeq(u.Host, ".") {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return contextValidation{}
		}
	}
	for i := 0; i < len(u.Host); i++ {
		c := u.Host[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' && c != '.' {
			return contextValidation{}
		}
	}
	password, present := u.User.Password()
	return contextValidation{accepted: present && password != "" && !evidence2Placeholder(password)}
}
