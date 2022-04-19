// Copyright 2021 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"cloud.google.com/go/alloydbconn/driver/pgxv4"
)

var (
	alloydbConnName = flag.String("alloydb_conn_name", os.Getenv("ALLOYDB_CONNECTION_NAME"), "AlloyDB instance connection name, in the form of 'project:region:instance'.")
	alloydbUser     = flag.String("alloydb_user", os.Getenv("ALLOYDB_USER"), "Name of database user.")
	alloydbPass     = flag.String("alloydb_pass", os.Getenv("ALLOYDB_PASS"), "Password for the database user; be careful when entering a password on the command line (it may go into your terminal's history).")
	alloydbDB       = flag.String("alloydb_db", os.Getenv("ALLOYDB_DB"), "Name of the database to connect to.")
)

func requirePostgresVars(t *testing.T) {
	switch "" {
	case *alloydbConnName:
		t.Fatal("'alloydb_conn_name' not set")
	case *alloydbUser:
		t.Fatal("'alloydb_user' not set")
	case *alloydbPass:
		t.Fatal("'alloydb_pass' not set")
	case *alloydbDB:
		t.Fatal("'alloydb_db' not set")
	}
}

func TestPostgresTCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	cleanup, err := pgxv4.RegisterDriver("alloydb1")
	if err != nil {
		t.Fatalf("failed to register driver: %v", err)
	}
	defer cleanup()

	dsn := fmt.Sprintf("host=%v user=%v password=%v database=%v sslmode=disable",
		*alloydbConnName, *alloydbUser, *alloydbPass, *alloydbDB)
	proxyConnTest(t, []string{*alloydbConnName}, "alloydb1", dsn)
}

func TestPostgresAuthWithToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	_, isFlex := os.LookupEnv("FLEX")
	if isFlex {
		t.Skip("disabling until we migrate tests to Kokoro")
	}
	requirePostgresVars(t)
	cleanup, err := pgxv4.RegisterDriver("alloydb2")
	if err != nil {
		t.Fatalf("failed to register driver: %v", err)
	}
	defer cleanup()
	tok, _, cleanup2 := removeAuthEnvVar(t)
	defer cleanup2()

	dsn := fmt.Sprintf("host=%v user=%v password=%v database=%v sslmode=disable",
		*alloydbConnName, *alloydbUser, *alloydbPass, *alloydbDB)
	proxyConnTest(t,
		[]string{"--token", tok.AccessToken, *alloydbConnName},
		"alloydb2", dsn)
}

func TestPostgresAuthWithCredentialsFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	_, isFlex := os.LookupEnv("FLEX")
	if isFlex {
		t.Skip("disabling until we migrate tests to Kokoro")
	}
	requirePostgresVars(t)
	cleanup, err := pgxv4.RegisterDriver("alloydb3")
	if err != nil {
		t.Fatalf("failed to register driver: %v", err)
	}
	defer cleanup()
	_, path, cleanup2 := removeAuthEnvVar(t)
	defer cleanup2()

	dsn := fmt.Sprintf("host=%v user=%v password=%v database=%v sslmode=disable",
		*alloydbConnName, *alloydbUser, *alloydbPass, *alloydbDB)
	proxyConnTest(t,
		[]string{"--credentials-file", path, *alloydbConnName},
		"alloydb3", dsn)
}
