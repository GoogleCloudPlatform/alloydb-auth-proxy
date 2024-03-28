// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import "testing"

func assert[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNewCommandWithConfigFile(t *testing.T) {
	tcs := []struct {
		desc   string
		args   []string
		setup  func()
		assert func(t *testing.T, c *Command)
	}{
		{
			desc:  "toml config file",
			args:  []string{"--config-file", "testdata/config-toml.toml"},
			setup: func() {},
			assert: func(t *testing.T, c *Command) {
				assert(t, 1, len(c.conf.Instances))
				assert(t, true, c.conf.Debug)
				assert(t, 5555, c.conf.Port)
				assert(t, true, c.conf.DebugLogs)
				assert(t, true, c.conf.AutoIAMAuthN)
			},
		},
		{
			desc:  "yaml config file",
			args:  []string{"--config-file", "testdata/config-yaml.yaml"},
			setup: func() {},
			assert: func(t *testing.T, c *Command) {
				assert(t, 1, len(c.conf.Instances))
				assert(t, true, c.conf.Debug)
			},
		},
		{
			desc:  "json config file",
			args:  []string{"--config-file", "testdata/config-json.json"},
			setup: func() {},
			assert: func(t *testing.T, c *Command) {
				assert(t, 1, len(c.conf.Instances))
				assert(t, true, c.conf.Debug)
			},
		},
		{
			desc:  "config file with two instances",
			args:  []string{"--config-file", "testdata/two-instances.toml"},
			setup: func() {},
			assert: func(t *testing.T, c *Command) {
				assert(t, 2, len(c.conf.Instances))
			},
		},
		{
			desc: "argument takes precedence over environment variable",
			args: []string{sampleURI},
			setup: func() {
				t.Setenv("ALLOYDB_PROXY_INSTANCE_URI", altURI)
			},
			assert: func(t *testing.T, c *Command) {
				assert(t, sampleURI, c.conf.Instances[0].Name)
			},
		},
		{
			desc: "environment variable takes precedence over config file",
			args: []string{"--config-file", "testdata/config.json"},
			setup: func() {
				t.Setenv("ALLOYDB_PROXY_INSTANCE_URI", altURI)
			},
			assert: func(t *testing.T, c *Command) {
				assert(t, altURI, c.conf.Instances[0].Name)
			},
		},
		{
			desc: "CLI flag takes precedence over environment variable",
			args: []string{sampleURI, "--debug"},
			setup: func() {
				t.Setenv("ALLOYDB_PROXY_DEBUG", "false")
			},
			assert: func(t *testing.T, c *Command) {
				assert(t, true, c.conf.Debug)
			},
		},
		{
			desc: "CLI flag takes precedence over config file",
			args: []string{
				sampleURI,
				"--config-file", "testdata/config.toml",
				"--debug=false",
			},
			setup: func() {},
			assert: func(t *testing.T, c *Command) {
				assert(t, false, c.conf.Debug)
			},
		},
		{
			desc: "environment variable takes precedence over config file",
			args: []string{
				sampleURI,
				"--config-file", "testdata/config.toml",
			},
			setup: func() {
				t.Setenv("ALLOYDB_PROXY_DEBUG", "false")
			},
			assert: func(t *testing.T, c *Command) {
				assert(t, false, c.conf.Debug)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			tc.setup()

			cmd, err := invokeProxyCommand(tc.args)
			if err != nil {
				t.Fatalf("want error = nil, got = %v", err)
			}

			tc.assert(t, cmd)
		})
	}
}
