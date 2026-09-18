package secrets

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"strconv"
	"strings"
)

// These are offline credential candidates, not issuer, checksum, scope, expiry,
// or live authorization checks. Provider documentation establishes confidentiality;
// the pinned scanner sources below supply otherwise undocumented candidate widths.
var auditedTokensRules1 = []catalogRuleSpec{
	{
		ID:       "adobe-client-secret",
		Regex:    `\b(p8e-[A-Za-z0-9-]{32})` + catalogRightBoundary,
		Keywords: []string{"p8e-"}, SecretGroup: 1,
		Source:      "https://developer.adobe.com/developer-console/docs/guides/authentication/ServerToServerAuthentication/",
		Description: "Adobe confidential OAuth client-secret candidate. The case-sensitive p8e- prefix and 32-character alphanumeric/hyphen body are the Titus v1.2.9 scanner subset, not an issuer grammar. Public client IDs are excluded.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/adobe.yml
	},
	{
		ID:       "alchemy-api-key",
		Regex:    `(?:\b(alcht_[A-Za-z0-9]{30})|\b(?:ALCHEMY_API_KEY|alchemy_api_key|ALCHEMY_KEY|alchemy_key|ALCHEMY_AUTH_TOKEN|alchemy_auth_token)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9_-]{24,64})|\b([Hh][Tt][Tt][Pp][Ss]://(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+(?:[Aa][Ll][Cc][Hh][Ee][Mm][Yy]\.[Cc][Oo][Mm]|[Aa][Ll][Cc][Hh][Ee][Mm][Yy][Aa][Pp][Ii]\.[Ii][Oo])/v2/[A-Za-z0-9_-]{24,64}))` + catalogRightBoundary,
		Keywords: []string{"alcht_", "alchemy"}, SecretGroup: 0,
		Source:          "https://www.alchemy.com/docs/best-practices-for-key-security-and-management",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Alchemy confidential alcht_ token, exact Alchemy API-key assignment, or official-host /v2/key RPC URL. Bare unprefixed bodies and loose provider proximity are excluded. The prefixed 30-alphanumeric and carrier 24-64 URL-safe body constraints come from TruffleHog v3.97.4 and Titus v1.2.9, not a provider length guarantee. DNS and token scopes are not checked.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/alchemy/alchemy.go
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/alchemy.yml
	},
	{
		ID:       "amazon-mws-auth-token",
		Regex:    `\b(amzn\.mws\.[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12})` + catalogRightBoundary,
		Keywords: []string{"amzn.mws."}, SecretGroup: 1,
		Source:      "https://github.com/amzn/selling-partner-api-models/blob/f944c32637cb796d3ea2f1a072bb384a366fabb1/models/authorization-api-model/authorization.json",
		Description: "Historical Amazon MWS authorization token, distinct from public seller/developer IDs and current SP-API credentials. Titus v1.2.9 supplies the amzn.mws. UUID-shaped subset; UUID version and continued service validity are not inferred.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/aws.yml
	},
	{
		ID:       "asana-personal-access-token",
		Regex:    `\b([0-9]+/[0-9]{16,}(?:/[0-9]{16,})?:[A-Za-z0-9]{32,})` + catalogRightBoundary,
		Keywords: []string{"/"}, SecretGroup: 1,
		Source:          "https://developers.asana.com/docs/personal-access-token",
		Description:     "Asana personal-access-token candidate containing the distinctive numeric-path and colon-separated secret components from TruffleHog v3.97.4. Ordinary numeric resource paths without the secret are excluded. Asana documents that tokens are confidential but their formats are opaque and may change; these historical widths are scanner subsets, not issuer validation. A 16 KiB defensive cap bounds result lifetime.",
		ValidateContext: validateAuditedAsanaToken1,
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/asanapersonalaccesstoken/asanapersonalaccesstoken.go
	},
	{
		ID:       "atlassian-api-token",
		Regex:    `(?:\b((?:ATATT|ATCTT3xFfG)[A-Za-z0-9+/=_-]{20,}=[A-Za-z0-9]{8})|\b(?:ATLASSIAN_API_TOKEN|atlassian_api_token|JIRA_API_TOKEN|jira_api_token|CONFLUENCE_API_TOKEN|confluence_api_token)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9]{24}))` + catalogRightBoundary,
		Keywords: []string{"ATATT", "ATCTT3xFfG", "atlassian_api_token", "jira_api_token", "confluence_api_token"}, SecretGroup: 0,
		Source:          "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Atlassian Cloud API-token candidates, including ATATT and ATCTT3xFfG forms with an eight-character terminal field, plus exact named legacy API-token assignments. Modern token lengths vary; the 20-character body floor is a conservative Titus-derived precision bound and the 16 KiB cap is defensive. Suffixes are not checksum-validated. Bare legacy 24-character strings and chunk-wide email/domain pairing are excluded.",
		Validate:        func(s string) bool { return len(s) <= maxStructuredCredentialBytes },
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/atlassian.yml
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/atlassian/v2/atlassian.go
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/jiratoken/v2/jiratoken_v2.go
	},
	{
		ID:       "authress-service-client-access-key",
		Regex:    `\b((?:sc|ext|scauth|authress)_[A-Za-z0-9]{5,30}\.[A-Za-z0-9]{4,6}\.acc[_-][A-Za-z0-9-]{10,32}\.[A-Za-z0-9+/_=-]{30,120})` + catalogRightBoundary,
		Keywords: []string{"sc_", "ext_", "scauth_", "authress_"}, SecretGroup: 1,
		Source:      "https://authress.io/knowledge-base/docs/authorization/service-clients/authress-implementation",
		Description: "Four-component Authress service-client key containing a parsed PKCS#8 Ed25519 private key in its final Base64 component. Public keys and arbitrary opaque final components are excluded. Prefix aliases and component widths follow Titus v1.2.9; the provider documents the component roles and private-key representation, not every scanner alias or width. No public-key lookup or JWT signing is performed.",
		Validate:    validAuditedAuthressKey1,
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/authress.yml
	},
	{
		ID:       "azure-api-management-repository-key",
		Regex:    `\b(git&[0-9]{12}&[A-Za-z0-9/+]{86}==)` + catalogRightBoundary,
		Keywords: []string{"git&"}, SecretGroup: 1,
		Source:      "https://learn.microsoft.com/en-us/rest/api/apimanagement/tenant-access-git/regenerate-primary-key?view=rest-apimanagement-2024-05-01",
		Description: "Historical Azure API Management Git repository password. TruffleHog v3.97.4 supplies the git&12-digits&Base64 shape; Base64 must encode exactly 64 bytes. The timestamp is not checked for expiration. Git configuration retirement does not turn previously confidential passwords into public IDs.",
		Validate:    validAuditedAPIMRepositoryKey1,
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/azureapimanagement/repositorykey/repositorykey.go
	},
	{
		ID:       "azure-container-registry-password",
		Regex:    `([A-Za-z0-9+/]{42}\+ACR[A-Za-z0-9]{6})` + catalogRightBoundary,
		Keywords: []string{"+ACR"}, SecretGroup: 1,
		Source:          "https://learn.microsoft.com/en-us/azure/container-registry/container-registry-authentication",
		Description:     "Azure Container Registry password candidate with the +ACR marker at the TruffleHog v3.97.4 fixed position. This is 52 characters (42+4+6), not an inferred 54-character length. Both boundaries reject embedding in a larger credential, including passwords beginning with plus/slash. No checksum or registry lookup is performed.",
		ValidateContext: validateAuditedACRPassword1,
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/azurecontainerregistry/azurecontainerregistry.go
	},
	{
		ID:       "azure-devops-personal-access-token",
		Regex:    `(?:\b([A-Za-z0-9]{76}AZDO[A-Za-z0-9]{4})|\b(?:ADO_PAT|ado_pat|AZURE_DEVOPS_PAT|azure_devops_pat|AZURE_DEVOPS_EXT_PAT|azure_devops_ext_pat)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9]{52}))` + catalogRightBoundary,
		Keywords: []string{"AZDO", "ado_pat", "azure_devops_pat", "azure_devops_ext_pat"}, SecretGroup: 0,
		Source:          "https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Azure DevOps password-equivalent PAT: documented 84-character modern AZDO form, or a 52-character legacy value in an exact Azure DevOps PAT assignment. The speculative 85-character Titus alternative and generic token/$token labels are excluded. No public organization ID or checksum check is included.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/azure.yml
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/azuredevops.yml
	},
	{
		ID:       "azure-entra-refresh-token",
		Regex:    `\b([01]\.A[A-Za-z0-9_-]{50,}(?:\.[0-9])?\.Ag[A-Za-z0-9_-]{250,}(?:\.A[A-Za-z0-9_-]{200,})?)` + catalogRightBoundary,
		Keywords: []string{"0.A", "1.A"}, SecretGroup: 1,
		Source:      "https://learn.microsoft.com/en-us/entra/identity-platform/refresh-tokens",
		Description: "Microsoft Entra refresh-token candidate with the pinned TruffleHog v3.97.4 multi-segment markers and minimum widths. The opaque token itself is a reusable credential even though Microsoft encrypts its internals; no decryption or recursive decoding is attempted. The 16 KiB total cap is defensive, not a protocol maximum.",
		Validate:    func(s string) bool { return len(s) <= maxStructuredCredentialBytes },
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/azure_entra/refreshtoken/refreshtoken.go
	},
	{
		ID:       "bannerbear-api-key",
		Regex:    `(?:\b(bb_(?:pr|ma)_[a-f0-9]{30})|\b(?:BANNERBEAR_API_KEY|bannerbear_api_key)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9]{22}tt))` + catalogRightBoundary,
		Keywords: []string{"bb_pr_", "bb_ma_", "bannerbear_api_key"}, SecretGroup: 0,
		Source:          "https://developers.bannerbear.com/v2/",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Bannerbear project/master Bearer API-key candidate. Modern prefixes and legacy 22-alphanumeric-plus-tt bodies follow TruffleHog v3.97.4 v2/v1; legacy values require the exact Bannerbear API-key assignment. Public project/template IDs and proximity-only strings are excluded.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/bannerbear/v1/bannerbear.go
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/bannerbear/v2/bannerbear.go
	},
	{
		ID:       "bitbucket-app-password",
		Regex:    `(?:\b(ATBB[A-Za-z0-9]{32})|\b[A-Za-z0-9_-]{1,30}:(ATBB[A-Za-z0-9_=.-]+)|\b(?:BITBUCKET_APP_PASSWORD|bitbucket_app_password)["']?[ \t]*[:=][ \t]*["']?(ATBB[A-Za-z0-9_=.-]+))` + catalogRightBoundary,
		Keywords: []string{"ATBB"}, SecretGroup: 0,
		Source:      "https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/",
		Description: "Historical Bitbucket app-password candidate: bare ATBB plus 32 alphanumerics from Titus v1.2.9, or TruffleHog v3.97.4's variable body in explicit username:password or Bitbucket app-password assignment carriers. Variable candidates are capped at 16 KiB. Public usernames and new Atlassian API-token families are distinct; retirement and validity are not checked.",
		Validate:    func(s string) bool { return len(s) <= maxStructuredCredentialBytes },
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/bitbucket.yml
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/bitbucketapppassword/bitbucketapppassword.go
	},
	{
		ID:       "bitbucket-datacenter-pat",
		Regex:    `\b(BBDC-[A-Za-z0-9+/@_-]{40,50})(?:$|[^A-Za-z0-9+/@_.=\-])`,
		Keywords: []string{"BBDC-"}, SecretGroup: 1,
		Source:      "https://confluence.atlassian.com/bitbucketserver/http-access-tokens-939515499.html",
		Description: "Bitbucket Data Center password-equivalent access-token candidate. BBDC- and the 40-50-character body follow TruffleHog v3.97.4; the right boundary includes the body-specific @ character. Neither the instance URL nor public token IDs are credentials.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/atlassiandatacenter/bitbucketdatacenter/bitbucketdatacenter.go
	},
	{
		ID:       "buildkite-legacy-api-token",
		Regex:    `\b(?:BUILDKITE_API_TOKEN|buildkite_api_token|BUILDKITE_ACCESS_TOKEN|buildkite_access_token)["']?[ \t]*[:=][ \t]*["']?([a-z0-9]{40})` + catalogRightBoundary,
		Keywords: []string{"buildkite_api_token", "buildkite_access_token"}, SecretGroup: 1,
		Source:          "https://buildkite.com/docs/apis/rest-api/access-token",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Legacy unprefixed Buildkite API token only in an exact Buildkite API/access-token assignment. The 40 lowercase-alphanumeric body is a TruffleHog v3.97.4 v1 constraint, not evidence that arbitrary Buildkite-adjacent hashes are credentials. Modern bkua_ tokens retain their existing native rule.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/buildkite/v1/buildkite.go
	},
	{
		ID:       "cerebras-api-key",
		Regex:    `\b(csk-[a-z0-9]{48})` + catalogRightBoundary,
		Keywords: []string{"csk-"}, SecretGroup: 1,
		Source:      "https://inference-docs.cerebras.ai/quickstart",
		Description: "Cerebras confidential Bearer API-key candidate. The case-sensitive csk- prefix and 48 lowercase-alphanumeric body follow Titus v1.2.9; unsupported entropy/digit requirements are not imported and no issuance claim is made.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/cerebras.yml
	},
	{
		ID:       "circleci-legacy-personal-access-token",
		Regex:    `\b(?:CIRCLECI_PERSONAL_ACCESS_TOKEN|circleci_personal_access_token|CIRCLECI_PERSONAL_TOKEN|circleci_personal_token)["']?[ \t]*[:=][ \t]*["']?([A-Fa-f0-9]{40})` + catalogRightBoundary,
		Keywords: []string{"circleci_personal_access_token", "circleci_personal_token"}, SecretGroup: 1,
		Source:          "https://circleci.com/changelog/new-format-for-api-access-tokens/",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Legacy 40-hex CircleCI personal token in an explicit personal-token assignment. Existing tokens survived the provider's 2023 format change. Generic Circle-Token/CIRCLE_TOKEN and project-token forms are not imported: project Status tokens can be published in build badges. Modern CCIPAT_ tokens retain their existing native rule.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/circleci/v1/circleci.go
	},
	{
		ID:       "clickup-personal-access-token",
		Regex:    `\b(pk_[0-9]{7,9}_[A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords: []string{"pk_"}, SecretGroup: 1,
		Source:      "https://developer.clickup.com/docs/authentication",
		Description: "ClickUp personal API-token candidate. The provider documents pk_ and authenticated user access; the 7-9 digit account and 32-character ASCII-alphanumeric body combine pinned TruffleHog and Betterleaks scanner subsets, not a complete issuance grammar. Prefix case is exact; unrelated pk_live_/pk_test_ publishable keys are excluded by the numeric account structure.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/clickuppersonaltoken/clickuppersonaltoken.go
		// Lowercase-body extension: Betterleaks 95237cf8eb4d8e9f67409595b245e674832992cf; see LICENSE.betterleaks.
	},
	{
		ID:       "cratesio-api-token",
		Regex:    `\b(cio[A-Za-z0-9]{32})` + catalogRightBoundary,
		Keywords: []string{catalogCIOKeyword}, SecretGroup: 1,
		Source:      "https://github.com/rust-lang/crates.io/blob/087b1b5bfd6b66cb200321776f2a162377cd37bb/crates/crates_io_database/src/utils/token.rs",
		Description: "crates.io secret API token using the issuer's cio prefix and 32-character alphanumeric generator. Public crate names, hashes and unprefixed values are not credentials. No scope or revocation verification is performed.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/crates.io.yml
	},
	{
		ID:       "deno-deploy-token",
		Regex:    `\b(dd[pw]_[A-Za-z0-9]{36})` + catalogRightBoundary,
		Keywords: []string{"ddp_", "ddw_"}, SecretGroup: 1,
		Source:      "https://docs.deno.com/subhosting/api/authentication/",
		Description: "Historical Deno Deploy ddp_/ddw_ Bearer-token candidates using the TruffleHog v3.97.4 prefixes and 36-character body. Deno documents bearer authorization separately from public organization IDs. These are scanner subsets, not a guarantee about current v2 token formats or the availability of retired APIs.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/deno/denodeploy.go
	},
	{
		ID:       "dependency-track-api-key",
		Regex:    `\b(odt_[A-Za-z0-9]{32,255})` + catalogRightBoundary,
		Keywords: []string{"odt_"}, SecretGroup: 1,
		Source:      "https://docs.dependencytrack.org/integrations/rest-api/",
		Description: "Dependency-Track prefixed API-key candidate. The odt_ prefix and 32-255 alphanumeric body follow Titus v1.2.9's supported subset; team IDs, stored hashes, legacy UUIDs and customized prefixes are excluded. Body widths are not a provider issuance guarantee.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/dependency_track.yml
	},
	{
		ID:       "discord-webhook",
		Regex:    `\b([Hh][Tt][Tt][Pp][Ss]://[Dd][Ii][Ss][Cc][Oo][Rr][Dd]\.[Cc][Oo][Mm]/api/(?:v[0-9]{1,2}/)?webhooks/[0-9]{17,20}/[A-Za-z0-9_-]{68})` + catalogRightBoundary,
		Keywords: []string{"discord.com/api/"}, SecretGroup: 1,
		Source:      "https://docs.discord.com/developers/resources/webhook",
		Description: "Complete Discord incoming-webhook credential URL, not a public webhook/snowflake ID. Exact official host/path and nonzero uint64 snowflake are checked. The 68-character URL-safe secret is the pinned Titus/TruffleHog subset; the API also documents other historical token lengths, which are not inferred into this rule. Scheme/host case handling is ASCII-only.",
		Validate:    validAuditedDiscordWebhook1,
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/discord.yml
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/discordwebhook/discordwebhook.go
	},
	{
		ID:       "discord-bot-token",
		Regex:    `\b([MNO][A-Za-z0-9_-]{20,30}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,40})` + catalogRightBoundary,
		Keywords: []string{"."}, SecretGroup: 1,
		Source:      "https://docs.discord.com/developers/reference#authentication",
		Description: "Discord bot-token candidate with a locally decoded decimal uint64 snowflake first component. The three-component body widths follow Titus v1.2.9 and TruffleHog v3.97.4; the provider's authentication example corroborates the encoded snowflake form. Arbitrary dotted strings, public IDs and JWTs are excluded. Timestamp/signature bodies remain opaque; no bot lookup or cryptographic verification.",
		Validate:    validAuditedDiscordBotToken1,
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/discord.yml
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/discordbottoken/discordbottoken.go
	},
	{
		ID:       "dockerhub-legacy-access-token",
		Regex:    `\b(?:DOCKERHUB_ACCESS_TOKEN|dockerhub_access_token|DOCKERHUB_TOKEN|dockerhub_token)["']?[ \t]*[:=][ \t]*["']?([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})` + catalogRightBoundary,
		Keywords: []string{"dockerhub_access_token", "dockerhub_token"}, SecretGroup: 1,
		Source:          "https://docs.docker.com/security/access-tokens/personal-access-tokens/",
		ValidateContext: validateAuditedAssignmentContext,
		Description:     "Historical UUID-shaped Docker Hub access token only in an exact DockerHub token assignment. The UUID shape is pinned TruffleHog v3.97.4 v1 evidence, not a general UUID secret heuristic; usernames and nearby UUIDs are excluded. Existing dckr_pat_/dckr_oat_ rules remain distinct.",
		// https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/dockerhub/v1/dockerhub.go
	},
	{
		ID:              "dropbox-access-token",
		Regex:           `(?:\b(sl\.(?:u\.)?[A-Za-z0-9_-]{130,})` + catalogRightBoundary + `|(?:^|[^A-Za-z0-9_.-])(?i:DROPBOX[_ .-](?:(?:ACCESS|API)[_ .-])?TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{11}AAAAAAAAAA[A-Za-z0-9_=-]{43})` + catalogRightBoundary + `)`,
		Keywords:        []string{"sl.", dropboxKeyword},
		SecretGroup:     0,
		Source:          "https://developers.dropbox.com/oauth-guide",
		Description:     "Dropbox OAuth bearer token. Original bare sl./sl.u. coverage retains its 130-character minimum, case-sensitive syntax and 16 KiB defensive cap. Historical 64-character long-lived candidates with the scanner's AAAAAAAAAA marker are additionally accepted only in exact Dropbox token assignments, with entropy/literal guards. The historical shape is a Betterleaks scanner subset, not an issuer grammar or current-issuance claim; public app/client IDs remain excluded.",
		Validate:        validBetterleaksCarrier1Dropbox,
		ValidateContext: validateBetterleaksCarrier1DropboxContext,
	},
	{
		ID:       "easypost-api-token",
		Regex:    `\b(EZ[AT]K[A-Za-z0-9]{54})` + catalogRightBoundary,
		Keywords: []string{"EZAK", "EZTK"}, SecretGroup: 1,
		Source:      "https://docs.easypost.com/docs/authentication",
		Description: "EasyPost production/test API key, used as the confidential Basic-auth username without a password. EZAK/EZTK and the 54-alphanumeric body follow Titus v1.2.9. Both roles are confidential per provider policy; public object IDs, entropy and digit-count heuristics are excluded.",
		// https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/easypost.yml
	},
}

func validAuditedAuthressKey1(s string) bool {
	encoded := s[strings.LastIndexByte(s, '.')+1:]
	var der [96]byte
	for _, encoding := range [...]*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		n, err := encoding.Strict().Decode(der[:], []byte(encoded))
		if err != nil {
			continue
		}
		key, err := x509.ParsePKCS8PrivateKey(der[:n])
		if err != nil {
			return false
		}
		_, ok := key.(ed25519.PrivateKey)
		return ok
	}
	return false
}

func validateAuditedAsanaToken1(value string, start, _ int, secret string) contextValidation {
	// A slash before the candidate would accept the tail of a longer path/token.
	return contextValidation{accepted: len(secret) <= maxStructuredCredentialBytes && (start == 0 || value[start-1] != '/')}
}

func validAuditedAPIMRepositoryKey1(s string) bool {
	var decoded [66]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s[strings.LastIndexByte(s, '&')+1:]))
	return err == nil && n == 64
}

func validateAuditedACRPassword1(value string, start, _ int, _ string) contextValidation {
	if start > 0 {
		c := value[start-1]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("_+/=.-", rune(c)) {
			return contextValidation{}
		}
	}
	return contextValidation{accepted: true}
}

func validAuditedDiscordWebhook1(s string) bool {
	_, path, ok := strings.Cut(s, "/webhooks/")
	if !ok {
		return false
	}
	id, _, _ := strings.Cut(path, "/")
	n, err := strconv.ParseUint(id, 10, 64)
	return err == nil && n != 0 && id[0] != '0'
}

func validAuditedDiscordBotToken1(s string) bool {
	first, _, _ := strings.Cut(s, ".")
	var decoded [24]byte
	n, err := base64.RawURLEncoding.Strict().Decode(decoded[:], []byte(first))
	if err != nil || n < 17 || n > 20 || decoded[0] == '0' {
		return false
	}
	for _, c := range decoded[:n] {
		if c < '0' || c > '9' {
			return false
		}
	}
	id, err := strconv.ParseUint(string(decoded[:n]), 10, 64)
	return err == nil && id != 0
}
