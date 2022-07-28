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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

func TestNewCommandArguments(t *testing.T) {
	withDefaults := func(c *proxy.Config) *proxy.Config {
		if c.UserAgent == "" {
			c.UserAgent = userAgent
		}
		if c.Addr == "" {
			c.Addr = "127.0.0.1"
		}
		if c.Port == 0 {
			c.Port = 5432
		}
		if c.Instances == nil {
			c.Instances = []proxy.InstanceConnConfig{{}}
		}
		if i := &c.Instances[0]; i.Name == "" {
			i.Name = "projects/proj/locations/region/clusters/clust/instances/inst"
		}
		if c.APIEndpointURL == "" {
			c.APIEndpointURL = "https://alloydb.googleapis.com/v1beta"
		}
		return c
	}
	tcs := []struct {
		desc string
		args []string
		want *proxy.Config
	}{
		{
			desc: "basic invocation with defaults",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Addr:      "127.0.0.1",
				Instances: []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "using the address flag",
			args: []string{"--address", "0.0.0.0", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Addr:      "0.0.0.0",
				Instances: []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "using the address (short) flag",
			args: []string{"-a", "0.0.0.0", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Addr:      "0.0.0.0",
				Instances: []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "using the address query param",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?address=0.0.0.0"},
			want: withDefaults(&proxy.Config{
				Addr: "127.0.0.1",
				Instances: []proxy.InstanceConnConfig{{
					Addr: "0.0.0.0",
					Name: "projects/proj/locations/region/clusters/clust/instances/inst",
				}},
			}),
		},
		{
			desc: "using the port flag",
			args: []string{"--port", "6000", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Port: 6000,
			}),
		},
		{
			desc: "using the port (short) flag",
			args: []string{"-p", "6000", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Port: 6000,
			}),
		},
		{
			desc: "using the port query param",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?port=6000"},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					Port: 6000,
				}},
			}),
		},
		{
			desc: "using the token flag",
			args: []string{"--token", "MYCOOLTOKEN", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Token: "MYCOOLTOKEN",
			}),
		},
		{
			desc: "using the token (short) flag",
			args: []string{"-t", "MYCOOLTOKEN", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Token: "MYCOOLTOKEN",
			}),
		},
		{
			desc: "using the credentiale file flag",
			args: []string{"--credentials-file", "/path/to/file", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				CredentialsFile: "/path/to/file",
			}),
		},
		{
			desc: "using the (short) credentiale file flag",
			args: []string{"-c", "/path/to/file", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				CredentialsFile: "/path/to/file",
			}),
		},
		{
			desc: "using the unix socket flag",
			args: []string{"--unix-socket", "/path/to/dir/", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				UnixSocket: "/path/to/dir/",
			}),
		},
		{
			desc: "using the (short) unix socket flag",
			args: []string{"-u", "/path/to/dir/", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				UnixSocket: "/path/to/dir/",
			}),
		},
		{
			desc: "using the unix socket query param",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket=/path/to/dir/"},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					UnixSocket: "/path/to/dir/",
				}},
			}),
		},
		{
			desc: "using the max connections flag",
			args: []string{"--max-connections", "1", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				MaxConnections: 1,
			}),
		},
		{
			desc: "using wait after signterm flag",
			args: []string{"--max-sigterm-delay", "10s", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				WaitOnClose: 10 * time.Second,
			}),
		},
		{
			desc: "enabling structured logging",
			args: []string{"--structured-logs", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				StructuredLogs: true,
			}),
		},
		{
			desc: "using the alloydbadmin-api-endpoint flag with the trailing slash",
			args: []string{"--alloydbadmin-api-endpoint", "https://test.googleapis.com/", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				APIEndpointURL: "https://test.googleapis.com",
			}),
		},
		{
			desc: "using the alloydbadmin-api-endpoint flag without the trailing slash",
			args: []string{"--alloydbadmin-api-endpoint", "https://test.googleapis.com", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				APIEndpointURL: "https://test.googleapis.com",
			}),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewCommand()
			// Keep the test output quiet
			c.SilenceUsage = true
			c.SilenceErrors = true
			// Disable execute behavior
			c.RunE = func(*cobra.Command, []string) error {
				return nil
			}
			c.SetArgs(tc.args)

			err := c.Execute()
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			if got := c.conf; !cmp.Equal(tc.want, got) {
				t.Fatalf("want = %#v\ngot = %#v\ndiff = %v", tc.want, got, cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestNewCommandWithGcloudAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Gcloud auth test")
	}
	tcs := []struct {
		desc string
		args []string
		want bool
	}{
		{
			desc: "using the gcloud auth flag",
			args: []string{"--gcloud-auth", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: true,
		},
		{
			desc: "using the (short) gcloud auth flag",
			args: []string{"-g", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewCommand()
			// Keep the test output quiet
			c.SilenceUsage = true
			c.SilenceErrors = true
			// Disable execute behavior
			c.RunE = func(*cobra.Command, []string) error {
				return nil
			}
			c.SetArgs(tc.args)

			err := c.Execute()
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			if got := c.conf.GcloudAuth; got != tc.want {
				t.Fatalf("want = %v, got = %v", tc.want, got)
			}
		})
	}
}

func TestNewCommandWithErrors(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
	}{
		{
			desc: "basic invocation missing instance connection name",
			args: []string{},
		},
		{
			desc: "when the query string is bogus",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?%=foo"},
		},
		{
			desc: "when the address query param is empty",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?address="},
		},
		{
			desc: "using the address flag with a bad IP address",
			args: []string{"--address", "bogus", "projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when the address query param is not an IP address",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?address=世界"},
		},
		{
			desc: "when the address query param contains multiple values",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?address=0.0.0.0&address=1.1.1.1&address=2.2.2.2"},
		},
		{
			desc: "when the query string is invalid",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?address=1.1.1.1?foo=2.2.2.2"},
		},
		{
			desc: "when the port query param contains multiple values",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?port=1&port=2"},
		},
		{
			desc: "when the port query param is not a number",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?port=hi"},
		},
		{
			desc: "when both token and credentials file are set",
			args: []string{
				"--token", "my-token",
				"--credentials-file", "/path/to/file", "projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when both token and gcloud auth are set",
			args: []string{
				"--token", "my-token",
				"--gcloud-auth", "proj:region:inst"},
		},
		{
			desc: "when both gcloud auth and credentials file are set",
			args: []string{
				"--gcloud-auth",
				"--credential-file", "/path/to/file", "proj:region:inst"},
		},
		{
			desc: "when the unix socket query param contains multiple values",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket=/one&unix-socket=/two"},
		},
		{
			desc: "using the unix socket flag with addr",
			args: []string{"-u", "/path/to/dir/", "-a", "127.0.0.1", "projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "using the unix socket flag with port",
			args: []string{"-u", "/path/to/dir/", "-p", "5432", "projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "using the unix socket and addr query params",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket=/path&address=127.0.0.1"},
		},
		{
			desc: "using the unix socket and port query params",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket=/path&port=5000"},
		},
		{
			desc: "using an invalid url for host flag",
			args: []string{"--host", "https://invalid:url[/]", "proj:region:inst"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewCommand()
			// Keep the test output quiet
			c.SilenceUsage = true
			c.SilenceErrors = true
			// Disable execute behavior
			c.RunE = func(*cobra.Command, []string) error {
				return nil
			}
			c.SetArgs(tc.args)

			err := c.Execute()
			if err == nil {
				t.Fatal("want error != nil, got = nil")
			}
		})
	}
}

type spyDialer struct {
	mu  sync.Mutex
	got string
}

func (s *spyDialer) instance() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.got
	return i
}

func (s *spyDialer) Dial(_ context.Context, inst string, _ ...alloydbconn.DialOption) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = inst
	return nil, errors.New("spy dialer does not dial")
}

func (*spyDialer) Close() error {
	return nil
}

func TestCommandWithCustomDialer(t *testing.T) {
	want := "projects/my-project/locations/my-region/clusters/my-cluster/instances/my-instance"
	s := &spyDialer{}
	c := NewCommand(WithDialer(s))
	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{"--port", "10000", want})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.ExecuteContext(ctx)

	// try will run f count times, returning early if f succeeds, or failing
	// when count has been exceeded.
	try := func(f func() error, count int) {
		var (
			attempts int
			err      error
		)
		for {
			if attempts == count {
				t.Fatal(err)
			}
			err = f()
			if err != nil {
				attempts++
				time.Sleep(time.Millisecond)
				continue
			}
			return
		}
	}
	// give the listener some time to start
	try(func() error {
		conn, err := net.Dial("tcp", "127.0.0.1:10000")
		if err != nil {
			return err
		}
		defer conn.Close()
		return nil
	}, 10)

	// give the proxy some time to run
	try(func() error {
		if got := s.instance(); got != want {
			return fmt.Errorf("want = %v, got = %v", want, got)
		}
		return nil
	}, 10)
}

func TestPrometheusMetricsEndpoint(t *testing.T) {
	c := NewCommand(WithDialer(&spyDialer{}))
	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{
		"--prometheus-namespace", "prometheus",
		"my-project:my-region:my-instance"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go c.ExecuteContext(ctx)

	// try to dial metrics server for a max of ~10s to give the proxy time to
	// start up.
	tryDial := func(addr string) (*http.Response, error) {
		var (
			resp     *http.Response
			attempts int
			err      error
		)
		for {
			if attempts > 10 {
				return resp, err
			}
			resp, err = http.Get(addr)
			if err != nil {
				attempts++
				time.Sleep(time.Second)
				continue
			}
			return resp, err
		}
	}
	resp, err := tryDial("http://localhost:9090/metrics") // default port set by http-port flag
	if err != nil {
		t.Fatalf("failed to dial metrics endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}
}
