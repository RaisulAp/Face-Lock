CREATE TABLE app_settings (
    key text PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    value jsonb NOT NULL,
    value_type text NOT NULL CHECK (value_type IN ('string','number','boolean','json')),
    description text NOT NULL,
    is_public boolean NOT NULL DEFAULT false,
    updated_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX app_settings_is_public_idx ON app_settings (is_public);

INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
    ('face.similarity_threshold', '0.45', 'number', 'Threshold kemiripan vektor embedding wajah (Cosine Similarity). Nilai awal konservatif, menunggu kalibrasi Fase 2.', false),
    ('face.min_reference_photos', '3', 'number', 'Jumlah foto referensi minimum untuk pendaftaran biometrik.', true),
    ('face.model_version', '"unset"', 'string', 'Versi model inferensi wajah yang aktif. Bernilai "unset" hingga kalibrasi Fase 2.', false),
    ('attendance.geofence_enabled', 'true', 'boolean', 'Aktifkan validasi perimeter geofence saat presensi.', true),
    ('attendance.max_distance_meter', '100', 'number', 'Batas atas dan nilai default radius geofence lokasi kantor dalam meter.', true),
    ('attendance.workday_start', '"08:00"', 'string', 'Jam mulai kerja standar (HH:mm).', true),
    ('attendance.workday_end', '"17:00"', 'string', 'Jam selesai kerja standar (HH:mm).', true),
    ('attendance.timezone', '"Asia/Jakarta"', 'string', 'Zona waktu standar operasional.', true),
    ('security.password_min_length', '10', 'number', 'Panjang minimum karakter kata sandi pengguna.', true),
    ('security.max_failed_login', '5', 'number', 'Batas percobaan login gagal sebelum akun dikunci sementara.', false),
    ('security.lockout_minutes', '15', 'number', 'Durasi akun terkunci setelah mencapai batas login gagal (menit).', false),
    ('auth.refresh_reuse_grace_seconds', '30', 'number', 'Toleransi window deteksi reuse token saat terjadi network race condition (detik).', false)
ON CONFLICT (key) DO NOTHING;
