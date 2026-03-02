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

//revive:disable-next-line:var-naming
package log

import (
	"fmt"
	"io"
	llog "log"
	"log/slog"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/alloydb-auth-proxy/alloydb"
	"google.golang.org/grpc/grpclog"
)

// StdLogger is the standard logger that distinguishes between info and error
// logs.
type StdLogger struct {
	stdLog *llog.Logger
	errLog *llog.Logger
}

// NewStdLogger create a Logger that uses out and err for informational and
// error messages.
func NewStdLogger(out, err io.Writer) alloydb.Logger {
	return &StdLogger{
		stdLog: llog.New(out, "", llog.LstdFlags),
		errLog: llog.New(err, "", llog.LstdFlags),
	}
}

// Infof logs informational messages.
func (l *StdLogger) Infof(format string, v ...any) {
	l.stdLog.Printf(format, v...)
}

// Errorf logs error messages.
func (l *StdLogger) Errorf(format string, v ...any) {
	l.errLog.Printf(format, v...)
}

// Debugf logs debug messages.
func (l *StdLogger) Debugf(format string, v ...any) {
	l.stdLog.Printf(format, v...)
}

// StructuredLogger writes log messages in JSON.
type StructuredLogger struct {
	stdLog *slog.Logger
	errLog *slog.Logger
}

// Infof logs informational messages.
func (l *StructuredLogger) Infof(format string, v ...any) {
	l.stdLog.Info(fmt.Sprintf(format, v...))
}

// Errorf logs error messages.
func (l *StructuredLogger) Errorf(format string, v ...any) {
	l.errLog.Error(fmt.Sprintf(format, v...))
}

// Debugf logs debug messages.
func (l *StructuredLogger) Debugf(format string, v ...any) {
	l.stdLog.Debug(fmt.Sprintf(format, v...))
}

// NewStructuredLogger creates a Logger that logs messages using JSON.
func NewStructuredLogger(quiet bool) alloydb.Logger {
	var infoHandler, errorHandler slog.Handler
	if quiet {
		infoHandler = slog.DiscardHandler
	} else {
		infoHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
		})
	}
	errorHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelError,
		ReplaceAttr: replaceAttr,
	})

	l := &StructuredLogger{
		stdLog: slog.New(infoHandler),
		errLog: slog.New(errorHandler),
	}
	return l
}

// grpcLogger adapts an alloydb.Logger to the grpclog.LoggerV2 interface,
// routing all gRPC log output to Debugf so it stays hidden unless debug
// logging is enabled.
type grpcLogger struct {
	l alloydb.Logger
}

// SetGRPCLogger installs l as the gRPC library logger.
func SetGRPCLogger(l alloydb.Logger) {
	grpclog.SetLoggerV2(&grpcLogger{l: l})
}

func (g *grpcLogger) Info(args ...any)                    { g.l.Debugf(fmt.Sprint(args...)) }
func (g *grpcLogger) Infoln(args ...any)                  { g.l.Debugf(fmt.Sprintln(args...)) }
func (g *grpcLogger) Infof(format string, args ...any)    { g.l.Debugf(format, args...) }
func (g *grpcLogger) Warning(args ...any)                 { g.l.Debugf(fmt.Sprint(args...)) }
func (g *grpcLogger) Warningln(args ...any)               { g.l.Debugf(fmt.Sprintln(args...)) }
func (g *grpcLogger) Warningf(format string, args ...any) { g.l.Debugf(format, args...) }
func (g *grpcLogger) Error(args ...any)                   { g.l.Debugf(fmt.Sprint(args...)) }
func (g *grpcLogger) Errorln(args ...any)                 { g.l.Debugf(fmt.Sprintln(args...)) }
func (g *grpcLogger) Errorf(format string, args ...any)   { g.l.Debugf(format, args...) }
func (g *grpcLogger) Fatal(args ...any)                   { g.l.Debugf(fmt.Sprint(args...)); os.Exit(1) }
func (g *grpcLogger) Fatalln(args ...any)                 { g.l.Debugf(fmt.Sprintln(args...)); os.Exit(1) }
func (g *grpcLogger) Fatalf(format string, args ...any)   { g.l.Debugf(format, args...); os.Exit(1) }
func (g *grpcLogger) V(int) bool                          { return false }

// replaceAttr remaps default Go logging keys to adhere to LogEntry format
// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if groups != nil {
		return a
	}

	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
	case slog.MessageKey:
		a.Key = "message"
	case slog.SourceKey:
		a.Key = "sourceLocation"
	case slog.TimeKey:
		a.Key = "timestamp"
		if a.Value.Kind() == slog.KindTime {
			a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
		}
	}
	return a
}
