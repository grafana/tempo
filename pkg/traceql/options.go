package traceql

// CompileOption is a functional option for Parse, Compile, Engine.CompileMetricsQueryRange, and Engine.CompileMetricsQueryRangeNonRaw.
// Parse and Compile silently ignore options specific to metrics queries like WithSpanOnlyFetch and WithTimeOverlapCutoff.
type CompileOption func(*compileOptions)

type compileOptions struct {
	skipTransformations []string
	allowUnsafeHints    bool

	// metrics query only
	spanOnlyFetch     *bool
	timeOverlapCutoff float64
}

func applyCompileOptions(opts ...CompileOption) compileOptions {
	var cfg compileOptions
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithSkipOptimization adds name to the list of AST transformations to skip during parsing.
// Use [TransformationAll] to skip all transformations.
func WithSkipOptimization(name string) CompileOption {
	return func(o *compileOptions) {
		o.skipTransformations = append(o.skipTransformations, name)
	}
}

// WithUnsafeHints controls whether the unsafe query hint [HintSkipASTTransformations]
// is honored during parsing. This does not affect other unsafe hints, which are read
// after parsing by the caller.
func WithUnsafeHints(v bool) CompileOption {
	return func(o *compileOptions) {
		o.allowUnsafeHints = v
	}
}

// WithSpanOnlyFetch sets whether to use the span-only fetch path. When not set the default is used, and
// this may be overridden by the query hint.
func WithSpanOnlyFetch(v bool) CompileOption {
	return func(o *compileOptions) {
		o.spanOnlyFetch = &v
	}
}

// WithTimeOverlapCutoff sets the overlap threshold (0 to 1) for trace-level timestamp filtering. When not
// set the default value is used.
func WithTimeOverlapCutoff(v float64) CompileOption {
	return func(o *compileOptions) {
		o.timeOverlapCutoff = v
	}
}
