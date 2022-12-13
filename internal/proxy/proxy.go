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
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

// InstanceConnConfig holds the configuration for an individual instance
// connection.
type InstanceConnConfig struct {
	// Name is the instance URI.
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

	// CredentialsJSON is a JSON representation of the service account key.
	CredentialsJSON string

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

	// FUSEDir enables a file system in user space at the provided path that
	// connects to the requested instance only when a client requests it.
	FUSEDir string

	// FUSETempDir sets the temporary directory where the FUSE mount will place
	// Unix domain sockets connected to Cloud SQL instances. The temp directory
	// is not accessed directly.
	FUSETempDir string

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

	// ImpersonateTarget is the service account to impersonate. The IAM
	// principal doing the impersonation must have the
	// roles/iam.serviceAccountTokenCreator role.
	ImpersonateTarget string
	// ImpersonateDelegates are the intermediate service accounts through which
	// the impersonation is achieved. Each delegate must have the
	// roles/iam.serviceAccountTokenCreator role.
	ImpersonateDelegates []string

	// StructuredLogs sets all output to use JSON in the LogEntry format.
	// See https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
	StructuredLogs bool
}

func (c *Config) credentialsOpt(l alloydb.Logger) (alloydbconn.Option, error) {
	// If service account impersonation is configured, set up an impersonated
	// credentials token source.
	if c.ImpersonateTarget != "" {
		var iopts []option.ClientOption
		switch {
		case c.Token != "":
			l.Infof("Impersonating service account with OAuth2 token")
			iopts = append(iopts, option.WithTokenSource(
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
			))
		case c.CredentialsFile != "":
			l.Infof("Impersonating service account with the credentials file at %q", c.CredentialsFile)
			iopts = append(iopts, option.WithCredentialsFile(c.CredentialsFile))
		case c.CredentialsJSON != "":
			l.Infof("Impersonating service account with JSON credentials environment variable")
			iopts = append(iopts, option.WithCredentialsJSON([]byte(c.CredentialsJSON)))
		case c.GcloudAuth:
			l.Infof("Impersonating service account with gcloud user credentials")
			ts, err := gcloud.TokenSource()
			if err != nil {
				return nil, err
			}
			iopts = append(iopts, option.WithTokenSource(ts))
		default:
			l.Infof("Impersonating service account with Application Default Credentials")
		}
		ts, err := impersonate.CredentialsTokenSource(
			context.Background(),
			impersonate.CredentialsConfig{
				TargetPrincipal: c.ImpersonateTarget,
				Delegates:       c.ImpersonateDelegates,
				Scopes:          []string{sqladmin.SqlserviceAdminScope},
			},
			iopts...,
		)
		if err != nil {
			return nil, err
		}
		return alloydbconn.WithTokenSource(ts), nil
	}
	// Otherwise, configure credentials as usual.
	switch {
	case c.Token != "":
		l.Infof("Authorizing with OAuth2 token")
		return alloydbconn.WithTokenSource(
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token}),
		), nil
	case c.CredentialsFile != "":
		l.Infof("Authorizing with the credentials file at %q", c.CredentialsFile)
		return alloydbconn.WithCredentialsFile(c.CredentialsFile), nil
	case c.CredentialsJSON != "":
		l.Infof("Authorizing with JSON credentials environment variable")
		return alloydbconn.WithCredentialsJSON([]byte(c.CredentialsJSON)), nil
	case c.GcloudAuth:
		l.Infof("Authorizing with gcloud user credentials")
		ts, err := gcloud.TokenSource()
		if err != nil {
			return nil, err
		}
		return alloydbconn.WithTokenSource(ts), nil
	default:
		l.Infof("Authorizing with Application Default Credentials")
		// Return no-op options to avoid having to handle nil in caller code
		return alloydbconn.WithOptions(), nil
	}
}

// DialerOptions builds appropriate list of options from the Config
// values for use by alloydbconn.NewClient()
func (c *Config) DialerOptions(l alloydb.Logger) ([]alloydbconn.Option, error) {
	opts := []alloydbconn.Option{
		alloydbconn.WithUserAgent(c.UserAgent),
	}
	co, err := c.credentialsOpt(l)
	if err != nil {
		return nil, err
	}
	opts = append(opts, co)

	if c.APIEndpointURL != "" {
		opts = append(opts, alloydbconn.WithAdminAPIEndpoint(c.APIEndpointURL))
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
	instURIRegex = regexp.MustCompile("projects/([^:]+(?::[^:]+)?)/locations/(.+)/clusters/(.+)/instances/(.+)")
	// unixRegex is the expected format for a Unix socket
	// e.g. project.region.cluster.instance
	unixRegex = regexp.MustCompile(`([^:]+(?:-[^:]+)?)\.(.+)\.(.+)\.(.+)`)
)

// parseInstanceURI validates the instance uri is in the proper format and
// returns the project, region, cluster, and instance name.
func parseInstanceURI(inst string) (string, string, string, string, error) {
	m := instURIRegex.FindSubmatch([]byte(inst))
	if m == nil {
		return "", "", "", "", fmt.Errorf("invalid instance name: %v", inst)
	}
	return string(m[1]), string(m[2]), string(m[3]), string(m[4]), nil
}

// UnixSocketDir returns a shorted instance connection name to prevent
// exceeding the Unix socket length, e.g., project.region.cluster.instance
func UnixSocketDir(dir, inst string) (string, error) {
	project, region, cluster, name, err := parseInstanceURI(inst)
	if err != nil {
		return "", err
	}
	// Colons are not allowed on Windows, but are present in legacy project
	// names (e.g., google.com:myproj). Replace any colon with an underscore to
	// support Windows. Underscores are not allowed in project names. So use an
	// underscore to have a Windows-friendly delimitor that can serve as a
	// marker to recover the legacy project name when necessary (e.g., FUSE).
	project = strings.ReplaceAll(project, ":", "_")
	shortName := strings.Join([]string{project, region, cluster, name}, ".")
	return filepath.Join(dir, shortName), nil
}

// toFullURI converts a shortened Unix socket name (e.g.,
// project.region.cluster.instance) into a full instance URI.
func toFullURI(short string) (string, error) {
	m := unixRegex.FindSubmatch([]byte(short))
	if m == nil {
		return "", fmt.Errorf("invalid short name: %v", short)
	}
	project := string(m[1])
	// Adjust short name for legacy projects. Google Cloud projects cannot have
	// underscores in them. When there's an underscore in the short name, it's a
	// marker for a colon. So replace the underscore with the original colon.
	project = strings.ReplaceAll(project, "_", ":")
	region := string(m[2])
	cluster := string(m[3])
	name := string(m[4])
	return fmt.Sprintf(
		"projects/%s/locations/%s/clusters/%s/instances/%s",
		project, region, cluster, name,
	), nil
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

	fuseMount
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

	c := &Client{
		logger:      l,
		dialer:      d,
		maxConns:    conf.MaxConnections,
		waitOnClose: conf.WaitOnClose,
	}

	if conf.FUSEDir != "" {
		return configureFUSE(c, conf)
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

	c.mnts = mnts

	return c, nil
}

// CheckConnections dials each registered instance and reports any errors that
// may have occurred.
func (c *Client) CheckConnections(ctx context.Context) error {
	var (
		wg    sync.WaitGroup
		errCh = make(chan error, len(c.mnts))
		mnts  = c.mnts
	)

	if c.fuseDir != "" {
		mnts = c.fuseMounts()
	}
	for _, m := range mnts {
		wg.Add(1)
		go func(inst string) {
			defer wg.Done()
			conn, err := c.dialer.Dial(ctx, inst)
			if err != nil {
				errCh <- err
				return
			}
			cErr := conn.Close()
			if cErr != nil {
				errCh <- fmt.Errorf("%v: %v", inst, cErr)
			}
		}(m.inst)
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

	if c.fuseDir != "" {
		return c.serveFuse(ctx, notify)
	}

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

// Close stops the dialer, closes any open FUSE mounts and any open listeners,
// and optionally waits for all connections to close before exiting.
func (c *Client) Close() error {
	mnts := c.mnts

	var mErr MultiErr

	if c.fuseDir != "" {
		if err := c.unmountFUSE(); err != nil {
			mErr = append(mErr, err)
		}
		mnts = c.fuseMounts()
	}

	// First, close all open socket listeners to prevent additional connections.
	for _, m := range mnts {
		err := m.Close()
		if err != nil {
			mErr = append(mErr, err)
		}
	}
	if c.fuseDir != "" {
		c.waitForFUSEMounts()
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
func (c *Client) serveSocketMount(_ context.Context, s *socketMount) error {
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
	// Change file permisions to allow access for user, group, and other.
	if network == "unix" {
		// Best effort. If this call fails, group and other won't have write
		// access.
		_ = os.Chmod(address, 0777)
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
