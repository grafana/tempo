package secrets

// Evidence-audit additions use explicit same-value secret carriers where a bare
// opaque body cannot safely identify its issuer. No source metadata is consulted.
// Body alphabets and widths below are supported scanner constraints, not claims
// of exhaustive provider issuance grammar. Pinned upstream sources are recorded
// per family and in the independent fixture evidence.
var auditedEvidenceRules1 = []catalogRuleSpec{
	{
		ID:              "ai21-api-key",
		Regex:           `\b(?i:AI21_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"ai21"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/AI21Labs/ai21-python/main/README.md",
		Description:     "AI21 SDK documents AI21_API_KEY authentication. Detect only an explicit same-value assignment; UUID identifiers and provider proximity alone remain excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/ai21.yml
	},
	{
		ID:              "airbrake-user-key",
		Regex:           `\b(?i:AIRBRAKE_USER_(?:API_)?KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9-]{40})` + catalogRightBoundary,
		Keywords:        []string{"airbrake"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.airbrake.io/docs/devops-tools/api/",
		Description:     "Airbrake distinguishes user keys granting access to project data from notifier project keys embedded in apps. Require an explicit user-key assignment; project/notifier keys excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/airbrake.yml
	},
	{
		ID:              "airtable-oauth-token",
		Regex:           `\b(?i:AIRTABLE_(?:OAUTH_|ACCESS_|REFRESH_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{1,256}\.v1\.[A-Za-z0-9_-]{1,256}\.[a-f0-9]{1,256})` + catalogRightBoundary,
		Keywords:        []string{"airtable"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://airtable.com/developers/web/api/oauth-reference",
		Description:     "OAuth access and refresh tokens are bearer credentials. Retain the upstream dotted v1 shape only under an explicit Airtable token assignment, with defensive 256-character component caps; no issuer grammar is inferred.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/airtableoauth
	},
	{
		ID:              "aiven-auth-token",
		Regex:           `\b(?i:AIVEN_(?:AUTH_|API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9+/=]{372})` + catalogRightBoundary,
		Keywords:        []string{"aiven"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://aiven.io/docs/platform/howto/create_authentication_token",
		Description:     "Aiven requires storing personal tokens safely and displays them only once. Only explicit Aiven token assignments are detected; the legacy 372-character body is a scanner constraint, not a Base64 decoding or issuance claim.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/aiven.yml
	},
	{
		ID:              "algolia-admin-api-key",
		Regex:           `\b(?i:ALGOLIA_ADMIN_(?:API_)?KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"algolia"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.algolia.com/doc/guides/security/api-keys",
		Description:     "Algolia explicitly calls Admin API keys confidential and Search-only API keys frontend-safe. Require the privileged ADMIN assignment; no arbitrary key, public app ID, or online ACL inference.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/algolia.yml
	},
	{
		ID:              "mulesoft-anypoint-access-token",
		Regex:           `\b(?i:(?:MULESOFT_)?ANYPOINT_(?:ACCESS_|AUTH_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"anypoint"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.mulesoft.com/access-management/connected-apps-developers",
		Description:     "MuleSoft tokens grant Anypoint access. UUID candidates require an explicit Anypoint access/auth-token assignment; public organization/client UUIDs and unassociated cross-pairs are excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/anypoint.yml
	},
	{
		ID:              "mulesoft-anypoint-client-secret",
		Regex:           `\b(?i:(?:MULESOFT_)?ANYPOINT_CLIENT_SECRET)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-fA-F0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"anypoint"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.mulesoft.com/access-management/connected-apps-developers",
		Description:     "Connected-app client secrets authenticate confidential OAuth clients. Detect only the explicit Anypoint client-secret assignment, never a client ID or cross-paired random hex.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/anypointoauth2
	},
	{
		ID:              "apideck-secret-key",
		Regex:           `\b(?i:APIDECK_(?:API|SECRET)_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(sk_live_[A-Za-z0-9-]{93})` + catalogRightBoundary,
		Keywords:        []string{"apideck"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/apideck-libraries/node-sdk/master/README.md",
		Description:     "Apideck SDK distinguishes API key authentication from the public appId. The shared sk_live_ prefix requires an explicit Apideck secret/API-key assignment; the separate 40-character application ID is excluded.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/apideck
	},
	{
		ID:              "apollo-io-api-key",
		Regex:           `\b(?i:APOLLO(?:_IO)?_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9_-]{22})` + catalogRightBoundary,
		Keywords:        []string{"apollo"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.apollo.io/reference/authentication",
		Description:     "The pinned source targets Apollo.io sales API, not Apollo GraphQL. Require an explicit Apollo API-key assignment with the supported 22-character candidate; generic Apollo proximity excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/apollo.yml
	},
	{
		ID:              "assemblyai-api-key",
		Regex:           `\b(?i:ASSEMBLYAI_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"assemblyai"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.assemblyai.com/docs/pre-recorded-audio/getting-started/transcribe-an-audio-file",
		Description:     "AssemblyAI documents ASSEMBLYAI_API_KEY as its authentication environment variable. Require that complete assignment; opaque 32-character strings alone remain excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/assemblyai.yml
	},
	{
		ID:              "baremetrics-api-key",
		Regex:           `\b(?i:BAREMETRICS_(?:API_KEY|ACCESS_TOKEN))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?((?:(?:sk|lk)_[A-Za-z0-9]{18,25}|[A-Za-z0-9_-]{25}))` + catalogRightBoundary,
		Keywords:        []string{"baremetrics"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developers.baremetrics.com/reference/authentication",
		Description:     "Baremetrics API keys and OAuth access tokens authorize account API access. Require an explicit Baremetrics key/token assignment; shared sk_/lk_ prefixes are not bare-provider attribution.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/baremetrics.yml
	},
	{
		ID:              "baseten-api-key",
		Regex:           `\b(?i:BASETEN_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{8}\.[A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"baseten"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.baseten.co/organization/api-keys",
		Description:     "Baseten displays personal/team keys once and requires them for deployment, inference and management. Require the Baseten API-key assignment around the scanner-supported dotted candidate.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/baseten.yml
	},
	{
		ID:              "braintrust-api-key",
		Regex:           `\b(?i:BRAINTRUST_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(sk-[A-Za-z0-9]{48})` + catalogRightBoundary,
		Keywords:        []string{"braintrust"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.braintrust.dev/docs/admin/authentication",
		Description:     "Braintrust API keys authenticate services. Its observed sk- body collides with other providers, so a complete Braintrust API-key assignment is mandatory.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/braintrust
	},
	{
		ID:              "budibase-api-key",
		Regex:           `\b(?i:(?:BUDIBASE_API_KEY|X-BUDIBASE-API-KEY))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-f0-9]{32}-[a-f0-9]{78,80})` + catalogRightBoundary,
		Keywords:        []string{"budibase"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.budibase.com/docs/public-api",
		Description:     "Budibase documents its user API key and exact x-budibase-api-key header, distinct from public workspace IDs. Accept only that header or a Budibase API-key assignment, not bare structured hex.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/budibase
	},
	{
		ID:              "checkout-secret-key",
		Regex:           `\b(?i:CHECKOUT_(?:(?:PREVIOUS|DEFAULT)_)?SECRET_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(sk_(?:test_)?[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"checkout"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/checkout/checkout-sdk-net/master/README.md",
		Description:     "Checkout SDK distinguishes previous-account secret keys from publishable keys. Require a named Checkout secret-key assignment for the legacy UUID family; customer/public IDs are excluded.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/checkout
	},
	{
		ID:              "cisco-meraki-api-key",
		Regex:           `\b(?i:(?:X-CISCO-MERAKI-API-KEY|(?:CISCO_)?MERAKI_(?:DASHBOARD_)?API_KEY))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-fA-F0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"meraki"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developer.cisco.com/meraki/api-v1/authorization/",
		Description:     "Meraki documents the X-Cisco-Meraki-API-Key authentication header. Require that header or an explicit Meraki API-key assignment, excluding unrelated SHA-1 identifiers.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/ciscomeraki.yml
	},
	{
		ID:              "clarifai-access-token",
		Regex:           `\b(?i:CLARIFAI_(?:PAT|API_KEY))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-fA-F0-9]{32,36})` + catalogRightBoundary,
		Keywords:        []string{"clarifai"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/Clarifai/clarifai-python/master/README.md",
		Description:     "Clarifai SDK documents CLARIFAI_PAT for scoped request authentication. Require an explicit PAT/API-key assignment; bare application/model IDs and proximity-only hex are excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/clarifai.yml
	},
	{
		ID:              "clickhouse-cloud-secret-key",
		Regex:           `\b(?i:CLICKHOUSE_(?:CLOUD_)?(?:API_)?(?:KEY_)?SECRET(?:_KEY)?)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(4b1d[A-Za-z0-9]{38})` + catalogRightBoundary,
		Keywords:        []string{"clickhouse"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://clickhouse.com/docs/products/cloud/features/admin-features/api/openapi",
		Description:     "ClickHouse Cloud separates public API key ID from the secret used in Basic authentication. Detect the secret only in a named ClickHouse secret assignment, not from the short hexadecimal prefix alone.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/clickhouse.yml
	},
	{
		ID:              "close-crm-api-key",
		Regex:           `\b(?i:CLOSE(?:_CRM|_IO)?_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(api_[A-Za-z0-9.]{45})` + catalogRightBoundary,
		Keywords:        []string{"close"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developer.close.com/api/overview/api-key-authentication",
		Description:     "Close API keys authenticate internal scripts as the Basic username. Require an explicit Close API-key assignment because api_ is shared; empty Basic passwords do not make the key public.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/closecrm
	},
	{
		ID:              "codecov-upload-token",
		Regex:           `\b(?i:CODECOV_TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"codecov"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.codecov.com/docs/codecov-tokens",
		Description:     "Codecov explicitly calls upload tokens secret and uses them to prevent forged coverage uploads. Require CODECOV_TOKEN assignment; bare UUIDs and provider proximity remain excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/codecov.yml
	},
	{
		ID:          "coderabbit-api-key",
		Regex:       `\b(cr-[a-f0-9]{58})` + catalogRightBoundary,
		Keywords:    []string{"cr-"},
		SecretGroup: 1,
		Source:      "https://docs.coderabbit.ai/cli",
		Description: "CodeRabbit documents cr- Agentic API keys for headless authentication. Detect the prefixed candidate using Titus body width, without a claimed issuer checksum or online validation.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/coderabbit.yml
	},
	{
		ID:              "cohere-api-key",
		Regex:           `\b(?i:(?:COHERE|CO)_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"cohere", "co_api_key"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/cohere-ai/cohere-python/main/README.md",
		Description:     "Cohere SDK documents CO_API_KEY to keep API keys out of source. Require that assignment or the fully spelled COHERE_API_KEY; no bare 40-character value or provider proximity attribution.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/cohere.yml
	},
	{
		ID:              "convertapi-secret",
		Regex:           `\b(?i:CONVERTAPI_(?:API_)?(?:SECRET|TOKEN))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(secret_[A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:        []string{"convertapi"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/ConvertAPI/convertapi-node/master/README.md",
		Description:     "ConvertAPI SDK uses an account credential to authorize conversions. Require a ConvertAPI secret/token assignment around the legacy secret_ candidate; shared bare prefix excluded.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/convertapi
	},
	{
		ID:          "courier-api-key",
		Regex:       `\b((?:pk|dk)_(?:prod|test)_[A-Za-z0-9]{28})` + catalogRightBoundary,
		Keywords:    []string{"pk_prod_", "pk_test_", "dk_prod_", "dk_test_"},
		SecretGroup: 1,
		Source:      "https://www.courier.com/docs/reference/api-overview",
		Description: "Courier states API keys are server-only secrets; published/draft keys use pk_/dk_ with prod/test environments. The 28-character body follows the pinned scanner; no public client token or unbounded arbitrary environment.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/courier
	},
	{
		ID:              "cursor-api-key",
		Regex:           `\b(?i:CURSOR_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(key_[a-f0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"cursor"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://cursor.com/docs/api",
		Description:     "Cursor API keys authorize cloud agents and administrative operations. Require CURSOR_API_KEY assignment; key_ alone cannot identify an issuer.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/cursor.yml
	},
	{
		ID:              "data-gov-api-key",
		Regex:           `\b(?i:(?:DATA_GOV|DATAGOV)_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"data_gov", "datagov"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://api.data.gov/docs/developer-manual/",
		Description:     "Data.gov explicitly specifies 40-character keys that must remain private. Require a Data.gov API-key assignment; the public DEMO_KEY and arbitrary quota-looking strings are excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/datagov.yml
	},
	{
		ID:              "deepgram-api-key",
		Regex:           `\b(?i:DEEPGRAM_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"deepgram"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developers.deepgram.com/guides/fundamentals/authenticating",
		Description:     "Deepgram explicitly says its privileged API keys must not appear in client-side code or public repositories. Require the DEEPGRAM_API_KEY assignment; the supported alphabet combines pinned upstream variants.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/deepgram.yml
	},
	{
		ID:              "deepseek-api-key",
		Regex:           `\b(?i:DEEPSEEK_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(sk-[a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"deepseek"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://api-docs.deepseek.com/",
		Description:     "DeepSeek documents DEEPSEEK_API_KEY for authenticated inference. Require that exact assignment, not the shared sk- prefix or proximity-only matching.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/deepseek.yml
	},
	{
		ID:              "diffbot-api-token",
		Regex:           `\b(?i:DIFFBOT_(?:API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"diffbot"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.diffbot.com/docs/authentication",
		Description:     "Diffbot authenticates API requests with a token. Require a complete Diffbot API-token assignment instead of a provider name near arbitrary 32-character text.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/diffbot.yml
	},
	{
		ID:              "ditto-words-api-key",
		Regex:           `\b(?i:DITTO(?:WORDS|_WORDS)?_(?:API_KEY|TOKEN))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}\.[a-z0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"ditto"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developer.dittowords.com/api-reference/authentication",
		Description:     "Ditto Words API keys grant all workspace data and must not be public or client-side. Require an explicit Ditto key/token assignment; unrelated Eclipse Ditto and Ditto Live identities are not inferred.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/ditto
	},
	{
		ID:              "ecostruxure-it-api-key",
		Regex:           `\b(?i:ECOSTRUXURE_?IT_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(AK1[A-Za-z0-9/]{50,55})` + catalogRightBoundary,
		Keywords:        []string{"ecostruxure"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/EcoStruxureIT-Public/IT-Expert-Rest-API/master/README.md",
		Description:     "EcoStruxure IT keys authorize access to equipment inventory and alarms. Require an explicit EcoStruxure IT API-key assignment because AK1 alone does not uniquely identify the issuer.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/ecostruxureit
	},
	{
		ID:              "elevenlabs-api-key",
		Regex:           `\b(?i:(?:ELEVENLABS_API_KEY|XI-API-KEY))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?((?:sk_[a-f0-9]{48}|[a-f0-9]{32}))` + catalogRightBoundary,
		Keywords:        []string{"elevenlabs", "xi-api-key"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://elevenlabs.io/docs/api-reference/authentication",
		Description:     "ElevenLabs explicitly calls API keys secret and documents the xi-api-key header. Accept that header or ELEVENLABS_API_KEY assignment for current and legacy scanner shapes; bare shared sk_ prefix excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/elevenlabs.yml
	},
	{
		ID:              "etherscan-api-key",
		Regex:           `\b(?i:ETHERSCAN_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Z0-9]{34})` + catalogRightBoundary,
		Keywords:        []string{"etherscan"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.etherscan.io/set-up-your-api-key",
		Description:     "Etherscan directs users to store API keys safely; keys authorize metered and paid endpoints. Require the explicit Etherscan API-key assignment, not a random 34-character identifier.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/etherscan
	},
	{
		ID:              "everhour-api-key",
		Regex:           `\b(?i:EVERHOUR_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{6}-[a-f0-9]{6}-[a-f0-9]{8})` + catalogRightBoundary,
		Keywords:        []string{"everhour"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://support.everhour.com/article/426-do-you-have-an-api-available",
		Description:     "Everhour exposes paid-account project/time data through an authenticated API. Detect the grouped scanner candidate only under an explicit Everhour API-key assignment; no bare-provider attribution.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/everhour
	},
	{
		ID:              "exa-api-key",
		Regex:           `\b(?i:EXA_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"exa"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://exa.ai/docs/reference/search-api-guide",
		Description:     "Exa explicitly documents EXA_API_KEY for authenticated search. Require the complete assignment; arbitrary UUIDs and generic X-API-Key headers do not identify Exa.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/exaai.yml
	},
	{
		ID:              "exportsdk-api-key",
		Regex:           `\b(?i:EXPORTSDK_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-z0-9]{5,15}_[a-z0-9-]{36})` + catalogRightBoundary,
		Keywords:        []string{"exportsdk"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://docs.exportsdk.com/docs/api/authentication/",
		Description:     "ExportSDK requires an API token for requests and keys can be revoked per integration. Require EXPORTSDK_API_KEY assignment containing the complete upstream composite; standalone account IDs excluded.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/exportsdk
	},
	{
		ID:              "fibery-api-token",
		Regex:           `\b(?i:FIBERY_(?:API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-f0-9]{8}\.[a-f0-9]{35})` + catalogRightBoundary,
		Keywords:        []string{"fibery"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developers.fibery.com/guides/getting-started/authentication",
		Description:     "Fibery explicitly says API tokens carry user privileges and must remain secret. Require a complete Fibery token assignment; workspace domains and nearby random hex are not sufficient.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/fibery
	},
	{
		ID:              "finage-api-key",
		Regex:           `\b(?i:FINAGE_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?(API_KEY[A-Z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"finage"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://finage.co.uk/blog/essential-api-security-tips-for-fintech-developers-in-2025--68334cd9357714b3eba5f2a8",
		Description:     "Finage security guidance recommends protected key storage. Require FINAGE_API_KEY assignment around the upstream API_KEY-prefixed candidate; generic configuration text is not issuer attribution.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/finage
	},
	{
		ID:              "finnhub-api-token",
		Regex:           `\b(?i:(?:FINNHUB_API_KEY|FINNHUB_TOKEN|X-FINNHUB-TOKEN))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{20})` + catalogRightBoundary,
		Keywords:        []string{"finnhub"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/Finnhub-Stock-API/finnhub-python/master/finnhub/client.py",
		Description:     "Finnhub SDK sends the API key as a token for account API requests. Require a Finnhub-specific key/token assignment or X-Finnhub-Token header, not service-name proximity.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/finnhub.yml
	},
	{
		ID:              "float-api-token",
		Regex:           `\b(?i:FLOAT_(?:API_)?(?:TOKEN|KEY))["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([a-f0-9]{16}[A-Za-z0-9+/]{42,43}=)` + catalogRightBoundary,
		Keywords:        []string{"float"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developer.float.com/",
		Description:     "Float bearer tokens authorize reading and modifying private team scheduling data. Require a Float token/key assignment around the supported composite; no arbitrary Base64 decoding or issuer-exclusive grammar claim.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/float
	},
	{
		ID:              "freshbooks-access-token",
		Regex:           `\b(?i:FRESHBOOKS_(?:ACCESS_|REFRESH_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{64})` + catalogRightBoundary,
		Keywords:        []string{"freshbooks"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.freshbooks.com/api/authentication",
		Description:     "FreshBooks OAuth access and refresh tokens authorize user accounts. Require an explicit FreshBooks token assignment; arbitrary 64-character strings and unrelated domains excluded.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/freshbooks.yml
	},
	{
		ID:              "freshworks-crm-api-key",
		Regex:           `\b(?i:(?:MY)?FRESHWORKS_(?:CRM_)?API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9_-]{22})` + catalogRightBoundary,
		Keywords:        []string{"freshworks"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://developers.freshworks.com/crm/api/",
		Description:     "Freshworks CRM user API keys authenticate with the users account privileges. Require a Freshworks API-key assignment, never a bundle alias or unassociated account/key cross-pair.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/myfreshworks
	},
	{
		ID:              "front-api-token",
		Regex:           `\b(?i:FRONT_(?:API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9]{36}\.[A-Za-z0-9._-]{188,244})` + catalogRightBoundary,
		Keywords:        []string{"front"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://dev.frontapp.com/docs/authentication",
		Description:     "Front API tokens grant scoped access to conversations and contacts. Require a Front token assignment with a literal separator; the upstream wildcard separator is not accepted as provider grammar.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/front
	},
	{
		ID:              "fullstory-api-key",
		Regex:           `\b(?i:FULLSTORY_API_KEY)["\']?[ \t]{0,4}[:=][ \t]{0,4}["\']?([A-Za-z0-9/+\-]{88}|na1\.[A-Za-z0-9+/]{100}|(?:na1|eu1)\.[A-Za-z0-9]{20,512})` + catalogRightBoundary,
		Keywords:        []string{"fullstory"},
		SecretGroup:     1,
		Source:          "https://developer.fullstory.com/server/authentication/",
		Description:     "Fullstory server API keys are displayed once and authorize data access/modification. Preserve both historical scanner subsets; add the documented na1/eu1 data-center formats with a bounded 20-512 alphanumeric token subset. Exact FULLSTORY_API_KEY assignment is required; region prefixes alone do not establish issuer or secret role.",
		ValidateContext: validateBetterleaksAssignment1,
	},
	{
		ID:              "guardian-api-key",
		Regex:           `\b(?i:GUARDIAN_API_KEY)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:        []string{"guardian"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://www.theguardian.com/open-platform/terms-and-conditions",
		Description:     "Guardian Open Platform terms prohibit sharing an API key with any third party. Require GUARDIAN_API_KEY assignment around the UUID candidate, not arbitrary identifiers or provider proximity.",
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/guardianapi
	},
	{
		ID:              "conversiontools-api-token",
		Regex:           `\b(?i:CONVERSIONTOOLS_(?:API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)` + catalogRightBoundary,
		Keywords:        []string{"conversiontools"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/ConversionTools/conversiontools-python/master/README.md",
		Description:     "ConversionTools SDK uses an account API token. Add only explicit provider token assignments containing a structurally signed compact JWT, using native JWT validation; arbitrary ey-prefixed dotted strings remain excluded.",
		Validate:        validJWT,
		// Body constraint source: https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/conversiontools
	},
	{
		ID:          "firebase-fcm-server-key",
		Regex:       `\b(AAAA[A-Za-z0-9_-]{7}:[A-Za-z0-9_-]{140})` + catalogRightBoundary,
		Keywords:    []string{"AAAA"},
		SecretGroup: 1,
		Source:      "https://firebase.google.com/docs/cloud-messaging/auth-server",
		Description: "Historical Firebase Cloud Messaging server credential, distinct from app-instance registration tokens. Recognizes the complete AAAA-prefixed server-key shape from Titus v1.2.9; body widths are scanner constraints, not a published generator. Legacy API retirement does not make exposed authentication material nonsecret.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/firebase.yml
	},
	{
		ID:              "gitter-access-token",
		Regex:           `\b(?i:GITTER_(?:ACCESS_|API_)?TOKEN)["']?[ \t]{0,4}[:=][ \t]{0,4}["']?([A-Za-z0-9_-]{40})` + catalogRightBoundary,
		Keywords:        []string{"gitter"},
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Source:          "https://raw.githubusercontent.com/gitterHQ/docs/master/02.Authentication.md",
		Description:     "Historical Gitter OAuth bearer credential in an explicit Gitter access/API-token assignment. The archived provider documentation establishes user-account access. The 40-character body follows the pinned scanners; 2023 revocation is not a reason to ignore exposed historical secrets.",
		// Body constraint source: https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/gitter.yml
	},
}
