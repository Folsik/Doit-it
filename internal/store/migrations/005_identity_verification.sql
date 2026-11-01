-- Identity verification (single KYC path; no tiered verification levels).
-- Do not store raw document images or biometric templates in this database.
CREATE TABLE IF NOT EXISTS identity_verification_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_session_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created'
      CHECK (status IN ('created','in_progress','pending_review','verified','rejected','expired','cancelled')),
    failure_reason_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE(provider, provider_session_id)
);
CREATE INDEX IF NOT EXISTS idx_identity_sessions_user_created
    ON identity_verification_sessions(user_id, created_at DESC);

-- A provider-verified person-level fingerprint, not a raw document-number hash.
-- This allows deduplication across national ID / passport / residence permit
-- when the KYC provider resolves them to the same natural person.
CREATE TABLE IF NOT EXISTS verified_identity_subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_subject_id TEXT,
    identity_fingerprint BYTEA NOT NULL UNIQUE,
    country_code CHAR(2) NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_subject_id)
);

-- Immutable provider event IDs prevent webhook replay from changing state twice.
CREATE TABLE IF NOT EXISTS identity_verification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    session_id UUID REFERENCES identity_verification_sessions(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_event_id)
);

-- Gate card selection / account-opening completion on identity verification.
