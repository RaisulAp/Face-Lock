/**
 * Camera stream management utilities.
 * Handles WebRTC getUserMedia acquisition, stream lifecycle, and permission errors.
 */

export interface CameraStreamOptions {
    facingMode?: "user" | "environment";
    idealWidth?: number;
    idealHeight?: number;
}

export type CameraErrorKind =
    | "NOT_SUPPORTED"
    | "NOT_SECURE"
    | "PERMISSION_DENIED"
    | "NOT_FOUND"
    | "IN_USE"
    | "OVERCONSTRAINED"
    | "UNKNOWN";

export interface CameraErrorDetails {
    kind: CameraErrorKind;
    message: string;
    originalError?: unknown;
}

export function isCameraSupported(): boolean {
    return (
        typeof navigator !== "undefined" &&
        !!navigator.mediaDevices &&
        typeof navigator.mediaDevices.getUserMedia === "function"
    );
}

export function isSecureCameraContext(): boolean {
    if (typeof window === "undefined") return true;
    // Localhost is considered secure even over http
    const isLocalhost =
        window.location.hostname === "localhost" ||
        window.location.hostname === "127.0.0.1" ||
        window.location.hostname === "::1";
    return window.isSecureContext || isLocalhost;
}

export async function requestCameraStream(options: CameraStreamOptions = {}): Promise<MediaStream> {
    if (!isSecureCameraContext()) {
        const err: CameraErrorDetails = {
            kind: "NOT_SECURE",
            message: "Akses kamera memerlukan konteks aman (HTTPS) atau localhost.",
        };
        throw err;
    }

    if (!isCameraSupported()) {
        const err: CameraErrorDetails = {
            kind: "NOT_SUPPORTED",
            message: "Browser Anda tidak mendukung akses kamera WebRTC.",
        };
        throw err;
    }

    const facingMode = options.facingMode ?? "user";
    const idealWidth = options.idealWidth ?? 1280;
    const idealHeight = options.idealHeight ?? 720;

    const constraints: MediaStreamConstraints = {
        audio: false,
        video: {
            facingMode: { ideal: facingMode },
            width: { ideal: idealWidth },
            height: { ideal: idealHeight },
        },
    };

    try {
        return await navigator.mediaDevices.getUserMedia(constraints);
    } catch (rawError: unknown) {
        const errorName = (rawError as { name?: string })?.name || "";
        let kind: CameraErrorKind = "UNKNOWN";
        let message = "Gagal mengakses kamera perangkat.";

        if (errorName === "NotAllowedError" || errorName === "PermissionDeniedError") {
            kind = "PERMISSION_DENIED";
            message = "Izin akses kamera ditolak. Harap izinkan akses kamera di pengaturan browser Anda.";
        } else if (errorName === "NotFoundError" || errorName === "DevicesNotFoundError") {
            kind = "NOT_FOUND";
            message = "Kamera tidak terdeteksi pada perangkat ini.";
        } else if (errorName === "NotReadableError" || errorName === "TrackStartError") {
            kind = "IN_USE";
            message = "Kamera sedang digunakan oleh aplikasi lain atau mengalami kendala hardware.";
        } else if (errorName === "OverconstrainedError" || errorName === "ConstraintNotSatisfiedError") {
            // Fallback to basic video without resolution constraints
            try {
                return await navigator.mediaDevices.getUserMedia({
                    audio: false,
                    video: { facingMode },
                });
            } catch {
                kind = "OVERCONSTRAINED";
                message = "Kamera tidak mendukung konfigurasi resolusi yang diminta.";
            }
        }

        const err: CameraErrorDetails = {
            kind,
            message,
            originalError: rawError,
        };
        throw err;
    }
}

export function stopCameraStream(stream: MediaStream | null): void {
    if (!stream) return;
    stream.getTracks().forEach((track) => {
        try {
            track.stop();
        } catch {
            // ignore track stop errors
        }
    });
}
