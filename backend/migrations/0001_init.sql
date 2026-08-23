-- Initial schema for stickstock's metadata store, meant to run against a
-- Supabase Postgres project (Supabase already provides the "auth" schema
-- and its "auth.users" table — this migration only adds the "public"
-- schema tables StickStock itself needs, plus a slim profile row per user).
--
-- Apply via the Supabase SQL Editor, or from anywhere with network access
-- to the project:
--   psql "$DATABASE_URL" -f backend/migrations/0001_init.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- One row per Supabase Auth user, for app-specific fields Supabase's own
-- auth.users doesn't carry (locale, and anything else added later).
-- Deliberately holds no password/credential data — Supabase Auth owns that.
CREATE TABLE IF NOT EXISTS profiles (
    id         UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    locale     TEXT NOT NULL DEFAULT 'en',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Auto-create a profile row whenever someone signs up through Supabase
-- Auth. This is the standard Supabase pattern for extending auth.users.
-- The frontend can set the initial locale by passing it as user metadata
-- on signUp(), e.g. supabase.auth.signUp({ email, password, options: {
-- data: { locale: 'ru' } } }).
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.profiles (id, locale)
    VALUES (NEW.id, COALESCE(NEW.raw_user_meta_data->>'locale', 'en'));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

CREATE TABLE IF NOT EXISTS data_sources (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL, -- postgres | mysql | mongodb | rest | file
    dsn        TEXT NOT NULL, -- TODO: encrypt at rest before production use
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS saved_queries (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id       UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    data_source_id UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    sql_text       TEXT NOT NULL,
    params         JSONB NOT NULL DEFAULT '{}'::jsonb,
    version        INT NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every edit to a saved query is archived here, giving Git-like history
CREATE TABLE IF NOT EXISTS saved_query_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saved_query_id  UUID NOT NULL REFERENCES saved_queries(id) ON DELETE CASCADE,
    version         INT NOT NULL,
    sql_text        TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dashboards (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    layout     JSONB NOT NULL DEFAULT '[]'::jsonb, -- react-grid-layout positions
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id    UUID NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    saved_query_id  UUID NOT NULL REFERENCES saved_queries(id) ON DELETE CASCADE,
    chart_type      TEXT NOT NULL, -- line | bar | pie | heatmap | table | boxplot | scatter | treemap
    config          JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- Scheduled automation: run a saved query on a cron schedule and deliver
CREATE TABLE IF NOT EXISTS scheduled_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    saved_query_id  UUID NOT NULL REFERENCES saved_queries(id) ON DELETE CASCADE,
    cron_expr       TEXT NOT NULL,
    delivery_kind   TEXT NOT NULL, -- email | telegram
    delivery_target TEXT NOT NULL, -- email address or telegram chat id
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_data_sources_owner ON data_sources(owner_id);
CREATE INDEX IF NOT EXISTS idx_saved_queries_owner ON saved_queries(owner_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_owner ON dashboards(owner_id);
CREATE INDEX IF NOT EXISTS idx_widgets_dashboard ON dashboard_widgets(dashboard_id);

-- Row Level Security: Supabase exposes Postgres directly to the browser
-- via PostgREST for some access patterns, and defense-in-depth is cheap
-- here even though today's Go backend is the only writer. Policies key
-- off auth.uid(), Supabase's function for "the calling user's id".
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE saved_queries ENABLE ROW LEVEL SECURITY;
ALTER TABLE saved_query_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboards ENABLE ROW LEVEL SECURITY;
ALTER TABLE dashboard_widgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE scheduled_reports ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS profiles_self ON profiles;
CREATE POLICY profiles_self ON profiles
    FOR ALL USING (id = auth.uid()) WITH CHECK (id = auth.uid());

DROP POLICY IF EXISTS data_sources_owner ON data_sources;
CREATE POLICY data_sources_owner ON data_sources
    FOR ALL USING (owner_id = auth.uid()) WITH CHECK (owner_id = auth.uid());

DROP POLICY IF EXISTS saved_queries_owner ON saved_queries;
CREATE POLICY saved_queries_owner ON saved_queries
    FOR ALL USING (owner_id = auth.uid()) WITH CHECK (owner_id = auth.uid());

DROP POLICY IF EXISTS saved_query_versions_owner ON saved_query_versions;
CREATE POLICY saved_query_versions_owner ON saved_query_versions
    FOR ALL USING (
        EXISTS (SELECT 1 FROM saved_queries q
                WHERE q.id = saved_query_versions.saved_query_id AND q.owner_id = auth.uid())
    );

DROP POLICY IF EXISTS dashboards_owner ON dashboards;
CREATE POLICY dashboards_owner ON dashboards
    FOR ALL USING (owner_id = auth.uid()) WITH CHECK (owner_id = auth.uid());

DROP POLICY IF EXISTS dashboard_widgets_owner ON dashboard_widgets;
CREATE POLICY dashboard_widgets_owner ON dashboard_widgets
    FOR ALL USING (
        EXISTS (SELECT 1 FROM dashboards d
                WHERE d.id = dashboard_widgets.dashboard_id AND d.owner_id = auth.uid())
    );

DROP POLICY IF EXISTS scheduled_reports_owner ON scheduled_reports;
CREATE POLICY scheduled_reports_owner ON scheduled_reports
    FOR ALL USING (owner_id = auth.uid()) WITH CHECK (owner_id = auth.uid());

-- Note: the Go backend currently connects with the Postgres service-role
-- connection string, which bypasses RLS entirely (by design — the backend
-- does its own ownership checks in Go, see internal/api/handlers). These
-- policies are what protect the data if/when anything ever queries
-- Supabase directly with a user's own JWT (e.g. a future frontend using
-- supabase-js against PostgREST instead of going through the Go API).
