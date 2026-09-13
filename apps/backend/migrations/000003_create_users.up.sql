CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NULL REFERENCES employees(id) ON DELETE RESTRICT,
    email citext NOT NULL CHECK (email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$'),
    password_hash text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    must_change_password boolean NOT NULL DEFAULT false,
    token_version integer NOT NULL DEFAULT 0,
    failed_login_count smallint NOT NULL DEFAULT 0,
    locked_until timestamptz NULL,
    last_login_at timestamptz NULL,
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE UNIQUE INDEX users_email_uniq       ON users (email)       WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_employee_id_uniq ON users (employee_id) WHERE deleted_at IS NULL AND employee_id IS NOT NULL;
CREATE INDEX        users_is_active_idx    ON users (is_active)   WHERE deleted_at IS NULL;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
