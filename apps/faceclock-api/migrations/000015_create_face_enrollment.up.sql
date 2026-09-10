-- Migration 000015: Face Enrollment Sessions and Photos (Fase 3 Staging Pipeline)

CREATE TABLE IF NOT EXISTS face_enrollment_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'committed', 'cancelled', 'expired')),
    mode text NOT NULL DEFAULT 'replace' CHECK (mode IN ('replace', 'append')),
    required_photos smallint NOT NULL,
    max_photos smallint NOT NULL,
    model_version text NOT NULL,
    created_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    expires_at timestamptz NOT NULL,
    committed_at timestamptz NULL,
    cancelled_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS face_enrollment_sessions_draft_uniq
    ON face_enrollment_sessions (employee_id) WHERE status = 'draft';

CREATE INDEX IF NOT EXISTS face_enrollment_sessions_expiry_idx
    ON face_enrollment_sessions (expires_at) WHERE status = 'draft';

DROP TRIGGER IF EXISTS face_enrollment_sessions_set_updated_at ON face_enrollment_sessions;
CREATE TRIGGER face_enrollment_sessions_set_updated_at
    BEFORE UPDATE ON face_enrollment_sessions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Add foreign key from face_references to face_enrollment_sessions
ALTER TABLE face_references DROP CONSTRAINT IF EXISTS face_references_enrollment_session_fkey;
ALTER TABLE face_references
    ADD CONSTRAINT face_references_enrollment_session_fkey
    FOREIGN KEY (enrollment_session_id) REFERENCES face_enrollment_sessions(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS face_enrollment_photos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES face_enrollment_sessions(id) ON DELETE CASCADE,
    position smallint NOT NULL CHECK (position BETWEEN 1 AND 20),
    embedding vector(512) NOT NULL,
    quality_score real NOT NULL,
    det_score real NULL,
    hints jsonb NOT NULL DEFAULT '[]'::jsonb,
    photo_key text NOT NULL,
    photo_sha256 text NOT NULL,
    photo_bytes integer NOT NULL,
    photo_mime text NOT NULL,
    capture_source text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS face_enrollment_photos_pos_uniq 
    ON face_enrollment_photos (session_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS face_enrollment_photos_sha_uniq 
    ON face_enrollment_photos (session_id, photo_sha256);
