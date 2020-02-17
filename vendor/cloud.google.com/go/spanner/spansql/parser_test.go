/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spansql

import (
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		in   string
		want Query
	}{
		{`SELECT 17`, Query{Select: Select{List: []Expr{IntegerLiteral(17)}}}},
		{`SELECT Alias FROM Characters WHERE Age < @ageLimit AND Alias IS NOT NULL ORDER BY Age DESC LIMIT @limit` + "\n\t",
			Query{
				Select: Select{
					List: []Expr{ID("Alias")},
					From: []SelectFrom{{
						Table: "Characters",
					}},
					Where: LogicalOp{
						Op: And,
						LHS: ComparisonOp{
							LHS: ID("Age"),
							Op:  Lt,
							RHS: Param("ageLimit"),
						},
						RHS: IsOp{
							LHS: ID("Alias"),
							Neg: true,
							RHS: Null,
						},
					},
				},
				Order: []Order{{
					Expr: ID("Age"),
					Desc: true,
				}},
				Limit: Param("limit"),
			},
		},
		{`SELECT COUNT(*) FROM Packages`,
			Query{
				Select: Select{
					List: []Expr{
						Func{
							Name: "COUNT",
							Args: []Expr{Star},
						},
					},
					From: []SelectFrom{{Table: "Packages"}},
				},
			},
		},
	}
	for _, test := range tests {
		got, err := ParseQuery(test.in)
		if err != nil {
			t.Errorf("ParseQuery(%q): %v", test.in, err)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ParseQuery(%q) incorrect.\n got %#v\nwant %#v", test.in, got, test.want)
		}
	}
}

func TestParseExpr(t *testing.T) {
	tests := []struct {
		in   string
		want Expr
	}{
		{`17`, IntegerLiteral(17)},
		{`-1`, IntegerLiteral(-1)},
		{`0xf00d`, IntegerLiteral(0xf00d)},
		{`-0xbeef`, IntegerLiteral(-0xbeef)},
		{`123.456e-67`, FloatLiteral(123.456e-67)},
		{`.1E4`, FloatLiteral(0.1e4)},
		{`58.`, FloatLiteral(58)},
		{`4e2`, FloatLiteral(4e2)},
		{`Count > 0`, ComparisonOp{LHS: ID("Count"), Op: Gt, RHS: IntegerLiteral(0)}},
		{`Name LIKE "Eve %"`, ComparisonOp{LHS: ID("Name"), Op: Like, RHS: StringLiteral("Eve %")}},
		{`Speech NOT LIKE "_oo"`, ComparisonOp{LHS: ID("Speech"), Op: NotLike, RHS: StringLiteral("_oo")}},
		{`A AND NOT B`, LogicalOp{LHS: ID("A"), Op: And, RHS: LogicalOp{Op: Not, RHS: ID("B")}}},
		{`X BETWEEN Y AND Z`, ComparisonOp{LHS: ID("X"), Op: Between, RHS: ID("Y"), RHS2: ID("Z")}},

		// OR is lower precedence than AND.
		{`A AND B OR C`, LogicalOp{LHS: LogicalOp{LHS: ID("A"), Op: And, RHS: ID("B")}, Op: Or, RHS: ID("C")}},
		{`A OR B AND C`, LogicalOp{LHS: ID("A"), Op: Or, RHS: LogicalOp{LHS: ID("B"), Op: And, RHS: ID("C")}}},
		// Parens to override normal precedence.
		{`A OR (B AND C)`, LogicalOp{LHS: ID("A"), Op: Or, RHS: Paren{Expr: LogicalOp{LHS: ID("B"), Op: And, RHS: ID("C")}}}},

		// This is the same as the WHERE clause from the test in ParseQuery.
		{`Age < @ageLimit AND Alias IS NOT NULL`,
			LogicalOp{
				LHS: ComparisonOp{LHS: ID("Age"), Op: Lt, RHS: Param("ageLimit")},
				Op:  And,
				RHS: IsOp{LHS: ID("Alias"), Neg: true, RHS: Null},
			},
		},

		// This used to be broken because the lexer didn't reset the token type.
		{`C < "whelp" AND D IS NOT NULL`,
			LogicalOp{
				LHS: ComparisonOp{LHS: ID("C"), Op: Lt, RHS: StringLiteral("whelp")},
				Op:  And,
				RHS: IsOp{LHS: ID("D"), Neg: true, RHS: Null},
			},
		},

		// Reserved keywords.
		{`TRUE AND FALSE`, LogicalOp{LHS: True, Op: And, RHS: False}},
		{`NULL`, Null},
	}
	for _, test := range tests {
		p := newParser(test.in)
		got, err := p.parseExpr()
		if err != nil {
			t.Errorf("[%s]: %v", test.in, err)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("[%s]: incorrect parse\n got <%T> %#v\nwant <%T> %#v", test.in, got, got, test.want, test.want)
		}
		if p.s != "" {
			t.Errorf("[%s]: Unparsed [%s]", test.in, p.s)
		}
	}
}

func TestParseDDL(t *testing.T) {
	tests := []struct {
		in   string
		want DDL
	}{
		{`CREATE TABLE FooBar (
			System STRING(MAX) NOT NULL,  # This is a comment.
			RepoPath STRING(MAX) NOT NULL,  -- This is another comment.
			Count INT64, /* This is a
			              * multiline comment. */
		) PRIMARY KEY(System, RepoPath);
		CREATE UNIQUE INDEX MyFirstIndex ON FooBar (
			Count DESC
		) STORING (Count), INTERLEAVE IN SomeTable;
		CREATE TABLE FooBarAux (
			System STRING(MAX) NOT NULL,
			RepoPath STRING(MAX) NOT NULL,
			Author STRING(MAX) NOT NULL,
		) PRIMARY KEY(System, RepoPath, Author),
		  INTERLEAVE IN PARENT FooBar ON DELETE CASCADE;

		ALTER TABLE FooBar ADD COLUMN TZ BYTES(20);
		ALTER TABLE FooBar DROP COLUMN TZ;
		ALTER TABLE FooBar SET ON DELETE NO ACTION;

		DROP INDEX MyFirstIndex;
		DROP TABLE FooBar;

		CREATE TABLE NonScalars (
			Dummy INT64 NOT NULL,
			Ids ARRAY<INT64>,
			Names ARRAY<STRING(MAX)>,
		) PRIMARY KEY (Dummy);
		`, DDL{List: []DDLStmt{
			CreateTable{
				Name: "FooBar",
				Columns: []ColumnDef{
					{Name: "System", Type: Type{Base: String, Len: MaxLen}, NotNull: true},
					{Name: "RepoPath", Type: Type{Base: String, Len: MaxLen}, NotNull: true},
					{Name: "Count", Type: Type{Base: Int64}},
				},
				PrimaryKey: []KeyPart{
					{Column: "System"},
					{Column: "RepoPath"},
				},
			},
			CreateIndex{
				Name:       "MyFirstIndex",
				Table:      "FooBar",
				Columns:    []KeyPart{{Column: "Count", Desc: true}},
				Unique:     true,
				Storing:    []string{"Count"},
				Interleave: "SomeTable",
			},
			CreateTable{
				Name: "FooBarAux",
				Columns: []ColumnDef{
					{Name: "System", Type: Type{Base: String, Len: MaxLen}, NotNull: true},
					{Name: "RepoPath", Type: Type{Base: String, Len: MaxLen}, NotNull: true},
					{Name: "Author", Type: Type{Base: String, Len: MaxLen}, NotNull: true},
				},
				PrimaryKey: []KeyPart{
					{Column: "System"},
					{Column: "RepoPath"},
					{Column: "Author"},
				},
				Interleave: &Interleave{
					Parent:   "FooBar",
					OnDelete: CascadeOnDelete,
				},
			},
			AlterTable{Name: "FooBar", Alteration: AddColumn{
				Def: ColumnDef{Name: "TZ", Type: Type{Base: Bytes, Len: 20}},
			}},
			AlterTable{Name: "FooBar", Alteration: DropColumn{Name: "TZ"}},
			AlterTable{Name: "FooBar", Alteration: SetOnDelete{Action: NoActionOnDelete}},
			DropIndex{Name: "MyFirstIndex"},
			DropTable{Name: "FooBar"},
			CreateTable{
				Name: "NonScalars",
				Columns: []ColumnDef{
					{Name: "Dummy", Type: Type{Base: Int64}, NotNull: true},
					{Name: "Ids", Type: Type{Array: true, Base: Int64}},
					{Name: "Names", Type: Type{Array: true, Base: String, Len: MaxLen}},
				},
				PrimaryKey: []KeyPart{{Column: "Dummy"}},
			},
		}}},
		// No trailing comma:
		{`ALTER TABLE T ADD COLUMN C2 INT64`, DDL{List: []DDLStmt{
			AlterTable{Name: "T", Alteration: AddColumn{
				Def: ColumnDef{Name: "C2", Type: Type{Base: Int64}},
			}},
		}}},
	}
	for _, test := range tests {
		got, err := ParseDDL(test.in)
		if err != nil {
			t.Errorf("ParseDDL(%q): %v", test.in, err)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ParseDDL(%q) incorrect.\n got %v\nwant %v", test.in, got, test.want)
		}
	}
}

func TestParseFailures(t *testing.T) {
	expr := func(p *parser) error {
		_, err := p.parseExpr()
		return err
	}

	tests := []struct {
		f    func(p *parser) error
		in   string
		desc string
	}{
		{expr, `0b337`, "binary literal"},
		{expr, `"foo\`, "unterminated string"},
		{expr, `"foo" AND "bar"`, "logical operation on string literals"},
	}
	for _, test := range tests {
		p := newParser(test.in)
		if test.f(p) == nil && p.Rem() == "" {
			t.Errorf("%s: parsing [%s] succeeded, should have failed", test.desc, test.in)
		}
	}
}
