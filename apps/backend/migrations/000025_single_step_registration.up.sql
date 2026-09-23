-- 000025: Single-step employee registration
--
-- Goals:
--  1. Let an employee row exist before HR completes the full profile
--     (employee_number may be NULL while the account is created, then the
--     employee fills in their real NIP from the portal).
--  2. Add the missing employees.office_location_id column. This fixes a
--     pre-existing bug: EmployeeFormPage.tsx has been sending
--     `office_location_id` since Fase 5 but the backend never persisted it,
--     so the admin's "Lokasi Kantor Utama" selection was silently dropped.
--  3. Track whether the employee has completed their own profile so admin
--     can filter "belum lengkap" in the employee list.

-- ---------------------------------------------------------------------------
-- 1. employee_number becomes optional
-- ---------------------------------------------------------------------------
ALTER TABLE employees
    ALTER COLUMN employee_number DROP NOT NULL;

-- The existing unique index already handles NULL correctly:
--   employees_employee_number_uniq ON (employee_number) WHERE deleted_at IS NULL
-- In PostgreSQL, NULLs are distinct in a unique index, so any number of
-- employees may have a NULL employee_number. No index change is needed.

-- Blank strings would defeat the "NULL means not yet filled" contract, so
-- normalise any existing blank values and block new ones.
UPDATE employees SET employee_number = NULL WHERE btrim(employee_number) = '';

ALTER TABLE employees
    ADD CONSTRAINT employees_employee_number_not_blank
    CHECK (employee_number IS NULL OR btrim(employee_number) <> '');

-- ---------------------------------------------------------------------------
-- 2. Primary office location (fixes the silently-dropped field)
-- ---------------------------------------------------------------------------
ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS office_location_id uuid
    REFERENCES office_locations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS employees_office_location_idx
    ON employees (office_location_id)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- 3. Profile completion tracking
-- ---------------------------------------------------------------------------
-- A profile counts as complete once the employee has supplied the fields HR
-- deliberately left to them: real NIP, position, and join date.
ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS profile_completed_at timestamptz NULL;

ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS profile_created_by_admin boolean NOT NULL DEFAULT false;

-- Backfill: everything that exists before this migration came from the old
-- two-step flow where an admin filled the full employee form, so treat it as
-- already complete.
UPDATE employees
SET profile_completed_at = COALESCE(profile_completed_at, created_at)
WHERE profile_completed_at IS NULL;

CREATE INDEX IF NOT EXISTS employees_profile_incomplete_idx
    ON employees (created_at DESC)
    WHERE deleted_at IS NULL AND profile_completed_at IS NULL;
