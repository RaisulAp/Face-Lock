import React from "react";
import { CheckCircle2, Trash2, Camera, User, Smile, Compass } from "lucide-react";
import type { SessionPhotoItem } from "../types";

interface SlotInfo {
    position: number;
    title: string;
    description: string;
    icon: React.ReactNode;
}

const DEFAULT_SLOTS: SlotInfo[] = [
    {
        position: 1,
        title: "Pose 1: Depan Lurus",
        description: "Tatap kamera secara langsung, ekspresi netral tanpa kacamata gelap.",
        icon: <User className="w-4 h-4" />,
    },
    {
        position: 2,
        title: "Pose 2: Senyum Natural",
        description: "Tersenyum santai dan wajar agar sistem mengenali variasi otot wajah.",
        icon: <Smile className="w-4 h-4" />,
    },
    {
        position: 3,
        title: "Pose 3: Variasi Sudut",
        description: "Sedikit miringkan kepala (5°-10°) atau variasi pencahayaan ringan.",
        icon: <Compass className="w-4 h-4" />,
    },
];

interface EnrollmentSlotGridProps {
    requiredPhotos: number;
    maxPhotos: number;
    uploadedPhotos: SessionPhotoItem[];
    activeSlotIndex: number;
    onSelectSlot: (index: number) => void;
    onDeletePhoto: (photoId: string) => void;
    disabled?: boolean;
}

export const EnrollmentSlotGrid: React.FC<EnrollmentSlotGridProps> = ({
    requiredPhotos,
    uploadedPhotos,
    activeSlotIndex,
    onSelectSlot,
    onDeletePhoto,
    disabled = false,
}) => {
    const totalSlots = Math.max(requiredPhotos, DEFAULT_SLOTS.length);

    return (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {Array.from({ length: totalSlots }).map((_, idx) => {
                const slotConfig = DEFAULT_SLOTS[idx] || {
                    position: idx + 1,
                    title: `Pose ${idx + 1}: Tambahan`,
                    description: "Foto referensi tambahan untuk meningkatkan akurasi verifikasi.",
                    icon: <Camera className="w-4 h-4" />,
                };

                const photo = uploadedPhotos[idx];
                const isUploaded = Boolean(photo);
                const isActive = activeSlotIndex === idx;

                return (
                    <div
                        key={idx}
                        onClick={() => !disabled && onSelectSlot(idx)}
                        className={`relative rounded-xl p-3.5 border transition-all cursor-pointer ${isActive
                                ? "border-primary-500 bg-primary-50/40 ring-2 ring-primary-500/20"
                                : isUploaded
                                    ? "border-emerald-300 bg-emerald-50/30"
                                    : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50/60"
                            }`}
                    >
                        <div className="flex items-start justify-between">
                            <div className="flex items-center space-x-2">
                                <span
                                    className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold ${isUploaded
                                            ? "bg-emerald-600 text-white"
                                            : isActive
                                                ? "bg-primary-600 text-white"
                                                : "bg-slate-100 text-slate-600"
                                        }`}
                                >
                                    {isUploaded ? <CheckCircle2 className="w-3.5 h-3.5" /> : idx + 1}
                                </span>
                                <span className="text-xs font-bold text-slate-800">{slotConfig.title}</span>
                            </div>

                            {isUploaded && (
                                <button
                                    type="button"
                                    title="Hapus foto ini"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        if (!disabled && photo) {
                                            onDeletePhoto(photo.id);
                                        }
                                    }}
                                    className="text-slate-400 hover:text-rose-600 p-1 rounded-md hover:bg-rose-50 transition-colors"
                                >
                                    <Trash2 className="w-3.5 h-3.5" />
                                </button>
                            )}
                        </div>

                        <p className="text-[11px] text-slate-500 mt-2 line-clamp-2">{slotConfig.description}</p>

                        <div className="mt-3 pt-2 border-t border-slate-100 flex items-center justify-between text-[11px]">
                            {isUploaded ? (
                                <span className="inline-flex items-center text-emerald-700 font-semibold">
                                    <CheckCircle2 className="w-3 h-3 mr-1" />
                                    Kualitas Memenuhi Syarat
                                </span>
                            ) : isActive ? (
                                <span className="inline-flex items-center text-primary-700 font-semibold">
                                    <Camera className="w-3 h-3 mr-1" />
                                    Sedang Dipilih
                                </span>
                            ) : (
                                <span className="text-slate-400">Belum Ada Foto</span>
                            )}
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
