// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package obfuscate implements quantizing and obfuscating of tags and resources for
// a set of spans matching a certain criteria.
//
// This module is used in the Datadog Agent, the Go tracing client (dd-trace-go) and in the
// OpenTelemetry Collector Datadog exporter./ End-user behavior is stable, but there are no
// stability guarantees on its public Go API. Nonetheless, if editing try to avoid breaking
// API changes if possible and double check the API usage on all module dependents.
package obfuscate

import (
	"bytes"

	"github.com/DataDog/datadog-go/v5/statsd"
	"go.uber.org/atomic"
)

// Obfuscator quantizes and obfuscates spans. The obfuscator is not safe for
// concurrent use.
type Obfuscator struct {
	opts                 *Config
	es                   *jsonObfuscator // nil if disabled
	mongo                *jsonObfuscator // nil if disabled
	sqlExecPlan          *jsonObfuscator // nil if disabled
	sqlExecPlanNormalize *jsonObfuscator // nil if disabled
	// sqlLiteralEscapes reports whether we should treat escape characters literally or as escape characters.
	// Different SQL engines behave in different ways and the tokenizer needs to be generic.
	sqlLiteralEscapes *atomic.Bool
	// queryCache keeps a cache of already obfuscated queries.
	queryCache *measuredCache
	log        Logger
}

// Logger is able to log certain log messages.
type Logger interface {
	// Debugf logs the given message using the given format.
	Debugf(format string, params ...interface{})
}

type noopLogger struct{}

func (noopLogger) Debugf(_ string, _ ...interface{}) {}

// setSQLLiteralEscapes sets whether or not escape characters should be treated literally by the SQL obfuscator.
func (o *Obfuscator) setSQLLiteralEscapes(ok bool) {
	if ok {
		o.sqlLiteralEscapes.Store(true)
	} else {
		o.sqlLiteralEscapes.Store(false)
	}
}

// useSQLLiteralEscapes reports whether escape characters will be treated literally by the SQL obfuscator.
// Some SQL engines require it and others don't. It will be detected as SQL queries are being obfuscated
// through calls to ObfuscateSQLString and automatically set for future.
func (o *Obfuscator) useSQLLiteralEscapes() bool {
	return o.sqlLiteralEscapes.Load()
}

// Config holds the configuration for obfuscating sensitive data for various span types.
type Config struct {
	// SQL holds the obfuscation configuration for SQL queries.
	SQL SQLConfig

	// ES holds the obfuscation configuration for ElasticSearch bodies.
	ES JSONConfig

	// Mongo holds the obfuscation configuration for MongoDB queries.
	Mongo JSONConfig

	// SQLExecPlan holds the obfuscation configuration for SQL Exec Plans. This is strictly for safety related obfuscation,
	// not normalization. Normalization of exec plans is configured in SQLExecPlanNormalize.
	SQLExecPlan JSONConfig

	// SQLExecPlanNormalize holds the normalization configuration for SQL Exec Plans.
	SQLExecPlanNormalize JSONConfig

	// HTTP holds the obfuscation settings for HTTP URLs.
	HTTP HTTPConfig

	// Statsd specifies the statsd client to use for reporting metrics.
	Statsd StatsClient

	// Logger specifies the logger to use when outputting messages.
	// If unset, no logs will be outputted.
	Logger Logger
}

// StatsClient implementations are able to emit stats.
type StatsClient interface {
	// Gauge reports a gauge stat with the given name, value, tags and rate.
	Gauge(name string, value float64, tags []string, rate float64) error
}

// SQLConfig holds the config for obfuscating SQL.
type SQLConfig struct {
	// DBMS identifies the type of database management system (e.g. MySQL, Postgres, and SQL Server).
	// Valid values for this can be found at https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/database.md#connection-level-attributes
	DBMS string `json:"dbms"`

	// TableNames specifies whether the obfuscator should also extract the table names that a query addresses,
	// in addition to obfuscating.
	TableNames bool `json:"table_names"`

	// CollectCommands specifies whether the obfuscator should extract and return commands as SQL metadata when obfuscating.
	CollectCommands bool `json:"collect_commands"`

	// CollectComments specifies whether the obfuscator should extract and return comments as SQL metadata when obfuscating.
	CollectComments bool `json:"collect_comments"`

	// ReplaceDigits specifies whether digits in table names and identifiers should be obfuscated.
	ReplaceDigits bool `json:"replace_digits"`

	// KeepSQLAlias reports whether SQL aliases ("AS") should be truncated.
	KeepSQLAlias bool `json:"keep_sql_alias"`

	// DollarQuotedFunc reports whether to treat "$func$" delimited dollar-quoted strings
	// differently and not obfuscate them as a string. To read more about dollar quoted
	// strings see:
	//
	// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-DOLLAR-QUOTING
	DollarQuotedFunc bool `json:"dollar_quoted_func"`

	// Cache reports whether the obfuscator should use a LRU look-up cache for SQL obfuscations.
	Cache bool
}

// SQLMetadata holds metadata collected throughout the obfuscation of an SQL statement. It is only
// collected when enabled via SQLConfig.
type SQLMetadata struct {
	// Size holds the byte size of the metadata collected.
	Size int64
	// TablesCSV is a comma-separated list of tables that the query addresses.
	TablesCSV string `json:"tables_csv"`
	// Commands holds commands executed in an SQL statement.
	// e.g. SELECT, UPDATE, INSERT, DELETE, etc.
	Commands []string `json:"commands"`
	// Comments holds comments in an SQL statement.
	Comments []string `json:"comments"`
}

// HTTPConfig holds the configuration settings for HTTP obfuscation.
type HTTPConfig struct {
	// RemoveQueryStrings determines query strings to be removed from HTTP URLs.
	RemoveQueryString bool

	// RemovePathDigits determines digits in path segments to be obfuscated.
	RemovePathDigits bool
}

// JSONConfig holds the obfuscation configuration for sensitive
// data found in JSON objects.
type JSONConfig struct {
	// Enabled will specify whether obfuscation should be enabled.
	Enabled bool

	// KeepValues will specify a set of keys for which their values will
	// not be obfuscated.
	KeepValues []string

	// ObfuscateSQLValues will specify a set of keys for which their values
	// will be passed through SQL obfuscation
	ObfuscateSQLValues []string
}

// NewObfuscator creates a new obfuscator
func NewObfuscator(cfg Config) *Obfuscator {
	if cfg.Logger == nil {
		cfg.Logger = noopLogger{}
	}
	o := Obfuscator{
		opts:              &cfg,
		queryCache:        newMeasuredCache(cacheOptions{On: cfg.SQL.Cache, Statsd: cfg.Statsd}),
		sqlLiteralEscapes: atomic.NewBool(false),
		log:               cfg.Logger,
	}
	if cfg.ES.Enabled {
		o.es = newJSONObfuscator(&cfg.ES, &o)
	}
	if cfg.Mongo.Enabled {
		o.mongo = newJSONObfuscator(&cfg.Mongo, &o)
	}
	if cfg.SQLExecPlan.Enabled {
		o.sqlExecPlan = newJSONObfuscator(&cfg.SQLExecPlan, &o)
	}
	if cfg.SQLExecPlanNormalize.Enabled {
		o.sqlExecPlanNormalize = newJSONObfuscator(&cfg.SQLExecPlanNormalize, &o)
	}
	if cfg.Statsd == nil {
		cfg.Statsd = &statsd.NoOpClient{}
	}
	return &o
}

// Stop cleans up after a finished Obfuscator.
func (o *Obfuscator) Stop() {
	o.queryCache.Close()
}

// compactWhitespaces compacts all whitespaces in t.
func compactWhitespaces(t string) string {
	n := len(t)
	r := make([]byte, n)
	spaceCode := uint8(32)
	isWhitespace := func(char uint8) bool { return char == spaceCode }
	nr := 0
	offset := 0
	for i := 0; i < n; i++ {
		if isWhitespace(t[i]) {
			copy(r[nr:], t[nr+offset:i])
			r[i-offset] = spaceCode
			nr = i + 1 - offset
			for j := i + 1; j < n; j++ {
				if !isWhitespace(t[j]) {
					offset += j - i - 1
					i = j
					break
				} else if j == n-1 {
					offset += j - i
					i = j
					break
				}
			}
		}
	}
	copy(r[nr:], t[nr+offset:n])
	r = r[:n-offset]
	return string(bytes.Trim(r, " "))
}

// replaceDigits replaces consecutive sequences of digits with '?',
// example: "jobs_2020_1597876964" --> "jobs_?_?"
func replaceDigits(buffer []byte) []byte {
	scanningDigit := false
	filtered := buffer[:0]
	for _, b := range buffer {
		// digits are encoded as 1 byte in utf8
		if isDigit(rune(b)) {
			if scanningDigit {
				continue
			}
			scanningDigit = true
			filtered = append(filtered, byte('?'))
			continue
		}
		scanningDigit = false
		filtered = append(filtered, b)
	}
	return filtered
}
