// Package config provides configuration structures for Unqueryvet analyzer.
package config

// UnqueryvetSettings holds the configuration for the Unqueryvet analyzer.
type UnqueryvetSettings struct {
	// CheckSQLBuilders enables checking SQL builders like Squirrel for SELECT * usage
	CheckSQLBuilders bool `mapstructure:"check-sql-builders" json:"check-sql-builders" yaml:"check-sql-builders"`

	// AllowedPatterns is a list of regex patterns that are allowed to use SELECT *
	// Example: ["SELECT \\* FROM temp_.*", "SELECT \\* FROM .*_backup"]
	AllowedPatterns []string `mapstructure:"allowed-patterns" json:"allowed-patterns" yaml:"allowed-patterns"`

	// IgnoredFunctions is a list of function name patterns to ignore
	// Example: ["debug.Query", "test.*", "mock.*"]
	IgnoredFunctions []string `mapstructure:"ignored-functions" json:"ignored-functions" yaml:"ignored-functions"`

	// IgnoredFiles is a list of glob patterns for files to ignore
	// Example: ["*_test.go", "testdata/**", "mock_*.go"]
	IgnoredFiles []string `mapstructure:"ignored-files" json:"ignored-files" yaml:"ignored-files"`

	// Severity defines the diagnostic severity: "error" or "warning" (default: "warning")
	Severity string `mapstructure:"severity" json:"severity" yaml:"severity"`

	// CheckAliasedWildcard enables detection of SELECT alias.* patterns (e.g., SELECT t.* FROM users t)
	CheckAliasedWildcard bool `mapstructure:"check-aliased-wildcard" json:"check-aliased-wildcard" yaml:"check-aliased-wildcard"`

	// CheckStringConcat enables detection of SELECT * in string concatenation (e.g., "SELECT * " + "FROM users")
	CheckStringConcat bool `mapstructure:"check-string-concat" json:"check-string-concat" yaml:"check-string-concat"`

	// CheckFormatStrings enables detection of SELECT * in format functions (e.g., fmt.Sprintf)
	CheckFormatStrings bool `mapstructure:"check-format-strings" json:"check-format-strings" yaml:"check-format-strings"`

	// CheckStringBuilder enables detection of SELECT * in strings.Builder usage
	CheckStringBuilder bool `mapstructure:"check-string-builder" json:"check-string-builder" yaml:"check-string-builder"`

	// CheckSubqueries enables detection of SELECT * in subqueries (e.g., SELECT * FROM (SELECT * FROM ...))
	CheckSubqueries bool `mapstructure:"check-subqueries" json:"check-subqueries" yaml:"check-subqueries"`

	// SQLBuilders defines which SQL builder libraries to check
	SQLBuilders SQLBuildersConfig `mapstructure:"sql-builders" json:"sql-builders" yaml:"sql-builders"`
}

// SQLBuildersConfig defines which SQL builder libraries to analyze.
type SQLBuildersConfig struct {
	// Squirrel enables checking github.com/Masterminds/squirrel
	Squirrel bool `mapstructure:"squirrel" json:"squirrel" yaml:"squirrel"`

	// GORM enables checking gorm.io/gorm
	GORM bool `mapstructure:"gorm" json:"gorm" yaml:"gorm"`

	// SQLx enables checking github.com/jmoiron/sqlx
	SQLx bool `mapstructure:"sqlx" json:"sqlx" yaml:"sqlx"`

	// Ent enables checking entgo.io/ent
	Ent bool `mapstructure:"ent" json:"ent" yaml:"ent"`

	// PGX enables checking github.com/jackc/pgx
	PGX bool `mapstructure:"pgx" json:"pgx" yaml:"pgx"`

	// Bun enables checking github.com/uptrace/bun
	Bun bool `mapstructure:"bun" json:"bun" yaml:"bun"`

	// SQLBoiler enables checking github.com/volatiletech/sqlboiler
	SQLBoiler bool `mapstructure:"sqlboiler" json:"sqlboiler" yaml:"sqlboiler"`

	// Jet enables checking github.com/go-jet/jet
	Jet bool `mapstructure:"jet" json:"jet" yaml:"jet"`
}

// DefaultSQLBuildersConfig returns the default SQL builders configuration with all checkers enabled.
func DefaultSQLBuildersConfig() SQLBuildersConfig {
	return SQLBuildersConfig{
		Squirrel:  true,
		GORM:      true,
		SQLx:      true,
		Ent:       true,
		PGX:       true,
		Bun:       true,
		SQLBoiler: true,
		Jet:       true,
	}
}

// DefaultSettings returns the default configuration for unqueryvet.
// By default, all detection features are enabled for maximum coverage.
func DefaultSettings() UnqueryvetSettings {
	return UnqueryvetSettings{
		CheckSQLBuilders:     true,
		CheckAliasedWildcard: true,
		CheckStringConcat:    true,
		CheckFormatStrings:   true,
		CheckStringBuilder:   true,
		CheckSubqueries:      true,
		Severity:             "warning",
		AllowedPatterns: []string{
			`(?i)COUNT\(\s*\*\s*\)`,
			`(?i)MAX\(\s*\*\s*\)`,
			`(?i)MIN\(\s*\*\s*\)`,
			`(?i)SELECT \* FROM information_schema\..*`,
			`(?i)SELECT \* FROM pg_catalog\..*`,
			`(?i)SELECT \* FROM sys\..*`,
		},
		IgnoredFunctions: []string{},
		IgnoredFiles:     []string{},
		SQLBuilders:      DefaultSQLBuildersConfig(),
	}
}
