-- Migration 000013: Employee Attendance Mode (Fase 3)

ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS attendance_mode text NOT NULL DEFAULT 'face' CHECK (attendance_mode IN ('face', 'manual'));

CREATE INDEX IF NOT EXISTS employees_attendance_mode_idx 
    ON employees (attendance_mode) 
    WHERE deleted_at IS NULL;
