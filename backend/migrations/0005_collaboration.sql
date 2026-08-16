-- Collaboration: sharing (public read-only links), owner/editor/viewer
-- roles, and comments on widgets. Plus basic freshness tracking on saved
-- queries for data-health visibility (stamped by QueryHandler.RunSaved).

ALTER TABLE dashboards ADD COLUMN IF NOT EXISTS share_token TEXT UNIQUE;

CREATE TABLE IF NOT EXISTS dashboard_collaborators (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    role         TEXT NOT NULL CHECK (role IN ('editor', 'viewer')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (dashboard_id, user_id)
);

CREATE TABLE IF NOT EXISTS widget_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    widget_id  UUID NOT NULL REFERENCES dashboard_widgets(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE saved_queries ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMPTZ;
ALTER TABLE saved_queries ADD COLUMN IF NOT EXISTS last_row_count INT;

ALTER TABLE dashboard_collaborators ENABLE ROW LEVEL SECURITY;
ALTER TABLE widget_comments ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS dashboard_collaborators_participant ON dashboard_collaborators;
CREATE POLICY dashboard_collaborators_participant ON dashboard_collaborators
    FOR SELECT USING (
        user_id = auth.uid()
        OR EXISTS (SELECT 1 FROM dashboards d WHERE d.id = dashboard_collaborators.dashboard_id AND d.owner_id = auth.uid())
    );

DROP POLICY IF EXISTS widget_comments_participant ON widget_comments;
CREATE POLICY widget_comments_participant ON widget_comments
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM dashboard_widgets dw
            JOIN dashboards d ON d.id = dw.dashboard_id
            WHERE dw.id = widget_comments.widget_id
            AND (
                d.owner_id = auth.uid()
                OR EXISTS (SELECT 1 FROM dashboard_collaborators dc WHERE dc.dashboard_id = d.id AND dc.user_id = auth.uid())
            )
        )
    );

CREATE INDEX IF NOT EXISTS idx_collaborators_dashboard ON dashboard_collaborators(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_user ON dashboard_collaborators(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_widget ON widget_comments(widget_id);
CREATE INDEX IF NOT EXISTS idx_dashboards_share_token ON dashboards(share_token) WHERE share_token IS NOT NULL;
