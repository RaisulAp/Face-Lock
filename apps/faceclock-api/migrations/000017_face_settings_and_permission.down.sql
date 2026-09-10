DELETE FROM role_permissions 
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'face.reindex');

DELETE FROM permissions WHERE name = 'face.reindex';

DELETE FROM app_settings WHERE key IN (
  'face.min_reference_photos',
  'face.max_reference_photos',
  'face.enrollment_session_ttl_minutes',
  'face.min_quality_score',
  'face.duplicate_check_enabled',
  'face.duplicate_threshold',
  'face.retention_days_after_resign',
  'face.consent_required'
);
