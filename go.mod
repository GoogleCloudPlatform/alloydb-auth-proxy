module github.com/GoogleCloudPlatform/alloydb-auth-proxy

go 1.16

require (
	cloud.google.com/go/alloydbconn v0.0.0-0.20220401153611-87e713b37755
	github.com/google/go-cmp v0.5.7
	github.com/lib/pq v1.10.5 // indirect
	github.com/spf13/cobra v1.2.1
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a
	golang.org/x/sys v0.0.0-20220406163625-3f8b81556e12 // indirect
	google.golang.org/api v0.74.0 // indirect
	google.golang.org/genproto v0.0.0-20220401170504-314d38edb7de // indirect
)

replace cloud.google.com/go/alloydbconn => ../alloydb-go-connector
