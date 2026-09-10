-- Migration 000012: Biometric Consents and Documents (Fase 3 UU PDP)

CREATE TABLE IF NOT EXISTS consent_documents (
    version text PRIMARY KEY CHECK (version ~ '^[0-9]{4}-[0-9]{2}-v[0-9]+$'),
    title text NOT NULL,
    body text NOT NULL,
    content_hash text NOT NULL,
    is_active boolean NOT NULL DEFAULT false,
    published_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS consent_documents_active_uniq 
    ON consent_documents (is_active) WHERE is_active;

DROP TRIGGER IF EXISTS consent_documents_set_updated_at ON consent_documents;
CREATE TRIGGER consent_documents_set_updated_at
    BEFORE UPDATE ON consent_documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Seed initial default biometric consent document
INSERT INTO consent_documents (version, title, body, content_hash, is_active, published_at)
VALUES (
    '2026-09-v1',
    'Persetujuan Pemrosesan Data Biometrik Pengenalan Wajah',
    '# Lembar Persetujuan Pemrosesan Data Pribadi Biometrik (UU PDP No. 27 Tahun 2022)

Dengan ini saya menyatakan memberikan persetujuan eksplisit kepada FaceClock untuk:
1. Mengumpulkan, merekam, dan memproses data foto wajah dan representasi vektor biometrik saya.
2. Menggunakan data biometrik tersebut secara eksklusif untuk tujuan pencatatan presensi kehadiran kerja.
3. Menyimpan data biometrik secara aman terenkripsi dan tidak membagikannya kepada pihak ketiga tanpa izin saya.
4. Memberikan hak kepada saya sewaktu-waktu untuk mencabut persetujuan atau meminta penghapusan data biometrik saya.',
    encode(sha256('# Lembar Persetujuan Pemrosesan Data Pribadi Biometrik (UU PDP No. 27 Tahun 2022)

Dengan ini saya menyatakan memberikan persetujuan eksplisit kepada FaceClock untuk:
1. Mengumpulkan, merekam, dan memproses data foto wajah dan representasi vektor biometrik saya.
2. Menggunakan data biometrik tersebut secara eksklusif untuk tujuan pencatatan presensi kehadiran kerja.
3. Menyimpan data biometrik secara aman terenkripsi dan tidak membagikannya kepada pihak ketiga tanpa izin saya.
4. Memberikan hak kepada saya sewaktu-waktu untuk mencabut persetujuan atau meminta penghapusan data biometrik saya.'::bytea), 'hex'),
    true,
    now()
) ON CONFLICT (version) DO NOTHING;

CREATE TABLE IF NOT EXISTS biometric_consents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    document_version text NOT NULL REFERENCES consent_documents(version) ON DELETE RESTRICT,
    content_hash text NOT NULL,
    status text NOT NULL CHECK (status IN ('granted', 'withdrawn')),
    method text NOT NULL CHECK (method IN ('self_web', 'self_mobile', 'admin_recorded')),
    granted_at timestamptz NOT NULL DEFAULT now(),
    withdrawn_at timestamptz NULL,
    withdrawn_reason text NULL,
    recorded_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    ip inet NULL,
    user_agent text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT biometric_consents_withdraw_chk CHECK ((status = 'withdrawn') = (withdrawn_at IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS biometric_consents_active_uniq
    ON biometric_consents (employee_id) WHERE status = 'granted';

CREATE INDEX IF NOT EXISTS biometric_consents_employee_idx 
    ON biometric_consents (employee_id, granted_at DESC);

DROP TRIGGER IF EXISTS biometric_consents_set_updated_at ON biometric_consents;
CREATE TRIGGER biometric_consents_set_updated_at
    BEFORE UPDATE ON biometric_consents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
