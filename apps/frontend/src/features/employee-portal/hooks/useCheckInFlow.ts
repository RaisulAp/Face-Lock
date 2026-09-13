import { useState, useCallback, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { employeeApi } from "../api";
import { useGeolocation } from "../../../hooks/useGeolocation";
import { useCamera } from "../../../hooks/useCamera";
import { useCapture } from "../../../hooks/useCapture";
import { generateIdempotencyKey } from "../../../lib/idempotency";
import { ApiError } from "../../../lib/api";
import type { AttendanceActionType, ClockSubmitResult } from "../types";

export function useCheckInFlow() {
    const queryClient = useQueryClient();

    // 1. Fetch Today's Attendance Context
    const {
        data: context,
        isLoading: isLoadingContext,
        error: contextError,
        refetch: refetchContext,
    } = useQuery({
        queryKey: ["attendance-context"],
        queryFn: employeeApi.getContext,
        staleTime: 60 * 1000,
    });

    // 2. Determine Default Action (Check-In vs Check-Out)
    const [userSelectedAction, setUserSelectedAction] = useState<AttendanceActionType | null>(null);

    const actionType: AttendanceActionType =
        userSelectedAction ??
        (context?.can_check_in ? "check_in" : context?.can_check_out ? "check_out" : "check_in");

    const setActionType = (type: AttendanceActionType) => {
        setUserSelectedAction(type);
    };

    // 3. Geolocation & Geofence (Haversine against active offices)
    const geo = useGeolocation({
        offices: context?.active_offices || [],
        maxAccuracyMeters: 150,
    });

    // 4. Camera and Media Pipeline
    const camera = useCamera({ facingMode: "user" });
    const capture = useCapture({ maxBytes: 800 * 1024, maxWidth: 960 });

    // 5. State & Idempotency Key Lifecycle
    const idempotencyKeyRef = useRef<string>(generateIdempotencyKey());
    const [capturedBlob, setCapturedBlob] = useState<Blob | null>(null);
    const [capturedPreviewUrl, setCapturedPreviewUrl] = useState<string | null>(null);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [submitError, setSubmitError] = useState<string | null>(null);
    const [activeHints, setActiveHints] = useState<string[]>([]);
    const [consecutiveFailures, setConsecutiveFailures] = useState(0);
    const [lastResult, setLastResult] = useState<ClockSubmitResult | null>(null);
    const [isFallbackModalOpen, setIsFallbackModalOpen] = useState(false);

    // Take photo from active video stream
    const takePhoto = useCallback(async () => {
        if (!camera.videoRef.current) return;
        const res = await capture.takePhoto(camera.videoRef.current);
        if (res) {
            setCapturedBlob(res.blob);
            setCapturedPreviewUrl(res.dataUrl);
            setSubmitError(null);
            setActiveHints([]);
            // Fresh photo gets a fresh idempotency key (ULID)
            idempotencyKeyRef.current = generateIdempotencyKey();
        }
    }, [camera.videoRef, capture]);

    // When photo is captured manually
    const onPhotoCaptured = useCallback((blob: Blob) => {
        setCapturedBlob(blob);
        setCapturedPreviewUrl(URL.createObjectURL(blob));
        setSubmitError(null);
        setActiveHints([]);
        idempotencyKeyRef.current = generateIdempotencyKey();
    }, []);

    // Retake photo
    const onRetakePhoto = useCallback(() => {
        if (capturedPreviewUrl) {
            URL.revokeObjectURL(capturedPreviewUrl);
        }
        setCapturedBlob(null);
        setCapturedPreviewUrl(null);
        setSubmitError(null);
        setActiveHints([]);
        capture.clearPhoto();
        idempotencyKeyRef.current = generateIdempotencyKey();
    }, [capturedPreviewUrl, capture]);

    // Submit attendance
    const submitAttendance = useCallback(
        async (options?: { allowFallback?: boolean; fallbackReason?: string; notes?: string }) => {
            if (!capturedBlob) {
                setSubmitError("Silakan ambil foto wajah terlebih dahulu.");
                return;
            }

            setIsSubmitting(true);
            setSubmitError(null);
            setActiveHints([]);

            try {
                const result = await employeeApi.submitClock(actionType, {
                    photo: capturedBlob,
                    type: actionType,
                    latitude: geo.coords?.latitude,
                    longitude: geo.coords?.longitude,
                    accuracy: geo.coords?.accuracy,
                    allow_fallback: options?.allowFallback,
                    fallback_reason: options?.fallbackReason,
                    note: options?.notes,
                    idempotency_key: idempotencyKeyRef.current,
                });

                setLastResult(result);
                setConsecutiveFailures(0);
                setIsFallbackModalOpen(false);

                // Invalidate context to update today's status
                queryClient.invalidateQueries({ queryKey: ["attendance-context"] });
                queryClient.invalidateQueries({ queryKey: ["attendance-history-me"] });
            } catch (err: unknown) {
                if (err instanceof ApiError) {
                    setSubmitError(err.message);
                    if (err.hints && err.hints.length > 0) {
                        setActiveHints(err.hints);
                    }

                    // If biometric match failed or quality error, increment failure counter
                    if (
                        err.code === "BIOMETRIC_MISMATCH" ||
                        err.code === "NO_FACE_DETECTED" ||
                        err.code === "MULTIPLE_FACES" ||
                        err.code === "FACE_QUALITY_FAILED"
                    ) {
                        setConsecutiveFailures((prev) => prev + 1);
                    }
                } else {
                    setSubmitError(err instanceof Error ? err.message : "Terjadi kesalahan pada jaringan. Coba lagi.");
                }
            } finally {
                setIsSubmitting(false);
            }
        },
        [capturedBlob, actionType, geo.coords, queryClient],
    );

    // Reset entire flow
    const resetFlow = useCallback(() => {
        if (capturedPreviewUrl) {
            URL.revokeObjectURL(capturedPreviewUrl);
        }
        setCapturedBlob(null);
        setCapturedPreviewUrl(null);
        setSubmitError(null);
        setActiveHints([]);
        setLastResult(null);
        capture.clearPhoto();
        idempotencyKeyRef.current = generateIdempotencyKey();
        refetchContext();
    }, [capturedPreviewUrl, capture, refetchContext]);

    return {
        context,
        isLoadingContext,
        contextError,
        actionType,
        setActionType,
        geo,
        camera,
        capture,
        takePhoto,
        capturedBlob,
        capturedPreviewUrl,
        onPhotoCaptured,
        onRetakePhoto,
        isSubmitting,
        submitError,
        activeHints,
        consecutiveFailures,
        lastResult,
        isFallbackModalOpen,
        setIsFallbackModalOpen,
        submitAttendance,
        resetFlow,
    };
}
