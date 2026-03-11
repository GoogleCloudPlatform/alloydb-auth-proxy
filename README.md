# AlloyDB Auth Proxy

[![CI][ci-badge]][ci-build]
[![Go Reference][pkg-badge]][pkg-docs]

[ci-badge]: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/actions/workflows/tests.yaml/badge.svg?event=push
[ci-build]: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/actions/workflows/tests.yaml?query=event%3Apush+branch%3Amain
[pkg-badge]: https://pkg.go.dev/badge/github.com/GoogleCloudPlatform/alloydb-auth-proxy.svg
[pkg-docs]: https://pkg.go.dev/github.com/GoogleCloudPlatform/alloydb-auth-proxy

The AlloyDB Auth Proxy is the recommended way to connect to AlloyDB. It provides:

- **Secure connections** — TLS 1.3 encryption and identity verification, independent of the database protocol
- **IAM-based authorization** — controls who can connect to your AlloyDB instances using Google Cloud IAM
- **No certificate management** — no SSL certificates, firewall rules, or IP allowlisting required
- **IAM database authentication** — optional support for automatic IAM DB authentication

> **Note:** The proxy does not configure the network. You must ensure it can
> reach your AlloyDB instance (e.g., by running the proxy inside the same VPC
> as your AlloyDB instance).

If you're using Go, Python, or Java, consider using the language connectors
instead—they embed the same functionality directly in your process:

| Language | Connector |
|----------|-----------|
| Go | [alloydb-go-connector][] |
| Python | [alloydb-python-connector][] |
| Java | [alloydb-java-connector][] |

[alloydb-go-connector]: https://github.com/GoogleCloudPlatform/alloydb-go-connector
[alloydb-python-connector]: https://github.com/GoogleCloudPlatform/alloydb-python-connector
[alloydb-java-connector]: https://github.com/GoogleCloudPlatform/alloydb-java-connector

---

## Quickstart

Get connected in five steps.

### 1. Install the proxy

Pick your platform below, or see [all installation options](#installation).

<!-- {x-release-please-start-version} -->
<details open>
<summary>Linux (amd64)</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.linux.amd64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Apple Silicon)</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.darwin.arm64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Intel)</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.darwin.amd64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Windows (x64)</summary>

```powershell
Invoke-WebRequest -Uri "https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1/alloydb-auth-proxy-x64.exe" -OutFile "alloydb-auth-proxy.exe"
```
</details>

<details>
<summary>Container image</summary>

```sh
docker pull gcr.io/alloydb-connectors/alloydb-auth-proxy:1.14.1
```
</details>
<!-- {x-release-please-end} -->

### 2. Authenticate

The proxy uses [Application Default Credentials (ADC)][adc] by default. Set
them up with gcloud:

```sh
gcloud auth application-default login
```

In Google-managed environments (Cloud Run, GKE, Compute Engine), ADC is
available automatically—no additional setup needed.

### 3. Find your instance URI

```sh
gcloud alloydb instances describe INSTANCE_NAME \
    --region=REGION \
    --cluster=CLUSTER_NAME \
    --format='value(name)'
```

The URI has the form:
```
projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```

### 4. Start the proxy

<details open>
<summary>Binary (Linux / Mac)</summary>

```sh
# By default, the proxy connects over Private Service Access—a private
# connection within the same VPC as your AlloyDB instance. Add --public-ip
# if your instance has a public IP and you are not connecting from within
# the VPC.
./alloydb-auth-proxy projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```
</details>

<details>
<summary>Binary (Windows)</summary>

```powershell
.\alloydb-auth-proxy.exe projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```
</details>

<!-- {x-release-please-start-version} -->
<details>
<summary>Container image</summary>

```sh
# Mounts your local gcloud credentials into the container
docker run --rm \
  -v "$HOME/.config/gcloud:/gcloud" \
  -e GOOGLE_APPLICATION_CREDENTIALS=/gcloud/application_default_credentials.json \
  -p 1.14.1.1:5432:5432 \
  gcr.io/alloydb-connectors/alloydb-auth-proxy:1.14.1 \
  --address 1.14.1.0 \
  projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```
</details>
<!-- {x-release-please-end} -->

You should see output like:
```
Authorizing with Application Default Credentials
Listening on 127.0.0.1:5432
The proxy has started successfully and is ready for new connections!
```

### 5. Connect

In a separate terminal, connect with any Postgres client:

```sh
psql "host=127.0.0.1 port=5432 user=DB_USER dbname=DB_NAME"
```

## Table of contents

- [Installation](#installation)
  - [Binary](#binary)
  - [Container image](#container-image)
  - [Build from source](#build-from-source)
- [Authentication](#authentication)
- [Usage](#usage)
  - [Basic usage](#basic-usage)
  - [Multiple instances](#multiple-instances)
  - [Custom address and port](#custom-address-and-port)
  - [Public IP](#public-ip)
  - [Auto IAM Authentication](#auto-iam-authentication)
  - [Per-instance configuration](#per-instance-configuration)
  - [Unix sockets](#unix-sockets)
  - [Config file](#config-file)
  - [Environment variables](#environment-variables)
- [Running behind a SOCKS5 proxy](#running-behind-a-socks5-proxy)
- [Observability](#observability)
  - [Health checks](#health-checks)
  - [Prometheus metrics](#prometheus-metrics)
  - [Cloud Monitoring and Cloud Trace](#cloud-monitoring-and-cloud-trace)
  - [Debug logging](#debug-logging)
  - [Admin server (pprof / graceful shutdown)](#admin-server-pprof--graceful-shutdown)
- [Reference](#reference)
- [Support policy](#support-policy)
- [Contributing](#contributing)

---

## Installation

### Binary

Download the latest binary for your OS and architecture from
[releases][releases].

<!-- {x-release-please-start-version} -->
<details open>
<summary>Linux amd64</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.linux.amd64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux 386</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.linux.386" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux arm64</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.linux.arm64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Linux arm</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.linux.arm" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Intel)</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.darwin.amd64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Mac (Apple Silicon)</summary>

```sh
URL="https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1"
wget "$URL/alloydb-auth-proxy.darwin.arm64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
```
</details>

<details>
<summary>Windows x64</summary>

```powershell
Invoke-WebRequest -Uri "https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1/alloydb-auth-proxy-x64.exe" -OutFile "alloydb-auth-proxy.exe"
```
</details>

<details>
<summary>Windows x86</summary>

```powershell
Invoke-WebRequest -Uri "https://storage.googleapis.com/alloydb-auth-proxy/v1.14.1/alloydb-auth-proxy-x86.exe" -OutFile "alloydb-auth-proxy.exe"
```
</details>
<!-- {x-release-please-end} -->

### Container image

Container images are available from [Artifact Registry][]:

- [`gcr.io/alloydb-connectors/alloydb-auth-proxy`](https://gcr.io/alloydb-connectors/alloydb-auth-proxy)
- [`us.gcr.io/alloydb-connectors/alloydb-auth-proxy`](https://us.gcr.io/alloydb-connectors/alloydb-auth-proxy)
- [`eu.gcr.io/alloydb-connectors/alloydb-auth-proxy`](https://eu.gcr.io/alloydb-connectors/alloydb-auth-proxy)
- [`asia.gcr.io/alloydb-connectors/alloydb-auth-proxy`](https://asia.gcr.io/alloydb-connectors/alloydb-auth-proxy)

> [!NOTE]
> These images were migrated from Google Container Registry (deprecated) to
> Artifact Registry, which is why they still use the `gcr.io` naming prefix.

Each image is tagged with the proxy version. Available tag variants:

Tag                | Base image
------------------ | -------------------------------------------
`VERSION`          | [distroless][] (default, non-root, minimal)
`VERSION-alpine`   | Alpine
`VERSION-bookworm` | Debian Bookworm

Use Alpine or Debian variants when you need a shell or debugging tools.

<!-- {x-release-please-start-version} -->
```sh
# Pull a specific version (recommended over :latest)
docker pull gcr.io/alloydb-connectors/alloydb-auth-proxy:1.14.1
```
<!-- {x-release-please-end} -->

Pin to a specific version tag and use CI automation to keep it updated.

**Running with Docker:**

```sh
docker run --rm \
  -v "$HOME/.config/gcloud:/gcloud" \
  -e GOOGLE_APPLICATION_CREDENTIALS=/gcloud/application_default_credentials.json \
  -p 127.0.0.1:5432:5432 \
  gcr.io/alloydb-connectors/alloydb-auth-proxy:1.13.11 \
  projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE \
  --address 0.0.0.0
```

### Build from source

Requires the latest version of [Go](https://go.dev/doc/install).

```sh
go install github.com/GoogleCloudPlatform/alloydb-auth-proxy@latest
```

The binary is placed in `$GOPATH/bin` or `$HOME/go/bin`.

[Artifact Registry]: https://cloud.google.com/artifact-registry
[distroless]: https://github.com/GoogleContainerTools/distroless

---

## Authentication

The proxy uses [Application Default Credentials (ADC)][adc] by default and
this is the recommended approach for most use cases. ADC automatically picks
up credentials from the environment—no flags needed:

```sh
# One-time setup on a developer machine
gcloud auth application-default login
```

In Google-managed environments (Cloud Run, GKE, Compute Engine), ADC is
already available and requires no additional configuration.

For less-common scenarios, the proxy also accepts explicit credentials via flags:

| Flag | Description |
|------|-------------|
| `--credentials-file PATH` | Path to a service account key JSON file |
| `--token TOKEN` | An OAuth2 Bearer token |

**Required IAM roles** for any principal connecting through the proxy:

- `roles/alloydb.client` (Cloud AlloyDB Client)
- `roles/serviceusage.serviceUsageConsumer` (Service Usage Consumer)

See [Roles and Permissions in AlloyDB][roles-and-permissions] for details.

**Service account impersonation** is also supported:

```sh
./alloydb-auth-proxy \
    --impersonate-service-account=SA@PROJECT.iam.gserviceaccount.com \
    projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```

For delegation chains, supply a comma-separated list where the first entry is
the target and each subsequent entry is a delegate:

```sh
./alloydb-auth-proxy \
    --impersonate-service-account=TARGET_SA,DELEGATE_SA \
    projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```

---

## Usage

All examples below assume valid credentials are present. Replace
`INSTANCE_URI` with the full instance path:

```
projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE
```

### Basic usage

```sh
# Listens on 127.0.0.1:5432 using private IP
./alloydb-auth-proxy INSTANCE_URI
```

### Multiple instances

```sh
# First instance: 127.0.0.1:5432, second: 127.0.0.1:5433
./alloydb-auth-proxy INSTANCE_URI_1 INSTANCE_URI_2
```

### Custom address and port

```sh
# Listen on all interfaces, port 6000
./alloydb-auth-proxy --address 0.0.0.0 --port 6000 INSTANCE_URI
```

### Public IP

```sh
./alloydb-auth-proxy --public-ip INSTANCE_URI
```

### Auto IAM Authentication

Lets the proxy supply the IAM principal's OAuth2 token as the database
password—no password prompt needed for the client.

```sh
./alloydb-auth-proxy --auto-iam-authn INSTANCE_URI
```

### Per-instance configuration

Override address, port, or other settings for individual instances using a
query-string appended to the instance URI (wrap in quotes to protect `&` from
the shell):

```sh
./alloydb-auth-proxy \
    'INSTANCE_URI_1?address=0.0.0.0&port=6000' \
    'INSTANCE_URI_2?address=127.0.0.1&port=7000&auto-iam-authn=true'
```

### Unix sockets

```sh
# All instances use a Unix socket under /run/alloydb
./alloydb-auth-proxy --unix-socket /run/alloydb INSTANCE_URI

# Per-instance path (Postgres appends .s.PGSQL.5432 automatically)
./alloydb-auth-proxy 'INSTANCE_URI?unix-socket-path=/path/to/socket'
```

### Config file

Instead of flags, you can supply a TOML, YAML, or JSON config file:

```sh
./alloydb-auth-proxy --config-file config.toml
```

Example `config.toml`:

```toml
instance-uri   = "projects/PROJECT/locations/REGION/clusters/CLUSTER/instances/INSTANCE"
auto-iam-authn = true
debug-logs     = true
```

Multiple instances:

```toml
instance-uri-0 = "INSTANCE_URI_1"
instance-uri-1 = "INSTANCE_URI_2"
```

### Environment variables

Every flag has an environment variable equivalent using the
`ALLOYDB_PROXY_` prefix (uppercase, underscores):

```sh
# Equivalent to --structured-logs
ALLOYDB_PROXY_STRUCTURED_LOGS=true ./alloydb-auth-proxy INSTANCE_URI

# Single instance via env var
ALLOYDB_PROXY_INSTANCE_URI=INSTANCE_URI ./alloydb-auth-proxy

# Multiple instances via env vars
ALLOYDB_PROXY_INSTANCE_URI_0=INSTANCE_URI_1 \
ALLOYDB_PROXY_INSTANCE_URI_1=INSTANCE_URI_2 \
    ./alloydb-auth-proxy
```

---

## Running behind a SOCKS5 proxy

```sh
ALL_PROXY=socks5://localhost:8000 \
HTTPS_PROXY=socks5://localhost:8000 \
    ./alloydb-auth-proxy INSTANCE_URI
```

`ALL_PROXY` routes TCP traffic to AlloyDB (supports `socks5` and `socks5h`).
Use `socks5h` to route DNS lookups through the proxy. `HTTPS_PROXY` routes
HTTP(S) traffic to the AlloyDB Admin API (optional).

---

## Observability

### Health checks

Enable HTTP health check endpoints (useful for Kubernetes probes):

```sh
./alloydb-auth-proxy --health-check INSTANCE_URI
```

| Endpoint | Returns 200 when... |
|----------|---------------------|
| `/startup` | Proxy has finished starting up |
| `/readiness` | Proxy is started, has available connections, and can reach all instances |
| `/liveness` | Always 200 — if unresponsive, restart the proxy |

Configure address and port with `--http-address` and `--http-port` (default:
`localhost:9090`).

### Prometheus metrics

```sh
./alloydb-auth-proxy --prometheus INSTANCE_URI
# Metrics available at http://localhost:9090/metrics
```

Use `--prometheus-namespace` to set a custom namespace prefix.

### Cloud Monitoring and Cloud Trace

```sh
./alloydb-auth-proxy --telemetry-project=PROJECT_ID INSTANCE_URI
```

Use `--disable-metrics` or `--disable-traces` to opt out of either. Use
`--telemetry-prefix` to customize the Cloud Monitoring metric prefix.

**Supported metrics:**

| Metric | Description |
|--------|-------------|
| `alloydbconn/dial_latency` | Distribution of dialer latencies (ms) |
| `alloydbconn/open_connections` | Current number of open AlloyDB connections |
| `alloydbconn/dial_failure_count` | Number of failed dial attempts |
| `alloydbconn/refresh_success_count` | Successful certificate refresh operations |
| `alloydbconn/refresh_failure_count` | Failed certificate refresh operations |

### Debug logging

```sh
./alloydb-auth-proxy --debug-logs INSTANCE_URI
```

Logs internal certificate refresh operations. Useful when diagnosing
unexpected proxy behavior.

### Admin server (pprof / graceful shutdown)

The admin server runs on `localhost:9091` and is disabled by default.

```sh
# Enable Go profiler at /debug/pprof/
./alloydb-auth-proxy --debug INSTANCE_URI

# Enable graceful shutdown via POST /quitquitquit
./alloydb-auth-proxy --quitquitquit INSTANCE_URI
```

Change the port with `--admin-port`. See the [pprof documentation][pprof] for
profiler usage.

---

## Reference

Run `./alloydb-auth-proxy --help` for full flag documentation, or browse the
rendered docs in [docs/cmd](docs/cmd).

**Commonly used flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-a, --address` | `127.0.0.1` | Address for instance listeners |
| `-p, --port` | `5432` | Starting port; subsequent instances increment |
| `-i, --auto-iam-authn` | false | Enable Auto IAM Authentication |
| `-c, --credentials-file` | | Path to service account key JSON |
| `-t, --token` | | OAuth2 Bearer token |
| `--public-ip` | false | Connect via public IP |
| `--psc` | false | Connect via Private Service Connect |
| `-u, --unix-socket` | | Directory for Unix socket listeners |
| `--lazy-refresh` | false | Refresh certs on-demand (for throttled CPUs) |
| `--health-check` | false | Enable `/startup`, `/liveness`, `/readiness` |
| `--prometheus` | false | Enable Prometheus `/metrics` endpoint |
| `--structured-logs` | false | Emit logs in LogEntry JSON format |
| `--max-connections` | 0 (unlimited) | Maximum simultaneous connections |
| `--config-file` | | Path to TOML/YAML/JSON config file |

---

## Support policy

### Major version lifecycle

This project follows [semantic versioning](https://semver.org/).

| Status | Description |
|--------|-------------|
| **Active** | Receives all new features and security fixes. Guaranteed for at least 1 year. |
| **Deprecated** | Security and critical bug fixes only. Publicly supported for 1 year after deprecation. |
| **Unsupported** | Major versions deprecated for ≥ 1 year. |

### Release cadence

The proxy targets a monthly release cadence. If no new features or fixes are
added, a new PATCH version is released with the latest dependencies.

---

## Contributing

Contributions are welcome. See the [CONTRIBUTING][contributing] document for
details.

This project is released with a [Contributor Code of Conduct][code-of-conduct].
By participating, you agree to abide by its terms.

---

[adc]:                   https://cloud.google.com/docs/authentication
[code-of-conduct]:       CONTRIBUTING.md#contributor-code-of-conduct
[contributing]:          CONTRIBUTING.md
[pprof]:                 https://pkg.go.dev/net/http/pprof
[releases]:              https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/releases
[roles-and-permissions]: https://cloud.google.com/alloydb/docs/auth-proxy/overview#how-authorized
