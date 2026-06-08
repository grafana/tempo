// Package sqlbuilders provides SQL builder library-specific checkers for SELECT * detection.
package sqlbuilders

import (
	"go/ast"
	"go/token"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

// SelectStarViolation represents a detected SELECT * usage in SQL builder code.
type SelectStarViolation struct {
	// Pos is the position in source code where the violation was detected
	Pos token.Pos
	// End is the end position of the violation
	End token.Pos
	// Message is the human-readable description of the violation
	Message string
	// Builder is the name of the SQL builder library
	Builder string
	// Context provides additional context about the violation type
	Context string
}

// SQLBuilderChecker is the interface for SQL builder library-specific checkers.
// Each SQL builder library (GORM, sqlx, etc.) implements this interface.
type SQLBuilderChecker interface {
	// Name returns the name of the SQL builder library
	Name() string

	// IsApplicable checks if the call expression might be from this SQL builder
	IsApplicable(call *ast.CallExpr) bool

	// CheckSelectStar checks a single call expression for SELECT * usage
	CheckSelectStar(call *ast.CallExpr) *SelectStarViolation

	// CheckChainedCalls analyzes method chains for SELECT * patterns
	CheckChainedCalls(call *ast.CallExpr) []*SelectStarViolation
}

// Registry holds all registered SQL builder checkers and provides a unified interface.
type Registry struct {
	checkers []SQLBuilderChecker
}

// NewRegistry creates a new Registry with checkers based on the configuration.
func NewRegistry(cfg *config.SQLBuildersConfig) *Registry {
	r := &Registry{
		checkers: make([]SQLBuilderChecker, 0),
	}

	// Register enabled checkers
	if cfg.Squirrel {
		r.checkers = append(r.checkers, NewSquirrelChecker())
	}
	if cfg.GORM {
		r.checkers = append(r.checkers, NewGORMChecker())
	}
	if cfg.SQLx {
		r.checkers = append(r.checkers, NewSQLxChecker())
	}
	if cfg.Ent {
		r.checkers = append(r.checkers, NewEntChecker())
	}
	if cfg.PGX {
		r.checkers = append(r.checkers, NewPGXChecker())
	}
	if cfg.Bun {
		r.checkers = append(r.checkers, NewBunChecker())
	}
	if cfg.SQLBoiler {
		r.checkers = append(r.checkers, NewSQLBoilerChecker())
	}
	if cfg.Jet {
		r.checkers = append(r.checkers, NewJetChecker())
	}

	return r
}

// Check analyzes a call expression against all registered checkers.
// Returns all violations found across all applicable checkers.
func (r *Registry) Check(call *ast.CallExpr) []*SelectStarViolation {
	var violations []*SelectStarViolation

	for _, checker := range r.checkers {
		if !checker.IsApplicable(call) {
			continue
		}

		// Check for direct SELECT * usage
		if v := checker.CheckSelectStar(call); v != nil {
			violations = append(violations, v)
		}

		// Check for SELECT * in method chains
		chainViolations := checker.CheckChainedCalls(call)
		violations = append(violations, chainViolations...)
	}

	return violations
}

// HasCheckers returns true if at least one checker is registered.
func (r *Registry) HasCheckers() bool {
	return len(r.checkers) > 0
}
