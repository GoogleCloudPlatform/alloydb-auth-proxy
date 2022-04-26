// Copyright 2020 Google LLC
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

package alloydbconn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/alloydb"
	"cloud.google.com/go/alloydbconn/internal/alloydbapi"
	"cloud.google.com/go/alloydbconn/internal/trace"
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
	"google.golang.org/api/option"
)

const (
	// defaultTCPKeepAlive is the default keep alive value used on connections
	// to a AlloyDB instance
	defaultTCPKeepAlive = 30 * time.Second
	// serverProxyPort is the port the server-side proxy receives connections on.
	serverProxyPort = "5433"
)

var (
	// versionString indicates the version of this library.
	//go:embed version.txt
	versionString string
	userAgent     = "alloydb-go-connector/" + strings.TrimSpace(versionString)

	// defaultKey is the default RSA public/private keypair used by the clients.
	defaultKey    *rsa.PrivateKey
	defaultKeyErr error
	keyOnce       sync.Once
)

func getDefaultKeys() (*rsa.PrivateKey, error) {
	keyOnce.Do(func() {
		defaultKey, defaultKeyErr = rsa.GenerateKey(rand.Reader, 2048)
	})
	return defaultKey, defaultKeyErr
}

// A Dialer is used to create connections to AlloyDB instance.
//
// Use NewDialer to initialize a Dialer.
type Dialer struct {
	lock sync.RWMutex
	// instances map instance URIs to *alloydb.Instance types
	instances      map[string]*alloydb.Instance
	key            *rsa.PrivateKey
	refreshTimeout time.Duration

	client *alloydbapi.Client

	// defaultDialCfg holds the constructor level DialOptions, so that it can
	// be copied and mutated by the Dial function.
	defaultDialCfg dialCfg

	// dialerID uniquely identifies a Dialer. Used for monitoring purposes,
	// *only* when a client has configured OpenCensus exporters.
	dialerID string

	// dialFunc is the function used to connect to the address on the named
	// network. By default it is golang.org/x/net/proxy#Dial.
	dialFunc func(cxt context.Context, network, addr string) (net.Conn, error)
}

// NewDialer creates a new Dialer.
//
// Initial calls to NewDialer make take longer than normal because generation of an
// RSA keypair is performed. Calls with a WithRSAKeyPair DialOption or after a default
// RSA keypair is generated will be faster.
func NewDialer(ctx context.Context, opts ...Option) (*Dialer, error) {
	cfg := &dialerConfig{
		refreshTimeout: 30 * time.Second,
		dialFunc:       proxy.Dial,
		useragents:     []string{userAgent},
	}
	for _, opt := range opts {
		opt(cfg)
		if cfg.err != nil {
			return nil, cfg.err
		}
	}
	// Add this to the end to make sure it's not overridden
	cfg.adminOpts = append(cfg.adminOpts, option.WithUserAgent(strings.Join(cfg.useragents, " ")))

	if cfg.rsaKey == nil {
		key, err := getDefaultKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA keys: %v", err)
		}
		cfg.rsaKey = key
	}

	client, err := alloydbapi.NewClient(ctx, cfg.adminOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create AlloyDB Admin API client: %v", err)
	}

	dialCfg := dialCfg{
		tcpKeepAlive: defaultTCPKeepAlive,
	}
	for _, opt := range cfg.dialOpts {
		opt(&dialCfg)
	}

	if err := trace.InitMetrics(); err != nil {
		return nil, err
	}
	d := &Dialer{
		instances:      make(map[string]*alloydb.Instance),
		key:            cfg.rsaKey,
		refreshTimeout: cfg.refreshTimeout,
		client:         client,
		defaultDialCfg: dialCfg,
		dialerID:       uuid.New().String(),
		dialFunc:       cfg.dialFunc,
	}
	return d, nil
}

// Dial returns a net.Conn connected to the specified AlloyDB instance. The
// instance argument must be the instance's URI, which is in the format
// projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>
func (d *Dialer) Dial(ctx context.Context, instance string, opts ...DialOption) (conn net.Conn, err error) {
	startTime := time.Now()
	var endDial trace.EndSpanFunc
	ctx, endDial = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn.Dial",
		trace.AddInstanceName(instance),
		trace.AddDialerID(d.dialerID),
	)
	defer func() {
		go trace.RecordDialError(context.Background(), instance, d.dialerID, err)
		endDial(err)
	}()
	cfg := d.defaultDialCfg
	for _, opt := range opts {
		opt(&cfg)
	}

	var endInfo trace.EndSpanFunc
	ctx, endInfo = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.InstanceInfo")
	i, err := d.instance(instance)
	if err != nil {
		endInfo(err)
		return nil, err
	}
	addr, tlsCfg, err := i.ConnectInfo(ctx)
	if err != nil {
		endInfo(err)
		return nil, err
	}
	endInfo(err)

	var connectEnd trace.EndSpanFunc
	ctx, connectEnd = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.Connect")
	defer func() { connectEnd(err) }()
	addr = net.JoinHostPort(addr, serverProxyPort)
	conn, err = d.dialFunc(ctx, "tcp", addr)
	if err != nil {
		// refresh the instance info in case it caused the connection failure
		i.ForceRefresh()
		return nil, errtype.NewDialError("failed to dial", i.String(), err)
	}
	if c, ok := conn.(*net.TCPConn); ok {
		if err := c.SetKeepAlive(true); err != nil {
			return nil, errtype.NewDialError("failed to set keep-alive", i.String(), err)
		}
		if err := c.SetKeepAlivePeriod(cfg.tcpKeepAlive); err != nil {
			return nil, errtype.NewDialError("failed to set keep-alive period", i.String(), err)
		}
	}
	tlsConn := tls.Client(conn, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		// refresh the instance info in case it caused the handshake failure
		i.ForceRefresh()
		_ = tlsConn.Close() // best effort close attempt
		return nil, errtype.NewDialError("handshake failed", i.String(), err)
	}
	latency := time.Since(startTime).Milliseconds()
	go func() {
		n := atomic.AddUint64(&i.OpenConns, 1)
		trace.RecordOpenConnections(ctx, int64(n), d.dialerID, i.String())
		trace.RecordDialLatency(ctx, instance, d.dialerID, latency)
	}()

	return newInstrumentedConn(tlsConn, func() {
		n := atomic.AddUint64(&i.OpenConns, ^uint64(0))
		trace.RecordOpenConnections(context.Background(), int64(n), d.dialerID, i.String())
	}), nil
}

// newInstrumentedConn initializes an instrumentedConn that on closing will
// decrement the number of open connects and record the result.
func newInstrumentedConn(conn net.Conn, closeFunc func()) *instrumentedConn {
	return &instrumentedConn{
		Conn:      conn,
		closeFunc: closeFunc,
	}
}

// instrumentedConn wraps a net.Conn and invokes closeFunc when the connection
// is closed.
type instrumentedConn struct {
	net.Conn
	closeFunc func()
}

// Close delegates to the underylying net.Conn interface and reports the close
// to the provided closeFunc only when Close returns no error.
func (i *instrumentedConn) Close() error {
	err := i.Conn.Close()
	if err != nil {
		return err
	}
	go i.closeFunc()
	return nil
}

// Close closes the Dialer; it prevents the Dialer from refreshing the information
// needed to connect. Additional dial operations may succeed until the information
// expires.
func (d *Dialer) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, i := range d.instances {
		i.Close()
	}
	return nil
}

func (d *Dialer) instance(instanceURI string) (*alloydb.Instance, error) {
	// Check instance cache
	d.lock.RLock()
	i, ok := d.instances[instanceURI]
	d.lock.RUnlock()
	if !ok {
		d.lock.Lock()
		// Recheck to ensure instance wasn't created between locks
		i, ok = d.instances[instanceURI]
		if !ok {
			// Create a new instance
			var err error
			i, err = alloydb.NewInstance(instanceURI, d.client, d.key, d.refreshTimeout, d.dialerID)
			if err != nil {
				d.lock.Unlock()
				return nil, err
			}
			d.instances[instanceURI] = i
		}
		d.lock.Unlock()
	}
	return i, nil
}
