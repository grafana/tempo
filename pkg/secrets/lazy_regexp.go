package secrets

import (
	"regexp"
	"sync"
)

// lazyRegexp defers compilation of a trusted native validator until first use.
// The compiled expression is shared read-only across tenants. OnceValue also
// preserves a compilation panic on every call instead of publishing a nil regex.
type lazyRegexp struct {
	compiled func() *regexp.Regexp
}

func newLazyRegexp(pattern string) *lazyRegexp {
	return &lazyRegexp{compiled: sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(pattern)
	})}
}

func (r *lazyRegexp) MatchString(s string) bool {
	return r.compiled().MatchString(s)
}

func (r *lazyRegexp) FindStringSubmatch(s string) []string {
	return r.compiled().FindStringSubmatch(s)
}

func (r *lazyRegexp) FindStringSubmatchIndex(s string) []int {
	return r.compiled().FindStringSubmatchIndex(s)
}
