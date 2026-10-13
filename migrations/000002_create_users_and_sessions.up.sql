-- Staff accounts and server-side sessions (B-06, FR-001, SEC-006).
-- Roles, permissions and location scope are added by B-07; nothing here grants access.

CREATE TABLE users (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Stored normalised (lower case); login is case-insensitive.
    username            text        NOT NULL,
    display_name        text        NOT NULL,
    email               text,
    -- argon2id PHC string; never returned by the API.
    password_hash       text        NOT NULL,
    status              text        NOT NULL DEFAULT 'ACTIVE',
    last_login_at       timestamptz,
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    version             integer     NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9][a-z0-9._-]{2,63}$'),
    CONSTRAINT users_display_name_not_blank CHECK (length(btrim(display_name)) > 0 AND length(display_name) <= 120),
    CONSTRAINT users_email_format CHECK (email IS NULL OR (email ~ '^[^@[:space:]]+@[^@[:space:]]+$' AND length(email) <= 254)),
    CONSTRAINT users_password_hash_argon2id CHECK (password_hash LIKE '$argon2id$%'),
    CONSTRAINT users_status_valid CHECK (status IN ('ACTIVE', 'DISABLED')),
    CONSTRAINT users_version_positive CHECK (version > 0)
);

CREATE UNIQUE INDEX users_username_key ON users (username);
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email)) WHERE email IS NOT NULL;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_sessions (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- SHA-256 of the opaque cookie token; the token itself is never stored.
    token_hash     bytea       NOT NULL,
    -- Per-session CSRF token sent back in the X-CSRF-Token header. Useless without the cookie.
    csrf_token     text        NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    -- Absolute expiry; the idle timeout is applied from last_seen_at by the API.
    expires_at     timestamptz NOT NULL,
    revoked_at     timestamptz,
    revoked_reason text,
    ip_address     inet,
    user_agent     text,

    CONSTRAINT user_sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT user_sessions_csrf_token_length CHECK (length(csrf_token) >= 32),
    CONSTRAINT user_sessions_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT user_sessions_revocation_consistent
        CHECK ((revoked_at IS NULL) = (revoked_reason IS NULL)),
    CONSTRAINT user_sessions_revoked_reason_valid
        CHECK (revoked_reason IS NULL OR revoked_reason IN
            ('LOGOUT', 'SUPERSEDED', 'USER_DISABLED', 'PASSWORD_CHANGED', 'ADMIN_REVOKED')),
    CONSTRAINT user_sessions_user_agent_length CHECK (user_agent IS NULL OR length(user_agent) <= 256)
);

CREATE UNIQUE INDEX user_sessions_token_hash_key ON user_sessions (token_hash);
CREATE INDEX user_sessions_active_user_idx ON user_sessions (user_id) WHERE revoked_at IS NULL;
