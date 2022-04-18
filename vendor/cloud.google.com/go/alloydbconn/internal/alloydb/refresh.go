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

package alloydb

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/alloydbapi"
	"cloud.google.com/go/alloydbconn/internal/trace"
	"golang.org/x/time/rate"
)

// fetchMetadata uses the AlloyDB Admin APIs get method to retreive the
// information about an AlloyDB instance that is used to create secure
// connections.
func fetchMetadata(ctx context.Context, cl *alloydbapi.Client, inst instanceURI) (ipAddr string, err error) {
	var end trace.EndSpanFunc
	ctx, end = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.FetchMetadata")
	defer func() { end(err) }()
	resp, err := cl.InstanceGet(ctx, inst.project, inst.region, inst.cluster, inst.name)
	if err != nil {
		return "", errtype.NewRefreshError("failed to get instance metadata", inst.String(), err)
	}
	return resp.IPAddress, nil
}

var errInvalidPEM = errors.New("certificate is not a valid PEM")

func parseCert(cert string) (*x509.Certificate, error) {
	b, _ := pem.Decode([]byte(cert))
	if b == nil {
		return nil, errInvalidPEM
	}
	return x509.ParseCertificate(b.Bytes)
}

// fetchEphemeralCert uses the AlloyDB Admin API's generateClientCertificate
// method to create a signed TLS certificate that authorized to connect via the
// AlloyDB instance's serverside proxy. The cert if valid for approximately one
// hour.
func fetchEphemeralCert(
	ctx context.Context,
	cl *alloydbapi.Client,
	inst instanceURI,
	key *rsa.PrivateKey,
) (cc certChain, err error) {
	var end trace.EndSpanFunc
	ctx, end = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.FetchEphemeralCert")
	defer func() { end(err) }()

	subj := pkix.Name{
		CommonName:         "alloydb-proxy",
		Country:            []string{"US"},
		Province:           []string{"CA"},
		Locality:           []string{"Sunnyvale"},
		Organization:       []string{"Google LLC"},
		OrganizationalUnit: []string{"Cloud"},
	}
	tmpl := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, key)
	if err != nil {
		return certChain{}, err
	}
	buf := &bytes.Buffer{}
	pem.Encode(buf, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	resp, err := cl.GenerateClientCert(ctx, inst.project, inst.region, inst.cluster, buf.Bytes())
	if err != nil {
		return certChain{}, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			err,
		)
	}
	// There should always be two certs in the chain. If this fails, the API has
	// broken its contract with the client.
	if len(resp.PemCertificateChain) != 2 {
		return certChain{}, errtype.NewRefreshError(
			"missing instance and root certificates",
			inst.String(),
			err,
		)
	}
	rc, err := parseCert(resp.PemCertificateChain[1]) // root cert
	if err != nil {
		return certChain{}, errtype.NewRefreshError(
			"failed to parse root cert",
			inst.String(),
			err,
		)
	}
	ic, err := parseCert(resp.PemCertificateChain[0]) // intermediate cert
	if err != nil {
		return certChain{}, errtype.NewRefreshError(
			"failed to parse intermediate cert",
			inst.String(),
			err,
		)
	}
	c, err := parseCert(resp.PemCertificate) // client cert
	if err != nil {
		return certChain{}, errtype.NewRefreshError(
			"failed to parse client cert",
			inst.String(),
			err,
		)
	}

	return certChain{
		root:         rc,
		intermediate: ic,
		client:       c,
	}, nil
}

// createTLSConfig returns a *tls.Config for connecting securely to the AlloyDB
// instance.
func createTLSConfig(inst instanceURI, cc certChain, k *rsa.PrivateKey) *tls.Config {
	certs := x509.NewCertPool()
	certs.AddCert(cc.root)

	return &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) != 2 {
				msg := fmt.Sprintf(
					"unexpected number of certificates to verify, want = 2, got = %v",
					len(rawCerts),
				)
				return errtype.NewDialError(msg, inst.String(), nil)
			}
			chain, err := x509.ParseCertificate(rawCerts[1])
			if err != nil {
				return errtype.NewDialError("failed to parse X.509 certificate", inst.String(), err)
			}

			opts := x509.VerifyOptions{Roots: certs}
			if _, err = chain.Verify(opts); err != nil {
				return errtype.NewDialError("failed to verify certificate", inst.String(), err)
			}

			// TODO: restore the server name check when it becomes the instance
			// UID
			// server, err := x509.ParseCertificate(rawCerts[0])
			// if err != nil {
			//  return errtype.NewDialError("failed to parse X.509 certificate", inst.String(), err)
			// }
			// serverName := "FIXME:INSTANCE_UID"
			// if server.Subject.CommonName != serverName {
			//  return errtype.NewDialError(
			//      fmt.Sprintf("certificate had CN %q, expected %q",
			//          chain.Subject.CommonName, serverName),
			//      inst.String(),
			//      nil,
			//  )
			// }
			return nil
		},
		Certificates: []tls.Certificate{tls.Certificate{
			Certificate: [][]byte{cc.client.Raw, cc.intermediate.Raw},
			PrivateKey:  k,
			Leaf:        cc.client,
		}},
		RootCAs:    certs,
		MinVersion: tls.VersionTLS13,
	}
}

// newRefresher creates a Refresher.
func newRefresher(
	client *alloydbapi.Client,
	timeout time.Duration,
	interval time.Duration,
	burst int,
	dialerID string,
) refresher {
	return refresher{
		client:        client,
		timeout:       timeout,
		clientLimiter: rate.NewLimiter(rate.Every(interval), burst),
		dialerID:      dialerID,
	}
}

// refresher manages the AlloyDB Admin API access to instance metadata and to
// ephemeral certificates.
type refresher struct {
	// client provides access to the AlloyDB Admin API
	client *alloydbapi.Client

	// timeout is the maximum amount of time a refresh operation should be allowed to take.
	timeout time.Duration

	// dialerID is the unique ID of the associated dialer.
	dialerID string

	// clientLimiter limits the number of refreshes.
	clientLimiter *rate.Limiter
}

type refreshResult struct {
	instanceIPAddr string
	conf           *tls.Config
	expiry         time.Time
}

type certChain struct {
	root         *x509.Certificate
	intermediate *x509.Certificate
	client       *x509.Certificate
}

func (r refresher) performRefresh(ctx context.Context, cn instanceURI, k *rsa.PrivateKey) (res refreshResult, err error) {
	var refreshEnd trace.EndSpanFunc
	ctx, refreshEnd = trace.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.RefreshConnection",
		trace.AddInstanceName(cn.String()),
	)
	defer func() {
		go trace.RecordRefreshResult(context.Background(), cn.String(), r.dialerID, err)
		refreshEnd(err)
	}()

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if ctx.Err() == context.Canceled {
		return refreshResult{}, ctx.Err()
	}

	// avoid refreshing too often to try not to tax the AlloyDB Admin API quotas
	err = r.clientLimiter.Wait(ctx)
	if err != nil {
		return refreshResult{}, errtype.NewDialError(
			"refresh was throttled until context expired",
			cn.String(),
			nil,
		)
	}

	type mdRes struct {
		ipAddr string
		err    error
	}
	mdCh := make(chan mdRes, 1)
	go func() {
		defer close(mdCh)
		addr, err := fetchMetadata(ctx, r.client, cn)
		mdCh <- mdRes{ipAddr: addr, err: err}
	}()

	type certRes struct {
		cc  certChain
		err error
	}
	certCh := make(chan certRes, 1)
	go func() {
		defer close(certCh)
		cc, err := fetchEphemeralCert(ctx, r.client, cn, k)
		certCh <- certRes{cc: cc, err: err}
	}()

	var ipAddr string
	select {
	case r := <-mdCh:
		if r.err != nil {
			return refreshResult{}, fmt.Errorf("failed to get instance IP address: %w", r.err)
		}
		ipAddr = r.ipAddr
	case <-ctx.Done():
		return refreshResult{}, fmt.Errorf("refresh failed: %w", ctx.Err())
	}

	var cc certChain
	select {
	case r := <-certCh:
		if r.err != nil {
			return refreshResult{}, fmt.Errorf("fetch ephemeral cert failed: %w", r.err)
		}
		cc = r.cc
	case <-ctx.Done():
		return refreshResult{}, fmt.Errorf("refresh failed: %w", ctx.Err())
	}

	c := createTLSConfig(cn, cc, k)
	var expiry time.Time
	// This should never not be the case, but we check to avoid a potential nil-pointer
	if len(c.Certificates) > 0 {
		expiry = c.Certificates[0].Leaf.NotAfter
	}
	return refreshResult{instanceIPAddr: ipAddr, conf: c, expiry: expiry}, nil
}
