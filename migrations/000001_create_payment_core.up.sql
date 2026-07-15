CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE terminal_status AS ENUM (
    'ACTIVE',
    'BLOCKED'
);

CREATE TYPE transaction_status AS ENUM (
    'CREATED',
    'AUTHORIZED',
    'CAPTURED',
    'REFUNDED',
    'DECLINED'
);

CREATE TYPE payment_method AS ENUM (
    'CREDIT_CARD',
    'DEBIT_CARD',
    'PIX'
);

CREATE TABLE merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE terminals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial TEXT NOT NULL UNIQUE,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    status terminal_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT terminals_id_merchant_unique UNIQUE (id, merchant_id)
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    merchant_id UUID NOT NULL REFERENCES merchants(id),
    terminal_id UUID NOT NULL,

    amount BIGINT NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL DEFAULT 'BRL'
        CHECK (currency ~ '^[A-Z]{3}$'),

    payment_method payment_method NOT NULL,
    status transaction_status NOT NULL DEFAULT 'CREATED',

    authorization_code TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT transactions_terminal_belongs_to_merchant
        FOREIGN KEY (terminal_id, merchant_id)
        REFERENCES terminals(id, merchant_id)
);

CREATE TABLE transaction_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    transaction_id UUID NOT NULL REFERENCES transactions(id),

    event TEXT NOT NULL CHECK (length(trim(event)) > 0),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    key TEXT NOT NULL,
    endpoint TEXT NOT NULL,

    response JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (key, endpoint)
);

CREATE INDEX idx_terminals_merchant_id
    ON terminals(merchant_id);

CREATE INDEX idx_transactions_merchant_created_at
    ON transactions(merchant_id, created_at DESC);

CREATE INDEX idx_transactions_terminal_created_at
    ON transactions(terminal_id, created_at DESC);

CREATE INDEX idx_transactions_status_created_at
    ON transactions(status, created_at DESC);

CREATE INDEX idx_transaction_events_transaction_created_at
    ON transaction_events(transaction_id, created_at ASC);

CREATE INDEX idx_idempotency_keys_created_at
    ON idempotency_keys(created_at);