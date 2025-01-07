package kong

// BeforeResolve is a documentation-only interface describing hooks that run before resolvers are applied.
type BeforeResolve interface {
	// This is not the correct signature - see README for details.
	BeforeResolve(args ...any) error
}

// BeforeApply is a documentation-only interface describing hooks that run before values are set.
type BeforeApply interface {
	// This is not the correct signature - see README for details.
	BeforeApply(args ...any) error
}

// AfterApply is a documentation-only interface describing hooks that run after values are set.
type AfterApply interface {
	// This is not the correct signature - see README for details.
	AfterApply(args ...any) error
}

// AfterRun is a documentation-only interface describing hooks that run after Run() returns.
type AfterRun interface {
	// This is not the correct signature - see README for details.
	// AfterRun is called after Run() returns.
	AfterRun(args ...any) error
}
