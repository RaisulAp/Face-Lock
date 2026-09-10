CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    family_id uuid NOT NULL,
    parent_id uuid NULL REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    is_revoked boolean NOT NULL DEFAULT false,
    revoked_reason text NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    ip_address inet NULL,
    user_agent text NULL
);

CREATE INDEX refresh_tokens_lookup_idx ON refresh_tokens (token_hash, is_revoked, expires_at);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_user_idx   ON refresh_tokens (user_id);
