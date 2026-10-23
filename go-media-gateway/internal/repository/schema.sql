CREATE TABLE IF NOT EXISTS asset (
    id            text PRIMARY KEY,
    tenant_id     text NOT NULL,
    content_hash  text NOT NULL,
    size_bytes    bigint NOT NULL CHECK (size_bytes > 0),
    mime          text NOT NULL,
    region        text NOT NULL,
    primary_node  text NOT NULL,
    replicas      text[] NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL,
    UNIQUE (tenant_id, content_hash)
);

CREATE TABLE IF NOT EXISTS usage_outbox (
    event_id     text PRIMARY KEY,
    tenant_id    text NOT NULL,
    asset_id     text NOT NULL,
    meter        text NOT NULL,
    quantity     bigint NOT NULL CHECK (quantity > 0),
    occurred_at  timestamptz NOT NULL,
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS ix_usage_outbox_unpublished
    ON usage_outbox (occurred_at, event_id)
    WHERE published_at IS NULL;
