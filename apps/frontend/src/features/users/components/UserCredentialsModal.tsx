import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, Eye, EyeOff, ShieldCheck, AlertCircle } from "lucide-react";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";

interface UserCredentialsModalProps {
    open: boolean;
    onClose: () => void;
    email: string;
    temporaryPassword?: string;
    actionTitle?: string;
}

export function UserCredentialsModal({
    open,
    onClose,
    email,
    temporaryPassword,
    actionTitle = "Akun Pengguna Berhasil Disimpan",
}: UserCredentialsModalProps) {
    const { t } = useTranslation(["user", "common"]);
    const [showPassword, setShowPassword] = useState(true);
    const [copiedAll, setCopiedAll] = useState(false);
    const [copiedPwd, setCopiedPwd] = useState(false);

    if (!open) return null;

    const handleCopyAll = async () => {
        const portalUrl = window.location.origin;
        const text = `Selamat! Akun FaceClock Anda telah siap:\n\nEmail: ${email}\nKata Sandi Sementara: ${temporaryPassword || "(Sesuai kata sandi yang Anda tentukan)"}\nAlamat Portal: ${portalUrl}\n\nCatatan: Harap segera ganti kata sandi setelah berhasil masuk pertama kali.`;
        await navigator.clipboard.writeText(text);
        setCopiedAll(true);
        setTimeout(() => setCopiedAll(false), 2500);
    };

    const handleCopyPassword = async () => {
        if (!temporaryPassword) return;
        await navigator.clipboard.writeText(temporaryPassword);
        setCopiedPwd(true);
        setTimeout(() => setCopiedPwd(false), 2500);
    };

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={actionTitle || t("credentials.title", { ns: "user" })}
            description={t("credentials.subtitle", { ns: "user" })}
            maxWidth="md"
        >
            <div className="space-y-4">
                {/* Success Banner */}
                <div className="p-3.5 bg-emerald-50 border border-emerald-200 rounded-xl flex items-center gap-3 text-emerald-900 text-xs">
                    <div className="w-8 h-8 rounded-lg bg-emerald-600 text-white flex items-center justify-center shrink-0">
                        <ShieldCheck className="w-5 h-5" />
                    </div>
                    <div>
                        <div className="font-bold text-emerald-950">Akses Siap Digunakan</div>
                        <div className="text-[11px] text-emerald-700 mt-0.5">
                            Pengguna dapat menggunakan email dan kata sandi di bawah untuk masuk ke FaceClock.
                        </div>
                    </div>
                </div>

                {/* Credentials Box */}
                <div className="p-4 bg-gray-50 border border-gray-200 rounded-xl space-y-3">
                    <div>
                        <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wider block mb-1">
                            Email Pengguna
                        </span>
                        <div className="font-semibold text-sm text-gray-900 font-mono bg-white px-3 py-2 rounded-lg border border-gray-200 select-all">
                            {email}
                        </div>
                    </div>

                    <div>
                        <div className="flex items-center justify-between mb-1">
                            <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wider">
                                Kata Sandi Sementara
                            </span>
                            {temporaryPassword && (
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    className="text-[11px] text-indigo-600 hover:text-indigo-800 flex items-center gap-1"
                                >
                                    {showPassword ? (
                                        <>
                                            <EyeOff className="w-3.5 h-3.5" />
                                            <span>Sembunyikan</span>
                                        </>
                                    ) : (
                                        <>
                                            <Eye className="w-3.5 h-3.5" />
                                            <span>Tampilkan</span>
                                        </>
                                    )}
                                </button>
                            )}
                        </div>

                        <div className="flex items-center gap-2">
                            <div className="flex-1 font-mono font-bold text-sm tracking-wide bg-white px-3 py-2 rounded-lg border border-gray-200 text-indigo-900 select-all min-h-[38px] flex items-center">
                                {temporaryPassword ? (
                                    showPassword ? (
                                        temporaryPassword
                                    ) : (
                                        "••••••••••••••••"
                                    )
                                ) : (
                                    <span className="text-xs text-gray-500 font-normal italic">
                                        (Kata sandi manual yang Anda masukkan saat pembuatan)
                                    </span>
                                )}
                            </div>

                            {temporaryPassword && (
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={handleCopyPassword}
                                    className="shrink-0 text-xs"
                                >
                                    {copiedPwd ? (
                                        <>
                                            <Check className="w-3.5 h-3.5 text-emerald-600 mr-1" />
                                            Tersalin
                                        </>
                                    ) : (
                                        <>
                                            <Copy className="w-3.5 h-3.5 mr-1" />
                                            Salin Sandi
                                        </>
                                    )}
                                </Button>
                            )}
                        </div>
                    </div>
                </div>

                {/* Warning Alert */}
                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start gap-2.5 text-amber-900 text-xs">
                    <AlertCircle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                    <div className="text-[11px] leading-relaxed">
                        <strong>Penting:</strong> Demi keamanan data perusahaan, kata sandi sementara tidak akan disimpan dalam bentuk teks biasa di sistem dan tidak akan bisa ditampilkan lagi setelah jendela ini ditutup.
                    </div>
                </div>

                {/* Modal Actions */}
                <div className="flex flex-col sm:flex-row items-center justify-between gap-2 pt-2 border-t border-gray-100">
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={handleCopyAll}
                        className="w-full sm:w-auto text-xs"
                    >
                        {copiedAll ? (
                            <>
                                <Check className="w-3.5 h-3.5 text-emerald-600 mr-1.5" />
                                Semua Kredensial Tersalin!
                            </>
                        ) : (
                            <>
                                <Copy className="w-3.5 h-3.5 mr-1.5" />
                                Salin Format Lengkap untuk Karyawan
                            </>
                        )}
                    </Button>

                    <Button
                        type="button"
                        variant="primary"
                        size="sm"
                        onClick={onClose}
                        className="w-full sm:w-auto text-xs"
                    >
                        Selesai & Tutup
                    </Button>
                </div>
            </div>
        </Modal>
    );
}
