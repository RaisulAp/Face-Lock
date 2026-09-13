-- Migration 000011: Face Biometric References and Attendance Records (Fase 2)

-- 1. Add biometric fields to employees table
ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS face_embedding vector(512) NULL,
    ADD COLUMN IF NOT EXISTS face_photo_path text NULL,
    ADD COLUMN IF NOT EXISTS face_enrolled_at timestamptz NULL,
    ADD COLUMN IF NOT EXISTS face_model_version text NULL;

CREATE INDEX IF NOT EXISTS employees_face_enrolled_at_idx 
    ON employees (face_enrolled_at) 
    WHERE deleted_at IS NULL AND face_enrolled_at IS NOT NULL;

-- 2. Create face_references table (supports multiple reference photos and future re-indexing)
CREATE TABLE IF NOT EXISTS face_references (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    photo_key text NOT NULL,
    embedding vector(512) NOT NULL,
    model_version text NOT NULL,
    quality_score double precision NOT NULL DEFAULT 1.0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS face_references_employee_id_idx 
    ON face_references (employee_id) 
    WHERE is_active = true;

-- 3. Create attendances table
CREATE TABLE IF NOT EXISTS attendances (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    type text NOT NULL CHECK (type IN ('in', 'out')),
    status text NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'pending_review')),
    photo_key text NOT NULL,
    face_embedding vector(512) NULL,
    similarity_score double precision NULL,
    distance double precision NULL,
    model_version text NOT NULL,
    notes text NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS attendances_employee_id_recorded_at_idx 
    ON attendances (employee_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS attendances_recorded_at_idx 
    ON attendances (recorded_at DESC);

CREATE INDEX IF NOT EXISTS attendances_status_idx 
    ON attendances (status);

-- 4. Seed face quality settings (REV-SET-01, Fase 2 § 3.1)
INSERT INTO app_settings (key, value, value_type, description, is_public) VALUES
  ('face.det_size',            '640',   'number',  'Ukuran input detektor SCRFD',                       false),
  ('face.min_det_score',       '0.60',  'number',  'Ambang minimum confidence deteksi wajah',           false),
  ('face.min_blur_var',        '40.0',  'number',  'Ambang minimum variance of Laplacian (ketajaman)',  false),
  ('face.min_brightness',      '55.0',  'number',  'Ambang minimum kecerahan crop wajah (0-255)',       false),
  ('face.max_brightness',      '215.0', 'number',  'Ambang maksimum kecerahan crop wajah (0-255)',      false),
  ('face.min_face_ratio',      '0.18',  'number',  'Rasio minimum tinggi wajah terhadap tinggi frame',  false),
  ('face.max_abs_yaw',         '0.35',  'number',  'Ambang maksimum proxy yaw (tak berdimensi)',        false),
  ('face.max_abs_pitch',       '0.30',  'number',  'Ambang maksimum proxy pitch (tak berdimensi)',      false),
  ('face.max_image_bytes',     '6291456','number', 'Ukuran maksimum satu foto (byte)',                  true),
  ('face.accepted_mime_types', '["image/jpeg","image/png","image/webp"]', 'json', 'Format foto yang diterima', true)
ON CONFLICT (key) DO NOTHING;

