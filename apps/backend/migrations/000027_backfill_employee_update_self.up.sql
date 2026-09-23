-- 000027: backfill permissions for databases seeded before 000026
--
-- 000026 inserts employee.update_self. Databases created from scratch by the
-- seeder already receive it via internal/seeder/seeder.go, so this migration is
-- intentionally a no-op guard: it re-asserts the permission and the employee
-- role grant using the exact same idempotent statements.
--
-- It exists as a separate migration so that environments which already ran the
-- seeder with the old allowed-list still end up with the grant, without having
-- to re-run 000026 (which golang-migrate will not do).

INSERT INTO permissions (name, resource, action, description)
VALUES (
    'employee.update_self',
    'employee',
    'update_self',
    'Perbarui data profil karyawan sendiri (NIP, jabatan, tanggal masuk, telepon)'
)
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = 'employee.update_self'
WHERE r.name = 'employee'
ON CONFLICT (role_id, permission_id) DO NOTHING;
