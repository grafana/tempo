package secrets

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math/bits"
	"net/url"
	"strings"
)

// Candidate constraints adapted from Titus v1.2.9 and TruffleHog v3.97.4.
// See LICENSE.titus, NOTICE.titus and the repository AGPL license. The
// corresponding fixture records separate provider evidence from scanner bounds.
var auditedTokensRules3 = []catalogRuleSpec{
	{
		ID:          "ramp-client-secret",
		Regex:       `\b(ramp_sec_[A-Za-z0-9]{48})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"ramp_sec_"},
		Source:      "https://docs.ramp.com/llms-guides/authorization.txt",
		Description: "Ramp confidential OAuth client-secret candidate, not the ramp_id_ client identifier. The 48-character alphanumeric body follows TruffleHog v3.97.4; no issuer validation.",
	},
	{
		ID:          "readme-api-key",
		Regex:       `\b(rdme_[a-z0-9]{70})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"rdme_"},
		Source:      "https://docs.readme.com/main/reference/authentication",
		Description: "ReadMe confidential documentation-management API-key candidate. The rdme_ prefix and 70 lowercase-alphanumeric body follow Titus v1.2.9 and TruffleHog v3.97.4; no issuer validation.",
	},
	{
		ID:          "recharge-api-key",
		Regex:       `\b(sk(?:_test)?_(?:1|2|3|5|10)x[123]_[A-Fa-f0-9]{64})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sk_"},
		Source:      "https://developer.rechargepayments.com/2021-11/getting-started/api-keys",
		Description: "Recharge confidential structured API-key candidate, excluding ambiguous unprefixed legacy hashes. Plan components and 64-hex tail follow TruffleHog v3.97.4, not a published complete issuer grammar.",
	},
	{
		ID:          "riot-api-key",
		Regex:       `\b(RGAPI-[A-Za-z0-9_-]{36})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"RGAPI-"},
		Source:      "https://developer.riotgames.com/docs/portal",
		Description: "Riot developer API-key candidate with RGAPI- and the bounded 36-character Titus v1.2.9 body. Public regional API hostnames are not credentials; no validity or expiration check.",
	},
	{
		ID:          "rootly-api-key",
		Regex:       `\b(rootly_[a-f0-9]{64})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"rootly_"},
		Source:      "https://docs.rootly.com/api-reference/overview",
		Description: "Rootly confidential incident-management API-key candidate. The rootly_ prefix and 64 lowercase-hex body follow TruffleHog v3.97.4; no API or issuer validation.",
	},
	{
		ID:          "saladcloud-api-key",
		Regex:       `\b(salad_cloud_[A-Za-z0-9]{1,7}_[A-Za-z0-9]{7,235})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"salad_cloud_"},
		Source:      "https://docs.salad.com/reference/api-usage",
		Description: "SaladCloud confidential API-key candidate with both nonempty components. Component bounds are TruffleHog v3.97.4 candidate constraints, not provider-issued lengths; public resource IDs and key-only prefixes are excluded.",
	},
	{
		ID:          "salesforce-access-token",
		Regex:       `\b(00[A-Za-z0-9]{13}![A-Za-z0-9_.]{96})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"!"},
		Source:      "https://developer.salesforce.com/blogs/2023/03/using-the-client-credentials-flow-for-easier-api-authentication",
		Description: "Salesforce confidential session/access-token candidate with organization prefix and opaque secret component. Widths follow Titus v1.2.9 and TruffleHog v3.97.4; public organization IDs and Salesforce hosts alone are excluded.",
	},
	{
		ID:          "salesforce-refresh-token",
		Regex:       `\b(5AEP861[A-Za-z0-9._=]{80,1000})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"5AEP861"},
		Source:      "https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_understanding_refresh_token_oauth.htm",
		Description: "Salesforce confidential refresh-token candidate using the pinned 5AEP861 prefix. The 80-character lower bound is scanner-derived and the 1000-character upper bound is defensive; no expiry or client-ID inference.",
	},
	{
		ID:              "scale-api-key",
		Regex:           `\b(?:SCALE_API_KEY|scale_api_key)[\t ]*["\']?[\t ]*[:=][\t ]*["\']?(live_[a-f0-9]{32})` + catalogRightBoundary,
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Keywords:        []string{"SCALE_API_KEY"},
		Source:          "https://scale.com/docs/api-reference/authentication",
		Description:     "Scale live API key in an exact same-value SCALE_API_KEY assignment. Bare live_ strings are shared with other products and not classified; 32 lowercase-hex body is a Titus v1.2.9 candidate constraint.",
	},
	{
		ID:          "scale-callback-secret",
		Regex:       `\b(live_auth_[a-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"live_auth_"},
		Source:      "https://scale.com/docs/api-reference/authentication",
		Description: "Scale callback authentication secret candidate, distinct from its live API key. The live_auth_ prefix and 32-hex body follow Titus v1.2.9; callback header comparison is performed by the receiver, not this detector.",
	},
	{
		ID:          "scalingo-api-token",
		Regex:       `\b(tk-us-[A-Za-z0-9_-]{48})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"tk-us-"},
		Source:      "https://developers.scalingo.com/tokens",
		Description: "Scalingo long-lived confidential API-exchange token with the provider-documented tk-us- prefix. The 48-character URL-safe body follows Titus v1.2.9 and provider examples, not an exhaustive issuance grammar.",
	},
	{
		ID:          "scrapfly-api-key",
		Regex:       `\b(scp-(?:live|test)-[a-z0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"scp-"},
		Source:      "https://scrapfly.io/docs/scrape-api/getting-started",
		Description: "Scrapfly confidential live/test prefixed API-key candidate. The 32 lowercase-alphanumeric body follows TruffleHog v3.97.4; legacy unprefixed random strings are excluded.",
	},
	{
		ID:          "shopify-client-secret",
		Regex:       `\b(shpss_[A-Fa-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"shpss_"},
		Source:      "https://shopify.dev/docs/apps/build/authentication-authorization/manage-credentials",
		Description: "Shopify confidential OAuth client-secret candidate, distinct from the public app client ID. The shpss_ prefix and 32-hex body follow Titus v1.2.9; no scope or issuer validation.",
	},
	{
		ID:          "shopify-custom-app-token",
		Regex:       `\b(shpca_[A-Fa-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"shpca_"},
		Source:      "https://shopify.dev/docs/apps/build/authentication-authorization/access-tokens",
		Description: "Shopify confidential custom-app access-token candidate using shpca_, separate from existing shpat_ and shppa_ families. Prefix and body width follow Titus v1.2.9; public shop domains and app IDs are excluded.",
	},
	{
		ID:          "sonarcloud-api-token",
		Regex:       `\b(sqco_[A-Za-z0-9]{59})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sqco_"},
		Source:      "https://docs.sonarsource.com/sonarqube-cloud/managing-your-account/managing-tokens",
		Description: "SonarCloud confidential sqco_ API-token candidate, separate from SonarQube sq[uap]_ tokens. The 59-character alphanumeric body follows TruffleHog v3.97.4, not issuer verification.",
	},
	{
		ID:          "spectralops-personal-token",
		Regex:       `\b(spu-[a-z0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"spu-"},
		Source:      "https://guides.spectralops.io/reference/authentication",
		Description: "SpectralOps confidential personal API-token candidate. The spu- prefix and 32 lowercase-alphanumeric body follow TruffleHog v3.97.4; no account or active-token validation.",
	},
	{
		ID:              "square-access-token",
		Regex:           `\b(?:(sq0atp-[A-Za-z0-9_-]{22})|(?:SQUARE_ACCESS_TOKEN|square_access_token)[\t ]*["\']?[\t ]*[:=][\t ]*["\']?(EAAA[A-Za-z0-9_+=-]{60}))` + catalogRightBoundary,
		SecretGroup:     0,
		ValidateContext: validateAuditedAssignmentContext,
		Keywords:        []string{"sq0atp-", "SQUARE_ACCESS_TOKEN"},
		Source:          "https://developer.squareup.com/docs/build-basics/access-tokens",
		Description:     "Square confidential legacy sq0atp- access-token family and exact SQUARE_ACCESS_TOKEN assignments for the ambiguous EAAA prefix. Candidate widths follow Titus v1.2.9 and TruffleHog v3.97.4; EAAA alone and public application IDs are excluded.",
	},
	{
		ID:          "square-client-secret",
		Regex:       `\b((?:sandbox-)?sq0cs[pb]-[A-Za-z0-9_-]{43})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sq0cs"},
		Source:      "https://developer.squareup.com/docs/build-basics/access-tokens",
		Description: "Square confidential OAuth application-secret candidate for production and sandbox, excluding sq0i public IDs. The 43-character body and sp/sb spellings are bounded pinned scanner assumptions; arbitrary sq0c suffixes are not accepted.",
	},
	{
		ID:          "stackhawk-api-key",
		Regex:       `\b(hawk\.[A-Za-z0-9_-]{20}\.[A-Za-z0-9_-]{20})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"hawk."},
		Source:      "https://docs.stackhawk.com/apidocs/",
		Description: "StackHawk long-lived confidential API-key candidate containing both components. The hawk. marker and 20-character URL-safe components follow Titus v1.2.9, not an issuance guarantee; token IDs alone are excluded.",
	},
	{
		ID:          "storyblok-personal-access-token",
		Regex:       `\b([A-Za-z0-9]{22}tt-[0-9]{6}-[A-Za-z0-9_-]{20})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"tt-"},
		Source:      "https://www.storyblok.com/docs/concepts/access-tokens",
		Description: "Storyblok confidential Management API personal-access-token candidate with all three components. Widths follow TruffleHog v3.97.4; shorter public/preview Content Delivery tokens do not determine confidentiality and are excluded.",
	},
	{
		ID:          "stripe-payment-intent-client-secret",
		Regex:       `\b(pi_[A-Za-z0-9]{24}_secret_[A-Za-z0-9]{25})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"_secret_"},
		Source:      "https://docs.stripe.com/api/payment_intents/object",
		Description: "Stripe scoped PaymentIntent client-secret candidate, not a publishable API key or public PaymentIntent ID. Component widths follow TruffleHog v3.97.4 and provider examples; no intent state or authorization check.",
	},
	{
		ID:          "tavily-api-key",
		Regex:       `\b(tvly-[A-Za-z0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"tvly-"},
		Source:      "https://docs.tavily.com/documentation/api-reference/introduction",
		Description: "Tavily confidential API-key candidate using the documented tvly- prefix. The 32-character alphanumeric body follows Titus v1.2.9 rather than a published complete issuer grammar.",
	},
	{
		ID:          "teamcity-api-token",
		Regex:       `\b(eyJ0eXAiOiAiVENWMiJ9\.[A-Za-z0-9_-]{36}\.[A-Za-z0-9_-]{48})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"eyJ0eXAiOiAiVENWMiJ9."},
		Source:      "https://www.jetbrains.com/help/teamcity/configuring-your-user-profile.html",
		Description: "TeamCity TCV2 confidential access-token candidate with its fixed Base64URL type marker. The 36/48 component widths follow Titus v1.2.9. This is a proprietary token, not a JWT: its payload is not required to be JSON and the header has no alg claim.",
	},
	{
		ID:          "tickettailor-api-key",
		Regex:       `\b(sk_[0-9]{4}_[0-9]{6}_[a-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"sk_"},
		Source:      "https://developers.tickettailor.com/docs/api/ticket-tailor-api/",
		Description: "TicketTailor confidential API-key candidate with numeric components and a secret tail. Component widths follow TruffleHog v3.97.4, not date semantics or an exhaustive issuer grammar.",
	},
	{
		ID:          "together-api-key",
		Regex:       `\b(tgp_v1_[A-Za-z0-9_-]{43})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"tgp_v1_"},
		Source:      "https://docs.together.ai/docs/quickstart",
		Description: "Together AI confidential versioned API-key candidate. The tgp_v1_ marker and 43-character URL-safe body follow Titus v1.2.9; no issuance or billing-scope validation.",
	},
	{
		ID:          "trufflehog-enterprise-secret",
		Regex:       `\b(thog-secret-[a-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"thog-secret-"},
		Source:      "https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/trufflehogenterprise/trufflehogenterprise.go",
		Description: "TruffleHog Enterprise confidential X-Thog-Secret component. The branded prefix and 32-hex body come from the provider-maintained v3.97.4 detector; thog-key- identifiers and tenant hostnames alone are excluded.",
	},
	{
		ID:          "twitch-stream-key",
		Regex:       `\b(live_[0-9]{8,12}_[A-Za-z0-9]{30,36})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"live_"},
		Source:      "https://dev.twitch.tv/docs/video-broadcast/",
		Description: "Twitch confidential broadcast stream-key candidate with both account and secret components. Account/secret widths follow Titus v1.2.9, not provider validity or a promise about account-ID lengths.",
	},
	{
		ID:              "twitch-oauth-secret",
		Regex:           `\b(?:TWITCH_CLIENT_SECRET|twitch_client_secret|TWITCH_ACCESS_TOKEN|twitch_access_token)[\t ]*["\']?[\t ]*[:=][\t ]*["\']?([a-z0-9]{30})` + catalogRightBoundary,
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Keywords:        []string{"TWITCH_CLIENT_SECRET", "TWITCH_ACCESS_TOKEN"},
		Source:          "https://dev.twitch.tv/docs/authentication/",
		Description:     "Twitch confidential client secret or access token in exact provider-qualified same-value assignments. The 30 lowercase-alphanumeric body follows TruffleHog v3.97.4; bare random values and client IDs remain excluded.",
	},
	{
		ID:          "ubidots-api-token",
		Regex:       `\b(BBFF-[A-Za-z0-9]{30,55})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"BBFF-"},
		Source:      "https://docs.ubidots.com/reference/authentication",
		Description: "Ubidots confidential API-token candidate with documented BBFF- prefix. The 30-character minimum follows TruffleHog v3.97.4; the 55-character maximum accommodates the provider authentication example, not an exhaustive generator grammar.",
	},
	{
		ID:          "voiceflow-api-key",
		Regex:       `\b(VF\.(?:(?:DM|WS)\.)?[A-Fa-f0-9]{24}\.[A-Za-z0-9]{16})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"VF."},
		Source:      "https://github.com/trufflesecurity/trufflehog/blob/v3.97.4/pkg/detectors/voiceflow/voiceflow.go",
		Description: "Voiceflow confidential Dialog Manager, Workspace and legacy Workspace API-key candidates with literal separators and both ID and secret. The 24-hex/16-alphanumeric layout follows the pinned detector and referenced provider runtime examples; no online verification or new vfp_ PAT coverage is implied.",
	},
	{
		ID:              "wakatime-api-key",
		Regex:           `\b(?:(waka_[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12})|(?:WAKATIME_API_KEY|wakatime_api_key)[\t ]*["']?[\t ]*[:=][\t ]*["']?([a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}))` + catalogRightBoundary,
		SecretGroup:     0,
		ValidateContext: validateAuditedAssignmentContext,
		Keywords:        []string{"waka_", "WAKATIME_API_KEY"},
		Source:          "https://github.com/wakatime/wakatime-cli/blob/5fb9caaef329a697c196ab67747d7875b5749c8b/pkg/params/params.go",
		Description:     "WakaTime confidential waka_-prefixed UUIDv4 key and exact same-value legacy API-key assignment. The provider CLI requires a lowercase UUIDv4 after the optional prefix, correcting Titus's unsupported unhyphenated alphanumeric candidate. Public OAuth client IDs and unassigned UUIDs are excluded; no issuer validation.",
	},
	{
		ID:              "weights-and-biases-legacy-api-key",
		Regex:           `\b(?:WANDB_API_KEY|wandb_api_key)[\t ]*["\']?[\t ]*[:=][\t ]*["\']?([a-f0-9]{40})` + catalogRightBoundary,
		SecretGroup:     1,
		ValidateContext: validateAuditedAssignmentContext,
		Keywords:        []string{"WANDB_API_KEY"},
		Source:          "https://docs.wandb.ai/platform/hosting/iam/api-keys",
		Description:     "Weights & Biases confidential legacy API-key candidate in an exact same-value WANDB_API_KEY assignment. Bare SHA1-like strings are excluded; 40 lowercase-hex characters follow the pinned scanner, not hash or issuance validation.",
	},
	{
		ID:          "zoho-oauth-token",
		Regex:       `\b(1000\.[a-f0-9]{32}\.[a-f0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"1000."},
		Source:      "https://www.zoho.com/crm/developer/docs/api/v8/access-refresh.html",
		Description: "Zoho confidential OAuth-token candidate with both opaque components. The 1000. marker and two 32-hex components follow Titus v1.2.9 and TruffleHog v3.97.4; no access/refresh role or expiry inference.",
	},
	{
		ID:          "zuplo-api-key",
		Regex:       `\b(zpka_[a-z0-9]{32}_[a-f0-9]{8})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"zpka_"},
		Source:      "https://zuplo.com/blog/api-key-authentication",
		Description: "Zuplo confidential API-key candidate with secret and checksum components. The provider documents the prefix and checksum role but not its algorithm; widths/alphabet follow Titus v1.2.9 and no invented checksum verification is performed.",
	},
	{
		ID:              "sonar-legacy-api-token",
		Regex:           `(?:\b(?:SONAR_TOKEN|sonar\.token)|-Dsonar\.token)[\t ]*["\']?[\t ]*[:=][\t ]*["\']?([a-z0-9]{40})` + catalogRightBoundary,
		SecretGroup:     1,
		ValidateContext: validateAuditedSonarAssignment3,
		Keywords:        []string{"SONAR_TOKEN", "sonar.token"},
		Source:          "https://docs.sonarsource.com/sonarqube-server/2025.5/analyzing-source-code/analysis-parameters/parameters-not-settable-in-ui/",
		Description:     "Sonar confidential legacy token in the explicit SONAR_TOKEN or sonar.token authentication assignment, including the documented -Dsonar.token CLI property. The CLI option must start the value or follow whitespace/a quote. The 40 lowercase-alphanumeric body follows the pinned legacy scanner; bare hashes, sonar project IDs and ambiguous sonar.login usernames are excluded.",
	},
	{
		ID:          "salesforce-client-credentials",
		Regex:       `(?s)(\{[^{}]{1,1000}\})`,
		SecretGroup: 1,
		Keywords:    []string{"{"},
		Source:      "https://developer.salesforce.com/blogs/2023/03/using-the-client-credentials-flow-for-easier-api-authentication",
		Description: "Salesforce complete flat JSON client-credentials carrier binding client_id, confidential client_secret and an instance_url on my.salesforce.com. Opaque component widths follow TruffleHog v3.97.4; the 1000-byte candidate bound is defensive. Public client IDs or hosts alone do not match.",
		Validate:    validAuditedSalesforceCredentials3,
	},
	{ // #nosec G101 -- This is a detection pattern for PGP armor, not private key material.
		ID:          "pgp-private-key",
		Regex:       `(?m)^(-----BEGIN PGP PRIVATE KEY BLOCK-----[\t ]*\r?\n(?:[A-Za-z][A-Za-z -]*:[^\r\n]*\r?\n)*\r?\n[A-Za-z0-9+/=\r\n]+-----END PGP PRIVATE KEY BLOCK-----)[\t ]*\r?$`,
		SecretGroup: 1,
		Keywords:    []string{"-----BEGIN PGP PRIVATE KEY BLOCK-----"},
		Source:      "https://www.rfc-editor.org/rfc/rfc9580.html#section-5.5.3",
		Description: "Unencrypted OpenPGP armored secret-key candidate. Requires complete armor, Base64, optional CRC24, definite packet lengths and a parsed v3/v4/v6 secret-key packet with S2K usage zero, algorithm-specific secret fields and the v3/v4 checksum. Public-only keyrings, encrypted or compressed blobs and arbitrary armor are excluded; 16 KiB is a defensive bound, and identity/signature trust is not established.",
		Validate:    validAuditedPGPPrivateKey3,
	},
	{
		ID:          "slack-workspace-access-token",
		Regex:       `\b(xoxa-[0-9]{10,13}-[0-9]{10,13}(?:-[0-9]{10,13})?-[A-Za-z0-9]{32})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"xoxa-"},
		Source:      "https://docs.slack.dev/changelog/2018/04/01/oauth-flow-changes-for-workspace-token-preview-apps",
		Description: "Slack historical confidential workspace access token with ID components and a nonempty secret tail. Numeric widths follow TruffleHog v3.97.4; the 32-character tail follows Titus v1.2.9 historical examples, not issuer validity. Retired tokens still contain historical secret material; ID-only prefixes are excluded.",
	},
	{
		ID:          "slack-workspace-refresh-token",
		Regex:       `\b(xoxr-[0-9]{10,13}-[0-9]{10,13}-[A-Za-z0-9]{24})` + catalogRightBoundary,
		SecretGroup: 1,
		Keywords:    []string{"xoxr-"},
		Source:      "https://docs.slack.dev/changelog/2018/08/01/workspace-token-rotation",
		Description: "Slack historical confidential workspace refresh token with both ID components and a secret tail. Numeric widths follow TruffleHog v3.97.4; the 24-character tail follows Titus v1.2.9 historical examples rather than an issuer guarantee. Retirement does not erase confidentiality of historical plaintext.",
	},
}

func validateAuditedSonarAssignment3(value string, matchStart, matchEnd int, secret string) contextValidation {
	if strings.HasPrefix(value[matchStart:matchEnd], "-D") && matchStart > 0 && !strings.ContainsRune(" \t\r\n\"'", rune(value[matchStart-1])) {
		return contextValidation{}
	}
	return validateAuditedAssignmentContext(value, matchStart, matchEnd, secret)
}

var (
	auditedSalesforceClientID3     = newLazyRegexp(`^3MVG9[A-Za-z0-9._+/=]{80,251}$`)
	auditedSalesforceClientSecret3 = newLazyRegexp(`^(?:[A-Za-z0-9+/=.]{64}|[0-9]{19})$`)
)

func validAuditedSalesforceCredentials3(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var fields struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		InstanceURL  string `json:"instance_url"`
	}
	if json.Unmarshal([]byte(s), &fields) != nil || !auditedSalesforceClientID3.MatchString(fields.ClientID) || !auditedSalesforceClientSecret3.MatchString(fields.ClientSecret) {
		return false
	}
	u, err := url.Parse(fields.InstanceURL)
	return err == nil && u.Scheme == httpsScheme && u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/") && strings.HasSuffix(u.Host, ".my.salesforce.com") && len(u.Host) > len(".my.salesforce.com")
}

// Only the explicitly armored OpenPGP protocol is decoded. This does not
// recursively inspect arbitrary Base64, compressed packets or encrypted blobs.
func validAuditedPGPPrivateKey3(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	_, body, ok := strings.Cut(s, "\n")
	if !ok {
		return false
	}
	// Armor headers precede the mandatory empty separator line.
	for {
		line, rest, found := strings.Cut(body, "\n")
		if !found {
			return false
		}
		body = rest
		if strings.TrimSuffix(line, "\r") == "" {
			break
		}
		if !strings.Contains(line, ":") {
			return false
		}
	}
	body, ok = strings.CutSuffix(body, "-----END PGP PRIVATE KEY BLOCK-----")
	if !ok {
		return false
	}
	body = strings.TrimSpace(body)
	var checksum []byte
	if i := strings.LastIndex(body, "\n="); i >= 0 {
		var err error
		checksum, err = base64.StdEncoding.Strict().DecodeString(body[i+2:])
		if err != nil || len(checksum) != 3 {
			return false
		}
		body = body[:i]
	}
	data, err := base64.StdEncoding.Strict().DecodeString(body)
	if err != nil || len(data) == 0 {
		return false
	}
	if len(checksum) != 0 {
		crc := uint32(0xb704ce)
		for _, b := range data {
			crc ^= uint32(b) << 16
			for range 8 {
				crc <<= 1
				if crc&0x1000000 != 0 {
					crc ^= 0x1864cfb
				}
			}
		}
		if crc&0xffffff != uint32(checksum[0])<<16|uint32(checksum[1])<<8|uint32(checksum[2]) {
			return false
		}
	}
	foundSecret := false
	for len(data) > 0 {
		tag, packet, rest, ok := auditedPGPPacket3(data)
		if !ok {
			return false
		}
		data = rest
		switch tag {
		case 5, 7:
			if auditedPGPSecretPacket3(packet) {
				foundSecret = true
			}
		case 2, 6, 12, 13, 14, 17:
			// Public keys, signatures and identities may accompany secret keys;
			// they never establish confidentiality on their own.
		default:
			return false
		}
	}
	return foundSecret
}

// RFC 4880 section 4.2: definite new/old packet lengths. Partial and
// indeterminate data packets are not a transferable private-key carrier.
func auditedPGPPacket3(data []byte) (byte, []byte, []byte, bool) {
	if len(data) < 2 || data[0]&0x80 == 0 {
		return 0, nil, nil, false
	}
	tag := (data[0] >> 2) & 15
	pos := 1
	var size uint32
	if data[0]&0x40 != 0 {
		tag = data[0] & 63
		first := data[pos]
		pos++
		switch {
		case first < 192:
			size = uint32(first)
		case first < 224:
			if pos >= len(data) {
				return 0, nil, nil, false
			}
			size = (uint32(first)-192)<<8 + uint32(data[pos]) + 192
			pos++
		case first == 255:
			if len(data)-pos < 4 {
				return 0, nil, nil, false
			}
			size = binary.BigEndian.Uint32(data[pos:])
			pos += 4
		default:
			return 0, nil, nil, false
		}
	} else {
		width := 1 << (data[0] & 3)
		if width == 8 || len(data)-pos < width {
			return 0, nil, nil, false
		}
		for range width {
			size = size<<8 | uint32(data[pos])
			pos++
		}
	}
	if size == 0 || uint64(size) > uint64(len(data)-pos) {
		return 0, nil, nil, false
	}
	end := pos + int(size)
	return tag, data[pos:end], data[end:], true
}

// Consume exactly one canonically encoded, nonzero multiprecision integer.
func auditedPGPMPI3(data []byte) ([]byte, bool) {
	if len(data) < 3 {
		return nil, false
	}
	nbits := int(binary.BigEndian.Uint16(data))
	nbytes := (nbits + 7) / 8
	if nbits == 0 || nbytes > len(data)-2 || nbits != (nbytes-1)*8+bits.Len8(data[2]) {
		return nil, false
	}
	return data[2+nbytes:], true
}

// RFC 4880/6637/9580 algorithm-specific framing, without signature or
// identity verification. S2K usage zero is essential: encrypted secret-key
// packets and GNU offline-key stubs do not expose plaintext private material.
func auditedPGPSecretPacket3(packet []byte) bool {
	if len(packet) < 8 {
		return false
	}
	version, algorithm, pos := packet[0], packet[5], 6
	if version == 3 {
		algorithm, pos = packet[7], 8
		if algorithm < 1 || algorithm > 3 {
			return false
		}
	} else if version != 4 && version != 6 {
		return false
	}
	publicEnd := -1
	if version == 6 {
		if len(packet)-pos < 4 {
			return false
		}
		size := binary.BigEndian.Uint32(packet[pos:])
		pos += 4
		if uint64(size) >= uint64(len(packet)-pos) {
			return false
		}
		publicEnd = pos + int(size)
	}
	data := packet[pos:]
	publicMPIs, secretMPIs, rawWidth := 0, 0, 0
	switch algorithm {
	case 1, 2, 3:
		publicMPIs, secretMPIs = 2, 4
	case 16:
		publicMPIs, secretMPIs = 3, 1
	case 17:
		publicMPIs, secretMPIs = 4, 1
	case 18, 19, 22:
		if len(data) < 2 || data[0] == 0 || data[0] == 255 || int(data[0]) >= len(data) {
			return false
		}
		data = data[1+int(data[0]):]
		publicMPIs, secretMPIs = 1, 1
	case 25, 27:
		rawWidth = 32
	case 26:
		rawWidth = 56
	case 28:
		rawWidth = 57
	default:
		return false
	}
	var rawPublic []byte
	if rawWidth != 0 {
		if len(data) < rawWidth {
			return false
		}
		rawPublic, data = data[:rawWidth], data[rawWidth:]
	}
	for range publicMPIs {
		rest, ok := auditedPGPMPI3(data)
		if !ok {
			return false
		}
		data = rest
	}
	if algorithm == 18 {
		if len(data) < 4 || data[0] != 3 || data[1] != 1 || data[2] < 8 || data[2] > 10 || data[3] < 7 || data[3] > 9 {
			return false
		}
		data = data[4:]
	}
	if publicEnd != -1 && len(packet)-len(data) != publicEnd {
		return false
	}
	if len(data) == 0 || data[0] != 0 {
		return false
	}
	data = data[1:]
	secret := data
	if version != 6 {
		if len(secret) < 3 {
			return false
		}
		secret = secret[:len(secret)-2]
		var checksum uint16
		for _, b := range secret {
			checksum += uint16(b)
		}
		if checksum != binary.BigEndian.Uint16(data[len(data)-2:]) {
			return false
		}
	}
	if rawWidth != 0 {
		if len(secret) != rawWidth {
			return false
		}
		if algorithm == 27 {
			return bytes.Equal(rawPublic, ed25519.NewKeyFromSeed(secret).Public().(ed25519.PublicKey))
		}
		return true
	}
	for range secretMPIs {
		rest, ok := auditedPGPMPI3(secret)
		if !ok {
			return false
		}
		secret = rest
	}
	return len(secret) == 0
}
