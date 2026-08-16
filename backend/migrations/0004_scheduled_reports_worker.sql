-- Tracks when each scheduled report last ran, so the worker (see
-- internal/scheduler) can compute whether its cron schedule is due
-- without needing its own separate state store.

ALTER TABLE scheduled_reports ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMPTZ;
