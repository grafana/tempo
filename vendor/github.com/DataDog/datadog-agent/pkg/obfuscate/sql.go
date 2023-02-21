// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var questionMark = []byte("?")

// metadataFinderFilter is a filter which attempts to collect metadata from a query, such as comments and tables.
// It is meant to run before all the other filters.
type metadataFinderFilter struct {
	collectTableNames bool
	collectCommands   bool
	collectComments   bool
	replaceDigits     bool

	// size holds the byte size of the metadata collected by the filter.
	size int64
	// tablesSeen keeps track of unique table names encountered by the filter.
	tablesSeen map[string]struct{}
	// tablesCSV specifies a comma-separated list of tables.
	tablesCSV strings.Builder
	// commands keeps track of commands encountered by the filter.
	commands []string
	// comments keeps track of comments encountered by the filter.
	comments []string
}

func (f *metadataFinderFilter) Filter(token, lastToken TokenKind, buffer []byte) (TokenKind, []byte, error) {
	if f.collectComments && token == Comment {
		// A comment with line-breaks will be brought to a single line.
		comment := strings.TrimSpace(strings.Replace(string(buffer), "\n", " ", -1))
		f.size += int64(len(comment))
		f.comments = append(f.comments, comment)
	}
	if f.collectCommands {
		switch token {
		case Select, Update, Insert, Delete, Join, Alter, Drop, Create, Grant, Revoke, Commit, Begin, Truncate:
			command := strings.ToUpper(token.String())
			f.size += int64(len(command))
			f.commands = append(f.commands, command)
		}
	}
	if f.collectTableNames {
		switch lastToken {
		case From, Join:
			// SELECT ... FROM [tableName]
			// DELETE FROM [tableName]
			// ... JOIN [tableName]
			if r, _ := utf8.DecodeRune(buffer); !unicode.IsLetter(r) {
				// first character in buffer is not a letter; we might have a nested
				// query like SELECT * FROM (SELECT ...)
				break
			}
			fallthrough
		case Update, Into:
			// UPDATE [tableName]
			// INSERT INTO [tableName]
			tableName := string(buffer)
			if f.replaceDigits {
				tableNameCopy := make([]byte, len(buffer))
				copy(tableNameCopy, buffer)
				tableName = string(replaceDigits(tableNameCopy))
			}
			f.storeTableName(tableName)
			return TableName, buffer, nil
		}
	}
	return token, buffer, nil
}

func (f *metadataFinderFilter) storeTableName(name string) {
	if _, ok := f.tablesSeen[name]; ok {
		return
	}
	if f.tablesSeen == nil {
		f.tablesSeen = make(map[string]struct{}, 1)
	}
	f.tablesSeen[name] = struct{}{}
	if f.tablesCSV.Len() > 0 {
		f.size++
		f.tablesCSV.WriteByte(',')
	}
	f.size += int64(len(name))
	f.tablesCSV.WriteString(name)
}

// Results returns metadata collected by the filter for an SQL statement.
func (f *metadataFinderFilter) Results() SQLMetadata {
	return SQLMetadata{
		Size:      f.size,
		TablesCSV: f.tablesCSV.String(),
		Commands:  f.commands,
		Comments:  f.comments,
	}
}

// Reset implements tokenFilter.
func (f *metadataFinderFilter) Reset() {
	for k := range f.tablesSeen {
		delete(f.tablesSeen, k)
	}
	f.size = 0
	f.tablesCSV.Reset()
	f.commands = f.commands[:0]
	f.comments = f.comments[:0]
}

// discardFilter is a token filter which discards certain elements from a query, such as
// comments and AS aliases by returning a nil buffer.
type discardFilter struct {
	keepSQLAlias bool
}

// Filter the given token so that a `nil` slice is returned if the token is in the token filtered list.
func (f *discardFilter) Filter(token, lastToken TokenKind, buffer []byte) (TokenKind, []byte, error) {
	// filters based on previous token
	switch lastToken {
	case FilteredBracketedIdentifier:
		if token != ']' {
			// we haven't found the closing bracket yet, keep going
			if token != ID {
				// the token between the brackets *must* be an identifier,
				// otherwise the query is invalid.
				return LexError, nil, fmt.Errorf("expected identifier in bracketed filter, got %d", token)
			}
			return FilteredBracketedIdentifier, nil, nil
		}
		fallthrough
	case As:
		if token == '[' {
			// the identifier followed by AS is an MSSQL bracketed identifier
			// and will continue to be discarded until we find the corresponding
			// closing bracket counter-part. See GitHub issue DataDog/datadog-trace-agent#475.
			return FilteredBracketedIdentifier, nil, nil
		}
		if f.keepSQLAlias {
			return token, buffer, nil
		}
		return Filtered, nil, nil
	}

	// filters based on the current token; if the next token should be ignored,
	// return the same token value (not FilteredGroupable) and nil
	switch token {
	case Comment:
		return Filtered, nil, nil
	case ';':
		return markFilteredGroupable(token), nil, nil
	case As:
		if !f.keepSQLAlias {
			return As, nil, nil
		}
		fallthrough
	default:
		return token, buffer, nil
	}
}

// Reset implements tokenFilter.
func (f *discardFilter) Reset() {}

// replaceFilter is a token filter which obfuscates strings and numbers in queries by replacing them
// with the "?" character.
type replaceFilter struct {
	replaceDigits bool
}

// Filter the given token so that it will be replaced if in the token replacement list
func (f *replaceFilter) Filter(token, lastToken TokenKind, buffer []byte) (tokenType TokenKind, tokenBytes []byte, err error) {
	switch lastToken {
	case Savepoint:
		return markFilteredGroupable(token), questionMark, nil
	case '=':
		switch token {
		case DoubleQuotedString:
			// double-quoted strings after assignments are eligible for obfuscation
			return markFilteredGroupable(token), questionMark, nil
		}
	}
	switch token {
	case DollarQuotedString, String, Number, Null, Variable, PreparedStatement, BooleanLiteral, EscapeSequence:
		return markFilteredGroupable(token), questionMark, nil
	case '?':
		// Cases like 'ARRAY [ ?, ? ]' should be collapsed into 'ARRAY [ ? ]'
		return markFilteredGroupable(token), questionMark, nil
	case TableName, ID:
		if f.replaceDigits {
			return token, replaceDigits(buffer), nil
		}
		fallthrough
	default:
		return token, buffer, nil
	}
}

// Reset implements tokenFilter.
func (f *replaceFilter) Reset() {}

// groupingFilter is a token filter which groups together items replaced by the replaceFilter. It is meant
// to run immediately after it.
type groupingFilter struct {
	groupFilter int // counts the number of values, e.g. 3 = ?, ?, ?
	groupMulti  int // counts the number of groups, e.g. 2 = (?, ?), (?, ?, ?)
}

// Filter the given token so that it will be discarded if a grouping pattern
// has been recognized. A grouping is composed by items like:
//   - '( ?, ?, ? )'
//   - '( ?, ? ), ( ?, ? )'
func (f *groupingFilter) Filter(token, lastToken TokenKind, buffer []byte) (tokenType TokenKind, tokenBytes []byte, err error) {
	// increasing the number of groups means that we're filtering an entire group
	// because it can be represented with a single '( ? )'
	if (lastToken == '(' && isFilteredGroupable(token)) || (token == '(' && f.groupMulti > 0) {
		f.groupMulti++
	}

	// Potential commands that could indicate the start of a subquery.
	isStartOfSubquery := token == Select || token == Delete || token == Update || token == ID

	switch {
	case f.groupMulti > 0 && lastToken == FilteredGroupableParenthesis && isStartOfSubquery:
		// this is the start of a new group that seems to be a nested query;
		// cancel grouping.
		f.Reset()
		return token, append([]byte("( "), buffer...), nil
	case isFilteredGroupable(token):
		// the previous filter has dropped this token so we should start
		// counting the group filter so that we accept only one '?' for
		// the same group
		f.groupFilter++

		if f.groupFilter > 1 {
			return markFilteredGroupable(token), nil, nil
		}
	case f.groupFilter > 0 && (token == ',' || token == '?'):
		// if we are in a group drop all commas
		return markFilteredGroupable(token), nil, nil
	case f.groupMulti > 1:
		// drop all tokens since we're in a counting group
		// and they're duplicated
		return markFilteredGroupable(token), nil, nil
	case token != ',' && token != '(' && token != ')' && !isFilteredGroupable(token):
		// when we're out of a group reset the filter state
		f.Reset()
	}

	return token, buffer, nil
}

// isFilteredGroupable reports whether token is to be considered filtered groupable.
func isFilteredGroupable(token TokenKind) bool {
	switch token {
	case FilteredGroupable, FilteredGroupableParenthesis:
		return true
	default:
		return false
	}
}

// markFilteredGroupable returns the appropriate TokenKind to mark this token as
// filtered groupable.
func markFilteredGroupable(token TokenKind) TokenKind {
	switch token {
	case '(':
		return FilteredGroupableParenthesis
	default:
		return FilteredGroupable
	}
}

// Reset resets the groupingFilter so that it may be used again.
func (f *groupingFilter) Reset() {
	f.groupFilter = 0
	f.groupMulti = 0
}

// ObfuscateSQLString quantizes and obfuscates the given input SQL query string. Quantization removes
// some elements such as comments and aliases and obfuscation attempts to hide sensitive information
// in strings and numbers by redacting them.
func (o *Obfuscator) ObfuscateSQLString(in string) (*ObfuscatedQuery, error) {
	return o.ObfuscateSQLStringWithOptions(in, &o.opts.SQL)
}

// ObfuscateSQLStringWithOptions accepts an optional SQLOptions to change the behavior of the obfuscator
// to quantize and obfuscate the given input SQL query string. Quantization removes some elements such as comments
// and aliases and obfuscation attempts to hide sensitive information in strings and numbers by redacting them.
func (o *Obfuscator) ObfuscateSQLStringWithOptions(in string, opts *SQLConfig) (*ObfuscatedQuery, error) {
	if v, ok := o.queryCache.Get(in); ok {
		return v.(*ObfuscatedQuery), nil
	}
	oq, err := o.obfuscateSQLString(in, opts)
	if err != nil {
		return oq, err
	}
	o.queryCache.Set(in, oq, oq.Cost())
	return oq, nil
}

func (o *Obfuscator) obfuscateSQLString(in string, opts *SQLConfig) (*ObfuscatedQuery, error) {
	lesc := o.useSQLLiteralEscapes()
	tok := NewSQLTokenizer(in, lesc, opts)
	out, err := attemptObfuscation(tok)
	if err != nil && tok.SeenEscape() {
		// If the tokenizer failed, but saw an escape character in the process,
		// try again treating escapes differently
		tok = NewSQLTokenizer(in, !lesc, opts)
		if out, err2 := attemptObfuscation(tok); err2 == nil {
			// If the second attempt succeeded, change the default behavior so that
			// on the next run we get it right in the first run.
			o.setSQLLiteralEscapes(!lesc)
			return out, nil
		}
	}
	return out, err
}

// ObfuscatedQuery specifies information about an obfuscated SQL query.
type ObfuscatedQuery struct {
	Query    string      `json:"query"`    // the obfuscated SQL query
	Metadata SQLMetadata `json:"metadata"` // metadata extracted from the SQL query
}

// Cost returns the number of bytes needed to store all the fields
// of this ObfuscatedQuery.
func (oq *ObfuscatedQuery) Cost() int64 {
	return int64(len(oq.Query)) + oq.Metadata.Size
}

// attemptObfuscation attempts to obfuscate the SQL query loaded into the tokenizer, using the given set of filters.
func attemptObfuscation(tokenizer *SQLTokenizer) (*ObfuscatedQuery, error) {
	var (
		out       = bytes.NewBuffer(make([]byte, 0, len(tokenizer.buf)))
		err       error
		lastToken TokenKind
		metadata  = metadataFinderFilter{
			collectTableNames: tokenizer.cfg.TableNames,
			collectCommands:   tokenizer.cfg.CollectCommands,
			collectComments:   tokenizer.cfg.CollectComments,
			replaceDigits:     tokenizer.cfg.ReplaceDigits,
		}
		discard  = discardFilter{keepSQLAlias: tokenizer.cfg.KeepSQLAlias}
		replace  = replaceFilter{replaceDigits: tokenizer.cfg.ReplaceDigits}
		grouping groupingFilter
	)
	defer metadata.Reset()
	// call Scan() function until tokens are available or if a LEX_ERROR is raised. After
	// retrieving a token, send it to the tokenFilter chains so that the token is discarded
	// or replaced.
	for {
		token, buff := tokenizer.Scan()
		if token == EndChar {
			break
		}
		if token == LexError {
			return nil, fmt.Errorf("%v", tokenizer.Err())
		}

		if token, buff, err = metadata.Filter(token, lastToken, buff); err != nil {
			return nil, err
		}
		if token, buff, err = discard.Filter(token, lastToken, buff); err != nil {
			return nil, err
		}
		if token, buff, err = replace.Filter(token, lastToken, buff); err != nil {
			return nil, err
		}
		if token, buff, err = grouping.Filter(token, lastToken, buff); err != nil {
			return nil, err
		}
		if buff != nil {
			if out.Len() != 0 {
				switch token {
				case ',':
				case '=':
					if lastToken == ':' {
						// do not add a space before an equals if a colon was
						// present before it.
						break
					}
					fallthrough
				default:
					out.WriteRune(' ')
				}
			}
			out.Write(buff)
		}
		lastToken = token
	}
	if out.Len() == 0 {
		return nil, errors.New("result is empty")
	}
	return &ObfuscatedQuery{
		Query:    out.String(),
		Metadata: metadata.Results(),
	}, nil
}

// ObfuscateSQLExecPlan obfuscates query conditions in the provided JSON encoded execution plan. If normalize=True,
// then cost and row estimates are also obfuscated away.
func (o *Obfuscator) ObfuscateSQLExecPlan(jsonPlan string, normalize bool) (string, error) {
	if normalize {
		return o.sqlExecPlanNormalize.obfuscate([]byte(jsonPlan))
	}
	return o.sqlExecPlan.obfuscate([]byte(jsonPlan))
}
