# AlloyDB Auth Proxy

The AlloyDB Auth Proxy is a binary that provides IAM-based authorization and
encryption when connecting to an AlloyDB instance.

See the [Connecting Overview][connection-overview] page for more information on
connecting to an AlloyDB instance, or the [About the proxy][about-proxy] page
for details on how the AlloyDB Auth Proxy works.

Note: The Proxy *cannot* provide a network path to an AlloyDB instance if one is
not already present (e.g., the proxy cannot access a VPC if it does not already
have access to it).

## Installation

For 64-bit Linux, run:

```
URL="https://storage.googleapis.com/alloydb-auth-proxy"
VERSION=v0.1.0 # see Releases for other versions

wget "$URL/$VERSION/alloydb-auth-proxy.linux.amd64" -O alloydb-auth-proxy

chmod +x cloud_sql_proxy
```

Releases for additional OS's and architectures and be found on the [releases
page][releases].

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
  base image (only supported from v1.17 up)
* `$VERSION-buster` - uses [`debian:buster`](https://hub.docker.com/_/debian) as
  a base image (only supported from v1.17 up)

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

``` bash
# Starts the proxy listening on 127.0.0.1:5432
alloydb-auth-proxy <INSTANCE_URI>
```

To connect to multiple instances, use:

``` bash
# For instance 1, the proxy listens on 127.0.0.1:5432
# For instance 2, the proxy listens on 127.0.0.1:5433
alloydb-auth-proxy <INSTANCE_URI_1> <INSTANCE_URI_2>
```

To override the default address the proxy listens on, use the `--address` flag:

``` bash
# Starts the proxy listening on 0.0.0.0:5432
alloydb-auth-proxy --address 0.0.0.0 <INSTANCE_URI>
```

To override the default port, use the `--port` flag:

``` bash
# Starts the proxy listening on 127.0.0.1:6000
alloydb-auth-proxy --port 6000 <INSTANCE_URI>
```

In addition, both `address` and `port` may be overrided on a per-instance level
with a query-string style syntax:

``` bash
alloydb-auth-proxy \
    '<INSTANCE_URI_1>?address=0.0.0.0&port=6000' \
    '<INSTANCE_URI_2>?address=127.0.0.1&port=7000'
```

Note: when using the query-string syntax, the instance URI and query parameters
must be wrapped in quotes.

## Credentials

The AlloyDB Auth Proxy uses a Cloud IAM account to authorize connections against
an AlloyDB instance. The proxy supports the following options:

1. A `-credential-file` flag for a service account key file.
2. A `-token` flag for a OAuth2 Bearer token
3. [Application Default Credentials (ADC)][adc] if neither of the above have
   been set.

Note: Any account connecting to AlloyDB instance will need one of the following
IAM roles:

- AlloyDB Client (preferred)
- AlloyDB Editor
- AlloyDB Admin

Or one may manually assign the following IAM permissions:

- `alloydb.instances.connect`
- `alloydb.instances.get`

See [Roles and Permissions in AlloyDB][roles-and-permissions] for details.

When the proxy authenticates under the Compute Engine VM's default service
account, the VM must have the `cloud-platform` API scope (i.e.,
"https://www.googleapis.com/auth/cloud-platform") and the associated project
must have the AlloyDB Admin API enabled. The default service account must also
have at least writer or editor privileges to any projects of target AlloyDB
instances.

## Running behind a Socks5 proxy

The AlloyDB Auth Proxy includes support for sending requests through a SOCKS5
proxy. If a SOCKS5 proxy is running on `localhost:8000`, the command to start
the AlloyDB Auth Proxy would look like:

```
ALL_PROXY=socks5://localhost:8000 \
HTTPS_PROXY=socks5://localhost:8000 \
    alloydb-auth-proxy <INSTANCE_URI>
```

The `ALL_PROXY` environment variable specifies the proxy for all TCP traffic to
and from a AlloyDB instance. The `ALL_PROXY` environment variable supports
`socks5` and `socks5h` protocols. To route DNS lookups through a proxy, use the
`socks5h` protocol.

The `HTTPS_PROXY` (or `HTTP_PROXY`) specifies the proxy for all HTTP(S) traffic
to the SQL Admin API. Specifying `HTTPS_PROXY` or `HTTP_PROXY` is only necessary
when you want to proxy this traffic. Otherwise, it is optional. See
[`http.ProxyFromEnvironment`](https://pkg.go.dev/net/http@go1.17.3#ProxyFromEnvironment)
for possible values.

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
[about-proxy]:           TODO
[code-of-conduct]:       CONTRIBUTING.md#contributor-code-of-conduct
[contributing]:          CONTRIBUTING.md
[releases]:              https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/releases
[roles-and-permissions]: https://cloud.google.com/alloydb/docs/roles-and-permissions
