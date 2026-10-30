CREATE TABLE IF NOT EXISTS onboarding_tokens (
    token_hash TEXT PRIMARY KEY,
    phone_e164 TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_onboarding_tokens_expiry ON onboarding_tokens(expires_at);
