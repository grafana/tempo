package secrets

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

// Candidate constraints adapted from Titus v1.2.9 and TruffleHog v3.97.4.
// See LICENSE.titus / NOTICE.titus and the repository AGPL license. Provider
// semantics are documented per rule; scanner widths are not issuer guarantees.
var auditedCarriersRules1 = []catalogRuleSpec{
	{
		ID:              "appcenter-api-token",
		Regex:           `\b(?:[aA][pP][pP][_-]?[cC][eE][nN][tT][eE][rR][_ .-](?:[aA][pP][iI][_ .-]?)?[tT][oO][kK][eE][nN])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"appcenter", "app_center", "app-center"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/appcenter/api-docs/",
		Description:     "App Center API token in an explicit provider token assignment; retired-service credentials remain historical secrets. Public app-secret SDK identifiers are not imported.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "asana-client-secret",
		Regex:           `\b(?:[aA][sS][aA][nN][aA][_ .-](?:[oO][aA][uU][tT][hH][_ .-])?[cC][lL][iI][eE][nN][tT][_ .-]?[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{30,40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"asana"},
		SecretGroup:     0,
		Source:          "https://developers.asana.com/docs/oauth",
		Description:     "Explicit Asana OAuth client-secret assignment; client IDs and unlabelled provider-proximate strings are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "asana-access-token",
		Regex:           `\b(?:[aA][sS][aA][nN][aA][_ .-](?:[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[pP][eE][rR][sS][oO][nN][aA][lL][_ .-]?[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[pP][aA][tT]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([01]/[a-fA-F0-9]{16,32}(?::[A-Za-z0-9]{32,64})?)(?:$|[^A-Za-z0-9_+/=.:\-])`,
		Keywords:        []string{"asana"},
		SecretGroup:     0,
		Source:          "https://developers.asana.com/docs/personal-access-token",
		Description:     "Asana token assignment with the upstream 0/ or 1/ token carrier; widths and optional colon component are scanner constraints.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "alibaba-access-key-secret",
		Regex:           `(?:\b(?:(?:[aA][lL][iI][bB][aA][bB][aA]|[aA][lL][iI][yY][uU][nN])[_ .-](?:[aA][cC][cC][eE][sS][sS][_ .-]?[kK][eE][yY][_ .-]?[sS][eE][cC][rR][eE][tT]|[sS][eE][cC][rR][eE][tT][_ .-]?[aA][cC][cC][eE][sS][sS][_ .-]?[kK][eE][yY]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{30})|\b(?:[aA][cC][cC][eE][sS][sS][kK][eE][yY][iI][dD])["']?[ \t]*[:=][ \t]*["']?LTAI[A-Za-z0-9]{17,21}["']?[ \t,;\r\n]{1,16}(?:[aA][cC][cC][eE][sS][sS][kK][eE][yY][sS][eE][cC][rR][eE][tT])["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9]{30}))` + `(?:$|[^A-Za-z0-9_+/=.~\-])` + `|` + betterCompositeAlibabaCandidate3,
		Keywords:        []string{"alibaba", "aliyun", "AccessKeyId", "AccessKeySecret", "SecurityToken", "Expiration"},
		SecretGroup:     0,
		Source:          "https://www.alibabacloud.com/help/en/ram/user-guide/create-an-accesskey-pair",
		Description:     "Alibaba/Alibaba Cloud AccessKeySecret explicitly assigned in-value; the LTAI identifier alone is public. Also accepts the tightly paired native AccessKeyId/AccessKeySecret serialization. Temporary STS credentials require literal AccessKeyId, AccessKeySecret and CAIS SecurityToken together, in any assignment order or a complete flat JSON credential object or Credentials envelope. Plain assignments consume the complete value; double-quoted colon keys require JSON braces. Temporary body widths, 64-byte field gaps, ten-newline and 2048-byte whole-value bounds are scanner limits, not STS issuance limits.",
		ValidateContext: validateBetterCompositeAlibaba3,
	},
	{
		ID:              "aws-session-token",
		Regex:           `\b(?:[aA][wW][sS]_[sS][eE][sS][sS][iI][oO][nN]_[tT][oO][kK][eE][nN])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9/+=]{16,1000})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"aws_session_token"},
		SecretGroup:     0,
		Source:          "https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html",
		Description:     "Explicit AWS_SESSION_TOKEN assignment carrying the confidential session component. The 1000-byte candidate bound is local, not an STS maximum.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-client-secret-assignment",
		Regex:           `\b(?:(?:[aA][zZ][uU][rR][eE]|[aA][rR][mM])_[cC][lL][iI][eE][nN][tT]_[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_.~-]{16,80})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"azure_client_secret", "arm_client_secret"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/python/api/azure-identity/azure.identity.environmentcredential",
		Description:     "Exact Azure/ARM client-secret assignments expose opaque legacy service-principal passwords. Modern Q~ candidates remain owned by azure-ad-client-secret.",
		Validate:        validAzureLegacyClientSecret1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "beamer-api-key",
		Regex:           `\b(?:[bB][eE][aA][mM][eE][rR][_ .-](?:[aA][pP][iI][_ .-]?[kK][eE][yY]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?((?:b_[A-Za-z0-9+/=_-]{44}|[A-Za-z0-9_+/]{45}=))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"beamer"},
		SecretGroup:     0,
		Source:          "https://getbeamer-api.pages.dev/",
		Description:     "Explicit Beamer API-key/token assignment or Beamer-Api-Key header. The short b_ prefix is not classified without Beamer context.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "bitly-access-token",
		Regex:           `\b(?:[bB][iI][tT][lL][yY][_ .-](?:[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[aA][pP][iI][_ .-]?[tT][oO][kK][eE][nN]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"bitly"},
		SecretGroup:     0,
		Source:          "https://dev.bitly.com/docs/getting-started/authentication/",
		Description:     "Explicit Bitly access-token assignment, not a source hash near the product name.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "browserstack-access-key",
		Regex:           `\b(?:[bB][rR][oO][wW][sS][eE][rR][sS][tT][aA][cC][kK][_ .-][aA][cC][cC][eE][sS][sS][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{20})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"browserstack"},
		SecretGroup:     0,
		Source:          "https://www.browserstack.com/docs/automate/api-reference/selenium/introduction",
		Description:     "Explicit BrowserStack access-key assignment; this is the confidential Basic-auth password, not its username. HTTP userinfo variants are covered separately by http-userinfo-credential.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "clearbit-api-key",
		Regex:           `\b(?:[cC][lL][eE][aA][rR][bB][iI][tT][_ .-](?:[aA][pP][iI][_ .-]?[kK][eE][yY]|[sS][eE][cC][rR][eE][tT][_ .-]?[kK][eE][yY]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9_]{35})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"clearbit"},
		SecretGroup:     0,
		Source:          "https://raw.githubusercontent.com/clearbit/clearbit-node/master/README.md",
		Description:     "Clearbit API-key assignment; opaque 35-character values are not classified by proximity alone.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "cloudsight-api-key",
		Regex:           `\b(?:[cC][lL][oO][uU][dD][sS][iI][gG][hH][tT][_ .-](?:[aA][pP][iI][_ .-]?[kK][eE][yY]|[aA][pP][iI][_ .-]?[sS][eE][cC][rR][eE][tT]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{20,24})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"cloudsight"},
		SecretGroup:     0,
		Source:          "https://raw.githubusercontent.com/cloudsight/cloudsight-python/master/README.md",
		Description:     "CloudSight SimpleAuth API key or OAuth signing-secret assignment; response image tokens and provider prose are not credential carriers.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "codacy-api-token",
		Regex:           `\b(?:[cC][oO][dD][aA][cC][yY][_ .-](?:(?:[aA][pP][iI]|[pP][rR][oO][jJ][eE][cC][tT]|[aA][cC][cC][oO][uU][nN][tT])[_ .-])?[tT][oO][kK][eE][nN])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{20,24})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"codacy"},
		SecretGroup:     0,
		Source:          "https://docs.codacy.com/codacy-api/api-tokens/",
		Description:     "Codacy API/project token in an explicit provider token assignment; public repository IDs and bare opaque values are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "codeclimate-reporter-token",
		Regex:           `\b(?:[cC][oO][dD][eE][cC][lL][iI][mM][aA][tT][eE]_[rR][eE][pP][oO]_[tT][oO][kK][eE][nN]|[cC][cC]_[tT][eE][sS][tT]_[rR][eE][pP][oO][rR][tT][eE][rR]_[iI][dD])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{64})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"codeclimate_repo_token", "cc_test_reporter_id"},
		SecretGroup:     0,
		Source:          "https://github.com/codeclimate/ruby-test-reporter/issues/34",
		Description:     "Exact Code Climate repository upload credential assignments. These grant only coverage upload, but the provider recommends secrecy because others can submit data on the repository behalf.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "confluent-legacy-api-secret",
		Regex:           `\b(?:(?:[cC][oO][nN][fF][lL][uU][eE][nN][tT]|[cC][cC][lL][oO][uU][dD])[_ .-](?:[aA][pP][iI][_ .-]?)?(?:[sS][eE][cC][rR][eE][tT]|[sS][eE][cC][rR][eE][tT][_ .-]?[kK][eE][yY]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{64})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"confluent", "ccloud"},
		SecretGroup:     0,
		Source:          "https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/service-accounts/api-keys/overview.html",
		Description:     "Legacy Confluent API secret in an explicitly labelled Confluent/ccloud assignment. Modern checksummed cflt secrets remain owned by confluent-secret-key; public API key IDs are excluded.",
		Validate:        validConfluentLegacySecret1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "coveralls-personal-token",
		Regex:           `\b(?:[cC][oO][vV][eE][rR][aA][lL][lL][sS][_ .-](?:[pP][eE][rR][sS][oO][nN][aA][lL][_ .-]?[tT][oO][kK][eE][nN]|[aA][pP][iI][_ .-]?[tT][oO][kK][eE][nN]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{37})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"coveralls"},
		SecretGroup:     0,
		Source:          "https://docs.coveralls.io/api-repos-endpoint",
		Description:     "Explicit Coveralls personal/API token assignment, not the public service/owner/repository path. The supported 37-character body is a scanner constraint.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "coze-personal-access-token",
		Regex:           `\b(?:[cC][oO][zZ][eE][_ .-](?:[aA][pP][iI][_ .-]?[tT][oO][kK][eE][nN]|[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[pP][aA][tT]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(pat_[A-Za-z0-9]{64})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"coze"},
		SecretGroup:     0,
		Source:          "https://www.coze.com/docs/developer_guides/pat",
		Description:     "Coze-labelled personal-access-token assignment. The shared pat_ prefix is not attributed without the provider carrier.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "customerio-tracking-api-key",
		Regex:           `\b(?:(?:[cC][uU][sS][tT][oO][mM][eE][rR][._-]?[iI][oO]|[cC][iI][oO])[_ .-](?:[tT][rR][aA][cC][kK](?:[iI][nN][gG])?[_ .-])?[aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{20})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"customer", catalogCIOKeyword},
		SecretGroup:     0,
		Source:          "https://docs.customer.io/api/track/",
		Description:     "Customer.io Track API key assignment; this is the password half of site-ID Basic authentication, not the public site identifier.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "customerio-app-api-key",
		Regex:           `\b(?:(?:[cC][uU][sS][tT][oO][mM][eE][rR][._-]?[iI][oO]|[cC][iI][oO])[_ .-][aA][pP][pP][_ .-][aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{32})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"customer", catalogCIOKeyword},
		SecretGroup:     0,
		Source:          "https://docs.customer.io/api/app/",
		Description:     "Customer.io App API bearer key in an explicit app-API-key assignment; this does not infer a credential from generic customer text.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "datadog-api-key",
		Regex:           `\b(?:(?:[dD][aA][tT][aA][dD][oO][gG]|[dD][dD])[_ .-][aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{32})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"datadog", "dd"},
		SecretGroup:     0,
		Source:          "https://docs.datadoghq.com/account_management/api-app-keys/",
		Description:     "Datadog API-key assignment or DD-API-KEY header. Site domains and public browser RUM client tokens are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "datadog-application-key",
		Regex:           `\b(?:(?:[dD][aA][tT][aA][dD][oO][gG]|[dD][dD])[_ .-](?:[aA][pP][pP]|[aA][pP][pP][lL][iI][cC][aA][tT][iI][oO][nN])[_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"datadog", "dd"},
		SecretGroup:     0,
		Source:          "https://docs.datadoghq.com/account_management/api-app-keys/",
		Description:     "Datadog application-key assignment or DD-APPLICATION-KEY header, distinct from API keys and intentionally public RUM client tokens.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "delighted-api-key",
		Regex:           `\b(?:[dD][eE][lL][iI][gG][hH][tT][eE][dD][_ .-][aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{20,40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"delighted"},
		SecretGroup:     0,
		Source:          "https://raw.githubusercontent.com/delighted/delighted-python/master/README.md",
		Description:     "Delighted API-key assignment, including delighted.api_key SDK syntax; the provider explicitly treats this key as a password.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "deviantart-client-secret",
		Regex:           `\b(?:[dD][eE][vV][iI][aA][nN][tT][_-]?[aA][rR][tT][_ .-](?:[cC][lL][iI][eE][nN][tT]|[oO][aA][uU][tT][hH])[_ .-]?[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{32})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"deviantart", "deviant_art", "deviant-art"},
		SecretGroup:     0,
		Source:          "https://www.deviantart.com/developers/authentication",
		Description:     "Explicit DeviantArt OAuth client-secret assignment, excluding public client IDs and unrelated hexadecimal strings.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "deviantart-access-token",
		Regex:           `\b(?:[dD][eE][vV][iI][aA][nN][tT][_-]?[aA][rR][tT][_ .-](?:[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{40,80})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"deviantart", "deviant_art", "deviant-art"},
		SecretGroup:     0,
		Source:          "https://www.deviantart.com/developers/authentication",
		Description:     "Explicit DeviantArt OAuth access-token assignment; the opaque body is not recognized bare.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "disqus-api-secret",
		Regex:           `\b(?:[dD][iI][sS][qQ][uU][sS][_ .-](?:[aA][pP][iI][_ .-]?[sS][eE][cC][rR][eE][tT]|[sS][eE][cC][rR][eE][tT][_ .-]?[kK][eE][yY]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{64})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"disqus"},
		SecretGroup:     0,
		Source:          "https://disqus.com/api/docs/requests/",
		Description:     "Disqus server-side API secret assignment. Disqus api_key is public for JavaScript and is deliberately not accepted.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "directory-entry-credential",
		Regex:           `\bDirectoryEntry[ \t]*\([ \t]*"[^"\\\r\n]{1,1000}"[ \t]*,[ \t]*"[^"\\\r\n]{1,128}"[ \t]*,[ \t]*"([^"\\\r\n]{1,128})"[ \t]*(?:,|\))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"DirectoryEntry"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/dotnet/api/system.directoryservices.directoryentry.-ctor",
		Description:     "DirectoryEntry constructor with three literal arguments. The third argument is a password, independently of entropy; variable and interpolated arguments are excluded. Literal widths are local bounds.",
		Validate:        validAuditedCarrierLiteral1,
		ValidateContext: validateLiteralCredentialCall1,
	},
	{
		ID:          "aspnet-machine-key",
		Regex:       `\b(?:validationKey[ \t]*=[ \t]*(?:"((?:[A-Fa-f0-9]{2}){20,128})"|'((?:[A-Fa-f0-9]{2}){20,128})')|decryptionKey[ \t]*=[ \t]*(?:"([A-Fa-f0-9]{32}|[A-Fa-f0-9]{48}|[A-Fa-f0-9]{64})"|'([A-Fa-f0-9]{32}|[A-Fa-f0-9]{48}|[A-Fa-f0-9]{64})'))`,
		Keywords:    []string{"validationKey", "decryptionKey"},
		SecretGroup: 0,
		Source:      "https://learn.microsoft.com/en-us/dotnet/api/system.web.configuration.machinekeysection",
		Description: "ASP.NET validationKey/decryptionKey XML attributes containing literal hexadecimal secrets. Validation MAC keys use even-byte Titus bounds; AES/3DES decryption supports 16/24/32-byte keys. AutoGenerate and malformed/mismatched quotes are excluded.",
	},
	{
		ID:              "atlassian-datacenter-token",
		Regex:           `\b(?:(?:[jJ][iI][rR][aA]|[cC][oO][nN][fF][lL][uU][eE][nN][cC][eE]|[aA][tT][lL][aA][sS][sS][iI][aA][nN])[_ .-](?:(?:[dD][aA][tT][aA][_ .-]?[cC][eE][nN][tT][eE][rR]|[dD][cC])[_ .-])?(?:[pP][eE][rR][sS][oO][nN][aA][lL][_ .-]?[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[pP][aA][tT]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([MNO][A-Za-z0-9+/]{43})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"jira", "confluence", "atlassian"},
		SecretGroup:     0,
		Source:          "https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html",
		Description:     "Atlassian Data Center PAT in an explicit Jira/Confluence token assignment. Strict Base64 decoding checks the pinned scanner structure of twelve decimal identifier bytes, colon, and twenty secret bytes. Bare encoded strings are excluded.",
		Validate:        validAtlassianDataCenterToken1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "aws-access-key-pair",
		Regex:       `\b(?:AKIA|ASIA)[A-Z0-9]{16}["']?[ \t\r\n]{0,8}[,:;][ \t\r\n]{0,8}["']?([A-Za-z0-9/+=]{40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:    []string{"AKIA", "ASIA"},
		SecretGroup: 0,
		Source:      "https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html",
		Description: "Tightly serialized AWS access-key-ID/signing-secret pair, including CSV and colon-separated credentials. Only actual AKIA/ASIA access-key IDs qualify; principal/account/resource IDs and arbitrary chunk-wide proximity do not.",
	},
	{
		ID:              "auth0-client-secret",
		Regex:           `(?:\b(?:[aA][uU][tT][hH]0_[cC][lL][iI][eE][nN][tT]_[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{32,128})|\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.auth0\.com/oauth/token[^{}\r\n]{0,100}[{]?[ \t]*\b(?:[cC][lL][iI][eE][nN][tT]_[iI][dD])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(?:[A-Za-z0-9_-]{32,60})["', \t]{1,16}\b(?:[cC][lL][iI][eE][nN][tT]_[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{32,128}))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"auth0_client_secret", "auth0.com"},
		SecretGroup:     0,
		Source:          "https://auth0.com/docs/secure/application-credentials",
		Description:     "Auth0 confidential client-secret assignment; public client IDs and tenant domains do not qualify. Endpoint-bound OAuth client_id/client_secret request fragments are also recognized.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "box-oauth-client-secret",
		Regex:           `(?:\b(?:[bB][oO][xX]_[cC][lL][iI][eE][nN][tT]_[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{32})|"boxAppSettings"[ \t\r\n]*:[ \t\r\n]*\{[ \t\r\n]*"clientID"[ \t\r\n]*:[ \t\r\n]*"[A-Za-z0-9]{32}"[ \t\r\n]*,[ \t\r\n]*"clientSecret"[ \t\r\n]*:[ \t\r\n]*"([A-Za-z0-9]{32})")` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"box_client_secret", "boxAppSettings"},
		SecretGroup:     0,
		Source:          "https://developer.box.com/guides/authentication/client-credentials/client-credentials-setup/",
		Description:     "Box confidential client-secret assignment or Box application-settings JSON clientID/clientSecret bundle. Public enterprise, application and subject IDs alone are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "calendly-personal-access-token",
		Regex:           `\b(?:[cC][aA][lL][eE][nN][dD][lL][yY][_ .-](?:[pP][eE][rR][sS][oO][nN][aA][lL][_ .-]?[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[aA][pP][iI][_ .-]?[tT][oO][kK][eE][nN]|[pP][aA][tT]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]+.[A-Za-z0-9_-]+.[A-Za-z0-9_-]+)` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"calendly"},
		SecretGroup:     0,
		Source:          "https://developer.calendly.com/how-to-authenticate-with-personal-access-tokens",
		Description:     "Calendly PAT assignment containing a locally parsed signed JWT. Bare JWTs, unsecured tokens and invalid signature framing remain excluded.",
		Validate:        validJWT,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "cisco-plaintext-enable-password",
		Regex:       `(?m)^[ \t]*enable[ \t]+(?:password|secret)[ \t]+(?:level[ \t]+(?:[0-9]|1[0-5])[ \t]+)?(?:0[ \t]+)?([^\s"'\\]{1,128})[ \t]*\r?$`,
		Keywords:    []string{"enable"},
		SecretGroup: 0,
		Source:      "https://www.cisco.com/c/en/us/td/docs/ion-en_US/ios-xml/ios/security/d1/sec-d1-cr-book/sec-cr-e1.html",
		Description: "Cisco enable password/secret commands containing plaintext input (implicit type or type 0). Hash types 5/8/9, encrypted types 6/7, negated commands and comments are not plaintext-value secrets.",
		Validate:    validAuditedCarrierLiteral1,
	},
	{
		ID:          "cisco-plaintext-pac-key",
		Regex:       `(?m)^[ \t]*pac[ \t]+key[ \t]+0[ \t]+([^\s"'\\]{1,128})[ \t]*\r?$`,
		Keywords:    []string{"pac"},
		SecretGroup: 0,
		Source:      "https://www.cisco.com/c/en/us/td/docs/ion-en_US/ios-xml/ios/security/m1/sec-m1-cr-book/pac_key_through_port_misuse.html",
		Description: "Cisco EAP-FAST PAC encryption-key command with explicit plaintext type 0. Type 6 AES ciphertext and type 7 hidden encodings are outside the assigned plaintext scope.",
		Validate:    validAuditedCarrierLiteral1,
	},
	{
		ID:              "cloudinary-api-credential",
		Regex:           `(?:\b([Cc][Ll][Oo][Uu][Dd][Ii][Nn][Aa][Rr][Yy]://[^\s"<>\x60]+)|\b(?:[cC][lL][oO][uU][dD][iI][nN][aA][rR][yY][_ .-][aA][pP][iI][_ .-]?[sS][eE][cC][rR][eE][tT])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9_-]{27}|[A-Za-z0-9]{32})(?:$|[^A-Za-z0-9_+/=.~\-]))`,
		Keywords:        []string{"cloudinary://", "cloudinary"},
		SecretGroup:     0,
		Source:          "https://cloudinary.com/documentation/node_integration#setting_the_cloudinary_url_environment_variable",
		Description:     "Cloudinary complete credential URI or exact API-secret assignment. Preserves existing URI framing, percent escapes, public-key and cloud-name checks. Assignment body widths 27 and 32 are supported scanner subsets, not issuer guarantees; public API keys and cloud names alone are excluded.",
		ValidateContext: validateBetterleaksCloudinaryCredential1,
	},
	{
		ID:              "couchbase-credential-uri",
		Regex:           `\b([Cc][Oo][Uu][Cc][Hh][Bb][Aa][Ss][Ee][Ss]?://[^\s"<>\x60]+)`,
		Keywords:        []string{"couchbase://", "couchbases://"},
		SecretGroup:     0,
		Source:          "https://docs.couchbase.com/go-sdk/current/howtos/managing-connections.html",
		Description:     "Couchbase URI containing explicit userinfo password, including Capella couchbases hosts. Public connection endpoints, passwordless users and query-only passwords are excluded. This is a URI credential candidate, not proof that a particular SDK accepts userinfo.",
		ValidateContext: validateCouchbaseCredentialURI1,
	},
	{
		ID:              "cypress-record-key",
		Regex:           `(?:\b(?:[cC][yY][pP][rR][eE][sS][sS]_[rR][eE][cC][oO][rR][dD]_[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12})|\bcypress[ \t]+run[ \t]+--record[ \t]+--key(?:=|[ \t]+)["']?([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"cypress_record_key", "cypress"},
		SecretGroup:     0,
		Source:          "https://docs.cypress.io/cloud/account-management/projects#Record-key",
		Description:     "Cypress record-key assignment or cypress run --record --key carrier. This is an upload/recording credential, not a public project ID or arbitrary UUID.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "database-literal-credentials",
		Regex:           `\b(?:DBI->connect|DBI\.connect|mysql_connect|mysql_pconnect|mysqli_connect)[ \t]*\([ \t]*(?:"[^"\\\r\n]{1,1000}"|'[^'\\\r\n]{1,1000}')[ \t]*,[ \t]*(?:"[^"\\\r\n]{1,128}"|'[^'\\\r\n]{1,128}')[ \t]*,[ \t]*(?:"([^"\\\r\n]{1,128})"|'([^'\\\r\n]{1,128})')[ \t]*(?:,|\))`,
		Keywords:        []string{"DBI", "mysql_connect", "mysql_pconnect", "mysqli_connect"},
		SecretGroup:     0,
		Source:          "https://metacpan.org/pod/DBI",
		Description:     "Perl/Ruby DBI and PHP MySQL connection constructors with three literal arguments. Quoting is language-aware: PHP/Perl double-quoted variable expansion and Ruby double-quoted interpolation are excluded, while single-quoted dollar/hash characters and Ruby plain dollar characters remain literal. Backslash escapes remain outside this bounded source-literal subset.",
		ValidateContext: validateDatabaseLiteralCall1,
	},
	{
		ID:              "python-database-password",
		Regex:           `(?:\b(?:psycopg2|mysql\.connector|pymysql|cx_Oracle)\.connect[ \t]*\([ \t]*(?:(?:host|port|user|username|database|dbname|dsn|sslmode)[ \t]*=[ \t]*(?:"[^"\\\r\n]{0,120}"|'[^'\\\r\n]{0,120}'|[0-9]{1,5})[ \t]*,[ \t]*){0,8}password[ \t]*=[ \t]*(?:"([^"\\\r\n]{1,128})"|'([^'\\\r\n]{1,128})')[ \t]*(?:,|\))|\b(?:mysql\.connector|pymysql)\.connect[ \t]*\([ \t]*(?:(?:host|port|user|username|database|dbname|dsn|sslmode)[ \t]*=[ \t]*(?:"[^"\\\r\n]{0,120}"|'[^'\\\r\n]{0,120}'|[0-9]{1,5})[ \t]*,[ \t]*){0,8}passwd[ \t]*=[ \t]*(?:"([^"\\\r\n]{1,128})"|'([^'\\\r\n]{1,128})')[ \t]*(?:,|\)))`,
		Keywords:        []string{"psycopg2", "mysql.connector", "pymysql", "cx_Oracle"},
		SecretGroup:     0,
		Source:          "https://www.psycopg.org/docs/module.html",
		Description:     "Recognized Python database driver connect call with literal password keyword, after zero or more literal connection parameters. sqlite3 and expressions are excluded; aliases passwd are accepted only for MySQL drivers.",
		Validate:        validAuditedCarrierLiteral1,
		ValidateContext: validateLiteralCredentialCall1,
	},
	{
		ID:          "django-secret-key",
		Regex:       `# SECURITY WARNING: keep the secret key used in production secret![ \t\r\n]{1,32}SECRET_KEY[ \t]*=[ \t]*(?:r?"([^"\\\r\n]{5,100})"|r?'([^'\\\r\n]{5,100})')`,
		Keywords:    []string{"SECRET_KEY"},
		SecretGroup: 0,
		Source:      "https://docs.djangoproject.com/en/5.1/ref/settings/#std-setting-SECRET_KEY",
		Description: "Django production-secret warning followed by a literal SECRET_KEY assignment. This key signs sessions/tokens; template or symbolic values and a bare generic SECRET_KEY assignment are not attributed to Django.",
		Validate:    validAuditedCarrierLiteral1,
	},
	{
		ID:              "azure-search-api-key",
		Regex:           `(?:\b(?:[aA][zZ][uU][rR][eE][_ .-][sS][eE][aA][rR][cC][hH][_ .-](?:(?:[aA][dD][mM][iI][nN]|[qQ][uU][eE][rR][yY]|[aA][pP][iI])[_ .-])?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{52})|(?:\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.search\.windows\.net(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)[^{}\r\n]{0,256}?\b(?:[aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{52})|\b(?:[aA][pP][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{52})(?:$|[^A-Za-z0-9_+/=.~\-])[^{}\r\n]{0,256}?\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.search\.windows\.net(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{catalogAzureKeyword, "search.windows.net"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/search/search-security-api-keys",
		Description:     "Azure AI Search API-key assignment or endpoint-bound api-key request/config fragment. Admin versus query privileges cannot be classified offline; public search endpoints alone are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-apim-subscription-key",
		Regex:           `\b(?:[oO][cC][pP]-[aA][pP][iI][mM]-[sS][uU][bB][sS][cC][rR][iI][pP][tT][iI][oO][nN]-[kK][eE][yY]|[sS][uU][bB][sS][cC][rR][iI][pP][tT][iI][oO][nN][-_][kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{32})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"ocp-apim-subscription-key", "subscription-key", "subscription_key"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/api-management/api-management-subscriptions",
		Description:     "Azure APIM subscription key in the documented header or subscription-key query/config field. Keys may be customized; the supported 32-character body follows the pinned audit scanners.",
		ValidateContext: validateAPIMSubscriptionKeyContext1,
	},
	{
		ID:              "azure-apim-direct-management-key",
		Regex:           `(?:\b(?:[aA][zZ][uU][rR][eE][_ .-][aA][pP][iI][mM][_ .-](?:(?:[pP][rR][iI][mM][aA][rR][yY]|[sS][eE][cC][oO][nN][dD][aA][rR][yY])[_ .-])?(?:[mM][aA][nN][aA][gG][eE][mM][eE][nN][tT][_ .-])?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)|(?:\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.management\.azure-api\.net(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)[^{}\r\n]{0,256}?\b(?:[pP][rR][iI][mM][aA][rR][yY][_ .-]?[kK][eE][yY]|[sS][eE][cC][oO][nN][dD][aA][rR][yY][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)|\b(?:[pP][rR][iI][mM][aA][rR][yY][_ .-]?[kK][eE][yY]|[sS][eE][cC][oO][nN][dD][aA][rR][yY][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)(?:$|[^A-Za-z0-9_+/=.~\-])[^{}\r\n]{0,256}?\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.management\.azure-api\.net(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{catalogAzureKeyword, "management.azure-api.net"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/rest/api/apimanagement/apimanagementrest/azure-api-management-rest-api-authentication",
		Description:     "APIM direct-management primary/secondary signing key in an exact APIM assignment or management-endpoint-bound key field. Base64 decoding requires the supported 64-byte key; subscription IDs and derived SAS tokens are different families.",
		Validate:        validAzureAccountKey1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-app-configuration-secret",
		Regex:           `\b(?:Endpoint[ \t]*=[ \t]*)?[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.azconfig\.io;[ \t]*Id=[A-Za-z0-9+/:_\-]{4,100};[ \t]*Secret=([A-Za-z0-9+/]{16,128}={0,2})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"azconfig.io"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/azure-app-configuration/howto-best-practices",
		Description:     "Azure App Configuration endpoint/Id/Secret connection string with a strictly decoded Base64 secret. Endpoint and ID alone, access-token references and placeholders are not secrets.",
		Validate:        validAuditedBase64Secret1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-shared-access-key",
		Regex:           `\b(?:[sS][hH][aA][rR][eE][dD][aA][cC][cC][eE][sS][sS][kK][eE][yY][nN][aA][mM][eE]|[sS][hH][aA][rR][eE][dD][sS][eE][cC][rR][eE][tT][iI][sS][sS][uU][eE][rR])[ \t]*=[ \t]*[A-Za-z0-9_.@\-]{1,80}[ \t]*;[ \t]*(?:[sS][hH][aA][rR][eE][dD][aA][cC][cC][eE][sS][sS][kK][eE][yY]|[sS][hH][aA][rR][eE][dD][sS][eE][cC][rR][eE][tT][vV][aA][lL][uU][eE])[ \t]*=[ \t]*([A-Za-z0-9+/]{20,100}={0,2})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"SharedAccessKeyName", "SharedSecretIssuer"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/service-bus-messaging/service-bus-sas",
		Description:     "Named Azure SAS policy/signing-key or legacy issuer/shared-secret pair. Service Bus and Notification Hubs share this carrier; public policy names and a bare SharedAccessKey token are not classified.",
		Validate:        validAuditedBase64Secret1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-storage-account-key",
		Regex:           `(?:\b(?:[aA][zZ][uU][rR][eE][_ .-][sS][tT][oO][rR][aA][gG][eE][_ .-](?:[aA][cC][cC][oO][uU][nN][tT][_ .-])?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)|\bAccountName[ \t]*=[ \t]*[a-z0-9]{3,24}[ \t]*;[ \t]*AccountKey[ \t]*=[ \t]*([A-Za-z0-9+/]{86}==))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{catalogAzureKeyword, "AccountName"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/storage/common/storage-configure-connection-string",
		Description:     "Azure Storage account key in a provider assignment or AccountName/AccountKey connection pair. Strict 64-byte Base64 recognition rejects the documented emulator key; account names and endpoint-only strings are public.",
		Validate:        validAzureAccountKey1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-batch-account-key",
		Regex:           `(?:\b(?:[aA][zZ][uU][rR][eE][_ .-][bB][aA][tT][cC][hH][_ .-](?:[aA][cC][cC][oO][uU][nN][tT][_ .-])?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)|(?:\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.[A-Za-z0-9-]{1,63}\.batch\.azure\.com(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)[^{}\r\n]{0,256}?\b(?:[aA][cC][cC][oO][uU][nN][tT][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)|\b(?:[aA][cC][cC][oO][uU][nN][tT][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9+/]{86}==)(?:$|[^A-Za-z0-9_+/=.~\-])[^{}\r\n]{0,256}?\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.[A-Za-z0-9-]{1,63}\.batch\.azure\.com(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{catalogAzureKeyword, "batch.azure.com"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/batch/batch-account-create-portal",
		Description:     "Azure Batch account signing key in an exact provider label or endpoint-bound AccountKey field. Requires the supported 64-byte Base64 key, not chunk-wide key/URL cross-products.",
		Validate:        validAzureAccountKey1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-cosmosdb-account-key",
		Regex:           `\bAccountEndpoint[ \t]*=[ \t]*https://[A-Za-z0-9-]{1,63}\.(?:documents\.azure\.com|table\.cosmos\.azure\.com)(?::443)?/?;[ \t]*AccountKey[ \t]*=[ \t]*([A-Za-z0-9+/]{86}==)` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"AccountEndpoint"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/cosmos-db/database-security",
		Description:     "Azure Cosmos DB AccountEndpoint/AccountKey connection string, including Table API endpoints. The 64-byte Base64 signing key is confidential; endpoint URLs and empty/placeholder keys are not.",
		Validate:        validAzureAccountKey1,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-openai-api-key",
		Regex:           `(?:\b(?:[aA][zZ][uU][rR][eE][_-][oO][pP][eE][nN][aA][iI][_-](?:[aA][pP][iI][_-])?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?((?:[A-Fa-f0-9]{32}|[A-Za-z0-9+/]{60,130}={0,2}))|(?:\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.openai\.azure\.com(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)[^{}\r\n]{0,256}?\b(?:[aA][pP][iI][_ .-]?[kK][eE][yY]|[oO][pP][eE][nN][aA][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?((?:[A-Fa-f0-9]{32}|[A-Za-z0-9+/]{60,130}={0,2}))|\b(?:[aA][pP][iI][_ .-]?[kK][eE][yY]|[oO][pP][eE][nN][aA][iI][_ .-]?[kK][eE][yY])["']?[ \t]{0,16}[:=][ \t]{0,16}["']?((?:[A-Fa-f0-9]{32}|[A-Za-z0-9+/]{60,130}={0,2}))(?:$|[^A-Za-z0-9_+/=.~\-])[^{}\r\n]{0,256}?\b[Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9-]{1,63}\.openai\.azure\.com(?::443)?(?:[/?][^\s"'<>]{0,200})?(?:[ \t"';]|$)))` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{catalogAzureKeyword, "openai.azure.com"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/azure/ai-services/authentication",
		Description:     "Azure OpenAI key in an exact Azure/OpenAI credential assignment or endpoint-bound api-key field. Supports legacy hex and longer opaque keys without inventing JQQJ/AAAB/ACOGk issuer-marker guarantees; widths remain defensive scanner constraints.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "azure-storage-sas-url",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]://[a-z0-9]{3,24}\.blob\.core\.windows\.net/[^\s"<>\x60]+)`,
		Keywords:        []string{"blob.core.windows.net"},
		SecretGroup:     0,
		Source:          "https://learn.microsoft.com/en-us/rest/api/storageservices/create-service-sas",
		Description:     "Azure Blob SAS URL with parsed, order-independent permissions/version/resource/time/signature parameters. A 32-byte decoded signature is required; no expiry, permissions grant, HMAC or remote validation is claimed.",
		ValidateContext: validateAzureStorageSASURL1,
	},
	{
		ID:              "blynk-device-token-url",
		Regex:           `(\bhttps://(?:[a-z0-9]{1,12}\.)?blynk\.cloud/external/api/[^\s"<>\x60]+)`,
		Keywords:        []string{blynkHost},
		SecretGroup:     0,
		Source:          "https://docs.blynk.io/en/blynk.cloud/device-https-api",
		Description:     "Blynk external device API URL containing a token query parameter. Exact endpoint/path attribution and the 32-character opaque token are required; query order is independent.",
		ValidateContext: validateBlynkDeviceURL1,
	},
	{
		ID:              "blynk-organization-token",
		Regex:           `(?:\bhttps://(?:[a-z0-9]{1,12}\.)?blynk\.cloud/api/[A-Za-z0-9_/?=&.-]{0,200}[ \t\\"'\-H]{1,32}\b(?:[aA][uU][tT][hH][oO][rR][iI][zZ][aA][tT][iI][oO][nN])[ \t]*:[ \t]*(?:[bB][eE][aA][rR][eE][rR])[ \t]+([A-Za-z0-9_-]{40})|\b(?:[aA][uU][tT][hH][oO][rR][iI][zZ][aA][tT][iI][oO][nN])[ \t]*:[ \t]*(?:[bB][eE][aA][rR][eE][rR])[ \t]+([A-Za-z0-9_-]{40})[ \t\\"']{1,32}\bhttps://(?:[a-z0-9]{1,12}\.)?blynk\.cloud/api/[A-Za-z0-9_/?=&.-]{0,200})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{blynkHost},
		SecretGroup:     0,
		Source:          "https://docs.blynk.io/en/blynk.cloud/organization-https-api/get-organization-info",
		Description:     "Blynk organization API endpoint immediately paired with an Authorization Bearer credential in either order; opaque tokens are not recognized without this complete carrier.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "blynk-oauth-client-secret",
		Regex:       `(?:\bhttps://(?:[a-z0-9]{1,12}\.)?blynk\.cloud/oauth2(?:/[A-Za-z0-9_-]{0,80})?[?&=A-Za-z0-9_ \t\\"'\-]{0,100}\boa2-client-id_[A-Za-z0-9_-]{32}(?::|&client_secret=)([A-Za-z0-9_-]{40})|\boa2-client-id_[A-Za-z0-9_-]{32}(?::|&client_secret=)([A-Za-z0-9_-]{40})[ \t\\"']{1,32}\bhttps://(?:[a-z0-9]{1,12}\.)?blynk\.cloud/oauth2(?:/[A-Za-z0-9_-]{0,80})?)` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:    []string{blynkHost},
		SecretGroup: 0,
		Source:      "https://docs.blynk.io/en/blynk.cloud/organization-https-api/authentication",
		Description: "Blynk OAuth endpoint paired with the oa2-client-id_ identifier and literal client secret in either order. The identifier alone is public; arbitrary opaque strings near blynk are not accepted.",
	},
	{
		ID:          "coinbase-cdp-private-key",
		Regex:       `\{[^{}]{1,1000}\}`,
		Keywords:    []string{"{"},
		SecretGroup: 0,
		Source:      "https://docs.cdp.coinbase.com/get-started/authentication/jwt-authentication",
		Description: "Coinbase CDP flat JSON credential with UUID id and 64-byte Base64 Ed25519 seed/public-key encoding, or organizations/.../apiKeys/... name and JSON-escaped unencrypted PEM. The public name/id alone, malformed JSON and public/encrypted keys are excluded.",
		Validate:    validCoinbaseCDPPrivateKey1,
	},
	{
		ID:          "curl-user-credential",
		Regex:       `\bcurl[ \t]+[^\r\n]{0,800}(?:-u|--user)[^\r\n]{1,200}`,
		Keywords:    []string{curlCommand},
		SecretGroup: 0,
		Source:      "https://curl.se/docs/manpage.html#-u",
		Description: "Literal curl -u/--user username:password credential. A bounded shell-word parser honors single/double quotes, option operands and attached/equal option forms, without executing substitutions. Unknown option grammars are conservatively rejected.",
		Validate:    validCurlUserCredential1,
	},
	{
		ID:              "codeclimate-api-token",
		Regex:           `\b(?:[cC][oO][dD][eE][cC][lL][iI][mM][aA][tT][eE][_ .-](?:[aA][pP][iI][_ .-]?[tT][oO][kK][eE][nN]|[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]|[tT][oO][kK][eE][nN]))["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{40})` + `(?:$|[^A-Za-z0-9_+/=.~\-])`,
		Keywords:        []string{"codeclimate"},
		SecretGroup:     0,
		Source:          "https://docs.codeclimate.com/v1.0/docs/api",
		Description:     "Historical Code Climate personal API access token in an explicit provider token assignment. This authenticates API access, unlike the distinct 64-hex reporter upload token; bare source hashes are excluded.",
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "coinbase-oauth-access-token",
		Regex:           `\b[cC][oO][iI][nN][bB][aA][sS][eE][_ .-](?:[oO][aA][uU][tT][hH][_ .-])?[aA][cC][cC][eE][sS][sS][_ .-]?[tT][oO][kK][eE][nN]["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9-]{32}|[A-Za-z0-9_-]{64})` + catalogRightBoundary,
		Keywords:        []string{"coinbase"},
		SecretGroup:     1,
		Source:          "https://docs.cdp.coinbase.com/coinbase-app/oauth2-integration/integrations",
		Description:     "Exact Coinbase OAuth access-token assignments retain the historical 32-character scope and add 64-character URL-safe scanner candidates. Provider documentation demonstrates 64-hex access tokens but also similarly shaped public client IDs, so the access-token role remains mandatory; no issuer verification.",
		ValidateContext: validateBetterleaksEvidenceAssignment1,
	},
}

// These recognizers retain only bounded candidate data. They do not decode
// arbitrary surrounding text or perform authentication/network verification.
func validAuditedCarrierLiteral1(s string) bool {
	if s == "" || len(s) > maxStructuredCredentialBytes || strings.TrimSpace(s) == "" {
		return false
	}
	for _, marker := range []string{"\u0024{", "\u0024(", "{{", "#{", "<", ">"} {
		if strings.Contains(s, marker) {
			return false
		}
	}
	switch strings.ToLower(s) {
	case passwordField, passwdField, secretField, placeholderChangeMe, redactedLiteral, placeholderLiteral, placeholderYourPassword, placeholderYourSecret, placeholderYourToken, placeholderYourAPIKey, "\u0024password", "\u0024passwd", "\u0024secret", "\u0024token", nullLiteral, noneLiteral:
		return false
	}
	return strings.Trim(s, "*") != ""
}

func validateLiteralCredentialCall1(value string, start, _ int, _ string) contextValidation {
	line := value[strings.LastIndexByte(value[:start], '\n')+1 : start]
	line = strings.TrimSpace(line)
	return contextValidation{accepted: !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "/*") && !strings.HasPrefix(line, "*")}
}

func validateDatabaseLiteralCall1(value string, start, end int, secret string) contextValidation {
	if !validateLiteralCredentialCall1(value, start, end, secret).accepted {
		return contextValidation{}
	}
	call := value[start:end]
	ruby := strings.HasPrefix(call, "DBI.connect")
	perl := strings.HasPrefix(call, "DBI->connect")
	open := strings.IndexByte(call, '(')
	if open < 0 {
		return contextValidation{}
	}
	rest := call[open+1:]
	for argument := range 3 {
		rest = strings.TrimLeft(rest, " \t")
		if len(rest) < 2 || rest[0] != '\'' && rest[0] != '"' {
			return contextValidation{}
		}
		quote := rest[0]
		closingIndex := strings.IndexByte(rest[1:], quote)
		if closingIndex < 0 {
			return contextValidation{}
		}
		literal := rest[1 : closingIndex+1]
		if quote == '"' && interpolatedDatabaseString1(literal, ruby, perl) {
			return contextValidation{}
		}
		rest = strings.TrimLeft(rest[closingIndex+2:], " \t")
		if argument < 2 {
			if rest == "" || rest[0] != ',' {
				return contextValidation{}
			}
			rest = rest[1:]
		}
	}
	// Source interpolation was handled above. Dollar/hash syntax in a
	// non-interpolating string must not be mistaken for a template here.
	switch strings.ToLower(secret) {
	case "", passwordField, passwdField, secretField, placeholderChangeMe, redactedLiteral, placeholderLiteral, placeholderYourPassword, placeholderYourSecret, placeholderYourToken, placeholderYourAPIKey, nullLiteral, noneLiteral:
		return contextValidation{}
	}
	return contextValidation{accepted: strings.TrimSpace(secret) != "" && strings.Trim(secret, "*") != ""}
}

func interpolatedDatabaseString1(s string, ruby, perl bool) bool {
	for i := range len(s) - 1 {
		c, next := s[i], s[i+1]
		if ruby {
			if c == '#' && (next == '{' || next == '@' || next == '$') {
				return true
			}
			continue
		}
		if perl {
			// Perl also interpolates special/positional scalar variables.
			if c == '$' || c == '@' && (isASCIIWordByte(next) || next >= 0x80 || next == '{' || next == '+' || next == '-') {
				return true
			}
			continue
		}
		// PHP variable names cannot begin with a digit; "$5" is literal.
		if c == '$' && (next >= 'a' && next <= 'z' || next >= 'A' && next <= 'Z' || next == '_' || next >= 0x80 || next == '{' || next == '$') {
			return true
		}
	}
	return false
}

func validateAPIMSubscriptionKeyContext1(value string, start, end int, secret string) contextValidation {
	if start == 0 || value[start-1] != '?' && value[start-1] != '&' {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	// Query separators are not expression operators. Preserve the complete
	// URL parameter and reject a token prefix inside a longer query value.
	const delimiters = " \t\r\n\"'<>`"
	left := start
	for left > 0 && start-left <= maxStructuredCredentialBytes && !strings.ContainsRune(delimiters, rune(value[left-1])) {
		left--
	}
	scheme := indexFoldedASCII(value[left:start], "https://")
	if scheme < 0 {
		scheme = indexFoldedASCII(value[left:start], "http://")
	}
	if scheme > 0 && value[left+scheme-1] == '=' {
		left += scheme
	}
	right := start
	for right < len(value) && right-left <= maxStructuredCredentialBytes && !strings.ContainsRune(delimiters, rune(value[right])) {
		right++
	}
	if right-left > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(value[left:right])
	if err != nil || u.Host == "" || (u.Scheme != httpScheme && u.Scheme != httpsScheme) || strings.Count(u.RawQuery, "&") > 32 {
		return contextValidation{}
	}
	name, _, found := strings.Cut(value[start:right], "=")
	if !found {
		return contextValidation{}
	}
	q, err := url.ParseQuery(u.RawQuery)
	return contextValidation{accepted: err == nil && len(q[name]) == 1 && q[name][0] == secret}
}

func validAzureLegacyClientSecret1(s string) bool {
	if len(s) >= 37 && len(s) <= 40 && s[3] >= '0' && s[3] <= '9' && s[4:6] == "Q~" {
		return false
	}
	return validAuditedCarrierLiteral1(s)
}

func validConfluentLegacySecret1(s string) bool {
	return !validConfluentSecret(s)
}

func validAuditedBase64Secret1(s string) bool {
	if len(s) > 256 || strings.ContainsAny(s, "\r\n") {
		return false
	}
	var decoded [192]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s))
	return err == nil && n > 0
}

func validAzureAccountKey1(s string) bool {
	// Microsoft publishes this emulator credential; it is not confidential.
	if s == "Eby8vdM02xNoGF+D2vYNm3kA1IQht/t39nyxJbMso1J4cC6JKMsHQqmUzRGWj7JpMFFEL4ZA+0EDK8L13cvRNg==" || len(s) != 88 {
		return false
	}
	var decoded [66]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s))
	return err == nil && n == 64
}

func validAtlassianDataCenterToken1(s string) bool {
	if len(s) != 44 {
		return false
	}
	var decoded [33]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s))
	if err != nil || n != 33 || decoded[12] != ':' {
		return false
	}
	for _, c := range decoded[:12] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validCoinbaseCDPPrivateKey1(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var key struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PrivateKey string `json:"privateKey"`
	}
	if json.Unmarshal([]byte(s), &key) != nil {
		return false
	}
	if validCoinbaseKeyID1(key.ID) && len(key.PrivateKey) == 88 {
		var decoded [66]byte
		n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(key.PrivateKey))
		return err == nil && n == 64
	}
	name, ok := strings.CutPrefix(key.Name, "organizations/")
	organization, id, found := strings.Cut(name, "/apiKeys/")
	return ok && found && validCoinbaseKeyID1(organization) && validCoinbaseKeyID1(id) && validPrivateKey(key.PrivateKey)
}

func validCoinbaseKeyID1(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := range len(s) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", rune(s[i])) {
			return false
		}
	}
	return true
}

func validateCloudinaryCredentialURI1(value string, start, end int, secret string) contextValidation {
	if !strings.Contains(secret, "://") {
		return contextValidation{accepted: validAuditedCarrierLiteral1(secret)}
	}
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "cloudinary" || u.User == nil || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return contextValidation{}
	}
	password, set := u.User.Password()
	id := u.User.Username()
	if !set || !validAuditedCarrierLiteral1(password) || len(id) != 15 || len(u.Host) < 3 || len(u.Host) > 50 {
		return contextValidation{}
	}
	for i := range len(id) {
		if id[i] < '0' || id[i] > '9' {
			return contextValidation{}
		}
	}
	for i := range len(u.Host) {
		c := u.Host[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (i == 0 || (c < '0' || c > '9') && c != '-') {
			return contextValidation{}
		}
	}
	return contextValidation{accepted: true}
}

func validateCouchbaseCredentialURI1(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "couchbase" && u.Scheme != "couchbases") || u.User == nil || u.User.Username() == "" || u.Hostname() == "" || !connectionURIHost(u.Host) {
		return contextValidation{}
	}
	password, set := u.User.Password()
	return contextValidation{accepted: set && validAuditedCarrierLiteral1(password)}
}

func validateAzureStorageSASURL1(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != httpsScheme || u.User != nil || u.Fragment != "" || !strings.HasSuffix(u.Host, ".blob.core.windows.net") || strings.Count(u.RawQuery, "&") > 32 {
		return contextValidation{}
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return contextValidation{}
	}
	for _, values := range q {
		if len(values) != 1 {
			return contextValidation{}
		}
	}
	if _, err := time.Parse("2006-01-02", q.Get("sv")); err != nil {
		return contextValidation{}
	}
	switch q.Get("sr") {
	case "b", "c", "d", "bs", "bv":
	default:
		return contextValidation{}
	}
	permissions := q.Get("sp")
	if permissions == "" || len(permissions) > 16 {
		return contextValidation{}
	}
	for _, c := range permissions {
		if !strings.ContainsRune("racwdxyltfmeopi", c) {
			return contextValidation{}
		}
	}
	if q.Get("st") == "" && q.Get("se") == "" {
		return contextValidation{}
	}
	for _, name := range []string{"st", "se"} {
		if stamp := q.Get(name); stamp != "" {
			if _, err := time.Parse(time.RFC3339, stamp); err != nil {
				return contextValidation{}
			}
		}
	}
	signature := q.Get("sig")
	if len(signature) != 44 {
		return contextValidation{}
	}
	var decoded [33]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(signature))
	return contextValidation{accepted: err == nil && n == 32}
}

func validateBlynkDeviceURL1(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != httpsScheme || u.User != nil || u.Fragment != "" || !strings.HasPrefix(u.Path, "/external/api/") || strings.Count(u.RawQuery, "&") > 32 {
		return contextValidation{}
	}
	if u.Host != blynkHost {
		region, found := strings.CutSuffix(u.Host, ".blynk.cloud")
		if !found || region == "" || strings.ContainsAny(region, ".:") {
			return contextValidation{}
		}
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q[tokenField]) != 1 {
		return contextValidation{}
	}
	token := q.Get(tokenField)
	if len(token) != 32 {
		return contextValidation{}
	}
	for i := range len(token) {
		c := token[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return contextValidation{}
		}
	}
	return contextValidation{accepted: true}
}

func validCurlUserCredential1(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	word, rest, ok := nextCurlLiteralWord1(s)
	if !ok || word != curlCommand {
		return false
	}
	for count := 0; count < 64; count++ {
		word, rest, ok = nextCurlLiteralWord1(rest)
		if !ok || word == "" || word == "--" {
			return false
		}
		var credential string
		switch {
		case word == "-u" || word == curlUserOption:
			credential, _, ok = nextCurlLiteralWord1(rest)
			if !ok {
				return false
			}
		case strings.HasPrefix(word, "--user="):
			credential = word[len("--user="):]
		case strings.HasPrefix(word, "-u") && len(word) > 2:
			credential = word[2:]
		default:
			switch word {
			case "-H", curlHeaderOption, "-d", curlDataOption, curlDataRawOption, curlDataBinaryOption, "-o", "--output", curlURLOption, "-x", "--proxy", "-X", curlRequestOption, "-A", "--user-agent", "-e", "--referer", "--connect-timeout", "--max-time", "-F", curlFormOption:
				_, rest, ok = nextCurlLiteralWord1(rest)
				if !ok {
					return false
				}
			case "-s", "-S", curlSilentShowErrorOption, "-L", "-f", "-i", "-I", "-k", curlSilentOption, curlShowErrorOption, curlLocationOption, curlFailOption, curlInsecureOption:
			default:
				if strings.HasPrefix(word, "-") {
					return false
				}
			}
			continue
		}
		user, password, present := strings.Cut(credential, ":")
		return present && (user == "" || validAuditedCarrierLiteral1(user)) && validAuditedCarrierLiteral1(password)
	}
	return false
}

// A small shell-word reader for curl options, not a shell interpreter. Dollar
// expansion, command substitution, command separators and unclosed quotes fail.
func nextCurlLiteralWord1(s string) (word, rest string, ok bool) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return "", "", false
	}
	// Ordinary words and single quoted spans need no materialization. Find the
	// first shell syntax byte; only escapes or concatenation need the reader
	// below. Byte-oriented searches preserve invalid UTF-8 verbatim.
	start, end := 0, 0
	switch s[0] {
	case '\'':
		start = 1
		end = strings.IndexByte(s[start:], '\'')
	case '"':
		start = 1
		end = strings.IndexAny(s[start:], "\"\\$`")
	default:
		end = strings.IndexAny(s, " \t'\"\\$`;|&<>\r\n")
	}
	if end < 0 {
		return s[start:], "", start == 0
	}
	end += start
	if start == 0 && (s[end] == ' ' || s[end] == '\t') {
		return s[:end], s[end:], end != 0
	}
	if start != 0 && s[end] == s[0] && (end+1 == len(s) || s[end+1] == ' ' || s[end+1] == '\t') {
		return s[start:end], s[end+1:], end != start
	}

	var b strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote == 0 && (c == ' ' || c == '\t') {
			return b.String(), s[i:], b.Len() != 0
		}
		if quote == 0 && strings.ContainsRune(";|&<>\r\n", rune(c)) {
			return "", "", false
		}
		if c == '\'' || c == '"' {
			if quote == 0 {
				quote = c
				continue
			}
			if quote == c {
				quote = 0
				continue
			}
		}
		if quote != '\'' && (c == '$' || c == 96) {
			return "", "", false
		}
		if c == '\\' && quote != '\'' {
			i++
			if i == len(s) {
				return "", "", false
			}
			c = s[i]
		}
		b.WriteByte(c)
	}
	return b.String(), "", quote == 0 && b.Len() != 0
}
