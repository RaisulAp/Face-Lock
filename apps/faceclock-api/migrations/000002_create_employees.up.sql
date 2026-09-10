-- Function to automatically maintain updated_at across tables
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE employees (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_number citext NOT NULL,
    full_name text NOT NULL CHECK (length(btrim(full_name)) BETWEEN 2 AND 120),
    department text NULL,
    position text NULL,
    phone text NULL CHECK (phone IS NULL OR phone ~ '^[0-9+][0-9 +()-]{6,19}$'),
    email citext NULL,
    join_date date NULL,
    employment_status text NOT NULL DEFAULT 'active' CHECK (employment_status IN ('active','inactive','resigned')),
    attendance_mode text NOT NULL DEFAULT 'face' CHECK (attendance_mode IN ('face','manual')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE UNIQUE INDEX employees_employee_number_uniq
  ON employees (employee_number) WHERE deleted_at IS NULL;
CREATE INDEX employees_department_idx  ON employees (department) WHERE deleted_at IS NULL;
CREATE INDEX employees_full_name_idx   ON employees (lower(full_name)) WHERE deleted_at IS NULL;
CREATE INDEX employees_status_idx      ON employees (employment_status) WHERE deleted_at IS NULL;

CREATE TRIGGER employees_set_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
