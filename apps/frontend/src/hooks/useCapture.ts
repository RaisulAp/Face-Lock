import { useState, useCallback } from "react";
import { captureVideoFrame, type CaptureOptions, type CaptureResult } from "../lib/media/capture";
import { evaluateVideoLuminance, type LuminanceResult } from "../lib/media/luminance";

export interface UseCaptureOptions extends CaptureOptions {
    checkLuminance?: boolean;
}

export interface UseCaptureResult {
    captured: CaptureResult | null;
    luminanceInfo: LuminanceResult | null;
    isCapturing: boolean;
    error: string | null;
    takePhoto: (video: HTMLVideoElement | null) => Promise<CaptureResult | null>;
    retakePhoto: () => void;
    clearPhoto: () => void;
}

export function useCapture(options: UseCaptureOptions = {}): UseCaptureResult {
    const [captured, setCaptured] = useState<CaptureResult | null>(null);
    const [luminanceInfo, setLuminanceInfo] = useState<LuminanceResult | null>(null);
    const [isCapturing, setIsCapturing] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);

    const takePhoto = useCallback(
        async (video: HTMLVideoElement | null): Promise<CaptureResult | null> => {
            if (!video) {
                setError("Elemen kamera tidak ditemukan.");
                return null;
            }

            setIsCapturing(true);
            setError(null);

            try {
                if (options.checkLuminance !== false) {
                    const lum = evaluateVideoLuminance(video);
                    setLuminanceInfo(lum);
                }

                const result = await captureVideoFrame(video, {
                    maxBytes: options.maxBytes,
                    maxWidth: options.maxWidth,
                    initialQuality: options.initialQuality,
                    mimeType: options.mimeType,
                });

                setCaptured(result);
                return result;
            } catch (err: unknown) {
                const msg = err instanceof Error ? err.message : "Gagal mengambil foto dari kamera.";
                setError(msg);
                return null;
            } finally {
                setIsCapturing(false);
            }
        },
        [options],
    );

    const retakePhoto = useCallback(() => {
        if (captured?.dataUrl) {
            URL.revokeObjectURL(captured.dataUrl);
        }
        setCaptured(null);
        setLuminanceInfo(null);
        setError(null);
    }, [captured]);

    const clearPhoto = useCallback(() => {
        retakePhoto();
    }, [retakePhoto]);

    return {
        captured,
        luminanceInfo,
        isCapturing,
        error,
        takePhoto,
        retakePhoto,
        clearPhoto,
    };
}
