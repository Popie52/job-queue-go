CREATE TABLE IF NOT EXISTS pending_jobs (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    priority INT NOT NULL,
    payload JSONB,
    attempts INT NOT NULL,
    max_retries INT NOT NULL
);

CREATE TABLE IF NOT EXISTS inflight_jobs (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    priority INT NOT NULL,
    payload JSONB,
    attempts INT NOT NULL,
    max_retries INT NOT NULL,
    picked_at TIMESTAMPTZ NOT NULL
);
