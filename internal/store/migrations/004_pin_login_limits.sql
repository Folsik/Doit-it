CREATE TABLE IF NOT EXISTS pin_login_attempts (
    phone_e164 TEXT PRIMARY KEY,
    failed_attempts SMALLINT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
