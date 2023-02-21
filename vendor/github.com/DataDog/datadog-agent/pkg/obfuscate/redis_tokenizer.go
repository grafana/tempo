// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"bytes"
	"strings"
)

// redisTokenType specifies the token type returned by the tokenizer.
type redisTokenType int

const (
	// redisTokenCommand is a command token. For compound tokens, it is
	// only the first part up to a space.
	redisTokenCommand redisTokenType = iota

	// redisTokenArgument is an argument token.
	redisTokenArgument
)

// String implements fmt.Stringer.
func (t redisTokenType) String() string {
	return map[redisTokenType]string{
		redisTokenCommand:  "command",
		redisTokenArgument: "argument",
	}[t]
}

// redisTokenizer tokenizes a Redis command string. The string can be on
// multiple lines. The tokenizer is capable of parsing quoted strings and escape
// sequences inside them.
type redisTokenizer struct {
	data  []byte
	ch    byte
	off   int
	done  bool
	state redisParseState
}

// redisParseState specifies the current state of the tokenizer.
type redisParseState int

const (
	// redisStateCommand specifies that we are about to parse a command.
	// It is usually the state at the beginning of the scan or after a
	// new line.
	redisStateCommand redisParseState = iota

	// redisStateArgument specifies that we are about to parse an argument
	// to a command or the rest of the tokens in a compound command.
	redisStateArgument
)

// newRedisTokenizer returns a new tokenizer for the given data.
func newRedisTokenizer(data []byte) *redisTokenizer {
	return &redisTokenizer{
		data:  bytes.TrimSpace(data),
		off:   -1,
		state: redisStateCommand,
	}
}

// scan returns the next token, it's type and a bool. The boolean specifies if
// the returned token was the last one.
func (t *redisTokenizer) scan() (tok string, typ redisTokenType, done bool) {
	switch t.state {
	case redisStateCommand:
		return t.scanCommand()
	default:
		return t.scanArg()
	}
}

// next advances the scanner to the next character.
func (t *redisTokenizer) next() {
	t.off++
	if t.off <= len(t.data)-1 {
		t.ch = t.data[t.off]
		return
	}
	t.done = true
}

// scanCommand scans a command from the buffer.
func (t *redisTokenizer) scanCommand() (tok string, typ redisTokenType, done bool) {
	var (
		str     strings.Builder
		started bool
	)
	for {
		t.next()
		if t.done {
			return str.String(), typ, t.done
		}
		switch t.ch {
		case ' ':
			if !started {
				// skip spaces preceding token
				t.skipSpace()
				break
			}
			// done scanning command
			t.state = redisStateArgument
			t.skipSpace()
			return str.String(), redisTokenCommand, t.done
		case '\n':
			return str.String(), redisTokenCommand, t.done
		default:
			str.WriteByte(t.ch)
		}
		started = true
	}
}

// scanArg scans an argument from the buffer.
func (t *redisTokenizer) scanArg() (tok string, typ redisTokenType, done bool) {
	var (
		str    strings.Builder
		quoted bool // in quoted string
		escape bool // escape sequence
	)
	for {
		t.next()
		if t.done {
			return str.String(), redisTokenArgument, t.done
		}
		switch t.ch {
		case '\\':
			str.WriteByte('\\')
			if !escape {
				// next character could be escaped
				escape = true
				continue
			}
		case '\n':
			if !quoted {
				// last argument, new command follows
				t.state = redisStateCommand
				return str.String(), redisTokenArgument, t.done
			}
			str.WriteByte('\n')
		case '"':
			str.WriteByte('"')
			if !escape {
				// this quote wasn't escaped, toggle quoted mode
				quoted = !quoted
			}
		case ' ':
			if !quoted {
				t.skipSpace()
				return str.String(), redisTokenArgument, t.done
			}
			str.WriteByte(' ')
		default:
			str.WriteByte(t.ch)
		}
		escape = false
	}
}

// unread is the reverse of next, unreading a character.
func (t *redisTokenizer) unread() {
	if t.off < 1 {
		return
	}
	t.off--
	t.ch = t.data[t.off]
}

// skipSpace moves the cursor forward until it meets the last space
// in a sequence of contiguous spaces.
func (t *redisTokenizer) skipSpace() {
	for t.ch == ' ' || t.ch == '\t' || t.ch == '\r' && !t.done {
		t.next()
	}
	if t.ch == '\n' {
		// next token is a command
		t.state = redisStateCommand
	} else {
		// don't steal the first non-space character
		t.unread()
	}
}
