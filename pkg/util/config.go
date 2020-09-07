package util

func PrefixConfig(prefix string, option string) string {
	if len(prefix) > 0 {
		return prefix + "." + option
	}

	return option
}
