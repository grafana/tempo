// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlerror // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

// Position represents the position in the input where an error occurred.
type Position interface {
	Line() int
	Offset() int
	Column() int

	unexported()
}

// Error represents an error that occurred during parsing/evaluation of OTTL statements, conditions,
// and value expressions. It provides a message and optionally a position in the input where
// the error occurred.
type Error interface {
	error
	// Message returns the unadorned error message.
	Message() string
	// Position returns the position in the input where the error occurred.
	Position() Position

	unexported()
}

type errorPosition struct {
	line   int
	offset int
	column int
}

func (e errorPosition) Line() int {
	return e.line
}

func (e errorPosition) Offset() int {
	return e.offset
}

func (e errorPosition) Column() int {
	return e.column
}

func (errorPosition) unexported() {}

// AsPosition converts a lexer.Position to an ottlerror.Position.
func AsPosition(position lexer.Position) Position {
	return &errorPosition{
		line:   position.Line,
		column: position.Column,
		offset: position.Offset,
	}
}

// AsLexerPosition converts an ottlerror.Position to a lexer.Position.
// If position is nil, it returns a zero-value lexer.Position.
func AsLexerPosition(position Position) lexer.Position {
	if position == nil {
		return lexer.Position{}
	}
	return lexer.Position{
		Line:   position.Line(),
		Column: position.Column(),
		Offset: position.Offset(),
	}
}

type ottlError struct {
	message  string
	position Position
}

func (e *ottlError) Error() string {
	return formatError(e)
}

func (e *ottlError) Position() Position {
	return e.position
}

func (e *ottlError) Message() string {
	return e.message
}

func (*ottlError) unexported() {}

type ottlErrorWrapper struct {
	err error
	ottlError
}

func (e *ottlErrorWrapper) Unwrap() error {
	return e.err
}

func formatError(err Error) string {
	pos := err.Position()
	if pos != nil && (pos.Line() != 0 || pos.Column() != 0) {
		return fmt.Sprintf("%d:%d: %s", pos.Line(), pos.Column(), err.Message())
	}
	return err.Message()
}

// positionAndMessage returns the position and unadorned message to reuse when err is
// already an Error, falling back to pos and err.Error() otherwise.
func positionAndMessage(pos lexer.Position, err error) (Position, string) {
	if ottlErr, ok := errors.AsType[Error](err); ok {
		return ottlErr.Position(), ottlErr.Message()
	}
	return AsPosition(pos), err.Error()
}

// FromError builds an Error from an existing error, preserving its position and message if
// err is already an Error, otherwise defaulting to pos.
func FromError(pos lexer.Position, err error) Error {
	position, message := positionAndMessage(pos, err)
	return &ottlErrorWrapper{
		err:       err,
		ottlError: ottlError{position: position, message: message},
	}
}

// Wrap wraps an error with a new message and position, preserving the original error
// for unwrapping. If err is already an Error, its own position is preserved in favor of
// pos.
func Wrap(pos lexer.Position, err error, message string) Error {
	position, _ := positionAndMessage(pos, err)
	return &ottlErrorWrapper{
		err: err,
		ottlError: ottlError{
			position: position,
			message:  message,
		},
	}
}

// Wrapf wraps an error with a new formatted message and position, preserving the original error
// for unwrapping.
func Wrapf(pos lexer.Position, err error, format string, args ...any) Error {
	position, innerMessage := positionAndMessage(pos, err)
	return &ottlErrorWrapper{
		err:       err,
		ottlError: ottlError{position: position, message: fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), innerMessage)},
	}
}

// Errorf creates a new Error with the given position and formatted message.
func Errorf(pos lexer.Position, format string, args ...any) Error {
	return &ottlError{
		message:  fmt.Sprintf(format, args...),
		position: AsPosition(pos),
	}
}

// New creates a new Error with the given position and message.
func New(pos lexer.Position, message string) Error {
	return &ottlError{
		message:  message,
		position: AsPosition(pos),
	}
}
