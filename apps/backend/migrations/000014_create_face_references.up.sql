-- Migration 000014: Evolve Face References to Phase 3 Specification

-- Allow photo_key to be NULL when photo_purged_at IS NOT NULL
ALTER TABLE face_references ALTER COLUMN photo_key DROP NOT NULL;

-- Add new columns to face_references
ALTER TABLE face_references
    ADD COLUMN IF NOT EXISTS det_score real NULL CHECK (det_score IS NULL OR det_score BETWEEN 0 AND 1),
    ADD COLUMN IF NOT EXISTS photo_sha256 text NULL,
    ADD COLUMN IF NOT EXISTS photo_bytes integer NOT NULL DEFAULT 1024 CHECK (photo_bytes > 0),
    ADD COLUMN IF NOT EXISTS photo_mime text NOT NULL DEFAULT 'image/jpeg' CHECK (photo_mime IN ('image/jpeg', 'image/png', 'image/webp')),
    ADD COLUMN IF NOT EXISTS photo_purged_at timestamptz NULL,
    ADD COLUMN IF NOT EXISTS capture_source text NOT NULL DEFAULT 'web_camera' CHECK (capture_source IN ('web_camera', 'mobile_camera', 'admin_upload')),
    ADD COLUMN IF NOT EXISTS position smallint NOT NULL DEFAULT 1 CHECK (position BETWEEN 1 AND 20),
    ADD COLUMN IF NOT EXISTS enrollment_session_id uuid NULL,
    ADD COLUMN IF NOT EXISTS enrolled_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS superseded_by uuid NULL REFERENCES face_references(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS deactivated_at timestamptz NULL,
    ADD COLUMN IF NOT EXISTS deactivated_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS deactivated_reason text NULL CHECK (deactivated_reason IS NULL OR deactivated_reason IN ('replaced', 're_enroll', 'admin_removed', 'consent_withdrawn', 'model_reindex', 'quality_review', 'employee_resigned')),
    ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

-- Backfill photo_sha256 for existing rows if any
UPDATE face_references 
SET photo_sha256 = encode(sha256(id::text::bytea), 'hex') 
WHERE photo_sha256 IS NULL;

ALTER TABLE face_references ALTER COLUMN photo_sha256 SET NOT NULL;

ALTER TABLE face_references DROP CONSTRAINT IF EXISTS face_references_photo_sha256_chk;
ALTER TABLE face_references ADD CONSTRAINT face_references_photo_sha256_chk CHECK (photo_sha256 ~ '^[0-9a-f]{64}$');

ALTER TABLE face_references DROP CONSTRAINT IF EXISTS face_references_deactivation_chk;
ALTER TABLE face_references ADD CONSTRAINT face_references_deactivation_chk CHECK ((is_active = false) = (deactivated_at IS NOT NULL));

ALTER TABLE face_references DROP CONSTRAINT IF EXISTS face_references_photo_chk;
ALTER TABLE face_references ADD CONSTRAINT face_references_photo_chk CHECK ((photo_key IS NOT NULL) OR (photo_purged_at IS NOT NULL));

CREATE INDEX IF NOT EXISTS face_references_lookup_idx
    ON face_references (employee_id, is_active, model_version);

CREATE INDEX IF NOT EXISTS face_references_active_model_idx
    ON face_references (model_version) WHERE is_active;

CREATE INDEX IF NOT EXISTS face_references_session_idx
    ON face_references (enrollment_session_id);

CREATE UNIQUE INDEX IF NOT EXISTS face_references_photo_uniq
    ON face_references (employee_id, photo_sha256) WHERE is_active;

DROP TRIGGER IF EXISTS face_references_set_updated_at ON face_references;
CREATE TRIGGER face_references_set_updated_at
    BEFORE UPDATE ON face_references
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
