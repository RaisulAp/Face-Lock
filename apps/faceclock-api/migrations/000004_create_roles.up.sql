CREATE TABLE roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name citext NOT NULL CHECK (name ~ '^[a-z][a-z0-9_]{2,49}$'),
    display_name text NOT NULL,
    description text NULL,
    is_system boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE UNIQUE INDEX roles_name_uniq ON roles (name) WHERE deleted_at IS NULL;

CREATE TRIGGER roles_set_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

INSERT INTO roles (name, display_name, description, is_system) VALUES
    ('super_admin', 'Super Administrator', 'Akses penuh ke seluruh konfigurasi, data, dan modul sistem.', true),
    ('admin', 'Administrator', 'Pengelolaan operasional pengguna, karyawan, presensi, dan pengaturan.', true),
    ('employee', 'Pegawai', 'Akses mandiri presensi dan profil diri.', true)
ON CONFLICT (name) WHERE deleted_at IS NULL DO NOTHING;
