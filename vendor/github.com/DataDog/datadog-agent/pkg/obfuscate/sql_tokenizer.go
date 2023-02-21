// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// tokenizer.go implemenents a lexer-like iterator that tokenizes SQL and CQL
// strings, so that an external component can filter or alter each token of the
// string. This implementation can't be used as a real SQL lexer (so a parser
// cannot build the AST) because many rules are ignored to make the tokenizer
// simpler.
// This implementation was inspired by https://github.com/youtube/vitess sql parser
// TODO: add the license to the NOTICE file

// TokenKind specifies the type of the token being scanned. It may be one of the defined
// constants below or in some cases the actual rune itself.
type TokenKind uint32

// EndChar is used to signal that the scanner has finished reading the query. This happens when
// there are no more characters left in the query or when invalid encoding is discovered. EndChar
// is an invalid rune value that can not be found in any valid string.
const EndChar = unicode.MaxRune + 1

// list of available tokens; this list has been reduced because we don't
// need a full-fledged tokenizer to implement a Lexer
const (
	LexError = TokenKind(57346) + iota

	ID
	Limit
	Null
	String
	DoubleQuotedString
	DollarQuotedString // https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-DOLLAR-QUOTING
	DollarQuotedFunc   // a dollar-quoted string delimited by the tag "$func$"; gets special treatment when feature "dollar_quoted_func" is set
	Number
	BooleanLiteral
	ValueArg
	ListArg
	Comment
	Variable
	Savepoint
	PreparedStatement
	EscapeSequence
	NullSafeEqual
	LE
	GE
	NE
	Not
	As
	Alter
	Drop
	Create
	Grant
	Revoke
	Commit
	Begin
	Truncate
	Select
	From
	Update
	Delete
	Insert
	Into
	Join
	TableName
	ColonCast

	// PostgreSQL specific JSON operators
	JSONSelect         // ->
	JSONSelectText     // ->>
	JSONSelectPath     // #>
	JSONSelectPathText // #>>
	JSONContains       // @>
	JSONContainsLeft   // <@
	JSONKeyExists      // ?
	JSONAnyKeysExist   // ?|
	JSONAllKeysExist   // ?&
	JSONDelete         // #-

	// FilteredGroupable specifies that the given token has been discarded by one of the
	// token filters and that it is groupable together with consecutive FilteredGroupable
	// tokens.
	FilteredGroupable

	// FilteredGroupableParenthesis is a parenthesis marked as filtered groupable. It is the
	// beginning of either a group of values ('(') or a nested query. We track is as
	// a special case for when it may start a nested query as opposed to just another
	// value group to be obfuscated.
	FilteredGroupableParenthesis

	// Filtered specifies that the token is a comma and was discarded by one
	// of the filters.
	Filtered

	// FilteredBracketedIdentifier specifies that we are currently discarding
	// a bracketed identifier (MSSQL).
	// See issue https://github.com/DataDog/datadog-trace-agent/issues/475.
	FilteredBracketedIdentifier
)

var tokenKindStrings = map[TokenKind]string{
	LexError:                     "LexError",
	ID:                           "ID",
	Limit:                        "Limit",
	Null:                         "Null",
	String:                       "String",
	DoubleQuotedString:           "DoubleQuotedString",
	DollarQuotedString:           "DollarQuotedString",
	DollarQuotedFunc:             "DollarQuotedFunc",
	Number:                       "Number",
	BooleanLiteral:               "BooleanLiteral",
	ValueArg:                     "ValueArg",
	ListArg:                      "ListArg",
	Comment:                      "Comment",
	Variable:                     "Variable",
	Savepoint:                    "Savepoint",
	PreparedStatement:            "PreparedStatement",
	EscapeSequence:               "EscapeSequence",
	NullSafeEqual:                "NullSafeEqual",
	LE:                           "LE",
	GE:                           "GE",
	NE:                           "NE",
	Not:                          "NOT",
	As:                           "As",
	Alter:                        "Alter",
	Drop:                         "Drop",
	Create:                       "Create",
	Grant:                        "Grant",
	Revoke:                       "Revoke",
	Commit:                       "Commit",
	Begin:                        "Begin",
	Truncate:                     "Truncate",
	Select:                       "Select",
	From:                         "From",
	Update:                       "Update",
	Delete:                       "Delete",
	Insert:                       "Insert",
	Into:                         "Into",
	Join:                         "Join",
	TableName:                    "TableName",
	ColonCast:                    "ColonCast",
	FilteredGroupable:            "FilteredGroupable",
	FilteredGroupableParenthesis: "FilteredGroupableParenthesis",
	Filtered:                     "Filtered",
	FilteredBracketedIdentifier:  "FilteredBracketedIdentifier",
	JSONSelect:                   "JSONSelect",
	JSONSelectText:               "JSONSelectText",
	JSONSelectPath:               "JSONSelectPath",
	JSONSelectPathText:           "JSONSelectPathText",
	JSONContains:                 "JSONContains",
	JSONContainsLeft:             "JSONContainsLeft",
	JSONKeyExists:                "JSONKeyExists",
	JSONAnyKeysExist:             "JSONAnyKeysExist",
	JSONAllKeysExist:             "JSONAllKeysExist",
	JSONDelete:                   "JSONDelete",
}

func (k TokenKind) String() string {
	str, ok := tokenKindStrings[k]
	if !ok {
		return "<unknown>"
	}
	return str
}

const (
	// DBMSSQLServer is a MS SQL Server
	DBMSSQLServer = "mssql"
	// DBMSPostgres is a PostgreSQL Server
	DBMSPostgres = "postgresql"
)

const escapeCharacter = '\\'

// SQLTokenizer is the struct used to generate SQL
// tokens for the parser.
type SQLTokenizer struct {
	pos      int    // byte offset of lastChar
	lastChar rune   // last read rune
	buf      []byte // buf holds the query that we are parsing
	off      int    // off is the index into buf where the unread portion of the query begins.
	err      error  // any error occurred while reading

	curlys uint32 // number of active open curly braces in top-level SQL escape sequences.

	literalEscapes bool // indicates we should not treat backslashes as escape characters
	seenEscape     bool // indicates whether this tokenizer has seen an escape character within a string

	cfg *SQLConfig
}

// NewSQLTokenizer creates a new SQLTokenizer for the given SQL string. The literalEscapes argument specifies
// whether escape characters should be treated literally or as such.
func NewSQLTokenizer(sql string, literalEscapes bool, cfg *SQLConfig) *SQLTokenizer {
	if cfg == nil {
		cfg = new(SQLConfig)
	}
	return &SQLTokenizer{
		buf:            []byte(sql),
		cfg:            cfg,
		literalEscapes: literalEscapes,
	}
}

// Reset the underlying buffer and positions
func (tkn *SQLTokenizer) Reset(in string) {
	tkn.pos = 0
	tkn.lastChar = 0
	tkn.buf = []byte(in)
	tkn.off = 0
	tkn.err = nil
}

// keywords used to recognize string tokens
var keywords = map[string]TokenKind{
	"NULL":      Null,
	"TRUE":      BooleanLiteral,
	"FALSE":     BooleanLiteral,
	"SAVEPOINT": Savepoint,
	"LIMIT":     Limit,
	"AS":        As,
	"ALTER":     Alter,
	"CREATE":    Create,
	"GRANT":     Grant,
	"REVOKE":    Revoke,
	"COMMIT":    Commit,
	"BEGIN":     Begin,
	"TRUNCATE":  Truncate,
	"DROP":      Drop,
	"SELECT":    Select,
	"FROM":      From,
	"UPDATE":    Update,
	"DELETE":    Delete,
	"INSERT":    Insert,
	"INTO":      Into,
	"JOIN":      Join,
}

// Err returns the last error that the tokenizer encountered, or nil.
func (tkn *SQLTokenizer) Err() error { return tkn.err }

func (tkn *SQLTokenizer) setErr(format string, args ...interface{}) {
	if tkn.err != nil {
		return
	}
	tkn.err = fmt.Errorf("at position %d: %v", tkn.pos, fmt.Errorf(format, args...))
}

// SeenEscape returns whether or not this tokenizer has seen an escape character within a scanned string
func (tkn *SQLTokenizer) SeenEscape() bool { return tkn.seenEscape }

// Scan scans the tokenizer for the next token and returns
// the token type and the token buffer.
func (tkn *SQLTokenizer) Scan() (TokenKind, []byte) {
	if tkn.lastChar == 0 {
		tkn.advance()
	}
	tkn.SkipBlank()

	switch ch := tkn.lastChar; {
	case isLeadingLetter(ch) &&
		!(tkn.cfg.DBMS == DBMSPostgres && ch == '@'):
		// The '@' symbol should not be considered part of an identifier in
		// postgres, so we skip this in the case where the DBMS is postgres
		// and ch is '@'.
		return tkn.scanIdentifier()
	case isDigit(ch):
		return tkn.scanNumber(false)
	default:
		tkn.advance()
		if tkn.lastChar == EndChar && tkn.err != nil {
			// advance discovered an invalid encoding. We should return early.
			return LexError, nil
		}
		switch ch {
		case EndChar:
			if tkn.err != nil {
				return LexError, nil
			}
			return EndChar, nil
		case ':':
			if tkn.lastChar == ':' {
				tkn.advance()
				return ColonCast, []byte("::")
			}
			if unicode.IsSpace(tkn.lastChar) {
				// example scenario: "autovacuum: VACUUM ANALYZE fake.table"
				return TokenKind(ch), tkn.bytes()
			}
			if tkn.lastChar != '=' {
				return tkn.scanBindVar()
			}
			fallthrough
		case '~':
			switch tkn.lastChar {
			case '*':
				tkn.advance()
				return TokenKind('~'), []byte("~*")
			default:
				return TokenKind(ch), tkn.bytes()
			}
		case '?':
			if tkn.cfg.DBMS == DBMSPostgres {
				switch tkn.lastChar {
				case '|':
					tkn.advance()
					return JSONAnyKeysExist, []byte("?|")
				case '&':
					tkn.advance()
					return JSONAllKeysExist, []byte("?&")
				default:
					return JSONKeyExists, tkn.bytes()
				}
			}
			fallthrough
		case '=', ',', ';', '(', ')', '+', '*', '&', '|', '^', ']':
			return TokenKind(ch), tkn.bytes()
		case '[':
			if tkn.cfg.DBMS == DBMSSQLServer {
				return tkn.scanString(']', DoubleQuotedString)
			}
			return TokenKind(ch), tkn.bytes()
		case '.':
			if isDigit(tkn.lastChar) {
				return tkn.scanNumber(true)
			}
			return TokenKind(ch), tkn.bytes()
		case '/':
			switch tkn.lastChar {
			case '/':
				tkn.advance()
				return tkn.scanCommentType1("//")
			case '*':
				tkn.advance()
				return tkn.scanCommentType2()
			default:
				return TokenKind(ch), tkn.bytes()
			}
		case '-':
			switch {
			case tkn.lastChar == '-':
				tkn.advance()
				return tkn.scanCommentType1("--")
			case tkn.lastChar == '>':
				if tkn.cfg.DBMS == DBMSPostgres {
					tkn.advance()
					switch tkn.lastChar {
					case '>':
						tkn.advance()
						return JSONSelectText, []byte("->>")
					default:
						return JSONSelect, []byte("->")
					}
				}
				fallthrough
			case isDigit(tkn.lastChar):
				return tkn.scanNumber(false)
			case tkn.lastChar == '.':
				tkn.advance()
				if isDigit(tkn.lastChar) {
					return tkn.scanNumber(true)
				}
				tkn.lastChar = '.'
				tkn.pos--
				fallthrough
			default:
				return TokenKind(ch), tkn.bytes()
			}
		case '#':
			switch tkn.cfg.DBMS {
			case DBMSSQLServer:
				return tkn.scanIdentifier()
			case DBMSPostgres:
				switch tkn.lastChar {
				case '>':
					tkn.advance()
					switch tkn.lastChar {
					case '>':
						tkn.advance()
						return JSONSelectPathText, []byte("#>>")
					default:
						return JSONSelectPath, []byte("#>")
					}
				case '-':
					tkn.advance()
					return JSONDelete, []byte("#-")
				default:
					return TokenKind(ch), tkn.bytes()
				}
			default:
				tkn.advance()
				return tkn.scanCommentType1("#")
			}
		case '<':
			switch tkn.lastChar {
			case '>':
				tkn.advance()
				return NE, []byte("<>")
			case '=':
				tkn.advance()
				switch tkn.lastChar {
				case '>':
					tkn.advance()
					return NullSafeEqual, []byte("<=>")
				default:
					return LE, []byte("<=")
				}
			case '@':
				if tkn.cfg.DBMS == DBMSPostgres {
					// check for JSONContainsLeft (<@)
					tkn.advance()
					return JSONContainsLeft, []byte("<@")
				}
				fallthrough
			default:
				return TokenKind(ch), tkn.bytes()
			}
		case '>':
			if tkn.lastChar == '=' {
				tkn.advance()
				return GE, []byte(">=")
			}
			return TokenKind(ch), tkn.bytes()
		case '!':
			switch tkn.lastChar {
			case '=':
				tkn.advance()
				return NE, []byte("!=")
			case '~':
				tkn.advance()
				switch tkn.lastChar {
				case '*':
					tkn.advance()
					return NE, []byte("!~*")
				default:
					return NE, []byte("!~")
				}
			default:
				if isValidCharAfterOperator(tkn.lastChar) {
					return Not, tkn.bytes()
				}
				tkn.setErr(`unexpected char "%c" (%d) after "!"`, tkn.lastChar, tkn.lastChar)
				return LexError, tkn.bytes()
			}
		case '\'':
			return tkn.scanString(ch, String)
		case '"':
			return tkn.scanString(ch, DoubleQuotedString)
		case '`':
			return tkn.scanString(ch, ID)
		case '%':
			if tkn.lastChar == '(' {
				return tkn.scanVariableIdentifier('%')
			}
			if isLetter(tkn.lastChar) {
				// format parameter (e.g. '%s')
				return tkn.scanFormatParameter('%')
			}
			// modulo operator (e.g. 'id % 8')
			return TokenKind(ch), tkn.bytes()
		case '$':
			if isDigit(tkn.lastChar) {
				// TODO(gbbr): the first digit after $ does not necessarily guarantee
				// that this isn't a dollar-quoted string constant. We might eventually
				// want to cover for this use-case too (e.g. $1$some text$1$).
				return tkn.scanPreparedStatement('$')
			}
			kind, tok := tkn.scanDollarQuotedString()
			if kind == DollarQuotedFunc {
				// this is considered an embedded query, we should try and
				// obfuscate it
				out, err := attemptObfuscation(NewSQLTokenizer(string(tok), tkn.literalEscapes, tkn.cfg))
				if err != nil {
					// if we can't obfuscate it, treat it as a regular string
					return DollarQuotedString, tok
				}
				tok = append(append([]byte("$func$"), []byte(out.Query)...), []byte("$func$")...)
			}
			return kind, tok
		case '@':
			if tkn.cfg.DBMS == DBMSPostgres {
				// For postgres the @ symbol is reserved as an operator
				// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-OPERATORS
				// And is used as a json operator
				// https://www.postgresql.org/docs/9.5/functions-json.html
				switch tkn.lastChar {
				case '>':
					tkn.advance()
					return JSONContains, []byte("@>")
				default:
					return TokenKind(ch), tkn.bytes()
				}
			}
			fallthrough
		case '{':
			if tkn.pos == 1 || tkn.curlys > 0 {
				// Do not fully obfuscate top-level SQL escape sequences like {{[?=]call procedure-name[([parameter][,parameter]...)]}.
				// We want these to display a bit more context than just a plain '?'
				// See: https://docs.oracle.com/cd/E13157_01/wlevs/docs30/jdbc_drivers/sqlescape.html
				tkn.curlys++
				return TokenKind(ch), tkn.bytes()
			}
			return tkn.scanEscapeSequence('{')
		case '}':
			if tkn.curlys == 0 {
				// A closing curly brace has no place outside an in-progress top-level SQL escape sequence
				// started by the '{' switch-case.
				tkn.setErr(`unexpected byte %d`, ch)
				return LexError, tkn.bytes()
			}
			tkn.curlys--
			return TokenKind(ch), tkn.bytes()
		default:
			tkn.setErr(`unexpected byte %d`, ch)
			return LexError, tkn.bytes()
		}
	}
}

// SkipBlank moves the tokenizer forward until hitting a non-whitespace character
// The whitespace definition used here is the same as unicode.IsSpace
func (tkn *SQLTokenizer) SkipBlank() {
	for unicode.IsSpace(tkn.lastChar) {
		tkn.advance()
	}
	tkn.bytes()
}

// toUpper is a modified version of bytes.ToUpper. It returns an upper-cased version of the byte
// slice src with all Unicode letters mapped to their upper case. It is modified to also accept a
// byte slice dst as an argument, the underlying storage of which (up to the capacity of dst)
// will be used as the destination of the upper-case copy of src, if it fits. As a special case,
// toUpper will return src if the byte slice is already upper-case. This function is used rather
// than bytes.ToUpper to improve the memory performance of the obfuscator by saving unnecessary
// allocations happening in bytes.ToUpper
func toUpper(src, dst []byte) []byte {
	dst = dst[:0]
	isASCII, hasLower := true, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasLower = hasLower || ('a' <= c && c <= 'z')
	}
	if cap(dst) < len(src) {
		dst = make([]byte, 0, len(src))
	}
	if isASCII { // optimize for ASCII-only byte slices.
		if !hasLower {
			// Just return src.
			return src
		}
		dst = dst[:len(src)]
		for i := 0; i < len(src); i++ {
			c := src[i]
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			dst[i] = c
		}
		return dst
	}
	// This *could* be optimized, but it's an uncommon case.
	return bytes.Map(unicode.ToUpper, src)
}

func (tkn *SQLTokenizer) scanIdentifier() (TokenKind, []byte) {
	tkn.advance()
	for isLetter(tkn.lastChar) || isDigit(tkn.lastChar) || strings.ContainsRune(".*$", tkn.lastChar) {
		tkn.advance()
	}

	t := tkn.bytes()
	// Space allows us to upper-case identifiers 256 bytes long or less without allocating heap
	// storage for them, since space is allocated on the stack. A size of 256 bytes was chosen
	// based on the allowed length of sql identifiers in various sql implementations.
	var space [256]byte
	upper := toUpper(t, space[:0])
	if keywordID, found := keywords[string(upper)]; found {
		return keywordID, t
	}
	return ID, t
}

func (tkn *SQLTokenizer) scanVariableIdentifier(prefix rune) (TokenKind, []byte) {
	for tkn.advance(); tkn.lastChar != ')' && tkn.lastChar != EndChar; tkn.advance() {
	}
	tkn.advance()
	if !isLetter(tkn.lastChar) {
		tkn.setErr(`invalid character after variable identifier: "%c" (%d)`, tkn.lastChar, tkn.lastChar)
		return LexError, tkn.bytes()
	}
	tkn.advance()
	return Variable, tkn.bytes()
}

func (tkn *SQLTokenizer) scanFormatParameter(prefix rune) (TokenKind, []byte) {
	tkn.advance()
	return Variable, tkn.bytes()
}

// scanDollarQuotedString scans a Postgres dollar-quoted string constant.
// See: https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-DOLLAR-QUOTING
func (tkn *SQLTokenizer) scanDollarQuotedString() (TokenKind, []byte) {
	kind, tag := tkn.scanString('$', String)
	if kind == LexError {
		return kind, tkn.bytes()
	}
	var (
		got int
		buf bytes.Buffer
	)
	delim := tag
	// on empty strings, tkn.scanString returns the delimiters
	if string(delim) != "$$" {
		// on non-empty strings, the delimiter is $tag$
		delim = append([]byte{'$'}, delim...)
		delim = append(delim, '$')
	}
	for {
		ch := tkn.lastChar
		tkn.advance()
		if ch == EndChar {
			tkn.setErr("unexpected EOF in dollar-quoted string")
			return LexError, buf.Bytes()
		}
		if byte(ch) == delim[got] {
			got++
			if got == len(delim) {
				break
			}
			continue
		}
		if got > 0 {
			_, err := buf.Write(delim[:got])
			if err != nil {
				tkn.setErr("error reading dollar-quoted string: %v", err)
				return LexError, buf.Bytes()
			}
			got = 0
		}
		buf.WriteRune(ch)
	}
	if tkn.cfg.DollarQuotedFunc && string(delim) == "$func$" {
		return DollarQuotedFunc, buf.Bytes()
	}
	return DollarQuotedString, buf.Bytes()
}

func (tkn *SQLTokenizer) scanPreparedStatement(prefix rune) (TokenKind, []byte) {
	// a prepared statement expect a digit identifier like $1
	if !isDigit(tkn.lastChar) {
		tkn.setErr(`prepared statements must start with digits, got "%c" (%d)`, tkn.lastChar, tkn.lastChar)
		return LexError, tkn.bytes()
	}

	// scanNumber keeps the prefix rune intact.
	// read numbers and return an error if any
	token, buff := tkn.scanNumber(false)
	if token == LexError {
		tkn.setErr("invalid number")
		return LexError, tkn.bytes()
	}
	return PreparedStatement, buff
}

func (tkn *SQLTokenizer) scanEscapeSequence(braces rune) (TokenKind, []byte) {
	for tkn.lastChar != '}' && tkn.lastChar != EndChar {
		tkn.advance()
	}

	// we've reached the end of the string without finding
	// the closing curly braces
	if tkn.lastChar == EndChar {
		tkn.setErr("unexpected EOF in escape sequence")
		return LexError, tkn.bytes()
	}

	tkn.advance()
	return EscapeSequence, tkn.bytes()
}

func (tkn *SQLTokenizer) scanBindVar() (TokenKind, []byte) {
	token := ValueArg
	if tkn.lastChar == ':' {
		token = ListArg
		tkn.advance()
	}
	if !isLetter(tkn.lastChar) && !isDigit(tkn.lastChar) {
		tkn.setErr(`bind variables should start with letters or digits, got "%c" (%d)`, tkn.lastChar, tkn.lastChar)
		return LexError, tkn.bytes()
	}
	for isLetter(tkn.lastChar) || isDigit(tkn.lastChar) || tkn.lastChar == '.' {
		tkn.advance()
	}
	return token, tkn.bytes()
}

func (tkn *SQLTokenizer) scanMantissa(base int) {
	for digitVal(tkn.lastChar) < base {
		tkn.advance()
	}
}

func (tkn *SQLTokenizer) scanNumber(seenDecimalPoint bool) (TokenKind, []byte) {
	if seenDecimalPoint {
		tkn.scanMantissa(10)
		goto exponent
	}

	if tkn.lastChar == '0' {
		// int or float
		tkn.advance()
		if tkn.lastChar == 'x' || tkn.lastChar == 'X' {
			// hexadecimal int
			tkn.advance()
			tkn.scanMantissa(16)
		} else {
			// octal int or float
			tkn.scanMantissa(8)
			if tkn.lastChar == '8' || tkn.lastChar == '9' {
				tkn.scanMantissa(10)
			}
			if tkn.lastChar == '.' || tkn.lastChar == 'e' || tkn.lastChar == 'E' {
				goto fraction
			}
		}
		goto exit
	}

	// decimal int or float
	tkn.scanMantissa(10)

fraction:
	if tkn.lastChar == '.' {
		tkn.advance()
		tkn.scanMantissa(10)
	}

exponent:
	if tkn.lastChar == 'e' || tkn.lastChar == 'E' {
		tkn.advance()
		if tkn.lastChar == '+' || tkn.lastChar == '-' {
			tkn.advance()
		}
		tkn.scanMantissa(10)
	}

exit:
	t := tkn.bytes()
	if len(t) == 0 {
		tkn.setErr("Parse error: ended up with zero-length number.")
		return LexError, nil
	}
	return Number, t
}

func (tkn *SQLTokenizer) scanString(delim rune, kind TokenKind) (TokenKind, []byte) {
	buf := bytes.NewBuffer(tkn.buf[:0])
	for {
		ch := tkn.lastChar
		tkn.advance()
		if ch == delim {
			if tkn.lastChar == delim {
				// doubling a delimiter is the default way to embed the delimiter within a string
				tkn.advance()
			} else {
				// a single delimiter denotes the end of the string
				break
			}
		} else if ch == escapeCharacter {
			tkn.seenEscape = true

			if !tkn.literalEscapes {
				// treat as an escape character
				ch = tkn.lastChar
				tkn.advance()
			}
		}
		if ch == EndChar {
			tkn.setErr("unexpected EOF in string")
			return LexError, buf.Bytes()
		}
		buf.WriteRune(ch)
	}
	if kind == ID && buf.Len() == 0 || bytes.IndexFunc(buf.Bytes(), func(r rune) bool { return !unicode.IsSpace(r) }) == -1 {
		// This string is an empty or white-space only identifier.
		// We should keep the start and end delimiters in order to
		// avoid creating invalid queries.
		// See: https://github.com/DataDog/datadog-trace-agent/issues/316
		return kind, append(runeBytes(delim), runeBytes(delim)...)
	}
	return kind, buf.Bytes()
}

func (tkn *SQLTokenizer) scanCommentType1(prefix string) (TokenKind, []byte) {
	for tkn.lastChar != EndChar {
		if tkn.lastChar == '\n' {
			tkn.advance()
			break
		}
		tkn.advance()
	}
	return Comment, tkn.bytes()
}

func (tkn *SQLTokenizer) scanCommentType2() (TokenKind, []byte) {
	for {
		if tkn.lastChar == '*' {
			tkn.advance()
			if tkn.lastChar == '/' {
				tkn.advance()
				break
			}
			continue
		}
		if tkn.lastChar == EndChar {
			tkn.setErr("unexpected EOF in comment")
			return LexError, tkn.bytes()
		}
		tkn.advance()
	}
	return Comment, tkn.bytes()
}

// advance advances the tokenizer to the next rune. If the decoder encounters an error decoding, or
// the end of the buffer is reached, tkn.lastChar will be set to EndChar. In case of a decoding
// error, tkn.err will also be set.
func (tkn *SQLTokenizer) advance() {
	ch, n := utf8.DecodeRune(tkn.buf[tkn.off:])
	if ch == utf8.RuneError && n < 2 {
		tkn.pos++
		tkn.lastChar = EndChar
		if n == 1 {
			tkn.setErr("invalid UTF-8 encoding beginning with 0x%x", tkn.buf[tkn.off])
		}
		return
	}
	if tkn.lastChar != 0 || tkn.pos > 0 {
		// we are past the first character
		tkn.pos += n
	}
	tkn.off += n
	tkn.lastChar = ch
}

// bytes returns all the bytes that were advanced over since its last call.
// This excludes tkn.lastChar, which will remain in the buffer
func (tkn *SQLTokenizer) bytes() []byte {
	if tkn.lastChar == EndChar {
		ret := tkn.buf[:tkn.off]
		tkn.buf = tkn.buf[tkn.off:]
		tkn.off = 0
		return ret
	}
	lastLen := utf8.RuneLen(tkn.lastChar)
	ret := tkn.buf[:tkn.off-lastLen]
	tkn.buf = tkn.buf[tkn.off-lastLen:]
	tkn.off = lastLen
	return ret
}

// Position exports the tokenizer's current position in the query
func (tkn *SQLTokenizer) Position() int {
	return tkn.pos
}

func isLeadingLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == '@'
}

func isLetter(ch rune) bool {
	return isLeadingLetter(ch) || ch == '#'
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch) - '0'
	case 'a' <= ch && ch <= 'f':
		return int(ch) - 'a' + 10
	case 'A' <= ch && ch <= 'F':
		return int(ch) - 'A' + 10
	}
	return 16 // larger than any legal digit val
}

func isDigit(ch rune) bool { return '0' <= ch && ch <= '9' }

// runeBytes converts the given rune to a slice of bytes.
func runeBytes(r rune) []byte {
	buf := make([]byte, utf8.UTFMax)
	n := utf8.EncodeRune(buf, r)
	return buf[:n]
}

// isValidCharAfterOperator returns true if c is a valid character after an operator
func isValidCharAfterOperator(c rune) bool {
	return c == '(' || c == '`' || c == '\'' || c == '"' || c == '+' || c == '-' || unicode.IsSpace(c) || isLetter(c) || isDigit(c)
}
