CREATE TABLE IF NOT EXISTS transfer_idempotency (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(100) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    transfer_id UUID REFERENCES transfers(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_transfer_idempotency_created_at
    ON transfer_idempotency(created_at);
