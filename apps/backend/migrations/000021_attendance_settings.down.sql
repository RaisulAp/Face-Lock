DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('attendance.create', 'attendance.review', 'attendance.read_team', 'location.manage')
);

DELETE FROM permissions WHERE name IN ('attendance.create', 'attendance.review', 'attendance.read_team', 'location.manage');

DELETE FROM app_settings WHERE key IN (
    'attendance.workday_cutoff_hour',
    'attendance.fallback_enabled',
    'attendance.outside_geofence_policy',
    'attendance.missing_location_policy',
    'attendance.max_gps_accuracy_meter',
    'attendance.allow_checkout_without_checkin',
    'attendance.min_minutes_between_checkin_checkout',
    'attendance.require_face_for_checkout',
    'attendance.checkout_without_face_status',
    'attendance.allow_fallback_without_enrollment',
    'attendance.max_note_length',
    'attendance.max_failed_attempts_per_hour',
    'attendance.photo_retention_days',
    'attendance.attempt_retention_days',
    'attendance.duplicate_photo_window_days',
    'attendance.liveness_policy',
    'attendance.liveness_max_attempts',
    'attendance.liveness_challenge_count',
    'attendance.liveness_timeout_seconds',
    'attendance.mocked_location_policy'
);
