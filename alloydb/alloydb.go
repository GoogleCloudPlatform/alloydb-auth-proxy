// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydb

import (
	"context"
	"io"
	"net"

	"cloud.google.com/go/alloydbconn"
)

// Dialer dials an AlloyDB instance.
type Dialer interface {
	// Dial returns a connection to the specified instance.
	Dial(ctx context.Context, inst string, opts ...alloydbconn.DialOption) (net.Conn, error)

	io.Closer
}

// Logger is the interface used throughout the project for logging.
type Logger interface {
	// Debugf is for reporting additional information about internal operations.
	Debugf(format string, args ...any)
	// Infof is for reporting informational messages.
	Infof(format string, args ...any)
	// Errorf is for reporting errors.
	Errorf(format string, args ...any)
}
