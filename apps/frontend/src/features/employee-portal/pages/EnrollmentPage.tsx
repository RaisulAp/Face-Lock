import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    ShieldCheck,
    AlertTriangle,
    UserCheck,
    Camera,
    CheckCircle2,
    Loader2,
    ArrowRight,
    RotateCcw,
    Sparkles,
} from "lucide-react";
import { employeeApi } from "../api";
import { EnrollmentSlotGrid } from "../components/EnrollmentSlotGrid";
import { EnrollmentSessionTimer } from "../components/EnrollmentSessionTimer";
import { CameraGate } from "../../../components/camera/CameraGate";
import { CameraCapture } from "../../../components/camera/CameraCapture";
import { HintCoach } from "../../../components/feedback/HintCoach";
import { useCamera } from "../../../hooks/useCamera";
import { useCapture } from "../../../hooks/useCapture";
import { ApiError } from "../../../lib/api";
import type { CommitSessionResult, EnrollmentSessionData } from "../types";

export const EnrollmentPage: React.FC = () => {
    const navigate = useNavigate();
    const queryClient = useQueryClient();

    // 1. Fetch Enrollment & Consent Status
    const {
        data: statusData,
        isLoading: isLoadingStatus,
        refetch: refetchStatus,
    } = useQuery({
        queryKey: ["enrollment-status-me"],
        queryFn: employeeApi.getMyEnrollmentStatus,
    });

    // Active Session State
    const [activeSession, setActiveSession] = useState<EnrollmentSessionData | null>(null);
    const [activeSlotIndex, setActiveSlotIndex] = useState<number>(0);
    const [isUploadingPhoto, setIsUploadingPhoto] = useState(false);
    const [uploadError, setUploadError] = useState<string | null>(null);
    const [uploadHints, setUploadHints] = useState<string[]>([]);
    const [commitResult, setCommitResult] = useState<CommitSessionResult | null>(null);
    const [isDuplicateModalOpen, setIsDuplicateModalOpen] = useState(false);
    const [duplicateReason, setDuplicateReason] = useState("");

    // Camera & Capture Pipeline
    const camera = useCamera({ facingMode: "user" });
    const capture = useCapture({ maxBytes: 800 * 1024, maxWidth: 960 });

    // Start Enrollment Session Mutation
    const startSessionMutation = useMutation({
        mutationFn: () => employeeApi.createEnrollmentSession("replace"),
        onSuccess: (session) => {
            setActiveSession(session);
            setActiveSlotIndex(session.photos.length);
            setUploadError(null);
            setUploadHints([]);
            setCommitResult(null);
        },
    });

    // Cancel Session Mutation
    const cancelSessionMutation = useMutation({
        mutationFn: (sessionId: string) => employeeApi.cancelEnrollmentSession(sessionId),
        onSuccess: () => {
            setActiveSession(null);
            refetchStatus();
        },
    });

    // Commit Session Mutation
    const commitSessionMutation = useMutation({
        mutationFn: ({ force, reason }: { force?: boolean; reason?: string }) => {
            if (!activeSession) throw new Error("Tidak ada sesi aktif");
            return employeeApi.commitEnrollmentSession(activeSession.id, force, reason);
        },
        onSuccess: (result) => {
            setCommitResult(result);
            setActiveSession(null);
            setIsDuplicateModalOpen(false);
            queryClient.invalidateQueries({ queryKey: ["enrollment-status-me"] });
            queryClient.invalidateQueries({ queryKey: ["attendance-context"] });
        },
        onError: (err: unknown) => {
            if (err instanceof ApiError && err.code === "FACE_BELONGS_TO_ANOTHER_EMPLOYEE") {
                setIsDuplicateModalOpen(true);
            } else {
                setUploadError(err instanceof Error ? err.message : "Gagal menyelesaikan sesi pendaftaran");
            }
        },
    });

    // Handle Capture & Upload
    const handlePhotoCaptured = async (blob: Blob) => {
        if (!activeSession) return;
        setIsUploadingPhoto(true);
        setUploadError(null);
        setUploadHints([]);

        try {
            await employeeApi.uploadEnrollmentPhoto(activeSession.id, blob, "web_camera");
            // Refetch current session to get updated photos
            const updated = await employeeApi.getEnrollmentSession(activeSession.id);
            setActiveSession(updated);

            // Advance to next slot
            if (updated.photos.length < updated.required_photos) {
                setActiveSlotIndex(updated.photos.length);
            }
        } catch (err: unknown) {
            if (err instanceof ApiError) {
                setUploadError(err.message);
                if (err.hints && err.hints.length > 0) {
                    setUploadHints(err.hints);
                }
            } else {
                setUploadError(err instanceof Error ? err.message : "Gagal mengunggah foto wajah");
            }
        } finally {
            setIsUploadingPhoto(false);
        }
    };

    const handleTriggerCapture = async () => {
        if (!camera.videoRef.current || !activeSession) return;
        const res = await capture.takePhoto(camera.videoRef.current);
        if (res?.blob) {
            await handlePhotoCaptured(res.blob);
        }
    };

    // Handle Delete Photo from Slot
    const handleDeletePhoto = async (photoId: string) => {
        if (!activeSession) return;
        try {
            await employeeApi.deleteEnrollmentPhoto(activeSession.id, photoId);
            const updated = await employeeApi.getEnrollmentSession(activeSession.id);
            setActiveSession(updated);
            setActiveSlotIndex(updated.photos.length);
        } catch (err: unknown) {
            setUploadError(err instanceof Error ? err.message : "Gagal menghapus foto");
        }
    };

    // Loading State
    if (isLoadingStatus) {
        return (
            <div className="max-w-2xl mx-auto p-8 text-center space-y-3">
                <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
                <p className="text-sm text-slate-500">Memeriksa status pendaftaran biometrik...</p>
            </div>
        );
    }

    // 1. Check Biometric Consent First (UU PDP No. 27/2022)
    const consentGranted = statusData?.consent?.status === "granted" && statusData?.consent?.is_current_version;

    if (!consentGranted) {
        return (
            <div className="max-w-2xl mx-auto space-y-6">
                <div className="bg-white rounded-2xl p-6 sm:p-8 border border-amber-200 shadow-sm text-center space-y-4">
                    <div className="w-16 h-16 rounded-full bg-amber-100 text-amber-700 mx-auto flex items-center justify-center">
                        <ShieldCheck className="w-8 h-8" />
                    </div>
                    <div className="space-y-1">
                        <h2 className="text-xl font-bold text-slate-900">Persetujuan Biometrik Diperlukan</h2>
                        <p className="text-sm text-slate-600 max-w-md mx-auto">
                            Berdasarkan UU Pelindungan Data Pribadi (UU PDP No. 27/2022), Anda wajib membaca dan menyetujui
                            Lembar Persetujuan Pemrosesan Data Biometrik sebelum dapat merekam wajah.
                        </p>
                    </div>
                    <div className="pt-2">
                        <button
                            type="button"
                            onClick={() => navigate("/portal/consent")}
                            className="inline-flex items-center justify-center px-6 py-3 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-medium text-sm transition-colors shadow-xs"
                        >
                            Baca & Setujui Lembar Konsen
                            <ArrowRight className="w-4 h-4 ml-2" />
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    // 2. Completed State
    if (commitResult) {
        return (
            <div className="max-w-2xl mx-auto space-y-6">
                <div className="bg-white rounded-2xl p-8 border border-emerald-200 shadow-sm text-center space-y-5">
                    <div className="w-16 h-16 rounded-full bg-emerald-100 text-emerald-700 mx-auto flex items-center justify-center">
                        <CheckCircle2 className="w-10 h-10" />
                    </div>
                    <div className="space-y-2">
                        <h2 className="text-2xl font-bold text-slate-900">Registrasi Wajah Berhasil!</h2>
                        <p className="text-sm text-slate-600 max-w-md mx-auto">
                            Data biometrik wajah Anda telah tersimpan dengan aman dan siap digunakan untuk absensi harian.
                        </p>
                    </div>

                    <div className="bg-emerald-50 rounded-xl p-4 max-w-sm mx-auto text-left text-xs text-emerald-900 space-y-2 border border-emerald-100">
                        <div className="flex justify-between">
                            <span className="text-emerald-700">Foto Referensi Aktif:</span>
                            <strong className="font-bold">{commitResult.active_reference_count} Foto</strong>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-emerald-700">Status Akun:</span>
                            <span className="font-semibold text-emerald-800">Siap Absensi</span>
                        </div>
                    </div>

                    <div className="flex flex-col sm:flex-row gap-3 justify-center pt-2">
                        <button
                            type="button"
                            onClick={() => setCommitResult(null)}
                            className="px-5 py-2.5 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-medium transition-colors"
                        >
                            Kembali ke Menu
                        </button>
                        <button
                            type="button"
                            onClick={() => navigate("/portal/attendance")}
                            className="px-6 py-2.5 rounded-xl bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium transition-colors shadow-xs inline-flex items-center justify-center"
                        >
                            Mulai Absensi Hari Ini
                            <ArrowRight className="w-4 h-4 ml-2" />
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    // 3. Active Session Mode
    if (activeSession) {
        const uploadedPhotos = activeSession.photos || [];
        const requiredPhotos = activeSession.required_photos || 3;
        const canCommit = uploadedPhotos.length >= requiredPhotos;
        const isSessionSlotFinished = activeSlotIndex >= requiredPhotos;

        return (
            <div className="max-w-2xl mx-auto space-y-6">
                {/* Session Top Bar */}
                <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-xs flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center space-x-2.5">
                        <div className="w-8 h-8 rounded-lg bg-primary-50 text-primary-700 flex items-center justify-center font-bold text-sm">
                            <Camera className="w-4 h-4" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold text-slate-900">Sesi Perekaman Wajah</h2>
                            <p className="text-xs text-slate-500">
                                Lengkapi {requiredPhotos} foto referensi untuk akurasi terbaik
                            </p>
                        </div>
                    </div>

                    <div className="flex items-center space-x-2">
                        <EnrollmentSessionTimer
                            expiresAt={activeSession.expires_at}
                            onExpired={() => {
                                setActiveSession(null);
                                refetchStatus();
                            }}
                        />
                        <button
                            type="button"
                            onClick={() => cancelSessionMutation.mutate(activeSession.id)}
                            disabled={cancelSessionMutation.isPending}
                            className="text-xs text-rose-600 hover:text-rose-700 px-2.5 py-1 rounded-lg border border-rose-200 hover:bg-rose-50 font-medium transition-colors"
                        >
                            Batalkan
                        </button>
                    </div>
                </div>

                {/* Slot Grid */}
                <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-xs space-y-3">
                    <div className="flex items-center justify-between">
                        <h3 className="text-sm font-bold text-slate-800">Daftar Foto Referensi</h3>
                        <span className="text-xs font-semibold text-primary-700 bg-primary-50 px-2.5 py-0.5 rounded-full">
                            {uploadedPhotos.length} / {requiredPhotos} Terpenuhi
                        </span>
                    </div>

                    <EnrollmentSlotGrid
                        requiredPhotos={requiredPhotos}
                        maxPhotos={activeSession.max_photos || 5}
                        uploadedPhotos={uploadedPhotos}
                        activeSlotIndex={activeSlotIndex}
                        onSelectSlot={(idx) => {
                            setActiveSlotIndex(idx);
                            setUploadError(null);
                            setUploadHints([]);
                        }}
                        onDeletePhoto={handleDeletePhoto}
                        disabled={isUploadingPhoto || commitSessionMutation.isPending}
                    />
                </div>

                {/* Camera Capture Area (Active Slot) */}
                {!isSessionSlotFinished && (
                    <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-xs space-y-4">
                        <div className="border-b border-slate-100 pb-3">
                            <h3 className="text-sm font-bold text-slate-800">Perekaman Pose {activeSlotIndex + 1}</h3>
                            <p className="text-xs text-slate-500">
                                Arahkan kamera ke wajah Anda di pencahayaan yang merata.
                            </p>
                        </div>

                        <CameraGate isLoading={camera.isLoading} error={camera.error} onRetry={camera.retry}>
                            <div className="space-y-4">
                                <CameraCapture
                                    videoRef={camera.videoRef}
                                    stream={camera.stream}
                                    isCapturing={capture.isCapturing || isUploadingPhoto}
                                    disabled={isUploadingPhoto || commitSessionMutation.isPending}
                                    onCapture={handleTriggerCapture}
                                    onToggleFacingMode={camera.toggleFacingMode}
                                    captureButtonLabel={
                                        isUploadingPhoto ? "Menganalisis Kualitas..." : `Ambil Foto Pose ${activeSlotIndex + 1}`
                                    }
                                    externalHint={uploadHints[0]}
                                />

                                {/* Upload Feedback */}
                                {isUploadingPhoto && (
                                    <div className="flex items-center justify-center space-x-2 text-xs text-primary-700 font-medium py-2">
                                        <Loader2 className="w-4 h-4 animate-spin" />
                                        <span>Menganalisis kualitas biometrik foto...</span>
                                    </div>
                                )}

                                {/* Hints Coach */}
                                <HintCoach hints={uploadHints} />

                                {/* Error Banner */}
                                {uploadError && (
                                    <div className="bg-rose-50 border border-rose-200 rounded-xl p-3.5 text-xs text-rose-800 flex items-start space-x-2.5">
                                        <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5 text-rose-600" />
                                        <div>
                                            <p className="font-semibold">Foto Belum Memenuhi Standar</p>
                                            <p>{uploadError}</p>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </CameraGate>
                    </div>
                )}

                {/* Commit / Selesaikan Section */}
                <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-xs space-y-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <h3 className="text-sm font-bold text-slate-900">Simpan Perekaman Biometrik</h3>
                            <p className="text-xs text-slate-500">
                                {canCommit
                                    ? "Semua foto wajib telah terpenuhi. Klik tombol di bawah untuk menyimpan."
                                    : `Masih membutuhkan minimal ${requiredPhotos - uploadedPhotos.length} foto lagi.`}
                            </p>
                        </div>

                        <button
                            type="button"
                            onClick={() => commitSessionMutation.mutate({})}
                            disabled={!canCommit || commitSessionMutation.isPending}
                            className="px-6 py-3 rounded-xl bg-primary-600 hover:bg-primary-700 text-white text-sm font-bold transition-all shadow-xs flex items-center space-x-2 disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                            {commitSessionMutation.isPending ? (
                                <>
                                    <Loader2 className="w-4 h-4 animate-spin" />
                                    <span>Menyimpan...</span>
                                </>
                            ) : (
                                <>
                                    <Sparkles className="w-4 h-4" />
                                    <span>Selesaikan & Daftarkan</span>
                                </>
                            )}
                        </button>
                    </div>
                </div>

                {/* Duplicate Face Confirmation Modal */}
                {isDuplicateModalOpen && (
                    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
                        <div className="bg-white rounded-2xl shadow-xl border border-slate-200 w-full max-w-md p-6 space-y-4">
                            <div className="flex items-center space-x-3 text-amber-700">
                                <AlertTriangle className="w-6 h-6" />
                                <h3 className="text-base font-bold text-slate-900">Kemiripan Wajah Terdeteksi</h3>
                            </div>
                            <p className="text-xs text-slate-600 leading-relaxed">
                                Sistem mendeteksi bahwa wajah pada foto memiliki kemiripan tinggi dengan profil karyawan lain.
                                Jika ini adalah pendaftaran yang sah (misalnya kembar identik atau re-registrasi), berikan
                                alasan verifikasi untuk melanjutkan.
                            </p>

                            <div>
                                <label className="block text-xs font-semibold text-slate-700 mb-1">
                                    Alasan Verifikasi Kemiripan: <span className="text-rose-500">*</span>
                                </label>
                                <textarea
                                    rows={3}
                                    value={duplicateReason}
                                    onChange={(e) => setDuplicateReason(e.target.value)}
                                    placeholder="Contoh: Re-registrasi akun profil karyawan pribadi yang diperbarui"
                                    className="w-full text-xs rounded-lg border border-slate-300 p-2.5 focus:outline-none focus:ring-2 focus:ring-primary-500/20"
                                />
                            </div>

                            <div className="flex justify-end space-x-3 pt-2">
                                <button
                                    type="button"
                                    onClick={() => setIsDuplicateModalOpen(false)}
                                    className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-xl"
                                >
                                    Batal
                                </button>
                                <button
                                    type="button"
                                    disabled={duplicateReason.trim().length < 3 || commitSessionMutation.isPending}
                                    onClick={() =>
                                        commitSessionMutation.mutate({
                                            force: true,
                                            reason: duplicateReason.trim(),
                                        })
                                    }
                                    className="px-4 py-2 text-xs font-bold text-white bg-amber-600 hover:bg-amber-700 rounded-xl disabled:opacity-50"
                                >
                                    Lanjutkan Override
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        );
    }

    // 4. Default Idle View (Status Overview & Start Button)
    const isEnrolled = statusData?.is_enrolled;
    const activeCount = statusData?.active_reference_count || 0;

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            <div className="bg-white rounded-2xl p-6 sm:p-8 border border-slate-200 shadow-xs space-y-6">
                <div className="flex items-start justify-between">
                    <div className="flex items-center space-x-3">
                        <div
                            className={`w-12 h-12 rounded-xl flex items-center justify-center ${isEnrolled ? "bg-emerald-100 text-emerald-700" : "bg-slate-100 text-slate-600"
                                }`}
                        >
                            <UserCheck className="w-6 h-6" />
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-900">Status Biometrik Wajah</h2>
                            <p className="text-xs text-slate-500">Pendaftaran wajah untuk sistem presensi cerdas</p>
                        </div>
                    </div>

                    <span
                        className={`text-xs px-3 py-1 rounded-full font-bold ${isEnrolled ? "bg-emerald-100 text-emerald-800" : "bg-slate-100 text-slate-600"
                            }`}
                    >
                        {isEnrolled ? "Aktif Terdaftar" : "Belum Terdaftar"}
                    </span>
                </div>

                <div className="bg-slate-50 rounded-xl p-4 border border-slate-100 space-y-3 text-xs">
                    <div className="flex justify-between text-slate-600">
                        <span>Jumlah Foto Referensi Aktif:</span>
                        <strong className="text-slate-900 font-semibold">{activeCount} Foto</strong>
                    </div>
                    <div className="flex justify-between text-slate-600">
                        <span>Status Konsen Biometrik (UU PDP):</span>
                        <strong className="text-emerald-700 font-semibold flex items-center">
                            <ShieldCheck className="w-3.5 h-3.5 mr-1" />
                            Disetujui ({statusData?.consent?.document_version})
                        </strong>
                    </div>
                    {statusData?.needs_re_enrollment && (
                        <div className="bg-amber-50 border border-amber-200 p-2.5 rounded-lg text-amber-800 flex items-center space-x-2">
                            <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600" />
                            <span>Model biometrik telah diperbarui. Disarankan melakukan pembaruan foto wajah.</span>
                        </div>
                    )}
                </div>

                <div className="pt-2">
                    <button
                        type="button"
                        onClick={() => startSessionMutation.mutate()}
                        disabled={startSessionMutation.isPending}
                        className="w-full py-3.5 px-4 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-bold text-sm transition-all shadow-xs flex items-center justify-center space-x-2"
                    >
                        {startSessionMutation.isPending ? (
                            <>
                                <Loader2 className="w-4 h-4 animate-spin" />
                                <span>Menyiapkan Sesi...</span>
                            </>
                        ) : (
                            <>
                                {isEnrolled ? (
                                    <>
                                        <RotateCcw className="w-4 h-4" />
                                        <span>Perbarui Foto Biometrik Wajah</span>
                                    </>
                                ) : (
                                    <>
                                        <Camera className="w-4 h-4" />
                                        <span>Mulai Pendaftaran Wajah Baru</span>
                                    </>
                                )}
                            </>
                        )}
                    </button>
                </div>
            </div>
        </div>
    );
};
