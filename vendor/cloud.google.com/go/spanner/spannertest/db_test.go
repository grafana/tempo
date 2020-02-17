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

package spannertest

import (
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"cloud.google.com/go/spanner/spansql"
)

var stdTestTable = spansql.CreateTable{
	Name: "Staff",
	Columns: []spansql.ColumnDef{
		{Name: "Tenure", Type: spansql.Type{Base: spansql.Int64}},
		{Name: "ID", Type: spansql.Type{Base: spansql.Int64}},
		{Name: "Name", Type: spansql.Type{Base: spansql.String}},
		{Name: "Cool", Type: spansql.Type{Base: spansql.Bool}},
		{Name: "Height", Type: spansql.Type{Base: spansql.Float64}},
	},
	PrimaryKey: []spansql.KeyPart{{Column: "Name"}, {Column: "ID"}},
}

func TestTableCreation(t *testing.T) {
	var db database
	st := db.ApplyDDL(stdTestTable)
	if st.Code() != codes.OK {
		t.Fatalf("Creating table: %v", st.Err())
	}

	// Snoop inside to check that it was constructed correctly.
	got, ok := db.tables["Staff"]
	if !ok {
		t.Fatal("Table didn't get registered")
	}
	want := table{
		cols: []colInfo{
			{Name: "Name", Type: spansql.Type{Base: spansql.String}},
			{Name: "ID", Type: spansql.Type{Base: spansql.Int64}},
			{Name: "Tenure", Type: spansql.Type{Base: spansql.Int64}},
			{Name: "Cool", Type: spansql.Type{Base: spansql.Bool}},
			{Name: "Height", Type: spansql.Type{Base: spansql.Float64}},
		},
		colIndex: map[string]int{
			"Tenure": 2, "ID": 1, "Cool": 3, "Name": 0, "Height": 4,
		},
		pkCols: 2,
	}
	if !reflect.DeepEqual(got.cols, want.cols) {
		t.Errorf("table.cols incorrect.\n got %v\nwant %v", got.cols, want.cols)
	}
	if !reflect.DeepEqual(got.colIndex, want.colIndex) {
		t.Errorf("table.colIndex incorrect.\n got %v\nwant %v", got.colIndex, want.colIndex)
	}
	if got.pkCols != want.pkCols {
		t.Errorf("table.pkCols incorrect.\n got %d\nwant %d", got.pkCols, want.pkCols)
	}
}

func TestTableData(t *testing.T) {
	var db database
	st := db.ApplyDDL(stdTestTable)
	if st.Code() != codes.OK {
		t.Fatalf("Creating table: %v", st.Err())
	}

	// Insert a subset of columns.
	err := db.Insert("Staff", []string{"ID", "Name", "Tenure", "Height"}, []*structpb.ListValue{
		// int64 arrives as a decimal string.
		listV(stringV("1"), stringV("Jack"), stringV("10"), floatV(1.85)),
		listV(stringV("2"), stringV("Daniel"), stringV("11"), floatV(1.83)),
	})
	if err != nil {
		t.Fatalf("Inserting data: %v", err)
	}
	// Insert a different set of columns.
	err = db.Insert("Staff", []string{"Name", "ID", "Cool", "Tenure", "Height"}, []*structpb.ListValue{
		listV(stringV("Sam"), stringV("3"), boolV(false), stringV("9"), floatV(1.75)),
		listV(stringV("Teal'c"), stringV("4"), boolV(true), stringV("8"), floatV(1.91)),
		listV(stringV("George"), stringV("5"), nullV(), stringV("6"), floatV(1.73)),
		listV(stringV("Harry"), stringV("6"), boolV(true), nullV(), nullV()),
	})
	if err != nil {
		t.Fatalf("Inserting more data: %v", err)
	}
	// Delete that last one.
	err = db.Delete("Staff", []*structpb.ListValue{listV(stringV("Harry"), stringV("6"))}, false)
	if err != nil {
		t.Fatalf("Deleting a row: %v", err)
	}
	// Turns out this guy isn't cool after all.
	err = db.Update("Staff", []string{"Name", "ID", "Cool"}, []*structpb.ListValue{
		// Missing columns should be left alone.
		listV(stringV("Daniel"), stringV("2"), boolV(false)),
	})
	if err != nil {
		t.Fatalf("Updating a row: %v", err)
	}

	// Read some specific keys.
	ri, err := db.Read("Staff", []string{"Name", "Tenure"}, []*structpb.ListValue{
		listV(stringV("George"), stringV("5")),
		listV(stringV("Harry"), stringV("6")), // should be silently ignored.
		listV(stringV("Sam"), stringV("3")),
	}, 0)
	if err != nil {
		t.Fatalf("Reading keys: %v", err)
	}
	all := slurp(ri)
	wantAll := [][]interface{}{
		{"George", int64(6)},
		{"Sam", int64(9)},
	}
	if !reflect.DeepEqual(all, wantAll) {
		t.Errorf("Read data wrong.\n got %v\nwant %v", all, wantAll)
	}

	// Read a subset of all rows, with a limit.
	ri, err = db.ReadAll("Staff", []string{"Tenure", "Name", "Height"}, 4)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	wantCols := []colInfo{
		{Name: "Tenure", Type: spansql.Type{Base: spansql.Int64}},
		{Name: "Name", Type: spansql.Type{Base: spansql.String}},
		{Name: "Height", Type: spansql.Type{Base: spansql.Float64}},
	}
	if !reflect.DeepEqual(ri.Cols, wantCols) {
		t.Errorf("ReadAll cols wrong.\n got %v\nwant %v", ri.Cols, wantCols)
	}
	all = slurp(ri)
	wantAll = [][]interface{}{
		{int64(10), "Jack", 1.85},
		{int64(11), "Daniel", 1.83},
		{int64(9), "Sam", 1.75},
		{int64(8), "Teal'c", 1.91},
	}
	if !reflect.DeepEqual(all, wantAll) {
		t.Errorf("ReadAll data wrong.\n got %v\nwant %v", all, wantAll)
	}

	// Add a DATE column, populate it with some data.
	st = db.ApplyDDL(spansql.AlterTable{
		Name: "Staff",
		Alteration: spansql.AddColumn{Def: spansql.ColumnDef{
			Name: "FirstSeen",
			Type: spansql.Type{Base: spansql.Date},
		}},
	})
	if st.Code() != codes.OK {
		t.Fatalf("Adding column: %v", st.Err())
	}
	err = db.Update("Staff", []string{"Name", "ID", "FirstSeen"}, []*structpb.ListValue{
		listV(stringV("Jack"), stringV("1"), stringV("1994-10-28")),
		listV(stringV("Daniel"), stringV("2"), stringV("1994-10-28")),
		listV(stringV("George"), stringV("5"), stringV("1997-07-27")),
	})
	if err != nil {
		t.Fatalf("Updating rows: %v", err)
	}

	// Do some complex queries.
	tests := []struct {
		q      string
		params queryParams
		want   [][]interface{}
	}{
		{
			`SELECT 17, "sweet", TRUE AND FALSE, NULL`,
			nil,
			[][]interface{}{{int64(17), "sweet", false, nil}},
		},
		{
			`SELECT Name FROM Staff WHERE Cool`,
			nil,
			[][]interface{}{{"Teal'c"}},
		},
		{
			`SELECT ID FROM Staff WHERE Cool IS NOT NULL ORDER BY ID DESC`,
			nil,
			[][]interface{}{{int64(4)}, {int64(3)}, {int64(2)}},
		},
		{
			`SELECT Name, Tenure FROM Staff WHERE Cool IS NULL OR Cool ORDER BY Name LIMIT 2`,
			nil,
			[][]interface{}{
				{"George", int64(6)},
				{"Jack", int64(10)},
			},
		},
		{
			`SELECT Name, ID FROM Staff WHERE @min <= Tenure AND Tenure < @lim ORDER BY Cool, Name DESC LIMIT @numResults`,
			queryParams{"min": int64(9), "lim": int64(11), "numResults": "100"},
			[][]interface{}{
				{"Jack", int64(1)},
				{"Sam", int64(3)},
			},
		},
		{
			// Expression in SELECT list.
			`SELECT Name, Cool IS NOT NULL FROM Staff WHERE Tenure > 8 ORDER BY NOT Cool, Name`,
			nil,
			[][]interface{}{
				{"Daniel", true}, // Daniel has Cool==true
				{"Jack", false},  // Jack has NULL Cool
				{"Sam", true},    // Sam has Cool==false
			},
		},
		{
			`SELECT Name, Height FROM Staff ORDER BY Height DESC LIMIT 2`,
			nil,
			[][]interface{}{
				{"Teal'c", 1.91},
				{"Jack", 1.85},
			},
		},
		{
			`SELECT Name FROM Staff WHERE Name LIKE "J%k" OR Name LIKE "_am"`,
			nil,
			[][]interface{}{
				{"Jack"},
				{"Sam"},
			},
		},
		{
			`SELECT Name, Height FROM Staff WHERE Height BETWEEN @min AND @max ORDER BY Height DESC`,
			queryParams{"min": 1.75, "max": 1.85},
			[][]interface{}{
				{"Jack", 1.85},
				{"Daniel", 1.83},
				{"Sam", 1.75},
			},
		},
		{
			`SELECT COUNT(*) FROM Staff WHERE Name < "T"`,
			nil,
			[][]interface{}{
				{int64(4)},
			},
		},
		{
			`SELECT Name FROM Staff WHERE FirstSeen >= @min`,
			queryParams{"min": "1996-01-01"},
			[][]interface{}{
				{"George"},
			},
		},
	}
	for _, test := range tests {
		q, err := spansql.ParseQuery(test.q)
		if err != nil {
			t.Errorf("ParseQuery(%q): %v", test.q, err)
			continue
		}
		ri, err := db.Query(q, test.params)
		if err != nil {
			t.Errorf("Query(%q, %v): %v", test.q, test.params, err)
			continue
		}
		all := slurp(ri)
		if !reflect.DeepEqual(all, test.want) {
			t.Errorf("Results from Query(%q, %v) are wrong.\n got %v\nwant %v", test.q, test.params, all, test.want)
		}
	}
}

func slurp(ri *resultIter) (all [][]interface{}) {
	for {
		row, ok := ri.Next()
		if !ok {
			return
		}
		all = append(all, row)
	}
}

func listV(vs ...*structpb.Value) *structpb.ListValue { return &structpb.ListValue{Values: vs} }
func stringV(s string) *structpb.Value                { return &structpb.Value{Kind: &structpb.Value_StringValue{s}} }
func floatV(f float64) *structpb.Value                { return &structpb.Value{Kind: &structpb.Value_NumberValue{f}} }
func boolV(b bool) *structpb.Value                    { return &structpb.Value{Kind: &structpb.Value_BoolValue{b}} }
func nullV() *structpb.Value                          { return &structpb.Value{Kind: &structpb.Value_NullValue{}} }
