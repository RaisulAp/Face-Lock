import { ulid } from "ulid";

/**
 * Idempotency Key Manager for Attendance Clock-In/Clock-Out.
 *
 * Rules:
 * - A fresh ULID is generated when user captures a new photo.
 * - If a network error occurs, the SAME ULID must be re-sent on retry so the backend
 *   recognizes the request as identical and avoids double-recording.
 * - If user decides to retake the photo, a brand new ULID must be generated.
 */

export function generateIdempotencyKey(): string {
    return ulid();
}
