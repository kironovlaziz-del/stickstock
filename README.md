# StickStock

Self-hosted BI platform: connect to data sources, browse and query them,
build dashboards, run real statistical analysis, share and collaborate,
export reports, deliver them on a schedule, look up OSINT data. Backend
in Go, analytics service in Python, frontend in Next.js, deployed to your
own VPS at **stickstock.lol**.

No LLM in the loop — analytics runs on a concrete library stack (Polars,
DuckDB, statsmodels, PyOD) rather than an AI assistant, so there's no
external API key or per-query cost for that part.

## Architecture

```
                     ┌────────────┐
   stickstock.lol ─▶ │   nginx    │  (TLS termination, reverse proxy)
                     └─────┬──────┘
                    ┌──────┼───────┐
              /api,│/ws           │ everything else
                    ▼              ▼
             ┌────────────┐  ┌────────────┐        ┌────────────────────┐
             │  backend   │  │  frontend  │──REST──▶  analytics-service │
             │    (Go)    │  │ (Next.js)  │  (Go   │      (Python)      │
             └─────┬──────┘  └─────┬──────┘  calls)└────────────────────┘
                    │               │                 Polars · DuckDB ·
                    │ DATABASE_URL  │ supabase-js       statsmodels · PyOD
                    │ (Postgres,TLS)│ (auth only)        (all local, no keys)
                    └───────┬───────┘
                            ▼
                     ┌────────────┐
                     │  Supabase  │  managed Postgres + Auth (Google/GitHub/
                     │ (external) │  email OAuth), not a container on the VPS
                     └────────────┘

   cmd/worker: a separate Go binary in the same image as `backend`, NOT a
   running container — invoked by host cron (see deploy step 12) to run
   scheduled reports and sweep the cache table.
```

Metadata lives in **Supabase**, not a local Postgres container (see "Why
Supabase" below). Redis is gone too; a `cache` table in the same Supabase
Postgres instance covers it. Four containers on the VPS: `backend`,
`frontend`, `analytics-service`, `nginx` — comfortable on a 4GB RAM / 2 CPU
box, plus `cmd/worker` invoked on a schedule rather than running
continuously.

- **backend/** (Go) — the API: data source connectors, query execution,
  dashboards with sharing/roles/comments, scheduled reports, exports,
  OSINT lookups, admin. Verifies JWTs Supabase Auth issues; never handles
  passwords itself.
- **frontend/** (Next.js 14, App Router, TypeScript) — auth, connections,
  a SQL editor + visual query builder, drag-resize dashboards, OSINT
  lookup UI. Talks to Supabase directly (via `supabase-js`) only for
  auth; everything else goes through the Go API with the session JWT
  attached.
- **analytics-service/** (Python/FastAPI) — heavier data processing:
  DuckDB SQL over CSV files, Polars aggregation, statsmodels/scipy stats,
  PyOD anomaly detection. Internal only, no secrets needed. Not yet
  called from the Go backend or frontend — see "Analytics stack" gaps.
- **locales/** — shared UI/message translations: `en`, `ru`, `uz`, `kk`, `tg`
  (also copied into `frontend/public/locales`).

### Why Supabase

The spec gives two storage modes — cloud (Supabase) or fully self-hosted —
and recommends Supabase for the deployment target specifically. That's the
right call for a 4GB RAM / 2 CPU VPS: Postgres + Redis running locally
alongside Go, Python, and Next.js would be tight, and Supabase throws in
Google/GitHub OAuth for free, which a hand-rolled auth system doesn't. If
you'd rather run fully self-hosted later (Режим 2 from the spec), nothing
in the Go code cares where Postgres physically lives — `DATABASE_URL` just
needs to point at a reachable Postgres. The piece that'd need rework is
auth, since the frontend's login/signup pages and the Go backend's
`middleware.RequireAuth` both currently assume Supabase issued the
session JWT.

## Frontend

Next.js 14, built without network access to `npm install` or run a real
build — so unlike the Go/Python code, none of this has been through a
compiler or bundler yet. Written conservatively (well-established
Next.js/Supabase-SSR/Recharts/react-grid-layout APIs, no exotic patterns)
and every `.ts`/`.tsx` file passed a brace/paren balance check, but
**`npm run build` on the VPS is the first real test** — see deploy step 8.

What's there:
- **Auth**: `/login`, `/signup` — email/password plus Google/GitHub OAuth,
  via `supabase-js` directly. Session refresh + route protection via
  `src/middleware.ts`.
- **App shell**: sidebar (Dashboards / Queries / Connections / OSINT) +
  top bar (language switcher, logout). Dark theme per the design brief —
  glassmorphism, blue→purple→teal gradient accent, Inter (chosen for
  Cyrillic coverage across ru/uz/kk/tg) + JetBrains Mono for code. The KPI
  card's blurred-gradient-glow-behind-a-sparkline is the one signature
  visual move, kept deliberately singular.
- **Connections**: list + create (postgres/mysql/mongodb/rest; file
  sources come from Upload, not this form).
- **Queries**: ad-hoc SQL runner, saved-query detail page (edit/run/
  delete/version history/export), and `/queries/builder` — a no-code
  visual query builder (pick table → columns → filters → group-by →
  aggregations → order/limit) that generates SQL client-side and runs it
  through the same `/api/queries/run` endpoint the hand-written editor
  uses. Filter values ride as named `:params`, never inlined into the SQL
  text.
- **Dashboards**: list, create, a view page rendering widgets on a
  react-grid-layout grid (read-only), an edit page with the same grid in
  drag/resize mode (persists on drag/resize *stop*, not every intermediate
  move). Chart rendering via Recharts — line/bar/pie/scatter get real
  chart types, heatmap/boxplot/treemap fall back to a line chart.
- **OSINT**: `/osint` — WHOIS / DNS / breach-check lookup form.
- **i18n**: loads the same locale JSON as the backend, switcher persists
  to `localStorage`.

What's not wired up on the frontend even though the backend supports it:
dashboard sharing/collaborator management/comments (all have working
API endpoints — see "API reference" — just no UI yet), scheduled reports
UI, admin panel UI, lineage graph visualization, KPI cards (component
exists, nothing feeds it data), global dashboard filters, fullscreen
mode, dedicated heatmap/boxplot/treemap renders.

**Before auth works**, two things need doing in the Supabase dashboard
(not config-file changes):
1. **Auth → URL Configuration**: Site URL `https://stickstock.lol`,
   Redirect URLs including `https://stickstock.lol/auth/callback`.
2. **Auth → Providers**: enable Google/GitHub if you want those buttons
   to work (each needs a client ID/secret from that provider's own
   console — Supabase's provider setup page shows the callback URL to
   register there).

## Analytics stack

| Language | Library | Used for |
|---|---|---|
| Python | Polars | `POST /api/analytics/aggregate` — filter/group/aggregate |
| Python | DuckDB | `POST /api/analytics/query` — SQL directly over a CSV, no DB load |
| Python | statsmodels | `POST /api/analytics/stats/regression` and `/forecast` (ARIMA) |
| Python | scipy | `POST /api/analytics/stats/ttest` (Welch's t-test) |
| Python | PyOD | `POST /api/analytics/anomalies` — Isolation Forest outlier detection |
| Python | scikit-learn | pulled in as PyOD's dependency; no endpoint of its own |
| Go | stdlib (`internal/analytics`) | mean/median/stddev/correlation/percent-change, unused by any endpoint yet |

**Not wired up**: nothing in the Go backend or Next.js frontend calls
analytics-service yet — it's a fully working, independently deployed
service with no caller. Wiring it in (e.g. `POST /api/queries/{id}/aggregate`
that runs the query then forwards rows to Polars) is probably the single
highest-value remaining backend task. Two gaps against the spec's
analytics list: no simple moving-average endpoint (ARIMA subsumes it) and
no correlation-*matrix* endpoint (Go's `Correlation` helper is pairwise
only).

Gota / the Go "GopherData" DataFrame ecosystem was considered for Go-side
aggregation and never added — no Go toolchain or network access in the
environment this was written in to verify the exact API against a real
build. Check `go doc github.com/go-gota/gota/dataframe` before wiring it
in if still wanted.

## Full feature map against the spec

Every numbered section from the requirements doc, honestly scored.
✅ done · 🟡 partial · ❌ not started.

| # | Section | Status | Notes |
|---|---|---|---|
| 1 | Data source connections | 🟡 | Postgres, MySQL, MongoDB, REST, and CSV-upload-as-table all work. SQL Server/Oracle/SQLite/Cassandra/Redis/1С/Didox: not started. No AES-256 at rest (plaintext DSN — see Notes), no connection edit/delete, no workspace grouping, no usage history |
| 2 | Storage (cloud/self-hosted) | 🟡 | Cloud mode (Supabase) implemented and is the deployment target. Self-hosted is a `DATABASE_URL` swap away (see "Why Supabase") but untested; no mode-switch UI, no cloud↔self-hosted migration tool, no "export everything" |
| 3 | Working with data | ✅/🟡 | Query engine, saved queries + Git-like versioning, **visual no-code query builder** all done. SQL editor is a plain textarea (no syntax highlighting/autocomplete). No query tags/folders/templates |
| 4 | Visualization & dashboards | ✅/🟡 | Dashboard/widget CRUD, **drag-resize grid**, chart rendering all done. No scheduled *dashboard* refresh (scheduled *reports* — a different thing — do exist, see #10), no global filters, no fullscreen mode, KPI card component exists but nothing feeds it data yet |
| 5 | Analytics ("no hallucinations") | ✅/🟡 | Trend/anomaly/correlation/forecast all live in analytics-service — see its gaps above, and the bigger gap that nothing calls it yet |
| 6 | Reports & export | ✅/🟡 | **CSV/XLSX/PDF export** of query results, **scheduled email/Telegram delivery** via `cmd/worker`. No PNG chart export (needs client-side canvas rendering), no report branding/templates, no whole-dashboard export (only single-query) |
| 7 | Collaboration | ✅/🟡 | **Public share links**, **owner/editor/viewer roles**, **widget comments** all implemented backend-side; no frontend UI for any of them yet. No "Data Stories" narrative mode, no dashboard-level change history (queries have version history; dashboards don't) |
| 8 | Security & admin | ✅/🟡 | Supabase Auth + RLS + **admin block/unblock**. No per-data-source roles (only dashboard-level), no audit log, DSN still plaintext, rate/size limits are just the 1000-row query cap |
| 9 | Admin functions | 🟡 | User list + block/unblock + admin-toggle, coarse system stats (`/api/admin/stats`). No connection-health monitoring, no system log viewer, no tariff/limit configuration |
| 10 | Worker / background jobs | ✅ | `cmd/worker` — cron-invoked (not always-on), runs due scheduled reports (cron-expr parsing via robfig/cron) and sweeps expired cache rows. No currency-rate updates, no OSINT-on-a-schedule, no cloud↔self-hosted migration job |
| 11 | OSINT integration | 🟡 | **WHOIS, DNS, HaveIBeenPwned breach check** — all real, working lookups, plus a frontend page. No company/counterparty composite lookup, no "digital footprint" aggregate view, no entity relationship graph, no enrichment pipeline joining OSINT results into dashboards |
| 12 | Developer tools | 🟡 | The REST API itself is extensive (see "API reference") — that's the deliverable here. No OpenAPI/Swagger docs, no WebSocket, no webhooks, no CLI |
| 13 | Interface & UX | ✅/🟡 | Dark theme, i18n (5 locales), drag&drop (dashboard grid), interactive charts (tooltips, no drill-down) all there. No light theme (dark-only per brief), no smart cross-entity search, no hotkeys |
| 14 | Extras | 🟡/❌ | Telegram is wired as a *report delivery channel* (not a full bot interface). Multi-tenancy exists in the loose sense that Supabase + RLS + per-user ownership already isolates users from each other, but there's no org/white-label layer on top. Real-time streaming, on-premise packaging: not started. **Plugin marketplace: deliberately not built** — see below |
| — | Custom scripts (user-defined transformations, "jq-like") | ❌ | Explicitly deferred — the platform is read-only by design (`queryengine.ValidateReadOnly` blocks every non-SELECT statement across every SQL-shaped connector). Adding either read-only transform scripts or real write/DDL access is a real architectural decision with a different risk profile each way; revisit deliberately, don't bolt it on |

### On the plugin marketplace specifically

Not built, and not stubbed with fake tables either — a shallow "plugins"
schema with no real execution engine behind it would look done without
being done. A real v1 needs the *plugin model* decided first (custom
connectors? custom chart types? server-side data transforms? UI panels?),
since that decision drives the architecture (sandboxing needs, a
permission model, how a plugin gets installed/versioned) far more than
"add a plugins table" would suggest. Worth a dedicated design pass if
still wanted, not something to retrofit.

## API reference

All routes below are under the Go backend (nginx proxies `/api/` to it).
Everything except `/api/health` and `/api/public/*` requires
`Authorization: Bearer <supabase-jwt>`. `/api/admin/*` additionally
requires `profiles.is_admin = true`.

**Profile**
`GET /api/me` · `PUT /api/me`

**Data sources**
`GET|POST /api/datasources` · `POST /api/datasources/upload` (CSV) ·
`GET /api/datasources/{id}/schema` (table/column list, for the query builder)

**Queries**
`POST /api/queries/run` (ad-hoc) · `GET|POST /api/queries` ·
`GET|PUT|DELETE /api/queries/{id}` · `GET /api/queries/{id}/versions` ·
`POST /api/queries/{id}/run` · `GET /api/queries/{id}/export?format=csv|xlsx|pdf` ·
`POST /api/export/query` (ad-hoc export)

**Dashboards**
`GET|POST /api/dashboards` · `GET|PUT|DELETE /api/dashboards/{id}` ·
`POST /api/dashboards/{id}/widgets` ·
`PUT|DELETE /api/dashboards/{id}/widgets/{widget_id}` ·
`POST|DELETE /api/dashboards/{id}/share` (create/revoke a public link) ·
`GET|POST /api/dashboards/{id}/collaborators` ·
`DELETE /api/dashboards/{id}/collaborators/{user_id}` ·
`GET|POST /api/dashboards/{id}/widgets/{widget_id}/comments` ·
`DELETE /api/comments/{comment_id}`

**Public** (no auth — the share token itself is the credential)
`GET /api/public/dashboards/{token}` ·
`POST /api/public/dashboards/{token}/widgets/{widget_id}/run`

**Scheduled reports**
`GET|POST /api/reports` · `PUT|DELETE /api/reports/{id}`

**Lineage**
`GET /api/lineage` — graph of data_source → saved_query → dashboard for
the caller's own data

**Admin**
`GET /api/admin/users` · `PUT /api/admin/users/{id}` (`is_admin`/`is_blocked`) ·
`GET /api/admin/stats`

**OSINT**
`POST /api/osint/lookup` — `{"type": "whois"|"dns"|"breach", "target": "..."}`

**analytics-service** (separate Python service, internal-only — not
proxied through nginx or called by the Go backend yet)
`POST /api/analytics/query` · `POST /api/analytics/aggregate` ·
`POST /api/analytics/stats/regression` · `POST /api/analytics/stats/ttest` ·
`POST /api/analytics/stats/forecast` · `POST /api/analytics/anomalies`

## Local development

```bash
cp .env.example .env   # fill in DATABASE_URL, JWT_SECRET, SUPABASE_URL, SUPABASE_ANON_KEY
docker run --rm -v "$(pwd)/backend:/src" -w /src golang:1.22-alpine go mod tidy  # generates go.sum
docker compose up --build
curl http://localhost/api/health
curl http://localhost:8000/health   # analytics-service, if its port is exposed for local testing
open http://localhost                # frontend, via nginx
```

Apply all six migrations (see below) to your Supabase project before any
of this does anything useful — `docker compose up` doesn't touch the
database. Promote yourself to admin manually once you've signed up once
(see deploy step 6).

## Deploying to your VPS over SSH, step by step

Assumes a fresh Ubuntu/Debian VPS, `stickstock.lol` already pointed at it.

**1. Create the Supabase project**: at [supabase.com](https://supabase.com),
new project → note the DB password. From **Settings → Database** copy the
connection string (service_role/direct, not anon). From **Settings → API**
copy the Project URL, anon public key, and JWT Secret.

Also now: **Authentication → URL Configuration** — Site URL
`https://stickstock.lol`, Redirect URLs including
`https://stickstock.lol/auth/callback`. **Authentication → Providers** —
enable Google/GitHub if you want those login buttons to work.

**2. Point DNS**: A records for `stickstock.lol` and `www.stickstock.lol`
→ your VPS's IP. Confirm with `dig stickstock.lol` before continuing.

**3. Connect and do basic setup**:
```bash
ssh root@YOUR_VPS_IP
apt update && apt upgrade -y
adduser deploy
usermod -aG sudo deploy
rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy
# open a SECOND terminal and confirm this works before closing root:
ssh deploy@YOUR_VPS_IP
```
On the VPS as `deploy`:
```bash
sudo apt install ufw -y
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
docker --version && docker compose version
```

**4. Get the code onto the VPS.** From your local machine:
```bash
scp stickstock.tar.gz deploy@YOUR_VPS_IP:~/
```
On the VPS:
```bash
tar -xzf stickstock.tar.gz
cd stickstock
```

**5. Configure secrets**:
```bash
cp .env.example .env
nano .env
```
Fill in `DATABASE_URL`, `JWT_SECRET`, `SUPABASE_URL`, `SUPABASE_ANON_KEY`
from step 1, set `ALLOWED_ORIGINS=https://stickstock.lol`. SMTP/Telegram/
HIBP vars are optional — leave blank to skip those features, nothing
else depends on them.

**6. Apply the database schema** — SQL Editor in the Supabase dashboard,
or from the VPS:
```bash
for f in backend/migrations/*.sql; do psql "$DATABASE_URL" -f "$f"; done
```
(No `psql`? `sudo apt install postgresql-client -y` first.) All six files
matter now: `0001` core schema, `0002` file uploads, `0003` cache,
`0004` scheduled-reports freshness column, `0005` collaboration
(sharing/roles/comments), `0006` admin flags.

**After signing up once through the frontend** (do this after step 8, once
the app is actually reachable), promote yourself to admin:
```sql
UPDATE profiles SET is_admin = true WHERE id = (SELECT id FROM auth.users WHERE email = 'you@example.com');
```

**7. Generate `go.sum`** (once, via a throwaway container — no Go install needed):
```bash
docker run --rm -v "$(pwd)/backend:/src" -w /src golang:1.22-alpine go mod tidy
```

**8. First boot, HTTP only** (certbot needs a plain-HTTP challenge first).
Edit `nginx/stickstock.conf`: comment out the `server { listen 443 ... }`
block and the `return 301 https://...` line in the `:80` block — comments
in the file mark exactly what to remove. Then:
```bash
docker compose up -d --build
docker compose ps
docker compose logs -f backend    # look for "listening on :8080"
docker compose logs -f frontend   # this is the untested one — check it built
```
If `frontend` fails, it's almost certainly a TypeScript/build error (see
"Frontend" above) — `docker compose build frontend` re-runs just that
build with full output.

**9. Get the SSL certificate**:
```bash
docker volume ls | grep certbot   # confirm actual names, usually prefixed
                                   # "stickstock_certbot_www" etc.
docker run --rm \
  -v stickstock_certbot_www:/var/www/certbot \
  -v stickstock_certbot_certs:/etc/letsencrypt \
  certbot/certbot certonly --webroot -w /var/www/certbot \
  -d stickstock.lol -d www.stickstock.lol \
  --email you@example.com --agree-tos --no-eff-email
```

**10. Enable HTTPS**: restore the two commented blocks in
`nginx/stickstock.conf`, then:
```bash
docker compose restart nginx
curl -I https://stickstock.lol/api/health   # expect HTTP/2 200
```

**11. Auto-renew the certificate**:
```bash
crontab -e
```
Add:
```
0 3 * * 1 docker run --rm -v stickstock_certbot_www:/var/www/certbot -v stickstock_certbot_certs:/etc/letsencrypt certbot/certbot renew --webroot -w /var/www/certbot --quiet && docker compose -f /home/deploy/stickstock/docker-compose.yml restart nginx
```

**12. Schedule the worker** (scheduled reports + cache cleanup — see
"Architecture": this is a one-shot binary in the `backend` image, not a
running container):
```bash
crontab -e
```
Add (runs every 5 minutes):
```
*/5 * * * * cd /home/deploy/stickstock && docker compose run --rm backend /app/worker >> /var/log/stickstock-worker.log 2>&1
```

**13. Ongoing operations**:
```bash
docker compose logs -f backend           # or analytics-service, frontend, nginx
docker compose ps
docker compose restart
docker compose up -d --build             # redeploy after new code
docker compose run --rm backend /app/worker   # run the worker manually, e.g. to test

# Supabase handles Postgres backups itself (project's Backups tab) — no
# local pg_dump step since there's no local Postgres.
```

## Database migrations

No local Postgres container auto-applies these (see "Why Supabase") —
apply `backend/migrations/*.sql` to your Supabase project in filename
order, via the SQL Editor or `psql`. All six matter (see deploy step 6).
Consider `golang-migrate` once tracking six-plus files by hand gets
tedious.

## Notes / things to revisit before production

- `data_sources.dsn` is stored as plaintext — encrypt it (AES-GCM, key
  from a secrets manager) before storing real credentials. The spec
  specifically asks for AES-256 here (section 1); not done.
- `queryengine.ValidateReadOnly` is a keyword blacklist, which can
  false-positive on a string literal containing a word like "created."
  The robust fix is a read-only DB role per data source — do that before
  letting untrusted users run ad-hoc SQL.
- The Go backend connects to Supabase with the service-role connection,
  bypassing Row Level Security by design (ownership/role checks happen in
  Go). The RLS policies across the migrations are what protect the data
  if anything ever queries Supabase directly with a user's own JWT (e.g.
  a future frontend calling PostgREST) — keep them in sync with the
  Go-side checks if either changes.
- Go (`go.mod`) and frontend (`package.json`) dependency versions are
  best-guess pins written without network access to verify or generate
  lock files. `go mod tidy` (step 7) and `npm install` (inside the
  frontend Docker build) resolve them for real on first deploy — that's
  the actual first compile/build for both, budget time accordingly.
- File data sources (CSV upload) share the app's own Postgres connection
  rather than a separate credential — `queryengine.ValidateFileScope`
  stops a query from reaching `public.*` tables by requiring it reference
  its own randomly-named `uploads.t_xxx` table. Substring check, not a
  parser — not airtight, same trade-off as the keyword blacklist above.
- REST/MongoDB data sources take a JSON DSL as their "query", not SQL —
  `queryengine.ValidateReadOnly` doesn't apply to them (see
  `connectors.IsSQLKind`), and `:param` binding only works for SQL-shaped
  sources currently.
- MySQL's driver wants `?` placeholders, not Postgres's `$1,$2` —
  `connectors.toMySQLArgs` rewrites them post-hoc. Fine for two SQL
  dialects; revisit if a third shows up.
- A dashboard's public share link (`share_token`) is a 24-byte random
  bearer credential — anyone holding the URL can view the dashboard *and*
  trigger live query execution against the owner's real data source
  (server-side; the DSN itself never reaches the client). Revoking is
  instant (`DELETE .../share`), but there's no link expiry or
  view-count/access-log yet.
- `RunSaved` (running a saved query) allows either the query's owner or
  anyone with dashboard-collaborator access to a dashboard that uses it
  via a widget — this is what makes shared dashboards show live data to
  viewers who don't own the underlying query. `Get`/`Update`/`Delete`/
  `Versions` on saved queries stayed owner-only; a collaborator can see
  chart *output* but not the underlying SQL.
- Nobody is `is_admin` by default — the first admin has to be promoted
  manually via SQL (deploy step 6), there's no bootstrap flow.
- OSINT's WHOIS/DNS lookups accept any target from an authenticated user
  with no allowlist/rate-limit beyond requiring auth — fine for a small
  trusted team, worth adding limits before opening this to a larger or
  less-trusted user base.
- Custom scripting (user-defined data transformations) is deliberately
  not implemented — see the feature map. If revisited, treat "read-only
  transform after a query" and "real write access to the user's DB" as
  two different features with two different risk profiles, not one.
