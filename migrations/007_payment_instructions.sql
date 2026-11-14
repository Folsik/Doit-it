-- External payment instructions are distinct from internal ledger transfers.
-- Provider adapters must not post internal ledger entries until the corresponding
-- funding/settlement model is implemented and reconciled.

CREATE TABLE IF NOT EXISTS payment_instructions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    source_account_id UUID NOT NULL REFERENCES accounts(id),
    rail TEXT NOT NULL CHECK (rail IN ('sepa','sepa_instant','local_bank','international_bank','swift','card')),
    direction TEXT NOT NULL DEFAULT 'outgoing' CHECK (direction IN ('outgoing','incoming')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL,
    fee_minor BIGINT NOT NULL DEFAULT 0 CHECK (fee_minor >= 0),
    recipient_currency CHAR(3) NOT NULL,
    recipient_amount_minor BIGINT CHECK (recipient_amount_minor > 0),
    fx_rate NUMERIC(20,10),
    status TEXT NOT NULL DEFAULT 'created' CHECK (status IN (
        'created','pending_submission','submitted','processing','completed',
        'rejected','failed','cancelled','returned'
    )),
    provider TEXT NOT NULL DEFAULT '',
    provider_reference TEXT,
    quote_id UUID,
    quote_expires_at TIMESTAMPTZ,
    idempotency_key TEXT NOT NULL,
    request_fingerprint BYTEA NOT NULL,
    recipient_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    failure_code TEXT,
    failure_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_payment_instructions_user_created
    ON payment_instructions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_instructions_provider_reference
    ON payment_instructions(provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_provider_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    instruction_id UUID REFERENCES payment_instructions(id),
    event_type TEXT NOT NULL,
    payload_fingerprint BYTEA NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    processing_error TEXT,
    UNIQUE (provider, provider_event_id)
);

CREATE TABLE IF NOT EXISTS payment_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID NOT NULL REFERENCES users(id),
    token_hash BYTEA NOT NULL UNIQUE,
    amount_minor BIGINT CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    description VARCHAR(140) NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','paid','expired','revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    paid_by_user_id UUID REFERENCES users(id),
    transfer_id UUID REFERENCES transfers(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_payment_links_creator_created
    ON payment_links(creator_user_id, created_at DESC);
