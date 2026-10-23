-- Opening entry is balanced: debit PROMO_EXPENSE 50000000 equals credit PREPAID 50000000.
-- 50000000 micros is 50.000000 USD of demo prepaid credit.

CREATE TABLE account (
    id uuid PRIMARY KEY,
    tenant_id varchar(64) NOT NULL,
    code varchar(32) NOT NULL,
    kind varchar(16) NOT NULL,
    balance_micros bigint NOT NULL,
    version bigint NOT NULL,
    CONSTRAINT uq_account_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT ck_account_balance_nonneg CHECK (balance_micros >= 0),
    CONSTRAINT ck_account_kind CHECK (kind IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE'))
);

CREATE INDEX ix_account_tenant ON account (tenant_id);

CREATE TABLE journal_entry (
    id uuid PRIMARY KEY,
    event_id varchar(160) NOT NULL,
    memo varchar(128) NOT NULL,
    occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT uq_journal_event UNIQUE (event_id)
);

CREATE TABLE ledger_line (
    id uuid PRIMARY KEY,
    entry_id uuid NOT NULL REFERENCES journal_entry (id),
    account_id uuid NOT NULL REFERENCES account (id),
    side varchar(6) NOT NULL,
    amount_micros bigint NOT NULL,
    CONSTRAINT ck_line_side CHECK (side IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ck_line_amount_pos CHECK (amount_micros > 0)
);

CREATE INDEX ix_ledger_line_entry ON ledger_line (entry_id);
CREATE INDEX ix_ledger_line_account ON ledger_line (account_id);

CREATE TABLE processed_usage (
    event_id varchar(160) PRIMARY KEY,
    journal_id uuid NOT NULL REFERENCES journal_entry (id),
    processed_at timestamptz NOT NULL
);

INSERT INTO account (id, tenant_id, code, kind, balance_micros, version) VALUES
    ('11111111-1111-4111-8111-111111111111', 'platform', 'PROMO_EXPENSE', 'EXPENSE', 50000000, 0),
    ('22222222-2222-4222-8222-222222222222', 'platform', 'REV_STORAGE', 'REVENUE', 0, 0),
    ('33333333-3333-4333-8333-333333333333', 'platform', 'REV_EGRESS', 'REVENUE', 0, 0),
    ('44444444-4444-4444-8444-444444444444', 'platform', 'REV_INFERENCE', 'REVENUE', 0, 0),
    ('55555555-5555-4555-8555-555555555555', 'demo', 'PREPAID', 'LIABILITY', 50000000, 0);

INSERT INTO journal_entry (id, event_id, memo, occurred_at, created_at) VALUES
    ('66666666-6666-4666-8666-666666666666', 'opening:demo:prepaid', 'opening balance', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');

INSERT INTO ledger_line (id, entry_id, account_id, side, amount_micros) VALUES
    ('77777777-7777-4777-8777-777777777777', '66666666-6666-4666-8666-666666666666', '11111111-1111-4111-8111-111111111111', 'DEBIT', 50000000),
    ('88888888-8888-4888-8888-888888888888', '66666666-6666-4666-8666-666666666666', '55555555-5555-4555-8555-555555555555', 'CREDIT', 50000000);

INSERT INTO processed_usage (event_id, journal_id, processed_at) VALUES
    ('opening:demo:prepaid', '66666666-6666-4666-8666-666666666666', '2026-01-01T00:00:00Z');
