package overrides

func doesNotContain(list []string, item string) bool {
	for _, l := range list {
		if item == l {
			return false
		}
	}
	return true
}
