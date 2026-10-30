package backendscheduler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The redaction query selector accepts only a single spanset filter over resource.*/span.*
// attributes, ANDed with at least one = (or || of =) condition. Everything
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

// TestRedactionQueryExistenceForms covers attr != nil, the negative spelling that is a
// positive match: the grammar rewrites it into a unary existence check, so it is not the
// complement match the `!=` rejection otherwise exists to prevent.
func TestRedactionQueryExistenceForms(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		wantErr string
	}{
		{name: "existence combined with equality", query: `{ resource.service.name != nil && span.foo = "x" }`},

		{
			name:    "existence alone has no positive filter",
			query:   `{ span.foo != nil }`,
			wantErr: "at least one = comparison",
		},
		{
			name:    "existence combined with or",
			query:   `{ span.a != nil || span.b != nil }`,
			wantErr: "at least one = comparison",
		},

		{
			name:    "negation against a value is still refused",
			query:   `{ span.foo != "bar" }`,
			wantErr: "not allowed in redaction query",
		},
		{
			name:    "negation against a number is still refused",
			query:   `{ span.http.status_code != 500 }`,
			wantErr: "not allowed in redaction query",
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
			name:    "ordered comparison stays refused",
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

// TestRedactionQueryPositiveFilter covers the shape rule: the query is an && of conditions, at least
// one of them an = comparison or an || of = comparisons; the rest may combine comparisons freely.
func TestRedactionQueryPositiveFilter(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		wantErr string
	}{
		{name: "or of equalities across attributes", query: `{ resource.namespace = "checkout" || span.tenantref = "acme" }`},
		{name: "or group anded with existence", query: `{ (resource.service.name = "a" || resource.service.name = "b") && span.foo != nil }`},
		{name: "existence before the positive filter", query: `{ span.foo != nil && span.bar != nil && resource.service.name = "a" }`},
		{name: "or of existence anded with equality", query: `{ resource.service.name = "a" && (span.a != nil || span.b != nil) }`},
		{name: "and of existence anded with equality", query: `{ resource.service.name = "a" && (span.a != nil && span.b != nil) }`},
		{name: "or of mixed groups anded with equality", query: `{ resource.service.name = "a" && (span.a != nil && span.x = "" || span.b != nil && span.x = "y") }`},

		{
			name:    "or mixing equality and existence",
			query:   `{ resource.service.name = "a" || span.foo != nil }`,
			wantErr: "at least one = comparison",
		},
		{
			name:    "and nested inside or",
			query:   `{ resource.service.name = "a" && span.foo = "x" || resource.service.name = "b" }`,
			wantErr: "at least one = comparison",
		},
		{
			name:    "unparenthesised or binds looser than and",
			query:   `{ resource.service.name = "a" && span.a != nil || span.b != nil }`,
			wantErr: "parenthesised",
		},
		{
			name:    "only existence checks",
			query:   `{ span.a != nil && span.b != nil }`,
			wantErr: "at least one = comparison",
		},
		{
			name:    "unsupported operator nested in an or",
			query:   `{ resource.service.name = "a" && (span.a != nil || span.b > 1) }`,
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
