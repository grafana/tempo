package participle

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

// Error represents an error while parsing.
//
// The format of an Error is in the form "[<filename>:][<line>:<pos>:] <message>".
//
// The error will contain positional information if available.
type Error interface {
	error
	// Unadorned message.
	Message() string
	// Closest position to error location.
	Position() lexer.Position
}

// FormatError formats an error in the form "[<filename>:][<line>:<pos>:] <message>"
func FormatError(err Error) string {
	msg := ""
	pos := err.Position()
	if pos.Filename != "" {
		msg += pos.Filename + ":"
	}
	if pos.Line != 0 || pos.Column != 0 {
		msg += fmt.Sprintf("%d:%d:", pos.Line, pos.Column)
	}
	if msg != "" {
		msg += " " + err.Message()
	} else {
		msg = err.Message()
	}
	return msg
}

// UnexpectedTokenError is returned by Parse when an unexpected token is encountered.
//
// This is useful for composing parsers in order to detect when a sub-parser has terminated.
type UnexpectedTokenError struct {
	Unexpected lexer.Token
	Expect     string
	expectNode node // Usable instead of Expect, delays creating the string representation until necessary
}

func (u *UnexpectedTokenError) Error() string { return FormatError(u) }

func (u *UnexpectedTokenError) Message() string { // nolint: golint
	var expected string
	if u.expectNode != nil {
		expected = fmt.Sprintf(" (expected %s)", u.expectNode)
	} else if u.Expect != "" {
		expected = fmt.Sprintf(" (expected %s)", u.Expect)
	}
	return fmt.Sprintf("unexpected token %q%s", u.Unexpected, expected)
}
func (u *UnexpectedTokenError) Position() lexer.Position { return u.Unexpected.Pos } // nolint: golint

// ParseError is returned when a parse error occurs.
//
// It is useful for differentiating between parse errors and other errors such
// as lexing and IO errors.
type ParseError struct {
	Msg string
	Pos lexer.Position
}

func (p *ParseError) Error() string            { return FormatError(p) }
func (p *ParseError) Message() string          { return p.Msg }
func (p *ParseError) Position() lexer.Position { return p.Pos }

// Errorf creates a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) Error {
	return &ParseError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

type wrappingParseError struct {
	err error
	ParseError
}

func (w *wrappingParseError) Unwrap() error { return w.err }

// Wrapf attempts to wrap an existing error in a new message.
//
// If "err" is a participle.Error, its positional information will be used and
// "pos" will be ignored.
//
// The returned error implements the Unwrap() method supported by the errors package.
func Wrapf(pos lexer.Position, err error, format string, args ...interface{}) Error {
	var msg string
	if perr, ok := err.(Error); ok {
		pos = perr.Position()
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), perr.Message())
	} else {
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), err.Error())
	}
	return &wrappingParseError{err: err, ParseError: ParseError{Msg: msg, Pos: pos}}
}
