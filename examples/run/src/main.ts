// Copyright 2026 Google LLC
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

// hello-alloydb: a Cloud Run quickstart that proves a working connection
// to AlloyDB via the Auth Proxy sidecar.

import path from "node:path";
import express, { Request, Response } from "express";
import { Client } from "pg";

// The Auth Proxy sidecar exposes a localhost listener. From the app's
// point of view this is a plain Postgres connection on 127.0.0.1 — the
// sidecar handles IAM authorization and the mTLS tunnel to AlloyDB.
const DB_HOST = process.env.DB_HOST ?? "127.0.0.1";
const DB_PORT = parseInt(process.env.DB_PORT ?? "5432", 10);
const DB_NAME = process.env.DB_NAME ?? "postgres";
const DB_USER = process.env.DB_USER;
if (!DB_USER) {
  console.error("DB_USER environment variable is required");
  process.exit(1);
}

type Status = {
  ok: boolean;
  host: string;
  port: number;
  version?: string;
  server_time?: string;
  database?: string;
  user?: string;
  error?: string;
};

async function collectStatus(): Promise<Status> {
  const s: Status = { ok: false, host: DB_HOST, port: DB_PORT };
  // IAM authentication: the Auth Proxy obtains an OAuth token for the
  // service account and presents it to AlloyDB as the password. The app
  // sends an empty password and connects in plaintext to the local
  // sidecar; TLS lives between the proxy and AlloyDB.
  const client = new Client({
    host: DB_HOST,
    port: DB_PORT,
    user: DB_USER,
    password: "",
    database: DB_NAME,
    ssl: false,
    connectionTimeoutMillis: 5_000,
  });
  try {
    await client.connect();
    const { rows } = await client.query(
      "SELECT version(), now(), current_database(), current_user",
    );
    const row = rows[0] as {
      version: string;
      now: Date;
      current_database: string;
      current_user: string;
    };
    s.ok = true;
    s.version = row.version;
    s.server_time = formatTimestamp(row.now);
    s.database = row.current_database;
    s.user = row.current_user;
  } catch (e) {
    const err = e as Error;
    s.error = `${err.name}: ${err.message}`;
  } finally {
    await client.end().catch(() => {});
  }
  return s;
}

function formatTimestamp(d: Date): string {
  return d.toISOString().replace("T", " ").replace(/\..*/, " UTC");
}

const VIEWS_DIR = path.join(__dirname, "..", "views");

const app = express();
app.set("view engine", "ejs");
app.set("views", VIEWS_DIR);

app.get("/", async (_req: Request, res: Response) => {
  const s = await collectStatus();
  res
    .status(s.ok ? 200 : 500)
    .set("Cache-Control", "no-store")
    .render("index", s);
});

app.get("/api/status", async (_req: Request, res: Response) => {
  const s = await collectStatus();
  res
    .status(s.ok ? 200 : 500)
    .set("Cache-Control", "no-store")
    .json(s);
});

app.get("/alloydb.svg", (_req: Request, res: Response) => {
  res
    .set("Cache-Control", "public, max-age=86400")
    .type("image/svg+xml")
    .sendFile(path.join(VIEWS_DIR, "alloydb.svg"));
});

app.get("/healthz", (_req: Request, res: Response) => {
  res.type("text/plain").send("ok");
});

const port = 8080;
app.listen(port, () => {
  console.log(`Listening on :${port}`);
});
