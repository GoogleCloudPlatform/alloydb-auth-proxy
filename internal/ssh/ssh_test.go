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

package ssh_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	ilog "github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/log"
	issh "github.com/GoogleCloudPlatform/alloydb-auth-proxy/internal/ssh"
)

var testLogger = ilog.NewStdLogger(os.Stdout, os.Stdout)

// generateKeyPair generates an ed25519 key pair and returns the private key
// and its corresponding SSH signer.
func generateKeyPair(t *testing.T) (ed25519.PrivateKey, cryptossh.Signer) {
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

// writePrivateKey writes an ed25519 private key to a temp file and returns the path.
func writePrivateKey(t *testing.T, priv ed25519.PrivateKey) string {
	t.Helper()
	keyBytes, err := cryptossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(keyBytes), 0600); err != nil {
		t.Fatal(err)
	}
	return keyFile
}

// writeKnownHosts writes a known_hosts file for the given address and host
// public key, returning the file path.
func writeKnownHosts(t *testing.T, addr string, hostPubKey cryptossh.PublicKey) string {
	t.Helper()
	host, port, _ := net.SplitHostPort(addr)
	var entry string
	if port == "22" {
		entry = fmt.Sprintf("%s %s\n", host, string(cryptossh.MarshalAuthorizedKey(hostPubKey)))
	} else {
		entry = fmt.Sprintf("[%s]:%s %s\n", host, port, string(cryptossh.MarshalAuthorizedKey(hostPubKey)))
	}
	knownHostsFile := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsFile, []byte(entry), 0600); err != nil {
		t.Fatal(err)
	}
	return knownHostsFile
}

// startEchoServer starts a TCP echo server and returns its address.
func startEchoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	return ln.Addr().String()
}

// verifyEcho writes msg through conn and verifies the same data is echoed back.
func verifyEcho(t *testing.T, conn net.Conn, msg string) {
	t.Helper()
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if got := string(buf); got != msg {
		t.Errorf("got %q, want %q", got, msg)
	}
}

// withoutAgent unsets SSH_AUTH_SOCK for the duration of the test. Using
// t.Setenv ensures the original value is restored on cleanup and prevents
// the test from being marked as parallel.
func withoutAgent(t *testing.T) {
	t.Helper()
	t.Setenv("SSH_AUTH_SOCK", "")
}

// startTestSSHServer starts a minimal SSH server on a random port for testing.
// It returns the server address and a cleanup function. The server accepts
// connections and supports direct-tcpip channel requests, forwarding them to
// the target address.
func startTestSSHServer(t *testing.T, hostSigner cryptossh.Signer, authorizedKey cryptossh.PublicKey) (string, func()) {
	t.Helper()

	config := &cryptossh.ServerConfig{
		PublicKeyCallback: func(conn cryptossh.ConnMetadata, key cryptossh.PublicKey) (*cryptossh.Permissions, error) {
			if bytes.Equal(key.Marshal(), authorizedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key")
		},
	}
	config.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleSSHConn(t, conn, config)
		}
	}()

	return listener.Addr().String(), func() { listener.Close() }
}

func handleSSHConn(_ *testing.T, conn net.Conn, config *cryptossh.ServerConfig) {
	sConn, chans, reqs, err := cryptossh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sConn.Close()

	go cryptossh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "direct-tcpip" {
			newCh.Reject(cryptossh.UnknownChannelType, "unsupported channel type")
			continue
		}
		ch, _, err := newCh.Accept()
		if err != nil {
			continue
		}

		// Parse the target address from the channel extra data.
		var payload struct {
			Host           string
			Port           uint32
			OriginatorIP   string
			OriginatorPort uint32
		}
		if err := cryptossh.Unmarshal(newCh.ExtraData(), &payload); err != nil {
			ch.Close()
			continue
		}

		target := net.JoinHostPort(payload.Host, fmt.Sprintf("%d", payload.Port))
		targetConn, err := net.Dial("tcp", target)
		if err != nil {
			ch.Close()
			continue
		}

		go func() {
			defer ch.Close()
			defer targetConn.Close()
			go io.Copy(ch, targetConn)
			io.Copy(targetConn, ch)
		}()
	}
}

func TestTunnelDialsThrough(t *testing.T) {
	clientPriv, clientSigner := generateKeyPair(t)
	_, hostSigner := generateKeyPair(t)

	sshAddr, cleanup := startTestSSHServer(t, hostSigner, clientSigner.PublicKey())
	defer cleanup()

	echoAddr := startEchoServer(t)
	keyFile := writePrivateKey(t, clientPriv)

	tunnel, err := issh.NewTunnel(testLogger, keyFile, "testuser", sshAddr, "none")
	if err != nil {
		t.Fatalf("NewTunnel: %v", err)
	}
	defer tunnel.Close()

	conn, err := tunnel.Dial(context.Background(), "tcp", echoAddr)
	if err != nil {
		t.Fatalf("Tunnel.Dial: %v", err)
	}
	defer conn.Close()

	verifyEcho(t, conn, "hello through the tunnel")
}

func TestTunnelDialsWithKnownHosts(t *testing.T) {
	clientPriv, clientSigner := generateKeyPair(t)
	_, hostSigner := generateKeyPair(t)

	sshAddr, cleanup := startTestSSHServer(t, hostSigner, clientSigner.PublicKey())
	defer cleanup()

	knownHostsFile := writeKnownHosts(t, sshAddr, hostSigner.PublicKey())
	keyFile := writePrivateKey(t, clientPriv)

	tunnel, err := issh.NewTunnel(testLogger, keyFile, "testuser", sshAddr, knownHostsFile)
	if err != nil {
		t.Fatalf("NewTunnel: %v", err)
	}
	defer tunnel.Close()

	conn, err := tunnel.Dial(context.Background(), "tcp", sshAddr)
	if err != nil {
		t.Fatalf("Tunnel.Dial: %v", err)
	}
	conn.Close()
}

func TestTunnelRejectsUnknownHostKey(t *testing.T) {
	clientPriv, clientSigner := generateKeyPair(t)
	_, hostSigner := generateKeyPair(t)

	sshAddr, cleanup := startTestSSHServer(t, hostSigner, clientSigner.PublicKey())
	defer cleanup()

	// Write a known_hosts file with a DIFFERENT key to simulate mismatch.
	_, differentSigner := generateKeyPair(t)
	knownHostsFile := writeKnownHosts(t, sshAddr, differentSigner.PublicKey())
	keyFile := writePrivateKey(t, clientPriv)

	_, err := issh.NewTunnel(testLogger, keyFile, "testuser", sshAddr, knownHostsFile)
	if err == nil {
		t.Fatal("expected error for host key mismatch, got nil")
	}
}

func TestNewTunnelInvalidKey(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "bad_key")
	if err := os.WriteFile(keyFile, []byte("not a key"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := issh.NewTunnel(testLogger, keyFile, "user", "127.0.0.1:22", "none")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestNewTunnelMissingKey(t *testing.T) {
	_, err := issh.NewTunnel(testLogger, "/nonexistent/key", "user", "127.0.0.1:22", "none")
	if err == nil {
		t.Fatal("expected error for missing key file, got nil")
	}
}

// startTestAgent starts an in-process SSH agent on a Unix socket and sets
// SSH_AUTH_SOCK for the duration of the test via t.Setenv (which
// automatically restores the original value on cleanup).
//
// The socket is placed in /tmp to avoid exceeding macOS's 104-byte Unix
// socket path limit.
func startTestAgent(t *testing.T, keys ...ed25519.PrivateKey) {
	t.Helper()
	keyring := agent.NewKeyring()
	for _, k := range keys {
		if err := keyring.Add(agent.AddedKey{PrivateKey: k}); err != nil {
			t.Fatal(err)
		}
	}

	dir, err := os.MkdirTemp("/tmp", "ssh-agent-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	sockPath := filepath.Join(dir, "a.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(keyring, conn)
		}
	}()

	t.Setenv("SSH_AUTH_SOCK", sockPath)
}

func TestTunnelDialsThroughAgent(t *testing.T) {
	clientPriv, clientSigner := generateKeyPair(t)
	_, hostSigner := generateKeyPair(t)

	sshAddr, sshCleanup := startTestSSHServer(t, hostSigner, clientSigner.PublicKey())
	defer sshCleanup()
	startTestAgent(t, clientPriv)

	echoAddr := startEchoServer(t)

	tunnel, err := issh.NewTunnel(testLogger, "", "testuser", sshAddr, "none")
	if err != nil {
		t.Fatalf("NewTunnel (agent): %v", err)
	}
	defer tunnel.Close()

	conn, err := tunnel.Dial(context.Background(), "tcp", echoAddr)
	if err != nil {
		t.Fatalf("Tunnel.Dial: %v", err)
	}
	defer conn.Close()

	verifyEcho(t, conn, "hello through agent tunnel")
}

// writeEncryptedKey uses ssh-keygen to create a passphrase-protected ed25519
// key and returns the path to the private key and the corresponding
// crypto signer. Tests that need ssh-keygen should call t.Skip if it's
// not available.
func writeEncryptedKey(t *testing.T) (keyFile string, priv ed25519.PrivateKey) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not found, skipping encrypted key test")
	}
	keyFile = filepath.Join(t.TempDir(), "id_ed25519")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyFile, "-N", "testpass", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, out)
	}
	// Read the public key to derive the private key identity. We need the
	// actual private key to load into the agent, so re-parse with passphrase.
	encData, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	rawKey, err := cryptossh.ParseRawPrivateKeyWithPassphrase(encData, []byte("testpass"))
	if err != nil {
		t.Fatal(err)
	}
	privKey, ok := rawKey.(*ed25519.PrivateKey)
	if !ok {
		t.Fatalf("expected *ed25519.PrivateKey, got %T", rawKey)
	}
	return keyFile, *privKey
}

func TestTunnelFallsBackToAgentForEncryptedKey(t *testing.T) {
	keyFile, clientPriv := writeEncryptedKey(t)

	clientSigner, err := cryptossh.NewSignerFromKey(clientPriv)
	if err != nil {
		t.Fatal(err)
	}

	_, hostSigner := generateKeyPair(t)

	sshAddr, sshCleanup := startTestSSHServer(t, hostSigner, clientSigner.PublicKey())
	defer sshCleanup()
	startTestAgent(t, clientPriv)

	tunnel, err := issh.NewTunnel(testLogger, keyFile, "testuser", sshAddr, "none")
	if err != nil {
		t.Fatalf("NewTunnel (encrypted key + agent fallback): %v", err)
	}
	defer tunnel.Close()

	conn, err := tunnel.Dial(context.Background(), "tcp", sshAddr)
	if err != nil {
		t.Fatalf("Tunnel.Dial: %v", err)
	}
	conn.Close()
}

func TestTunnelEncryptedKeyNoAgentReturnsError(t *testing.T) {
	keyFile, _ := writeEncryptedKey(t)
	withoutAgent(t)

	_, err := issh.NewTunnel(testLogger, keyFile, "user", "127.0.0.1:22", "none")
	if err == nil {
		t.Fatal("expected error for encrypted key with no agent, got nil")
	}
	if !strings.Contains(err.Error(), "passphrase-protected") {
		t.Fatalf("expected passphrase-related error, got: %v", err)
	}
}

func TestTunnelNoKeyNoAgentReturnsError(t *testing.T) {
	withoutAgent(t)

	_, err := issh.NewTunnel(testLogger, "", "user", "127.0.0.1:22", "none")
	if err == nil {
		t.Fatal("expected error for no key and no agent, got nil")
	}
	if !strings.Contains(err.Error(), "SSH_AUTH_SOCK") {
		t.Fatalf("expected SSH_AUTH_SOCK-related error, got: %v", err)
	}
}
