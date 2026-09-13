CREATE TABLE role_permissions (
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id uuid NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX role_permissions_permission_id_idx ON role_permissions (permission_id);

-- super_admin gets ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- admin gets all EXCEPT role.create, role.update, role.delete, role.assign_permission, user.delete
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.name NOT IN ('role.create', 'role.update', 'role.delete', 'role.assign_permission', 'user.delete')
ON CONFLICT DO NOTHING;

-- employee gets employee.read_self, face.enroll_self, face.read_self, attendance.checkin, attendance.read_self, settings.read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'employee'
  AND p.name IN ('employee.read_self', 'face.enroll_self', 'face.read_self', 'attendance.checkin', 'attendance.read_self', 'settings.read')
ON CONFLICT DO NOTHING;
