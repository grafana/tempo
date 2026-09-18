package backendscheduler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/traceql"
)

// The redaction query selector accepts only a single spanset filter restricted to
// equality on resource.*/span.* attributes joined by && / ||. Everything
// else is rejected at submission. See designDocRedactionTraceQLQuery.md "Query subset".
func TestValidateRedactionQuery(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		// --- accepted ---
		{
			name:    "single service equality",
			query:   `{resource.service_name = "a"}`,
			wantErr: false,
		},
		{
			name:    "motivating case: several service equalities OR'd",
			query:   `{resource.service_name = "a" || resource.service_name = "b" || resource.service_name = "c"}`,
			wantErr: false,
		},
		{
			name:    "equality with AND",
			query:   `{resource.namespace = "prod" && span.http.target = "/checkout"}`,
			wantErr: false,
		},
		// --- rejected: operators outside the subset ---
		{
			name:    "regex match",
			query:   `{resource.service_name =~ "a.*"}`,
			wantErr: true,
		},
		{
			name:    "ordered comparison",
			query:   `{span.http.status_code > 400}`,
			wantErr: true,
		},
		// Negation is rejected: its blast radius is the complement of the match set
		// (potentially all data), so a typo is as catastrophic as a bad regex on a
		// delete path.
		{
			name:    "negation (complement blast radius)",
			query:   `{span.http.target != "/health"}`,
			wantErr: true,
		},
		{
			name:    "negation within AND",
			query:   `{resource.namespace = "prod" && span.http.target != "/health"}`,
			wantErr: true,
		},
		// --- rejected: shape outside a single spanset filter ---
		{
			name:    "pipeline aggregate",
			query:   `{resource.service_name = "a"} | count() > 2`,
			wantErr: true,
		},
		{
			name:    "multiple spanset filters / structural",
			query:   `{resource.service_name = "a"} >> {span.name = "b"}`,
			wantErr: true,
		},
		// --- rejected: unscoped attribute ---
		{
			name:    "unscoped attribute",
			query:   `{.service_name = "a"}`,
			wantErr: true,
		},
		// --- rejected: parent-scoped attributes are structural (ancestor span), not the
		// matched span's own resource/span attributes ---
		{
			name:    "parent resource attribute",
			query:   `{parent.resource.service_name = "a"}`,
			wantErr: true,
		},
		{
			name:    "parent span attribute",
			query:   `{parent.span.http.route = "/x"}`,
			wantErr: true,
		},
		// --- rejected: unparseable ---
		{
			name:    "invalid syntax",
			query:   `{resource.service_name = }`,
			wantErr: true,
		},
		{
			name:    "empty query",
			query:   ``,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedactionQuery(tc.query)
			if tc.wantErr && err == nil {
				t.Fatalf("query %q: expected rejection, got nil error", tc.query)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("query %q: expected acceptance, got error: %v", tc.query, err)
			}
		})
	}
}

// TestRedactionQueryExistenceForms covers the two negative spellings that are positive matches.
//
// `attr != nil` and `attr != ""` both select spans that HAVE the attribute, so they are not the
// complement match the `!=` rejection exists to prevent. `!=` against any other value still is.
func TestRedactionQueryExistenceForms(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		wantErr string
	}{
		{name: "attr != nil is an existence check", query: `{ span.foo != nil }`},
		{name: "attr != empty string", query: `{ span.foo != "" }`},
		{name: "empty string on the left", query: `{ "" != span.foo }`},
		{name: "existence combined with equality", query: `{ resource.service.name != nil && span.foo = "x" }`},
		{name: "existence combined with or", query: `{ span.a != "" || span.b != nil }`},

		{
			name:    "negation against a value is still refused",
			query:   `{ span.foo != "bar" }`,
			wantErr: `!= is allowed only against ""`,
		},
		{
			name:    "negation against a number is still refused",
			query:   `{ span.http.status_code != 500 }`,
			wantErr: `!= is allowed only against ""`,
		},
		{
			name:    "existence on an unscoped attribute is refused",
			query:   `{ .foo != nil }`,
			wantErr: "must be scoped to resource. or span.",
		},
		{
			name:    "existence on a parent-scoped attribute is refused",
			query:   `{ parent.span.foo != nil }`,
			wantErr: "must not be parent-scoped",
		},
		{
			name:    "ordered comparison against the empty string stays refused",
			query:   `{ span.foo > "" }`,
			wantErr: "not allowed in redaction query",
		},
		{
			name:    "not-regex stays refused",
			query:   `{ span.foo !~ "bar" }`,
			wantErr: "not allowed in redaction query",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedactionQuery(tc.query)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestEmptyStringComparisonExcludesMissingAttributes pins the TraceQL semantics that make
// `attr != ""` safe to allow, which live in another package and know nothing about redaction.
//
// The safety argument is: a span without the attribute resolves it to nil (traceql's
// Attribute.execute returns StaticNil when AttributeFor reports the attribute absent), and
// Static.NotEquals returns false whenever either operand is nil. So `attr != ""` selects spans where
// the attribute is present and non-empty, NOT the spans that lack it.
//
// If that nil rule ever changed -- to SQL-style propagation, or to treating nil as unequal to
// everything -- `attr != ""` would silently become a complement match on an irreversible delete, and
// the validator would keep accepting it. This test is what fails in that case. It deliberately
// asserts against traceql's exported behaviour rather than against the validator.
func TestEmptyStringComparisonExcludesMissingAttributes(t *testing.T) {
	var (
		missing  = traceql.NewStaticNil()
		empty    = traceql.NewStaticString("")
		nonEmpty = traceql.NewStaticString("something")
	)

	require.False(t, missing.NotEquals(&empty),
		`a missing attribute must not match attr != "" -- if it does, the selector becomes a complement match`)
	require.False(t, empty.NotEquals(&empty),
		`an empty value must not match attr != ""`)
	require.True(t, nonEmpty.NotEquals(&empty),
		`a present, non-empty value must match attr != "", or the selector matches nothing`)

	// The same rule is what makes `attr != nil` an existence check rather than a match on everything.
	require.False(t, missing.NotEquals(&missing), "nil is not unequal to nil")
}
