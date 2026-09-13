DROP TRIGGER IF EXISTS face_references_set_updated_at ON face_references;
DROP INDEX IF EXISTS face_references_photo_uniq;
DROP INDEX IF EXISTS face_references_session_idx;
DROP INDEX IF EXISTS face_references_active_model_idx;
DROP INDEX IF EXISTS face_references_lookup_idx;

ALTER TABLE face_references
    DROP CONSTRAINT IF EXISTS face_references_photo_chk,
    DROP CONSTRAINT IF EXISTS face_references_deactivation_chk,
    DROP CONSTRAINT IF EXISTS face_references_photo_sha256_chk,
    DROP COLUMN IF EXISTS det_score,
    DROP COLUMN IF EXISTS photo_sha256,
    DROP COLUMN IF EXISTS photo_bytes,
    DROP COLUMN IF EXISTS photo_mime,
    DROP COLUMN IF EXISTS photo_purged_at,
    DROP COLUMN IF EXISTS capture_source,
    DROP COLUMN IF EXISTS position,
    DROP COLUMN IF EXISTS enrollment_session_id,
    DROP COLUMN IF EXISTS enrolled_by,
    DROP COLUMN IF EXISTS superseded_by,
    DROP COLUMN IF EXISTS deactivated_at,
    DROP COLUMN IF EXISTS deactivated_by,
    DROP COLUMN IF EXISTS deactivated_reason,
    DROP COLUMN IF EXISTS updated_at;
