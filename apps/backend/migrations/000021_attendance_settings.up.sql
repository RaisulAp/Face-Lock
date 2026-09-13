-- 20 Attendance & Geofencing Settings (Plan/05-Fase4.md § 3.5 + REV-SET-03/04/06/07)
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
    ('attendance.workday_cutoff_hour', '0'::jsonb, 'number', 'Jam pemisah hari kerja (0-11); >0 untuk shift malam', true),
    ('attendance.fallback_enabled', 'true'::jsonb, 'boolean', 'Izinkan jalur fallback saat verifikasi wajah gagal', true),
    ('attendance.outside_geofence_policy', '"reject"'::jsonb, 'string', 'Perlakuan absensi di luar radius: reject | pending_review', true),
    ('attendance.missing_location_policy', '"reject"'::jsonb, 'string', 'Perlakuan saat lokasi tidak tersedia/tidak akurat: reject | pending_review', true),
    ('attendance.max_gps_accuracy_meter', '100'::jsonb, 'number', 'Akurasi GPS terburuk yang masih diterima (meter)', true),
    ('attendance.allow_checkout_without_checkin', 'false'::jsonb, 'boolean', 'Izinkan check-out tanpa check-in di hari yang sama', true),
    ('attendance.min_minutes_between_checkin_checkout', '1'::jsonb, 'number', 'Jeda minimum check-in ke check-out (menit)', true),
    ('attendance.require_face_for_checkout', 'true'::jsonb, 'boolean', 'Wajibkan verifikasi wajah saat check-out', true),
    ('attendance.checkout_without_face_status', '"pending_review"'::jsonb, 'string', 'Status check-out saat verifikasi wajah dimatikan', false),
    ('attendance.allow_fallback_without_enrollment', 'false'::jsonb, 'boolean', 'Izinkan absen fallback bagi karyawan yang belum enroll', false),
    ('attendance.max_note_length', '500'::jsonb, 'number', 'Panjang maksimum catatan absensi', true),
    ('attendance.max_failed_attempts_per_hour', '10'::jsonb, 'number', 'Batas percobaan gagal per karyawan per jam', false),
    ('attendance.photo_retention_days', '365'::jsonb, 'number', 'Hari sebelum foto absensi dihapus', false),
    ('attendance.attempt_retention_days', '90'::jsonb, 'number', 'Hari sebelum telemetri percobaan dihapus', false),
    ('attendance.duplicate_photo_window_days', '7'::jsonb, 'number', 'Rentang deteksi foto identik (anti-replay)', false),
    ('attendance.liveness_policy', '"preferred"'::jsonb, 'string', 'Kebijakan liveness: off | preferred | required', true),
    ('attendance.liveness_max_attempts', '3'::jsonb, 'number', 'Maksimal percobaan liveness sebelum ditolak', false),
    ('attendance.liveness_challenge_count', '2'::jsonb, 'number', 'Jumlah aksi tantangan liveness aktif', false),
    ('attendance.liveness_timeout_seconds', '20'::jsonb, 'number', 'Batas waktu tantangan liveness (detik)', false),
    ('attendance.mocked_location_policy', '"reject"'::jsonb, 'string', 'Perlakuan deteksi fake GPS / mocked location: reject | pending_review', false)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    is_public = EXCLUDED.is_public;

-- Rekonsiliasi makna (REV-SET-04)
UPDATE app_settings
SET description = 'Batas atas dan nilai default radius_meter saat membuat office_location baru. Radius efektif ditentukan per lokasi.'
WHERE key = 'attendance.max_distance_meter';

-- Permissions from Plan/05-Fase4.md § 2.5
INSERT INTO permissions (name, resource, action, description) VALUES
    ('attendance.create', 'attendance', 'create', 'Melakukan check-in/out untuk diri sendiri'),
    ('attendance.review', 'attendance', 'review', 'Approve/reject absensi pending_review'),
    ('attendance.read_team', 'attendance', 'read_team', 'Lihat absensi anggota departemen/tim'),
    ('location.manage', 'location', 'manage', 'Buat, ubah, hapus lokasi kantor')
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description;

-- Grant permissions to roles
-- super_admin gets ALL
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- admin gets attendance.create, attendance.review, attendance.read_team, location.manage
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.name IN ('attendance.create', 'attendance.review', 'attendance.read_team', 'location.manage')
ON CONFLICT DO NOTHING;

-- employee gets attendance.create, location.read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'employee'
  AND p.name IN ('attendance.create', 'location.read')
ON CONFLICT DO NOTHING;
