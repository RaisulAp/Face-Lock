DROP TRIGGER IF EXISTS trg_prevent_roles_mutation ON roles;
DROP FUNCTION IF EXISTS prevent_roles_mutation();
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_only_system_roles;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS user_roles_user_id_uniq;
