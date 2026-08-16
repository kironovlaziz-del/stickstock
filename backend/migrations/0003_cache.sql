-- Simple key-value cache table, used instead of Redis (see docker-compose.yml
-- comment and README "Architecture" — one less service to run on a
-- 4GB RAM / 2 CPU VPS). Backed by backend/internal/cache.

CREATE TABLE IF NOT EXISTS cache (
    key        TEXT PRIMARY KEY,
    value      JSONB NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Partial index: only expiring rows need to be found quickly for cleanup;
-- permanent entries (expires_at IS NULL) don't need to be scanned for that.
CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at) WHERE expires_at IS NOT NULL;

ALTER TABLE cache ENABLE ROW LEVEL SECURITY;
-- No per-row owner concept here (cache keys are namespaced by convention,
-- e.g. "schema:<data_source_id>") — only the backend's service-role
-- connection touches this table, so no user-facing policy is defined.
