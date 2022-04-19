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

package proxy_test

import (
	"context"
	"net"
	"os"
	"testing"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	"github.com/spf13/cobra"
)

type fakeDialer struct{}

type testCase struct {
	desc      string
	in        *proxy.Config
	wantAddrs []string
}

func (fakeDialer) Dial(ctx context.Context, inst string, opts ...alloydbconn.DialOption) (net.Conn, error) {
	return nil, nil
}

func (fakeDialer) Close() error {
	return nil
}

func TestClientInitialization(t *testing.T) {
	ctx := context.Background()
	cluster1 := "/projects/proj/locations/region/clusters/clust/instances/inst1"
	cluster2 := "/projects/proj/locations/region/clusters/clust/instances/inst2"

	tcs := []testCase{
		{
			desc: "multiple instances",
			in: &proxy.Config{
				Addr: "127.0.0.1",
				Port: 5000,
				Instances: []proxy.InstanceConnConfig{
					{Name: cluster1},
					{Name: cluster2},
				},
			},
			wantAddrs: []string{"127.0.0.1:5000", "127.0.0.1:5001"},
		},
		{
			desc: "with instance address",
			in: &proxy.Config{
				Addr: "1.1.1.1", // bad address, binding shouldn't happen here.
				Port: 5000,
				Instances: []proxy.InstanceConnConfig{
					{Addr: "0.0.0.0", Name: cluster1},
				},
			},
			wantAddrs: []string{"0.0.0.0:5000"},
		},
		{
			desc: "with instance port",
			in: &proxy.Config{
				Addr: "127.0.0.1",
				Port: 5000,
				Instances: []proxy.InstanceConnConfig{
					{Name: cluster1, Port: 6000},
				},
			},
			wantAddrs: []string{"127.0.0.1:6000"},
		},
		{
			desc: "with global port and instance port",
			in: &proxy.Config{
				Addr: "127.0.0.1",
				Port: 5000,
				Instances: []proxy.InstanceConnConfig{
					{Name: cluster1},
					{Name: cluster2, Port: 6000},
				},
			},
			wantAddrs: []string{
				"127.0.0.1:5000",
				"127.0.0.1:6000",
			},
		},
		{
			desc: "with incrementing automatic port selection",
			in: &proxy.Config{
				Addr: "127.0.0.1",
				Port: 5432, // default port
				Instances: []proxy.InstanceConnConfig{
					{Name: cluster1},
					{Name: cluster2},
				},
			},
			wantAddrs: []string{
				"127.0.0.1:5432",
				"127.0.0.1:5433",
			},
		},
	}
	_, isFlex := os.LookupEnv("FLEX")
	if !isFlex {
		tcs = append(tcs, testCase{
			desc: "IPv6 support",
			in: &proxy.Config{
				Addr: "::1",
				Port: 5000,
				Instances: []proxy.InstanceConnConfig{
					{Name: cluster1},
				},
			},
			wantAddrs: []string{"[::1]:5000"},
		})
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, err := proxy.NewClient(ctx, fakeDialer{}, &cobra.Command{}, tc.in)
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}
			defer c.Close()
			for _, addr := range tc.wantAddrs {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatalf("want error = nil, got = %v", err)
				}
				err = conn.Close()
				if err != nil {
					t.Logf("failed to close connection: %v", err)
				}
			}
		})
	}
}
