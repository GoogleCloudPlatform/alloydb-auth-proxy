// Copyright 2026 Google LLC
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

package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
)

const (
	// defaultKeepaliveInterval is how often the client sends keepalive requests.
	defaultKeepaliveInterval = 30 * time.Second

	// keepaliveTimeout is the maximum time to wait for a keepalive response.
	// If no response arrives within this window the connection is treated as
	// dead and a reconnect is triggered.
	keepaliveTimeout = 10 * time.Second

	// defaultDialTimeout is how long to wait for an SSH connection to establish.
	defaultDialTimeout = 10 * time.Second

	// maxReconnectBackoff is the upper bound for exponential reconnect backoff.
	maxReconnectBackoff = 5 * time.Minute
)

// Tunnel represents an SSH tunnel through a bastion host.
type Tunnel struct {
	mu     sync.Mutex
	client *ssh.Client
	// agentConn is the connection to the SSH agent socket. It is nil when
	// a key file is used directly instead of the agent.
	agentConn net.Conn
	// reconnecting is true while a reconnect goroutine is running. It
	// prevents multiple concurrent reconnect attempts.
	reconnecting bool

	// initialReconnectBackoff is the starting backoff duration for reconnect
	// attempts. It defaults to 1 second when zero. Tests can set it to a
	// smaller value to avoid waiting through real backoff durations.
	initialReconnectBackoff time.Duration

	keyPath        string
	user           string
	addr           string
	knownHostsPath string

	// closeOnce ensures Close only runs once, preventing a panic from
	// double-closing the done channel.
	closeOnce sync.Once
	// done is closed when Close is called to stop the keepalive loop.
	done chan struct{}

	logger alloydb.Logger
}

// defaultKnownHostsPath returns the platform-appropriate default known_hosts
// file path (~/.ssh/known_hosts). It returns an empty string if the home
// directory cannot be determined.
func defaultKnownHostsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

// resolveHostKeyCallback determines the ssh.HostKeyCallback to use based on
// the provided knownHostsPath value:
//   - "none": explicitly disables host key verification.
//   - non-empty path: uses that file (returns an error if unreadable).
//   - empty string: uses ~/.ssh/known_hosts if it exists, otherwise
//     returns an error directing the user to use --ssh-known-hosts.
func resolveHostKeyCallback(l alloydb.Logger, knownHostsPath string) (ssh.HostKeyCallback, error) {
	if knownHostsPath == "none" {
		l.Errorf("WARNING: SSH host key verification is disabled")
		return ssh.InsecureIgnoreHostKey(), nil
	}

	if knownHostsPath == "" {
		knownHostsPath = defaultKnownHostsPath()
		if knownHostsPath == "" {
			return nil, fmt.Errorf(
				"unable to determine default known_hosts path; " +
					"specify --ssh-known-hosts explicitly or set to \"none\" to disable host key verification",
			)
		}
		if _, err := os.Stat(knownHostsPath); err != nil {
			return nil, fmt.Errorf(
				"default known_hosts file %q not found; "+
					"specify --ssh-known-hosts explicitly or set to \"none\" to disable host key verification",
				knownHostsPath,
			)
		}
	}

	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read known_hosts file %q: %v", knownHostsPath, err)
	}
	return cb, nil
}

// connectAgent connects to the SSH agent via the SSH_AUTH_SOCK environment
// variable. It returns the agent connection and an ssh.AuthMethod that
// delegates signing to the agent.
func connectAgent() (net.Conn, ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, fmt.Errorf("SSH_AUTH_SOCK is not set; start an SSH agent and load your key with ssh-add")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to SSH agent at %q: %v", sock, err)
	}
	agentClient := agent.NewClient(conn)
	return conn, ssh.PublicKeysCallback(agentClient.Signers), nil
}

// resolveAuthMethod determines the ssh.AuthMethod to use and, when an agent
// connection is opened, returns it so the caller can manage its lifetime.
//
// When keyPath is provided, the key file is read and parsed directly. If the
// key is passphrase-protected, the method falls back to the SSH agent. When
// keyPath is empty, the SSH agent is used directly.
func resolveAuthMethod(l alloydb.Logger, keyPath string) (ssh.AuthMethod, net.Conn, error) {
	if keyPath == "" {
		l.Infof("No --ssh-key provided; using SSH agent for authentication")
		conn, auth, err := connectAgent()
		if err != nil {
			return nil, nil, err
		}
		return auth, conn, nil
	}

	fi, err := os.Stat(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read SSH private key: %v", err)
	}
	// Unix file permissions are not meaningful on Windows, where
	// Perm() always returns 0666 or 0444.
	if runtime.GOOS != "windows" {
		if perm := fi.Mode().Perm(); perm&0077 != 0 {
			return nil, nil, fmt.Errorf(
				"SSH private key %q has permissions %04o; must be 0600 or stricter to prevent other users from reading it",
				keyPath, perm,
			)
		}
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read SSH private key: %v", err)
	}
	// Zero the raw key bytes after parsing to limit how long private key
	// material remains in memory.
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		var ppErr *ssh.PassphraseMissingError
		if errors.As(err, &ppErr) {
			l.Infof("SSH key %q is passphrase-protected; falling back to SSH agent", keyPath)
			conn, auth, agentErr := connectAgent()
			if agentErr != nil {
				return nil, nil, fmt.Errorf(
					"SSH key %q is passphrase-protected and SSH agent is unavailable: %v; "+
						"load the key into an agent with ssh-add or use an unencrypted key",
					keyPath, agentErr,
				)
			}
			return auth, conn, nil
		}
		return nil, nil, fmt.Errorf("unable to parse SSH private key: %v", err)
	}

	return ssh.PublicKeys(signer), nil, nil
}

// NewTunnel establishes an SSH connection to the bastion host using the
// provided private key (or SSH agent) and address. The address should be in
// the form "host:port".
//
// When keyPath is empty, authentication is performed via the SSH agent
// (SSH_AUTH_SOCK). When keyPath points to a passphrase-protected key, the
// method automatically falls back to the SSH agent.
//
// The knownHostsPath controls host key verification:
//   - "": auto-detect ~/.ssh/known_hosts; error if not found.
//   - "none": explicitly skip host key verification.
//   - any other value: use that file path for host key verification.
func NewTunnel(l alloydb.Logger, keyPath, user, addr, knownHostsPath string) (*Tunnel, error) {
	authMethod, agentConn, err := resolveAuthMethod(l, keyPath)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := resolveHostKeyCallback(l, knownHostsPath)
	if err != nil {
		if agentConn != nil {
			agentConn.Close()
		}
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         defaultDialTimeout,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		if agentConn != nil {
			agentConn.Close()
		}
		return nil, fmt.Errorf("unable to connect to bastion host %q: %v", addr, err)
	}

	t := &Tunnel{
		client:         client,
		logger:         l,
		keyPath:        keyPath,
		user:           user,
		addr:           addr,
		knownHostsPath: knownHostsPath,
		agentConn:      agentConn,
		done:           make(chan struct{}),
	}
	go t.keepalive()
	return t, nil
}

// keepalive sends periodic SSH keepalive requests to detect a dead connection
// early rather than waiting for the OS TCP timeout. If a keepalive fails, it
// spawns a reconnect goroutine (if one is not already running).
func (t *Tunnel) keepalive() {
	ticker := time.NewTicker(defaultKeepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			t.mu.Lock()
			c := t.client
			already := t.reconnecting
			t.mu.Unlock()

			if already {
				continue
			}

			// Run SendRequest in a goroutine so we can enforce a timeout.
			// On a black-hole network the request never returns an error; it
			// just blocks until the OS TCP timeout fires (potentially minutes).
			// The goroutine sends on a buffered channel so it never leaks even
			// if we move on due to a timeout or tunnel close.
			type result struct{ err error }
			ch := make(chan result, 1)
			go func() {
				_, _, err := c.SendRequest("keepalive@openssh.com", true, nil)
				ch <- result{err}
			}()

			var sendErr error
			select {
			case <-t.done:
				return
			case <-time.After(keepaliveTimeout):
				sendErr = errors.New("keepalive timed out")
			case r := <-ch:
				sendErr = r.err
			}

			if sendErr != nil {
				t.mu.Lock()
				t.reconnecting = true
				t.mu.Unlock()
				go t.reconnect()
			}
		}
	}
}

// backoffWait waits for the current backoff duration, then doubles it (up to
// maxReconnectBackoff). It returns false if the tunnel is closed during the
// wait, signaling the caller to abort.
func (t *Tunnel) backoffWait(backoff *time.Duration) bool {
	select {
	case <-t.done:
		return false
	case <-time.After(*backoff):
	}
	*backoff *= 2
	if *backoff > maxReconnectBackoff {
		*backoff = maxReconnectBackoff
	}
	return true
}

// reconnect attempts to re-establish the SSH connection by re-resolving
// authentication (which refreshes a potentially stale agent connection)
// and rebuilding the SSH client config from scratch.
//
// It retries indefinitely with exponential backoff until either a connection
// succeeds or the tunnel is closed. The caller must set t.reconnecting to true
// before calling reconnect; reconnect clears the flag on return.
func (t *Tunnel) reconnect() {
	defer func() {
		t.mu.Lock()
		t.reconnecting = false
		t.mu.Unlock()
	}()

	backoff := t.initialReconnectBackoff
	if backoff == 0 {
		backoff = time.Second
	}
	for attempt := 1; ; attempt++ {
		select {
		case <-t.done:
			return
		default:
		}

		t.logger.Infof("SSH tunnel reconnecting to %s (attempt %d)", t.addr, attempt)

		// Re-resolve auth to get a fresh agent connection if needed.
		authMethod, agentConn, err := resolveAuthMethod(t.logger, t.keyPath)
		if err != nil {
			t.logger.Errorf("SSH tunnel reconnect auth resolution failed: %v", err)
			if !t.backoffWait(&backoff) {
				return
			}
			continue
		}

		hostKeyCallback, err := resolveHostKeyCallback(t.logger, t.knownHostsPath)
		if err != nil {
			if agentConn != nil {
				agentConn.Close()
			}
			t.logger.Errorf("SSH tunnel reconnect host key resolution failed: %v", err)
			if !t.backoffWait(&backoff) {
				return
			}
			continue
		}

		config := &ssh.ClientConfig{
			User:            t.user,
			Auth:            []ssh.AuthMethod{authMethod},
			HostKeyCallback: hostKeyCallback,
			Timeout:         defaultDialTimeout,
		}

		client, err := ssh.Dial("tcp", t.addr, config)
		if err != nil {
			if agentConn != nil {
				agentConn.Close()
			}
			t.logger.Errorf("SSH tunnel reconnect to %s failed: %v", t.addr, err)
			if !t.backoffWait(&backoff) {
				return
			}
			continue
		}

		// Dial succeeded. Swap in the new client under the lock, but
		// first verify the tunnel hasn't been closed in the meantime.
		t.mu.Lock()
		select {
		case <-t.done:
			// Tunnel closed while we were dialing. Discard the
			// new client to avoid leaking it.
			t.mu.Unlock()
			client.Close()
			if agentConn != nil {
				agentConn.Close()
			}
			return
		default:
		}
		old := t.client
		t.client = client
		oldAgent := t.agentConn
		t.agentConn = agentConn
		t.mu.Unlock()

		if old != nil {
			old.Close()
		}
		if oldAgent != nil {
			oldAgent.Close()
		}
		t.logger.Infof("SSH tunnel reconnected to %s", t.addr)
		return
	}
}

// Dial creates a new connection through the SSH tunnel to the specified
// network address. It implements the signature required by
// alloydbconn.WithDialFunc.
//
// The context governs only the dial phase (establishing the SSH channel).
// Once the connection is returned, its lifetime is managed by the caller
// via Close, not by context cancellation.
//
// The SSH channel returned by the underlying client does not support
// SetDeadline. To work around this, Dial uses net.Pipe to create a
// deadline-capable connection and proxies data between the pipe and the
// SSH channel in background goroutines.
func (t *Tunnel) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	t.mu.Lock()
	c := t.client
	r := t.reconnecting
	t.mu.Unlock()

	if r {
		return nil, errors.New("SSH tunnel is reconnecting, try again shortly")
	}

	sshConn, err := c.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("SSH tunnel dial failed: %v", err)
	}

	caller, proxy := net.Pipe()
	// isClosedError reports whether err is a closed-connection or EOF error,
	// both of which are expected during normal teardown.
	isClosedError := func(err error) bool {
		return errors.Is(err, io.EOF) ||
			errors.Is(err, io.ErrClosedPipe) ||
			errors.Is(err, net.ErrClosed)
	}

	go func() {
		defer proxy.Close()
		_, err := io.Copy(proxy, sshConn)
		if err != nil && !isClosedError(err) {
			t.logger.Errorf("SSH tunnel copy (remote->local) for %s error: %v", addr, err)
		}
	}()
	go func() {
		defer sshConn.Close()
		_, err := io.Copy(sshConn, proxy)
		if err != nil && !isClosedError(err) {
			t.logger.Errorf("SSH tunnel copy (local->remote) for %s error: %v", addr, err)
		}
	}()

	return caller, nil
}

// Close closes the underlying SSH connection, the agent connection (if any),
// and stops the keepalive loop. It is safe to call Close more than once.
func (t *Tunnel) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.done)
		t.mu.Lock()
		defer t.mu.Unlock()
		err = t.client.Close()
		if t.agentConn != nil {
			t.agentConn.Close()
		}
	})
	return err
}
