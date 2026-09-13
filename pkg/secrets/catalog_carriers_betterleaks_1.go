package secrets

import "strings"

// Carrier bounds are adapted from Betterleaks 95237cf8eb4d (LICENSE.betterleaks).
// Provider sources establish private roles, not live validity or complete issuer
// grammars. Exact role assignments replace upstream provider-proximity windows;
// the native entropy and literal guards do not import the upstream BPE filter.
var betterleaksCarriersRules1 = []catalogRuleSpec{
	{
		ID:              "abuseipdb-api-key-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:ABUSEIPDB[_ .-]API[_ .-]?KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Fa-f0-9]{80})` + catalogRightBoundary,
		Keywords:        []string{"abuseipdb"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://docs.abuseipdb.com/",
		Description:     "AbuseIPDB confidential API key in an exact provider-qualified API-key assignment. The 80-hex body is a Betterleaks scanner bound, not an issuer guarantee. The distinct existing URL family remains unchanged; unqualified Key headers, public IDs, loose provider proximity and dynamic assignments are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "airtable-legacy-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:AIRTABLE_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{17})` + catalogRightBoundary,
		Keywords:        []string{"airtable_api_key"},
		Entropy:         3,
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/Airtable/airtable.js/v0.11.0/README.md",
		Description:     "Historical Airtable secret API key in the SDK's AIRTABLE_API_KEY assignment, using the audited 17-character scanner subset. Airtable disabled legacy API access in February 2024; detection identifies historical secret material, not current usability. Base/table IDs and current personal/OAuth tokens are not this family.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bitbucket-client-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BITBUCKET[_ .-](?:(?:OAUTH[_ .-])?CLIENT[_ .-]SECRET|CONSUMER[_ .-]SECRET))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9=_-]{64})` + catalogRightBoundary,
		Keywords:        []string{"bitbucket"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/",
		Description:     "Bitbucket OAuth consumer/client secret in an exact provider-qualified secret assignment. The 64-character body is a Betterleaks scanner constraint; public consumer keys/client IDs and generic Bitbucket values are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bitrise-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BITRISE[_ .-](?:(?:PERSONAL|WORKSPACE)[_ .-])?(?:(?:ACCESS|API)[_ .-])?TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{60,120})` + catalogRightBoundary,
		Keywords:        []string{"bitrise"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://docs.bitrise.io/en/bitrise-ci/api/authenticating-with-the-bitrise-api.html",
		Description:     "Bitrise personal or workspace API access token in an exact provider-qualified token assignment. Both documented token roles are confidential; the 60-120 URL-safe characters are supported scanner bounds, not an issuer grammar. App slugs, build IDs, token names and dynamic references are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "bittrex-api-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:BITTREX[_ .-](?:API[_ .-]SECRET|SECRET[_ .-]KEY))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"bittrex"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://bittrex.zendesk.com/hc/en-us/articles/360031921872-How-to-create-an-API-key-",
		Description:     "Historical Bittrex signing secret only in an exact API-secret or secret-key assignment. The paired API/access key is an identifier, not this finding. Thirty-two alphanumeric characters are the Betterleaks scanner subset; current service availability and issuer validity are not asserted.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "civo-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:CIVO[_ .-](?:TOKEN|API[_ .-]?KEY))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{50})` + catalogRightBoundary,
		Keywords:        []string{"civo"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/civo/terraform-provider-civo/master/docs/index.md",
		Description:     "Civo confidential API key in the documented CIVO_TOKEN or exact Civo API-key assignment. Fifty alphanumeric characters are the audited scanner subset; resource IDs and API endpoint URLs do not qualify.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "couchbase-capella-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:COUCHBASE[_ .-])?CAPELLA[_ .-](?:AUTHENTICATION[_ .-]TOKEN|API[_ .-](?:KEY[_ .-])?(?:TOKEN|SECRET)))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{60,120}={0,2})` + catalogRightBoundary,
		Keywords:        []string{"capella"},
		Entropy:         4,
		SecretGroup:     1,
		Source:          "https://docs.couchbase.com/cloud/management-api-guide/management-api-start.html",
		Description:     "Capella Management API secret token in CAPELLA_AUTHENTICATION_TOKEN or an exact Capella API-token/secret assignment. Provider docs distinguish the hidden token from the public API key ID. The 60-120 Base64-alphabet characters with optional padding are scanner bounds, not decoded or authenticated issuer structure; unqualified Couchbase values and key IDs are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "databento-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:DATABENTO[_ .-]API[_ .-]?KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(db-[A-Za-z0-9]{29})` + catalogRightBoundary,
		Keywords:        []string{"databento"},
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/databento/databento-python/main/README.md",
		Description:     "Databento secret API key in the documented DATABENTO_API_KEY carrier. The provider specifies a 32-character key beginning db-; the alphanumeric body is the pinned scanner subset. The short db- marker alone, dataset names and public IDs are not classified.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "discord-client-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:DISCORD[_ .-](?:OAUTH[_ .-])?CLIENT[_ .-]SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9=_-]{32})` + catalogRightBoundary,
		Keywords:        []string{"discord"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://docs.discord.com/developers/topics/oauth2",
		Description:     "Discord OAuth client secret in an exact provider-qualified client-secret assignment, using the audited 32-character opaque subset. Public client IDs, application IDs, bot public keys and unrelated Discord labels are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "dropbox-app-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:DROPBOX[_ .-](?:APP|CLIENT)[_ .-]SECRET)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{15})` + catalogRightBoundary,
		Keywords:        []string{dropboxKeyword},
		Entropy:         3,
		SecretGroup:     1,
		Source:          "https://developers.dropbox.com/oauth-guide",
		Description:     "Dropbox confidential OAuth app/client secret in an exact secret assignment, not the identically sized public app key/client ID. The upstream 15-character opaque body is only a scanner subset; it is not asserted to be a Dropbox access-token grammar.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "gitea-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:GITEA[_ .-](?:(?:ACCESS|API|PERSONAL[_ .-]ACCESS)[_ .-])?TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-f0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"gitea"},
		Entropy:         3.3,
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/go-gitea/gitea/main/models/auth/access_token.go",
		Description:     "Gitea personal access token in an exact provider-qualified token assignment. The provider generates 20 random bytes encoded as 40 lowercase hex characters. Commit hashes, stored token hashes, token names and unlabelled hex values are excluded; public-only token scopes still identify an authenticated user.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "greptile-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:GREPTILE_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{48})` + catalogRightBoundary,
		Keywords:        []string{"greptile_api_key"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/greptileai/examples/86852dd0c740e33cbe8a226267863ff884186c39/pr-review-bot/.env.example",
		Description:     "Greptile Bearer API credential in the provider example's GREPTILE_API_KEY assignment. The 48-character Base64-alphabet body is a Betterleaks scanner constraint, not a decoder or issuer guarantee; API URLs and repository identifiers are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "influxdb-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:INFLUX(?:DB)?[_ .-](?:TOKEN|API[_ .-](?:TOKEN|KEY)))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/=_-]{88,1000})` + catalogRightBoundary,
		Keywords:        []string{"influx"},
		Entropy:         4,
		SecretGroup:     1,
		Source:          "https://docs.influxdata.com/influxdb/v2/admin/tokens/use-tokens/",
		Description:     "InfluxDB plaintext API token in the documented INFLUX_TOKEN or exact InfluxDB API-token/key assignment. Eighty-eight characters is the Betterleaks lower candidate bound and 1000 is a defensive cap, not an exhaustive issuer grammar. Token hashes, org IDs, bucket IDs and dynamic expressions are excluded.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "infomaniak-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:INFOMANIAK[_ .-](?:(?:API|ACCESS)[_ .-])?TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{60,100})` + catalogRightBoundary,
		Keywords:        []string{"infomaniak"},
		Entropy:         4,
		SecretGroup:     1,
		Source:          "https://raw.githubusercontent.com/Infomaniak/terraform-provider-infomaniak/master/docs/index.md",
		Description:     "Infomaniak confidential API token in the provider's INFOMANIAK_TOKEN or an exact API/access-token assignment. The 60-100 URL-safe characters are scanner bounds; product/account IDs and host configuration do not qualify.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
	{
		ID:              "kimi-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:(?:KIMI|MOONSHOT)[_ .-]API[_ .-]?KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(sk-[A-Za-z0-9_-]{48})` + catalogRightBoundary,
		Keywords:        []string{"kimi", "moonshot"},
		Entropy:         3.5,
		SecretGroup:     1,
		Source:          "https://platform.moonshot.ai/docs/guide/start-using-kimi-api",
		Description:     "Kimi/Moonshot confidential API key in the documented MOONSHOT_API_KEY or exact Kimi API-key assignment. The sk- plus 48-character body is the pinned scanner subset; sk- is not provider-unique and never establishes Kimi attribution without the role carrier.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: withAuditedAssignmentContext(validateProvider1AssignmentContext),
	},
}

// Extension hooks preserve the existing modern-prefix validator unchanged.
func validBetterleaksCarrier1Dropbox(s string) bool {
	if strings.HasPrefix(s, "sl.") {
		return len(s) <= maxStructuredCredentialBytes
	}
	return validAuditedCarrierLiteral2(s) && shannonEntropy(s) >= 3.5
}

func validateBetterleaksCarrier1DropboxContext(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(secret, "sl.") {
		return contextValidation{accepted: true}
	}
	if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	return validateProvider1AssignmentContext(value, start, end, secret)
}
