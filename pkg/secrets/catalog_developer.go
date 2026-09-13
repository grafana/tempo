package secrets

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"strconv"
	"strings"
)

const gitHubFormatSource = "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/about-authentication-to-github"

// developerRuleSpecs describes credential candidates, not proof of issuance,
// active authorization, or complete provider-format conformance.
// Portions adapt pinned Gitleaks v8.30.1; see LICENSE.gitleaks. Shared opaque
// formats use its matching constraints and entropy policy unless the per-family
// compatibility ledger records a provider-source-backed subset or extension.
// Example/value allowlists and inline gitleaks:allow directives are not imported.
//
// GitLab model-specific definitions below use its shared token generator:
// https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/lib/authn/token_field/base.rb
// and Devise's documented 20-character URL-safe alphabet with lIO0 translated:
// https://raw.githubusercontent.com/heartcombo/devise/main/lib/devise.rb
// Default token-prefix semantics: https://docs.gitlab.com/security/tokens/
// Sonar token-type letters are defined by:
// https://raw.githubusercontent.com/SonarSource/sonarqube/master/server/sonar-db-dao/src/main/java/org/sonar/db/user/TokenType.java
var developerRuleSpecs = []catalogRuleSpec{
	{
		ID:          "anthropic-api-key",
		Regex:       "\\b(sk-ant-api03-[a-zA-Z0-9_\\-]{93}AA)(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"sk-ant-api03"},
		Entropy:     0,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Claude API api03 candidate using the pinned Gitleaks v8.30.1 body, terminal marker, capture and delimiter constraints. Provider documentation identifies the prefix but does not establish the complete issuer grammar; other revisions are not implied.",
	},
	{
		ID:          "anthropic-admin-api-key",
		Regex:       "\\b(sk-ant-admin01-[a-zA-Z0-9_\\-]{93}AA)(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"sk-ant-admin01"},
		Entropy:     0,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Claude Console administrative candidate using the pinned Gitleaks v8.30.1 admin01 body, terminal marker, capture and delimiter constraints. The documented Enterprise api01 prefix has no established body contract and is outside this baseline; that exclusion does not establish invalidity.",
	},
	{
		ID:          "apify-api-token",
		Regex:       `\b(apify_api_(?:[A-Za-z0-9]{34,38}|[A-Za-z0-9-]{36}))` + catalogRightBoundary,
		Keywords:    []string{"apify_api_"},
		SecretGroup: 1,
		Source:      "https://docs.apify.com/integrations/api",
		Description: "Apify confidential API-token candidate, distinct from public Actor/run/storage identifiers. Retains the TruffleHog v3.97.4 36-character hyphen-capable body and adds the Titus v1.2.9 34-38 alphanumeric subset. These are bounded scanner constraints, not an exhaustive issuer grammar; no entropy or online validity requirement.",
		// Body constraints adapted from pinned TruffleHog apify and Titus apify.yml rules.
	},
	{
		ID:          "artifactory-api-key",
		Regex:       `\b(AKCp[A-Za-z0-9]{69})` + catalogRightBoundary,
		Keywords:    []string{"AKCp"},
		SecretGroup: 1,
		Source:      "https://docs.jfrog.com/user-management/docs/identity-tokens",
		Description: "Historical Artifactory API key: JFrog documents the AKCp prefix and 69 alphanumeric body characters. API keys are deprecated, but remain confidential in historical telemetry.",
	},
	{
		ID:          "artifactory-reference-token",
		Regex:       `\b(cmVmd[A-Za-z0-9]{59})` + catalogRightBoundary,
		Keywords:    []string{"cmVmd"},
		SecretGroup: 1,
		Source:      "https://docs.jfrog.com/user-management/docs/identity-tokens",
		Description: "JFrog documented 64-character reference token, not the underlying JWT or public token identifier. Other reference-token revisions are not implied.",
	},
	{
		ID:          "buildkite-api-token",
		Regex:       `\b(bkua_(?:[a-z0-9]{40}|[a-z0-9]{53}))` + catalogRightBoundary,
		Keywords:    []string{"bkua_"},
		SecretGroup: 1,
		Source:      "https://buildkite.com/docs/platform/security/tokens",
		Description: "Buildkite user API access-token candidates with the documented bkua_ prefix. Preserve the historical 40-character scanner subset and add the current 53-character documented masked-example width. Agent, registry, portal and unprefixed legacy tokens remain separate families; no digit-count, checksum or entropy heuristic.",
	},
	{
		ID:          "circleci-personal-access-token",
		Regex:       `\b(CCIPAT_[1-9A-HJ-NP-Za-km-z]{22}_[A-Fa-f0-9]{40})` + catalogRightBoundary,
		Keywords:    []string{"CCIPAT_"},
		SecretGroup: 1,
		Source:      "https://circleci.com/changelog/new-format-for-api-access-tokens/",
		Description: "CircleCI personal access-token candidate using the provider's CCIPAT_<base58-UUID>_<40-char-hex> format. The 22-character UUID segment follows the provider example and TruffleHog v3.97.4 scanner width, not a claim about every UUID serialization. No sample-position letter/digit restrictions, UUID version, checksum or issuer validation. CCIPRJ_ project tokens and unprefixed legacy values are excluded.",
	},
	{
		ID:          "clojars-api-token",
		Regex:       `\b(CLOJARS_[0-9a-f]{60})` + catalogRightBoundary,
		Keywords:    []string{"CLOJARS_"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/clojars/clojars-web/main/src/clojars/db.clj",
		Description: "Clojars deployment secret generated from 30 random bytes encoded as 60 lowercase hexadecimal characters.",
	},
	{
		ID:          "dockerhub-personal-access-token",
		Regex:       `\b(dckr_pat_[A-Za-z0-9_-]{27})` + catalogRightBoundary,
		Keywords:    []string{"dckr_pat_"},
		SecretGroup: 1,
		Source:      "https://docs.docker.com/security/access-tokens/personal-access-tokens/",
		Description: "Docker Hub personal access-token candidate. Docker documents these as password-equivalent secrets and demonstrates dckr_pat_ in its API schema; the 27-character URL-safe body is a pinned Titus v1.2.9 scanner constraint, not a public issuer specification. No username is needed for offline detection; organization tokens and unprefixed historical values are separate.",
		// Body constraint adapted from https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/dockerhub.yml.
	},
	{
		ID:          "dockerhub-organization-access-token",
		Regex:       `\b(dckr_oat_[A-Za-z0-9_-]{32})` + catalogRightBoundary,
		Keywords:    []string{"dckr_oat_"},
		SecretGroup: 1,
		Source:      "https://docs.docker.com/security/access-tokens/organization-access-tokens/",
		Description: "Docker Hub organization access-token candidate, a password-equivalent automation secret rather than a public organization name. Docker's API schema demonstrates dckr_oat_; the 32-character URL-safe body follows TruffleHog v3.97.4, not an issuer width guarantee. No organization name, username or network verification is required for offline detection.",
		// Body constraint adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/dockerhub/v2/dockerhub.go.
	},
	{
		ID:          "figma-personal-access-token",
		Regex:       `\b(figp_[A-Za-z0-9_=-]{40,54})` + catalogRightBoundary,
		Keywords:    []string{"figp_"},
		SecretGroup: 1,
		Source:      "https://developers.figma.com/docs/rest-api/personal-access-tokens/",
		Description: "Figma figp_ credential candidate using the supported TruffleHog v3.97.4 v3 scanner shape: 40-54 alphanumeric, underscore, equals or hyphen body characters. The provider establishes personal-token confidentiality, not this complete prefix/body contract. Includes terminal equals/hyphen without a word-boundary truncation; no historical figd_/OAuth alternatives, decoding or issuer authentication is inferred.",
		// Candidate shape adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/figmapersonalaccesstoken/v3/figmapersonalaccesstoken.go.
	},
	{
		ID:          githubPATRuleID,
		Regex:       "ghp_[0-9a-zA-Z]{36}",
		Keywords:    []string{"ghp_"},
		Entropy:     3,
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "GitHub classic PAT candidate using the pinned Gitleaks v8.30.1 matching and entropy constraints. The provider documents the opaque checksum design but not enough details for independent checksum rejection. Substring matches and omitted example allowlists are intentional compatibility tradeoffs, not issuer verification.",
	},
	{
		ID:          "github-oauth",
		Regex:       "gho_[0-9a-zA-Z]{36}",
		Keywords:    []string{"gho_"},
		Entropy:     3,
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "GitHub OAuth access candidate using the pinned Gitleaks v8.30.1 matching and entropy constraints. The provider documents the opaque checksum design but not enough details for independent checksum rejection. The upstream substring behavior does not authenticate an entire input value.",
	},
	{
		ID:          "github-app-token",
		Regex:       `\b((?:gh[us]_[A-Za-z0-9]{36}|ghs_[0-9]+_[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+))` + catalogRightBoundary,
		Keywords:    []string{"ghu_", "ghs_"},
		Entropy:     3,
		SecretGroup: 1,
		Source:      gitHubFormatSource,
		Description: "GitHub App user/legacy installation candidates retain the provider-documented opaque shape and native token delimiters with the pinned Gitleaks v8.30.1 entropy threshold. The documented stateless installation extension uses a decimal app identifier and three opaque dot-separated URL-safe segments, bounded to 16 KiB. Provider guidance requires opaque handling, so JWT claims, algorithms and signatures are not decoded or validated; detection is not issuer authentication.",
		Validate:    boundedGitHubAppToken,
	},
	{
		ID:          "github-fine-grained-pat",
		Regex:       "github_pat_\\w{82}",
		Keywords:    []string{"github_pat_"},
		Entropy:     3,
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "GitHub fine-grained PAT candidate using the pinned Gitleaks v8.30.1 fixed-width ASCII body, substring matching and entropy constraints. Provider documentation establishes the prefix, not an exhaustive issuer grammar.",
	},
	{
		ID:          "github-refresh-token",
		Regex:       "ghr_[0-9a-zA-Z]{36}",
		Keywords:    []string{"ghr_"},
		Entropy:     3,
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "GitHub App refresh candidate using the pinned Gitleaks v8.30.1 fixed-width substring and entropy constraints. A match inside a longer opaque value is a detection-contract result, not proof of that entire value or its checksum.",
	},
	{
		ID:          "gitlab-cicd-job-token",
		Regex:       `\b(glcbt-[1-9a-f][0-9a-f]{0,15}_[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glcbt-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/lib/ci/builds/token_prefix.rb",
		Description: "GitLab opaque CI job token: glcbt-, positive hexadecimal partition, underscore, then a 20-character Devise friendly token. The partition is defensively limited to 16 hex digits. JWT job tokens are outside this rule.",
	},
	{
		ID:          "gitlab-deploy-token",
		Regex:       `\b(gldt-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"gldt-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/app/models/deploy_token.rb",
		Description: "GitLab repository/package deployment credential with the default provider prefix and 20-character Devise friendly-token body. Custom instance prefixes and unrelated older unprefixed formats are excluded.",
	},
	{
		ID:          "gitlab-feature-flag-client-token",
		Regex:       `\b(glffct-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glffct-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/app/models/operations/feature_flags_client.rb",
		Description: "GitLab feature-flag retrieval credential with the default provider prefix and 20-character Devise friendly-token body. Custom instance prefixes and unrelated older unprefixed formats are excluded.",
	},
	{
		ID:          "gitlab-feed-token",
		Regex:       `\b(glft-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glft-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/app/models/user.rb",
		Description: "GitLab personalized private feed credential with the default provider prefix and 20-character Devise friendly-token body. Custom instance prefixes and unrelated older unprefixed formats are excluded.",
	},
	{
		ID:          "gitlab-ptt",
		Regex:       `\b(glptt-(?:[A-HJ-NP-Za-km-z1-9_-]{20}|[0-9a-f]{40}))` + catalogRightBoundary,
		Keywords:    []string{"glptt-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v16.11.0/app/models/ci/trigger.rb",
		Description: "GitLab pipeline trigger credential using either the current 20-character Devise friendly-token body or the source-backed v16.11 SecureRandom.hex(20) body of 40 lowercase hex characters. Default glptt- prefix only; no online validity inference.",
	},
	{
		ID:          "gitlab-incoming-mail-token",
		Regex:       `\b(?:(glimt-[0-9a-z]{20,25})|incoming\+[A-Za-z0-9._-]{1,160}-[1-9][0-9]{0,19}-((?:glimt-)?[0-9a-z]{20,25})-(?:issue|merge-request)@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?){1,8})` + catalogRightBoundary,
		Keywords:    []string{"glimt-", "incoming+"},
		SecretGroup: 0,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v13.12.0/app/models/user.rb",
		Description: "GitLab incoming-mail secrets retain standalone glimt- values and add complete incoming+project-id-secret-issue/merge-request email carriers, including historical unprefixed base36 secrets. v13.12 generates SecureRandom.hex rendered in base36; public project IDs, secretless addresses and arbitrary unprefixed strings do not qualify. The existing 20-25-character secret bounds remain.",
	},
	{
		ID:          "gitlab-kubernetes-agent-token",
		Regex:       `\b(glagent-(?:[A-HJ-NP-Za-km-z1-9_-]{20}|[A-HJ-NP-Za-km-z1-9_-]{49}[AQgw]|[A-Za-z0-9_-]{27,235}\.01\.[0-9a-z]{9}))` + catalogRightBoundary,
		Keywords:    []string{"glagent-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v16.11.0/app/models/clusters/agent_token.rb",
		Description: "GitLab Kubernetes agent credential: preserves current friendly-20 and CRC-validated routable-v1 forms, and adds the historical Devise.friendly_token(50) generator. Its 37 random bytes encode to 50 URL-safe characters with final character A/Q/g/w, after lIO0 translation; no loose length-only fallback.",
		Validate:    validGitLabAgentToken,
	},
	{
		ID:          "gitlab-oauth-app-secret",
		Regex:       `\b(gloas-[0-9a-f]{64})` + catalogRightBoundary,
		Keywords:    []string{"gloas-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/lib/gitlab/doorkeeper_secret_storing/token/unique_application_token.rb",
		Description: "GitLab OAuth confidential application secret generated from 32 random bytes encoded as lowercase hex. Application IDs and unprefixed OAuth values are excluded.",
	},
	{
		ID:          "gitlab-pat",
		Regex:       `\b(glpat-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glpat-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/lib/authn/token_field/base.rb",
		Description: "GitLab legacy non-routable access secret using the default glpat- prefix and Devise friendly-token body. The right delimiter rejects a truncated prefix of a routable token. Custom PAT prefixes are excluded.",
	},
	{
		ID:          "gitlab-pat-routable",
		Regex:       `\b(glpat-[A-Za-z0-9_-]{27,235}(?:\.01)?\.[0-9a-z]{9})` + catalogRightBoundary,
		Keywords:    []string{"glpat-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v17.11.0/lib/authn/token_field/generator/routable_token.rb",
		Description: "GitLab default-prefix routable access tokens, both source-backed one-dot legacy framing and current v1. Each version checks its own byte order, routing structure, base64 canonicality, encoded length and CRC32. Malformed or unknown versioned forms never bypass the existing v1 validator.",
		Validate:    validBetterleaksGitLabRoutable1,
	},
	{
		ID:          "gitlab-runner-authentication-token",
		Regex:       `\b(glrt(?:r)?-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glrt-", "glrtr-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/master/app/models/ci/runner.rb",
		Description: "GitLab non-routable runner authentication credential using glrt- or glrtr-. glrtr- identifies a runner created through registration, not the registration secret itself. Default Devise friendly-token body only.",
	},
	{
		ID:          "gitlab-runner-authentication-token-routable",
		Regex:       `\b((?:glrt(?:r)?-[A-Za-z0-9_-]{27,235}(?:\.01)?\.[0-9a-z]{9}|glrt-t[23]_[A-Za-z0-9_-]{27,235}\.[0-9a-z]{9}))` + catalogRightBoundary,
		Keywords:    []string{"glrt-", "glrtr-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v17.11.0/lib/authn/token_field/generator/routable_token.rb",
		Description: "GitLab routable runner authentication tokens retain current glrt-/glrtr- v1 validation and add source-backed one-dot framing. v17.10 group/project runners used glrt-t2_/glrt-t3_; v17.11 uses glrt-/glrtr-. Legacy and v1 frames both require structural, length and CRC32 checks; no t1 or unknown-version fallback.",
		Validate:    validBetterleaksGitLabRoutable1,
	},
	{
		ID:          "gitlab-scim-token",
		Regex:       `\b(glsoat-[A-HJ-NP-Za-km-z1-9_-]{20})` + catalogRightBoundary,
		Keywords:    []string{"glsoat-"},
		SecretGroup: 1,
		Source:      "https://gitlab.com/gitlab-org/gitlab/-/raw/master/ee/app/models/scim_oauth_access_token.rb",
		Description: "GitLab SCIM provisioning credential: the model uses the default glsoat- prefix and shared 20-character Devise friendly-token generator. The URL-safe alphabet translates lIO0 to sxyz. Custom instance prefixes are outside this rule; issuer validity is not checked.",
	},
	{
		ID:          "gitlab-session-cookie",
		Regex:       `\b_gitlab_session=([A-Za-z0-9_-]{32,512})` + catalogRightBoundary,
		Keywords:    []string{"_gitlab_session="},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v16.11.0/config/initializers/session_store.rb",
		Description: "GitLab v16.11.0 unsigned Redis session credential in the production non-Geo cookie carrier. Rails Compatibility generates a 32-character lowercase hexadecimal ID; preserves the optional configured URL-safe token prefix within a defensive 512-character total bound. Custom cookie names, signed/encrypted stores and other serializers are outside this source-defined subset; no session lookup occurs.",
		Validate:    validGitLabSessionCookie,
	},
	{
		ID:          "groq-api-key",
		Regex:       `\b(gsk_[A-Za-z0-9]{50,54})` + catalogRightBoundary,
		Keywords:    []string{"gsk_"},
		SecretGroup: 1,
		Source:      "https://console.groq.com/docs/production-readiness/security-onboarding",
		Description: "Groq confidential API-key candidate with the provider-example gsk_ prefix. The 50-54 alphanumeric body is a conservative Titus v1.2.9 scanner constraint, not a published issuer grammar. Case-sensitive prefix and native token boundaries; no entropy, digit-count or online validity requirement.",
		// Body constraint adapted from https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/groq.yml.
	},
	{
		ID:          "huggingface-access-token",
		Regex:       "\\b(hf_(?i:[a-z]{34}))(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"hf_"},
		Entropy:     2,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Hugging Face user access candidate using the pinned Gitleaks v8.30.1 alphabetic body, capture, delimiters and entropy threshold. The provider guide establishes prefix ownership, not complete historical or future token grammar.",
	},
	{
		ID:          "langsmith-personal-access-token",
		Regex:       `\b(lsv2_pt_[a-f0-9]{32}_[a-f0-9]{10})` + catalogRightBoundary,
		Keywords:    []string{"lsv2_pt_"},
		SecretGroup: 1,
		Source:      "https://docs.langchain.com/langsmith/administration-overview",
		Description: "LangSmith personal API credential with the documented lsv2_pt_ prefix. The 32-hex/underscore/10-hex lowercase body is a TruffleHog v3.97.4 scanner constraint, not a provider-published generator or checksum guarantee. Service keys, legacy ls__ keys and public workspace IDs are distinct; no entropy or online validity filter.",
		// Body constraint adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/langsmith/langsmith.go.
	},
	{
		ID:          "langsmith-service-key",
		Regex:       `\b(lsv2_sk_[a-f0-9]{32}_[a-f0-9]{10})` + catalogRightBoundary,
		Keywords:    []string{"lsv2_sk_"},
		SecretGroup: 1,
		Source:      "https://docs.langchain.com/langsmith/administration-overview",
		Description: "LangSmith service-account API credential with the documented lsv2_sk_ prefix. The 32-hex/underscore/10-hex lowercase body follows TruffleHog v3.97.4 rather than a public issuer grammar; the suffix is not cryptographically checked. Personal tokens, legacy ls__ keys and public workspace IDs are distinct; no workspace context or verification request is required.",
		// Body constraint adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/langsmith/langsmith.go.
	},
	{
		ID:          "linear-api-key",
		Regex:       "lin_api_(?i)[a-z0-9]{40}",
		Keywords:    []string{"lin_api_"},
		Entropy:     2,
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "Linear personal API-key candidate using the pinned Gitleaks v8.30.1 matching, case handling, fixed-width body and entropy constraints. The provider announcement establishes the prefix, not issuer-validity or exhaustive body grammar.",
	},
	{
		ID:          "linear-oauth-token",
		Regex:       `\b(lin_oauth_[A-Za-z0-9]{16,256})` + catalogRightBoundary,
		Keywords:    []string{"lin_oauth_"},
		SecretGroup: 1,
		Source:      "https://linear.app/changelog/2021-08-19-github-secret-scanning",
		Description: "Historical Linear OAuth access-token candidate with the provider-announced lin_oauth_ prefix. Retains the existing bounded 16-256 alphanumeric subset because no issuer body implementation is public. It is not mapped to the differently typed upstream Linear client-secret rule; opaque newer revisions and arbitrary client secrets are not attributed.",
	},
	{
		ID:          "notion-api-token",
		Regex:       "\\b(ntn_[0-9]{11}[A-Za-z0-9]{32}[A-Za-z0-9]{3})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"ntn_"},
		Entropy:     4,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Notion current ntn_ candidate using the pinned Gitleaks v8.30.1 numeric-leading body, capture, delimiters and entropy constraints. Notion requires treating tokens as opaque; the matching contract is not issuer validation and does not cover the historical secret_ representation.",
	},
	{
		ID:          "notion-legacy-api-token",
		Regex:       `\b(?i:NOTION_(?:API_KEY|TOKEN))["']?\s*[:=]\s*["']?(secret_[A-Za-z0-9]{16,256})` + catalogRightBoundary,
		Keywords:    []string{"NOTION_API_KEY", "NOTION_TOKEN"},
		SecretGroup: 1,
		Source:      "https://developers.notion.com/page/changelog",
		Description: "Historical Notion secret_ integration candidate only in an explicit Notion-specific API_KEY or TOKEN assignment in the scanned value. The documented prefix transition establishes this credential type but not its body grammar; preserves the existing 16-256 alphanumeric subset as a bounded ownership heuristic. No current-token width is transferred to the legacy representation.",
	},
	{
		ID:          "npm-access-token",
		Regex:       "(?i)\\b(npm_[a-z0-9]{36})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"npm_"},
		Entropy:     2,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "npm prefixed access-token candidate using the pinned Gitleaks v8.30.1 matching, case handling, capture, delimiters and entropy constraints. Provider checksum documentation lacks the details needed for a rejecting checksum validator; no issuance claim is made.",
	},
	{
		ID:          "openai-api-key",
		Regex:       "\\b(sk-(?:proj|svcacct|admin)-(?:[A-Za-z0-9_-]{74}|[A-Za-z0-9_-]{58})T3BlbkFJ(?:[A-Za-z0-9_-]{74}|[A-Za-z0-9_-]{58})\\b|sk-[a-zA-Z0-9]{20}T3BlbkFJ[a-zA-Z0-9]{20})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"t3blbkfj"},
		Entropy:     3,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "OpenAI API-key candidate using the full pinned Gitleaks v8.30.1 marker-bearing historical, project, service-account and administrative alternatives, captures, delimiters and entropy threshold. Explicit assignment context is no longer required. The inherited marker and widths are scanner constraints, not a provider grammar guarantee; arbitrary opaque assignments are outside this baseline.",
	},
	{
		ID:          "openai-admin-api-key",
		Regex:       "\\b(sk-admin-(?:[A-Za-z0-9_-]{74}|[A-Za-z0-9_-]{58})T3BlbkFJ(?:[A-Za-z0-9_-]{74}|[A-Za-z0-9_-]{58})\\b)(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"sk-admin-"},
		Entropy:     3,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "OpenAI organization administrative candidate specializing the documented administrative branch of pinned Gitleaks v8.30.1 openai-api-key. Inherits its full marker-bearing body, capture, delimiters and entropy threshold rather than recognizing arbitrary sk-admin- strings. The provider create response establishes the credential type and prefix, not these scanner body assumptions.",
	},
	{
		ID:          "openrouter-api-key",
		Regex:       `\b(sk-or-v1-[0-9a-f]{64})` + catalogRightBoundary,
		Keywords:    []string{"sk-or-v1-"},
		SecretGroup: 1,
		Source:      "https://openrouter.ai/docs/guides/features/guardrails/secret-formats",
		Description: "OpenRouter confidential API-key candidate: sk-or-v1- and 64 lowercase hexadecimal characters, documented by its Secrets guardrail and illustrated by its key-creation response. This is a supported detection shape, not a complete issuance grammar or active-key check. Public key hashes and masked labels are excluded; no entropy or digit-count heuristic.",
	},
	{
		ID:          "perplexity-api-key",
		Regex:       "\\b(pplx-[a-zA-Z0-9]{48})(?:[\\x60'\"\\s;]|\\\\[nr]|$|\\b)",
		Keywords:    []string{"pplx-"},
		Entropy:     4,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Perplexity API-key candidate using the pinned Gitleaks v8.30.1 fixed-width body, capture, delimiter and entropy constraints. Provider examples establish the prefix only; word-boundary adjacency acceptance is scanner compatibility rather than production accuracy evidence.",
	},
	{
		ID:          "postman-api-token",
		Regex:       "\\b(PMAK-(?i)[a-f0-9]{24}\\-[a-f0-9]{34})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"pmak-"},
		Entropy:     3,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Postman API-key candidate using the pinned Gitleaks v8.30.1 hexadecimal segments, case handling, capture, delimiters and entropy constraints. The official screenshot establishes the prefix but not the body widths; collection access keys remain separate.",
	},
	{
		ID:          "prefect-api-token",
		Regex:       "\\b(pnu_[a-zA-Z0-9]{36})(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"pnu_"},
		Entropy:     2,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Prefect Cloud pnu_ candidate using the pinned Gitleaks v8.30.1 fixed-width body, capture, delimiters and entropy threshold. The CLI also recognizes pnb_, but does not establish that variant body; it is not inferred by substituting a prefix in this baseline.",
	},
	{
		ID:          "pypi-upload-token",
		Regex:       `\b(pypi-[A-Za-z0-9_-]+={0,2})` + catalogRightBoundary,
		Keywords:    []string{"pypi-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/pypi/warehouse/main/warehouse/macaroons/services.py",
		Description: "PyPI upload secret serialized as a URL-safe base64 macaroon v2. Checks full binary framing, UUID identifier, first-party caveats and 32-byte signature per libmacaroons format, not signature authenticity. A 16 KiB decoded limit bounds candidate work; no short fixed token length is assumed.",
		Validate:    validPyPIToken,
	},
	{
		ID:          "replicate-api-token",
		Regex:       `\b(r8_[A-Za-z0-9]{37})` + catalogRightBoundary,
		Keywords:    []string{"r8_"},
		SecretGroup: 1,
		Source:      "https://replicate.com/docs/topics/security/api-tokens",
		Description: "Replicate confidential API-token candidate: the provider explicitly specifies 40 total characters beginning r8_. The 37-character alphanumeric body alphabet follows Titus v1.2.9 and is not a published exhaustive alphabet guarantee. Public model/version identifiers and other prefixes are excluded; no entropy, checksum or active-token check.",
		// Body alphabet adapted from https://github.com/praetorian-inc/titus/blob/v1.2.9/pkg/rule/rules/replicate.yml.
	},
	{
		ID:          "rubygems-api-token",
		Regex:       `\b(rubygems_[0-9a-f]{48})` + catalogRightBoundary,
		Keywords:    []string{"rubygems_"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/rubygems/rubygems.org/master/app/controllers/concerns/api_keyable.rb",
		Description: "RubyGems API publishing secret generated as rubygems_ plus 24 random bytes encoded in lowercase hex. Unprefixed legacy keys are not attributed from generic hex strings.",
	},
	{
		ID:          "sonar-api-token",
		Regex:       `\b(sq[uap]_[0-9a-f]{40})` + catalogRightBoundary,
		Keywords:    []string{"squ_", "sqa_", "sqp_"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/SonarSource/sonarqube/master/server/sonar-webserver-auth/src/main/java/org/sonar/server/usertoken/TokenGeneratorImpl.java",
		Description: "SonarQube user, global-analysis and project-analysis secrets: sq plus token-type letter and 20 random bytes in lowercase hex. Project-badge tokens (sqb_) and legacy unprefixed hashes are outside this rule.",
	},
	{
		ID:          "weights-and-biases-api-token",
		Regex:       `\b(wandb_v1_[A-Za-z0-9]{27}_[A-Za-z0-9]{49})` + catalogRightBoundary,
		Keywords:    []string{"wandb_v1_"},
		SecretGroup: 1,
		Source:      "https://docs.wandb.ai/platform/hosting/iam/api-keys",
		Description: "Weights & Biases complete prefixed API-secret candidate: wandb_v1_, 27 alphanumeric characters, underscore, then 49 alphanumeric characters. These component widths follow TruffleHog v3.97.4 v2, not a published provider generator. The required second component excludes public-ID-only prefixes rather than inferring confidentiality from wandb_v1_ alone. Unprefixed legacy keys and other revisions are outside this rule; no entropy or digit requirement.",
		// Candidate shape adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/weightsandbiases/v2/weightsandbiases.go.
	},
	{
		ID:          "xai-api-key",
		Regex:       `\b(xai-[A-Za-z0-9_]{80})` + catalogRightBoundary,
		Keywords:    []string{"xai-"},
		SecretGroup: 1,
		Source:      "https://docs.x.ai/developers/rest-api-reference/management/auth",
		Description: "xAI confidential API-key candidate, distinct from public key IDs and redacted display values. Provider creation examples establish xai-; the 80-character alphanumeric/underscore body follows TruffleHog v3.97.4 as a conservative scanner constraint, not a complete issuer grammar. Case-sensitive prefix and native token boundaries; no online validity check.",
		// Body constraint adapted from https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/xai/xai.go.
	},
	{
		ID:          "huggingface-organization-api-token",
		Regex:       "\\b(api_org_(?i:[a-z]{34}))(?:[\\x60'\"\\s;]|\\\\[nr]|$)",
		Keywords:    []string{"api_org_"},
		Entropy:     2,
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Historical Hugging Face organization bearer candidate using the pinned Gitleaks v8.30.1 alphabetic body, capture, delimiters and entropy threshold. The official historical SDK distinguishes this prefix from personal tokens but does not specify the body generator.",
	},
	{
		ID:          "gitlab-rrt",
		Regex:       `\b(GR1348941[A-Za-z0-9_-]{16,256})` + catalogRightBoundary,
		Keywords:    []string{"GR1348941"},
		Entropy:     3,
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/gitlabhq/gitlabhq/v16.11.0/app/models/concerns/runners_token_prefixable.rb",
		Description: "GitLab prefixed runner registration secret. The v16.11.0 project generator uses Devise's 20-character friendly body, but manually assigned prefixed tokens are stored and accepted without that grammar. Preserves these opaque candidates within a defensive 16-256 URL-safe body bound; generator violations alone do not establish invalidity. Unprefixed instance tokens are excluded. Registers a runner rather than authenticating one; no issuance check.",
	},
}

// boundedGitHubAppToken applies only a defensive work bound. The candidate
// expression checks the documented carrier; GitHub requires opaque handling,
// so this path deliberately does not decode or introspect the embedded JWT.
func boundedGitHubAppToken(token string) bool {
	return len(token) <= 16*1024
}

// The routing frame is publicly specified by GitLab's RoutableToken generator:
// 16 random bytes, routing text, one routing-length byte, then a version,
// base36 encoded-body length and seven-character base36 CRC32 over the prefix.
func validGitLabRoutableToken(token string) bool {
	_, body, found := strings.Cut(token, "-")
	if !found {
		return false
	}
	encoded, footer, found := strings.Cut(body, ".01.")
	if !found || len(footer) != 9 || len(encoded) > 235 {
		return false
	}
	size, err := strconv.ParseUint(footer[:2], 36, 16)
	if err != nil || int(size) != len(encoded) {
		return false
	}
	checksum, err := strconv.ParseUint(footer[2:], 36, 32)
	if err != nil || uint32(checksum) != crc32.ChecksumIEEE([]byte(token[:len(token)-7])) {
		return false
	}
	var decoded [176]byte
	n, err := base64.RawURLEncoding.Strict().Decode(decoded[:], []byte(encoded))
	if err != nil || n < 20 || int(decoded[n-1]) != n-17 {
		return false
	}
	routing := decoded[16 : n-1]
	var previous byte
	hasOwner := false
	for len(routing) > 0 {
		line, rest, separated := bytes.Cut(routing, []byte{'\n'})
		if len(line) < 3 || line[1] != ':' || line[0] <= previous || (separated && len(rest) == 0) {
			return false
		}
		switch line[0] {
		case 'c', 'o':
			hasOwner = true
		case 'g', 'p', 't', 'u':
		default:
			return false
		}
		previous = line[0]
		routing = rest
	}
	return hasOwner
}

func validGitLabAgentToken(token string) bool {
	return !strings.ContainsRune(token, '.') || validGitLabRoutableToken(token)
}

// GitLab's RedisStore prepends session_cookie_token_prefix to the public ID.
// redis-actionpack 5.4.0 includes Rails 7.0.8.1 Compatibility, whose generator
// calls SecureRandom.hex(16). The candidate rule bounds the URL-safe prefix;
// checking the generated suffix preserves that supported custom configuration.
func validGitLabSessionCookie(value string) bool {
	if len(value) < 32 || len(value) > 512 {
		return false
	}
	for i := len(value) - 32; i < len(value); i++ {
		c := value[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// Warehouse issues v2 macaroons with UUID identifiers and first-party caveats.
// Binary framing comes from https://github.com/rescrv/libmacaroons/blob/master/doc/format.txt.
// PyPI's serializer uses unpadded URL-safe base64; correctly padded copies are
// accepted too. No key lookup or cryptographic signature verification occurs.
func validPyPIToken(token string) bool {
	if !strings.HasPrefix(token, "pypi-") {
		return false
	}
	body := token[len("pypi-"):]
	if len(body) > 21848 { // Base64 upper bound for 16 KiB of binary data.
		return false
	}
	encoding := base64.RawURLEncoding.Strict()
	if strings.HasSuffix(body, "=") {
		encoding = base64.URLEncoding.Strict()
	}
	data, err := encoding.DecodeString(body)
	if err != nil || len(data) < 1 || len(data) > 16*1024 || data[0] != 2 {
		return false
	}
	kind, field, rest, ok := readPyPIField(data[1:])
	if ok && kind == 1 { // Optional location is not an authenticated issuer.
		kind, field, rest, ok = readPyPIField(rest)
	}
	if !ok || kind != 2 || !validPyPIIdentifier(field) {
		return false
	}
	kind, _, rest, ok = readPyPIField(rest)
	if !ok || kind != 0 { // End of header.
		return false
	}
	for {
		kind, field, rest, ok = readPyPIField(rest)
		if !ok {
			return false
		}
		if kind == 0 { // End of caveats.
			break
		}
		if kind != 2 || len(field) == 0 { // First-party caveat identifier.
			return false
		}
		kind, _, rest, ok = readPyPIField(rest)
		if !ok || kind != 0 {
			return false
		}
	}
	kind, field, rest, ok = readPyPIField(rest)
	return ok && kind == 6 && len(field) == 32 && len(rest) == 0
}

func readPyPIField(data []byte) (uint64, []byte, []byte, bool) {
	kind, n := binary.Uvarint(data)
	if n <= 0 || n > 3 {
		return 0, nil, nil, false
	}
	data = data[n:]
	if kind == 0 {
		return kind, nil, data, true
	}
	size, n := binary.Uvarint(data)
	if n <= 0 || n > 3 || size > uint64(len(data)-n) {
		return 0, nil, nil, false
	}
	data = data[n:]
	return kind, data[:int(size)], data[int(size):], true
}

func validPyPIIdentifier(identifier []byte) bool {
	if len(identifier) != 36 {
		return false
	}
	for i, c := range identifier {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
	}
	return true
}
