package secrets

import "strings"

// Candidate constraints adapted from Betterleaks commit 95237cf8eb4d
// (MIT; see LICENSE.betterleaks). Primary sources and independent synthetic
// fixtures distinguish confidential roles from scanner-supported body bounds.
var betterleaksTokensRules1 = []catalogRuleSpec{
	{
		ID:          "aikido-ci-token",
		Regex:       `\b(AIK_CI_[A-Za-z0-9]{20,44})` + catalogRightBoundary,
		Keywords:    []string{"AIK_CI_"},
		SecretGroup: 1,
		Source:      "https://help.aikido.dev/pr-and-release-gating/cli-for-pr-and-release-gating/aikido-ci-api",
		Description: "Aikido CI authentication tokens are displayed once and supplied in X-AIK-API-SECRET. AIK_CI_ with a 20-44 alphanumeric body is the pinned scanner subset; no client IDs are reported.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[7:]) },
	},
	{
		ID:          "aikido-client-secret",
		Regex:       `\b(AIK_SECRET_[A-Za-z0-9]{64})` + catalogRightBoundary,
		Keywords:    []string{"AIK_SECRET_"},
		SecretGroup: 1,
		Source:      "https://apidocs.aikido.dev/reference/authorization",
		Description: "Aikido confidential OAuth client secrets must not be shared with users. The role-bearing AIK_SECRET_ scanner subset is recognized independently of the public AIK_CLIENT_ companion; possession of the client ID is not necessary for a secret leak.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[11:]) },
	},
	{
		ID:          "asaas-api-token",
		Regex:       `(?:^|[^A-Za-z0-9_./+\-])(\$aact_(?:prod|hmlg)_[A-Za-z0-9_-]{20,100})` + catalogRightBoundary,
		Keywords:    []string{"$aact_"},
		SecretGroup: 1,
		Source:      "https://docs.asaas.com/docs/authentication",
		Description: "Asaas explicitly documents $aact_prod_ and $aact_hmlg_ as confidential API key prefixes for production and sandbox. The 20-100 URL-safe body range is the pinned scanner subset.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[11:]) },
	},
	{
		ID:          "brave-search-api-key",
		Regex:       `\b(BSA[A-Za-z0-9_-]{24,40})` + catalogRightBoundary,
		Keywords:    []string{"BSA"},
		SecretGroup: 1,
		Source:      "https://api-dashboard.search.brave.com/documentation/guides/authentication",
		Description: "Brave Search keys are confidential X-Subscription-Token credentials, never client-side identifiers. BSA and the 24-40 URL-safe body range are pinned scanner constraints.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[3:]) },
	},
	{
		ID:          "buildkite-service-token",
		Regex:       `\b(bkaa_[A-Za-z0-9_-]{75}|bkaj_[A-Za-z0-9_-]{333}|bkar_[A-Za-z0-9_-]{73}|bkct_[A-Za-z0-9_-]{73}|bkpt_[A-Za-z0-9_-]{199}|bkpat_[A-Za-z0-9_-]{54}|bkps_[A-Za-z0-9_-]{64})` + catalogRightBoundary,
		Keywords:    []string{"bkaa_", "bkaj_", "bkar_", "bkct_", "bkpt_", "bkpat_", "bkps_"},
		SecretGroup: 1,
		Source:      "https://buildkite.com/docs/platform/security/tokens",
		Description: "Buildkite documents these seven agent, registry, portal-token and portal-secret prefixes and their masked example widths. Public token UUIDs and user access tokens are not this service-token family.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[strings.IndexByte(s, '_')+1:]) },
	},
	{
		ID:          "canadian-digital-service-notify-api-key",
		Regex:       `\b(ApiKey-v1[ \t]{1,4}gcntfy-[A-Za-z0-9_]{1,128}-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})` + catalogRightBoundary,
		Keywords:    []string{"gcntfy-"},
		SecretGroup: 1,
		Source:      "https://documentation.notification.canada.ca/en/start.html",
		Description: "GC Notify documents the complete ApiKey-v1 gcntfy-name-serviceUUID-secretUUID authorization value. Both UUIDs and the confidential final UUID must be present; service IDs and key names alone are not credentials. Name and whitespace limits are scanner bounds.",
	},
	{
		ID:          "canva-client-secret",
		Regex:       `\b(cnvca[A-Za-z0-9_-]{51})` + catalogRightBoundary,
		Keywords:    []string{"cnvca"},
		SecretGroup: 1,
		Source:      "https://www.canva.dev/docs/connect/api-reference/authentication/generate-access-token/",
		Description: "Canva documents cnvca as the confidential client_secret prefix and requires backend-only authentication. A client ID is public and is not necessary to recognize an exposed prefixed secret; the 51-character suffix is scanner-bounded.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[5:]) },
	},
	{
		ID:          "cartesia-api-key",
		Regex:       `\b(sk_car_[A-Za-z0-9_]{20})` + catalogRightBoundary,
		Keywords:    []string{"sk_car_"},
		SecretGroup: 1,
		Source:      "https://docs.cartesia.ai/get-started/authenticate-your-client-applications.md",
		Description: "Cartesia documents sk_car_ API keys for trusted server authentication and forbids sending them to browsers. Short-lived browser access tokens are not inferred; the 20-character body is a scanner subset.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[7:]) },
	},
	{
		ID:          "cockroachlabs-cloud-api-key",
		Regex:       `\b(CCDB1_[A-Za-z0-9]{22}_[A-Za-z0-9]{40})` + catalogRightBoundary,
		Keywords:    []string{"CCDB1_"},
		SecretGroup: 1,
		Source:      "https://www.cockroachlabs.com/docs/cockroachcloud/managing-access",
		Description: "CockroachDB Cloud secret keys contain an API key and secret and must never be public. CCDB1_ and 22/40 segmented alphanumeric widths are the pinned scanner subset, not a checksum validation.",
	},
	{
		ID:          "datastax-astra-application-token",
		Regex:       `\b(AstraCS:[A-Za-z0-9]{20,512})` + catalogRightBoundary,
		Keywords:    []string{"AstraCS:"},
		SecretGroup: 1,
		Source:      "https://docs.datastax.com/en/astra-db-serverless/api-reference/compare-dataapi-to-stargate.html",
		Description: "DataStax documents AstraCS: application tokens as authorization for Data and Stargate APIs. Recognize complete 20-512 alphanumeric scanner candidates; the upper bound is a local resource limit, not issuer grammar.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[8:]) },
	},
	{
		ID:          "devcycle-server-sdk-key",
		Regex:       `\b(dvc_server_[A-Za-z0-9]{8,32})` + catalogRightBoundary,
		Keywords:    []string{"dvc_server_"},
		SecretGroup: 1,
		Source:      "https://docs.devcycle.com/platform/account-management/keys/",
		Description: "DevCycle server SDK keys must remain secret because they expose full project configuration. The dvc_server_ 8-32 alphanumeric scanner subset excludes the deliberately public client and mobile keys.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[11:]) },
	},
	{
		ID:          "devin-api-credential",
		Regex:       `\b(apk_user_[A-Za-z0-9+/]{120,180}={0,2}|apk_[A-Za-z0-9+/]{80,100}={0,2}|cog_[a-z2-7]{52})` + catalogRightBoundary,
		Keywords:    []string{"apk_user_", "apk_", "cog_"},
		SecretGroup: 1,
		Source:      "https://docs.devin.ai/api-reference/authentication",
		Description: "Devin documents legacy apk_user_ personal keys, apk_ organization service keys and current cog_ credentials. cog_ now also authenticates human PATs, so no service-user-only issuer claim is made. Body bounds follow the pinned scanner, not a Base64 issuer contract.",
		Validate:    validBetterleaksDevinToken1,
	},
	{
		ID:          "docker-swarm-join-token",
		Regex:       `\b(SWMTKN-(?:1-|2-1-)[a-z0-9]{50}-[a-z0-9]{25})` + catalogRightBoundary,
		Keywords:    []string{"SWMTKN-1-", "SWMTKN-2-1-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/moby/swarmkit/master/ca/config.go",
		Description: "SwarmKit generates SWMTKN-1- and FIPS SWMTKN-2-1- tokens with a 256-bit CA digest in 50 zero-padded base36 characters and a 128-bit joining secret in 25 base36 characters. Both components are required; digest-only material is public and insufficient. Native bounds reject numeric overflow; parser-only formats not emitted by the generator are outside this rule.",
		Validate:    validBetterleaksSwarmJoinToken1,
	},
	{
		ID:          "docker-swarm-unlock-key",
		Regex:       `\b(SWMKEY-1-[A-Za-z0-9+/]{43})` + catalogRightBoundary,
		Keywords:    []string{"SWMKEY-1-"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/moby/swarmkit/master/manager/encryption/encryption.go",
		Description: "SwarmKit generates 32-byte secret encryption keys and serializes SWMKEY-1- plus unpadded standard Base64. Strict decoding checks canonical trailing bits and decoded length; this is the actual manager unlock key, not encrypted raft data.",
		Validate:    validBetterleaksSwarmUnlockKey1,
	},
	{
		ID:          "elastic-cloud-api-key",
		Regex:       `\b(essu_[A-Za-z0-9_-]{60,200}={0,2})` + catalogRightBoundary,
		Keywords:    []string{"essu_"},
		SecretGroup: 1,
		Source:      "https://www.elastic.co/docs/deploy-manage/api-keys/elastic-cloud-api-keys",
		Description: "Elastic Cloud API keys authorize management of organizational resources and must be stored safely. essu_ with 60-200 URL-safe characters and optional trailing padding is the pinned scanner subset; no JWT or Base64 issuer structure is invented.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[5:]) },
	},
	{
		ID:          "gemini-authorization-api-key",
		Regex:       `\b(AQ\.Ab8RN6[A-Za-z0-9_-]{44})` + catalogRightBoundary,
		Keywords:    []string{"AQ.Ab8RN6"},
		SecretGroup: 1,
		Source:      "https://ai.google.dev/gemini-api/docs/api-key",
		Description: "Google authorization keys authenticate as a bound service account, unlike standard project/billing API keys. Recognizes the distinctive AQ.Ab8RN6 plus 44 URL-safe-character Betterleaks scanner subset directly, including GEMINI_API_KEY and GOOGLE_API_KEY assignments and x-goog-api-key headers. Prefix mapping and width are scanner evidence, not an exhaustive issuer grammar or proof of validity. Standard AIza keys and Gemini Exchange credentials are outside this rule.",
	},
	{
		ID:          "encoded-live-secret-key",
		Regex:       `\b(sk_live_a2V5Xz[A-Za-z0-9+/]{69}=?)` + catalogRightBoundary,
		Keywords:    []string{"sk_live_a2V5Xz"},
		SecretGroup: 1,
		Source:      "https://workos.com/docs/reference/api-authentication",
		Description: "Highnote and WorkOS private API-key candidates share the sk_live_a2V5Xz scanner representation. Recognize this confidential structure without requiring provider attribution or an assignment. Strict Base64 decoding checks a 56-byte key_ payload; the supported representation is a scanner subset, not an exhaustive issuer grammar or proof of liveness.",
		Validate:    validBetterleaksSharedLiveKey1,
	},
	{
		ID:          "lichess-personal-access-token",
		Regex:       `\b(lip_[A-Za-z0-9_]{16,60})` + catalogRightBoundary,
		Keywords:    []string{"lip_"},
		SecretGroup: 1,
		Source:      "https://raw.githubusercontent.com/lichess-org/lila/master/modules/oauth/src/main/AccessTokenApi.scala",
		Description: "Lichess personal access tokens authenticate user-scoped API calls and are stored by hashed bearer identity. lip_ with a 16-60 alphanumeric/underscore body is the pinned scanner subset, not a public token database ID.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[4:]) },
	},
	{
		ID:          "llama-cloud-api-key",
		Regex:       `\b(llx-[A-Za-z0-9]{44,52})` + catalogRightBoundary,
		Keywords:    []string{"llx-"},
		SecretGroup: 1,
		Source:      "https://developers.llamaindex.ai/llamaparse/",
		Description: "LlamaCloud documents llx- API keys, shown only once and scoped to a user and project. The 44-52 alphanumeric body is the pinned scanner subset; public project IDs are not matched.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[4:]) },
	},
	{
		ID:          "mailersend-api-token",
		Regex:       `\b(mlsn\.[A-Za-z0-9]{30,100})` + catalogRightBoundary,
		Keywords:    []string{"mlsn."},
		SecretGroup: 1,
		Source:      "https://developers.mailersend.com/general.html",
		Description: "MailerSend API tokens authorize domain-scoped sending and account operations. mlsn. and the 30-100 alphanumeric body are pinned scanner constraints; no public domain ID or arbitrary bearer value is attributed to MailerSend.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[5:]) },
	},
	{
		ID:              "mem0-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?i:MEM0_API_KEY)["\']?[ \t]{0,4}[:=][ \t]{0,4}["\']?(m0-[A-Za-z0-9]{24,44})` + catalogRightBoundary,
		Keywords:        []string{"MEM0_API_KEY"},
		SecretGroup:     1,
		Source:          "https://docs.mem0.ai/platform/quickstart",
		Description:     "Mem0 documents MEM0_API_KEY as the credential used to read and write memories. Require that exact private-role assignment for the short m0- scanner prefix, rather than generic mem0 proximity or public user IDs.",
		ValidateContext: validateBetterleaksAssignment1,
	},
	{
		ID:          "mercury-production-api-token",
		Regex:       `\b(mercury_production_[a-z]{3,6}_[A-Za-z0-9]{40,50}_yrucrem)` + catalogRightBoundary,
		Keywords:    []string{"mercury_production_"},
		SecretGroup: 1,
		Source:      "https://docs.mercury.com/docs/getting-started.md",
		Description: "Mercury documents the production token sentinel pair and secret-token: authentication carrier, warning that stolen tokens permit account access. Preserve both sentinels and confidential middle material; 3-6/40-50 component bounds are scanner subsets.",
	},
	{
		ID:          "mergify-application-key",
		Regex:       `\b(mergify_application_key_[A-Za-z0-9_-]{40,200})` + catalogRightBoundary,
		Keywords:    []string{"mergify_application_key_"},
		SecretGroup: 1,
		Source:      "https://docs.mergify.com/api/usage.md",
		Description: "Mergify application keys are dashboard-generated bearer credentials with application permissions. The explicit mergify_application_key_ role prefix and 40-200 URL-safe body are pinned scanner constraints; GitHub credentials and Mergify user tokens remain their own families.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[24:]) },
	},
	{
		ID:          "midtrans-server-key",
		Regex:       `(?:^|[^A-Za-z0-9_./+\-])((?:SB-)?Mid-server-[A-Za-z0-9_]{10,20})` + catalogRightBoundary,
		Keywords:    []string{"Mid-server-"},
		SecretGroup: 1,
		Source:      "https://docs.midtrans.com/docs/api-authorization-headers.md",
		Description: "Midtrans requires keeping production and sandbox Server Keys confidential and using Client Keys on public frontends instead. Recognize Mid-server- and SB-Mid-server- with the 10-20 alphanumeric/underscore scanner body; public Mid-client- and SB-Mid-client- keys are excluded.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(strings.TrimPrefix(s, "SB-")[11:]) },
	},
	{
		ID:          "miro-oauth-token",
		Regex:       `\b(eyJtaXJv[A-Za-z0-9-]{10,64}_[A-Za-z0-9_-]{20,64})` + catalogRightBoundary,
		Keywords:    []string{"eyJtaXJv"},
		SecretGroup: 1,
		Source:      "https://developers.miro.com/docs/getting-started-with-oauth",
		Description: "Miro documents access and refresh tokens sharing a Base64-encoded JSON miro.origin prefix plus an opaque secret tail. Strictly decode the provider-specific prefix and require the origin field; a public client ID or origin metadata without a secret tail is not a token. Component bounds are the pinned scanner subset.",
		Validate:    validBetterleaksMiroToken1,
	},
	{
		ID:          "neon-api-key",
		Regex:       `\b(napi_[A-Za-z0-9]{64})` + catalogRightBoundary,
		Keywords:    []string{"napi_"},
		SecretGroup: 1,
		Source:      "https://neon.com/docs/manage/api-keys",
		Description: "Neon documents napi_ API keys as bearer secrets displayed only once, distinct from key IDs and project IDs. The 64-alphanumeric body follows the pinned scanner; documentation examples and bit-count wording are not treated as an exact issuance grammar.",
		Validate:    func(s string) bool { return validAuditedCarrierLiteral2(s[5:]) },
	},
}

// Family-specific structure checks run only after a selective catalog hit.
func validBetterleaksDevinToken1(s string) bool {
	if strings.HasPrefix(s, "apk_user_") {
		return validAuditedCarrierLiteral2(s[9:])
	}
	if strings.HasPrefix(s, "apk_") || strings.HasPrefix(s, "cog_") {
		return validAuditedCarrierLiteral2(s[4:])
	}
	return false
}

func validBetterleaksSwarmJoinToken1(s string) bool {
	offset := len("SWMTKN-1-")
	if strings.HasPrefix(s, "SWMTKN-2-1-") {
		offset = len("SWMTKN-2-1-")
	}
	if len(s) != offset+50+1+25 {
		return false
	}
	// Equal-width lowercase base36 strings compare in numeric order. These
	// maxima are 2^256-1 and 2^128-1, matching SwarmKit's generated material.
	return s[offset:offset+50] <= "6dp5qcb22im238nr3wvp0ic7q99w035jmy2iw7i6n43d37jtof" && s[offset+51:] <= "f5lxx1zz5pnorynqglhzmsp33"
}

func validBetterleaksSwarmUnlockKey1(s string) bool {
	if len(s) != 52 {
		return false
	}
	var decoded [32]byte
	n, err := serviceRawBase64.Decode(decoded[:], []byte(s[9:]))
	return err == nil && n == len(decoded)
}

func validBetterleaksSharedLiveKey1(s string) bool {
	if len(s) != 83 && len(s) != 84 {
		return false
	}
	var decoded [57]byte
	encoding := serviceRawBase64
	if s[len(s)-1] == '=' {
		encoding = serviceBase64
	}
	n, err := encoding.Decode(decoded[:], []byte(s[8:]))
	return err == nil && n == 56 && string(decoded[:4]) == "key_"
}

func validBetterleaksMiroToken1(s string) bool {
	separator := strings.IndexByte(s, '_')
	if separator < 0 {
		return false
	}
	var metadata struct {
		Origin string `json:"miro.origin"`
	}
	return serviceJSON(s[:separator], serviceRawBase64URL, &metadata) &&
		validAuditedCarrierLiteral2(metadata.Origin) && validAuditedCarrierLiteral2(s[separator+1:])
}

// Compose the shared assignment parser with the independent header-literal
// checks. Pure prefixed rules never call this assignment-only validator.
func validateBetterleaksAssignment1(value string, start, end int, secret string) contextValidation {
	if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	return validateAuditedHeaderLiteral2(value, start, end, secret)
}

func validateBetterleaksCloudsmithContext1(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(secret, "csa_") {
		return contextValidation{accepted: true}
	}
	if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	return validateAuditedHeaderLiteral2(value, start, end, secret)
}

func validateBetterleaksFacebookContext1(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(secret, "EAACEdEose0cBA") {
		return contextValidation{accepted: true}
	}
	return validateBetterleaksAssignment1(value, start, end, secret)
}
