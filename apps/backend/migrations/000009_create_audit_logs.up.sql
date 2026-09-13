CREATE TABLE audit_logs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    action text NOT NULL,
    resource_type text NULL,
    resource_id text NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    ip inet NULL,
    user_agent text NULL,
    request_id text NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_actor_idx    ON audit_logs (actor_user_id, created_at DESC);
CREATE INDEX audit_logs_action_idx   ON audit_logs (action, created_at DESC);
CREATE INDEX audit_logs_resource_idx ON audit_logs (resource_type, resource_id, created_at DESC);
CREATE INDEX audit_logs_created_idx  ON audit_logs (created_at DESC);
