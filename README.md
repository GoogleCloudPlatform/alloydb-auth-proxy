# AlloyDB Auth Proxy

[![CI][ci-badge]][ci-build]
[![Go Reference][pkg-badge]][pkg-docs]

[ci-badge]: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/actions/workflows/tests.yaml/badge.svg?event=push
[ci-build]: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/actions/workflows/tests.yaml?query=event%3Apush+branch%3Amain
[pkg-badge]: https://pkg.go.dev/badge/github.com/GoogleCloudPlatform/alloydb-auth-proxy.svg
[pkg-docs]: https://pkg.go.dev/github.com/GoogleCloudPlatform/alloydb-auth-proxy

The AlloyDB Auth Proxy is a binary that provides IAM-based authorization and
encryption when connecting to an AlloyDB instance.

See the [Connecting Overview][connection-overview] page for more information on
connecting to an AlloyDB instance, or the [About the proxy][about-proxy] page
for details on how the AlloyDB Auth Proxy works.

If you're using Go, or Python, consider using the corresponding AlloyDB Connector
which does everything the Proxy does, but in a native process:

- [AlloyDB Go connector][go connector]
- [AlloyDB Python connector][python connector]

Note: The Proxy *cannot* provide a network path to an AlloyDB instance if one is
not already present (e.g., the proxy cannot access a VPC if it does not already
have access to it).

[go connector]: https://github.com/GoogleCloudPlatform/alloydb-go-connector
[python connector]: https://github.com/GoogleCloudPlatform/alloydb-python-connector

## Installation

Check for the latest version on the [releases page][releases] and use the
following instructions for your OS and CPU architecture.

<details>
<summary>Linux amd64</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.linux.amd64" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux 386</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.linux.386" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux arm64</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.linux.arm64" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux arm</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.linux.arm" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Intel)</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.darwin.amd64" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Apple Silicon)</summary>

``` sh
# see Releases for other versions
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2"

wget "$URL/alloydb-auth-proxy.darwin.arm64" -O alloydb-auth-proxy

chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Windows x64</summary>

``` sh
# see Releases for other versions
wget https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2/alloydb-auth-proxy-x64.exe -O alloydb-auth-proxy.exe
```
</details>

<details>
<summary>Windows x86</summary>

``` sh
# see Releases for other versions
wget https://storage.googleapis.com/alloydb-auth-proxy/v1.2.2/alloydb-auth-proxy-x86.exe -O alloydb-auth-proxy.exe
```
</details>


### Container Images

There are containerized versions of the proxy available from the following
Google Cloud Container Registry repositories:

* `gcr.io/alloydb-connectors/alloydb-auth-proxy`
* `us.gcr.io/alloydb-connectors/alloydb-auth-proxy`
* `eu.gcr.io/alloydb-connectors/alloydb-auth-proxy`
* `asia.gcr.io/alloydb-connectors/alloydb-auth-proxy`

Each image is tagged with the associated proxy version. The following tags are
currently supported:

* `$VERSION` - default image (recommended)
* `$VERSION-alpine` - uses [`alpine:3`](https://hub.docker.com/_/alpine) as a
  base image
* `$VERSION-buster` - uses [`debian:buster`](https://hub.docker.com/_/debian) as
* `$VERSION-bullseye` - uses [`debian:bullseye`](https://hub.docker.com/_/debian) as
  a base image

We recommend using the latest version of the proxy and updating the version
regularly. However, we also recommend pinning to a specific tag and avoid the
latest tag. Note: the tagged version is only that of the proxy. Changes in base
images may break specific setups, even on non-major version increments. As such,
it's a best practice to test changes before deployment, and use automated
rollbacks to revert potential failures.

### Install from Source

To install from source, ensure you have the latest version of [Go
installed](https://go.dev/doc/install).

Then, simply run:

```
go install github.com/GoogleCloudPlatform/alloydb-auth-proxy@latest
```

The `alloydb-auth-proxy` will be placed in `$GOPATH/bin` or `$HOME/go/bin`.

## Credentials

The AlloyDB Auth Proxy uses a Cloud IAM account to authorize connections against
an AlloyDB instance. The proxy supports the following options:

1. A `-credential-file` flag for a service account key file.
2. A `-token` flag for a OAuth2 Bearer token
3. [Application Default Credentials (ADC)][adc] if neither of the above have
   been set.

Note: Any account connecting to AlloyDB instance will need one of the following
IAM roles:

- Cloud AlloyDB Client (`roles/alloydb.client`) (preferred)
- Cloud AlloyDB Admin (`roles/alloydb.admin`)

See [Roles and Permissions in AlloyDB][roles-and-permissions] for details.

When the proxy authenticates under the Compute Engine VM's default service
account, the VM must have the `cloud-platform` API scope (i.e.,
"https://www.googleapis.com/auth/cloud-platform") and the associated project
must have the AlloyDB API enabled. The default service account must also
have at least writer or editor privileges to any projects of target AlloyDB
instances.

## Usage

All the following invocations assume valid credentials are present in the
environment. The following examples all reference an `INSTANCE_URI`,
which takes the form:

```
projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>
```

To find the `INSTANCE_URI`, take the name from:

```
# To find your instance in among all instances
gcloud alpha alloydb instances list

# or to describe your particular instance
gcloud alpha alloydb instances describe \
    --cluster <CLUSTER_NAME> \
    --region <REGION> \
    <INSTANCE_NAME>
```

### Example invocations

Note: the following invocations assume you have downloaded the
`alloydb-auth-proxy` into the same directory. Consider moving the proxy into a
well-known location to have it available on your `PATH`.

``` bash
# Starts the proxy listening on 127.0.0.1:5432
./alloydb-auth-proxy <INSTANCE_URI>
```

To connect to multiple instances, use:

``` bash
# For instance 1, the proxy listens on 127.0.0.1:5432
# For instance 2, the proxy listens on 127.0.0.1:5433
./alloydb-auth-proxy <INSTANCE_URI_1> <INSTANCE_URI_2>
```

To override the default address the proxy listens on, use the `--address` flag:

``` bash
# Starts the proxy listening on 0.0.0.0:5432
./alloydb-auth-proxy --address 0.0.0.0 <INSTANCE_URI>
```

To override the default port, use the `--port` flag:

``` bash
# Starts the proxy listening on 127.0.0.1:6000
./alloydb-auth-proxy --port 6000 <INSTANCE_URI>
```

In addition, both `address` and `port` may be overrided on a per-instance level
with a query-string style syntax:

``` bash
./alloydb-auth-proxy \
    '<INSTANCE_URI_1>?address=0.0.0.0&port=6000' \
    '<INSTANCE_URI_2>?address=127.0.0.1&port=7000'
```

Note: when using the query-string syntax, the instance URI and query parameters
must be wrapped in quotes.

## Running behind a Socks5 proxy

The AlloyDB Auth Proxy includes support for sending requests through a SOCKS5
proxy. If a SOCKS5 proxy is running on `localhost:8000`, the command to start
the AlloyDB Auth Proxy would look like:

```
ALL_PROXY=socks5://localhost:8000 \
HTTPS_PROXY=socks5://localhost:8000 \
    ./alloydb-auth-proxy <INSTANCE_URI>
```

The `ALL_PROXY` environment variable specifies the proxy for all TCP traffic to
and from a AlloyDB instance. The `ALL_PROXY` environment variable supports
`socks5` and `socks5h` protocols. To route DNS lookups through a proxy, use the
`socks5h` protocol.

The `HTTPS_PROXY` (or `HTTP_PROXY`) specifies the proxy for all HTTP(S) traffic
to the AlloyDB Admin API. Specifying `HTTPS_PROXY` or `HTTP_PROXY` is only necessary
when you want to proxy this traffic. Otherwise, it is optional. See
[`http.ProxyFromEnvironment`](https://pkg.go.dev/net/http@go1.17.3#ProxyFromEnvironment)
for possible values.

## Support for Metrics and Tracing

The Proxy supports [Cloud Monitoring][], [Cloud Trace][], and [Prometheus][].

Supported metrics include:

- `alloydbconn/dial_latency`: The distribution of dialer latencies (ms)
- `alloydbconn/open_connections`: The current number of open AlloyDB
  connections
- `alloydbconn/dial_failure_count`: The number of failed dial attempts
- `alloydbconn/refresh_success_count`: The number of successful certificate
  refresh operations
- `alloydbconn/refresh_failure_count`: The number of failed refresh
  operations.

Supported traces include:

- `cloud.google.com/go/alloydbconn.Dial`: The dial operation including
  refreshing an ephemeral certificate and connecting to the instance
- `cloud.google.com/go/alloydbconn/internal.InstanceInfo`: The call to retrieve
  instance metadata (e.g., IP address, etc)
- `cloud.google.com/go/alloydbconn/internal.Connect`: The connection attempt
  using the ephemeral certificate
- AlloyDB API client operations

To enable Cloud Monitoring and Cloud Trace, use the `--telemetry-project` flag
with the project where you want to view metrics and traces. To configure the
metrics prefix used by Cloud Monitoring, use the `--telemetry-prefix` flag. When
enabling telementry, both Cloud Monitoring and Cloud Trace are enabled. To
disable Cloud Monitoring, use `--disable-metrics`. To disable Cloud Trace, use
`--disable-traces`.

To enable Prometheus, use the `--prometheus` flag. This will start an HTTP
server on localhost with a `/metrics` endpoint. The Prometheus namespace may
optionally be set with `--prometheus-namespace`.

[cloud monitoring]: https://cloud.google.com/monitoring
[cloud trace]: https://cloud.google.com/trace
[prometheus]: https://prometheus.io/

## Localhost Admin Server

The Proxy includes support for an admin server on localhost. By default,
the admin server is not enabled. To enable the server, pass the --debug or
--quitquitquit flag. This will start the server on localhost at port 9091.
To change the port, use the --admin-port flag.

When --debug is set, the admin server enables Go's profiler available at
/debug/pprof/.

See the [documentation on pprof][pprof] for details on how to use the
profiler.

When --quitquitquit is set, the admin server adds an endpoint at
/quitquitquit. The admin server exits gracefully when it receives a POST
request at /quitquitquit.

[pprof]: https://pkg.go.dev/net/http/pprof.

## Support policy

### Major version lifecycle

This project uses [semantic versioning](https://semver.org/), and uses the
following lifecycle regarding support for a major version:

**Active** - Active versions get all new features and security fixes (that
wouldn’t otherwise introduce a breaking change). New major versions are
guaranteed to be "active" for a minimum of 1 year.

**Deprecated** - Deprecated versions continue to receive security and critical
bug fixes, but do not receive new features. Deprecated versions will be publicly
supported for 1 year.

**Unsupported** - Any major version that has been deprecated for >=1 year is
considered publicly unsupported.

### Supported Go Versions

We test and support at least the latest 3 Go versions. Changes in supported Go
versions will be considered a minor change, and will be noted in the release
notes.

### Release cadence

The AlloyDB Auth Proxy aims for a minimum monthly release cadence. If no new
features or fixes have been added, a new PATCH version with the latest
dependencies is released.

## Contributing

Contributions are welcome. Please, see the [CONTRIBUTING][contributing] document
for details.

Please note that this project is released with a Contributor Code of Conduct. By
participating in this project you agree to abide by its terms.  See [Contributor
Code of Conduct][code-of-conduct] for more information.

[adc]:                   https://cloud.google.com/docs/authentication
[about-proxy]:           https://cloud.google.com/alloydb/docs/auth-proxy/overview
[code-of-conduct]:       CONTRIBUTING.md#contributor-code-of-conduct
[connection-overview]:   https://cloud.google.com/alloydb/docs/auth-proxy/connect
[contributing]:          CONTRIBUTING.md
[releases]:              https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/releases
[roles-and-permissions]: https://cloud.google.com/alloydb/docs/auth-proxy/overview#how-authorized
