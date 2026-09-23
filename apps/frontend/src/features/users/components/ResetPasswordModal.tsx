import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles, Lock, AlertCircle } from "lucide-react";
import type { User } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

interface ResetPasswordModalProps {
    open: boolean;
    onClose: () => void;
    user: User | null;
    onSubmit: (password?: string) => Promise<string | void>;
    isSubmitting: boolean;
}

export function ResetPasswordModal({
    open,
    onClose,
    user,
    onSubmit,
    isSubmitting,
}: ResetPasswordModalProps) {
    const { t } = useTranslation(["user", "common"]);
    const [mode, setMode] = useState<"auto" | "manual">("auto");
    const [customPassword, setCustomPassword] = useState("");
    const [error, setError] = useState<string | null>(null);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (mode === "manual") {
            if (!customPassword || customPassword.length < 10) {
                setError("Kata sandi manual harus memiliki panjang minimal 10 karakter.");
                return;
            }
        }

        try {
            await onSubmit(mode === "manual" ? customPassword : undefined);
            setCustomPassword("");
            setMode("auto");
            onClose();
        } catch (err: unknown) {
            const msg =
                err instanceof Error ? err.message : "Gagal mereset kata sandi pengguna.";
            setError(msg);
        }
    };

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={`${t("resetPassword.title", { ns: "user" })}: ${user?.email}`}
            description={t("resetPassword.subtitle", { ns: "user" })}
            maxWidth="md"
        >
            <form onSubmit={handleSubmit} className="space-y-4">
                {error && (
                    <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-lg text-xs flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
                        <span>{error}</span>
                    </div>
                )}

                {/* Choice of generation */}
                <div className="grid grid-cols-2 gap-2 p-1 bg-gray-100 rounded-xl">
                    <button
                        type="button"
                        onClick={() => {
                            setMode("auto");
                            setError(null);
                        }}
                        className={`py-2 px-3 rounded-lg text-xs font-semibold flex items-center justify-center gap-1.5 transition ${mode === "auto"
                            ? "bg-white text-indigo-700 shadow-2xs"
                            : "text-gray-600 hover:text-gray-900"
                            }`}
                    >
                        <Sparkles className="w-3.5 h-3.5 text-amber-500" />
                        <span>Otomatis (Aman)</span>
                    </button>
                    <button
                        type="button"
                        onClick={() => {
                            setMode("manual");
                            setError(null);
                        }}
                        className={`py-2 px-3 rounded-lg text-xs font-semibold flex items-center justify-center gap-1.5 transition ${mode === "manual"
                            ? "bg-white text-indigo-700 shadow-2xs"
                            : "text-gray-600 hover:text-gray-900"
                            }`}
                    >
                        <Lock className="w-3.5 h-3.5" />
                        <span>Tentukan Sendiri</span>
                    </button>
                </div>

                {mode === "auto" ? (
                    <div className="p-4 bg-indigo-50/60 border border-indigo-100 rounded-xl space-y-1 text-xs text-indigo-950">
                        <div className="font-bold flex items-center gap-1.5">
                            <Sparkles className="w-4 h-4 text-indigo-600" />
                            Kata Sandi Acak Otomatis
                        </div>
                        <p className="text-[11px] text-indigo-800 leading-relaxed">
                            Sistem akan menghasilkan kata sandi sementara yang aman. Anda akan melihat dan dapat menyalin kata sandi tersebut langsung ke clipboard setelah menekan tombol di bawah.
                        </p>
                    </div>
                ) : (
                    <div className="space-y-1.5">
                        <label className="block text-xs font-semibold text-gray-700">
                            Kata Sandi Baru (Minimal 10 Karakter)
                        </label>
                        <Input
                            type="password"
                            required
                            minLength={10}
                            placeholder="Ketik kata sandi baru..."
                            value={customPassword}
                            onChange={(e) => setCustomPassword(e.target.value)}
                            className="text-xs"
                        />
                        <p className="text-[10px] text-gray-400">
                            Pastikan kombinasi mengandung huruf, angka, atau simbol untuk keamanan akun.
                        </p>
                    </div>
                )}

                <div className="pt-2 flex items-center justify-end gap-2 border-t border-gray-100">
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={onClose}
                        disabled={isSubmitting}
                    >
                        Batal
                    </Button>
                    <Button
                        type="submit"
                        variant="primary"
                        size="sm"
                        isLoading={isSubmitting}
                    >
                        Konfirmasi Reset Sandi
                    </Button>
                </div>
            </form>
        </Modal>
    );
}
