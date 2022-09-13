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
	"sync/atomic"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/gcloud"
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
	// connected to the AlloyDB instance. If set, takes precedence over Addr
	// and Port.
	UnixSocket string
}

// Config contains all the configuration provided by the caller.
type Config struct {
	// UserAgent is the user agent to use when sending requests to the Admin
	// API.
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

	// APIEndpointURL is the URL of the AlloyDB Admin API.
	APIEndpointURL string

	// Instances are configuration for individual instances. Instance
	// configuration takes precedence over global configuration.
	Instances []InstanceConnConfig

	// MaxConnections are the maximum number of connections the Client may
	// establish to the AlloyDB server side proxy before refusing additional
	// connections. A zero-value indicates no limit.
	MaxConnections uint64

	// WaitOnClose sets the duration to wait for connections to close before
	// shutting down. Not setting this field means to close immediately
	// regardless of any open connections.
	WaitOnClose time.Duration

	// StructuredLogs sets all output to use JSON in the LogEntry format.
	// See https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
	StructuredLogs bool
}

// DialerOptions builds appropriate list of options from the Config
// values for use by alloydbconn.NewClient()
func (c *Config) DialerOptions(l alloydb.Logger) ([]alloydbconn.Option, error) {
	opts := []alloydbconn.Option{
		alloydbconn.WithUserAgent(c.UserAgent),
	}
	opts = append(opts, alloydbconn.WithAdminAPIEndpoint(c.APIEndpointURL))
	switch {
	case c.Token != "":
		l.Infof("Authorizing with the -token flag")
		opts = append(opts, alloydbconn.WithTokenSource(
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
		))
	case c.CredentialsFile != "":
		l.Infof("Authorizing with the credentials file at %q", c.CredentialsFile)
		opts = append(opts, alloydbconn.WithCredentialsFile(
			c.CredentialsFile,
		))
	case c.GcloudAuth:
		l.Infof("Authorizing with gcloud user credentials")
		ts, err := gcloud.TokenSource()
		if err != nil {
			return nil, err
		}
		opts = append(opts, alloydbconn.WithTokenSource(ts))
	default:
		l.Infof("Authorizing with Application Default Credentials")
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
	// 'projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
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

// Client proxies connections from a local client to the remote server side
// proxy for multiple AlloyDB instances.
type Client struct {
	// connCount tracks the number of all open connections from the Client to
	// all AlloyDB instances.
	connCount uint64

	// maxConns is the maximum number of allowed connections tracked by
	// connCount. If not set, there is no limit.
	maxConns uint64

	dialer alloydb.Dialer

	// mnts is a list of all mounted sockets for this client
	mnts []*socketMount

	// waitOnClose is the maximum duration to wait for open connections to close
	// when shutting down.
	waitOnClose time.Duration

	logger alloydb.Logger
}

// NewClient completes the initial setup required to get the proxy to a "steady" state.
func NewClient(ctx context.Context, d alloydb.Dialer, l alloydb.Logger, conf *Config) (*Client, error) {
	// Check if the caller has configured a dialer.
	// Otherwise, initialize a new one.
	if d == nil {
		dialerOpts, err := conf.DialerOptions(l)
		if err != nil {
			return nil, fmt.Errorf("error initializing dialer: %v", err)
		}
		d, err = alloydbconn.NewDialer(ctx, dialerOpts...)
		if err != nil {
			return nil, fmt.Errorf("error initializing dialer: %v", err)
		}
	}

	var mnts []*socketMount
	pc := newPortConfig(conf.Port)
	for _, inst := range conf.Instances {
		m, err := newSocketMount(ctx, conf, pc, inst)
		if err != nil {
			for _, m := range mnts {
				mErr := m.Close()
				if mErr != nil {
					l.Errorf("failed to close mount: %v", mErr)
				}
			}
			return nil, fmt.Errorf("[%v] Unable to mount socket: %v", inst.Name, err)
		}

		l.Infof("[%s] Listening on %s", inst.Name, m.Addr())
		mnts = append(mnts, m)
	}

	c := &Client{
		mnts:        mnts,
		logger:      l,
		dialer:      d,
		maxConns:    conf.MaxConnections,
		waitOnClose: conf.WaitOnClose,
	}
	return c, nil
}

// CheckConnections dials each registered instance and reports any errors that
// may have occurred.
func (c *Client) CheckConnections(ctx context.Context) error {
	var (
		wg    sync.WaitGroup
		errCh = make(chan error, len(c.mnts))
	)
	for _, mnt := range c.mnts {
		wg.Add(1)
		go func(m *socketMount) {
			defer wg.Done()
			conn, err := c.dialer.Dial(ctx, m.inst, m.dialOpts...)
			if err != nil {
				errCh <- err
				return
			}
			cErr := conn.Close()
			if err != nil {
				errCh <- fmt.Errorf("%v: %v", m.inst, cErr)
			}
		}(mnt)
	}
	wg.Wait()

	var mErr MultiErr
	for i := 0; i < len(c.mnts); i++ {
		select {
		case err := <-errCh:
			mErr = append(mErr, err)
		default:
			continue
		}
	}
	if len(mErr) > 0 {
		return mErr
	}
	return nil
}

// ConnCount returns the number of open connections and the maximum allowed
// connections. Returns 0 when the maximum allowed connections have not been set.
func (c *Client) ConnCount() (uint64, uint64) {
	return atomic.LoadUint64(&c.connCount), c.maxConns
}

// Serve starts proxying connections for all configured instances using the
// associated socket.
func (c *Client) Serve(ctx context.Context, notify func()) error {
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
	notify()
	return <-exitCh
}

// MultiErr is a group of errors wrapped into one.
type MultiErr []error

// Error returns a single string representing one or more errors.
func (m MultiErr) Error() string {
	l := len(m)
	if l == 1 {
		return m[0].Error()
	}
	var errs []string
	for _, e := range m {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, ", ")
}

func (c *Client) Close() error {
	var mErr MultiErr
	// First, close all open socket listeners to prevent additional connections.
	for _, m := range c.mnts {
		err := m.Close()
		if err != nil {
			mErr = append(mErr, err)
		}
	}
	// Next, close the dialer to prevent any additional refreshes.
	cErr := c.dialer.Close()
	if cErr != nil {
		mErr = append(mErr, cErr)
	}
	if c.waitOnClose == 0 {
		if len(mErr) > 0 {
			return mErr
		}
		return nil
	}
	timeout := time.After(c.waitOnClose)
	tick := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-tick:
			if atomic.LoadUint64(&c.connCount) > 0 {
				continue
			}
		case <-timeout:
		}
		break
	}
	open := atomic.LoadUint64(&c.connCount)
	if open > 0 {
		mErr = append(mErr, fmt.Errorf("%d connection(s) still open after waiting %v", open, c.waitOnClose))
	}
	if len(mErr) > 0 {
		return mErr
	}
	return nil
}

// serveSocketMount persistently listens to the socketMounts listener and proxies connections to a
// given AlloyDB instance.
func (c *Client) serveSocketMount(ctx context.Context, s *socketMount) error {
	for {
		cConn, err := s.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				c.logger.Errorf("[%s] Error accepting connection: %v", s.inst, err)
				// For transient errors, wait a small amount of time to see if it resolves itself
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}
		// handle the connection in a separate goroutine
		go func() {
			c.logger.Infof("[%s] accepted connection from %s\n", s.inst, cConn.RemoteAddr())

			// A client has established a connection to the local socket. Before
			// we initiate a connection to the AlloyDB backend, increment the
			// connection counter. If the total number of connections exceeds
			// the maximum, refuse to connect and close the client connection.
			count := atomic.AddUint64(&c.connCount, 1)
			defer atomic.AddUint64(&c.connCount, ^uint64(0))

			if c.maxConns > 0 && count > c.maxConns {
				c.logger.Infof("max connections (%v) exceeded, refusing new connection", c.maxConns)
				_ = cConn.Close()
				return
			}

			// give a max of 30 seconds to connect to the instance
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			sConn, err := c.dialer.Dial(ctx, s.inst)
			if err != nil {
				c.logger.Errorf("[%s] failed to connect to instance: %v\n", s.inst, err)
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

func newSocketMount(ctx context.Context, conf *Config, pc *portConfig, inst InstanceConnConfig) (*socketMount, error) {
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

	lc := net.ListenConfig{KeepAlive: 30 * time.Second}
	ln, err := lc.Listen(ctx, network, address)
	if err != nil {
		return nil, err
	}
	m := &socketMount{inst: inst.Name, listener: ln}
	return m, nil
}

func (s *socketMount) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *socketMount) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

// close stops the mount from listening for any more connections
func (s *socketMount) Close() error {
	return s.listener.Close()
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
				c.logger.Errorf(errDesc)
			} else {
				c.logger.Infof(errDesc)
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
