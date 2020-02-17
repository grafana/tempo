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

func TestSQL(t *testing.T) {
	reparseDDL := func(s string) (interface{}, error) {
		ddl, err := ParseDDLStmt(s)
		return ddl, err
	}
	reparseQuery := func(s string) (interface{}, error) {
		q, err := ParseQuery(s)
		return q, err
	}
	reparseExpr := func(s string) (interface{}, error) {
		e, err := newParser(s).parseExpr()
		return e, err
	}

	tests := []struct {
		data    interface{ SQL() string }
		sql     string
		reparse func(string) (interface{}, error)
	}{
		{
			CreateTable{
				Name: "Ta",
				Columns: []ColumnDef{
					{Name: "Ca", Type: Type{Base: Bool}, NotNull: true},
					{Name: "Cb", Type: Type{Base: Int64}},
					{Name: "Cc", Type: Type{Base: Float64}},
					{Name: "Cd", Type: Type{Base: String, Len: 17}},
					{Name: "Ce", Type: Type{Base: String, Len: MaxLen}},
					{Name: "Cf", Type: Type{Base: Bytes, Len: 4711}},
					{Name: "Cg", Type: Type{Base: Bytes, Len: MaxLen}},
					{Name: "Ch", Type: Type{Base: Date}},
					{Name: "Ci", Type: Type{Base: Timestamp}},
					{Name: "Cj", Type: Type{Array: true, Base: Int64}},
					{Name: "Ck", Type: Type{Array: true, Base: String, Len: MaxLen}},
				},
				PrimaryKey: []KeyPart{
					{Column: "Ca"},
					{Column: "Cb", Desc: true},
				},
			},
			`CREATE TABLE Ta (
  Ca BOOL NOT NULL,
  Cb INT64,
  Cc FLOAT64,
  Cd STRING(17),
  Ce STRING(MAX),
  Cf BYTES(4711),
  Cg BYTES(MAX),
  Ch DATE,
  Ci TIMESTAMP,
  Cj ARRAY<INT64>,
  Ck ARRAY<STRING(MAX)>,
) PRIMARY KEY(Ca, Cb DESC)`,
			reparseDDL,
		},
		{
			CreateTable{
				Name: "Tsub",
				Columns: []ColumnDef{
					{Name: "SomeId", Type: Type{Base: Int64}, NotNull: true},
					{Name: "OtherId", Type: Type{Base: Int64}, NotNull: true},
				},
				PrimaryKey: []KeyPart{
					{Column: "SomeId"},
					{Column: "OtherId"},
				},
				Interleave: &Interleave{
					Parent:   "Ta",
					OnDelete: CascadeOnDelete,
				},
			},
			`CREATE TABLE Tsub (
  SomeId INT64 NOT NULL,
  OtherId INT64 NOT NULL,
) PRIMARY KEY(SomeId, OtherId),
  INTERLEAVE IN PARENT Ta ON DELETE CASCADE`,
			reparseDDL,
		},
		{
			DropTable{
				Name: "Ta",
			},
			"DROP TABLE Ta",
			reparseDDL,
		},
		{
			CreateIndex{
				Name:  "Ia",
				Table: "Ta",
				Columns: []KeyPart{
					{Column: "Ca"},
					{Column: "Cb", Desc: true},
				},
			},
			"CREATE INDEX Ia ON Ta(Ca, Cb DESC)",
			reparseDDL,
		},
		{
			DropIndex{
				Name: "Ia",
			},
			"DROP INDEX Ia",
			reparseDDL,
		},
		{
			AlterTable{
				Name:       "Ta",
				Alteration: AddColumn{Def: ColumnDef{Name: "Ca", Type: Type{Base: Bool}}},
			},
			"ALTER TABLE Ta ADD COLUMN Ca BOOL",
			reparseDDL,
		},
		{
			AlterTable{
				Name:       "Ta",
				Alteration: DropColumn{Name: "Ca"},
			},
			"ALTER TABLE Ta DROP COLUMN Ca",
			reparseDDL,
		},
		{
			AlterTable{
				Name:       "Ta",
				Alteration: SetOnDelete{Action: NoActionOnDelete},
			},
			"ALTER TABLE Ta SET ON DELETE NO ACTION",
			reparseDDL,
		},
		{
			AlterTable{
				Name:       "Ta",
				Alteration: SetOnDelete{Action: CascadeOnDelete},
			},
			"ALTER TABLE Ta SET ON DELETE CASCADE",
			reparseDDL,
		},
		{
			Query{
				Select: Select{
					List: []Expr{ID("A"), ID("B")},
					From: []SelectFrom{{Table: "Table"}},
					Where: LogicalOp{
						LHS: ComparisonOp{
							LHS: ID("C"),
							Op:  Lt,
							RHS: StringLiteral("whelp"),
						},
						Op: And,
						RHS: IsOp{
							LHS: ID("D"),
							Neg: true,
							RHS: Null,
						},
					},
				},
				Order: []Order{{Expr: ID("OCol"), Desc: true}},
				Limit: IntegerLiteral(1000),
			},
			`SELECT A, B FROM Table WHERE C < "whelp" AND D IS NOT NULL ORDER BY OCol DESC LIMIT 1000`,
			reparseQuery,
		},
		{
			Query{
				Select: Select{
					List: []Expr{IntegerLiteral(7)},
				},
			},
			`SELECT 7`,
			reparseQuery,
		},
		{
			ComparisonOp{LHS: ID("X"), Op: NotBetween, RHS: ID("Y"), RHS2: ID("Z")},
			`X NOT BETWEEN Y AND Z`,
			reparseExpr,
		},
	}
	for _, test := range tests {
		sql := test.data.SQL()
		if sql != test.sql {
			t.Errorf("%v.SQL() wrong.\n got %s\nwant %s", test.data, sql, test.sql)
			continue
		}

		// As a sanity check, confirm that parsing the SQL produces the original input.
		data, err := test.reparse(sql)
		if err != nil {
			t.Errorf("Reparsing %q: %v", sql, err)
			continue
		}
		if !reflect.DeepEqual(data, test.data) {
			t.Errorf("Reparsing %q wrong.\n got %v\nwant %v", sql, data, test.data)
		}
	}
}
