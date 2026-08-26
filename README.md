# StickStock

**Self-hosted BI platform** – connect to any data source, query and visualize data, run advanced statistical analysis. Backend in Go, analytics in Python, deploy on your own VPS. No cloud lock-in – all data stays on your infrastructure.

---

## ✨ Key Features

- **Self-hosted** – full control over your data, no external dependencies except a Postgres database.
- **Multiple data sources** – Postgres, MySQL, MongoDB, REST APIs, and CSV uploads (via DuckDB).
- **Visual query builder** – build SQL queries without writing code.
- **Dashboards & widgets** – 10+ chart types (bar, line, pie, scatter, table, heatmap, boxplot, treemap, KPI, forecast) with drag‑and‑drop layout.
- **Advanced analytics** – linear regression, t‑tests, ARIMA forecasting, anomaly detection (Isolation Forest), and aggregations (Polars).
- **Enterprise‑grade security** – JWT authentication, encrypted credentials (AES‑256‑GCM), SSRF protection, rate limiting, CSP, SQL injection protection, and audit logging.
- **Collaboration** – share dashboards via public links (with expiration), invite team members (editor/viewer roles), and comment on widgets.
- **Scheduled reports** – send query results via email or Telegram using cron expressions.
- **Asynchronous analytics** – heavy computations run in the background, results are fetched later.
- **Lightweight** – runs on a 4 GB RAM / 2 CPU VPS.

---

## 🏗 Architecture
stickstock.lol ──▶ nginx (TLS termination, reverse proxy)
│
├── /api, /ws ──▶ Go backend (REST API, connectors, query engine)
│ │
│ └── REST ──▶ Python analytics service (FastAPI, Polars, DuckDB, statsmodels, PyOD)
│
└── / ──▶ Next.js frontend (React, Tailwind, Recharts)


- **backend/** – Go API, connectors, query engine, caching, dashboard/widget CRUD, authentication, audit.
- **analytics-service/** – Python/FastAPI with Polars, DuckDB, statsmodels, scipy, PyOD.
- **frontend/** – Next.js application (React, Tailwind, Recharts, react‑grid‑layout).
- **nginx/** – reverse proxy with SSL termination and rate limiting.

---

## 🔧 Technology Stack

| Component | Language / Libraries |
|-----------|----------------------|
| Backend   | Go 1.22, `pgx`, `jwt-go`, `crypto/aes`, `cron` |
| Analytics | Python 3.11, FastAPI, Polars, DuckDB, statsmodels, PyOD, scipy |
| Frontend  | Next.js 14, React, Tailwind, Recharts, react-grid-layout |
| Database  | PostgreSQL (Supabase or local), with cache table |
| Auth      | Custom JWT (username/password, bcrypt) |
| Deployment| Docker, Docker Compose, nginx, Let's Encrypt |

---

## 📦 Installation

### Prerequisites

- Linux VPS (Ubuntu 22.04 recommended)
- Docker and Docker Compose installed
- Domain name (optional, for HTTPS)
- Postgres database (Supabase free tier works)

### Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/stickstock.git
   cd stickstock
   2.Configure environment: cp .env.example .env
nano .env
Fill in:

    DATABASE_URL – your Postgres connection string.

    JWT_SECRET – generate with openssl rand -base64 32.

    ENCRYPTION_KEY – generate with openssl rand -base64 32.

    APP_URL – your public URL (e.g., https://stickstock.lol).

3.Run database migrations (SQL files in backend/migrations/) in your Postgres database.

4.Build and start containers: docker compose build
docker compose up -d

5.(Optional) Configure SSL with Let's Encrypt:
docker compose stop nginx
certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com
docker compose up -d nginx

6.Access the platform at https://yourdomain.com.

👤 Usage

    Sign up – create an account with username and password.

    Add connections – go to Connections, add your data sources (Postgres, MySQL, MongoDB, REST).

    Run queries – use the SQL editor or visual query builder.

    Create dashboards – add widgets, choose chart types, arrange them.

    Share – generate public links for dashboards.

    Schedule reports – set up cron jobs to receive query results by email or Telegram.

    Admin panel – manage users, view audit logs, see system stats.

    📘 Documentation

Full documentation is available in the docs/ folder:

    Installation Guide

    User Guide

    API Reference

    🔐 Security

    Authentication: JWT with HS256, stored in localStorage (can be moved to HttpOnly cookies if desired).

    Credentials: All DSNs are encrypted with AES‑256‑GCM.

    SQL Injection: Parameterized queries and read‑only validation.

    SSRF: REST connector validates target URLs and blocks private IPs.

    Rate Limiting: Login/register endpoints are limited to 5 requests per minute per IP.

    CSP: Strict Content‑Security‑Policy with strict-dynamic.

    Audit Logging: All user actions are logged in audit_log table.

    🧪 Testing

Run backend tests:
docker run --rm -v $(pwd)/backend:/app -w /app golang:1.22 go test -v ./...

🗄️ Backup

A universal backup script is provided in scripts/backup_all.sh. It detects the database type from DATABASE_URL and creates a compressed dump. Supports Postgres, MySQL, and MongoDB.

./scripts/backup_all.sh

Backups are stored in backups/ and rotated automatically (keep 7 days).


 Roadmap

    Semantic layer (reusable datasets and metrics).

    Full integration of analytics into dashboards.

    Batch dashboard loading (one request instead of N).

    Streaming export without row limits.

    OIDC/LDAP authentication.

    📄 License

Commercial license – contact the author for purchase.

📬 Contact

    Project URL: https://stickstock.lol

    Email: frulenhurram@gmail.com

    GitHub: https://github.com/kironovlaziz-del/stickstock

    

    © 2026 StickStock. All rights reserved.
