package secrets

import (
	"encoding/base64"
	"hash/crc32"
	"strconv"
	"strings"
)

// Betterleaks candidates are pinned to 95237cf8eb4d. Opaque body widths are
// scanner bounds, not issuer guarantees; exact roles disambiguate public keys.
var betterleaksEvidenceRules1 = []catalogRuleSpec{
	{
		ID:              "amplitude-secret-key",
		Regex:           `\b(?i:AMPLITUDE_(?:PROJECT_)?SECRET_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Fa-f0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"amplitude"},
		SecretGroup:     1,
		ValidateContext: validateBetterleaksEvidenceAssignment1,
		Source:          "https://amplitude.com/docs/apis/keys-and-tokens",
		Description:     "Amplitude private project Secret Key in an exact same-value secret-key assignment. Public Analytics API keys and client-side deployment keys are excluded; 32 hex characters are a pinned scanner bound, not a published issuer grammar.",
	},
	{
		ID:              "circleci-project-api-token",
		Regex:           `\b(?i:CIRCLECI_PROJECT_(?:ADMIN|READ_ONLY)_(?:API_)?TOKEN)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([A-Fa-f0-9]{40})` + catalogRightBoundary,
		Keywords:        []string{"circleci"},
		SecretGroup:     1,
		ValidateContext: validateBetterleaksEvidenceAssignment1,
		Source:          "https://circleci.com/docs/guides/toolkit/managing-api-tokens/",
		Description:     "CircleCI project tokens explicitly identified as Admin or Read Only, which access project API data. Status tokens intended for published badges and unspecified project tokens are excluded. Forty hex characters are a legacy scanner bound, not a complete issuance grammar.",
	},
	{
		ID:              "minimax-api-key",
		Regex:           `\b(?i:MINIMAX_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(sk-api-[A-Za-z0-9_-]{119})` + catalogRightBoundary,
		Keywords:        []string{"minimax"},
		SecretGroup:     1,
		ValidateContext: validateBetterleaksEvidenceAssignment1,
		Source:          "https://platform.minimax.io/docs/guides/quickstart-preparation.md",
		Description:     "MiniMax documents MINIMAX_API_KEY for its secret API key. The sk-api- prefix and 119-character body remain scanner candidate constraints and require that exact carrier; no bare-prefix issuer attribution or API probe.",
	},
	{
		ID:              "privateai-api-token",
		Regex:           `\b(?i:PRIVATE[_-]?AI[_-]API[_-]KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?([a-z0-9]{32})` + catalogRightBoundary,
		Keywords:        []string{"privateai", "private_ai", "private-ai"},
		SecretGroup:     1,
		ValidateContext: validateBetterleaksEvidenceAssignment1,
		Source:          "https://docs.getlimina.ai/product-guides/thin-client.md",
		Description:     "Private AI (now Limina) cloud API authentication key in an exact provider API-key assignment. Provider mentions, bare identifiers and unauthenticated self-hosted container settings do not qualify. The 32-character lowercase-alphanumeric body is a pinned scanner subset.",
	},
	{
		ID:          "redirect-pizza-api-token",
		Regex:       `\b(rpa_[A-Za-z0-9]{30})` + catalogRightBoundary,
		Keywords:    []string{"rpa_"},
		SecretGroup: 1,
		Validate:    validBetterleaksRedirectToken1,
		Source:      "https://redirect.pizza/support/api",
		Description: "redirect.pizza documents rpa_ as its bearer API-token prefix for private redirect/team operations. The 30-character body is a scanner-supported shape, not a full issuance claim. RunPod shares the private prefix but its separate longer scanner candidate cannot be truncated into this rule.",
	},
	{
		ID:              "runpod-api-key",
		Regex:           `\b(?i:RUNPOD_API_KEY)["']?[ \t]{0,16}[:=][ \t]{0,16}["']?(rpa_[A-Z0-9]{40}[A-Za-z0-9]{6})` + catalogRightBoundary,
		Keywords:        []string{"runpod"},
		SecretGroup:     1,
		ValidateContext: validateBetterleaksEvidenceAssignment1,
		Source:          "https://docs.runpod.io/get-started/api-keys",
		Description:     "RunPod keys must be treated like passwords. Require an explicit RunPod API-key assignment for the scanner's shared rpa_ prefix and 40-uppercase-plus-6-alphanumeric body; no claim that the prefix uniquely identifies RunPod or that a synthetic candidate has been issued.",
	},
	{
		ID:          "settlemint-service-access-token",
		Regex:       `\b(sm_sat_[A-Za-z0-9]{16})` + catalogRightBoundary,
		Keywords:    []string{"sm_sat_"},
		SecretGroup: 1,
		Validate:    validBetterleaksSettleMintToken1,
		Source:      "https://raw.githubusercontent.com/settlemint/sdk/6232a2479f20d5801d4fcb40733a70aaced6c50a/sdk/utils/src/logging/mask-tokens.ts",
		Description: "SettleMint's official SDK explicitly masks sm_sat_ service-account tokens as sensitive credentials. Sixteen alphanumeric body characters are the pinned Betterleaks scanner subset; the SDK does not promise this exact issuance width.",
	},
}

// Combine the existing provider-role, assignment and literal checks.
func validateBetterleaksEvidenceAssignment1(value string, start, end int, secret string) contextValidation {
	if result := validateEvidence2Credential(value, start, end, secret); !result.accepted {
		return result
	}
	return contextValidation{accepted: validAuditedCarrierLiteral2(secret)}
}

func validBetterleaksRedirectToken1(secret string) bool {
	return len(secret) == 34 && validAuditedCarrierLiteral2(secret[4:])
}

func validBetterleaksSettleMintToken1(secret string) bool {
	return len(secret) == 23 && validAuditedCarrierLiteral2(secret[7:])
}

func validateBetterleaksCloudinaryCredential1(value string, start, end int, secret string) contextValidation {
	if !strings.Contains(secret, "://") {
		if result := validateBetterleaksEvidenceAssignment1(value, start, end, secret); !result.accepted {
			return result
		}
	} else if result := validateAuditedAssignmentContext(value, start, end, secret); !result.accepted {
		return result
	}
	return validateCloudinaryCredentialURI1(value, start, end, secret)
}

// v17.9-v17.11 used routing text, 16 random bytes, and a random-length byte,
// followed by one dot and length+CRC. Current v1 uses a different frame and must
// still pass the existing validator; malformed versioned tokens never fall back.
func validBetterleaksGitLabRoutable1(token string) bool {
	if strings.Contains(token, ".01.") {
		return validGitLabRoutableToken(token)
	}
	_, body, found := strings.Cut(token, "-")
	if !found {
		return false
	}
	if strings.HasPrefix(token, "glrt-t") {
		if len(body) < 3 || (body[1] != '2' && body[1] != '3') || body[2] != '_' {
			return false
		}
		body = body[3:]
	}
	encoded, footer, found := strings.Cut(body, ".")
	if !found || len(footer) != 9 || len(encoded) < 27 || len(encoded) > 235 {
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
	if err != nil || n < 20 || decoded[n-1] != 16 {
		return false
	}
	routing := decoded[:n-17]
	var previous byte
	hasOwner := false
	for len(routing) > 0 {
		lineEnd := 0
		for lineEnd < len(routing) && routing[lineEnd] != '\n' {
			lineEnd++
		}
		if lineEnd < 3 || routing[1] != ':' || routing[0] <= previous {
			return false
		}
		switch routing[0] {
		case 'o':
			hasOwner = true
		case 'c', 'g', 'p', 't', 'u':
		default:
			return false
		}
		previous = routing[0]
		if lineEnd == len(routing) {
			break
		}
		routing = routing[lineEnd+1:]
		if len(routing) == 0 {
			return false
		}
	}
	return hasOwner
}
