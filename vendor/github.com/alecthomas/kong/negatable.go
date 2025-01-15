package kong

// negatableDefault is a placeholder value for the Negatable tag to indicate
// the negated flag is --no-<flag-name>. This is needed as at the time of
// parsing a tag, the field's flag name is not yet known.
const negatableDefault = "_"

// negatableFlagName returns the name of the flag for a negatable field, or
// an empty string if the field is not negatable.
func negatableFlagName(name, negation string) string {
	switch negation {
	case "":
		return ""
	case negatableDefault:
		return "--no-" + name
	default:
		return "--" + negation
	}
}
