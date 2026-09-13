DROP TRIGGER IF EXISTS employees_set_updated_at ON employees;
DROP TABLE IF EXISTS employees CASCADE;
DROP FUNCTION IF EXISTS set_updated_at CASCADE;
