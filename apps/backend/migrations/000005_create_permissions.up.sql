CREATE TABLE permissions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE CHECK (name ~ '^[a-z_]+\.[a-z_]+$'),
    resource text NOT NULL,
    action text NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX permissions_resource_idx ON permissions (resource);

INSERT INTO permissions (name, resource, action, description) VALUES
    ('user.read', 'user', 'read', 'Lihat daftar & detail user'),
    ('user.create', 'user', 'create', 'Buat user'),
    ('user.update', 'user', 'update', 'Ubah user (termasuk aktif/nonaktif)'),
    ('user.delete', 'user', 'delete', 'Soft-delete user'),
    ('user.assign_role', 'user', 'assign_role', 'Set role milik user'),
    ('user.reset_password', 'user', 'reset_password', 'Reset password user lain'),
    ('employee.read', 'employee', 'read', 'Lihat semua karyawan'),
    ('employee.read_self', 'employee', 'read_self', 'Lihat data karyawan diri sendiri'),
    ('employee.create', 'employee', 'create', 'Tambah karyawan'),
    ('employee.update', 'employee', 'update', 'Ubah karyawan'),
    ('employee.delete', 'employee', 'delete', 'Soft-delete karyawan'),
    ('role.read', 'role', 'read', 'Lihat role'),
    ('role.create', 'role', 'create', 'Buat role'),
    ('role.update', 'role', 'update', 'Ubah role'),
    ('role.delete', 'role', 'delete', 'Hapus role'),
    ('role.assign_permission', 'role', 'assign_permission', 'Set permission milik role'),
    ('permission.read', 'permission', 'read', 'Lihat katalog permission'),
    ('settings.read', 'settings', 'read', 'Baca app settings'),
    ('settings.update', 'settings', 'update', 'Ubah app settings'),
    ('audit.read', 'audit', 'read', 'Baca audit log'),
    ('face.enroll_self', 'face', 'enroll_self', 'Enroll wajah sendiri'),
    ('face.read_self', 'face', 'read_self', 'Lihat referensi wajah sendiri'),
    ('face.enroll_any', 'face', 'enroll_any', 'Enroll wajah karyawan lain'),
    ('face.read_any', 'face', 'read_any', 'Lihat referensi wajah siapa pun'),
    ('face.delete_any', 'face', 'delete_any', 'Nonaktifkan referensi wajah siapa pun'),
    ('attendance.checkin', 'attendance', 'checkin', 'Melakukan check-in/out untuk diri sendiri'),
    ('attendance.read_self', 'attendance', 'read_self', 'Lihat riwayat absensi sendiri'),
    ('attendance.read_all', 'attendance', 'read_all', 'Lihat absensi semua karyawan'),
    ('attendance.approve', 'attendance', 'approve', 'Approve/reject absensi pending_review'),
    ('attendance.export', 'attendance', 'export', 'Export rekap'),
    ('location.read', 'location', 'read', 'Lihat office_locations'),
    ('location.create', 'location', 'create', 'Tambah office_locations'),
    ('location.update', 'location', 'update', 'Ubah office_locations'),
    ('location.delete', 'location', 'delete', 'Hapus office_locations')
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description;
