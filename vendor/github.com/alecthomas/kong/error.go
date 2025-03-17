package kong

// ParseError is the error type returned by Kong.Parse().
//
// It contains the parse Context that triggered the error.
type ParseError struct {
	error
	Context  *Context
	exitCode int
}

// Unwrap returns the original cause of the error.
func (p *ParseError) Unwrap() error { return p.error }

// ExitCode returns the status that Kong should exit with if it fails with a ParseError.
func (p *ParseError) ExitCode() int {
	if p.exitCode == 0 {
		return exitNotOk
	}
	return p.exitCode
}
