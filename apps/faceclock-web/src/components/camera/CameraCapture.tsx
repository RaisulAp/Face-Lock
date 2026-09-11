import React, { useEffect, useState } from "react";
import { Camera, SwitchCamera, Sun, Moon } from "lucide-react";
import { FaceFrameOverlay, type FrameStatus } from "./FaceFrameOverlay";
import { evaluateVideoLuminance, type LuminanceResult } from "../../lib/media/luminance";

interface CameraCaptureProps {
    videoRef: React.RefObject<HTMLVideoElement | null>;
    stream: MediaStream | null;
    isCapturing?: boolean;
    disabled?: boolean;
    captureButtonLabel?: string;
    onCapture: () => void;
    onToggleFacingMode?: () => void;
    frameStatus?: FrameStatus;
    externalHint?: string | null;
}

export const CameraCapture: React.FC<CameraCaptureProps> = ({
    videoRef,
    stream,
    isCapturing = false,
    disabled = false,
    captureButtonLabel = "Ketuk untuk mengambil foto",
    onCapture,
    onToggleFacingMode,
    frameStatus = "idle",
    externalHint,
}) => {
    const [luminance, setLuminance] = useState<LuminanceResult | null>(null);

    // Periodic luminance advisory checking (D25)
    useEffect(() => {
        if (!videoRef.current || !stream) return;

        const interval = setInterval(() => {
            if (videoRef.current) {
                const res = evaluateVideoLuminance(videoRef.current);
                setLuminance(res);
            }
        }, 1500);

        return () => clearInterval(interval);
    }, [videoRef, stream]);

    // Combined hint: prefer external hint (e.g. from backend inference failure) over local advisory
    const activeHint = externalHint || luminance?.advisoryText;

    return (
        <div className="relative w-full max-w-md mx-auto aspect-[3/4] sm:aspect-[4/5] rounded-3xl overflow-hidden bg-black shadow-2xl flex flex-col justify-between select-none">
            {/* 
        CRITICAL ARCHITECTURAL RULE:
        CSS transform: scaleX(-1) mirrors the visual preview for intuitive selfie UX.
        The canvas capture itself will capture natural unmirrored pixels.
      */}
            <video
                ref={videoRef}
                autoPlay
                playsInline
                muted
                className="absolute inset-0 w-full h-full object-cover -scale-x-100"
            />

            {/* Face Frame & Alignment Overlay */}
            <FaceFrameOverlay status={isCapturing ? "capturing" : frameStatus} hintMessage={activeHint} />

            {/* Top Action Bar */}
            <div className="relative z-10 w-full p-4 flex items-center justify-between pointer-events-auto">
                {/* Lighting Indicator Badge */}
                {luminance?.advisoryCode ? (
                    <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-amber-500/80 backdrop-blur-md text-slate-900 text-xs font-semibold shadow-md">
                        {luminance.advisoryCode === "too_dark" ? (
                            <Moon className="w-3.5 h-3.5 text-slate-900" />
                        ) : (
                            <Sun className="w-3.5 h-3.5 text-slate-900" />
                        )}
                        <span>{luminance.advisoryCode === "too_dark" ? "Kurang Terang" : "Terlalu Silau"}</span>
                    </div>
                ) : (
                    <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-slate-900/60 backdrop-blur-md text-white/80 text-[11px] font-medium">
                        <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                        Kamera Siap
                    </div>
                )}

                {/* Flip camera toggle button */}
                {onToggleFacingMode && (
                    <button
                        type="button"
                        onClick={onToggleFacingMode}
                        className="p-2.5 rounded-full bg-slate-900/60 hover:bg-slate-900/80 backdrop-blur-md text-white transition-all shadow-md active:scale-95"
                        title="Ganti Kamera"
                    >
                        <SwitchCamera className="w-5 h-5" />
                    </button>
                )}
            </div>

            {/* Bottom Shutter Action Bar */}
            <div className="relative z-10 w-full pb-6 pt-2 flex flex-col items-center justify-center pointer-events-auto">
                <button
                    type="button"
                    onClick={onCapture}
                    disabled={isCapturing || disabled}
                    className="relative group p-1.5 rounded-full border-4 border-white/80 hover:border-white transition-all shadow-lg active:scale-95 disabled:opacity-50 disabled:pointer-events-none"
                    title="Ambil Foto"
                >
                    <div className="w-16 h-16 rounded-full bg-white group-hover:bg-indigo-50 flex items-center justify-center transition-all">
                        {isCapturing ? (
                            <div className="w-6 h-6 rounded-full border-3 border-indigo-600 border-t-transparent animate-spin" />
                        ) : (
                            <Camera className="w-7 h-7 text-indigo-600" />
                        )}
                    </div>
                </button>
                <span className="text-white/80 text-xs font-medium mt-2 tracking-wide">{captureButtonLabel}</span>
            </div>
        </div>
    );
};
