-- Migration 000016: Face Model Reindex Jobs and Items (Fase 3 § 3.7)

CREATE TABLE IF NOT EXISTS face_reindex_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    from_model_version text NOT NULL,
    to_model_version text NOT NULL CHECK (to_model_version <> from_model_version),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    total_count integer NOT NULL DEFAULT 0,
    processed_count integer NOT NULL DEFAULT 0,
    succeeded_count integer NOT NULL DEFAULT 0,
    failed_count integer NOT NULL DEFAULT 0,
    employees_ready_count integer NOT NULL DEFAULT 0,
    employees_incomplete_count integer NOT NULL DEFAULT 0,
    error text NULL,
    created_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    started_at timestamptz NULL,
    finished_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS face_reindex_jobs_running_uniq
    ON face_reindex_jobs ((true)) WHERE status IN ('pending', 'running');

DROP TRIGGER IF EXISTS face_reindex_jobs_set_updated_at ON face_reindex_jobs;
CREATE TRIGGER face_reindex_jobs_set_updated_at
    BEFORE UPDATE ON face_reindex_jobs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS face_reindex_items (
    job_id uuid NOT NULL REFERENCES face_reindex_jobs(id) ON DELETE CASCADE,
    face_reference_id uuid NOT NULL REFERENCES face_references(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ok', 'failed', 'skipped')),
    new_reference_id uuid NULL REFERENCES face_references(id) ON DELETE SET NULL,
    hints jsonb NOT NULL DEFAULT '[]'::jsonb,
    reason text NULL,
    processed_at timestamptz NULL,
    PRIMARY KEY (job_id, face_reference_id)
);

CREATE INDEX IF NOT EXISTS face_reindex_items_status_idx 
    ON face_reindex_items (job_id, status);
