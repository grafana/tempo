package secrets

// Matching portions adapted from Gitleaks v8.30.1 are licensed under LICENSE.gitleaks.
// Cloudflare modern, Google OAuth and Pinecone candidate constraints adapt
// TruffleHog v3.97.4 under the repository's AGPL-3.0 license; modified September 2026.

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	dopplerFormatSource = "https://docs.doppler.com/reference/auth-token-formats"
	vaultFormatSource   = "https://developer.hashicorp.com/vault/docs/concepts/tokens"
)

// These are credential candidates, not assertions that a credential was issued,
// remains active, or has any particular permissions. Bounds explicitly described
// as candidate bounds are Tempo limits, not promises about future vendor formats.
var cloudRuleSpecs = []catalogRuleSpec{
	{
		ID:       awsSecretAccessKeyRuleID,
		Regex:    `(?i)\b(?:aws_secret_access_key|secretaccesskey)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9/+=]{40})` + catalogRightBoundary,
		Keywords: []string{"aws_secret_access_key", "secretaccesskey"}, SecretGroup: 1,
		Source:      "https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html",
		Description: "AWS secret access key in an explicitly named environment/profile or SecretAccessKey assignment, using the documented 40-character example form. The public access-key ID and opaque bare strings are not detected; IAM and STS access-key examples use the same secret component.",
	},
	{
		ID:          "aws-amazon-bedrock-api-key-long-lived",
		Regex:       `\b(ABSK[A-Za-z0-9+/]{109,269}={0,2})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"absk"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Long-term Amazon Bedrock API-key candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Replaces the speculative encoded-tail range; the provider documents the console-key recognizer prefix, not a complete issuance grammar.",
	},
	{
		ID:       "aws-amazon-bedrock-api-key-short-lived",
		Regex:    `\b(bedrock-api-key-[A-Za-z0-9+/]+={0,2})` + catalogRightBoundary,
		Keywords: []string{"bedrock-api-key-"}, SecretGroup: 1, Validate: validBedrockShortKey,
		Source:      "https://github.com/aws/aws-bedrock-token-generator-python/blob/228eec2bfcf209d53dc776c902e00d9e6548e508/aws_bedrock_token_generator/token_generator.py",
		Description: "Short-term Bedrock token containing the complete base64-encoded presigned request and Version=1. Local validation requires Bedrock action, SigV4 credential scope, date, expiry, signed host header and a 256-bit signature; the public hostname marker alone is rejected. The 16 KiB encoded bound is defensive. No signature, expiry-against-current-time or credential verification is performed.",
	},
	{
		ID:          "azure-ad-client-secret",
		Regex:       `(?:^|[\\'"\x60\s>=:(,)])([a-zA-Z0-9_~.]{3}\dQ~[a-zA-Z0-9_~.-]{31,34})(?:$|[\\'"\x60\s<),])`,
		Keywords:    []string{"q~"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Identifiable Azure/Entra client-app password candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Microsoft classifier examples do not establish the issuer grammar or checksum validity.",
	},
	{
		ID:          "cloudflare-api-key",
		Regex:       `(?:(?i:\bcloudflare_api_token["']?[ \t]*[:=][ \t]*["']?)([A-Za-z0-9]{40})|\b(cf[ua]t_[A-Za-z0-9]{40}[a-f0-9]{8}))` + catalogRightBoundary,
		Keywords:    []string{"cloudflare_api_token", "cfut_", "cfat_"},
		SecretGroup: 0, // First populated capture preserves legacy secret-only extraction.
		Source:      "https://developers.cloudflare.com/fundamentals/api/get-started/token-formats/",
		Description: "Cloudflare user/account API-token candidates: unchanged legacy 40-character alphanumeric explicit CLOUDFLARE_API_TOKEN assignments, plus bare cfut_/cfat_ values. Modern 40-alphanumeric bodies and eight-lowercase-hex suffixes use pinned TruffleHog v3.97.4 v2 scanner constraints. Cloudflare documents the prefixes, 40-character body and CRC32 suffix, but not the complete checksum grammar. No checksum, entropy or authentication verification is performed.",
	},
	{
		ID:          "cloudflare-global-api-key",
		Regex:       `(?:(?i:\bcloudflare_global_api_key["']?[ \t]*[:=][ \t]*["']?)([a-f0-9]{37,45})|\b(cfk_[A-Za-z0-9]{40}[a-f0-9]{8}))` + catalogRightBoundary,
		Keywords:    []string{"cloudflare_global_api_key", "cfk_"},
		SecretGroup: 0, // First populated capture preserves legacy secret-only extraction.
		Source:      "https://developers.cloudflare.com/fundamentals/api/get-started/token-formats/",
		Description: "Cloudflare global API-key candidates: unchanged legacy 37-45 lowercase-hex explicit CLOUDFLARE_GLOBAL_API_KEY assignments, plus bare cfk_ values. Modern 40-alphanumeric bodies and eight-lowercase-hex suffixes use pinned TruffleHog v3.97.4 v2 scanner constraints. Cloudflare documents the prefix, 40-character body and CRC32 suffix, but not the complete checksum grammar. No checksum, entropy or authentication verification is performed.",
	},
	{
		ID:          "cloudflare-origin-ca-key",
		Regex:       `\b(v1\.0-[a-f0-9]{24}-[a-f0-9]{146})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"cloudflare", "v1.0-"},
		SecretGroup: 1,
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Cloudflare Origin CA service-key candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. The mandatory version prefix is provider-documented; component lengths remain example-backed. Historical secrets remain in scope despite announced September 2026 removal.",
	},
	{
		ID:          "databricks-api-token",
		Regex:       `\b(dapi[a-f0-9]{32}(?:-\d)?)(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"dapi"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Databricks personal-access-token candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Does not infer arbitrary generation suffixes from a Microsoft scanner example; matching a candidate within a suffixed value is not suffix validation.",
	},
	{
		ID:          "digitalocean-access-token",
		Regex:       `\b(doo_v1_[a-f0-9]{64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"doo_v1_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "DigitalOcean OAuth access-token candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. The provider documents a distinct prefix, not the former broad candidate body range.",
	},
	{
		ID:          "digitalocean-pat",
		Regex:       `\b(dop_v1_[a-f0-9]{64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"dop_v1_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "DigitalOcean personal-access-token candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. The provider documents a distinct prefix, not the former broad candidate body range.",
	},
	{
		ID:          "digitalocean-refresh-token",
		Regex:       `(?i)\b(dor_v1_[a-f0-9]{64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"dor_v1_"},
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "DigitalOcean OAuth refresh-token candidate using the pinned Gitleaks v8.30.1 full matching and capture baseline. The provider documents a distinct prefix; alphabet and width here are compatibility constraints rather than issuance promises.",
	},
	{
		ID: "doppler-api-token", Regex: `\b(dp\.pt\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.pt."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler personal token with the documented dp.pt. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "doppler-cli-token", Regex: `\b(dp\.ct\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.ct."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler CLI token with the documented dp.ct. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "doppler-service-token", Regex: `\b(dp\.st\.(?:[a-z0-9_\-]{2,35}\.)?[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.st."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler service token, including the optional documented 2-35-character configuration component and 40-44-character alphanumeric secret.",
	},
	{
		ID: "doppler-service-account-token", Regex: `\b(dp\.sa\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.sa."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler service-account token with the documented dp.sa. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "doppler-service-account-identity-token", Regex: `\b(dp\.said\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.said."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler short-lived service-account identity token with the documented dp.said. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "doppler-scim-token", Regex: `\b(dp\.scim\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.scim."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler SCIM token with the documented dp.scim. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "doppler-audit-token", Regex: `\b(dp\.audit\.[A-Za-z0-9]{40,44})` + catalogRightBoundary,
		Keywords: []string{"dp.audit."}, SecretGroup: 1,
		Source:      dopplerFormatSource,
		Description: "Doppler audit token with the documented dp.audit. prefix and 40-44 alphanumeric characters.",
	},
	{
		ID: "flyio-access-token", Regex: `\b((?:fm1a_|fm1r_|fm2_)[A-Za-z0-9+/]{64,1000}={0,2})` + catalogRightBoundary,
		Keywords: []string{"fm1a_", "fm1r_", "fm2_"}, SecretGroup: 1, Validate: validFlyAccessToken,
		Source:      "https://github.com/superfly/macaroon/blob/0f6cd2b7301a6d2ea66171bc966c904a01443c89/macaroon.go",
		Description: "Fly.io standard-base64 macaroon permission, authentication/discharge and v2 labels. Validates the four-field MessagePack envelope, both nonce revisions, 16-byte nonce, complete type/value caveat containers and 32-byte authenticator. Caveat bodies are checked for framing, not type-specific semantics or authorization; signatures and bundle relationships are not verified. The 64-1000-character body bound is defensive. Opaque fo1_ tokens and unlabelled encodings are not covered.",
	},
	{
		ID:          "harness-api-key",
		Regex:       `(?:pat|sat)\.[a-zA-Z0-9_-]{22}\.[a-zA-Z0-9]{24}\.[a-zA-Z0-9]{20}`,
		Keywords:    []string{"pat.", "sat."},
		SecretGroup: 0,
		Source:      gitleaksBaselineSource,
		Description: "Harness personal/service-account token candidate using the pinned Gitleaks v8.30.1 full matching and whole-match capture baseline. Provider documentation establishes four components, not the former broad identifier and secret bounds.",
	},
	{
		ID:          "hashicorp-tf-api-token",
		Regex:       `(?i)[a-z0-9]{14}\.(?-i:atlasv1)\.[a-z0-9\-_=]{60,70}`,
		Keywords:    []string{"atlasv1"},
		SecretGroup: 0,
		Entropy:     3.5,
		Source:      gitleaksBaselineSource,
		Description: "Terraform Cloud/Enterprise token candidate using the pinned Gitleaks v8.30.1 full matching, whole-match capture and entropy baseline. Official lifecycle documentation does not establish the former broad component bounds.",
	},
	{
		ID:          "heroku-api-key",
		Regex:       `(?i)[\w.-]{0,50}?(?:heroku)(?:[ \t\w.-]{0,20})[\s'"]{0,3}(?:=|>|:{1,3}=|\|\||:|=>|\?=|,)[\x60'"\s=]{0,5}([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"heroku"},
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Historical Heroku UUID-shaped credential candidate using the pinned Gitleaks v8.30.1 full assignment context and secret capture. Provider context must occur in the scanned value; bare UUIDs are not recognized by this family. Older unregenerated credentials remain in scope.",
	},
	{
		ID:          gcpAPIKeyRuleID,
		Regex:       `\b(AIza[\w-]{35})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"aiza"},
		SecretGroup: 1,
		Entropy:     4,
		Source:      gitleaksBaselineSource,
		Description: "Google API-key candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. A match does not establish confidentiality: restricted website or Firebase keys can be intentionally public, while other uses authorize requests or consume quota and billing.",
	},
	{
		ID:          "google-oauth-access-token",
		Regex:       `\b(ya29\.[A-Za-z0-9_-]{10,})` + catalogRightBoundary,
		Keywords:    []string{"ya29."},
		SecretGroup: 1,
		Source:      "https://developer.chrome.com/docs/webstore/using-api",
		Description: "Google OAuth access-token candidate with the example-backed ya29. prefix. The URL-safe ASCII body and ten-character minimum follow pinned TruffleHog v3.97.4 scanner constraints, not an issuance grammar; opaque tokens have no inferred internal structure or added maximum. Google requires access tokens to be stored securely. Client IDs, refresh tokens and other token prefixes are excluded; no expiry, scope or live verification is performed.",
	},
	{
		ID:          "heroku-api-key-v2",
		Regex:       `\b((HRKU-AA[0-9a-zA-Z_-]{58}))(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"hrku-aa"},
		SecretGroup: 1,
		Entropy:     4,
		Source:      gitleaksBaselineSource,
		Description: "Prefixed Heroku OAuth-token candidate using the pinned Gitleaks v8.30.1 full matching, first-secret capture and entropy baseline. The published total length alone does not justify a broader tail alphabet or dropping the upstream recognition bytes.",
	},
	{
		ID:          "infracost-api-token",
		Regex:       `\b(ico-[a-zA-Z0-9]{32})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"ico-"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "Hosted Infracost API-key candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. The historical maintainer source supports the hosted prefix and total length, not a universal grammar for legacy or user-defined self-hosted keys.",
	},
	{
		ID: "openshift-user-token", Regex: `\b(sha256~[A-Za-z0-9_][A-Za-z0-9_\-]{41}[AEIMQUYcgkosw048])` + catalogRightBoundary,
		Keywords: []string{"sha256~"}, SecretGroup: 1,
		Source:      "https://github.com/openshift/oauth-server/blob/master/pkg/osinserver/tokengen.go",
		Description: "OpenShift bearer token: sha256~ followed by 32 random bytes in canonical unpadded base64url; generation excludes a leading dash. The random.go encoder documents 43 characters. Nonsecret storage names produced by pkg/server/crypto/sha256.go share this shape, so name collisions cannot be distinguished from a bare value and are deliberately not prefix-suppressed.",
	},
	{
		ID:              "pinecone-api-key",
		Regex:           `(?:\b(pcsk_[A-Za-z0-9]{5,6}_[A-Za-z0-9]{63})|\b(?i:PINECONE_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))` + catalogRightBoundary,
		Keywords:        []string{"pcsk_", "pinecone_api_key"},
		SecretGroup:     0,
		Source:          "https://docs.pinecone.io/reference/cli/command-reference",
		Description:     "Pinecone confidential pcsk_ API-key candidate with the existing 5-6-character label and 63-character body, or an opaque UUID in the official PINECONE_API_KEY assignment. Widths remain pinned scanner subsets, not issuer guarantees. Public project/key-management UUIDs and labels are excluded; no issuance or permission verification is performed.",
		ValidateContext: validateBetterCarrierPinecone2,
	},
	{
		ID:          "planetscale-api-token",
		Regex:       `\b(pscale_tkn_(?i)[\w=\.-]{32,64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"pscale_tkn_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "PlanetScale service-token-secret candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Public management IDs are distinct; documentation does not establish the former broad tail bounds.",
	},
	{
		ID:          "planetscale-oauth-token",
		Regex:       `\b(pscale_oauth_[\w=\.-]{32,64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"pscale_oauth_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "PlanetScale OAuth-token candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Access and refresh roles have distinct documented presentations; no additional tail grammar is inferred from examples.",
	},
	{
		ID:          "planetscale-password",
		Regex:       `(?i)\b(pscale_pw_(?i)[\w=\.-]{32,64})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"pscale_pw_"},
		SecretGroup: 1,
		Entropy:     3,
		Source:      gitleaksBaselineSource,
		Description: "PlanetScale Vitess branch-password candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. Password-management names and usernames are distinct; no Postgres role-password coverage is implied.",
	},
	{
		ID:          "pulumi-api-token",
		Regex:       `\b(pul-[a-f0-9]{40})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{"pul-"},
		SecretGroup: 1,
		Entropy:     2,
		Source:      gitleaksBaselineSource,
		Description: "Pulumi access-token candidate using the pinned Gitleaks v8.30.1 full matching, capture and entropy baseline. The provider documents the post-June-2019 scanning prefix, not the former broad body range; older unprefixed tokens are not inferred.",
	},
	{
		ID:              "tencent-secret-key",
		Regex:           `\b(?i:TENCENTCLOUD_SECRET_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Za-z0-9]{16,128})` + catalogRightBoundary,
		Keywords:        []string{"tencentcloud_secret_key"},
		SecretGroup:     1,
		Source:          "https://www.tencentcloud.com/document/product/598/32675",
		Description:     "Tencent Cloud confidential SecretKey in an exact in-value TENCENTCLOUD_SECRET_KEY assignment, as used by the official SDK environment providers and Terraform provider. The 16-128 alphanumeric body is a local opaque-candidate subset, not an issuer grammar. SecretId identifiers, bare values, unqualified secret_key fields and attribute-name-only context are excluded. Shared literal and assignment guards reject placeholders, references and expression continuations; no live authentication is attempted.",
		Validate:        validAuditedCarrierLiteral2,
		ValidateContext: validateBetterCarrierAssignment2,
	},
	{
		ID: "vault-service-token", Regex: `\b(hvs\.[A-Za-z0-9_\-]{24,1000}(?:\.[A-Za-z0-9]{1,5})?)` + catalogRightBoundary,
		Keywords: []string{"hvs."}, SecretGroup: 1, Validate: validVaultServiceToken,
		Source:      "https://github.com/hashicorp/vault/blob/41e571b6606e3691e0ce3f82ae23d6db0e445118/vault/token_store.go",
		Description: "Modern Vault hvs. service tokens: 24-base62 random/root IDs, or version-1 SignedToken protobuf with a 32-byte HMAC and a nested nonempty token ID. Checks complete framing, not authorization, and preserves namespace suffixes. The 1000-character body cap is defensive. Historical single-letter s. candidates and other modern opaque/future internals are outside the curated native scope; tenants can configure targeted legacy rules.",
	},
	{
		ID: "vault-batch-token", Regex: `\b(hvb\.[A-Za-z0-9_\-]{24,1000}(?:\.[A-Za-z0-9]{1,5})?)` + catalogRightBoundary,
		Keywords: []string{"hvb."}, SecretGroup: 1, Validate: validVaultBatchToken,
		Source:      vaultFormatSource,
		Description: "Modern Vault hvb. encrypted batch tokens. Local validation checks unpadded base64url length and the version-1/version-2 AES-GCM envelope with at least 33 decoded bytes, preserving optional namespace suffixes. The 1000-character cap is defensive; authentication tags are not verified. Historical single-letter b. candidates are outside native scope and may be covered by targeted tenant rules.",
	},
	{
		ID: "vault-recovery-token", Regex: `\b(hvr\.[A-Za-z0-9]{24})` + catalogRightBoundary,
		Keywords: []string{"hvr."}, SecretGroup: 1,
		Source:      vaultFormatSource,
		Description: "Modern Vault recovery-token subset: hvr. values with the source-generated 24-base62 body. Generation is not authenticated. Historical single-letter r. candidates are outside the curated native scope; tenants needing them can configure targeted legacy rules.",
	},
	{
		ID:          "yandex-access-token",
		Regex:       `(?i)[\w.-]{0,50}?(?:yandex)(?:[ \t\w.-]{0,20})[\s'"]{0,3}(?:=|>|:{1,3}=|\|\||:|=>|\?=|,)[\x60'"\s=]{0,5}(t1\.[A-Z0-9a-z_-]+[=]{0,2}\.[A-Z0-9a-z_-]{86}[=]{0,2})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{yandexKeyword},
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Yandex IAM-token candidate using the pinned Gitleaks v8.30.1 full provider-assignment context and secret capture. Context must occur in the scanned value. The core representation already agrees with provider documentation: nonempty first component without an artificial maximum, an 86-character second component and optional padding. Bare-token misses are context differences, not length defects; signatures, expiry and issuance are not verified.",
	},
	{
		ID:          "yandex-api-key",
		Regex:       `(?i)[\w.-]{0,50}?(?:yandex)(?:[ \t\w.-]{0,20})[\s'"]{0,3}(?:=|>|:{1,3}=|\|\||:|=>|\?=|,)[\x60'"\s=]{0,5}(AQVN[A-Za-z0-9_\-]{35,38})(?:[\x60'"\s;]|\\[nr]|$)`,
		Keywords:    []string{yandexKeyword},
		SecretGroup: 1,
		Source:      gitleaksBaselineSource,
		Description: "Yandex service-account API-key candidate using the pinned Gitleaks v8.30.1 full provider-assignment context and secret capture. Provider context must occur in the scanned value; a masked example is not a complete issuer grammar.",
	},
	{
		ID: "yandex-secret-access-key", Regex: `\b(YC[A-Za-z0-9_\-]{38})` + catalogRightBoundary,
		Keywords: []string{"YC"}, SecretGroup: 1,
		Source:      "https://yandex.cloud/en/docs/iam/concepts/authorization/access-key",
		Description: "Yandex AWS-compatible secret access key: exactly 40 characters, beginning YC, with letters, digits, underscore and hyphen. The separately documented 25-character public key ID is excluded; this is a signing secret rather than an access token.",
	},
}

func validBedrockShortKey(candidate string) bool {
	const prefix = "bedrock-api-key-"
	if !strings.HasPrefix(candidate, prefix) || len(candidate) > 16*1024 {
		return false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(candidate[len(prefix):])
	if err != nil {
		return false
	}
	query, ok := strings.CutPrefix(string(decoded), "bedrock.amazonaws.com/?")
	if !ok {
		return false
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return false
	}
	// Ambiguous duplicate parameters are not a structurally valid generator output.
	for _, v := range values {
		if len(v) != 1 {
			return false
		}
	}
	if values.Get("Action") != "CallWithBearerToken" || values.Get("Version") != "1" ||
		values.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" || values.Get("X-Amz-SignedHeaders") != hostField {
		return false
	}
	signature := values.Get("X-Amz-Signature")
	if len(signature) != 64 {
		return false
	}
	for i := range signature {
		if (signature[i] < '0' || signature[i] > '9') && (signature[i] < 'a' || signature[i] > 'f') {
			return false
		}
	}
	date := values.Get("X-Amz-Date")
	if _, err := time.Parse("20060102T150405Z", date); err != nil {
		return false
	}
	expires, err := strconv.Atoi(values.Get("X-Amz-Expires"))
	if err != nil || expires < 1 || expires > 43200 {
		return false
	}
	scope := strings.Split(values.Get("X-Amz-Credential"), "/")
	return len(scope) == 5 && len(scope[0]) >= 16 && len(scope[0]) <= 128 &&
		scope[1] == date[:8] && scope[2] != "" && scope[3] == "bedrock" && scope[4] == "aws4_request"
}

func validVaultBatchToken(candidate string) bool {
	body, ok := strings.CutPrefix(candidate, "hvb.")
	if !ok {
		return false
	}
	body, _, _ = strings.Cut(body, ".")

	// Modern batch tokens encode four term bytes, one version byte,
	// a 12-byte nonce and at least a 16-byte GCM tag as unpadded base64url.
	// The regexp already checks the alphabet; a one-character remainder
	// cannot encode bytes. Only the header needs decoding to inspect its version.
	if len(body) < 44 || len(body)%4 == 1 {
		return false
	}
	var header [6]byte
	if _, err := base64.RawURLEncoding.Decode(header[:], []byte(body[:8])); err != nil {
		return false
	}
	return header[4] == 1 || header[4] == 2
}
