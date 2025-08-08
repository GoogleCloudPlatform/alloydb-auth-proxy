// Copyright 2025 Google LLC
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
	"net"

	"cloud.google.com/go/alloydb/apiv1alpha/alloydbpb"
	"cloud.google.com/go/alloydbconn"
	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"github.com/jackc/pgx/v5/pgproto3"
	"google.golang.org/api/option"

	alloydbadmin "cloud.google.com/go/alloydb/apiv1alpha"
)

type controlPlaneProxy struct {
	client *alloydbadmin.AlloyDBAdminClient
	logger alloydb.Logger

	database string
}

func newControlPlaneProxy(ctx context.Context, l alloydb.Logger, opts ...option.ClientOption) (*controlPlaneProxy, error) {
	adminCl, err := alloydbadmin.NewAlloyDBAdminClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &controlPlaneProxy{client: adminCl, logger: l}, nil

}

func (p *controlPlaneProxy) proxyConn(ctx context.Context, instURI string, client net.Conn, _ ...alloydbconn.DialOption) error {
	// TODO: encapsulate this nonsense!
	// This is only necessary to get the database version to return to the client.
	// It's probably worth caching the response to reduce startup time.
	proj, reg, cluster, _, err := ParseInstanceURI(instURI)
	if err != nil {
		return err
	}
	clusterURI := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", proj, reg, cluster)
	resp, err := p.client.GetCluster(ctx, &alloydbpb.GetClusterRequest{Name: clusterURI})
	if err != nil {
		return err
	}

	b := pgproto3.NewBackend(client, client)
	dbVersion := resp.GetDatabaseVersion().String()

	if err := p.handleStartup(b, client, dbVersion); err != nil {
		p.logger.Errorf("Client startup failed: %v", err)
		return err
	}

	for {
		msg, err := b.Receive()
		if err != nil {
			p.logger.Errorf("Receiving message failed: %v\n", err)
			return nil
		}

		switch got := msg.(type) {
		case *pgproto3.Query:
			err := p.handleQuery(ctx, instURI, client, got)
			if err != nil {
				return err
			}
		case *pgproto3.Terminate:
			fmt.Println("terminating...")
			return nil
		default:
			fmt.Printf("received message other than Query from client: %#v\n", msg)
			return nil
		}
	}
}

func (p *controlPlaneProxy) handleQuery(ctx context.Context, instURI string, client net.Conn, q *pgproto3.Query) error {
	p.logger.Debugf("Client query: %v", q.String)

	// If there is no query provided, don't send it to ExecuteSQL. Just
	// respond immediately with an EmptyQueryResponse.
	if q.String == "" {
		var buf []byte
		buf = mustEncode((&pgproto3.EmptyQueryResponse{}).Encode(nil))
		_, err := client.Write(buf)
		if err != nil {
			p.logger.Errorf("Writing query response failed: %v\n", err)
			return err
		}
	}

	// TODO: figure out how to support filling in a partial query when a user
	// presses <TAB> in psql. Right now it's ignored.

	// TODO: the control plane should return the numbers of rows affected
	// for INSERTS, UPDATES, DELETES, etc.
	// See CommandComplete here for details:
	// https://www.postgresql.org/docs/current/protocol-message-formats.html

	req := &alloydbpb.ExecuteSqlRequest{
		Database:     p.database,
		Instance:     instURI,
		SqlStatement: q.String,
	}
	resp, err := p.client.ExecuteSql(ctx, req)
	if err != nil {
		p.logger.Errorf("ExecuteSQL error: %v\n", err)
		return err
	}
	p.logger.Debugf("ExecuteSQL response: %v", resp)

	if resp.GetMetadata().GetStatus() == alloydbpb.ExecuteSqlMetadata_ERROR {
		p.logger.Errorf("ExecuteSQL failed: %v", resp.GetMetadata().GetMessage())
		buf := mustEncode((&pgproto3.ErrorResponse{
			Message: resp.GetMetadata().GetMessage(),
		}).Encode(nil))
		_, err = client.Write(buf)
		if err != nil {
			p.logger.Errorf("Writing query response failed: %v\n", err)
			return err
		}
	}

	var buf []byte
	for _, res := range resp.GetSqlResults() {
		var desc []pgproto3.FieldDescription

		for i, col := range res.GetColumns() {
			desc = append(desc, pgproto3.FieldDescription{
				// The field name
				Name: []byte(col.GetName()),
				// If the field can be identified as a column of a specific
				// table, the object ID of the table; otherwise zero.
				TableOID: 0,
				// If the field can be identified as a column of a specific
				// table, the attribute number of the column; otherwise
				// zero.
				TableAttributeNumber: uint16(i),
				// The object ID of the field's data type.
				// TODO
				DataTypeOID: datatypeOID(col.GetType()),
				// -1 indicates a “varlena” type (one that has a length
				// word). See pg_type.typlen.
				// https://www.postgresql.org/docs/current/catalog-pg-type.html
				DataTypeSize: -1,
				// -1 means no modifier needed. See pg_attribute.atttypmod.
				// https://www.postgresql.org/docs/current/catalog-pg-attribute.html
				TypeModifier: -1,
				// 0 is text, 1 is binary
				Format: 0,
			})
		}
		buf = mustEncode((&pgproto3.RowDescription{Fields: desc}).Encode(nil))

		for _, row := range res.GetRows() {
			var rawValues [][]byte
			for _, rowValue := range row.GetValues() {
				if rowValue.GetNullValue() {
					rawValues = append(rawValues, []byte(""))
					continue
				}
				rawValues = append(rawValues, []byte(*rowValue.Value))
			}
			buf = mustEncode((&pgproto3.DataRow{Values: rawValues}).Encode(buf))
		}
	}
	buf = mustEncode((&pgproto3.CommandComplete{CommandTag: []byte(q.String)}).Encode(buf))
	buf = mustEncode((&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf))
	_, err = client.Write(buf)
	if err != nil {
		fmt.Printf("error writing query response: %v\n", err)
		return err
	}
	return nil
}

var dataTypeOIDMapping = map[string]uint32{
	"BOOL":        16,
	"BYTEA":       17,
	"CHAR":        18,
	"NAME":        19,
	"INT8":        20,
	"INT2":        21,
	"INT4":        23,
	"TEXT":        25,
	"OID":         26,
	"XID":         28,
	"CID":         29,
	"JSON":        114,
	"POINT":       600,
	"LSEG":        601,
	"PATH":        602,
	"BOX":         603,
	"POLYGON":     604,
	"LINE":        628,
	"CIDR":        650,
	"FLOAT4":      700,
	"FLOAT8":      701,
	"MONEY":       790,
	"MACADDR":     829,
	"INET":        869,
	"DATE":        1082,
	"TIME":        1083,
	"TIMESTAMP":   1114,
	"TIMESTAMPTZ": 1184,
	"INTERVAL":    1186,
	"NUMERIC":     1700,
	"UUID":        2950,
	"XML":         142,
	"JSONB":       3802,
}

// TODO: how to support custom datatypes? Anything else?
func datatypeOID(datatype string) uint32 {
	return dataTypeOIDMapping[datatype]
}

func (p *controlPlaneProxy) checkConn(ctx context.Context, instURI string, _ ...alloydbconn.DialOption) error {
	return nil
}

func (p *controlPlaneProxy) Close() error {
	return p.client.Close()
}

func (p *controlPlaneProxy) handleStartup(b *pgproto3.Backend, conn net.Conn, version string) error {
	m, err := b.ReceiveStartupMessage()
	if err != nil {
		return fmt.Errorf("error receiving startup message: %w", err)
	}

	p.logger.Debugf("Client message: (%T) %v\n", m, m)

	switch msg := m.(type) {
	case *pgproto3.StartupMessage:
		// TODO: the client may specify the user, but all ExecuteSQL RPCs use the
		// environment's IAM principal. We should probably require the user specify
		// their database user explicitly and then confirm the environment's IAM
		// principal matches before proceeding. Postgres has an ErrorResponse we
		// could send when there is a user mismatch. This ensures tools like
		// psql report the correct username.
		// NOTE: https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=$TOKEN
		// will report the current IAM principal's email.
		p.database = msg.Parameters["database"]

		buf := mustEncode((&pgproto3.AuthenticationOk{}).Encode(nil))
		// TODO: BackendKeyData?

		// Set as many ParameterStatus messages as we can:
		// https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-ASYNC
		buf = mustEncode((&pgproto3.ParameterStatus{Name: "server_encoding", Value: "utf-8"}).Encode(buf))
		buf = mustEncode((&pgproto3.ParameterStatus{Name: "server_version", Value: versionNumber(version)}).Encode(buf))
		buf = mustEncode((&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf))
		_, err = conn.Write(buf)
		if err != nil {
			return fmt.Errorf("error sending ready for query: %w", err)
		}
	case *pgproto3.SSLRequest:
		_, err = conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny SSL request: %w", err)
		}
		return p.handleStartup(b, conn, version) // only 1-layer of recursion here.
	default:
		return fmt.Errorf("Client sent unexpected message = (%T) %v", msg, msg)
	}

	return nil
}

func mustEncode(buf []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return buf
}

func versionNumber(version string) string {
	switch version {
	case "POSTGRES_14":
		return "14.x"
	case "POSTGRES_15":
		return "15.x"
	case "POSTGRES_16":
		return "16.x"
	case "POSTGRES_17":
		return "17.x"
	default:
		return "unknown"
	}
}
