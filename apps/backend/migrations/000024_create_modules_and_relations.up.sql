-- 000024_create_modules_and_relations.up.sql

CREATE TABLE IF NOT EXISTS modules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code varchar(50) NOT NULL UNIQUE,
    name varchar(100) NOT NULL,
    description text NOT NULL DEFAULT '',
    icon varchar(50) NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS modules_sort_order_idx ON modules (sort_order);

-- Insert 8 predefined business modules
INSERT INTO modules (code, name, description, icon, sort_order) VALUES
    ('attendance', 'Presensi & Absensi', 'Pencatatan kehadiran harian, check-in GPS & swafoto, persetujuan lembur/koreksi, dan rekapitulasi data kehadiran.', 'Clock', 1),
    ('employee', 'Karyawan & Pegawai', 'Manajemen data profil pegawai, nomor induk (NIP), jabatan, departemen, dan status kepegawaian.', 'Users', 2),
    ('face', 'Biometrik Wajah (AI)', 'Pengambilan foto referensi biometrik (enrollment), galeri vektor wajah, dan sinkronisasi model pengenalan AI.', 'ScanFace', 3),
    ('location', 'Lokasi Kantor & Geofence', 'Pengaturan titik koordinat GPS kantor dan radius geofencing tempat presensi sah.', 'MapPin', 4),
    ('user', 'Pengguna Sistem & Akun', 'Pengelolaan akun login (email, kata sandi, status keaktifan, dan penguncian).', 'UserCheck', 5),
    ('role', 'Peran & Hak Akses (RBAC)', 'Pengaturan kelompok hak akses pengguna, pembuatan peran baru, dan katalog izin.', 'Shield', 6),
    ('settings', 'Pengaturan Sistem', 'Konfigurasi parameter ambang toleransi wajah, waktu sesi, dan kebijakan aplikasi.', 'Sliders', 7),
    ('audit', 'Jejak Audit & Keamanan', 'Catatan riwayat aktivitas operasional penting dan audit kepatuhan data.', 'FileText', 8)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    sort_order = EXCLUDED.sort_order,
    updated_at = now();

-- Add module_id to permissions
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS module_id uuid REFERENCES modules(id) ON DELETE RESTRICT;

-- Link permissions to modules based on resource
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'attendance') WHERE resource = 'attendance';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'employee') WHERE resource = 'employee';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'face') WHERE resource = 'face';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'location') WHERE resource = 'location';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'user') WHERE resource = 'user';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'role') WHERE resource IN ('role', 'permission');
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'settings') WHERE resource = 'settings';
UPDATE permissions SET module_id = (SELECT id FROM modules WHERE code = 'audit') WHERE resource = 'audit';

-- Create trigger function to auto-assign module_id on new/updated permissions
CREATE OR REPLACE FUNCTION set_permission_module_id()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.module_id IS NULL THEN
        IF NEW.resource IN ('role', 'permission') THEN
            SELECT id INTO NEW.module_id FROM modules WHERE code = 'role';
        ELSE
            SELECT id INTO NEW.module_id FROM modules WHERE code = NEW.resource;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_permission_module_id ON permissions;
CREATE TRIGGER trg_set_permission_module_id
BEFORE INSERT OR UPDATE ON permissions
FOR EACH ROW
EXECUTE FUNCTION set_permission_module_id();

-- Enforce NOT NULL on module_id once populated
ALTER TABLE permissions ALTER COLUMN module_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS permissions_module_id_idx ON permissions (module_id);
