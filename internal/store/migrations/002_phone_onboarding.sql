ALTER TABLE users
    ALTER COLUMN email DROP NOT NULL,
    ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone_e164 TEXT,
    ADD COLUMN IF NOT EXISTS app_pin_hash TEXT,
    ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_e164
    ON users(phone_e164) WHERE phone_e164 IS NOT NULL;

CREATE TABLE IF NOT EXISTS phone_otp_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_e164 TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts SMALLINT NOT NULL DEFAULT 0 CHECK (attempts >= 0 AND attempts <= 5),
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_phone_otp_phone_created
    ON phone_otp_challenges(phone_e164, created_at DESC);

CREATE TABLE IF NOT EXISTS issued_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id),
    scheme TEXT NOT NULL CHECK (scheme IN ('visa','mastercard')),
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested','pending_provider','active','failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, scheme)
);
