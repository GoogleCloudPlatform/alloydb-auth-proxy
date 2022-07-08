// Copyright 2021 Google LLC
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

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/gcloud"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// InstanceConnConfig holds the configuration for an individual instance
// connection.
type InstanceConnConfig struct {
	// Name is the instance connection name.
	Name string
	// Addr is the address on which to bind a listener for the instance.
	Addr string
	// Port is the port on which to bind a listener for the instance.
	Port int
	// UnixSocket is the directory where a Unix socket will be created,
	// connected to the Cloud SQL instance. If set, takes precedence over Addr
	// and Port.
	UnixSocket string
}

// Config contains all the configuration provided by the caller.
type Config struct {
	// UserAgent is the user agent to use when connecting to the cloudsql instance
	UserAgent string

	// Token is the Bearer token used for authorization.
	Token string

	// CredentialsFile is the path to a service account key.
	CredentialsFile string

	// GcloudAuth set whether to use Gcloud's config helper to retrieve a
	// token for authentication.
	GcloudAuth bool

	// Addr is the address on which to bind all instances.
	Addr string

	// Port is the initial port to bind to. Subsequent instances bind to
	// increments from this value.
	Port int

	// UnixSocket is the directory where Unix sockets will be created,
	// connected to any Instances. If set, takes precedence over Addr and Port.
	UnixSocket string

	// Instances are configuration for individual instances. Instance
	// configuration takes precedence over global configuration.
	Instances []InstanceConnConfig

	// Dialer specifies the dialer to use when connecting to AlloyDB
	// instances.
	Dialer alloydb.Dialer
}

// DialerOptions builds appropriate list of options from the Config
// values for use by alloydbconn.NewClient()
func (c *Config) DialerOptions() ([]alloydbconn.Option, error) {
	opts := []alloydbconn.Option{
		alloydbconn.WithUserAgent(c.UserAgent),
	}
	switch {
	case c.Token != "":
		opts = append(opts, alloydbconn.WithTokenSource(
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
		))
	case c.CredentialsFile != "":
		opts = append(opts, alloydbconn.WithCredentialsFile(
			c.CredentialsFile,
		))
	case c.GcloudAuth:
		ts, err := gcloud.TokenSource()
		if err != nil {
			return nil, err
		}
		opts = append(opts, alloydbconn.WithTokenSource(ts))
	default:
	}

	return opts, nil
}

type portConfig struct {
	global int
}

func newPortConfig(global int) *portConfig {
	return &portConfig{
		global: global,
	}
}

// nextPort returns the next port based on the initial global value.
func (c *portConfig) nextPort() int {
	p := c.global
	c.global++
	return p
}

var (
	// Instance URI is in the format:
	// '/projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
	// Additionally, we have to support legacy "domain-scoped" projects (e.g. "google.com:PROJECT")
	instURIRegex = regexp.MustCompile("projects/([^:]+(:[^:]+)?)/locations/([^:]+)/clusters/([^:]+)/instances/([^:]+)")
)

// UnixSocketDir returns a shorted instance connection name to prevent exceeding
// the Unix socket length.
func UnixSocketDir(dir, inst string) (string, error) {
	m := instURIRegex.FindSubmatch([]byte(inst))
	if m == nil {
		return "", fmt.Errorf("invalid instance name: %v", inst)
	}
	project := string(m[1])
	region := string(m[3])
	cluster := string(m[4])
	name := string(m[5])
	shortName := strings.Join([]string{project, region, cluster, name}, ".")
	return filepath.Join(dir, shortName), nil
}

// Client represents the state of the current instantiation of the proxy.
type Client struct {
	cmd    *cobra.Command
	dialer alloydb.Dialer

	// mnts is a list of all mounted sockets for this client
	mnts []*socketMount
}

// NewClient completes the initial setup required to get the proxy to a "steady" state.
func NewClient(ctx context.Context, cmd *cobra.Command, conf *Config) (*Client, error) {
	// Check if the caller has configured a dialer.
	// Otherwise, initialize a new one.
	d := conf.Dialer
	if d == nil {
		var err error
		dialerOpts, err := conf.DialerOptions()
		if err != nil {
			return nil, fmt.Errorf("error initializing dialer: %v", err)
		}
		d, err = alloydbconn.NewDialer(ctx, dialerOpts...)
		if err != nil {
			return nil, fmt.Errorf("error initializing dialer: %v", err)
		}
	}

	pc := newPortConfig(conf.Port)
	var mnts []*socketMount
	for _, inst := range conf.Instances {
		var (
			// network is one of "tcp" or "unix"
			network string
			// address is either a TCP host port, or a Unix socket
			address string
		)
		// IF
		//   a global Unix socket directory is NOT set AND
		//   an instance-level Unix socket is NOT set
		//   (e.g.,  I didn't set a Unix socket globally or for this instance)
		// OR
		//   an instance-level TCP address or port IS set
		//   (e.g., I'm overriding any global settings to use TCP for this
		//   instance)
		// use a TCP listener.
		// Otherwise, use a Unix socket.
		if (conf.UnixSocket == "" && inst.UnixSocket == "") ||
			(inst.Addr != "" || inst.Port != 0) {
			network = "tcp"

			a := conf.Addr
			if inst.Addr != "" {
				a = inst.Addr
			}

			var np int
			switch {
			case inst.Port != 0:
				np = inst.Port
			case conf.Port != 0:
				np = pc.nextPort()
			default:
				np = pc.nextPort()
			}

			address = net.JoinHostPort(a, fmt.Sprint(np))
		} else {
			network = "unix"

			dir := conf.UnixSocket
			if dir == "" {
				dir = inst.UnixSocket
			}
			ud, err := UnixSocketDir(dir, inst.Name)
			if err != nil {
				return nil, err
			}
			// Create the parent directory that will hold the socket.
			if _, err := os.Stat(ud); err != nil {
				if err = os.Mkdir(ud, 0777); err != nil {
					return nil, err
				}
			}
			// use the Postgres-specific socket name
			address = filepath.Join(ud, ".s.PGSQL.5432")
		}

		m := &socketMount{inst: inst.Name}
		addr, err := m.listen(ctx, network, address)
		if err != nil {
			for _, m := range mnts {
				m.close()
			}
			return nil, fmt.Errorf("[%v] Unable to mount socket: %v", inst.Name, err)
		}

		cmd.Printf("[%s] Listening on %s\n", inst.Name, addr.String())
		mnts = append(mnts, m)
	}

	return &Client{mnts: mnts, cmd: cmd, dialer: d}, nil
}

// Serve listens on the mounted ports and beging proxying the connections to the instances.
func (c *Client) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	exitCh := make(chan error)
	for _, m := range c.mnts {
		go func(mnt *socketMount) {
			err := c.serveSocketMount(ctx, mnt)
			if err != nil {
				select {
				// Best effort attempt to send error.
				// If this send fails, it means the reading goroutine has
				// already pulled a value out of the channel and is no longer
				// reading any more values. In other words, we report only the
				// first error.
				case exitCh <- err:
				default:
					return
				}
			}
		}(m)
	}
	return <-exitCh
}

// Close triggers the proxyClient to shutdown.
func (c *Client) Close() {
	defer c.dialer.Close()
	for _, m := range c.mnts {
		m.close()
	}
}

// serveSocketMount persistently listens to the socketMounts listener and proxies connections to a
// given AlloyDB instance.
func (c *Client) serveSocketMount(ctx context.Context, s *socketMount) error {
	if s.listener == nil {
		return fmt.Errorf("[%s] mount doesn't have a listener set", s.inst)
	}
	for {
		cConn, err := s.listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				c.cmd.PrintErrf("[%s] Error accepting connection: %v\n", s.inst, err)
				// For transient errors, wait a small amount of time to see if it resolves itself
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}
		// handle the connection in a separate goroutine
		go func() {
			c.cmd.Printf("[%s] accepted connection from %s\n", s.inst, cConn.RemoteAddr())

			// give a max of 30 seconds to connect to the instance
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			sConn, err := c.dialer.Dial(ctx, s.inst)
			if err != nil {
				c.cmd.Printf("[%s] failed to connect to instance: %v\n", s.inst, err)
				cConn.Close()
				return
			}
			c.proxyConn(s.inst, cConn, sConn)
		}()
	}
}

// socketMount is a tcp/unix socket that listens for an AlloyDB instance.
type socketMount struct {
	inst     string
	listener net.Listener
}

// listen causes a socketMount to create a Listener at the specified network address.
func (s *socketMount) listen(ctx context.Context, network string, address string) (net.Addr, error) {
	lc := net.ListenConfig{KeepAlive: 30 * time.Second}
	l, err := lc.Listen(ctx, network, address)
	if err != nil {
		return nil, err
	}
	s.listener = l
	return s.listener.Addr(), nil
}

// close stops the mount from listening for any more connections
func (s *socketMount) close() error {
	err := s.listener.Close()
	s.listener = nil
	return err
}

// proxyConn sets up a bidirectional copy between two open connections
func (c *Client) proxyConn(inst string, client, server net.Conn) {
	// only allow the first side to give an error for terminating a connection
	var o sync.Once
	cleanup := func(errDesc string, isErr bool) {
		o.Do(func() {
			client.Close()
			server.Close()
			if isErr {
				c.cmd.PrintErrln(errDesc)
			} else {
				c.cmd.Println(errDesc)
			}
		})
	}

	// copy bytes from client to server
	go func() {
		buf := make([]byte, 8*1024) // 8kb
		for {
			n, cErr := client.Read(buf)
			var sErr error
			if n > 0 {
				_, sErr = server.Write(buf[:n])
			}
			switch {
			case cErr == io.EOF:
				cleanup(fmt.Sprintf("[%s] client closed the connection", inst), false)
				return
			case cErr != nil:
				cleanup(fmt.Sprintf("[%s] connection aborted - error reading from client: %v", inst, cErr), true)
				return
			case sErr == io.EOF:
				cleanup(fmt.Sprintf("[%s] instance closed the connection", inst), false)
				return
			case sErr != nil:
				cleanup(fmt.Sprintf("[%s] connection aborted - error writing to instance: %v", inst, cErr), true)
				return
			}
		}
	}()

	// copy bytes from server to client
	buf := make([]byte, 8*1024) // 8kb
	for {
		n, sErr := server.Read(buf)
		var cErr error
		if n > 0 {
			_, cErr = client.Write(buf[:n])
		}
		switch {
		case sErr == io.EOF:
			cleanup(fmt.Sprintf("[%s] instance closed the connection", inst), false)
			return
		case sErr != nil:
			cleanup(fmt.Sprintf("[%s] connection aborted - error reading from instance: %v", inst, sErr), true)
			return
		case cErr == io.EOF:
			cleanup(fmt.Sprintf("[%s] client closed the connection", inst), false)
			return
		case cErr != nil:
			cleanup(fmt.Sprintf("[%s] connection aborted - error writing to client: %v", inst, sErr), true)
			return
		}
	}
}
