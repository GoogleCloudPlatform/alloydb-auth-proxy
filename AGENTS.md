# Agents Instructions

## What the Proxy Does (and Does Not Do)

The proxy provides **authentication** (IAM-based) and **encryption** (mTLS 1.3). It does **not** tunnel network traffic — it connects directly to the AlloyDB instance IP. Users must have network connectivity to the instance independently. This is the most common source of confusion. See [Local Development](#local-development).

---

## Prerequisites

| Requirement | Notes |
|---|---|
| Google Cloud SDK | Required for ADC. `brew install --cask google-cloud-sdk` on macOS |
| GCP project with billing | AlloyDB requires billing enabled |
| AlloyDB cluster + primary instance | See [Infrastructure Setup](#infrastructure-setup) |
| Two IAM roles | `roles/alloydb.client` AND `roles/serviceusage.serviceUsageConsumer` — both required |
| Application Default Credentials | `gcloud auth application-default login` with the correct account |

---

## Authentication Setup

```bash
gcloud auth login
gcloud config set project PROJECT_ID
gcloud auth application-default login
```

**Wrong account cached**: `gcloud init` may default to a previously cached account. If you see permission errors:
```bash
gcloud auth list                       # verify active account
gcloud projects describe PROJECT_ID    # errors here = wrong account

gcloud auth revoke --all
gcloud auth application-default revoke --quiet
gcloud auth login
gcloud auth application-default login
```

**ADC quota project warning** (non-fatal, indicates wrong account):
```
Cannot add the project to ADC as the quota project because the account does not have
the "serviceusage.services.use" permission on this project.
```
Re-authenticate with the account that owns the project.

---

## Infrastructure Setup

### 1. Enable APIs
```bash
gcloud services enable \
  alloydb.googleapis.com \
  servicenetworking.googleapis.com \
  cloudresourcemanager.googleapis.com
```

### 2. Configure Private Services Access
Required before cluster creation — no fallback.
```bash
gcloud compute addresses create google-managed-services-default \
  --global --purpose=VPC_PEERING --prefix-length=16 --network=default

gcloud services vpc-peerings connect \
  --service=servicenetworking.googleapis.com \
  --ranges=google-managed-services-default \
  --network=default
```

> **If VPC peering already exists** (e.g. Firebase project), `connect` fails with `Cannot modify allocated ranges in CreateConnection`. Use `update` instead:
> ```bash
> gcloud compute addresses list --global    # find existing range name
> gcloud services vpc-peerings update \
>   --service=servicenetworking.googleapis.com \
>   --ranges=EXISTING_RANGE,google-managed-services-default \
>   --network=default --force
> ```

### 3. Create cluster and instance
Expect **5–10 minutes per step** (~15–20 min total).

**Choose instance size** — memory scales automatically at 8 GB per vCPU:

| `--cpu-count` | Memory |
|---|---|
| 2 | 16 GB |
| 4 | 32 GB |
| 8 | 64 GB |
| 16 | 128 GB |
| 32 | 256 GB |
| 64 | 512 GB |
| 96 | 768 GB |

```bash
gcloud alloydb clusters create CLUSTER_ID \
  --region=REGION --password=STRONG_PASSWORD --network=default

gcloud alloydb instances create INSTANCE_ID \
  --instance-type=PRIMARY --cpu-count=CPUS \
  --cluster=CLUSTER_ID --region=REGION
```

For a read pool instance, also specify node count:
```bash
gcloud alloydb instances create INSTANCE_ID \
  --instance-type=READ_POOL --cpu-count=CPUS \
  --read-pool-node-count=NODES \
  --cluster=CLUSTER_ID --region=REGION
```

### 4. Grant IAM roles
Both are required. Missing either causes a cryptic proxy startup failure that does not name the absent role.
```bash
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="user:YOUR_EMAIL" --role="roles/alloydb.client"

gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="user:YOUR_EMAIL" --role="roles/serviceusage.serviceUsageConsumer"
```

---

## Local Development

AlloyDB instances are assigned a private IP by default (`10.x.x.x`), unreachable from a laptop. The proxy starts cleanly but every connection times out:
```
failed to connect to instance: dial tcp 10.x.x.x:5433: i/o timeout
```
For local dev, enable a public IP on the instance. For production, run the proxy inside the VPC (GKE, Cloud Run, Compute Engine).

### Enable public IP
```bash
gcloud alloydb instances update INSTANCE_ID \
  --cluster=CLUSTER_ID --region=REGION \
  --assign-inbound-public-ip=ASSIGN_IPV4 \
  --database-flags=password.enforce_complexity=on
```

> Enabling `password.enforce_complexity=on` invalidates existing passwords. Reset immediately:
> ```bash
> gcloud alloydb users set-password postgres \
>   --cluster=CLUSTER_ID --region=REGION --project=PROJECT_ID \
>   --password='StrongPassword123!'
> ```

### Download and run the proxy
```bash
# macOS Apple Silicon
curl -o alloydb-auth-proxy \
  https://storage.googleapis.com/alloydb-auth-proxy/v1.14.3/alloydb-auth-proxy.darwin.arm64
chmod +x alloydb-auth-proxy

# macOS Intel
curl -o alloydb-auth-proxy \
  https://storage.googleapis.com/alloydb-auth-proxy/v1.14.3/alloydb-auth-proxy.darwin.amd64
chmod +x alloydb-auth-proxy

# Linux amd64
curl -o alloydb-auth-proxy \
  https://storage.googleapis.com/alloydb-auth-proxy/v1.14.3/alloydb-auth-proxy.linux.amd64
chmod +x alloydb-auth-proxy

# Run (local dev with public IP)
./alloydb-auth-proxy --public-ip \
  "projects/PROJECT_ID/locations/REGION/clusters/CLUSTER_ID/instances/INSTANCE_ID"

# Use a non-default port if 5432 is taken by a local Postgres instance
./alloydb-auth-proxy --public-ip --port 5433 \
  "projects/PROJECT_ID/locations/REGION/clusters/CLUSTER_ID/instances/INSTANCE_ID"
```

> **Port conflict**: The proxy defaults to 5432. Check with `lsof -i :5432`; use `--port` to pick a free port.

---

## Database User Setup

Connect as `postgres` through the proxy, then:
```sql
CREATE DATABASE app_db;
CREATE USER app_user WITH PASSWORD 'StrongPassword123!';
GRANT ALL PRIVILEGES ON DATABASE app_db TO app_user;
```

Connect to `app_db`, then:
```sql
-- Required in PostgreSQL 15+ (AlloyDB runs PG 15+)
GRANT ALL ON SCHEMA public TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO app_user;
```

> `GRANT ALL PRIVILEGES ON DATABASE` does not include schema-level access in PG 15+. Without `GRANT ALL ON SCHEMA public`, the user gets `permission denied for schema public` when creating tables.

> Running `GRANT ALL PRIVILEGES ON ALL TABLES` produces warnings for AlloyDB's internal system views (`google_db_advisor_*`, `g_columnar_*`). These are safe to ignore.

---

## Known Issues

**Health check returns 200 before AlloyDB is reachable (#909)**
`/startup` and `/readiness` return HTTP 200 as soon as the local TCP listener starts — before any connection to AlloyDB is established. In PSC environments, the network path can take 10+ seconds after startup. Add an application-level retry loop rather than relying solely on the proxy health check.

**Kubernetes Jobs: proxy doesn't exit cleanly (#727)**
The proxy sidecar doesn't exit with code 0 when the app container finishes, failing the Job. Use `--quitquitquit` to enable the admin endpoint and call it from the app before exit:
```bash
curl -X POST http://localhost:9091/quitquitquit
```

**Auto IAM auth with impersonated service accounts (#890)**
`--auto-iam-authn` combined with `--impersonate-service-account` fails when Managed Connection Pooling is enabled. Use `--credentials-file` with a dedicated service account key instead.

**Noisy metrics permission logs (#884)**
Missing `roles/monitoring.metricWriter` causes a non-fatal `PermissionDenied` log every 60 seconds. Grant the role or suppress with `--disable-metrics`.

**Auto IAM auth with SQLAlchemy (#724)**
SQLAlchemy requires an explicit password field even when using IAM auth. Pass an empty string:
```python
engine = create_engine(
    "postgresql+psycopg2://iam_user@localhost:5432/db",
    connect_args={"password": ""}
)
```

---

## Error Reference

| Error | Cause | Fix |
|---|---|---|
| `dial tcp 10.x.x.x: i/o timeout` | Private IP unreachable from laptop | Enable public IP; use `--public-ip` |
| `address already in use` on 5432 | Local Postgres running | Use `--port 5433` |
| `password authentication failed` | Wrong password or complexity flag invalidated it | `gcloud alloydb users set-password` |
| `permission denied for schema public` | Missing schema grant (PG 15+) | `GRANT ALL ON SCHEMA public TO user` |
| `password complexity flag...is required` | Enabling public IP without complexity flag | Add `--database-flags=password.enforce_complexity=on` |
| `Cannot modify allocated ranges in CreateConnection` | VPC peering already exists | Use `vpc-peerings update` not `connect` |
| `failed to export metrics: PermissionDenied` | Missing `monitoring.metricWriter` (non-fatal) | `--disable-metrics` or grant role |
| DNS `unknown port` recurring every 60s | Malformed instance URI | Verify: `projects/P/locations/R/clusters/C/instances/I` |

---

## Common Commands

```bash
# Check proxy is running
lsof -i :5432

# Describe instance (state + IPs)
gcloud alloydb instances describe INSTANCE \
  --cluster=CLUSTER --region=REGION \
  --format="value(state, ipAddress, publicIpAddress)"

# Reset a user's password
gcloud alloydb users set-password USERNAME \
  --cluster=CLUSTER --region=REGION --password='NewPassword'

# Connect via psql
PGPASSWORD='...' psql "host=127.0.0.1 port=5433 user=postgres dbname=postgres"
```

---

## Deployment Patterns

| Environment | Proxy flag | Notes |
|---|---|---|
| Local dev | `--public-ip` | Instance must have public IP enabled |
| GKE / Kubernetes | _(none)_ | Run as sidecar; pod is inside VPC |
| Cloud Run | _(none)_ | Requires Serverless VPC Connector |
| Compute Engine | _(none)_ | VM is inside VPC |

---

## References

- [AlloyDB Auth Proxy repo](https://github.com/GoogleCloudPlatform/alloydb-auth-proxy)
- [Connect via public IP](https://cloud.google.com/alloydb/docs/connect-public-ip)
- [Connect using the auth proxy](https://cloud.google.com/alloydb/docs/auth-proxy/connect)
