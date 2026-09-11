CREATE TABLE IF NOT EXISTS attendance_attempts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    type varchar(20) NOT NULL,
    server_timestamp timestamptz NOT NULL DEFAULT now(),
    work_date date NOT NULL,
    outcome varchar(50) NOT NULL,
    matched_similarity double precision,
    threshold_used double precision,
    model_version varchar(50),
    quality_score double precision,
    hints jsonb NOT NULL DEFAULT '[]'::jsonb,
    geofence_status varchar(50),
    distance_meter double precision,
    allow_fallback boolean NOT NULL DEFAULT false,
    attendance_id uuid REFERENCES attendances(id) ON DELETE SET NULL,
    failure_reason text,
    request_id varchar(100),
    ip inet,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT attendance_attempts_type_chk CHECK (type IN ('check_in', 'check_out')),
    CONSTRAINT attendance_attempts_outcome_chk CHECK (
        outcome IN (
            'matched', 'below_threshold', 'face_not_usable', 'inference_unavailable',
            'no_reference', 'geofence_rejected', 'rule_rejected', 'duplicate_photo',
            'rate_limited', 'manual_mode', 'liveness_failed', 'location_mocked', 'success'
        )
    )
);

CREATE INDEX IF NOT EXISTS attendance_attempts_employee_idx
    ON attendance_attempts (employee_id, server_timestamp DESC);

CREATE INDEX IF NOT EXISTS attendance_attempts_outcome_idx
    ON attendance_attempts (outcome, server_timestamp DESC);

CREATE INDEX IF NOT EXISTS attendance_attempts_ratelimit_idx
    ON attendance_attempts (employee_id, server_timestamp DESC)
    WHERE outcome NOT IN ('matched', 'success');

CREATE INDEX IF NOT EXISTS attendance_attempts_cleanup_idx
    ON attendance_attempts (server_timestamp ASC);
