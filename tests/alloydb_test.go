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
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
)

var (
	alloydbConnName = flag.String(
		"alloydb_conn_name",
		os.Getenv("ALLOYDB_CONNECTION_NAME"),
		"AlloyDB instance connection name, in the form of projects/proj/locations/region/clusters/clust/instances/inst",
	)
	alloydbUser = flag.String(
		"alloydb_user",
		os.Getenv("ALLOYDB_USER"),
		"Name of database user.",
	)
	alloydbPass = flag.String(
		"alloydb_pass",
		os.Getenv("ALLOYDB_PASS"),
		"Password for the database user",
	)
	alloydbDB = flag.String(
		"alloydb_db",
		os.Getenv("ALLOYDB_DB"),
		"Name of the database to connect to.",
	)
	impersonatedUser = flag.String(
		"impersonated_user",
		os.Getenv("IMPERSONATED_USER"),
		"Name of the service account that supports impersonation (impersonator must have roles/iam.serviceAccountTokenCreator)",
	)
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

func postgresDSN() string {
	return fmt.Sprintf("host=%v user=%s password=%s database=%s sslmode=disable",
		*alloydbConnName, *alloydbUser, *alloydbPass, *alloydbDB)
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

	dir, err := proxy.UnixSocketDir(tmpDir, *alloydbConnName)
	if err != nil {
		t.Fatalf("invalid connection name: %v", *alloydbConnName)
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s database=%s sslmode=disable",
		dir,
		*alloydbUser, *alloydbPass, *alloydbDB)

	proxyConnTest(t,
		[]string{"--unix-socket", tmpDir, *alloydbConnName}, "pgx", dsn)
}

func TestPostgresImpersonation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	proxyConnTest(t, []string{
		"--impersonate-service-account", *impersonatedUser,
		*alloydbConnName},
		"pgx", postgresDSN())
}

func TestPostgresAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	creds := keyfile(t)
	tok, path, cleanup := removeAuthEnvVar(t, true)
	defer cleanup()

	tcs := []struct {
		desc string
		args []string
	}{
		{
			desc: "with token",
			args: []string{"--token", tok.AccessToken, *alloydbConnName},
		},
		{
			desc: "with token and impersonation",
			args: []string{
				"--token", tok.AccessToken,
				"--impersonate-service-account", *impersonatedUser,
				*alloydbConnName},
		},
		{
			desc: "with credentials file",
			args: []string{"--credentials-file", path, *alloydbConnName},
		},
		{
			desc: "with credentials file and impersonation",
			args: []string{
				"--credentials-file", path,
				"--impersonate-service-account", *impersonatedUser,
				*alloydbConnName},
		},
		{
			desc: "with credentials JSON",
			args: []string{"--json-credentials", string(creds), *alloydbConnName},
		},
		{
			desc: "with credentials JSON and impersonation",
			args: []string{
				"--json-credentials", string(creds),
				"--impersonate-service-account", *impersonatedUser,
				*alloydbConnName},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			proxyConnTest(t, tc.args, "pgx", postgresDSN())
		})
	}
}

func TestPostgresGcloudAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres integration tests")
	}
	requirePostgresVars(t)

	tcs := []struct {
		desc string
		args []string
	}{
		{
			desc: "gcloud user authentication",
			args: []string{"--gcloud-auth", *alloydbConnName},
		},
		{
			desc: "gcloud user authentication with impersonation",
			args: []string{
				"--gcloud-auth",
				"--impersonate-service-account", *impersonatedUser,
				*alloydbConnName},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			proxyConnTest(t, tc.args, "pgx", postgresDSN())
		})
	}

}
