-- 000024_create_modules_and_relations.down.sql

DROP TRIGGER IF EXISTS trg_set_permission_module_id ON permissions;
DROP FUNCTION IF EXISTS set_permission_module_id();
DROP INDEX IF EXISTS permissions_module_id_idx;
ALTER TABLE permissions DROP COLUMN IF EXISTS module_id;
DROP TABLE IF EXISTS modules CASCADE;
