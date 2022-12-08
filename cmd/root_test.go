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
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

const sampleURI = "projects/proj/locations/region/clusters/clust/instances/inst"

func invokeProxyCommand(args []string) (*Command, error) {
	c := NewCommand()
	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true
	// Disable execute behavior
	c.RunE = func(*cobra.Command, []string) error {
		return nil
	}
	c.SetArgs(args)

	err := c.Execute()

	return c, err
}

func withDefaults(c *proxy.Config) *proxy.Config {
	if c.UserAgent == "" {
		c.UserAgent = userAgent
	}
	if c.Addr == "" {
		c.Addr = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.FUSEDir == "" {
		if c.Instances == nil {
			c.Instances = []proxy.InstanceConnConfig{{}}
		}
		if i := &c.Instances[0]; i.Name == "" {
			i.Name = sampleURI
		}
	}
	if c.FUSETempDir == "" {
		c.FUSETempDir = filepath.Join(os.TempDir(), "csql-tmp")
	}
	if c.APIEndpointURL == "" {
		c.APIEndpointURL = "https://alloydb.googleapis.com/v1beta"
	}
	return c
}

func TestNewCommandArguments(t *testing.T) {
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
		{
			desc: "using the JSON credentials",
			args: []string{"--json-credentials", `{"json":"goes-here"}`, "projects/proj/locations/region/clusters/clust/instances/inst"}, want: withDefaults(&proxy.Config{
				CredentialsJSON: `{"json":"goes-here"}`,
			}),
		},
		{
			desc: "using the (short) JSON credentials",
			args: []string{"-j", `{"json":"goes-here"}`, "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				CredentialsJSON: `{"json":"goes-here"}`,
			}),
		},
		{
			desc: "using the impersonate service account flag",
			args: []string{"--impersonate-service-account",
				"sv1@developer.gserviceaccount.com,sv2@developer.gserviceaccount.com,sv3@developer.gserviceaccount.com",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				ImpersonateTarget: "sv1@developer.gserviceaccount.com",
				ImpersonateDelegates: []string{
					"sv3@developer.gserviceaccount.com",
					"sv2@developer.gserviceaccount.com",
				},
			}),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, err := invokeProxyCommand(tc.args)
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			if got := c.conf; !cmp.Equal(tc.want, got) {
				t.Fatalf("want = %#v\ngot = %#v\ndiff = %v", tc.want, got, cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestNewCommandWithEnvironmentConfigPrivateFields(t *testing.T) {
	tcs := []struct {
		desc     string
		envName  string
		envValue string
		isValid  func(cmd *Command) bool
	}{
		{
			desc:     "using the disable traces envvar",
			envName:  "ALLOYDB_DISABLE_TRACES",
			envValue: "true",
			isValid: func(cmd *Command) bool {
				return cmd.disableTraces == true
			},
		},
		{
			desc:     "using the telemetry sample rate envvar",
			envName:  "ALLOYDB_TELEMETRY_SAMPLE_RATE",
			envValue: "500",
			isValid: func(cmd *Command) bool {
				return cmd.telemetryTracingSampleRate == 500
			},
		},
		{
			desc:     "using the disable metrics envvar",
			envName:  "ALLOYDB_DISABLE_METRICS",
			envValue: "true",
			isValid: func(cmd *Command) bool {
				return cmd.disableMetrics == true
			},
		},
		{
			desc:     "using the telemetry project envvar",
			envName:  "ALLOYDB_TELEMETRY_PROJECT",
			envValue: "mycoolproject",
			isValid: func(cmd *Command) bool {
				return cmd.telemetryProject == "mycoolproject"
			},
		},
		{
			desc:     "using the telemetry prefix envvar",
			envName:  "ALLOYDB_TELEMETRY_PREFIX",
			envValue: "myprefix",
			isValid: func(cmd *Command) bool {
				return cmd.telemetryPrefix == "myprefix"
			},
		},
		{
			desc:     "using the prometheus envvar",
			envName:  "ALLOYDB_PROMETHEUS",
			envValue: "true",
			isValid: func(cmd *Command) bool {
				return cmd.prometheus == true
			},
		},
		{
			desc:     "using the prometheus namespace envvar",
			envName:  "ALLOYDB_PROMETHEUS_NAMESPACE",
			envValue: "myns",
			isValid: func(cmd *Command) bool {
				return cmd.prometheusNamespace == "myns"
			},
		},
		{
			desc:     "using the health check envvar",
			envName:  "ALLOYDB_HEALTH_CHECK",
			envValue: "true",
			isValid: func(cmd *Command) bool {
				return cmd.healthCheck == true
			},
		},
		{
			desc:     "using the http address envvar",
			envName:  "ALLOYDB_HTTP_ADDRESS",
			envValue: "0.0.0.0",
			isValid: func(cmd *Command) bool {
				return cmd.httpAddress == "0.0.0.0"
			},
		},
		{
			desc:     "using the http port envvar",
			envName:  "ALLOYDB_HTTP_PORT",
			envValue: "5555",
			isValid: func(cmd *Command) bool {
				return cmd.httpPort == "5555"
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			os.Setenv(tc.envName, tc.envValue)
			defer os.Unsetenv(tc.envName)

			c, err := invokeProxyCommand([]string{
				"projects/proj/locations/region/clusters/clust/instances/inst",
			})
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			if !tc.isValid(c) {
				t.Fatal("want valid, got invalid")
			}
		})
	}
}

func TestNewCommandWithEnvironmentConfig(t *testing.T) {
	tcs := []struct {
		desc     string
		envName  string
		envValue string
		want     *proxy.Config
	}{
		{
			desc:     "using the address envvar",
			envName:  "ALLOYDB_ADDRESS",
			envValue: "0.0.0.0",
			want: withDefaults(&proxy.Config{
				Addr: "0.0.0.0",
			}),
		},
		{
			desc:     "using the port envvar",
			envName:  "ALLOYDB_PORT",
			envValue: "6000",
			want: withDefaults(&proxy.Config{
				Port: 6000,
			}),
		},
		{
			desc:     "using the token envvar",
			envName:  "ALLOYDB_TOKEN",
			envValue: "MYCOOLTOKEN",
			want: withDefaults(&proxy.Config{
				Token: "MYCOOLTOKEN",
			}),
		},
		{
			desc:     "using the credentiale file envvar",
			envName:  "ALLOYDB_CREDENTIALS_FILE",
			envValue: "/path/to/file",
			want: withDefaults(&proxy.Config{
				CredentialsFile: "/path/to/file",
			}),
		},
		{
			desc:     "using the JSON credentials",
			envName:  "ALLOYDB_JSON_CREDENTIALS",
			envValue: `{"json":"goes-here"}`,
			want: withDefaults(&proxy.Config{
				CredentialsJSON: `{"json":"goes-here"}`,
			}),
		},
		{
			desc:     "using the gcloud auth envvar",
			envName:  "ALLOYDB_GCLOUD_AUTH",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				GcloudAuth: true,
			}),
		},
		{
			desc:     "using the api-endpoint envvar",
			envName:  "ALLOYDB_ALLOYDBADMIN_API_ENDPOINT",
			envValue: "https://test.googleapis.com/",
			want: withDefaults(&proxy.Config{
				APIEndpointURL: "https://test.googleapis.com",
			}),
		},
		{
			desc:     "using the unix socket envvar",
			envName:  "ALLOYDB_UNIX_SOCKET",
			envValue: "/path/to/dir/",
			want: withDefaults(&proxy.Config{
				UnixSocket: "/path/to/dir/",
			}),
		},
		{
			desc:     "enabling structured logging",
			envName:  "ALLOYDB_STRUCTURED_LOGS",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				StructuredLogs: true,
			}),
		},
		{
			desc:     "using the max connections envvar",
			envName:  "ALLOYDB_MAX_CONNECTIONS",
			envValue: "1",
			want: withDefaults(&proxy.Config{
				MaxConnections: 1,
			}),
		},
		{
			desc:     "using wait after signterm envvar",
			envName:  "ALLOYDB_MAX_SIGTERM_DELAY",
			envValue: "10s",
			want: withDefaults(&proxy.Config{
				WaitOnClose: 10 * time.Second,
			}),
		},
		{
			desc:     "using the imopersonate service accounn envvar",
			envName:  "ALLOYDB_IMPERSONATE_SERVICE_ACCOUNT",
			envValue: "sv1@developer.gserviceaccount.com,sv2@developer.gserviceaccount.com,sv3@developer.gserviceaccount.com",
			want: withDefaults(&proxy.Config{
				ImpersonateTarget: "sv1@developer.gserviceaccount.com",
				ImpersonateDelegates: []string{
					"sv3@developer.gserviceaccount.com",
					"sv2@developer.gserviceaccount.com",
				},
			}),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			os.Setenv(tc.envName, tc.envValue)
			defer os.Unsetenv(tc.envName)

			c, err := invokeProxyCommand([]string{sampleURI})
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			if got := c.conf; !cmp.Equal(tc.want, got) {
				t.Fatalf("want = %#v\ngot = %#v\ndiff = %v", tc.want, got, cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestNewCommandWithEnvironmentConfigInstanceConnectionName(t *testing.T) {
	u := "projects/proj/locations/region/clusters/clust/instances/inst"
	tcs := []struct {
		desc string
		env  map[string]string
		args []string
		want *proxy.Config
	}{
		{
			desc: "with one instance connection name",
			env: map[string]string{
				"ALLOYDB_INSTANCE_URI": u,
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u},
			}}),
		},
		{
			desc: "with multiple instance connection names",
			env: map[string]string{
				"ALLOYDB_INSTANCE_URI_0": u + "0",
				"ALLOYDB_INSTANCE_URI_1": u + "1",
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "0"},
				{Name: u + "1"},
			}}),
		},
		{
			desc: "when the index skips a number",
			env: map[string]string{
				"ALLOYDB_INSTANCE_URI_0": u + "0",
				"ALLOYDB_INSTANCE_URI_2": u + "2",
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "0"},
			}}),
		},
		{
			desc: "when there are CLI args provided",
			env: map[string]string{
				"ALLOYDB_INSTANCE_URI": u,
			},
			args: []string{u + "1"},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "1"},
			}}),
		},
		{
			desc: "when only an index instance connection name is defined",
			env: map[string]string{
				"ALLOYDB_INSTANCE_URI_0": u,
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u},
			}}),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var cleanup []string
			for k, v := range tc.env {
				os.Setenv(k, v)
				cleanup = append(cleanup, k)
			}
			defer func() {
				for _, k := range cleanup {
					os.Unsetenv(k)
				}
			}()

			c, err := invokeProxyCommand(tc.args)
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
				"--gcloud-auth",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when both gcloud auth and credentials file are set",
			args: []string{
				"--gcloud-auth",
				"--credentials-file", "/path/to/file",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when both token and credentials JSON are set",
			args: []string{
				"--token", "a-token",
				"--json-credentials", `{"json":"here"}`,
				"projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when both credentials file and credentials JSON are set",
			args: []string{
				"--credentials-file", "/a/file",
				"--json-credentials", `{"json":"here"}`,
				"projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "when both gcloud auth and credentials JSON are set",
			args: []string{
				"--gcloud-auth",
				"--json-credentials", `{"json":"here"}`,
				"projects/proj/locations/region/clusters/clust/instances/inst"},
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
			args: []string{"--host", "https://invalid:url[/]",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
		},
		{
			desc: "using fuse-tmp-dir without fuse",
			args: []string{"--fuse-tmp-dir", "/mydir"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := invokeProxyCommand(tc.args)
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
	c.SetArgs([]string{"--prometheus", "my-project:my-region:my-instance"})

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
