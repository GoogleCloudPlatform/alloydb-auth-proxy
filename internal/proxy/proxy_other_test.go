// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows

package proxy_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
)

func verifySocketPermissions(t *testing.T, addr string) {
	fi, err := os.Stat(addr)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", addr, err)
	}
	if fm := fi.Mode(); fm != 0700|os.ModeSocket {
		t.Fatalf("file mode: want = %v, got = %v", 0700|os.ModeSocket, fm)
	}
	dir := filepath.Dir(addr)
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", dir, err)
	}
	if dm := di.Mode(); dm != 0700|os.ModeDir {
		t.Fatalf("dir mode: want = %v, got = %v", 0700|os.ModeDir, dm)
	}
}

func TestUnixSocketCustomMode(t *testing.T) {
	ctx := context.Background()
	testDir, cleanup := createTempDir(t)
	defer cleanup()

	inst1 := "projects/proj/locations/region/clusters/clust/instances/inst1"
	inst2 := "projects/proj/locations/region/clusters/clust/instances/inst2"
	customInstMode := os.FileMode(0770)

	in := &proxy.Config{
		UnixSocket:     testDir,
		UnixSocketMode: 0777,
		Instances: []proxy.InstanceConnConfig{
			{Name: inst1},
			{Name: inst2, UnixSocketMode: &customInstMode},
		},
	}

	c, err := proxy.NewClient(ctx, &fakeDialer{}, testLogger, in)
	if err != nil {
		t.Fatalf("proxy.NewClient failed: %v", err)
	}
	defer c.Close()

	sock1 := filepath.Join(testDir, "proj.region.clust.inst1", ".s.PGSQL.5432")
	fi1, err := os.Stat(sock1)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", sock1, err)
	}
	if fm := fi1.Mode(); fm != 0777|os.ModeSocket {
		t.Fatalf("inst1 socket file mode: want = %v, got = %v", 0777|os.ModeSocket, fm)
	}
	di1, err := os.Stat(filepath.Dir(sock1))
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", filepath.Dir(sock1), err)
	}
	if dm := di1.Mode(); dm != 0777|os.ModeDir {
		t.Fatalf("inst1 dir mode: want = %v, got = %v", 0777|os.ModeDir, dm)
	}

	sock2 := filepath.Join(testDir, "proj.region.clust.inst2", ".s.PGSQL.5432")
	fi2, err := os.Stat(sock2)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", sock2, err)
	}
	if fm := fi2.Mode(); fm != 0770|os.ModeSocket {
		t.Fatalf("inst2 socket file mode: want = %v, got = %v", 0770|os.ModeSocket, fm)
	}
	di2, err := os.Stat(filepath.Dir(sock2))
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", filepath.Dir(sock2), err)
	}
	if dm := di2.Mode(); dm != 0770|os.ModeDir {
		t.Fatalf("inst2 dir mode: want = %v, got = %v", 0770|os.ModeDir, dm)
	}
}

func TestFuseClosesGracefully(t *testing.T) {
	fuseTempDir := filepath.Join(t.TempDir(), "fuse-tmp")
	c, err := proxy.NewClient(
		context.Background(), nil, testLogger,
		&proxy.Config{
			FUSEDir:     t.TempDir(),
			FUSETempDir: fuseTempDir,
			Token:       "mytoken",
		})
	if err != nil {
		t.Fatal(err)
	}
	di, err := os.Stat(fuseTempDir)
	if err != nil {
		t.Fatalf("os.Stat(%v): %v", fuseTempDir, err)
	}
	if dm := di.Mode(); dm != 0700|os.ModeDir {
		t.Fatalf("FUSETempDir mode: want = %v, got = %v", 0700|os.ModeDir, dm)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
