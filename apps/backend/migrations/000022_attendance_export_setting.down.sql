-- Migration 000022 rollback
DELETE FROM app_settings WHERE key = 'attendance.export_max_rows';
