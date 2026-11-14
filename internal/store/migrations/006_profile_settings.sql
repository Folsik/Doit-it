-- Profile settings are user-scoped; no fixture data or external KYC approval.
CREATE TABLE user_profile_settings (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 address_line1 TEXT NOT NULL DEFAULT '',
 address_line2 TEXT NOT NULL DEFAULT '',
 city TEXT NOT NULL DEFAULT '',
 postal_code TEXT NOT NULL DEFAULT '',
 residence_country CHAR(2),
 notifications_email BOOLEAN NOT NULL DEFAULT TRUE,
 notifications_push BOOLEAN NOT NULL DEFAULT TRUE,
 notifications_security BOOLEAN NOT NULL DEFAULT TRUE,
 appearance TEXT NOT NULL DEFAULT 'system' CHECK (appearance IN ('system','light','dark')),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
