import React from "react";

export type FrameStatus = "idle" | "capturing" | "success" | "warning" | "error";

interface FaceFrameOverlayProps {
    status?: FrameStatus;
    hintMessage?: string | null;
}

export const FaceFrameOverlay: React.FC<FaceFrameOverlayProps> = ({ status = "idle", hintMessage }) => {
    let borderColor = "border-white/70";
    let glowColor = "shadow-none";

    if (status === "capturing") {
        borderColor = "border-indigo-400";
        glowColor = "shadow-[0_0_25px_rgba(99,102,241,0.5)]";
    } else if (status === "success") {
        borderColor = "border-emerald-400";
        glowColor = "shadow-[0_0_30px_rgba(52,211,153,0.6)]";
    } else if (status === "warning") {
        borderColor = "border-amber-400";
        glowColor = "shadow-[0_0_25px_rgba(251,191,36,0.5)]";
    } else if (status === "error") {
        borderColor = "border-rose-500";
        glowColor = "shadow-[0_0_30px_rgba(244,63,94,0.6)]";
    }

    return (
        <div className="absolute inset-0 pointer-events-none flex flex-col items-center justify-center p-4">
            {/* Semi-transparent dark vignette mask outside the oval */}
            <div className="relative w-full max-w-[280px] sm:max-w-[320px] aspect-[3/4] flex items-center justify-center">
                {/* The Oval Boundary */}
                <div
                    className={`w-full h-full rounded-[48%] border-2 sm:border-3 transition-all duration-300 ${borderColor} ${glowColor} flex flex-col items-center justify-between p-6`}
                >
                    {/* Top guide: forehead / eyes alignment indicator */}
                    <div className="w-16 h-1 rounded-full bg-white/40 mt-4" />

                    {/* Center alignment crosshair (subtle) */}
                    <div className="relative w-8 h-8 flex items-center justify-center opacity-30">
                        <div className="absolute w-full h-[1px] bg-white" />
                        <div className="absolute h-full w-[1px] bg-white" />
                    </div>

                    {/* Bottom guide: chin indicator */}
                    <div className="w-12 h-1 rounded-full bg-white/40 mb-4" />
                </div>
            </div>

            {/* Real-time coaching message below or within frame */}
            {hintMessage && (
                <div className="mt-4 px-4 py-2 rounded-full bg-slate-900/80 backdrop-blur-sm text-white text-xs sm:text-sm font-medium text-center shadow-lg border border-white/20 animate-fade-in max-w-[90%]">
                    {hintMessage}
                </div>
            )}
        </div>
    );
};
