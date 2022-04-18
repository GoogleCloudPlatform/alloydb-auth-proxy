// Copyright 2020 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydbconn

import (
	"context"
	"crypto/rsa"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"cloud.google.com/go/alloydbconn/errtype"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	apiopt "google.golang.org/api/option"
)

// An Option is an option for configuring a Dialer.
type Option func(d *dialerConfig)

type dialerConfig struct {
	rsaKey         *rsa.PrivateKey
	adminOpts      []apiopt.ClientOption
	dialOpts       []DialOption
	dialFunc       func(ctx context.Context, network, addr string) (net.Conn, error)
	refreshTimeout time.Duration
	tokenSource    oauth2.TokenSource
	useragents     []string
	// err tracks any dialer options that may have failed.
	err error
}

// WithOptions turns a list of Option's into a single Option.
func WithOptions(opts ...Option) Option {
	return func(d *dialerConfig) {
		for _, opt := range opts {
			opt(d)
		}
	}
}

// WithCredentialsFile returns an Option that specifies a service account
// or refresh token JSON credentials file to be used as the basis for
// authentication.
func WithCredentialsFile(filename string) Option {
	return func(d *dialerConfig) {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			d.err = errtype.NewConfigError(err.Error(), "n/a")
			return
		}
		opt := WithCredentialsJSON(b)
		opt(d)
	}
}

// WithCredentialsJSON returns an Option that specifies a service account
// or refresh token JSON credentials to be used as the basis for authentication.
func WithCredentialsJSON(b []byte) Option {
	return func(d *dialerConfig) {
		c, err := google.CredentialsFromJSON(context.Background(), b) // TODO: add AlloyDB scope
		if err != nil {
			d.err = errtype.NewConfigError(err.Error(), "n/a")
			return
		}
		d.tokenSource = c.TokenSource
		d.adminOpts = append(d.adminOpts, apiopt.WithCredentials(c))
	}
}

// WithUserAgent returns an Option that sets the User-Agent.
func WithUserAgent(ua string) Option {
	return func(d *dialerConfig) {
		d.useragents = append(d.useragents, ua)
	}
}

// WithDefaultDialOptions returns an Option that specifies the default
// DialOptions used.
func WithDefaultDialOptions(opts ...DialOption) Option {
	return func(d *dialerConfig) {
		d.dialOpts = append(d.dialOpts, opts...)
	}
}

// WithTokenSource returns an Option that specifies an OAuth2 token source
// to be used as the basis for authentication.
func WithTokenSource(s oauth2.TokenSource) Option {
	return func(d *dialerConfig) {
		d.tokenSource = s
		d.adminOpts = append(d.adminOpts, apiopt.WithTokenSource(s))
	}
}

// WithRSAKey returns an Option that specifies a rsa.PrivateKey used to represent the client.
func WithRSAKey(k *rsa.PrivateKey) Option {
	return func(d *dialerConfig) {
		d.rsaKey = k
	}
}

// WithRefreshTimeout returns an Option that sets a timeout on refresh operations. Defaults to 30s.
func WithRefreshTimeout(t time.Duration) Option {
	return func(d *dialerConfig) {
		d.refreshTimeout = t
	}
}

// WithHTTPClient configures the underlying AlloyDB Admin API client with the
// provided HTTP client. This option is generally unnecessary except for
// advanced use-cases.
func WithHTTPClient(client *http.Client) Option {
	return func(d *dialerConfig) {
		d.adminOpts = append(d.adminOpts, apiopt.WithHTTPClient(client))
	}
}

// WithAdminAPIEndpoint configures the underlying AlloyDB Admin API client to
// use the provided URL.
func WithAdminAPIEndpoint(url string) Option {
	return func(d *dialerConfig) {
		d.adminOpts = append(d.adminOpts, apiopt.WithEndpoint(url))
	}
}

// WithDialFunc configures the function used to connect to the address on the
// named network. This option is generally unnecessary except for advanced
// use-cases.
func WithDialFunc(dial func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(d *dialerConfig) {
		d.dialFunc = dial
	}
}

// A DialOption is an option for configuring how a Dialer's Dial call is executed.
type DialOption func(d *dialCfg)

type dialCfg struct {
	tcpKeepAlive time.Duration
}

// DialOptions turns a list of DialOption instances into an DialOption.
func DialOptions(opts ...DialOption) DialOption {
	return func(cfg *dialCfg) {
		for _, opt := range opts {
			opt(cfg)
		}
	}
}

// WithTCPKeepAlive returns a DialOption that specifies the tcp keep alive period for the connection returned by Dial.
func WithTCPKeepAlive(d time.Duration) DialOption {
	return func(cfg *dialCfg) {
		cfg.tcpKeepAlive = d
	}
}
