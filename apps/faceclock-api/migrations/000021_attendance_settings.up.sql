-- 20 Attendance & Geofencing Settings (Plan/05-Fase4.md § 2.8 + REV-SET-03/04/06/07)
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
    ('attendance_mode', '"face_with_fallback"'::jsonb, 'string', 'Mode absensi: face_strict, face_with_fallback, manual', true),
    ('max_distance_meter', '100'::jsonb, 'number', 'Radius geofence kantor (meter), default per kantor', true),
    ('geofence_fail_action', '"reject"'::jsonb, 'string', 'Aksi saat di luar geofence: reject atau pending_review', false),
    ('max_daily_fallback_per_employee', '1'::jsonb, 'number', 'Maksimal fallback per karyawan per hari', false),
    ('checkout_requires_face', 'true'::jsonb, 'boolean', 'Check-out memerlukan verifikasi wajah', true),
    ('min_checkout_interval_minutes', '5'::jsonb, 'number', 'Interval minimum antara check-in dan check-out', false),
    ('fallback_requires_review', 'true'::jsonb, 'boolean', 'Semua fallback manual harus di-review admin', false),
    ('anti_spoofing_strictness', '"medium"'::jsonb, 'string', 'Ketatan deteksi anti-spoofing: low, medium, high', false),
    ('work_day_cutoff_hour', '4'::jsonb, 'number', 'Jam pergantian hari kerja (04:00 AM)', false),
    ('attendance_timezone', '"Asia/Jakarta"'::jsonb, 'string', 'Zona waktu operasional absensi', true),
    ('max_failed_attempts_lockout', '5'::jsonb, 'number', 'Maksimal percobaan gagal sebelum cooldown', false),
    ('failed_attempts_window_minutes', '15'::jsonb, 'number', 'Window waktu percobaan gagal (menit)', false),
    ('photo_retention_days', '90'::jsonb, 'number', 'Hari retensi foto absensi sebelum di-purge', false),
    ('max_gps_accuracy_meter', '50'::jsonb, 'number', 'Akurasi GPS maksimal yang diterima (meter)', true),
    ('require_gps_accuracy', 'true'::jsonb, 'boolean', 'Wajibkan validasi akurasi GPS', true),
    ('liveness_mode', '"passive"'::jsonb, 'string', 'Mode liveness detection: disabled, passive, active_challenge', true),
    ('liveness_reject_action', '"allow_fallback"'::jsonb, 'string', 'Aksi saat liveness gagal: reject atau allow_fallback', false),
    ('liveness_score_threshold', '0.70'::jsonb, 'number', 'Threshold skor liveness pasif (0.0-1.0)', false),
    ('liveness_active_challenge_count', '2'::jsonb, 'number', 'Jumlah challenge aktif yang harus diselesaikan', false),
    ('location_allow_mocked', 'false'::jsonb, 'boolean', 'Izinkan absensi dengan fake GPS/mocked location', false)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    is_public = EXCLUDED.is_public;

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
