-- Migration 000011 Down

DROP TABLE IF EXISTS attendances CASCADE;
DROP TABLE IF EXISTS face_references CASCADE;

ALTER TABLE employees
    DROP COLUMN IF EXISTS face_embedding,
    DROP COLUMN IF EXISTS face_photo_path,
    DROP COLUMN IF EXISTS face_enrolled_at,
    DROP COLUMN IF EXISTS face_model_version;

DELETE FROM app_settings WHERE key IN (
  'face.det_size',
  'face.min_det_score',
  'face.min_blur_var',
  'face.min_brightness',
  'face.max_brightness',
  'face.min_face_ratio',
  'face.max_abs_yaw',
  'face.max_abs_pitch',
  'face.max_image_bytes',
  'face.accepted_mime_types'
);

