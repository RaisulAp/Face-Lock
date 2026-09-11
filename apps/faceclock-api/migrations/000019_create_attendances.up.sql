-- Drop prototype table from migration 000011
DROP TABLE IF EXISTS attendances CASCADE;

CREATE TABLE attendances (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    work_date date NOT NULL,
    type varchar(20) NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'approved',
    method varchar(20) NOT NULL,
    server_timestamp timestamptz NOT NULL DEFAULT now(),
    client_reported_at timestamptz NOT NULL,
    clock_skew_seconds integer NOT NULL,
    latitude double precision,
    longitude double precision,
    distance_meter double precision,
    matched_office_id uuid REFERENCES office_locations(id) ON DELETE SET NULL,
    location_is_mocked boolean NOT NULL DEFAULT false,
    matched_similarity double precision,
    threshold_used double precision,
    model_version varchar(50),
    quality_score double precision,
    photo_key varchar(500),
    photo_sha256 char(64) NOT NULL,
    photo_bytes integer NOT NULL,
    photo_mime varchar(50) NOT NULL DEFAULT 'image/jpeg',
    photo_purged_at timestamptz,
    liveness_passed boolean,
    liveness_supported boolean NOT NULL DEFAULT false,
    liveness_method varchar(50),
    liveness_challenges jsonb,
    fallback_reason varchar(50),
    fallback_note text,
    reviewed_by uuid REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at timestamptz,
    review_notes text,
    idempotency_key varchar(100),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT attendances_photo_chk CHECK ((photo_key IS NOT NULL) OR (photo_purged_at IS NOT NULL)),
    CONSTRAINT attendances_type_chk CHECK (type IN ('check_in', 'check_out')),
    CONSTRAINT attendances_status_chk CHECK (status IN ('approved', 'rejected', 'pending_review')),
    CONSTRAINT attendances_method_chk CHECK (method IN ('face_verified', 'fallback_manual', 'fallback_offline', 'system_timeout')),
    CONSTRAINT attendances_fallback_chk CHECK (
        (method = 'face_verified' AND fallback_reason IS NULL) OR
        (method <> 'face_verified' AND fallback_reason IS NOT NULL)
    ),
    CONSTRAINT attendances_fallback_reason_chk CHECK (
        fallback_reason IS NULL OR
        fallback_reason IN (
            'face_not_matched', 'camera_broken', 'face_not_usable',
            'no_reference_enrolled', 'system_error', 'service_unavailable',
            'outside_geofence', 'liveness_failed', 'location_mocked', 'other'
        )
    ),
    CONSTRAINT attendances_fallback_never_approved_chk CHECK (
        (method <> 'face_verified' AND status = 'approved' AND reviewed_by IS NOT NULL) OR
        (method <> 'face_verified' AND status <> 'approved') OR
        (method = 'face_verified')
    )
);

CREATE UNIQUE INDEX attendances_one_checkin_per_day
    ON attendances (employee_id, work_date)
    WHERE type = 'check_in' AND status <> 'rejected';

CREATE UNIQUE INDEX attendances_one_checkout_per_day
    ON attendances (employee_id, work_date)
    WHERE type = 'check_out' AND status <> 'rejected';

CREATE UNIQUE INDEX attendances_idempotency_uidx
    ON attendances (employee_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX attendances_employee_date_idx
    ON attendances (employee_id, work_date DESC);

CREATE INDEX attendances_pending_review_idx
    ON attendances (status, server_timestamp ASC)
    WHERE status = 'pending_review';

CREATE INDEX attendances_photo_sha_idx
    ON attendances (photo_sha256);

CREATE INDEX attendances_photo_purge_idx
    ON attendances (server_timestamp ASC)
    WHERE photo_purged_at IS NULL;
