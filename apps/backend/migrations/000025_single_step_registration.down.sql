-- Rollback for 000025_single_step_registration
--
-- NOTE: restoring NOT NULL on employees.employee_number will fail if any row
-- still has a NULL employee_number (i.e. an employee registered through the
-- single-step flow who never completed their profile). Those rows must be
-- fixed before this down migration can succeed.

DROP INDEX IF EXISTS employees_profile_incomplete_idx;
DROP INDEX IF EXISTS employees_office_location_idx;

ALTER TABLE employees DROP COLUMN IF EXISTS profile_created_by_admin;
ALTER TABLE employees DROP COLUMN IF EXISTS profile_completed_at;

ALTER TABLE employees DROP COLUMN IF EXISTS office_location_id;

ALTER TABLE employees DROP CONSTRAINT IF EXISTS employees_employee_number_not_blank;

-- Fill NULLs before re-adding NOT NULL so the rollback does not fail on
-- in-flight registrations. Uses the same EMP-<year>-<seq> shape as the app.
UPDATE employees
SET employee_number = 'EMP-LEGACY-' || upper(substr(replace(id::text, '-', ''), 1, 10))
WHERE employee_number IS NULL;

ALTER TABLE employees
    ALTER COLUMN employee_number SET NOT NULL;
