module github.com/GoogleCloudPlatform/alloydb-auth-proxy

go 1.16

require (
	cloud.google.com/go/alloydbconn v0.2.0
	contrib.go.opencensus.io/exporter/prometheus v0.4.1
	contrib.go.opencensus.io/exporter/stackdriver v0.13.13
	github.com/google/go-cmp v0.5.8
	github.com/lib/pq v1.10.5 // indirect
	github.com/spf13/cobra v1.5.0
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.21.0
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
	golang.org/x/sys v0.0.0-20220712014510-0a85c31ab51e
)
