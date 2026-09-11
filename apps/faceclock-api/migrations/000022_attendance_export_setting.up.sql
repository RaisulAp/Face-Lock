-- Migration 000022: Attendance export max rows setting (Fase 5 - Plan/06-Fase5.md § 2.1 & § 12)
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
    ('attendance.export_max_rows', '100000'::jsonb, 'number', 'Batas maksimal baris ekspor data absensi', false)
ON CONFLICT (key) DO NOTHING;
