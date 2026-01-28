// Package formatter is what powers jsonnetfmt, a Jsonnet formatter.
// It works similar to most other code formatters. Basically said, it takes the
// contents of a file and returns them properly formatted. Behaviour can be
// customized using formatter.Options.
package formatter

import "github.com/google/go-jsonnet/internal/formatter"

// StringStyle controls how the reformatter rewrites string literals.
// Strings that contain a ' or a " use the optimal syntax to avoid escaping
// those characters.
type StringStyle = formatter.StringStyle

const (
	// StringStyleDouble means "this".
	StringStyleDouble StringStyle = iota
	// StringStyleSingle means 'this'.
	StringStyleSingle
	// StringStyleLeave means strings are left how they were found.
	StringStyleLeave
)

// CommentStyle controls how the reformatter rewrites comments.
// Comments that look like a #! hashbang are always left alone.
type CommentStyle = formatter.CommentStyle

const (
	// CommentStyleHash means #.
	CommentStyleHash CommentStyle = iota
	// CommentStyleSlash means //.
	CommentStyleSlash
	// CommentStyleLeave means comments are left as they are found.
	CommentStyleLeave
)

// Options is a set of parameters that control the reformatter's behaviour.
type Options = formatter.Options

// DefaultOptions returns the recommended formatter behaviour.
func DefaultOptions() Options {
	return formatter.DefaultOptions()
}

// Format returns code that is equivalent to its input but better formatted
// according to the given options.
func Format(filename string, input string, options Options) (string, error) {
	return formatter.Format(filename, input, options)
}
