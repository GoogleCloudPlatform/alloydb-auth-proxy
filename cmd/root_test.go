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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/proxy"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

const (
	sampleURI = "projects/proj/locations/region/clusters/clust/instances/inst"
	altURI    = "projects/proj/locations/region/clusters/clust2/instances/inst2"
)

// pointer returns the address of v and makes it easy to take the address of a
// predeclared identifier. Compare:
//
//	t := true
//	pt := &t
//
// vs
//
//	pt := pointer(true)
func pointer[T any](v T) *T {
	return &v
}

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
		c.UserAgent = defaultUserAgent
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
		c.FUSETempDir = filepath.Join(os.TempDir(), "alloydb-tmp")
	}
	if c.HTTPAddress == "" {
		c.HTTPAddress = "localhost"
	}
	if c.HTTPPort == "" {
		c.HTTPPort = "9090"
	}
	if c.AdminPort == "" {
		c.AdminPort = "9091"
	}
	if c.TelemetryTracingSampleRate == 0 {
		c.TelemetryTracingSampleRate = 10_000
	}
	if c.APIEndpointURL == "" {
		c.APIEndpointURL = "https://alloydb.googleapis.com"
	}
	return c
}

func TestUserAgentWithVersionEnvVar(t *testing.T) {
	os.Setenv("ALLOYDB_PROXY_USER_AGENT", "some-runtime/0.0.1")
	defer os.Unsetenv("ALLOYDB_PROXY_USER_AGENT")

	cmd, err := invokeProxyCommand([]string{
		"projects/proj/locations/region/clusters/clust/instances/inst",
	})
	if err != nil {
		t.Fatalf("want error = nil, got = %v", err)
	}

	want := "some-runtime/0.0.1"
	got := cmd.conf.UserAgent
	if !strings.Contains(got, want) {
		t.Errorf("expected user agent to contain: %v; got: %v", want, got)
	}
}

func TestUserAgent(t *testing.T) {
	cmd, err := invokeProxyCommand(
		[]string{
			"--user-agent",
			"some-runtime/0.0.1",
			"projects/proj/locations/region/clusters/clust/instances/inst",
		},
	)
	if err != nil {
		t.Fatalf("want error = nil, got = %v", err)
	}

	want := "some-runtime/0.0.1"
	got := cmd.conf.UserAgent
	if !strings.Contains(got, want) {
		t.Errorf("expected userAgent to contain: %v; got: %v", want, got)
	}
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
			desc: "Auto IAM AuthN",
			args: []string{
				"--auto-iam-authn",
				"projects/proj/locations/region/clusters/clust/instances/inst",
			},
			want: withDefaults(&proxy.Config{
				AutoIAMAuthN: true,
				Instances:    []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "Auto IAM AuthN query param (key only)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst?auto-iam-authn",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					AutoIAMAuthN: pointer(true),
					Name:         "projects/proj/locations/region/clusters/clust/instances/inst",
				}},
			}),
		},
		{
			desc: "Auto IAM AuthN query param (t & f)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst1?auto-iam-authn=t",
				"projects/proj/locations/region/clusters/clust/instances/inst2?auto-iam-authn=f",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					AutoIAMAuthN: pointer(true),
					Name:         "projects/proj/locations/region/clusters/clust/instances/inst1",
				}, {
					AutoIAMAuthN: pointer(false),
					Name:         "projects/proj/locations/region/clusters/clust/instances/inst2",
				}},
			}),
		},
		{
			desc: "Auto IAM AuthN query param (true & false)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst1?auto-iam-authn=true",
				"projects/proj/locations/region/clusters/clust/instances/inst2?auto-iam-authn=false",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					AutoIAMAuthN: pointer(true),
					Name:         "projects/proj/locations/region/clusters/clust/instances/inst1",
				}, {
					AutoIAMAuthN: pointer(false),
					Name:         "projects/proj/locations/region/clusters/clust/instances/inst2",
				}},
			}),
		},
		{
			desc: "Public IP",
			args: []string{
				"--public-ip",
				"projects/proj/locations/region/clusters/clust/instances/inst",
			},
			want: withDefaults(&proxy.Config{
				PublicIP:  true,
				Instances: []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "Public IP query param (key only)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst?public-ip",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					PublicIP: pointer(true),
					Name:     "projects/proj/locations/region/clusters/clust/instances/inst",
				}},
			}),
		},
		{
			desc: "Public IP query param (t & f)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst1?public-ip=t",
				"projects/proj/locations/region/clusters/clust/instances/inst2?public-ip=f",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					PublicIP: pointer(true),
					Name:     "projects/proj/locations/region/clusters/clust/instances/inst1",
				}, {
					PublicIP: pointer(false),
					Name:     "projects/proj/locations/region/clusters/clust/instances/inst2",
				}},
			}),
		},
		{
			desc: "Public IP query param (true & false)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst1?public-ip=true",
				"projects/proj/locations/region/clusters/clust/instances/inst2?public-ip=false",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					PublicIP: pointer(true),
					Name:     "projects/proj/locations/region/clusters/clust/instances/inst1",
				}, {
					PublicIP: pointer(false),
					Name:     "projects/proj/locations/region/clusters/clust/instances/inst2",
				}},
			}),
		},
		{
			desc: "PSC",
			args: []string{
				"--psc",
				"projects/proj/locations/region/clusters/clust/instances/inst",
			},
			want: withDefaults(&proxy.Config{
				PSC:       true,
				Instances: []proxy.InstanceConnConfig{{Name: "projects/proj/locations/region/clusters/clust/instances/inst"}},
			}),
		},
		{
			desc: "PSC query param (true & false)",
			args: []string{
				"projects/proj/locations/region/clusters/clust/instances/inst1?psc=true",
				"projects/proj/locations/region/clusters/clust/instances/inst2?psc=false",
			},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					PSC:  pointer(true),
					Name: "projects/proj/locations/region/clusters/clust/instances/inst1",
				}, {
					PSC:  pointer(false),
					Name: "projects/proj/locations/region/clusters/clust/instances/inst2",
				}},
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
			desc: "using the unix socket path query param",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket-path=/path/to/file"},
			want: withDefaults(&proxy.Config{
				Instances: []proxy.InstanceConnConfig{{
					UnixSocketPath: "/path/to/file",
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
			desc: "using min-sigterm-delay flag",
			args: []string{"--min-sigterm-delay", "10s", "projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				WaitBeforeClose: 10 * time.Second,
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
				"sv1@developer.gserviceaccount.com",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				ImpersonationChain: "sv1@developer.gserviceaccount.com",
			}),
		},
		{
			desc: "using the debug flag",
			args: []string{"--debug",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				Debug: true,
			}),
		},
		{
			desc: "using the lazy refresh flag",
			args: []string{"--lazy-refresh",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				LazyRefresh: true,
			}),
		},
		{
			desc: "using the admin port flag",
			args: []string{"--admin-port", "7777",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				AdminPort: "7777",
			}),
		},
		{
			desc: "using the quitquitquit flag",
			args: []string{"--quitquitquit",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				QuitQuitQuit: true,
			}),
		},
		{
			desc: "using the run-connection-test flag",
			args: []string{"--run-connection-test",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				RunConnectionTest: true,
			}),
		},
		{
			desc: "using the debug logs flag",
			args: []string{"--debug-logs",
				"projects/proj/locations/region/clusters/clust/instances/inst"},
			want: withDefaults(&proxy.Config{
				DebugLogs: true,
			}),
		},
		{
			desc: "using the static connection info flag",
			args: []string{
				"--static-connection-info",
				"myfile.json",
				"projects/proj/locations/region/clusters/clust/instances/inst",
			},
			want: withDefaults(&proxy.Config{
				StaticConnectionInfo: "myfile.json",
			}),
		},
		{
			desc: "using the disable built-in telemetry flag",
			args: []string{
				"--disable-built-in-telemetry",
				"projects/proj/locations/region/clusters/clust/instances/inst",
			},
			want: withDefaults(&proxy.Config{
				DisableBuiltInTelemetry: true,
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

func TestNewCommandWithEnvironmentConfig(t *testing.T) {
	tcs := []struct {
		desc     string
		envName  string
		envValue string
		want     *proxy.Config
	}{
		{
			desc:     "using the address envvar",
			envName:  "ALLOYDB_PROXY_ADDRESS",
			envValue: "0.0.0.0",
			want: withDefaults(&proxy.Config{
				Addr: "0.0.0.0",
			}),
		},
		{
			desc:     "using the port envvar",
			envName:  "ALLOYDB_PROXY_PORT",
			envValue: "6000",
			want: withDefaults(&proxy.Config{
				Port: 6000,
			}),
		},
		{
			desc:     "using the token envvar",
			envName:  "ALLOYDB_PROXY_TOKEN",
			envValue: "MYCOOLTOKEN",
			want: withDefaults(&proxy.Config{
				Token: "MYCOOLTOKEN",
			}),
		},
		{
			desc:     "using the credentiale file envvar",
			envName:  "ALLOYDB_PROXY_CREDENTIALS_FILE",
			envValue: "/path/to/file",
			want: withDefaults(&proxy.Config{
				CredentialsFile: "/path/to/file",
			}),
		},
		{
			desc:     "using the JSON credentials",
			envName:  "ALLOYDB_PROXY_JSON_CREDENTIALS",
			envValue: `{"json":"goes-here"}`,
			want: withDefaults(&proxy.Config{
				CredentialsJSON: `{"json":"goes-here"}`,
			}),
		},
		{
			desc:     "using the gcloud auth envvar",
			envName:  "ALLOYDB_PROXY_GCLOUD_AUTH",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				GcloudAuth: true,
			}),
		},
		{
			desc:     "using the public-ip envvar",
			envName:  "ALLOYDB_PROXY_PUBLIC_IP",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				PublicIP: true,
			}),
		},
		{
			desc:     "using the api-endpoint envvar",
			envName:  "ALLOYDB_PROXY_ALLOYDBADMIN_API_ENDPOINT",
			envValue: "https://test.googleapis.com/",
			want: withDefaults(&proxy.Config{
				APIEndpointURL: "https://test.googleapis.com",
			}),
		},
		{
			desc:     "using the unix socket envvar",
			envName:  "ALLOYDB_PROXY_UNIX_SOCKET",
			envValue: "/path/to/dir/",
			want: withDefaults(&proxy.Config{
				UnixSocket: "/path/to/dir/",
			}),
		},
		{
			desc:     "enabling structured logging",
			envName:  "ALLOYDB_PROXY_STRUCTURED_LOGS",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				StructuredLogs: true,
			}),
		},
		{
			desc:     "using the max connections envvar",
			envName:  "ALLOYDB_PROXY_MAX_CONNECTIONS",
			envValue: "1",
			want: withDefaults(&proxy.Config{
				MaxConnections: 1,
			}),
		},
		{
			desc:     "using wait after signterm envvar",
			envName:  "ALLOYDB_PROXY_MAX_SIGTERM_DELAY",
			envValue: "10s",
			want: withDefaults(&proxy.Config{
				WaitOnClose: 10 * time.Second,
			}),
		},
		{
			desc:     "using the imopersonate service accounn envvar",
			envName:  "ALLOYDB_PROXY_IMPERSONATE_SERVICE_ACCOUNT",
			envValue: "sv1@developer.gserviceaccount.com",
			want: withDefaults(&proxy.Config{
				ImpersonationChain: "sv1@developer.gserviceaccount.com",
			}),
		},
		{
			desc:     "using the disable traces envvar",
			envName:  "ALLOYDB_PROXY_DISABLE_TRACES",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				DisableTraces: true,
			}),
		},
		{
			desc:     "using the telemetry sample rate envvar",
			envName:  "ALLOYDB_PROXY_TELEMETRY_SAMPLE_RATE",
			envValue: "500",
			want: withDefaults(&proxy.Config{
				TelemetryTracingSampleRate: 500,
			}),
		},
		{
			desc:     "using the disable metrics envvar",
			envName:  "ALLOYDB_PROXY_DISABLE_METRICS",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				DisableMetrics: true,
			}),
		},
		{
			desc:     "using the telemetry project envvar",
			envName:  "ALLOYDB_PROXY_TELEMETRY_PROJECT",
			envValue: "mycoolproject",
			want: withDefaults(&proxy.Config{
				TelemetryProject: "mycoolproject",
			}),
		},
		{
			desc:     "using the telemetry prefix envvar",
			envName:  "ALLOYDB_PROXY_TELEMETRY_PREFIX",
			envValue: "myprefix",
			want: withDefaults(&proxy.Config{
				TelemetryPrefix: "myprefix",
			}),
		},
		{
			desc:     "using the prometheus envvar",
			envName:  "ALLOYDB_PROXY_PROMETHEUS",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				Prometheus: true,
			}),
		},
		{
			desc:     "using the prometheus namespace envvar",
			envName:  "ALLOYDB_PROXY_PROMETHEUS_NAMESPACE",
			envValue: "myns",
			want: withDefaults(&proxy.Config{
				PrometheusNamespace: "myns",
			}),
		},
		{
			desc:     "using the health check envvar",
			envName:  "ALLOYDB_PROXY_HEALTH_CHECK",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				HealthCheck: true,
			}),
		},
		{
			desc:     "using the http address envvar",
			envName:  "ALLOYDB_PROXY_HTTP_ADDRESS",
			envValue: "0.0.0.0",
			want: withDefaults(&proxy.Config{
				HTTPAddress: "0.0.0.0",
			}),
		},
		{
			desc:     "using the http port envvar",
			envName:  "ALLOYDB_PROXY_HTTP_PORT",
			envValue: "5555",
			want: withDefaults(&proxy.Config{
				HTTPPort: "5555",
			}),
		},
		{
			desc:     "using the debug envvar",
			envName:  "ALLOYDB_PROXY_DEBUG",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				Debug: true,
			}),
		},
		{
			desc:     "using the admin port envvar",
			envName:  "ALLOYDB_PROXY_ADMIN_PORT",
			envValue: "7777",
			want: withDefaults(&proxy.Config{
				AdminPort: "7777",
			}),
		},
		{
			desc:     "using the quitquitquit envvar",
			envName:  "ALLOYDB_PROXY_QUITQUITQUIT",
			envValue: "true",
			want: withDefaults(&proxy.Config{
				QuitQuitQuit: true,
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
				"ALLOYDB_PROXY_INSTANCE_URI": u,
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u},
			}}),
		},
		{
			desc: "with multiple instance connection names",
			env: map[string]string{
				"ALLOYDB_PROXY_INSTANCE_URI_0": u + "0",
				"ALLOYDB_PROXY_INSTANCE_URI_1": u + "1",
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "0"},
				{Name: u + "1"},
			}}),
		},
		{
			desc: "when the index skips a number",
			env: map[string]string{
				"ALLOYDB_PROXY_INSTANCE_URI_0": u + "0",
				"ALLOYDB_PROXY_INSTANCE_URI_2": u + "2",
			},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "0"},
			}}),
		},
		{
			desc: "when there are CLI args provided",
			env: map[string]string{
				"ALLOYDB_PROXY_INSTANCE_URI": u,
			},
			args: []string{u + "1"},
			want: withDefaults(&proxy.Config{Instances: []proxy.InstanceConnConfig{
				{Name: u + "1"},
			}}),
		},
		{
			desc: "when only an index instance connection name is defined",
			env: map[string]string{
				"ALLOYDB_PROXY_INSTANCE_URI_0": u,
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
			desc: "when the instance uri is bogus",
			args: []string{"projects/proj/locations/region/clusters/clust/"},
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
			desc: "using the unix socket and unix-socket-path",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket=/path&unix-socket-path=/another/path"},
		},
		{
			desc: "using the unix socket path and addr query params",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket-path=/path&address=127.0.0.1"},
		},
		{
			desc: "using the unix socket path and port query params",
			args: []string{"projects/proj/locations/region/clusters/clust/instances/inst?unix-socket-path=/path&port=5000"},
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
		{
			desc: "run-connection-test with fuse",
			args: []string{
				"--run-connection-test",
				"--fuse", "myfusedir",
			},
		}}

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
				time.Sleep(100 * time.Millisecond)
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

func tryDial(method, addr string) (*http.Response, error) {
	var (
		resp     *http.Response
		attempts int
		err      error
	)
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	req := &http.Request{Method: method, URL: u}
	for {
		if attempts > 10 {
			return resp, err
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			attempts++
			time.Sleep(time.Second)
			continue
		}
		return resp, err
	}
}

func TestPrometheusMetricsEndpoint(t *testing.T) {
	c := NewCommand(WithDialer(&spyDialer{}))
	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{"--prometheus", "projects/my-project/locations/my-region/clusters/my-cluster/instances/my-instance?port=5321"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go c.ExecuteContext(ctx)

	// try to dial metrics server for a max of ~10s to give the proxy time to
	// start up.
	resp, err := tryDial("GET", "http://localhost:9090/metrics") // default port set by http-port flag
	if err != nil {
		t.Fatalf("failed to dial metrics endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}
}

func TestPProfServer(t *testing.T) {
	c := NewCommand(WithDialer(&spyDialer{}))
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{"--debug", "--admin-port", "9191",
		"projects/proj/locations/region/clusters/clust/instances/inst?port=5323"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.ExecuteContext(ctx)
	resp, err := tryDial("GET", "http://localhost:9191/debug/pprof/")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}
}

func TestQuitQuitQuitHTTPPost(t *testing.T) {
	c := NewCommand(WithDialer(&spyDialer{}))
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{"--quitquitquit", "--admin-port", "9192",
		"projects/proj/locations/region/clusters/clust/instances/inst"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		err := c.ExecuteContext(ctx)
		errCh <- err
	}()
	resp, err := tryDial("HEAD", "http://localhost:9192/quitquitquit")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected a 400 status, got = %v", resp.StatusCode)
	}
	resp, err = tryDial("POST", "http://localhost:9192/quitquitquit")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}

	var gotErr error
	select {
	case err := <-errCh:
		gotErr = err
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for error")
	}

	if !errors.Is(gotErr, errQuitQuitQuit) {
		t.Fatalf("want = %v, got = %v", errQuitQuitQuit, gotErr)
	}
}

func TestQuitQuitQuitGet(t *testing.T) {
	c := NewCommand(WithDialer(&spyDialer{}))
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{"--quitquitquit", "--admin-port", "9193",
		"projects/proj/locations/region/clusters/clust/instances/inst"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		err := c.ExecuteContext(ctx)
		errCh <- err
	}()
	resp, err := tryDial("HEAD", "http://localhost:9193/quitquitquit")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected a 400 status, got = %v", resp.StatusCode)
	}
	resp, err = tryDial("GET", "http://localhost:9193/quitquitquit")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}
	if want, got := errQuitQuitQuit, <-errCh; !errors.Is(got, want) {
		t.Fatalf("want = %v, got = %v", want, got)
	}
}

type errorDialer struct {
	spyDialer
}

var errCloseFailed = errors.New("close failed")

func (*errorDialer) Close() error {
	return errCloseFailed
}

func TestQuitQuitQuitWithErrors(t *testing.T) {
	c := NewCommand(WithDialer(&errorDialer{}))
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{
		"--quitquitquit", "--admin-port", "9194",
		"projects/proj/locations/region/clusters/clust/instances/inst"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		err := c.ExecuteContext(ctx)
		errCh <- err
	}()
	resp, err := tryDial("POST", "http://localhost:9194/quitquitquit")
	if err != nil {
		t.Fatalf("failed to dial endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected a 200 status, got = %v", resp.StatusCode)
	}
	// The returned error is the error from closing the dialer.
	got := <-errCh
	if !strings.Contains(got.Error(), "close failed") {
		t.Fatalf("want = %v, got = %v", errCloseFailed, got)
	}
}
