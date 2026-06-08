package funk

type options struct {
	allowZero bool
}

type option func(*options)

func newOptions(values ...option) *options {
	opts := &options{
		allowZero: false,
	}
	for _, o := range values {
		o(opts)
	}
	return opts
}

// WithAllowZero allows zero values.
func WithAllowZero() func(*options) {
	return func(opts *options) {
		opts.allowZero = true
	}
}
