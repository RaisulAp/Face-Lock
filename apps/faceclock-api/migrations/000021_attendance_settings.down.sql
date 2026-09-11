DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('attendance.create', 'attendance.review', 'attendance.read_team', 'location.manage')
);

DELETE FROM permissions WHERE name IN ('attendance.create', 'attendance.review', 'attendance.read_team', 'location.manage');

DELETE FROM app_settings WHERE key IN (
    'attendance_mode', 'max_distance_meter', 'geofence_fail_action',
    'max_daily_fallback_per_employee', 'checkout_requires_face',
    'min_checkout_interval_minutes', 'fallback_requires_review',
    'anti_spoofing_strictness', 'work_day_cutoff_hour',
    'attendance_timezone', 'max_failed_attempts_lockout',
    'failed_attempts_window_minutes', 'photo_retention_days',
    'max_gps_accuracy_meter', 'require_gps_accuracy',
    'liveness_mode', 'liveness_reject_action',
    'liveness_score_threshold', 'liveness_active_challenge_count',
    'location_allow_mocked'
);
