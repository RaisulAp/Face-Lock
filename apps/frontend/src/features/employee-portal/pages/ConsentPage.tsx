import React, { useState, useRef, useEffect, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ShieldCheck, AlertTriangle, FileText, CheckCircle2, Clock, Loader2, Lock } from "lucide-react";
import { employeeApi } from "../api";
import { renderMarkdownSafely } from "../../../lib/markdown";

export const ConsentPage: React.FC = () => {
    const queryClient = useQueryClient();
    const scrollContainerRef = useRef<HTMLDivElement>(null);

    // Scroll to bottom gate
    const [hasScrolledToBottom, setHasScrolledToBottom] = useState(false);
    const [agreedCheckbox, setAgreedCheckbox] = useState(false);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    // Withdrawal modal state
    const [isWithdrawModalOpen, setIsWithdrawModalOpen] = useState(false);
    const [withdrawReason, setWithdrawReason] = useState("");

    // 1. Fetch Document and Current Consent
    const { data: docData, isLoading: isLoadingDoc } = useQuery({
        queryKey: ["consent-document"],
        queryFn: employeeApi.getConsentDocument,
    });

    const { data: myConsent, isLoading: isLoadingMyConsent } = useQuery({
        queryKey: ["consent-me"],
        queryFn: employeeApi.getMyConsent,
    });

    // Check scroll position
    const handleScroll = () => {
        if (!scrollContainerRef.current || hasScrolledToBottom) return;
        const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
        // Allow a small threshold (e.g. 20px)
        if (scrollTop + clientHeight >= scrollHeight - 20) {
            setHasScrolledToBottom(true);
        }
    };

    // Check if content fits without scroll
    useEffect(() => {
        if (scrollContainerRef.current) {
            const { scrollHeight, clientHeight } = scrollContainerRef.current;
            if (scrollHeight <= clientHeight + 10) {
                setHasScrolledToBottom(true);
            }
        }
    }, [docData]);

    // Grant Consent Mutation
    const grantMutation = useMutation({
        mutationFn: (version: string) => employeeApi.grantConsent(version),
        onSuccess: () => {
            setErrorMessage(null);
            queryClient.invalidateQueries({ queryKey: ["consent-me"] });
            queryClient.invalidateQueries({ queryKey: ["enrollment-status-me"] });
            queryClient.invalidateQueries({ queryKey: ["attendance-context"] });
        },
        onError: (err: unknown) => {
            setErrorMessage(err instanceof Error ? err.message : "Gagal menyimpan persetujuan konsen.");
        },
    });

    // Withdraw Consent Mutation
    const withdrawMutation = useMutation({
        mutationFn: (reason: string) => employeeApi.withdrawConsent(reason),
        onSuccess: () => {
            setIsWithdrawModalOpen(false);
            setWithdrawReason("");
            queryClient.invalidateQueries({ queryKey: ["consent-me"] });
            queryClient.invalidateQueries({ queryKey: ["enrollment-status-me"] });
            queryClient.invalidateQueries({ queryKey: ["attendance-context"] });
        },
        onError: (err: unknown) => {
            setErrorMessage(err instanceof Error ? err.message : "Gagal menarik persetujuan konsen.");
        },
    });

    const docBody = docData?.body ?? "";
    const sanitizedHtml = useMemo(() => {
        if (!docBody) return "";
        return renderMarkdownSafely(docBody);
    }, [docBody]);

    const isGranted = myConsent?.status === "granted" && myConsent.is_current_version;

    if (isLoadingDoc || isLoadingMyConsent) {
        return (
            <div className="max-w-2xl mx-auto p-8 text-center space-y-3">
                <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
                <p className="text-sm text-slate-500">Memuat lembar persetujuan biometrik...</p>
            </div>
        );
    }

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            {/* Header Banner */}
            <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs flex flex-wrap items-center justify-between gap-4">
                <div className="flex items-center space-x-3">
                    <div className="w-12 h-12 rounded-xl bg-primary-50 text-primary-700 flex items-center justify-center">
                        <ShieldCheck className="w-6 h-6" />
                    </div>
                    <div>
                        <h1 className="text-lg font-bold text-slate-900">Persetujuan Biometrik Wajah</h1>
                        <p className="text-xs text-slate-500">
                            Kepatuhan UU Pelindungan Data Pribadi (UU PDP No. 27/2022)
                        </p>
                    </div>
                </div>

                <div>
                    {isGranted ? (
                        <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-bold bg-emerald-100 text-emerald-800">
                            <CheckCircle2 className="w-3.5 h-3.5 mr-1" />
                            Persetujuan Aktif ({myConsent?.document_version})
                        </span>
                    ) : (
                        <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-bold bg-amber-100 text-amber-800">
                            <Clock className="w-3.5 h-3.5 mr-1" />
                            Belum Disetujui
                        </span>
                    )}
                </div>
            </div>

            {/* Document Reader Container */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-xs overflow-hidden">
                <div className="p-4 border-b border-slate-100 bg-slate-50 flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                        <FileText className="w-4 h-4 text-slate-500" />
                        <span className="text-xs font-bold text-slate-700">
                            {docData?.title || "Lembar Persetujuan Biometrik"}
                        </span>
                    </div>
                    <span className="text-[11px] font-mono text-slate-500">
                        Versi Dokumen: {docData?.version || "v1.0"}
                    </span>
                </div>

                {/* Scrollable Markdown Reader */}
                <div
                    ref={scrollContainerRef}
                    onScroll={handleScroll}
                    className="p-6 max-h-[380px] overflow-y-auto prose prose-xs max-w-none text-slate-700 leading-relaxed space-y-4"
                    dangerouslySetInnerHTML={{ __html: sanitizedHtml }}
                />

                {/* Scroll helper banner */}
                {!hasScrolledToBottom && !isGranted && (
                    <div className="bg-primary-50 px-4 py-2.5 border-t border-primary-100 flex items-center justify-between text-xs text-primary-800">
                        <span>Gulir dokumen ke bagian paling bawah untuk membuka opsi persetujuan.</span>
                        <Lock className="w-4 h-4 text-primary-600 shrink-0 ml-2" />
                    </div>
                )}
            </div>

            {/* Consent Action & Checkbox */}
            {!isGranted ? (
                <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-4">
                    <label
                        className={`flex items-start space-x-3 p-3 rounded-xl border transition-all ${!hasScrolledToBottom
                                ? "opacity-50 cursor-not-allowed bg-slate-50 border-slate-200"
                                : "cursor-pointer hover:bg-slate-50 border-slate-200"
                            }`}
                    >
                        <input
                            type="checkbox"
                            disabled={!hasScrolledToBottom}
                            checked={agreedCheckbox}
                            onChange={(e) => setAgreedCheckbox(e.target.checked)}
                            className="mt-1 h-4 w-4 text-primary-600 rounded border-slate-300 focus:ring-primary-500"
                        />
                        <span className="text-xs text-slate-700 leading-relaxed">
                            Saya telah membaca, memahami secara menyeluruh, dan dengan ini memberikan persetujuan eksplisit
                            atas pemrosesan data biometrik wajah saya untuk sistem presensi kerja sesuai ketentuan UU PDP
                            No. 27/2022.
                        </span>
                    </label>

                    {errorMessage && (
                        <div className="bg-rose-50 border border-rose-200 rounded-xl p-3 text-xs text-rose-800 flex items-center space-x-2">
                            <AlertTriangle className="w-4 h-4 shrink-0 text-rose-600" />
                            <span>{errorMessage}</span>
                        </div>
                    )}

                    <div className="flex justify-end">
                        <button
                            type="button"
                            disabled={!agreedCheckbox || grantMutation.isPending || !docData?.version}
                            onClick={() => docData?.version && grantMutation.mutate(docData.version)}
                            className="px-6 py-3 rounded-xl bg-primary-600 hover:bg-primary-700 text-white font-bold text-sm transition-all shadow-xs flex items-center space-x-2 disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                            {grantMutation.isPending ? (
                                <>
                                    <Loader2 className="w-4 h-4 animate-spin" />
                                    <span>Menyimpan Persetujuan...</span>
                                </>
                            ) : (
                                <>
                                    <ShieldCheck className="w-4 h-4" />
                                    <span>Beri Persetujuan Biometrik</span>
                                </>
                            )}
                        </button>
                    </div>
                </div>
            ) : (
                /* Granted State Controls */
                <div className="bg-white rounded-2xl p-6 border border-slate-200 shadow-xs space-y-4">
                    <div className="flex items-center justify-between">
                        <div className="space-y-1">
                            <h3 className="text-sm font-bold text-slate-900">Hak Penarikan Persetujuan</h3>
                            <p className="text-xs text-slate-500">
                                Sesuai hak subjek data UU PDP, Anda dapat menarik persetujuan kapan saja.
                            </p>
                        </div>

                        <button
                            type="button"
                            onClick={() => setIsWithdrawModalOpen(true)}
                            className="px-4 py-2 rounded-xl text-xs font-bold text-rose-600 bg-rose-50 hover:bg-rose-100 border border-rose-200 transition-colors"
                        >
                            Tarik Persetujuan
                        </button>
                    </div>
                </div>
            )}

            {/* Withdrawal Modal */}
            {isWithdrawModalOpen && (
                <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
                    <div className="bg-white rounded-2xl shadow-xl border border-slate-200 w-full max-w-md p-6 space-y-4 animate-in fade-in zoom-in-95 duration-200">
                        <div className="flex items-center space-x-3 text-rose-600">
                            <AlertTriangle className="w-6 h-6" />
                            <h3 className="text-base font-bold text-slate-900">Tarik Persetujuan Biometrik?</h3>
                        </div>

                        <p className="text-xs text-slate-600 leading-relaxed">
                            Jika Anda menarik persetujuan, seluruh data referensi biometrik wajah Anda akan dinonaktifkan
                            dari sistem. Presensi kerja Anda selanjutnya akan dialihkan ke mode manual / persetujuan atasan.
                        </p>

                        <div>
                            <label className="block text-xs font-semibold text-slate-700 mb-1">
                                Alasan Penarikan Persetujuan: <span className="text-rose-500">*</span>
                            </label>
                            <textarea
                                rows={3}
                                value={withdrawReason}
                                onChange={(e) => setWithdrawReason(e.target.value)}
                                placeholder="Contoh: Menginginkan presensi melalui jalur manual"
                                className="w-full text-xs rounded-lg border border-slate-300 p-2.5 focus:outline-none focus:ring-2 focus:ring-rose-500/20 focus:border-rose-500"
                            />
                        </div>

                        <div className="flex justify-end space-x-3 pt-2">
                            <button
                                type="button"
                                onClick={() => setIsWithdrawModalOpen(false)}
                                disabled={withdrawMutation.isPending}
                                className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-xl"
                            >
                                Batal
                            </button>
                            <button
                                type="button"
                                disabled={withdrawReason.trim().length < 3 || withdrawMutation.isPending}
                                onClick={() => withdrawMutation.mutate(withdrawReason.trim())}
                                className="px-4 py-2 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-xl disabled:opacity-50 flex items-center space-x-2"
                            >
                                {withdrawMutation.isPending ? (
                                    <>
                                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                        <span>Memproses...</span>
                                    </>
                                ) : (
                                    <span>Konfirmasi Penarikan</span>
                                )}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
