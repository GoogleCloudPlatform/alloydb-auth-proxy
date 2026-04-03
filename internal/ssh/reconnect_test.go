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

// Internal tests for reconnect behavior. This file uses package ssh (not
// ssh_test) so it can access unexported fields of Tunnel directly.
package ssh

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cryptossh "golang.org/x/crypto/ssh"

	ilog "github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/log"
)

var reconnectTestLogger = ilog.NewStdLogger(os.Stdout, os.Stdout)

func genKeyPair(t *testing.T) (ed25519.PrivateKey, cryptossh.Signer) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := cryptossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return priv, signer
}

func writeKeyFile(t *testing.T, priv ed25519.PrivateKey) string {
	t.Helper()
	keyBytes, err := cryptossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(keyBytes), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDialDuringReconnect verifies that Dial returns an error mentioning
// "reconnecting" (not "closed") when a reconnect goroutine is in flight.
func TestDialDuringReconnect(t *testing.T) {
	tun := &Tunnel{
		done:         make(chan struct{}),
		reconnecting: true,
	}
	_, err := tun.Dial(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reconnecting") {
		t.Errorf("error %q should mention reconnecting", err)
	}
}

// TestReconnectRetriesPastOldCap verifies that reconnect keeps retrying past
// the previously hardcoded limit of 10 attempts and eventually succeeds once
// the server becomes available.
func TestReconnectRetriesPastOldCap(t *testing.T) {
	clientPriv, clientSigner := genKeyPair(t)
	_, hostSigner := genKeyPair(t)

	const rejectFirst = 12 // beyond the old cap of 10
	var attempts atomic.Int32

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	sshCfg := &cryptossh.ServerConfig{
		PublicKeyCallback: func(_ cryptossh.ConnMetadata, key cryptossh.PublicKey) (*cryptossh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientSigner.PublicKey().Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	sshCfg.AddHostKey(hostSigner)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			n := attempts.Add(1)
			if n < rejectFirst {
				// Close immediately so ssh.Dial fails.
				conn.Close()
				continue
			}
			go func(c net.Conn) {
				sConn, chans, reqs, err := cryptossh.NewServerConn(c, sshCfg)
				if err != nil {
					return
				}
				defer sConn.Close()
				go cryptossh.DiscardRequests(reqs)
				for newCh := range chans {
					newCh.Reject(cryptossh.UnknownChannelType, "not needed")
				}
			}(conn)
		}
	}()

	keyFile := writeKeyFile(t, clientPriv)
	tun := &Tunnel{
		logger:                  reconnectTestLogger,
		keyPath:                 keyFile,
		user:                    "testuser",
		addr:                    listener.Addr().String(),
		knownHostsPath:          "none",
		done:                    make(chan struct{}),
		reconnecting:            true,
		initialReconnectBackoff: time.Millisecond,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		tun.reconnect()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("reconnect did not succeed within timeout")
	}

	tun.mu.Lock()
	c := tun.client
	tun.mu.Unlock()

	if c == nil {
		t.Fatal("expected client to be non-nil after successful reconnect")
	}
	c.Close()

	if got := int(attempts.Load()); got < rejectFirst {
		t.Errorf("expected at least %d dial attempts, got %d", rejectFirst, got)
	}
}
