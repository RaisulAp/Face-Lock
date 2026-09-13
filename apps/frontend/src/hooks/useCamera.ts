import { useState, useEffect, useRef, useCallback } from "react";
import {
    requestCameraStream,
    stopCameraStream,
    type CameraErrorDetails,
    type CameraStreamOptions,
} from "../lib/media/camera";

export interface UseCameraResult {
    stream: MediaStream | null;
    videoRef: React.RefObject<HTMLVideoElement | null>;
    isLoading: boolean;
    error: CameraErrorDetails | null;
    facingMode: "user" | "environment";
    startCamera: () => Promise<void>;
    stopCamera: () => void;
    toggleFacingMode: () => void;
    retry: () => Promise<void>;
}

export function useCamera(options: CameraStreamOptions = {}): UseCameraResult {
    const [stream, setStream] = useState<MediaStream | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<CameraErrorDetails | null>(null);
    const [facingMode, setFacingMode] = useState<"user" | "environment">(options.facingMode ?? "user");

    const videoRef = useRef<HTMLVideoElement | null>(null);
    const streamRef = useRef<MediaStream | null>(null);

    const stopCamera = useCallback(() => {
        if (streamRef.current) {
            stopCameraStream(streamRef.current);
            streamRef.current = null;
            setStream(null);
        }
    }, []);

    const startCamera = useCallback(async () => {
        setIsLoading(true);
        setError(null);
        stopCamera();

        try {
            const newStream = await requestCameraStream({
                facingMode,
                idealWidth: options.idealWidth ?? 1280,
                idealHeight: options.idealHeight ?? 720,
            });

            streamRef.current = newStream;
            setStream(newStream);

            if (videoRef.current) {
                videoRef.current.srcObject = newStream;
                try {
                    await videoRef.current.play();
                } catch {
                    // Playback might need user gesture in some browsers
                }
            }
        } catch (err: unknown) {
            const camErr = err as CameraErrorDetails;
            setError(camErr);
        } finally {
            setIsLoading(false);
        }
    }, [facingMode, options.idealWidth, options.idealHeight, stopCamera]);

    const toggleFacingMode = useCallback(() => {
        setFacingMode((prev) => (prev === "user" ? "environment" : "user"));
    }, []);

    const retry = useCallback(async () => {
        await startCamera();
    }, [startCamera]);

    // Handle stream lifecycle when video element attaches
    useEffect(() => {
        if (videoRef.current && stream) {
            videoRef.current.srcObject = stream;
            videoRef.current.play().catch(() => { });
        }
    }, [stream]);

    // Initial startup
    useEffect(() => {
        let mounted = true;

        const run = async () => {
            await Promise.resolve();
            if (!mounted) return;
            await startCamera();
            if (!mounted) {
                stopCamera();
            }
        };

        void run();

        return () => {
            mounted = false;
            stopCamera();
        };
    }, [facingMode, startCamera, stopCamera]);

    return {
        stream,
        videoRef,
        isLoading,
        error,
        facingMode,
        startCamera,
        stopCamera,
        toggleFacingMode,
        retry,
    };
}
