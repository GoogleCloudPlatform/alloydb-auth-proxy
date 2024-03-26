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

	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	alloydbInstanceURI = flag.String(
		"alloydb_conn_name",
		os.Getenv("ALLOYDB_INSTANCE_URI"),
		`AlloyDB instance connection name, in the form of
projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>`,
	)
	alloydbPSCInstanceURI = flag.String(
		"alloydb_psc_conn_name",
		os.Getenv("ALLOYDB_PSC_INSTANCE_URI"),
		`AlloyDB PSC instance connection name, in the form of
projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>`,
	)
	alloydbUser = flag.String(
		"alloydb_user",
		os.Getenv("ALLOYDB_USER"),
		"Name of database user.",
	)
	alloydbIAMUser = flag.String(
		"alloydb_iam_user",
		os.Getenv("ALLOYDB_IAM_USER"),
		"Name of database user.",
	)
	alloydbPass = flag.String(
		"alloydb_pass",
		os.Getenv("ALLOYDB_PASS"),
		"Password for the database user.",
	)
	alloydbDB = flag.String(
		"alloydb_db",
		os.Getenv("ALLOYDB_DB"),
		"Name of the database to connect to.",
	)
)

func requirePostgresVars(t *testing.T) {
	switch "" {
	case *alloydbInstanceURI:
		t.Fatal("'alloydb_conn_name' not set")
	case *alloydbUser:
		t.Fatal("'alloydb_user' not set")
	case *alloydbIAMUser:
		t.Fatal("'alloydb_iam_user' not set")
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

	dsn := fmt.Sprintf(
		"host=127.0.0.1 port=10000 user=%v password=%v database=%v sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB,
	)
	proxyConnTest(t, []string{*alloydbInstanceURI, "--port=10000"}, "pgx", dsn)
}

func TestPostgresAutoIAMAuthN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	dsn := fmt.Sprintf("host=127.0.0.1 port=10001 user=%v database=%v sslmode=disable",
		*alloydbIAMUser, *alloydbDB)
	proxyConnTest(t, []string{
		*alloydbInstanceURI, "--auto-iam-authn", "--port=10001",
	}, "pgx", dsn)
}

func createTempDir(t *testing.T) (string, func()) {
	testDir, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return testDir, func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("failed to cleanup temp dir: %v", err)
		}
	}
}

func TestPostgresUnix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)
	tmpDir, cleanup := createTempDir(t)
	defer cleanup()

	dir, err := proxy.UnixSocketDir(tmpDir, *alloydbInstanceURI)
	if err != nil {
		t.Fatalf("invalid connection name: %v", *alloydbInstanceURI)
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s database=%s sslmode=disable",
		dir,
		*alloydbUser, *alloydbPass, *alloydbDB)

	proxyConnTest(t,
		[]string{"--unix-socket", tmpDir, *alloydbInstanceURI}, "pgx", dsn)
}

func TestPostgresAuthWithToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)
	tok, _, cleanup2 := removeAuthEnvVar(t, true)
	defer cleanup2()

	dsn := fmt.Sprintf(
		"host=localhost port=10002 user=%v password=%v database=%v sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB,
	)
	proxyConnTest(t,
		[]string{"--token", tok.AccessToken, *alloydbInstanceURI,
			"--port=10002",
		},
		"pgx", dsn)
}

func TestPostgresAuthWithCredentialsFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)
	_, path, cleanup2 := removeAuthEnvVar(t, false)
	defer cleanup2()

	dsn := fmt.Sprintf(
		"host=localhost port=10003 user=%v password=%v database=%v sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB)
	proxyConnTest(t,
		[]string{"--credentials-file", path, *alloydbInstanceURI,
			"--port=10003",
		},
		"pgx", dsn)
}

func TestPostgresAuthWithCredentialsJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)
	creds := keyfile(t)
	_, _, cleanup := removeAuthEnvVar(t, false)
	defer cleanup()

	dsn := fmt.Sprintf(
		"host=localhost port=10004 user=%s password=%s database=%s sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB)
	proxyConnTest(t,
		[]string{"--json-credentials", string(creds), *alloydbInstanceURI,
			"--port=10004",
		},
		"pgx", dsn)
}

func TestAuthWithGcloudAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	dsn := fmt.Sprintf(
		"host=localhost port=10005 user=%s password=%s database=%s sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB,
	)
	proxyConnTest(t,
		[]string{"--gcloud-auth", *alloydbInstanceURI, "--port=10005"},
		"pgx", dsn)
}

func TestPostgresPublicIP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	dsn := fmt.Sprintf(
		"host=127.0.0.1 port=10006 user=%v password=%v database=%v sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB,
	)
	proxyConnTest(t, []string{*alloydbInstanceURI, "--public-ip", "--port=10006"}, "pgx", dsn)
}

func TestPostgresPSC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	dsn := fmt.Sprintf(
		"host=127.0.0.1 port=10007 user=%v password=%v database=%v sslmode=disable",
		*alloydbUser, *alloydbPass, *alloydbDB,
	)
	proxyConnTest(t, []string{
		*alloydbPSCInstanceURI, "--psc", "--port=10007"}, "pgx", dsn,
	)
}
