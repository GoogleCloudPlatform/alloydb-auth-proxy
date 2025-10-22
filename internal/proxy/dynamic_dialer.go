package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"github.com/jackc/pgx/v5/pgproto3"
)

type Dialer struct {
	inner *alloydbconn.Dialer
}

func NewDynamicDialer(ctx context.Context, opts ...alloydbconn.Option) (alloydb.Dialer, error) {
	inner, err := alloydbconn.NewDialer(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &Dialer{inner: inner}, nil
}

func (d *Dialer) Dial(ctx context.Context, inst string, opts ...alloydbconn.DialOption) (net.Conn, error) {
	return &dynamicConn{
		dialer: d.inner,
		ctx:    ctx,
		inst:   inst,
		opts:   opts,
		done:   make(chan struct{}),
	}, nil
}

func (d *Dialer) Close() error {
	return d.inner.Close()
}

type dynamicConn struct {
	once   sync.Once
	dialer *alloydbconn.Dialer
	ctx    context.Context
	inst   string
	opts   []alloydbconn.DialOption
	done   chan struct{}

	net.Conn
}

func (d *dynamicConn) Close() error {
	if d.Conn != nil {
		return d.Conn.Close()
	}
	return nil
}

func (d *dynamicConn) Read(b []byte) (int, error) {
	<-d.done
	return d.Conn.Read(b)
}

func (d *dynamicConn) Write(b []byte) (int, error) {
	var (
		n        int
		outerErr error
	)
	d.once.Do(func() {
		backend := pgproto3.NewBackend(bytes.NewReader(b), io.Discard)
		msg, err := backend.ReceiveStartupMessage()
		if err != nil {
			outerErr = err
			return
		}
		startup, ok := msg.(*pgproto3.StartupMessage)
		if !ok {
			outerErr = fmt.Errorf("received invalid message: %T", msg)
			return
		}
		inst := startup.Parameters["alloydb"]
		delete(startup.Parameters, "alloydb")

		conn, err := d.dialer.Dial(d.ctx, inst, d.opts...)
		if err != nil {
			outerErr = err
			return
		}
		d.Conn = conn
		close(d.done)

		buf := &bytes.Buffer{}
		data, err := msg.Encode(buf.Bytes())
		if err != nil {
			outerErr = err
			return
		}
		if n, err = d.Conn.Write(data); err != nil {
			outerErr = err
			return
		}
	})
	if outerErr != nil {
		return 0, outerErr
	}
	// If n is non-zero, it means this call is the first call.
	if n != 0 {
		return n, nil
	}

	return d.Conn.Write(b)
}
