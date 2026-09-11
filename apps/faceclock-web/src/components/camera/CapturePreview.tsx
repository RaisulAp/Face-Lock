import React from "react";
import { RefreshCw, CheckCircle2, ArrowRight } from "lucide-react";
import type { CaptureResult } from "../../lib/media/capture";

export interface CapturePreviewProps {
    capture?: CaptureResult | null;
    previewUrl?: string;
    sizeKb?: number;
    isSubmitting?: boolean;
    submitLabel?: string;
    disabled?: boolean;
    onRetake: () => void;
    onSubmit?: () => void;
}

export const CapturePreview: React.FC<CapturePreviewProps> = ({
    capture,
    previewUrl,
    sizeKb,
    isSubmitting = false,
    submitLabel = "Kirim Absensi",
    disabled = false,
    onRetake,
    onSubmit,
}) => {
    const displayUrl = capture?.dataUrl || previewUrl || "";
    const displaySizeKb = sizeKb ?? (capture?.sizeBytes ? Math.round(capture.sizeBytes / 1024) : null);
    const isActionDisabled = isSubmitting || disabled;

    return (
        <div className="w-full max-w-md mx-auto flex flex-col items-center">
            <div className="relative w-full aspect-[3/4] sm:aspect-[4/5] rounded-3xl overflow-hidden bg-black shadow-2xl border border-slate-200 dark:border-slate-800">
                <img src={displayUrl} alt="Hasil Pengambilan Foto" className="w-full h-full object-cover" />

                {/* Quality indicator badge */}
                <div className="absolute top-4 left-4 z-10 px-3 py-1 rounded-full bg-slate-900/70 backdrop-blur-md text-white text-xs font-medium border border-white/10 flex items-center gap-1.5">
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                    <span>Foto Siap {displaySizeKb ? `(${displaySizeKb} KB)` : ""}</span>
                </div>
            </div>

            {/* Action buttons (if onSubmit is provided) */}
            {onSubmit && (
                <div className="w-full grid grid-cols-2 gap-3 mt-5">
                    <button
                        type="button"
                        onClick={onRetake}
                        disabled={isActionDisabled}
                        className="flex items-center justify-center gap-2 py-3 px-4 rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-sm font-semibold transition-all shadow-sm active:scale-98 disabled:opacity-50"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Ambil Ulang
                    </button>

                    <button
                        type="button"
                        onClick={onSubmit}
                        disabled={isActionDisabled}
                        className="flex items-center justify-center gap-2 py-3 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold transition-all shadow-md hover:shadow-indigo-500/25 active:scale-98 disabled:opacity-50"
                    >
                        {isSubmitting ? (
                            <>
                                <div className="w-4 h-4 rounded-full border-2 border-white border-t-transparent animate-spin" />
                                <span>Memproses...</span>
                            </>
                        ) : (
                            <>
                                <span>{submitLabel}</span>
                                <ArrowRight className="w-4 h-4" />
                            </>
                        )}
                    </button>
                </div>
            )}
        </div>
    );
};
