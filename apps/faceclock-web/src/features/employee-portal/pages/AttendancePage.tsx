import React from "react";
import { useNavigate } from "react-router-dom";
import { LogIn, LogOut, AlertTriangle, UserCheck, ShieldAlert, Loader2, RefreshCw } from "lucide-react";
import { useCheckInFlow } from "../hooks/useCheckInFlow";
import { TodayStatusCard } from "../components/TodayStatusCard";
import { AttendanceResultCard } from "../components/AttendanceResultCard";
import { FallbackModal } from "../components/FallbackModal";
import { LocationGate } from "../../../components/location/LocationGate";
import { LocationStatus } from "../../../components/location/LocationStatus";
import { CameraGate } from "../../../components/camera/CameraGate";
import { CameraCapture } from "../../../components/camera/CameraCapture";
import { CapturePreview } from "../../../components/camera/CapturePreview";
import { HintCoach } from "../../../components/feedback/HintCoach";

export const AttendancePage: React.FC = () => {
    const navigate = useNavigate();
    const flow = useCheckInFlow();

    // If user hasn't enrolled face biometrics yet
    if (!flow.isLoadingContext && flow.context && !flow.context.has_enrolled_face) {
        return (
            <div className="max-w-2xl mx-auto space-y-6">
                <TodayStatusCard context={flow.context} isLoading={flow.isLoadingContext} />

                <div className="bg-white rounded-2xl p-6 sm:p-8 border border-amber-200 shadow-sm text-center space-y-4">
                    <div className="w-16 h-16 rounded-full bg-amber-100 text-amber-700 mx-auto flex items-center justify-center">
                        <UserCheck className="w-8 h-8" />
                    </div>
                    <div className="space-y-1">
                        <h2 className="text-xl font-bold text-slate-900">Biometrik Wajah Belum Terdaftar</h2>
                        <p className="text-sm text-slate-600 max-w-md mx-auto">
                            Anda belum memiliki data biometrik wajah terverifikasi. Untuk melakukan absensi mandiri, silakan
                            lakukan persetujuan konsen dan registrasi wajah terlebih dahulu.
                        </p>
                    </div>
                    <div className="pt-2">
                        <button
                            type="button"
                            onClick={() => navigate("/portal/enrollment")}
                            className="inline-flex items-center justify-center px-6 py-3 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-medium text-sm transition-colors shadow-xs"
                        >
                            Mulai Registrasi Wajah
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    // If attendance result is ready
    if (flow.lastResult) {
        return (
            <div className="max-w-2xl mx-auto space-y-6">
                <TodayStatusCard context={flow.context} isLoading={flow.isLoadingContext} />
                <AttendanceResultCard
                    result={flow.lastResult}
                    actionType={flow.actionType}
                    onReset={flow.resetFlow}
                    onViewHistory={() => navigate("/portal/history")}
                />
            </div>
        );
    }

    const isCheckInDisabled = flow.actionType === "check_in" && flow.context && !flow.context.can_check_in;
    const isCheckOutDisabled = flow.actionType === "check_out" && flow.context && !flow.context.can_check_out;

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            {/* 1. Today Status Card */}
            <TodayStatusCard context={flow.context} isLoading={flow.isLoadingContext} />

            {/* 2. Action Type Switcher (Check-In vs Check-Out) */}
            <div className="bg-white rounded-2xl p-2 border border-slate-200 shadow-xs flex items-center space-x-2">
                <button
                    type="button"
                    onClick={() => flow.setActionType("check_in")}
                    className={`flex-1 py-3 px-4 rounded-xl flex items-center justify-center space-x-2 text-sm font-semibold transition-all ${flow.actionType === "check_in"
                            ? "bg-primary-600 text-white shadow-sm"
                            : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                        }`}
                >
                    <LogIn className="w-4 h-4" />
                    <span>Check-In (Masuk)</span>
                </button>

                <button
                    type="button"
                    onClick={() => flow.setActionType("check_out")}
                    className={`flex-1 py-3 px-4 rounded-xl flex items-center justify-center space-x-2 text-sm font-semibold transition-all ${flow.actionType === "check_out"
                            ? "bg-primary-600 text-white shadow-sm"
                            : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                        }`}
                >
                    <LogOut className="w-4 h-4" />
                    <span>Check-Out (Pulang)</span>
                </button>
            </div>

            {/* Notice if action is normally unavailable */}
            {isCheckInDisabled && (
                <div className="bg-amber-50 border border-amber-200 rounded-xl p-3.5 flex items-start space-x-3 text-xs text-amber-800">
                    <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600 mt-0.5" />
                    <div>
                        <p className="font-semibold">Check-In Sudah Tercatat Hari Ini</p>
                        <p>
                            Anda telah melakukan check-in sebelumnya. Jika perlu mengubah atau memperbaiki data, silakan
                            hubungi admin HR.
                        </p>
                    </div>
                </div>
            )}

            {isCheckOutDisabled && flow.context && !flow.context.today_check_in && (
                <div className="bg-amber-50 border border-amber-200 rounded-xl p-3.5 flex items-start space-x-3 text-xs text-amber-800">
                    <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600 mt-0.5" />
                    <div>
                        <p className="font-semibold">Belum Melakukan Check-In</p>
                        <p>Anda tidak dapat melakukan check-out sebelum tercatat check-in pada tanggal kerja hari ini.</p>
                    </div>
                </div>
            )}

            {/* 3. Location Gate & Status */}
            <LocationGate isLoading={flow.geo.isLoading} error={flow.geo.error} onRetry={flow.geo.refreshLocation}>
                <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-xs space-y-4">
                    <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                        <h3 className="text-sm font-bold text-slate-800">Lokasi Presensi</h3>
                        <button
                            type="button"
                            onClick={flow.geo.refreshLocation}
                            className="text-xs text-primary-600 hover:text-primary-700 font-medium inline-flex items-center"
                        >
                            <RefreshCw className="w-3 h-3 mr-1" />
                            Perbarui GPS
                        </button>
                    </div>
                    <LocationStatus
                        isLoading={flow.geo.isLoading}
                        error={flow.geo.error}
                        nearest={flow.geo.nearest}
                        onRefresh={flow.geo.refreshLocation}
                    />
                </div>

                {/* 4. Camera & Capture Stream */}
                <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-xs space-y-4">
                    <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                        <div>
                            <h3 className="text-sm font-bold text-slate-800">Verifikasi Wajah Biometrik</h3>
                            <p className="text-xs text-slate-500">
                                Posisikan wajah Anda tegak lurus di dalam bingkai oval.
                            </p>
                        </div>
                        {flow.consecutiveFailures >= 3 && (
                            <button
                                type="button"
                                onClick={() => flow.setIsFallbackModalOpen(true)}
                                className="text-xs text-amber-700 bg-amber-50 hover:bg-amber-100 border border-amber-200 px-2.5 py-1.5 rounded-lg font-semibold inline-flex items-center"
                            >
                                <ShieldAlert className="w-3.5 h-3.5 mr-1 text-amber-600" />
                                Opsi Fallback
                            </button>
                        )}
                    </div>

                    <CameraGate isLoading={flow.camera.isLoading} error={flow.camera.error} onRetry={flow.camera.retry}>
                        {flow.capturedPreviewUrl ? (
                            /* Photo Preview Step */
                            <div className="space-y-4">
                                <CapturePreview
                                    capture={flow.capture.captured}
                                    previewUrl={flow.capturedPreviewUrl}
                                    onRetake={flow.onRetakePhoto}
                                    disabled={flow.isSubmitting}
                                />

                                {/* Hints Coach */}
                                <HintCoach hints={flow.activeHints} />

                                {/* Error Banner */}
                                {flow.submitError && (
                                    <div className="bg-rose-50 border border-rose-200 rounded-xl p-3.5 text-xs text-rose-800 flex items-start space-x-2.5">
                                        <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5 text-rose-600" />
                                        <div className="flex-1">
                                            <p className="font-semibold">Presensi Belum Berhasil</p>
                                            <p>{flow.submitError}</p>
                                        </div>
                                    </div>
                                )}

                                {/* Submit Action Buttons */}
                                <div className="flex gap-3 pt-2">
                                    <button
                                        type="button"
                                        onClick={flow.onRetakePhoto}
                                        disabled={flow.isSubmitting}
                                        className="flex-1 py-3 px-4 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 font-semibold text-sm transition-colors disabled:opacity-50"
                                    >
                                        Foto Ulang
                                    </button>

                                    <button
                                        type="button"
                                        onClick={() => flow.submitAttendance()}
                                        disabled={flow.isSubmitting}
                                        className="flex-1 py-3 px-4 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-semibold text-sm transition-colors shadow-sm flex items-center justify-center space-x-2 disabled:opacity-50"
                                    >
                                        {flow.isSubmitting ? (
                                            <>
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                                <span>Memverifikasi...</span>
                                            </>
                                        ) : (
                                            <span>Kirim {flow.actionType === "check_in" ? "Check-In" : "Check-Out"}</span>
                                        )}
                                    </button>
                                </div>
                            </div>
                        ) : (
                            /* Live Camera Capture Step */
                            <div className="space-y-4">
                                <CameraCapture
                                    videoRef={flow.camera.videoRef}
                                    stream={flow.camera.stream}
                                    isCapturing={flow.capture.isCapturing}
                                    disabled={Boolean(isCheckInDisabled || isCheckOutDisabled)}
                                    onCapture={flow.takePhoto}
                                    onToggleFacingMode={flow.camera.toggleFacingMode}
                                    captureButtonLabel={
                                        flow.actionType === "check_in" ? "Ambil Foto Masuk" : "Ambil Foto Pulang"
                                    }
                                    externalHint={flow.activeHints[0]}
                                />

                                {flow.consecutiveFailures >= 3 && (
                                    <div className="bg-amber-50 border border-amber-200 rounded-xl p-3.5 flex items-start justify-between text-xs text-amber-800">
                                        <div className="space-y-1 mr-3">
                                            <p className="font-semibold">Kesulitan Verifikasi Biometrik?</p>
                                            <p>
                                                Sistem mendeteksi beberapa kali kendala pencocokan wajah. Anda dapat menggunakan opsi
                                                pengajuan fallback.
                                            </p>
                                        </div>
                                        <button
                                            type="button"
                                            onClick={() => flow.setIsFallbackModalOpen(true)}
                                            className="shrink-0 px-3 py-1.5 bg-amber-600 hover:bg-amber-700 text-white rounded-lg font-medium"
                                        >
                                            Buka Fallback
                                        </button>
                                    </div>
                                )}
                            </div>
                        )}
                    </CameraGate>
                </div>
            </LocationGate>

            {/* Fallback Modal */}
            <FallbackModal
                isOpen={flow.isFallbackModalOpen}
                onClose={() => flow.setIsFallbackModalOpen(false)}
                onSubmitFallback={(reason, notes) =>
                    flow.submitAttendance({
                        allowFallback: true,
                        fallbackReason: reason,
                        notes,
                    })
                }
                isSubmitting={flow.isSubmitting}
            />
        </div>
    );
};
