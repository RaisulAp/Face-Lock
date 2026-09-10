DROP TABLE IF EXISTS face_enrollment_photos CASCADE;
ALTER TABLE face_references DROP CONSTRAINT IF EXISTS face_references_enrollment_session_fkey;
DROP TABLE IF EXISTS face_enrollment_sessions CASCADE;
