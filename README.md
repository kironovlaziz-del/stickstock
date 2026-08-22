# StickStock

**Self-hosted BI platform** – connect to any data source, query and visualize data, run real statistical analysis. Backend in Go, analytics service in Python, deploy on your own VPS. No LLM, no external API keys – all analytics runs locally on Polars, DuckDB, statsmodels, and PyOD.

---

## Key Features

- **Self‑hosted** – full control over your data, no cloud lock‑in.
- **Multiple data sources** – Postgres, MySQL, MongoDB, REST APIs, and CSV uploads (through isolated DuckDB).
- **Visual query builder** – build SQL queries without writing code.
- **Dashboards & widgets** – 8 chart types (bar, line, pie, scatter, table, heatmap, boxplot, treemap) with drag‑and‑drop layout.
- **Advanced analytics** – linear regression, t‑tests, ARIMA forecasting, anomaly detection (PyOD), and aggregations (Polars).
- **Enterprise‑grade security** – JWT (HS256/ES256) with JWKS support, read‑only SQL, encrypted credentials (AES‑256‑GCM), SSRF protection, rate limiting, and fail‑closed access control.
- **Collaboration** – share dashboards via public links (with expiration and rate limits), add team members (editor/viewer roles), and comment on widgets.
- **Scheduled reports** – send query results via email or Telegram using cron expressions.
- **Multi‑language** – English, Russian, Uzbek, Kazakh, Tajik.
- **Lightweight** – runs on a 4 GB RAM / 2 CPU VPS.

---

## Architecture
┌────────────┐
stickstock.lol ─▶ │ nginx │ (TLS termination, reverse proxy)
└─────┬──────┘
│ /api, /ws
┌─────▼──────┐ ┌────────────────────┐
│ backend │──REST──▶ analytics-service │
│ (Go) │ │ (Python) │
└─────┬──────┘ └────────────────────┘
│ Polars · DuckDB ·
│ DATABASE_URL statsmodels · PyOD
│ (Postgres wire (all local, no API keys)
│ protocol, TLS)
┌─────▼──────┐
│ Supabase │ managed Postgres + Auth (Google/GitHub/
│ (external) │ email OAuth), not a container on the VPS
└────────────┘

- **backend/** – Go API, connectors, query engine, cache (Postgres table), dashboard/widget CRUD, authentication (JWT from Supabase).
- **analytics-service/** – Python/FastAPI with Polars, DuckDB, statsmodels, scipy, PyOD for heavier data processing.
- **frontend/** – Next.js application (React, Tailwind, Recharts, react‑grid‑layout).
- **nginx/** – reverse proxy with SSL termination.

---

## Security Hardening (what we fixed)

We have performed a thorough security audit and implemented the following:

- **CSV isolation** – uploaded CSV files are stored outside Postgres and queried via an isolated DuckDB instance (no access to internal tables).
- **Encrypted credentials** – all data source DSNs are encrypted at rest using AES‑256‑GCM (key from environment).
- **SSRF protection** – REST connector validates target URLs, blocks private IP ranges and internal Docker hostnames, and restricts HTTP methods (GET/HEAD only).
- **Read‑only SQL** – only SELECT statements are allowed; writes are blocked at the query engine level.
- **Fail‑closed authentication** – if a user profile is missing or blocked, access is denied (no fallback to open).
- **Rate limiting** – public dashboard endpoints have per‑IP/per‑token rate limits (10 requests/minute) and link expiration (30 days).
- **Worker locking** – scheduled reports use `SELECT ... FOR UPDATE SKIP LOCKED` to prevent duplicate executions.
- **RLS tightened** – direct `UPDATE`/`INSERT`/`DELETE` on critical tables are revoked from `authenticated` role; all writes go through Go API.
- **Secure logging** – decryption errors are logged, but no sensitive data is exposed.

---

## Technology Stack

| Component | Language / Libraries |
|-----------|----------------------|
| Backend   | Go 1.22, `pgx`, `jwt-go`, `crypto/aes`, `net` |
| Analytics | Python 3.11, FastAPI, Polars, DuckDB, statsmodels, PyOD, scipy |
| Frontend  | Next.js 14, React, Tailwind, Recharts, react-grid-layout |
| Database  | PostgreSQL (via Supabase), with a `cache` table instead of Redis |
| Auth      | Supabase Auth (JWT with HS256/ES256, JWKS support) |
| Deployment| Docker, Docker Compose, nginx, Let's Encrypt |

---

## Quick Start (Local Development)

1. Clone the repository and copy environment variables:
   ```bash
   cp .env.example .env

2.  Fill in DATABASE_URL, JWT_SECRET, SUPABASE_URL, SUPABASE_ANON_KEY from your Supabase project.
3. Generate an encryption key for DSN encryption:
openssl rand -base64 32
and add it to .env as ENCRYPTION_KEY.
4. Build and start the stack:
docker compose up -d --build

5. Access http://localhost (or your domain if configured).
Note: The frontend uses build‑time NEXT_PUBLIC_* variables – you need to rebuild the frontend container if these change.

Deploy on a VPS (Production)

Detailed deployment instructions are in the original README, but the key steps are:

    1.Point your domain to the VPS IP.

    2.Install Docker and Docker Compose.

    3.Clone the repo, set up .env with your Supabase credentials and ENCRYPTION_KEY.

    4.Run the database migrations in your Supabase SQL editor.

    5.Build and start: docker compose up -d --build

6. Obtain SSL certificates via Let's Encrypt (standalone mode):
    docker compose stop nginx
docker run --rm -p 80:80 -p 443:443 \
  -v /etc/letsencrypt:/etc/letsencrypt \
  certbot/certbot certonly --standalone \
  -d yourdomain.com -d www.yourdomain.com \
  --email your@email.com --agree-tos
docker compose up -d nginx

7. Set up a cron job to auto‑renew certificates.

Using StickStock

    1.Sign up / login – email/password or Google/GitHub OAuth.

    2.Add a data source – go to Connections and enter the DSN (Postgres, MySQL, MongoDB, or REST). Test the connection before saving.

    3.Run queries – use the visual query builder or write raw SQL/JSON. Saved queries can be reused in dashboards.

    4.Create dashboards – add widgets, choose chart types, and arrange them.

    5.Share dashboards – generate a public link with an expiration date; the link can be rate‑limited.

    6.Schedule reports – set up cron expressions to send query results via email or Telegram.

    7.Admin panel – manage users, block/unblock, grant admin rights.


Key Endpoints (API)

Endpoint	                                                                Description

GET /api/health	                                                          Health check.
GET /api/me	                                                              Current user profile.
CRUD /api/datasources	                                                    Data source management.
POST /api/queries/run	                                                    Execute ad‑hoc query.
CRUD /api/queries	                                                        Saved queries with versioning.
CRUD /api/dashboards	                                                    Dashboards with widgets.
POST /api/dashboards/{id}/share	                                          Generate public share link.
GET /api/public/dashboards/{token}	                                      Public dashboard view.
POST /api/admin/users	                                                    Admin user management.
POST /api/reports	                                                        Scheduled reports.

Full OpenAPI documentation will be generated after we integrate swaggo/swag.


## Testing

Run all backend tests:

```bash
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.22 go test -v ./...

Or run specific package:
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.22 go test -v ./internal/analytics


Future Roadmap

    Semantic layer – define reusable datasets, metrics, and dimensions.

    Full integration of analytics‑service (ARIMA, regression, anomaly detection) into the frontend.

    Batch dashboard loading – one request instead of N widget queries.

    Export improvements – streaming export without the 1000‑row limit.



License
Commercial license – contact the author for purchase.


Contact

    Project URL: https://stickstock.lol

    Email: frulenhurram@gmail.com

    GitHub: https://github.com/kironovlaziz-del/stickstock


© 2026 StickStock. All rights reserved.

