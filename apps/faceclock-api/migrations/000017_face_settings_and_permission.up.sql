-- Migration 000017: Phase 3 Settings and Reindex Permission

INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
  ('face.min_reference_photos',        '3',    'number',  'Jumlah minimum foto referensi aktif per karyawan', true),
  ('face.max_reference_photos',        '5',    'number',  'Jumlah maksimum foto referensi aktif per karyawan', true),
  ('face.enrollment_session_ttl_minutes','30', 'number',  'Masa berlaku sesi enrollment (menit)',              true),
  ('face.min_quality_score',           '0.35', 'number',  'Skor kualitas minimum agar foto diterima sebagai referensi', false),
  ('face.duplicate_check_enabled',     'true', 'boolean', 'Periksa apakah wajah sudah terdaftar di karyawan lain', false),
  ('face.duplicate_threshold',         '0.50', 'number',  'Ambang similarity untuk deteksi wajah duplikat',    false),
  ('face.retention_days_after_resign', '365',  'number',  'Hari sebelum data wajah karyawan resign dihapus',   false),
  ('face.consent_required',            'true', 'boolean', 'Wajibkan consent biometrik sebelum enrollment',     true)
ON CONFLICT (key) DO UPDATE SET 
  value = EXCLUDED.value,
  description = EXCLUDED.description,
  is_public = EXCLUDED.is_public;

-- Permission BARU: face.reindex
INSERT INTO permissions (name, resource, action, description) VALUES
  ('face.reindex', 'face', 'reindex', 'Menjalankan job regenerasi embedding saat model berganti')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('super_admin', 'admin') AND p.name = 'face.reindex'
ON CONFLICT DO NOTHING;
