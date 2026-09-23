-- 000026: employee.update_self
--
-- The single-step registration flow (000025) creates an employee whose HR
-- fields are deliberately blank, expecting the employee to complete them from
-- the portal. That flow had no permission to guard the new endpoint, so this
-- adds a narrowly scoped one.
--
-- Scope note: employee.update_self is intentionally distinct from
-- employee.update. The latter is admin-facing and covers fields an employee
-- must never change on themselves (employment_status, department, office
-- location). The self-service endpoint only accepts NIP, position, join date
-- and phone.

INSERT INTO permissions (name, resource, action, description)
VALUES (
    'employee.update_self',
    'employee',
    'update_self',
    'Perbarui data profil karyawan sendiri (NIP, jabatan, tanggal masuk, telepon)'
)
ON CONFLICT (name) DO NOTHING;

-- Grant it to the built-in employee role, matching how the other *_self
-- permissions are seeded (see internal/seeder/seeder.go).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = 'employee.update_self'
WHERE r.name = 'employee'
ON CONFLICT (role_id, permission_id) DO NOTHING;
