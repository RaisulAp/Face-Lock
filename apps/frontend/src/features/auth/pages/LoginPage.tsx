import { useState, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../../../lib/auth/useAuth";
import { getErrorMessage } from "../../../lib/errors/messages";
import { ApiError } from "../../../lib/api";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Lock, Mail, AlertCircle, Clock } from "lucide-react";
import { useTranslation } from "react-i18next";
import { LanguageSwitcher } from "../../../components/ui/LanguageSwitcher";

export function LoginPage() {
    const { login, isAuthenticated } = useAuth();
    const navigate = useNavigate();
    const location = useLocation();
    const { t } = useTranslation(["auth", "common"]);

    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const from = (location.state as { from?: { pathname: string } })?.from?.pathname || "/";

    useEffect(() => {
        if (isAuthenticated) {
            navigate(from, { replace: true });
        }
    }, [isAuthenticated, navigate, from]);

    if (isAuthenticated) {
        return null;
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setErrorMessage(null);
        setIsLoading(true);

        try {
            await login(email.trim(), password);
            navigate(from, { replace: true });
        } catch (err: unknown) {
            if (err instanceof ApiError) {
                setErrorMessage(getErrorMessage(err.code, err.message));
            } else if (err instanceof Error) {
                setErrorMessage(err.message);
            } else {
                setErrorMessage(t("login.errorDefault", { ns: "auth" }));
            }
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-indigo-950 flex items-center justify-center p-4">
            {/* Language switcher — top-right corner */}
            <div className="absolute top-4 right-4">
                <LanguageSwitcher />
            </div>

            <div className="max-w-md w-full">
                {/* Brand Card */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-indigo-600 shadow-xl shadow-indigo-600/30 text-white mb-3">
                        <Clock className="w-8 h-8" />
                    </div>
                    <h1 className="text-2xl font-bold text-white tracking-tight">{t("appName", { ns: "common" })}</h1>
                    <p className="text-xs text-slate-400 mt-1">{t("appTagline", { ns: "common" })}</p>
                </div>

                {/* Login Form Container */}
                <div className="bg-white rounded-2xl shadow-2xl p-6 sm:p-8 border border-white/20">
                    <h2 className="text-lg font-bold text-gray-900 mb-1">{t("login.title", { ns: "auth" })}</h2>
                    <p className="text-xs text-gray-500 mb-6">{t("login.subtitle", { ns: "auth" })}</p>

                    {errorMessage && (
                        <div className="mb-5 p-3 rounded-xl bg-rose-50 border border-rose-200 flex items-start gap-2.5 text-rose-800 text-xs">
                            <AlertCircle className="w-4 h-4 shrink-0 text-rose-600 mt-0.5" />
                            <span>{errorMessage}</span>
                        </div>
                    )}

                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">{t("login.emailLabel", { ns: "auth" })}</label>
                            <div className="relative">
                                <Input
                                    type="email"
                                    required
                                    placeholder={t("login.emailPlaceholder", { ns: "auth" })}
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    className="pl-9"
                                    disabled={isLoading}
                                />
                                <Mail className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">{t("login.passwordLabel", { ns: "auth" })}</label>
                            <div className="relative">
                                <Input
                                    type="password"
                                    required
                                    placeholder={t("login.passwordPlaceholder", { ns: "auth" })}
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    className="pl-9"
                                    disabled={isLoading}
                                />
                                <Lock className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                            </div>
                        </div>

                        <Button type="submit" variant="primary" className="w-full mt-2" isLoading={isLoading}>
                            {isLoading ? t("login.submittingButton", { ns: "auth" }) : t("login.submitButton", { ns: "auth" })}
                        </Button>
                    </form>

                    <div className="mt-6 pt-4 border-t border-gray-100 text-center">
                        <p className="text-[11px] text-gray-400">FaceClock v1.0 • Aman, Terenkripsi & Terverifikasi</p>
                    </div>
                </div>
            </div>
        </div>
    );
}
