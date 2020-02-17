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

/*
This file holds tests for the in-memory fake for comparing it against a real Cloud Spanner.

By default it uses the Spanner client against the in-memory fake; set the
-test_db flag to "projects/P/instances/I/databases/D" to make it use a real
Cloud Spanner database instead. You may need to provide -timeout=5m too.
*/

package spannertest

import (
	"context"
	"flag"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	dbadmin "cloud.google.com/go/spanner/admin/database/apiv1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbadminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

var testDBFlag = flag.String("test_db", "", "Fully-qualified database name to test against; empty means use an in-memory fake.")

func dbName() string {
	if *testDBFlag != "" {
		return *testDBFlag
	}
	return "projects/fake-proj/instances/fake-instance/databases/fake-db"
}

func makeClient(t *testing.T) (*spanner.Client, *dbadmin.DatabaseAdminClient, func()) {
	// Despite the docs, this context is also used for auth,
	// so it needs to be long-lived.
	ctx := context.Background()

	if *testDBFlag != "" {
		t.Logf("Using real Spanner DB %s", *testDBFlag)
		dialOpt := option.WithGRPCDialOption(grpc.WithTimeout(5 * time.Second))
		client, err := spanner.NewClient(ctx, *testDBFlag, dialOpt)
		if err != nil {
			t.Fatalf("Connecting to %s: %v", *testDBFlag, err)
		}
		adminClient, err := dbadmin.NewDatabaseAdminClient(ctx, dialOpt)
		if err != nil {
			client.Close()
			t.Fatalf("Connecting DB admin client: %v", err)
		}
		return client, adminClient, func() { client.Close(); adminClient.Close() }
	}

	// Don't use SPANNER_EMULATOR_HOST because we need the raw connection for
	// the database admin client anyway.

	t.Logf("Using in-memory fake Spanner DB")
	srv, err := NewServer("localhost:0")
	if err != nil {
		t.Fatalf("Starting in-memory fake: %v", err)
	}
	srv.SetLogger(t.Logf)
	dialCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, srv.Addr, grpc.WithInsecure())
	if err != nil {
		srv.Close()
		t.Fatalf("Dialing in-memory fake: %v", err)
	}
	client, err := spanner.NewClient(ctx, dbName(), option.WithGRPCConn(conn))
	if err != nil {
		srv.Close()
		t.Fatalf("Connecting to in-memory fake: %v", err)
	}
	adminClient, err := dbadmin.NewDatabaseAdminClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		srv.Close()
		t.Fatalf("Connecting to in-memory fake DB admin: %v", err)
	}
	return client, adminClient, func() {
		client.Close()
		adminClient.Close()
		conn.Close()
		srv.Close()
	}
}

func TestSpannerBasics(t *testing.T) {
	client, adminClient, cleanup := makeClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Do a trivial query to verify the connection works.
	it := client.Single().Query(ctx, spanner.NewStatement("SELECT 1"))
	row, err := it.Next()
	if err != nil {
		t.Fatalf("Getting first row of trivial query: %v", err)
	}
	var value int64
	if err := row.Column(0, &value); err != nil {
		t.Fatalf("Decoding row data from trivial query: %v", err)
	}
	if value != 1 {
		t.Errorf("Trivial query gave %d, want 1", value)
	}
	// There shouldn't be a next row.
	_, err = it.Next()
	if err != iterator.Done {
		t.Errorf("Reading second row of trivial query gave %v, want iterator.Done", err)
	}
	it.Stop()

	// Drop any previous test table/index, and make a fresh one in a few stages.
	const tableName = "Characters"
	err = updateDDL(t, adminClient, "DROP INDEX AgeIndex")
	// NotFound is an acceptable failure mode here.
	if st, _ := status.FromError(err); st.Code() == codes.NotFound {
		err = nil
	}
	if err != nil {
		t.Fatalf("Dropping old index: %v", err)
	}
	err = updateDDL(t, adminClient, "DROP TABLE "+tableName)
	// NotFound is an acceptable failure mode here.
	if st, _ := status.FromError(err); st.Code() == codes.NotFound {
		err = nil
	}
	if err != nil {
		t.Fatalf("Dropping old table: %v", err)
	}
	err = updateDDL(t, adminClient,
		`CREATE TABLE `+tableName+` (
			FirstName STRING(20) NOT NULL,
			LastName STRING(20) NOT NULL,
			Alias STRING(MAX),
		) PRIMARY KEY (FirstName, LastName)`)
	if err != nil {
		t.Fatalf("Setting up fresh table: %v", err)
	}
	err = updateDDL(t, adminClient,
		`ALTER TABLE `+tableName+` ADD COLUMN Age INT64`,
		`CREATE INDEX AgeIndex ON `+tableName+` (Age DESC)`)
	if err != nil {
		t.Fatalf("Adding new column: %v", err)
	}

	// Insert some data.
	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert(tableName,
			[]string{"FirstName", "LastName", "Alias", "Age"},
			[]interface{}{"Steve", "Rogers", "Captain America", 101}),
		spanner.Insert(tableName,
			[]string{"LastName", "FirstName", "Age", "Alias"},
			[]interface{}{"Romanoff", "Natasha", 35, "Black Widow"}),
		spanner.Insert(tableName,
			[]string{"Age", "Alias", "FirstName", "LastName"},
			[]interface{}{49, "Iron Man", "Tony", "Stark"}),
		spanner.Insert(tableName,
			[]string{"FirstName", "Alias", "LastName"}, // no Age
			[]interface{}{"Clark", "Superman", "Kent"}),
		// Two rows with the same value in one column,
		// but with distinct primary keys.
		spanner.Insert(tableName,
			[]string{"FirstName", "LastName", "Alias"},
			[]interface{}{"Peter", "Parker", "Spider-Man"}),
		spanner.Insert(tableName,
			[]string{"FirstName", "LastName", "Alias"},
			[]interface{}{"Peter", "Quill", "Star-Lord"}),
	})
	if err != nil {
		t.Fatalf("Applying mutations: %v", err)
	}

	// Delete some data.
	_, err = client.Apply(ctx, []*spanner.Mutation{
		// Whoops. DC, not MCU.
		spanner.Delete(tableName, spanner.Key{"Clark", "Kent"}),
	})
	if err != nil {
		t.Fatalf("Applying mutations: %v", err)
	}

	// Read a single row.
	row, err = client.Single().ReadRow(ctx, tableName, spanner.Key{"Tony", "Stark"}, []string{"Alias", "Age"})
	if err != nil {
		t.Fatalf("Reading single row: %v", err)
	}
	var alias string
	var age int64
	if err := row.Columns(&alias, &age); err != nil {
		t.Fatalf("Decoding single row: %v", err)
	}
	if alias != "Iron Man" || age != 49 {
		t.Errorf(`Single row read gave (%q, %d), want ("Iron Man", 49)`, alias, age)
	}

	// Read all rows, and do a local age sum.
	rows := client.Single().Read(ctx, tableName, spanner.AllKeys(), []string{"Age"})
	var ageSum int64
	err = rows.Do(func(row *spanner.Row) error {
		var age spanner.NullInt64
		if err := row.Columns(&age); err != nil {
			return err
		}
		if age.Valid {
			ageSum += age.Int64
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Iterating over all row read: %v", err)
	}
	if want := int64(101 + 35 + 49); ageSum != want {
		t.Errorf("Age sum after iterating over all rows = %d, want %d", ageSum, want)
	}

	// Do a more complex query to find the aliases of the two oldest non-centenarian characters.
	stmt := spanner.NewStatement(`SELECT Alias FROM ` + tableName + ` WHERE Age < @ageLimit AND Alias IS NOT NULL ORDER BY Age DESC LIMIT @limit`)
	stmt.Params = map[string]interface{}{
		"ageLimit": 100,
		"limit":    2,
	}
	rows = client.Single().Query(ctx, stmt)
	var oldFolk []string
	err = rows.Do(func(row *spanner.Row) error {
		var alias string
		if err := row.Columns(&alias); err != nil {
			return err
		}
		oldFolk = append(oldFolk, alias)
		return nil
	})
	if err != nil {
		t.Fatalf("Iterating over complex query: %v", err)
	}
	if want := []string{"Iron Man", "Black Widow"}; !reflect.DeepEqual(oldFolk, want) {
		t.Errorf("Complex query results = %v, want %v", oldFolk, want)
	}

	// Apply an update.
	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.Update(tableName,
			[]string{"FirstName", "LastName", "Age"},
			[]interface{}{"Steve", "Rogers", 102}),
	})
	if err != nil {
		t.Fatalf("Applying mutations: %v", err)
	}
	row, err = client.Single().ReadRow(ctx, tableName, spanner.Key{"Steve", "Rogers"}, []string{"Age"})
	if err != nil {
		t.Fatalf("Reading single row: %v", err)
	}
	if err := row.Columns(&age); err != nil {
		t.Fatalf("Decoding single row: %v", err)
	}
	if age != 102 {
		t.Errorf("After updating Captain America, age = %d, want 102", age)
	}

	// Do a query where the result type isn't deducible from the first row.
	stmt = spanner.NewStatement(`SELECT Age FROM ` + tableName + ` WHERE FirstName = "Peter"`)
	rows = client.Single().Query(ctx, stmt)
	var nullPeters int
	err = rows.Do(func(row *spanner.Row) error {
		var age spanner.NullInt64
		if err := row.Column(0, &age); err != nil {
			return err
		}
		if age.Valid {
			t.Errorf("Got non-NULL Age %d for a Peter", age.Int64)
		} else {
			nullPeters++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Counting Peters with NULL Ages: %v", err)
	}
	if nullPeters != 2 {
		t.Errorf("Found %d Peters with NULL Ages, want 2", nullPeters)
	}

	// Check handling of array types.
	err = updateDDL(t, adminClient, `ALTER TABLE `+tableName+` ADD COLUMN Allies ARRAY<STRING(20)>`)
	if err != nil {
		t.Fatalf("Adding new array-typed column: %v", err)
	}
	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.Update(tableName,
			[]string{"FirstName", "LastName", "Allies"},
			[]interface{}{"Steve", "Rogers", []string{}}),
		spanner.Update(tableName,
			[]string{"FirstName", "LastName", "Allies"},
			[]interface{}{"Tony", "Stark", []string{"Black Widow", "Spider-Man"}}),
	})
	if err != nil {
		t.Fatalf("Applying mutations: %v", err)
	}
	row, err = client.Single().ReadRow(ctx, tableName, spanner.Key{"Tony", "Stark"}, []string{"Allies"})
	if err != nil {
		t.Fatalf("Reading row with array value: %v", err)
	}
	var names []string
	if err := row.Column(0, &names); err != nil {
		t.Fatalf("Unpacking array value: %v", err)
	}
	if want := []string{"Black Widow", "Spider-Man"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Read array value: got %q, want %q", names, want)
	}
	row, err = client.Single().ReadRow(ctx, tableName, spanner.Key{"Steve", "Rogers"}, []string{"Allies"})
	if err != nil {
		t.Fatalf("Reading row with empty array value: %v", err)
	}
	if err := row.Column(0, &names); err != nil {
		t.Fatalf("Unpacking empty array value: %v", err)
	}
	if len(names) > 0 {
		t.Errorf("Read empty array value: got %q", names)
	}
}

func updateDDL(t *testing.T, adminClient *dbadmin.DatabaseAdminClient, statements ...string) error {
	t.Helper()
	ctx := context.Background()
	t.Logf("DDL update: %q", statements)
	op, err := adminClient.UpdateDatabaseDdl(ctx, &dbadminpb.UpdateDatabaseDdlRequest{
		Database:   dbName(),
		Statements: statements,
	})
	if err != nil {
		t.Fatalf("Starting DDL update: %v", err)
	}
	return op.Wait(ctx)
}
