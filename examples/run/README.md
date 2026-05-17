# Hello from AlloyDB — Cloud Run + Auth Proxy quickstart

A minimal "hello world" service that deploys to **Cloud Run**, runs the
**AlloyDB Auth Proxy** as a sidecar, and proves a working IAM-authenticated
connection to **AlloyDB**. Open the service URL and you'll see a confirmation
page rendering live data from your instance (PostgreSQL version, server time,
current database, current user).

> **AlloyDB has no free tier.** A running cluster bills by the hour even
> when idle. When you're done, follow [Clean up](#clean-up-to-stop-being-billed)
> to stop the charges.

## What this quickstart teaches

How to connect securely from a Cloud Run service to AlloyDB by running the
Auth Proxy as a sidecar **with IAM database authentication**. The app sees a
plain Postgres connection on `127.0.0.1`; the sidecar handles IAM token
exchange, mTLS, and certificate rotation in a separate process.

The reference app is **Node.js + TypeScript** (Express + `pg`) because that
combination has the widest reach. The language is **incidental** — the
`service.yaml` pattern works for any Postgres client in any language.

## What you get

```
                  ┌─────────── Cloud Run service ───────────┐
                  │                                         │
 ┌─────────┐      │  ┌─────┐         ┌──────────────────┐   │      ┌─────────┐
 │ Browser │─────▶│  │ App │────────▶│   Auth Proxy     │───┼─────▶│ AlloyDB │
 └─────────┘      │  └─────┘         │ ──────────────── │   │      └─────────┘
                  │                  │  • mTLS tunnel   │   │
    HTTPS         │  loopback        │  • IAM authn/z   │   │          mTLS
                  │  (no TLS)        │  • cert rotation │   │
                  │                  └──────────────────┘   │
                  └─────────────────────────────────────────┘
```

The app runs `SELECT version(), now(), current_database(), current_user`
against the local Auth Proxy listener and renders the result.

| Path          | Response                  |
| ------------- | ------------------------- |
| `/`           | HTML status page          |
| `/api/status` | JSON status (same data)   |
| `/healthz`    | `ok` (for uptime checks)  |

The app image is published as:

```
gcr.io/alloydb-connectors/hello-alloydb:latest
```

You don't need to build or push it. See
[Image API (the forever contract)](#image-api-the-forever-contract) for the
guarantees that image makes.

## Prerequisites

### Have these already

- A GCP project with billing enabled and the AlloyDB, Cloud Run, and IAM APIs
  enabled.
- A VPC (the `default` VPC is fine) with [Private Services Access configured
  for AlloyDB][psa].
- An [AlloyDB cluster and primary instance][cluster-create] in that VPC.
- The [gcloud CLI][gcloud-install] installed and authenticated to your
  project.

This quickstart **does not cover** creating the VPC, PSA range, or AlloyDB
cluster — follow the linked docs for that. It owns everything from IAM and
the Cloud Run service onward.

[psa]: https://cloud.google.com/alloydb/docs/configure-connectivity
[cluster-create]: https://cloud.google.com/alloydb/docs/cluster-create
[gcloud-install]: https://cloud.google.com/sdk/docs/install

### Set these shell variables

Set these once at the start of your terminal session — every command below
copy-pastes from here without further substitution.

```sh
export PROJECT_ID=your-project-id
export REGION=us-central1            # region of your AlloyDB cluster
export CLUSTER=your-cluster-id
export INSTANCE=your-primary-instance-id
```

## Step 1 — Set up IAM

Create a dedicated service account, grant it the two AlloyDB roles it needs,
turn on IAM authentication on the instance, and register the service account
as an AlloyDB user.

```sh
# 1a. Create a service account just for this service.
gcloud iam service-accounts create hello-alloydb \
  --project="$PROJECT_ID" \
  --display-name="hello-alloydb Cloud Run quickstart"

# 1b. Grant the API-side role: the Auth Proxy uses this to call the
#     AlloyDB API and fetch connection info + ephemeral certificates.
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/alloydb.client"

# 1c. Grant the DB-as-user role: required for IAM database authentication.
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/alloydb.databaseUser"

# 1d. Turn on IAM authentication on the instance.
gcloud alloydb instances update "$INSTANCE" \
  --cluster="$CLUSTER" \
  --region="$REGION" \
  --database-flags=alloydb.iam_authentication=on

# 1e. Register the service account as an AlloyDB user.
gcloud alloydb users create \
  "hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --cluster="$CLUSTER" \
  --region="$REGION" \
  --type=IAM_BASED
```

Step 1d may trigger an instance restart (a few minutes). The DB username
that the app connects with is the service account email **with the
`.gserviceaccount.com` suffix stripped** — `hello-alloydb@$PROJECT_ID.iam`.
AlloyDB stores it that way internally; the `service.yaml` already sets it
correctly.

## Step 2 — Deploy

`service.yaml` uses shell-style placeholders (`$PROJECT_ID`, `$REGION`,
`$CLUSTER`, `$INSTANCE`). Expand them with `sed` and pipe straight to
`gcloud run services replace`:

```sh
sed -e "s|\$PROJECT_ID|$PROJECT_ID|g" \
    -e "s|\$REGION|$REGION|g" \
    -e "s|\$CLUSTER|$CLUSTER|g" \
    -e "s|\$INSTANCE|$INSTANCE|g" \
    service.yaml \
  | gcloud run services replace - --region="$REGION"
```

Then allow unauthenticated access so you can open the URL in a browser:

```sh
gcloud run services add-iam-policy-binding hello-alloydb \
  --region="$REGION" \
  --member="allUsers" \
  --role="roles/run.invoker"
```

## Step 3 — Open the URL

`gcloud run services replace` prints the service URL. Open it in a browser
and you should see **"Hello from AlloyDB!"** with values returned by your
instance.

If the connection fails, the page renders the driver error in place so you
can debug. The [Troubleshooting](#troubleshooting) table covers the common
causes.

## Image API (the forever contract)

`gcr.io/alloydb-connectors/hello-alloydb:latest` is a stable contract.
New optional features may be added; **these will not be removed or renamed**:

**Environment variables**

| Name      | Required | Default     |
| --------- | -------- | ----------- |
| `DB_USER` | yes      | —           |
| `DB_HOST` | no       | `127.0.0.1` |
| `DB_PORT` | no       | `5432`      |
| `DB_NAME` | no       | `postgres`  |

**Behavior**

- Listens on `:8080`.
- Connects with an empty password (IAM authn — the Auth Proxy supplies the
  token) and `sslmode=disable` (TLS lives between the proxy and AlloyDB,
  not between the app and the proxy).

**Routes**

- `GET /` — HTML status page.
- `GET /api/status` — JSON status.
- `GET /healthz` — plain `ok`.

## When to use the Auth Proxy (vs. an embedded Connector)

Reach for the Auth Proxy when:

- You're deploying an **existing app** and don't want to modify its code.
- Your app is in a **language without an AlloyDB Connector library**.
- You want IAM auth + mTLS handled in a **separate process** from your app.

For *new* apps in **Go**, **Java**, or **Python**, an embedded AlloyDB
**Connector** library is often a better fit — same IAM auth and mTLS, no
sidecar required. See:
[Go][go-conn] · [Java][java-conn] · [Python][py-conn].

[go-conn]: https://github.com/GoogleCloudPlatform/alloydb-go-connector
[java-conn]: https://github.com/GoogleCloudPlatform/alloydb-java-connector
[py-conn]: https://github.com/GoogleCloudPlatform/alloydb-python-connector

## Troubleshooting

| # | Symptom                                                                | Likely cause                                                          | Fix                                                                                                          |
| - | ---------------------------------------------------------------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| 1 | App log: `password authentication failed for user "..."`               | `DB_USER` doesn't match the IAM role name.                            | Set `DB_USER=hello-alloydb@$PROJECT_ID.iam` (drop `.gserviceaccount.com`).                                   |
| 2 | App log: `connection refused` to `127.0.0.1:5432`                      | Proxy sidecar isn't listening yet (or crashed).                       | Check the `alloydb-auth-proxy` container logs in Cloud Run.                                                  |
| 3 | Proxy log: `permission denied` / `403` calling the AlloyDB API         | Service account is missing `roles/alloydb.client`.                    | Grant it (Step 1b).                                                                                          |
| 4 | Proxy log: `i/o timeout` or `no route to host` reaching the instance   | Direct VPC egress misconfigured, or the subnet is in the wrong region. | Confirm the `network-interfaces` annotation in `service.yaml` matches a subnet in `$REGION`.                 |
| 5 | Proxy log / AlloyDB error: `IAM authentication is not enabled`         | `alloydb.iam_authentication=on` flag is missing on the instance.       | Re-run Step 1d.                                                                                              |
| 6 | Auth fails *despite* correct username and the flag being on            | Service account is missing `roles/alloydb.databaseUser`.              | Grant it (Step 1c).                                                                                          |
| 7 | Revision fails to become ready / startup probe failed                  | Proxy can't reach AlloyDB during startup.                              | Check proxy logs first; usually one of #3–#6.                                                                |
| 8 | Service deploys but the URL returns Cloud Run **403**                   | "Allow unauthenticated" binding is missing.                            | Re-run the `add-iam-policy-binding` command in [Step 2](#step-2--deploy).                                    |

## Clean up to stop being billed

```sh
# 1. Delete the Cloud Run service.
gcloud run services delete hello-alloydb --region="$REGION"

# 2. Delete the AlloyDB IAM database user.
gcloud alloydb users delete \
  "hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --cluster="$CLUSTER" --region="$REGION"

# 3. Remove the project-level IAM bindings.
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/alloydb.client"
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/alloydb.databaseUser"

# 4. Delete the service account.
gcloud iam service-accounts delete \
  "hello-alloydb@$PROJECT_ID.iam.gserviceaccount.com"
```

Cloud Run scales to zero, so it stops costing when nobody hits it — but the
**AlloyDB cluster bills hourly regardless of traffic**. If the cluster only
exists for this quickstart, also delete it:

```sh
gcloud alloydb clusters delete "$CLUSTER" --region="$REGION" --force
```

(`--force` also deletes the primary instance and any read pools.)

## What's in this directory

| File                  | Purpose                                                                                  |
| --------------------- | ---------------------------------------------------------------------------------------- |
| `src/main.ts`         | Express app, AlloyDB connection, route handlers.                                         |
| `views/index.ejs`     | Status page template (EJS).                                                              |
| `views/alloydb.svg`   | Branding asset, served at `/alloydb.svg`.                                                |
| `package.json`        | Node.js dependencies and scripts.                                                        |
| `package-lock.json`   | Locked dependency versions.                                                              |
| `tsconfig.json`       | TypeScript compiler config.                                                              |
| `Dockerfile`          | Multi-stage Node.js container image used to publish `gcr.io/alloydb-connectors/hello-alloydb`. |
| `.dockerignore`       | Files excluded from the build context.                                                   |
| `service.yaml`        | Cloud Run service with the Auth Proxy as a sidecar.                                      |
