CREATE TABLE IF NOT EXISTS analytics_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES profiles(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,          -- regression, forecast, anomalies, ttest
    status      TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    input       JSONB NOT NULL,         -- исходные данные запроса
    output      JSONB,                  -- результат (когда завершено)
    error       TEXT,                   -- сообщение об ошибке, если failed
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_analytics_tasks_user_id ON analytics_tasks(user_id);
CREATE INDEX idx_analytics_tasks_status ON analytics_tasks(status);
CREATE INDEX idx_analytics_tasks_created ON analytics_tasks(created_at DESC);
