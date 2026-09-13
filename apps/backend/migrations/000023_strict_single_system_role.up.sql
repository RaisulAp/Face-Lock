-- Migration 000023: Strict single system role enforcement
-- 1. Remove all custom/non-system roles and associated role permissions
DELETE FROM roles WHERE is_system = false;

-- 2. Consolidate any multi-role users to single highest-priority role (super_admin > admin > employee)
DELETE FROM user_roles ur
WHERE ctid NOT IN (
    SELECT DISTINCT ON (user_id) ctid
    FROM user_roles
    JOIN roles r ON user_roles.role_id = r.id
    ORDER BY user_id,
        CASE r.name
            WHEN 'super_admin' THEN 1
            WHEN 'admin' THEN 2
            WHEN 'employee' THEN 3
            ELSE 4
        END
);

-- 3. Ensure all users have exactly 1 role (default to 'employee' if missing)
INSERT INTO user_roles (user_id, role_id, assigned_at)
SELECT u.id, (SELECT id FROM roles WHERE name = 'employee' LIMIT 1), NOW()
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id)
ON CONFLICT DO NOTHING;

-- 4. Enforce at most 1 role per user at database level (1 user cannot have 2 roles)
ALTER TABLE user_roles ADD CONSTRAINT user_roles_user_id_uniq UNIQUE (user_id);

-- 5. Prevent custom roles by enforcing is_system = true
ALTER TABLE roles ADD CONSTRAINT roles_only_system_roles CHECK (is_system = true);

-- 6. Trigger to prevent adding custom roles or deleting existing system roles
CREATE OR REPLACE FUNCTION prevent_roles_mutation()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.name NOT IN ('super_admin', 'admin', 'employee') OR NEW.is_system IS NOT TRUE THEN
            RAISE EXCEPTION 'Custom roles are not permitted. Only predefined system roles are allowed.';
        END IF;
        RETURN NEW;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'System roles cannot be deleted.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_roles_mutation ON roles;
CREATE TRIGGER trg_prevent_roles_mutation
BEFORE INSERT OR DELETE ON roles
FOR EACH ROW
EXECUTE FUNCTION prevent_roles_mutation();
