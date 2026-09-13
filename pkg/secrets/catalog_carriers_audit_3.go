package secrets

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode/utf16"
)

// Complete Authorization: Token credentials documented by the audited APIs.
// #nosec G101 -- Regular expression for credential shapes, not a credential value.
const auditedAuthorizationTokenBody = `(?:[A-Za-z0-9]{12}|[A-Za-z0-9]{32}|[A-Za-z0-9]{64}|[a-z0-9]{40}|[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})`

// Candidate constraints informed by Titus v1.2.9 (LICENSE.titus) and
// TruffleHog v3.97.4 (AGPL-3.0); protocol parsers are implemented natively.
var auditedCarriersRules3 = []catalogRuleSpec{
	{
		ID:              "oauth-client-secret-assignment",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{24})"|'([A-Za-z0-9_-]{24})'|([A-Za-z0-9_-]{24})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{clientSecretField},
		Source:          "https://www.rfc-editor.org/rfc/rfc6749#section-2.3.1",
		Description:     "Exact OAuth client_secret literal assignment in the scanned value. The 24-character alphanumeric/underscore/hyphen form is a Titus v1.2.9 candidate constraint, not Google attribution or the full OAuth grammar; public client IDs and references are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "onelogin-oauth-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9]{64})"|'(?:[A-Za-z0-9]{64})'|(?:[A-Za-z0-9]{64}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{64})"|'([A-Za-z0-9]{64})'|([A-Za-z0-9]{64}))|(?:"[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{64})"|'([A-Za-z0-9]{64})'|([A-Za-z0-9]{64}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Oo][Nn][Ee][Ll][Oo][Gg][Ii][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9]{64})"|'(?:[A-Za-z0-9]{64})'|(?:[A-Za-z0-9]{64})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"ONELOGIN_CLIENT_SECRET"},
		Source:          "https://developers.onelogin.com/api-docs/1/oauth20-tokens/generate-tokens-2/",
		Description:     "Complete named ONELOGIN_CLIENT_ID and ONELOGIN_CLIENT_SECRET credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "onepagecrm-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd]"|'[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd]'|[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{24})"|'(?:[a-z0-9]{24})'|(?:[a-z0-9]{24}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=))|(?:"[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd]"|'[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd]'|[Oo][Nn][Ee][Pp][Aa][Gg][Ee][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{24})"|'(?:[a-z0-9]{24})'|(?:[a-z0-9]{24})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"ONEPAGECRM_API_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/onepagecrm",
		Description:     "Complete named ONEPAGECRM_USER_ID and ONEPAGECRM_API_KEY credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "openvpn-oauth-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40})"|'(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40})'|(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{64,1000})"|'([A-Za-z0-9_-]{64,1000})'|([A-Za-z0-9_-]{64,1000}))|(?:"[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{64,1000})"|'([A-Za-z0-9_-]{64,1000})'|([A-Za-z0-9_-]{64,1000}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Oo][Pp][Ee][Nn][Vv][Pp][Nn]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40})"|'(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40})'|(?:[A-Za-z0-9-]{3,40}\.[A-Za-z0-9-]{3,40})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"OPENVPN_CLIENT_SECRET"},
		Source:          "https://openvpn.net/cloud-docs/developer/creating-api-credentials.html",
		Description:     "Named OpenVPN client ID and client secret in one value, in either order. Explicit provider-qualified secret naming establishes confidentiality without inferring a tenant from unrelated domains. TruffleHog v3.97.4 ID shape and 64-character lower bound are retained; the 1000-character cap and 514-byte pairing window are defensive.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "paypal-oauth-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99}))"|'(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99}))'|(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99})))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_.-]{44,120})"|'([A-Za-z0-9_.-]{44,120})'|([A-Za-z0-9_.-]{44,120}))|(?:"[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_.-]{44,120})"|'([A-Za-z0-9_.-]{44,120})'|([A-Za-z0-9_.-]{44,120}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]"|'[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd]'|[Pp][Aa][Yy][Pp][Aa][Ll]_[Cc][Ll][Ii][Ee][Nn][Tt]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99}))"|'(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99}))'|(?:(?:[A-Za-z0-9_.]{7}-[A-Za-z0-9_.]{72}|[A-Za-z0-9_.]{5}-[A-Za-z0-9_.]{38}|A[A-Za-z0-9_-]{78,99}))))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PAYPAL_CLIENT_SECRET"},
		Source:          "https://developer.paypal.com/api/rest/authentication/",
		Description:     "Complete named PayPal OAuth client ID and confidential client secret, in either field order. The union of Titus v1.2.9 and TruffleHog v3.97.4 candidate bounds covers legacy and current-looking shapes without treating public client IDs as secrets or claiming issuer validity; pairing is bounded to 514 bytes.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "percy-project-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Ee][Rr][Cc][Yy]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Ee][Rr][Cc][Yy]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Ee][Rr][Cc][Yy]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Fa-f0-9]{64})"|'([A-Fa-f0-9]{64})'|([A-Fa-f0-9]{64})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PERCY_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/percy",
		Description:     "Exact in-value PERCY_TOKEN secret assignment. Pinned scanner candidate width/alphabet retained; bare opaque values and public identifiers are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "phrase-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Hh][Rr][Aa][Ss][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Hh][Rr][Aa][Ss][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Hh][Rr][Aa][Ss][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-z0-9]{64})"|'([a-z0-9]{64})'|([a-z0-9]{64})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PHRASE_ACCESS_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/phraseaccesstoken",
		Description:     "Exact in-value PHRASE_ACCESS_TOKEN secret assignment. Pinned scanner candidate width/alphabet retained; bare opaque values and public identifiers are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pinata-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{20})"|'(?:[a-z0-9]{20})'|(?:[a-z0-9]{20}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-z0-9]{64})"|'([a-z0-9]{64})'|([a-z0-9]{64}))|(?:"[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Ii][Nn][Aa][Tt][Aa]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-z0-9]{64})"|'([a-z0-9]{64})'|([a-z0-9]{64}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Ii][Nn][Aa][Tt][Aa]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{20})"|'(?:[a-z0-9]{20})'|(?:[a-z0-9]{20})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PINATA_SECRET_API_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/pinata",
		Description:     "Complete named PINATA_API_KEY and PINATA_SECRET_API_KEY credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "leankit-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll]"|'[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll]'|[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.leankit\.com)"|'(?:https://[A-Za-z0-9-]+\.leankit\.com)'|(?:https://[A-Za-z0-9-]+\.leankit\.com))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-f0-9]{128})"|'([a-f0-9]{128})'|([a-f0-9]{128}))|(?:"[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-f0-9]{128})"|'([a-f0-9]{128})'|([a-f0-9]{128}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll]"|'[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll]'|[Ll][Ee][Aa][Nn][Kk][Ii][Tt]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.leankit\.com)"|'(?:https://[A-Za-z0-9-]+\.leankit\.com)'|(?:https://[A-Za-z0-9-]+\.leankit\.com)))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"LEANKIT_API_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/planviewleankit",
		Description:     "Complete named LEANKIT_URL and LEANKIT_API_TOKEN credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "plivo-auth-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd]"|'[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd]'|[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Z]{20})"|'(?:[A-Z]{20})'|(?:[A-Z]{20}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{40})"|'([A-Za-z0-9_-]{40})'|([A-Za-z0-9_-]{40}))|(?:"[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{40})"|'([A-Za-z0-9_-]{40})'|([A-Za-z0-9_-]{40}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd]"|'[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd]'|[Pp][Ll][Ii][Vv][Oo]_[Aa][Uu][Tt][Hh]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:[A-Z]{20})"|'(?:[A-Z]{20})'|(?:[A-Z]{20})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PLIVO_AUTH_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/plivo",
		Description:     "Complete named PLIVO_AUTH_ID and PLIVO_AUTH_TOKEN credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "poloniex-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3})"|'(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3})'|(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([a-f0-9]{128})"|'([a-f0-9]{128})'|([a-f0-9]{128}))|(?:"[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([a-f0-9]{128})"|'([a-f0-9]{128})'|([a-f0-9]{128}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Oo][Ll][Oo][Nn][Ii][Ee][Xx]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3})"|'(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3})'|(?:[A-Z0-9]{8}(?:-[A-Z0-9]{8}){3})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"POLONIEX_API_SECRET"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/poloniex",
		Description:     "Complete named POLONIEX_API_KEY and POLONIEX_API_SECRET credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "railway-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Tt][Oo][Kk][Ee][Nn]"|'[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Tt][Oo][Kk][Ee][Nn]'|[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})'|([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}))|(?:"[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Rr][Aa][Ii][Ll][Ww][Aa][Yy]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})"|'([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})'|([a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"RAILWAY_TOKEN", "RAILWAY_API_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/railwayapp",
		Description:     "Exact in-value RAILWAY_TOKEN secret assignment. Pinned scanner candidate width/alphabet retained; bare opaque values and public identifiers are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "razorpay-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd]"|'[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd]'|[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:rzp_(?:live|test)_[A-Za-z0-9]{14})"|'(?:rzp_(?:live|test)_[A-Za-z0-9]{14})'|(?:rzp_(?:live|test)_[A-Za-z0-9]{14}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{24})"|'([A-Za-z0-9]{24})'|([A-Za-z0-9]{24}))|(?:"[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt]'|[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{24})"|'([A-Za-z0-9]{24})'|([A-Za-z0-9]{24}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd]"|'[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd]'|[Rr][Aa][Zz][Oo][Rr][Pp][Aa][Yy]_[Kk][Ee][Yy]_[Ii][Dd])[ \t]*[:=][ \t]*(?:"(?:rzp_(?:live|test)_[A-Za-z0-9]{14})"|'(?:rzp_(?:live|test)_[A-Za-z0-9]{14})'|(?:rzp_(?:live|test)_[A-Za-z0-9]{14})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"RAZORPAY_KEY_SECRET"},
		Source:          "https://razorpay.com/docs/api/authentication/",
		Description:     "Complete named RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "repairshopr-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll]"|'[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll]'|[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.repairshopr\.com)"|'(?:https://[A-Za-z0-9-]+\.repairshopr\.com)'|(?:https://[A-Za-z0-9-]+\.repairshopr\.com))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9-]{51})"|'([A-Za-z0-9-]{51})'|([A-Za-z0-9-]{51}))|(?:"[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9-]{51})"|'([A-Za-z0-9-]{51})'|([A-Za-z0-9-]{51}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll]"|'[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll]'|[Rr][Ee][Pp][Aa][Ii][Rr][Ss][Hh][Oo][Pp][Rr]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.repairshopr\.com)"|'(?:https://[A-Za-z0-9-]+\.repairshopr\.com)'|(?:https://[A-Za-z0-9-]+\.repairshopr\.com)))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"REPAIRSHOPR_API_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/repairshopr",
		Description:     "Complete named REPAIRSHOPR_URL and REPAIRSHOPR_API_KEY credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "revampcrm-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]"|'[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]'|[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9.@-]{1,254})"|'(?:[A-Za-z0-9.@-]{1,254})'|(?:[A-Za-z0-9.@-]{1,254}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|'([A-Za-z0-9]{40})'|([A-Za-z0-9]{40}))|(?:"[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|'([A-Za-z0-9]{40})'|([A-Za-z0-9]{40}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]"|'[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]'|[Rr][Ee][Vv][Aa][Mm][Pp][Cc][Rr][Mm]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9.@-]{1,254})"|'(?:[A-Za-z0-9.@-]{1,254})'|(?:[A-Za-z0-9.@-]{1,254})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"REVAMPCRM_API_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/revampcrm",
		Description:     "Named RevampCRM username and API key pair, in either field order. The upstream 40-character key constraint is retained, but its arbitrary 25-30-character username length is not an issuer rule; usernames are locally bounded at 254 bytes and pairs at 514 intervening bytes.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "robinhood-crypto-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]"|'[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]'|[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=))|(?:"[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]"|'[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy]'|[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Pp][Rr][Ii][Vv][Aa][Tt][Ee]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Oo][Bb][Ii][Nn][Hh][Oo][Oo][Dd]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|(?:rh-api-[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"ROBINHOOD_PRIVATE_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/robinhoodcrypto",
		Description:     "Complete named ROBINHOOD_API_KEY and ROBINHOOD_PRIVATE_KEY credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Key32,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rownd-app-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy]"|'[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy]'|[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})"|'(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})'|(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt]'|[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([a-z0-9]{48})"|'([a-z0-9]{48})'|([a-z0-9]{48}))|(?:"[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt]"|'[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt]'|[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([a-z0-9]{48})"|'([a-z0-9]{48})'|([a-z0-9]{48}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy]"|'[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy]'|[Xx]-[Rr][Oo][Ww][Nn][Dd]-[Aa][Pp][Pp]-[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})"|'(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})'|(?:[a-z0-9]{8}(?:-[a-z0-9]{4}){3}-[a-z0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"X-Rownd-App-Secret"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/rownd",
		Description:     "Complete Rownd x-rownd-app-key and x-rownd-app-secret header pair in either order. The 18-digit application ID is a resource selector, not an additional authentication secret; the official request headers determine credential roles. TruffleHog v3.97.4 candidate body shapes and a local 514-byte pairing bound are used.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "runrunit-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Aa][Pp][Pp]-[Kk][Ee][Yy]"|'[Aa][Pp][Pp]-[Kk][Ee][Yy]'|[Aa][Pp][Pp]-[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-f0-9]{32})"|'(?:[a-f0-9]{32})'|(?:[a-f0-9]{32}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]"|'[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]'|[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{18,20})"|'([A-Za-z0-9]{18,20})'|([A-Za-z0-9]{18,20}))|(?:"[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]"|'[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn]'|[Uu][Ss][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{18,20})"|'([A-Za-z0-9]{18,20})'|([A-Za-z0-9]{18,20}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Aa][Pp][Pp]-[Kk][Ee][Yy]"|'[Aa][Pp][Pp]-[Kk][Ee][Yy]'|[Aa][Pp][Pp]-[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[a-f0-9]{32})"|'(?:[a-f0-9]{32})'|(?:[a-f0-9]{32})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"User-Token"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/runrunit",
		Description:     "Complete named App-Key and User-Token credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "salesmate-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll]"|'[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll]'|[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.salesmate\.io)"|'(?:https://[A-Za-z0-9-]+\.salesmate\.io)'|(?:https://[A-Za-z0-9-]+\.salesmate\.io))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))|(?:"[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Ss][Ee][Ss][Ss][Ii][Oo][Nn]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll]"|'[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll]'|[Ss][Aa][Ll][Ee][Ss][Mm][Aa][Tt][Ee]_[Uu][Rr][Ll])[ \t]*[:=][ \t]*(?:"(?:https://[A-Za-z0-9-]+\.salesmate\.io)"|'(?:https://[A-Za-z0-9-]+\.salesmate\.io)'|(?:https://[A-Za-z0-9-]+\.salesmate\.io)))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SALESMATE_SESSION_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/salesmate",
		Description:     "Complete named SALESMATE_URL and SALESMATE_SESSION_TOKEN credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "saucelabs-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]"|'[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]'|[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9_.-]{2,70})"|'(?:[A-Za-z0-9_.-]{2,70})'|(?:[A-Za-z0-9_.-]{2,70}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy]"|'[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy]'|[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))|(?:"[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy]"|'[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy]'|[Ss][Aa][Uu][Cc][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]"|'[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee]'|[Ss][Aa][Uu][Cc][Ee]_[Uu][Ss][Ee][Rr][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9_.-]{2,70})"|'(?:[A-Za-z0-9_.-]{2,70})'|(?:[A-Za-z0-9_.-]{2,70})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SAUCE_ACCESS_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/saucelabs",
		Description:     "Complete named SAUCE_USERNAME and SAUCE_ACCESS_KEY credential pair in one value, in either field order. Companion distance is locally bounded to 514 bytes; body constraints come from TruffleHog v3.97.4 rather than an issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "scaleway-secret-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Cc][Ww]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Kk][Ee][Yy]"|'[Ss][Cc][Ww]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Kk][Ee][Yy]'|[Ss][Cc][Ww]_[Ss][Ee][Cc][Rr][Ee][Tt]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SCW_SECRET_KEY"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/scalewaykey",
		Description:     "Exact in-value SCW_SECRET_KEY secret assignment. Pinned scanner candidate width/alphabet retained; bare opaque values and public identifiers are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "scalr-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee]"|'[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee]'|[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9-]+\.scalr\.io)"|'(?:[A-Za-z0-9-]+\.scalr\.io)'|(?:[A-Za-z0-9-]+\.scalr\.io))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9._]{136})"|'([A-Za-z0-9._]{136})'|([A-Za-z0-9._]{136}))|(?:"[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn]"|'[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn]'|[Ss][Cc][Aa][Ll][Rr]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9._]{136})"|'([A-Za-z0-9._]{136})'|([A-Za-z0-9._]{136}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee]"|'[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee]'|[Ss][Cc][Aa][Ll][Rr]_[Hh][Oo][Ss][Tt][Nn][Aa][Mm][Ee])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9-]+\.scalr\.io)"|'(?:[A-Za-z0-9-]+\.scalr\.io)'|(?:[A-Za-z0-9-]+\.scalr\.io)))(?:$|[\s\x22\x27\x60,;}&\]])` + `|` + `(?:^|[^A-Za-z0-9_.-])(?i:scalr_(?:api_|access_)?token)["']?[ \t]*[:=][ \t]*["']?(eyJ[A-Za-z0-9_-]{10,1000}\.[A-Za-z0-9_-]{20,1000}\.[A-Za-z0-9_-]{43})` + catalogRightBoundary,
		Keywords:        []string{"scalr"},
		Source:          "https://docs.scalr.io/reference/api-tokens",
		Description:     "Preserves the named SCALR_HOSTNAME/136-character SCALR_TOKEN pair in either order and its original 514-byte companion bound. Additionally accepts an explicit SCALR_TOKEN, SCALR_API_TOKEN or SCALR_ACCESS_TOKEN assignment containing a structurally valid HS256 JWT with iss=user and at- jti. The legacy issuer marker and segment bounds are scanner constraints, not issuer guarantees; bare JWTs and provider proximity remain excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateBetterStructuredScalr4,
	},
	{
		ID:              "vonage-api-credentials",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9_-]{8})"|'(?:[A-Za-z0-9_-]{8})'|(?:[A-Za-z0-9_-]{8}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]'|(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{16})"|'([A-Za-z0-9_-]{16})'|([A-Za-z0-9_-]{16}))|(?:"(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]"|'(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt]'|(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{16})"|'([A-Za-z0-9_-]{16})'|([A-Za-z0-9_-]{16}))(?:[ \t\r\n,;]|[ \t\r\n,;][\s\S]{0,512}?[ \t\r\n,;])(?:"(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|(?:[Vv][Oo][Nn][Aa][Gg][Ee]|[Nn][Ee][Xx][Mm][Oo])_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"(?:[A-Za-z0-9_-]{8})"|'(?:[A-Za-z0-9_-]{8})'|(?:[A-Za-z0-9_-]{8})))(?:$|[\s\x22\x27\x60,;}&\]])|\bhttps://rest\.nexmo\.com/account/get-balance\?(?:api_key=[A-Za-z0-9_-]{8}&api_secret=([A-Za-z0-9_-]{16})|api_secret=([A-Za-z0-9_-]{16})&api_key=[A-Za-z0-9_-]{8})(?:$|[\s"\x27&])`,
		Keywords:        []string{"VONAGE_API_SECRET", "NEXMO_API_SECRET", "rest.nexmo.com"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/nexmoapikey",
		Description:     "Complete named Nexmo/Vonage API key and secret pair in either order, or the official rest.nexmo.com account-balance query carrier. The 8/16-character constraints follow TruffleHog v3.97.4; config pairing is locally bounded to 514 bytes. Identifiers alone and loose provider proximity are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateCarrier3VonageAssignment,
	},
	{
		ID:              "okta-api-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn][ \t]*:[ \t]*[Ss][Ss][Ww][Ss][ \t]+(00[A-Za-z0-9_-]{40})|(?:"[Oo][Kk][Tt][Aa]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]"|'[Oo][Kk][Tt][Aa]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn]'|[Oo][Kk][Tt][Aa]_[Aa][Pp][Ii]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(00[A-Za-z0-9_-]{40})"|'(00[A-Za-z0-9_-]{40})'|(00[A-Za-z0-9_-]{40})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"SSWS", "OKTA_API_TOKEN"},
		Source:          "https://developer.okta.com/docs/guides/create-an-api-token/main/",
		Description:     "Okta SSWS authorization header or exact OKTA_API_TOKEN assignment. The 00 plus 40 URL-safe character body is a pinned Titus/TruffleHog candidate constraint, not issuance proof; tenant domains and bare 00 values do not establish secrecy.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "opsgenie-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn][ \t]*:[ \t]*[Gg][Ee][Nn][Ii][Ee][Kk][Ee][Yy][ \t]+([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})|(?:"[Oo][Pp][Ss][Gg][Ee][Nn][Ii][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Oo][Pp][Ss][Gg][Ee][Nn][Ii][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Oo][Pp][Ss][Gg][Ee][Nn][Ii][Ee]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"GenieKey", "OPSGENIE_API_KEY"},
		Source:          "https://docs.opsgenie.com/docs/authentication",
		Description:     "Opsgenie GenieKey authorization header or exact OPSGENIE_API_KEY assignment. UUID shape alone is not confidential; the provider-specific carrier is required.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pagerduty-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn][ \t]*:[ \t]*[Tt][Oo][Kk][Ee][Nn][ \t]+[Tt][Oo][Kk][Ee][Nn][ \t]*=[ \t]*((?:u\+[A-Za-z0-9_+-]{18}|[A-Za-z0-9_-]{20}|[a-fA-F0-9]{32}))|(?:"[Pp][Aa][Gg][Ee][Rr][Dd][Uu][Tt][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Pp][Aa][Gg][Ee][Rr][Dd][Uu][Tt][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Pp][Aa][Gg][Ee][Rr][Dd][Uu][Tt][Yy]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"((?:u\+[A-Za-z0-9_+-]{18}|[A-Za-z0-9_-]{20}|[a-fA-F0-9]{32}))"|'((?:u\+[A-Za-z0-9_+-]{18}|[A-Za-z0-9_-]{20}|[a-fA-F0-9]{32}))'|((?:u\+[A-Za-z0-9_+-]{18}|[A-Za-z0-9_-]{20}|[a-fA-F0-9]{32}))))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{tokenField, "PAGERDUTY_API_KEY"},
		Source:          "https://developer.pagerduty.com/docs/authentication",
		Description:     "PagerDuty Token token= authorization or exact PAGERDUTY_API_KEY assignment. Supports Titus legacy 20-character keys and 32-hex integration keys only in explicit PagerDuty fields; unrelated UUIDs, routing IDs and u+ text are not standalone secrets.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "postmark-server-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:[Xx]-[Pp][Oo][Ss][Tt][Mm][Aa][Rr][Kk]-[Ss][Ee][Rr][Vv][Ee][Rr]-[Tt][Oo][Kk][Ee][Nn][ \t]*:[ \t]*([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})|(?:"[Pp][Oo][Ss][Tt][Mm][Aa][Rr][Kk]_[Ss][Ee][Rr][Vv][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Oo][Ss][Tt][Mm][Aa][Rr][Kk]_[Ss][Ee][Rr][Vv][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Oo][Ss][Tt][Mm][Aa][Rr][Kk]_[Ss][Ee][Rr][Vv][Ee][Rr]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})"|'([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})'|([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"X-Postmark-Server-Token", "POSTMARK_SERVER_TOKEN"},
		Source:          "https://postmarkapp.com/developer/api/overview",
		Description:     "Exact Postmark server-token header or named assignment. The explicit server-token carrier distinguishes this confidential UUID-like value from public identifiers; no server validation is performed.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pushbullet-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Uu][Ss][Hh][Bb][Uu][Ll][Ll][Ee][Tt]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Uu][Ss][Hh][Bb][Uu][Ll][Ll][Ee][Tt]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Uu][Ss][Hh][Bb][Uu][Ll][Ll][Ee][Tt]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_.]{34})"|'([A-Za-z0-9_.]{34})'|([A-Za-z0-9_.]{34})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PUSHBULLET_ACCESS_TOKEN"},
		Source:          "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/pushbulletapikey",
		Description:     "Exact in-value PUSHBULLET_ACCESS_TOKEN secret assignment. Pinned scanner candidate width/alphabet retained; bare opaque values and public identifiers are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "rapidapi-api-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:[Xx]-[Rr][Aa][Pp][Ii][Dd][Aa][Pp][Ii]-[Kk][Ee][Yy][ \t]*:[ \t]*([A-Za-z0-9_-]{50})|(?:"[Rr][Aa][Pp][Ii][Dd][Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Rr][Aa][Pp][Ii][Dd][Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Rr][Aa][Pp][Ii][Dd][Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9_-]{50})"|'([A-Za-z0-9_-]{50})'|([A-Za-z0-9_-]{50})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"X-RapidAPI-Key", "RAPIDAPI_KEY"},
		Source:          "https://docs.rapidapi.com/docs/keys",
		Description:     "Complete X-RapidAPI-Key header or exact RAPIDAPI_KEY assignment. The 50-character URL-safe body is the pinned scanner constraint; bare opaque values are excluded.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "pdflayer-access-key",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://api\.pdflayer\.com/api/convert\?[^\s"<>\x60]+)`,
		Keywords:        []string{"api.pdflayer.com"},
		Source:          "https://pdflayer.com/documentation",
		Description:     "Complete PdfLayer conversion URL with an access_key query credential. The exact provider host/path and a framed 32-character candidate are required; query components are parsed rather than found inside other parameter values. Candidate length is locally bounded to 16 KiB.",
		ValidateContext: validateCarrier3PDFLayer,
	},
	{
		ID:              "particle-access-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Aa][Rr][Tt][Ii][Cc][Ll][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]"|'[Pp][Aa][Rr][Tt][Ii][Cc][Ll][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn]'|[Pp][Aa][Rr][Tt][Ii][Cc][Ll][Ee]_[Aa][Cc][Cc][Ee][Ss][Ss]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9]{40})"|'([A-Za-z0-9]{40})'|([A-Za-z0-9]{40}))|https://api\.particle\.io/v1(?:/[^\s"<>\x60]*)?[ \t\r\n"\x27\\]{0,1000}access_token=([A-Za-z0-9]{40})|access_token=([A-Za-z0-9]{40})[ \t\r\n"\x27\\]{1,1000}https://api\.particle\.io/v1(?:/[^\s"<>\x60]*)?)(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"particle"},
		Source:          "https://docs.particle.io/reference/cloud-apis/api/",
		Description:     "Particle access_token request fragment with the exact api.particle.io/v1 endpoint in either order, or exact PARTICLE_ACCESS_TOKEN assignment. Forty alphanumeric characters are the pinned Titus scanner constraint; Authorization Bearer requests are independently recognized by the HTTP bearer carrier.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateCarrier3ParticleAssignment,
	},
	{
		ID:          "thingsboard-device-token",
		Regex:       `\b((?:[Hh][Tt][Tt][Pp][Ss]?://|[Cc][Oo][Aa][Pp][Ss]?://)(?:[A-Za-z0-9-]+\.)?thingsboard\.cloud/api/v1/[a-z0-9]{20}(?:/[^\s"<>\x60]*)?)(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:    []string{"thingsboard.cloud"},
		Source:      "https://thingsboard.io/docs/paas/reference/http-api/",
		Description: "ThingsBoard Cloud HTTP/CoAP device access credential in its API path. The 20-character lowercase-alphanumeric token is a Titus candidate constraint; exact provider host and a complete token segment are required. Resource existence or request success is not inferred.",
	},
	{
		ID:              "thingsboard-provisioning-secret",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Rr][Oo][Vv][Ii][Ss][Ii][Oo][Nn][Dd][Ee][Vv][Ii][Cc][Ee][Ss][Ee][Cc][Rr][Ee][Tt]"|'[Pp][Rr][Oo][Vv][Ii][Ss][Ii][Oo][Nn][Dd][Ee][Vv][Ii][Cc][Ee][Ss][Ee][Cc][Rr][Ee][Tt]'|[Pp][Rr][Oo][Vv][Ii][Ss][Ii][Oo][Nn][Dd][Ee][Vv][Ii][Cc][Ee][Ss][Ee][Cc][Rr][Ee][Tt])[ \t]*[:=][ \t]*(?:"([a-z0-9]{20})"|'([a-z0-9]{20})'|([a-z0-9]{20})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"provisionDeviceSecret"},
		Source:          "https://thingsboard.io/docs/paas/user-guide/device-provisioning/",
		Description:     "Literal provisionDeviceSecret provisioning field inside the scanned value. The separate provisionDeviceKey identifier alone is not confidential; the 20-character candidate grammar follows Titus rather than a provider issuance guarantee.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "tines-webhook-url",
		Regex:       `\b([Hh][Tt][Tt][Pp][Ss]://[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.tines\.com/webhook/[a-z0-9]{32}/[a-z0-9]{32}/?(?:\?[^\s"<>\x60]*)?)(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:    []string{"tines.com/webhook/"},
		Source:      "https://github.com/trufflesecurity/trufflehog/tree/v3.97.4/pkg/detectors/tineswebhook",
		Description: "Complete tenant-scoped Tines HTTPS webhook capability. The two 32-character path segments follow the pinned TruffleHog v3.97.4 signature, not issuer validation; unrelated URLs and incomplete or overlong path secrets are excluded.",
	},
	{
		ID:          "zapier-webhook-url",
		Regex:       `\b([Hh][Tt][Tt][Pp][Ss]://hooks\.zapier\.com/hooks/catch/[0-9]{5,10}/[a-z0-9]{5,10}/?(?:\?[^\s"<>\x60]*)?)(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:    []string{"hooks.zapier.com/hooks/catch/"},
		Source:      "https://help.zapier.com/hc/en-us/articles/8496288690317-Trigger-Zaps-from-webhooks",
		Description: "Complete HTTPS Zapier catch-hook capability URL. The 5-10 digit account and 5-10 lowercase-alphanumeric hook bounds are Titus candidate constraints, not full issuer grammar; exact host/path and whole-segment boundaries are required.",
	},
	{
		ID:          "database-connection-string",
		Regex:       `\b([A-Za-z][A-Za-z0-9 _]{0,40}[ \t]*=[ \t]*(?:\{(?:[^}\r\n]|}})*\}|"(?:[^"\r\n]|"")*"|\x27(?:[^\x27\r\n]|\x27\x27)*\x27|[^;\r\n"\x27{}]*)(?:;[ \t]*[A-Za-z][A-Za-z0-9 _]{0,40}[ \t]*=[ \t]*(?:\{(?:[^}\r\n]|}})*\}|"(?:[^"\r\n]|"")*"|\x27(?:[^\x27\r\n]|\x27\x27)*\x27|[^;\r\n"\x27{}]*))+;?)`,
		Keywords:    []string{";"},
		Source:      "https://learn.microsoft.com/en-us/dotnet/api/system.data.odbc.odbcconnection.connectionstring",
		Description: "ODBC/ADO-style semicolon connection string containing a literal Password/Pwd and either a user field or a database endpoint field. Braced values with doubled closing braces and quoted values with doubled quotes are parsed in field order; public connection metadata, empty passwords and references are excluded. Complete candidate and password parsing are bounded to 16 KiB.",
		Validate:    validCarrier3DatabaseString,
	},
	{
		ID:          "postgres-keyword-credentials",
		Regex:       `\b(?:pg_connect|pg_pconnect|psycopg2\.connect)[ \t]*\([ \t]*(?:"((?:[^"\\\r\n]|\\.)*)"|\x27((?:[^\x27\\\r\n]|\\.)*)\x27)`,
		Keywords:    []string{"pg_connect", "pg_pconnect", "psycopg2.connect"},
		Source:      "https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING",
		Description: "Literal PostgreSQL keyword DSN passed to pg_connect, pg_pconnect or psycopg2.connect. Libpq whitespace, equals, single-quoted values and backslash escapes are parsed; a nonempty password suffices because libpq supplies missing connection defaults. Unsupported MySQL positional-string APIs, variables and passwordless DSNs are not inferred; parsing is bounded to 16 KiB.",
		Validate:    validCarrier3PostgresDSN,
	},
	{
		ID:          "azure-redis-connection-string",
		Regex:       `\b([A-Za-z0-9](?:[A-Za-z0-9.-]{0,98}[A-Za-z0-9])?\.redis\.cache\.windows\.net:[0-9]+(?:,[A-Za-z][A-Za-z0-9]*=[^,\s"\x27]+)+)(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:    []string{"redis.cache.windows.net"},
		Source:      "https://learn.microsoft.com/en-us/azure/azure-cache-for-redis/cache-configure",
		Description: "Azure Redis comma-delimited connection string with explicit password and exact redis.cache.windows.net endpoint. Field order is not fixed, and the provider access key is confidential without relying on optional ssl or abortConnect defaults. The 44-character candidate width follows TruffleHog v3.97.4; candidate parsing is bounded to 16 KiB.",
		Validate:    validCarrier3AzureRedis,
	},
	{
		ID:          "redis-client-credentials",
		Regex:       `\b(redis\.(?:client\.Redis|connection\.(?:Connection|SSLConnection|ConnectionPool))\([^\r\n]{1,1000}\))`,
		Keywords:    []string{"redis.client", "redis.connection"},
		Source:      "https://raw.githubusercontent.com/redis/redis-py/3.5.3/redis/connection.py",
		Description: "Explicit redis-py client/connection constructor or call-style representation containing host and a literal password keyword in one value. Source expressions and None are excluded; this does not assume standard repr includes passwords. Fields are parsed in either order with quoted-string boundaries and a local 1000-character call bound.",
		Validate:    validCarrier3RedisCall,
	},
	{
		ID:              "http-userinfo-credential",
		Regex:           `\b([Hh][Tt][Tt][Pp][Ss]?://[^\s"<>\x60]+@[^\s"<>\x60]+)`,
		Keywords:        []string{"@"},
		Source:          "https://www.rfc-editor.org/rfc/rfc3986#section-3.2.1",
		Description:     "HTTP(S) URI with nonempty username and password in parsed userinfo. Percent escapes are decoded only after URI framing; public user-only URLs, empty passwords and references are excluded. Scheme folding is ASCII-only and complete candidate parsing is bounded to 16 KiB; no DNS or authentication checks occur.",
		ValidateContext: validateCarrier3HTTPUserinfo,
	},
	{
		ID:          "windows-command-credential",
		Regex:       `(?:^|[^A-Za-z0-9_.-])((?:[Cc][Oo][Nn][Nn][Ee][Cc][Tt]-[Vv][Ii][Ss][Ee][Rr][Vv][Ee][Rr]|[Pp][Ss][Ee][Xx][Ee][Cc]|[Pp][Ss][Ee][Xx][Ee][Cc]\.[Ee][Xx][Ee]|[Pp][Ss][Ee][Xx][Ee][Cc]64\.[Ee][Xx][Ee]|[Ss][Cc][Hh][Tt][Aa][Ss][Kk][Ss]|[Ss][Cc][Hh][Tt][Aa][Ss][Kk][Ss]\.[Ee][Xx][Ee]|[Nn][Ee][Tt] [Uu][Ss][Ee]|[Nn][Ee][Tt]\.[Ee][Xx][Ee] [Uu][Ss][Ee]|[Cc][Mm][Dd][Kk][Ee][Yy]|[Cc][Mm][Dd][Kk][Ee][Yy]\.[Ee][Xx][Ee])[ \t]+[^\r\n]{1,1000})(?:\r?\n|$)`,
		Keywords:    []string{"Connect-VIServer", "psexec", "psexec.exe", "psexec64.exe", "schtasks", "schtasks.exe", "net use", "net.exe use", cmdkeyCommand, "cmdkey.exe"},
		Source:      "https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/cmdkey",
		Description: "Literal Windows credential command: PsExec -u/-p, schtasks /ru-/rp or /u-/p, net use /user with positional password, cmdkey /add-/generic with /user-/pass, or Connect-VIServer -User/-Password. Tokenization respects quoted arguments and flag order, rejects variable/prompt expressions and compound commands, and bounds each command to 1000 characters and 128 arguments.",
		Validate:    validCarrier3WindowsCommand,
	},
	{
		ID:          "powershell-plaintext-credential",
		Regex:       `(?:^|[^A-Za-z0-9_.-])([Cc][Oo][Nn][Vv][Ee][Rr][Tt][Tt][Oo]-[Ss][Ee][Cc][Uu][Rr][Ee][Ss][Tt][Rr][Ii][Nn][Gg][ \t]+[^\r\n]{1,1000}|\[(?:[Ss][Yy][Ss][Tt][Ee][Mm]\.)?[Nn][Ee][Tt]\.[Nn][Ee][Tt][Ww][Oo][Rr][Kk][Cc][Rr][Ee][Dd][Ee][Nn][Tt][Ii][Aa][Ll]\]::[Nn][Ee][Ww]\([^\r\n]{1,1000}\))(?:\r?\n|$)`,
		Keywords:    []string{"ConvertTo-SecureString", "NetworkCredential"},
		Source:      "https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.security/convertto-securestring",
		Description: "Explicit ConvertTo-SecureString -AsPlainText literal or NetworkCredential constructor with a username and password literal. PowerShell quote boundaries, doubled single quotes, flag order and nonliteral references are handled without evaluating code; candidates are bounded to 1000 characters.",
		Validate:    validCarrier3PowerShell,
	},
	{
		ID:          "snmp-rw-community",
		Regex:       `(?m)^[ \t]*[Ss][Nn][Mm][Pp]-[Ss][Ee][Rr][Vv][Ee][Rr][ \t]+[Cc][Oo][Mm][Mm][Uu][Nn][Ii][Tt][Yy][ \t]+([^\s]{1,64})[ \t]+[Rr][Ww](?:[ \t]+[A-Za-z0-9_-]+)?[ \t]*\r?$`,
		Keywords:    []string{"snmp-server"},
		Source:      "https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/snmp/configuration/15-mt/snmp-15-mt-book/nm-snmp-cfg-snmp-support.html",
		Description: "Cisco SNMP read-write community configuration. A complete line with community and RW roles is required; public/default community examples and RO configuration are excluded. Community candidates retain the Titus 64-character upper bound.",
		Validate:    validCarrier3SNMP,
	},
	{
		ID:              "sql-created-user-password",
		Regex:           `(?:^|[^A-Za-z0-9_.-])[Cc][Rr][Ee][Aa][Tt][Ee][ \t]+(?:[Uu][Ss][Ee][Rr]|[Ll][Oo][Gg][Ii][Nn])[ \t]+[^\s;]{1,200}[ \t]+(?:[Ii][Dd][Ee][Nn][Tt][Ii][Ff][Ii][Ee][Dd][ \t]+[Bb][Yy]|[Ww][Ii][Tt][Hh][ \t]+[Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd])[ \t]*=?[ \t]*(?:\x27((?:[^\x27\r\n]|\x27\x27){1,1000})\x27|"((?:[^"\r\n]|""){1,1000})")`,
		Keywords:        []string{"create"},
		Source:          "https://learn.microsoft.com/en-us/sql/t-sql/statements/create-login-transact-sql",
		Description:     "CREATE USER/LOGIN plaintext password literal in IDENTIFIED BY or WITH PASSWORD syntax. SQL doubled-quote escapes are retained while expressions, HASHED suffixes, PostgreSQL md5/SCRAM verifier literals and MySQL authentication hashes are excluded. Password width is locally bounded to 1000 characters.",
		ValidateContext: validateCarrier3SQLPassword,
	},
	{
		ID:          "windows-unattend-password",
		Regex:       `(<AdministratorPassword>[\s\S]*?</AdministratorPassword>|<AutoLogon>[\s\S]*?</AutoLogon>)`,
		Keywords:    []string{unattendAdministratorPassword, "AutoLogon"},
		Source:      "https://learn.microsoft.com/en-us/windows-hardware/customize/desktop/unattend/microsoft-windows-shell-setup-useraccounts-administratorpassword",
		Description: "Unattend AdministratorPassword/Value or AutoLogon/Password/Value XML carrier. The complete element path, unique scalar fields and PlainText semantics are parsed; hidden values require Base64 UTF-16LE with the matching Password/AdministratorPassword suffix. Empty values, references, malformed XML and unrelated Value elements are excluded; parsing is bounded to 16 KiB.",
		Validate:    validCarrier3Unattend,
	},
	{
		ID:              "wireguard-private-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Rr][Ii][Vv][Aa][Tt][Ee][Kk][Ee][Yy]"|'[Pp][Rr][Ii][Vv][Aa][Tt][Ee][Kk][Ee][Yy]'|[Pp][Rr][Ii][Vv][Aa][Tt][Ee][Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=)))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PrivateKey"},
		Source:          "https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8",
		Description:     "WireGuard PrivateKey literal assignment with canonical padded Base64 encoding of exactly 32 bytes. The identically shaped PublicKey field is not confidential; key derivation, clamping and issuance are not inferred.",
		Validate:        validCarrier3Key32,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "wireguard-preshared-key",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Pp][Rr][Ee][Ss][Hh][Aa][Rr][Ee][Dd][Kk][Ee][Yy]"|'[Pp][Rr][Ee][Ss][Hh][Aa][Rr][Ee][Dd][Kk][Ee][Yy]'|[Pp][Rr][Ee][Ss][Hh][Aa][Rr][Ee][Dd][Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([A-Za-z0-9+/]{43}=)"|'([A-Za-z0-9+/]{43}=)'|([A-Za-z0-9+/]{43}=)))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{"PresharedKey"},
		Source:          "https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8",
		Description:     "WireGuard PresharedKey literal assignment with canonical padded Base64 encoding of exactly 32 bytes. A public key has the same encoding but not this confidential role; no entropy or generator-only bit requirements are imposed.",
		Validate:        validCarrier3Key32,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "phpmailer-smtp-credentials",
		Regex:       `(\$[A-Za-z_][A-Za-z0-9_]*->(?:Host|Username|Password)[ \t]*=[ \t]*(?:\x27(?:[^\x27\\\r\n]|\\.)*\x27|"(?:[^"\\\r\n]|\\.)*")[ \t]*;(?:[\s\S]{0,500}?\$[A-Za-z_][A-Za-z0-9_]*->(?:Host|Username|Password)[ \t]*=[ \t]*(?:\x27(?:[^\x27\\\r\n]|\\.)*\x27|"(?:[^"\\\r\n]|\\.)*")[ \t]*;){2})`,
		Keywords:    []string{"->Host", "->Username", "->Password"},
		Source:      "https://github.com/PHPMailer/PHPMailer/blob/v6.10.0/examples/smtp.phps",
		Description: "PHPMailer-style SMTP Host, Username and Password literal assignments on the same PHP object in one value. Property ordering and intervening lines are supported; mixed objects, interpolation, missing components and references are rejected. Each intervening span is bounded to 500 bytes and total parsing to 16 KiB.",
		Validate:    validCarrier3PHPMailer,
	},
	{
		ID:              "vault-legacy-service-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(s\.[A-Za-z0-9_-]{24,128})"|'(s\.[A-Za-z0-9_-]{24,128})'|(s\.[A-Za-z0-9_-]{24,128}))|(?:"[Vv][Aa][Uu][Ll][Tt]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Ss][Ee][Rr][Vv][Ii][Cc][Ee]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(s\.[A-Za-z0-9_-]{24,128})"|'(s\.[A-Za-z0-9_-]{24,128})'|(s\.[A-Za-z0-9_-]{24,128})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{vaultTokenEnvironment, "VAULT_SERVICE_TOKEN"},
		Source:          vaultFormatSource,
		Description:     "Legacy pre-1.10 Vault service token in an exact VAULT_TOKEN or VAULT_SERVICE_TOKEN assignment. The documented s. prefix is too ambiguous to scan alone; Titus candidate body bounds are retained without claiming issuance or decoding encrypted batch-token contents.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "vault-legacy-batch-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(b\.[A-Za-z0-9_-]{24,500})"|'(b\.[A-Za-z0-9_-]{24,500})'|(b\.[A-Za-z0-9_-]{24,500}))|(?:"[Vv][Aa][Uu][Ll][Tt]_[Bb][Aa][Tt][Cc][Hh]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Bb][Aa][Tt][Cc][Hh]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Bb][Aa][Tt][Cc][Hh]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(b\.[A-Za-z0-9_-]{24,500})"|'(b\.[A-Za-z0-9_-]{24,500})'|(b\.[A-Za-z0-9_-]{24,500})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{vaultTokenEnvironment, "VAULT_BATCH_TOKEN"},
		Source:          vaultFormatSource,
		Description:     "Legacy pre-1.10 Vault batch token in an exact VAULT_TOKEN or VAULT_BATCH_TOKEN assignment. The documented b. prefix is too ambiguous to scan alone; Titus candidate body bounds are retained without claiming issuance or decoding encrypted batch-token contents.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:              "vault-legacy-recovery-token",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(r\.[A-Za-z0-9_-]{24,500})"|'(r\.[A-Za-z0-9_-]{24,500})'|(r\.[A-Za-z0-9_-]{24,500}))|(?:"[Vv][Aa][Uu][Ll][Tt]_[Rr][Ee][Cc][Oo][Vv][Ee][Rr][Yy]_[Tt][Oo][Kk][Ee][Nn]"|'[Vv][Aa][Uu][Ll][Tt]_[Rr][Ee][Cc][Oo][Vv][Ee][Rr][Yy]_[Tt][Oo][Kk][Ee][Nn]'|[Vv][Aa][Uu][Ll][Tt]_[Rr][Ee][Cc][Oo][Vv][Ee][Rr][Yy]_[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(r\.[A-Za-z0-9_-]{24,500})"|'(r\.[A-Za-z0-9_-]{24,500})'|(r\.[A-Za-z0-9_-]{24,500})))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{vaultTokenEnvironment, "VAULT_RECOVERY_TOKEN"},
		Source:          vaultFormatSource,
		Description:     "Legacy pre-1.10 Vault recovery token in an exact VAULT_TOKEN or VAULT_RECOVERY_TOKEN assignment. The documented r. prefix is too ambiguous to scan alone; Titus candidate body bounds are retained without claiming issuance or decoding encrypted batch-token contents.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
	{
		ID:          "vault-unseal-key",
		Regex:       `(?:^|[^A-Za-z0-9_.-])(?:[Uu][Nn][Ss][Ee][Aa][Ll] [Kk][Ee][Yy][ \t]+[0-9]+[ \t]*:[ \t]*|[Vv][Aa][Uu][Ll][Tt][ \t]+[Oo][Pp][Ee][Rr][Aa][Tt][Oo][Rr][ \t]+[Uu][Nn][Ss][Ee][Aa][Ll][ \t]+)([A-Za-z0-9+/]{43}(?:[A-Za-z0-9+/]|=))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:    []string{"Unseal Key", "vault"},
		Source:      "https://raw.githubusercontent.com/hashicorp/vault/v1.9.0/shamir/shamir.go",
		Description: "Vault Unseal Key numbered output or vault operator unseal command. Canonical Base64 must decode to a 32-byte unsplit key or 33-byte Shamir share (32-byte AES key plus one share byte); generic unseal proximity and arbitrary Base64 are not recognized.",
		Validate:    validCarrier3Unseal,
	},
	{
		ID:              "truenas-api-key",
		Regex:           `(\{[^{}]*"method"[ \t\r\n]*:[ \t\r\n]*"auth\.login_with_api_key"[^{}]*\})|(?:^|[^A-Za-z0-9_.-])(?:"[Tt][Rr][Uu][Ee][Nn][Aa][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy]"|'[Tt][Rr][Uu][Ee][Nn][Aa][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy]'|[Tt][Rr][Uu][Ee][Nn][Aa][Ss]_[Aa][Pp][Ii]_[Kk][Ee][Yy])[ \t]*[:=][ \t]*(?:"([0-9]+-[A-Za-z0-9]{64})"|'([0-9]+-[A-Za-z0-9]{64})'|([0-9]+-[A-Za-z0-9]{64}))(?:$|[\s\x22\x27\x60,;}&\]])|^[Bb][Ee][Aa][Rr][Ee][Rr][ \t]+([0-9]+-[A-Za-z0-9]{64})$`,
		Keywords:        []string{"auth.login_with_api_key", "TRUENAS_API_KEY", "Bearer"},
		Source:          "https://raw.githubusercontent.com/truenas/middleware/TS-24.10.2/src/middlewared/middlewared/plugins/api_key.py",
		Description:     "TrueNAS numeric-ID plus 64-alphanumeric API key in a complete auth.login_with_api_key JSON request, exact TRUENAS_API_KEY assignment, or standalone Bearer credential value. JSON method and single string parameter are parsed; token structure follows the pinned TrueNAS generator, with a local 16 KiB bound. Full Authorization headers use the separate HTTP bearer rule.",
		Validate:        validCarrier3TrueNAS,
		ValidateContext: validateCarrier3TrueNASAssignment,
	},
	{
		ID:              "http-token-header-credential",
		Regex:           `(?:^|[^A-Za-z0-9_.-])(?:(?:"[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]"|'[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn]'|[Aa][Uu][Tt][Hh][Oo][Rr][Ii][Zz][Aa][Tt][Ii][Oo][Nn])[ \t]*[:=][ \t]*(?:"(?:[Tt][Oo][Kk][Ee][Nn][ \t]+(` + auditedAuthorizationTokenBody + `))"|'(?:[Tt][Oo][Kk][Ee][Nn][ \t]+(` + auditedAuthorizationTokenBody + `))'|(?:[Tt][Oo][Kk][Ee][Nn][ \t]+(` + auditedAuthorizationTokenBody + `)))|(?:"[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]"|'[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn]'|[Aa][Cc][Cc][Ee][Ss][Ss]-[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(?:([A-Za-z0-9_.]{34}))"|'(?:([A-Za-z0-9_.]{34}))'|(?:([A-Za-z0-9_.]{34})))|(?:"[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]"|'[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn]'|[Xx]-[Aa][Uu][Tt][Hh]-[Tt][Oo][Kk][Ee][Nn])[ \t]*[:=][ \t]*(?:"(?:([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))"|'(?:([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))'|(?:([a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}))))(?:$|[\s\x22\x27\x60,;}&\]])`,
		Keywords:        []string{authorizationHeader, "Access-Token", "X-Auth-Token"},
		Source:          "https://docs.pushbullet.com/#authentication",
		Description:     "Complete authentication-header carriers: documented Token 12/32/64-alphanumeric, 40-lowercase-alphanumeric and UUID subsets; Access-Token with 34 opaque characters; X-Auth-Token with a UUID. Names and schemes use ASCII case equivalence. Bare identifiers, variables, expression continuations and incomplete values are excluded; these supported scanner subsets do not establish issuer validity.",
		Validate:        validCarrier3Literal,
		ValidateContext: validateAuditedAssignmentContext,
	},
}

// These recognizers only inspect plaintext carriers. Bounds are local work limits,
// not service issuance promises. No verifier, metadata or recursive decoder runs.
func validCarrier3Literal(s string) bool {
	if s == "" || len(s) > maxStructuredCredentialBytes || strings.ContainsAny(s, "\x00\r\n") || strings.Trim(s, "*") == "" {
		return false
	}
	if strings.HasPrefix(s, "$") || strings.HasPrefix(s, "{{") || strings.HasPrefix(s, "<") ||
		strings.HasPrefix(s, "%") && strings.HasSuffix(s, "%") {
		return false
	}
	switch strings.ToLower(s) {
	case "*", nullLiteral, noneLiteral, undefinedLiteral, passwordField, passwdField, secretField, placeholderChangeMe, redactedLiteral, exampleLiteral, placeholderYourPassword, placeholderYourToken, placeholderYourAPIKey:
		return false
	}
	return true
}

func validCarrier3Key32(s string) bool {
	if len(s) != 44 {
		return false
	}
	var decoded [33]byte
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s))
	return err == nil && n == 32
}

func validCarrier3Unseal(s string) bool {
	var decoded [33]byte
	if len(s) != 44 {
		return false
	}
	n, err := base64.StdEncoding.Strict().Decode(decoded[:], []byte(s))
	return err == nil && (n == 32 || n == 33)
}

func validateCarrier3HTTPUserinfo(value string, start, end int, secret string) contextValidation {
	if start > 0 && value[start-1] == ':' {
		return contextValidation{}
	}
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User == nil || u.User.Username() == "" {
		return contextValidation{}
	}
	password, present := u.User.Password()
	return contextValidation{accepted: present && validCarrier3Literal(password)}
}

func validateCarrier3PDFLayer(value string, start, end int, secret string) contextValidation {
	s, ok := frameConnectionURI(value, start, end, secret)
	if !ok || len(s) > maxStructuredCredentialBytes {
		return contextValidation{}
	}
	u, err := url.Parse(s)
	if err != nil || u.Host != "api.pdflayer.com" || u.Path != "/api/convert" || u.User != nil {
		return contextValidation{}
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return contextValidation{}
	}
	for _, key := range query[accessKeyField] {
		if len(key) == 32 && carrier3ASCIIAlnum(key) && validCarrier3Literal(key) {
			return contextValidation{accepted: true}
		}
	}
	return contextValidation{}
}

func carrier3ASCIIAlnum(s string) bool {
	for i := range len(s) {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func validCarrier3DatabaseString(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	var password, user, endpoint bool
	fields := 0
	for len(s) != 0 {
		s = strings.TrimLeft(s, " \t;")
		if s == "" {
			break
		}
		key, rest, ok := strings.Cut(s, "=")
		if !ok {
			return false
		}
		key = strings.ToLower(strings.TrimSpace(key))
		rest = strings.TrimLeft(rest, " \t")
		var content string
		if rest != "" && (rest[0] == '{' || rest[0] == '\'' || rest[0] == '"') {
			closer := rest[0]
			if closer == '{' {
				closer = '}'
			}
			end := 1
			for ; end < len(rest); end++ {
				if rest[end] != closer {
					continue
				}
				if end+1 < len(rest) && rest[end+1] == closer {
					end++
					continue
				}
				break
			}
			if end == len(rest) {
				return false
			}
			content, s = rest[1:end], strings.TrimLeft(rest[end+1:], " \t")
			if s != "" && s[0] != ';' {
				return false
			}
		} else {
			content, s, _ = strings.Cut(rest, ";")
			content = strings.TrimSpace(content)
		}
		fields++
		switch key {
		case passwordField, "pwd":
			password = password || validCarrier3Literal(content)
		case userField, "user id", "userid", "uid":
			user = user || content != ""
		case serverField, "data source", "address", "addr", "network address", "dsn", "driver":
			endpoint = endpoint || content != ""
		}
	}
	return fields >= 2 && password && (user || endpoint)
}

func validCarrier3PostgresDSN(s string) bool {
	if len(s) > maxStructuredCredentialBytes || strings.ContainsRune(s, 0) {
		return false
	}
	password := false
	for s != "" {
		s = strings.TrimLeft(s, " \t\r\n")
		if s == "" {
			break
		}
		keyEnd := 0
		for keyEnd < len(s) && s[keyEnd] != '=' && !strings.ContainsRune(" \t\r\n", rune(s[keyEnd])) {
			keyEnd++
		}
		if keyEnd == 0 {
			return false
		}
		key := s[:keyEnd]
		s = strings.TrimLeft(s[keyEnd:], " \t\r\n")
		if s == "" || s[0] != '=' {
			return false
		}
		s = strings.TrimLeft(s[1:], " \t\r\n")
		quoted := len(s) != 0 && s[0] == '\''
		if quoted {
			s = s[1:]
		}
		var literal strings.Builder
		end := 0
		for end < len(s) {
			c := s[end]
			if c == '\\' {
				end++
				if end == len(s) {
					return false
				}
				literal.WriteByte(s[end])
				end++
				continue
			}
			if quoted && c == '\'' || !quoted && strings.ContainsRune(" \t\r\n", rune(c)) {
				break
			}
			literal.WriteByte(c)
			end++
		}
		if quoted {
			if end == len(s) {
				return false
			}
			end++
			if end < len(s) && !strings.ContainsRune(" \t\r\n", rune(s[end])) {
				return false
			}
		}
		if key == passwordField && validCarrier3Literal(literal.String()) {
			password = true
		}
		s = s[end:]
	}
	return password
}

func validCarrier3AzureRedis(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	_, fields, ok := strings.Cut(s, ",")
	if !ok {
		return false
	}
	for field := range strings.SplitSeq(fields, ",") {
		key, value, _ := strings.Cut(field, "=")
		if strings.EqualFold(key, passwordField) && len(value) == 44 && validCarrier3Literal(value) {
			return true
		}
	}
	return false
}

// The Redis form is a Python literal keyword call, not evaluation of Python.
func validCarrier3RedisCall(s string) bool {
	start, end := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if start < 0 || end <= start || len(s) > maxStructuredCredentialBytes {
		return false
	}
	body := s[start+1 : end]
	var host, password bool
	var quote byte
	var closers [32]byte
	stack := closers[:0]
	argumentStart := 0
	for i := 0; i <= len(body); i++ {
		if i == len(body) || body[i] == ',' && quote == 0 && len(stack) == 0 {
			if quote != 0 || len(stack) != 0 {
				return false
			}
			key, raw, assigned := strings.Cut(body[argumentStart:i], "=")
			if assigned {
				content, literal := carrier3RedisStringLiteral(strings.TrimSpace(raw))
				switch strings.TrimSpace(key) {
				case hostField:
					host = host || literal && content != ""
				case passwordField:
					password = password || literal && validCarrier3Literal(content)
				}
			}
			argumentStart = i + 1
			continue
		}
		c := body[i]
		if quote != 0 {
			switch c {
			case '\\':
				i++
				if i == len(body) {
					return false
				}
			case quote:
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case '(', '[', '{':
			if len(stack) == cap(stack) {
				return false
			}
			stack = append(stack, mapCarrier3RedisCloser(c))
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != c {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return host && password
}

func mapCarrier3RedisCloser(open byte) byte {
	switch open {
	case '(':
		return ')'
	case '[':
		return ']'
	default:
		return '}'
	}
}

func carrier3RedisStringLiteral(s string) (string, bool) {
	if len(s) < 2 || s[0] != '\'' && s[0] != '"' {
		return "", false
	}
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case s[0]:
			return s[1:i], i == len(s)-1
		}
	}
	return "", false
}

// This small command lexer is deliberately not a shell interpreter. It recognizes
// literal words, including quoted spaces and PowerShell doubled single quotes.
// It retains argument slices and rejects expressions/compound command operators.
func carrier3CommandWords(s string, words *[128]string) (int, bool) {
	n := 0
	for s != "" {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			break
		}
		if n == len(words) {
			return 0, false
		}
		end := 0
		var quote byte
	wordLoop:
		for end < len(s) {
			c := s[end]
			if quote != 0 {
				if c == quote {
					if quote == '\'' && end+1 < len(s) && s[end+1] == '\'' {
						end += 2
						continue
					}
					quote = 0
				}
			} else {
				switch c {
				case '\'', '"':
					quote = c
				case ' ', '\t':
					break wordLoop
				default:
					if strings.ContainsRune(";|&<>`\r\n", rune(c)) {
						return 0, false
					}
				}
			}
			end++
		}
		if quote != 0 || end == 0 {
			return 0, false
		}
		words[n] = s[:end]
		n++
		s = s[end:]
	}
	return n, true
}

func carrier3CommandLiteral(s string) bool {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		quote := s[0]
		s = s[1 : len(s)-1]
		if quote == '"' && strings.ContainsAny(s, "$`") {
			return false
		}
	}
	if strings.ContainsAny(s, "$`%") || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "/") {
		return false
	}
	return validCarrier3Literal(s)
}

func validCarrier3WindowsCommand(secret string) bool {
	secret = strings.TrimSuffix(strings.TrimSpace(secret), ";")
	var storage [128]string
	n, ok := carrier3CommandWords(secret, &storage)
	if !ok || n < 2 {
		return false
	}
	words := storage[:n]
	command := strings.TrimSuffix(strings.ToLower(words[0]), ".exe")
	var user, password, target bool
	positional := 0
	for i := 1; i < len(words); i++ {
		word := strings.ToLower(words[i])
		key, inline, hasInline := strings.Cut(word, ":")
		if hasInline {
			inline = words[i][len(key)+1:]
		}
		isUser, isPassword := false, false
		switch command {
		case "psexec", "psexec64":
			if !target && (strings.HasPrefix(words[i], `\\`) || strings.HasPrefix(words[i], "@")) {
				target = true
				continue
			}
			switch word {
			case "-u", "-p":
				isUser, isPassword = word == "-u", word == "-p"
			case "-a", "-n", "-r", "-w", "-g":
				i++
				if i >= len(words) {
					return false
				}
				continue
			case "-i":
				// The session number is optional; an executable is not.
				if next := i + 1; next < len(words) && carrier3DecimalWord(words[next]) {
					i = next
				}
				continue
			case "-c", "-f", "-v", "-d", "-e", "-h", "-l", "-s", "-x",
				"-low", "-belownormal", "-normal", "-abovenormal", "-high", "-realtime",
				"-background", "-arm", "-accepteula", "-nobanner":
				continue
			default:
				// PsExec's remaining words belong to the remote executable.
				return user && password
			}
		case "connect-viserver":
			isUser, isPassword = word == "-user", word == "-password"
			switch word {
			case "-server", "-port", "-protocol", "-credential", "-session", "-location":
				i++
				if i >= len(words) {
					return false
				}
				continue
			}
		case "schtasks":
			isUser = word == "/ru" || word == "/u"
			isPassword = word == "/rp" || word == "/p"
			if !isUser && !isPassword {
				switch word {
				case "/s", "/tn", "/tr", "/sc", "/mo", "/d", "/m", "/i", "/st", "/et",
					"/du", "/sd", "/ed", "/rl", "/xml", "/delay", "/fo":
					i++
					if i >= len(words) {
						return false
					}
				case "/create", "/change", "/run", "/end", "/delete", "/query", "/showsid",
					"/f", "/it", "/np", "/z", "/v", "/nh", "/hresult", "/enable", "/disable":
				default:
					return false
				}
				continue
			}
		case cmdkeyCommand:
			isUser, isPassword = key == "/user" && hasInline, key == "/pass" && hasInline
			if (key == "/add" || key == "/generic") && hasInline && inline != "" {
				target = true
			}
		case "net":
			if i == 1 {
				if word != "use" {
					return false
				}
				continue
			}
			isUser = key == "/user" && hasInline
			if !strings.HasPrefix(word, "/") {
				positional++
				if positional == 1 && len(words[i]) == 2 && words[i][1] == ':' {
					// An optional mapped drive precedes the UNC target.
					continue
				}
				if !target {
					target = strings.HasPrefix(words[i], `\\`)
				} else if carrier3CommandLiteral(words[i]) {
					password = true
				}
			}
		}
		if !isUser && !isPassword {
			continue
		}
		argument := inline
		if !hasInline {
			remaining := words[i+1:]
			if len(remaining) == 0 {
				return false
			}
			argument = remaining[0]
			i++
		}
		literal := carrier3CommandLiteral(argument)
		if isUser {
			user = user || literal
		} else {
			password = password || literal
		}
	}
	if command == cmdkeyCommand || command == "net" {
		return user && password && target
	}
	return user && password
}

func carrier3DecimalWord(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

var carrier3NetworkCredential = newLazyRegexp(`^\[(?:[Ss][Yy][Ss][Tt][Ee][Mm]\.)?[Nn][Ee][Tt]\.[Nn][Ee][Tt][Ww][Oo][Rr][Kk][Cc][Rr][Ee][Dd][Ee][Nn][Tt][Ii][Aa][Ll]\]::[Nn][Ee][Ww]\([ \t]*(?:"([^"$` + "`" + `]*)"|'((?:[^']|'')*)')[ \t]*,[ \t]*(?:"([^"$` + "`" + `]*)"|'((?:[^']|'')*)')[ \t]*(?:,[ \t]*(?:"[^"$` + "`" + `]*"|'(?:[^']|'')*')[ \t]*)?\)$`)

func validCarrier3PowerShell(s string) bool {
	s = strings.TrimSuffix(strings.TrimSpace(s), ";")
	if strings.HasPrefix(s, "[") {
		m := carrier3NetworkCredential.FindStringSubmatch(s)
		if m == nil {
			return false
		}
		user, password := m[1], m[3]
		if m[2] != "" {
			user = m[2]
		}
		if m[4] != "" {
			password = m[4]
		}
		return user != "" && validCarrier3Literal(password)
	}
	var storage [128]string
	n, ok := carrier3CommandWords(s, &storage)
	if !ok {
		return false
	}
	words := storage[:n]
	plaintext, literal, valueSet := false, false, false
	for i := 1; i < len(words); i++ {
		switch strings.ToLower(words[i]) {
		case "-asplaintext":
			plaintext = true
		case "-force":
		case "-string":
			remaining := words[i+1:]
			if len(remaining) == 0 || valueSet {
				return false
			}
			literal = carrier3CommandLiteral(remaining[0])
			i++
			valueSet = true
		default:
			if valueSet || strings.HasPrefix(words[i], "-") {
				return false
			}
			literal = carrier3CommandLiteral(words[i])
			valueSet = true
		}
	}
	return plaintext && literal && valueSet
}

func validCarrier3SNMP(s string) bool {
	return validCarrier3Literal(s) && !strings.EqualFold(s, "public") && !strings.EqualFold(s, "private")
}

func validateCarrier3SQLPassword(value string, _, end int, secret string) contextValidation {
	if !validCarrier3Literal(secret) {
		return contextValidation{}
	}
	suffix := strings.TrimLeft(value[end:], " \t")
	if len(suffix) >= 6 && strings.EqualFold(suffix[:6], "HASHED") {
		return contextValidation{}
	}
	lower := strings.ToLower(secret)
	if strings.HasPrefix(lower, "scram-sha-256$") || strings.HasPrefix(lower, "0x") ||
		len(lower) == 35 && strings.HasPrefix(lower, "md5") || len(lower) == 41 && lower[0] == '*' {
		return contextValidation{}
	}
	return contextValidation{accepted: true}
}

func validCarrier3Unattend(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	decoder := xml.NewDecoder(strings.NewReader(s))
	var stack [8]string
	depth, values, modes := 0, 0, 0
	var root, literal, mode string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false
		}
		switch t := token.(type) {
		case xml.StartElement:
			if depth == len(stack) || len(t.Attr) != 0 {
				return false
			}
			if depth == 0 {
				if root != "" {
					return false
				}
				root = t.Name.Local
			}
			stack[depth] = t.Name.Local
			depth++
			credentialDepth := root == unattendAdministratorPassword && depth == 2 || root == "AutoLogon" && depth == 3 && stack[1] == unattendPassword
			if credentialDepth && (t.Name.Local == "Value" || t.Name.Local == "PlainText") {
				text, ok := carrier3XMLScalar(decoder)
				if !ok {
					return false
				}
				depth--
				if t.Name.Local == "Value" {
					values++
					literal = text
				} else {
					modes++
					mode = strings.TrimSpace(text)
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return false
		}
	}
	if depth != 0 || values != 1 || modes > 1 || mode != "" && mode != "true" && mode != "false" {
		return false
	}
	if mode != "false" {
		return validCarrier3Literal(literal)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(literal)
	if err != nil || len(decoded)%2 != 0 {
		return false
	}
	units := make([]uint16, len(decoded)/2)
	for i := range units {
		units[i] = uint16(decoded[i*2]) | uint16(decoded[i*2+1])<<8
	}
	text := string(utf16.Decode(units))
	marker := unattendPassword
	if root == unattendAdministratorPassword {
		marker = unattendAdministratorPassword
	}
	password, present := strings.CutSuffix(text, marker)
	return present && !strings.ContainsRune(password, '\ufffd') && validCarrier3Literal(password)
}

var carrier3PHPMailAssignment = newLazyRegexp(`\$([A-Za-z_][A-Za-z0-9_]*)->(Host|Username|Password)[ \t]*=[ \t]*(?:'((?:[^'\\\r\n]|\\.)*)'|"((?:[^"\\\r\n]|\\.)*)")[ \t]*;`)

func validCarrier3PHPMailer(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	object := ""
	var host, user, password bool
	for scan := 0; scan < len(s); {
		m := carrier3PHPMailAssignment.FindStringSubmatchIndex(s[scan:])
		if m == nil {
			break
		}
		name := s[scan+m[2] : scan+m[3]]
		if object != "" && object != name {
			return false
		}
		object = name
		property := s[scan+m[4] : scan+m[5]]
		left, right := m[6], m[7]
		if left < 0 {
			left, right = m[8], m[9]
		}
		content := s[scan+left : scan+right]
		switch property {
		case "Host":
			host = host || content != ""
		case "Username":
			user = user || content != ""
		case unattendPassword:
			password = password || validCarrier3Literal(content) && (m[8] < 0 || !strings.ContainsRune(content, '$'))
		}
		scan += m[1]
	}
	return host && user && password
}

func validCarrier3TrueNAS(s string) bool {
	if len(s) > maxStructuredCredentialBytes {
		return false
	}
	if strings.HasPrefix(s, "{") {
		var request struct {
			Method string   `json:"method"`
			Params []string `json:"params"`
		}
		if err := json.Unmarshal([]byte(s), &request); err != nil || request.Method != "auth.login_with_api_key" || len(request.Params) != 1 {
			return false
		}
		s = request.Params[0]
	}
	id, secret, ok := strings.Cut(s, "-")
	if !ok || id == "" || len(secret) != 64 || !carrier3ASCIIAlnum(secret) {
		return false
	}
	for i := range len(id) {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	return true
}

func carrier3XMLScalar(decoder *xml.Decoder) (string, bool) {
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		switch t := token.(type) {
		case xml.CharData:
			text.Write([]byte(t))
		case xml.EndElement:
			return text.String(), true
		case xml.Comment:
		default:
			return "", false
		}
	}
}

func validateCarrier3VonageAssignment(value string, start, end int, secret string) contextValidation {
	if strings.HasPrefix(value[start:end], "https://rest.nexmo.com/account/get-balance?") {
		return contextValidation{accepted: true}
	}
	return validateAuditedAssignmentContext(value, start, end, secret)
}

func validateCarrier3ParticleAssignment(value string, start, end int, secret string) contextValidation {
	if indexFoldedASCII(value[start:end], "particle_access_token") >= 0 {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	return contextValidation{accepted: true}
}

func validateCarrier3TrueNASAssignment(value string, start, end int, secret string) contextValidation {
	if indexFoldedASCII(value[start:end], "truenas_api_key") >= 0 && !strings.HasPrefix(secret, "{") {
		return validateAuditedAssignmentContext(value, start, end, secret)
	}
	return contextValidation{accepted: true}
}
