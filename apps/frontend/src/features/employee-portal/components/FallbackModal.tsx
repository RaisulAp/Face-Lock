import React, { useState } from "react";
import { AlertCircle, ShieldAlert, X } from "lucide-react";

interface FallbackModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSubmitFallback: (reason: string, notes: string) => void;
    isSubmitting?: boolean;
}

const FALLBACK_REASONS = [
    { id: "face_not_recognized", label: "Wajah Tidak Dikenali Sistem" },
    { id: "poor_lighting", label: "Pencahayaan Lokasi Tidak Mendukung" },
    { id: "camera_issue", label: "Kendala Teknis Kamera / Perangkat" },
    { id: "appearance_change", label: "Perubahan Penampilan Fisik / Medis" },
    { id: "other", label: "Alasan Lainnya" },
];

export const FallbackModal: React.FC<FallbackModalProps> = ({
    isOpen,
    onClose,
    onSubmitFallback,
    isSubmitting = false,
}) => {
    const [selectedReason, setSelectedReason] = useState(FALLBACK_REASONS[0]?.id || "camera_broken");
    const [notes, setNotes] = useState("");
    const [error, setError] = useState<string | null>(null);

    if (!isOpen) return null;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (notes.trim().length < 5) {
            setError("Mohon berikan penjelasan minimal 5 karakter mengenai kendala Anda.");
            return;
        }
        setError(null);
        onSubmitFallback(selectedReason, notes.trim());
    };

    return (
        <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl shadow-xl border border-slate-200 w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-200">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
                    <div className="flex items-center space-x-2.5">
                        <div className="w-8 h-8 rounded-lg bg-amber-100 flex items-center justify-center text-amber-700">
                            <ShieldAlert className="w-5 h-5" />
                        </div>
                        <div>
                            <h3 className="font-bold text-slate-900 text-base">Ajukan Presensi Fallback</h3>
                            <p className="text-xs text-slate-500">Verifikasi manual oleh atasan</p>
                        </div>
                    </div>
                    <button
                        type="button"
                        onClick={onClose}
                        disabled={isSubmitting}
                        className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-100 transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Content & Form */}
                <form onSubmit={handleSubmit} className="p-6 space-y-4">
                    <div className="bg-amber-50 border border-amber-200 rounded-xl p-3.5 text-xs text-amber-800 flex items-start space-x-2.5">
                        <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-amber-700" />
                        <p>
                            Presensi fallback tetap menggunakan tangkapan foto Anda, namun status akan menjadi{" "}
                            <strong>Menunggu Verifikasi (Pending Review)</strong> sampai disetujui secara manual oleh
                            Admin/Atasan.
                        </p>
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-2">Pilih Alasan Kendala:</label>
                        <div className="space-y-2">
                            {FALLBACK_REASONS.map((r) => (
                                <label
                                    key={r.id}
                                    className={`flex items-center p-2.5 rounded-lg border text-xs cursor-pointer transition-colors ${selectedReason === r.id
                                            ? "border-primary-500 bg-primary-50/50 text-primary-900 font-medium"
                                            : "border-slate-200 hover:bg-slate-50 text-slate-700"
                                        }`}
                                >
                                    <input
                                        type="radio"
                                        name="fallback_reason"
                                        value={r.id}
                                        checked={selectedReason === r.id}
                                        onChange={() => setSelectedReason(r.id)}
                                        className="mr-2.5 text-primary-600 focus:ring-primary-500"
                                    />
                                    {r.label}
                                </label>
                            ))}
                        </div>
                    </div>

                    <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                            Catatan Tambahan: <span className="text-rose-500">*</span>
                        </label>
                        <textarea
                            rows={3}
                            value={notes}
                            onChange={(e) => {
                                setNotes(e.target.value);
                                if (error) setError(null);
                            }}
                            placeholder="Jelaskan secara singkat kendala yang Anda alami..."
                            className="w-full text-xs rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 text-slate-800 placeholder-slate-400"
                        />
                        {error && <p className="text-xs text-rose-600 mt-1 font-medium">{error}</p>}
                    </div>

                    {/* Action Buttons */}
                    <div className="flex items-center justify-end space-x-3 pt-2">
                        <button
                            type="button"
                            onClick={onClose}
                            disabled={isSubmitting}
                            className="px-4 py-2 rounded-xl text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 transition-colors"
                        >
                            Batal
                        </button>
                        <button
                            type="submit"
                            disabled={isSubmitting}
                            className="px-4 py-2 rounded-xl text-xs font-medium text-white bg-primary-600 hover:bg-primary-700 transition-colors shadow-xs flex items-center space-x-2 disabled:opacity-50"
                        >
                            {isSubmitting ? <span>Mengirim...</span> : <span>Ajukan Presensi</span>}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};
