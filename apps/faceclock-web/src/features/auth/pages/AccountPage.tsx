import { useState } from "react";
import { useAuth } from "../../../lib/auth/useAuth";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { User, KeyRound, Shield, LogOut } from "lucide-react";

export function AccountPage() {
    const { user, logout } = useAuth();
    const toast = useToast();

    const [oldPassword, setOldPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [isChanging, setIsChanging] = useState(false);
    const [isLoggingOutAll, setIsLoggingOutAll] = useState(false);

    const handlePasswordChange = async (e: React.FormEvent) => {
        e.preventDefault();
        if (newPassword !== confirmPassword) {
            toast.show({
                type: "error",
                title: "Validasi Gagal",
                message: "Konfirmasi kata sandi baru tidak cocok.",
            });
            return;
        }

        if (newPassword.length < 8) {
            toast.show({
                type: "error",
                title: "Validasi Gagal",
                message: "Kata sandi baru minimal 8 karakter.",
            });
            return;
        }

        setIsChanging(true);
        try {
            await api.put("/api/v1/auth/password", {
                old_password: oldPassword,
                new_password: newPassword,
            });
            toast.show({
                type: "success",
                title: "Berhasil",
                message: "Kata sandi berhasil diperbarui.",
            });
            setOldPassword("");
            setNewPassword("");
            setConfirmPassword("");
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Mengubah Kata Sandi",
                message: "Periksa kembali kata sandi lama Anda.",
            });
        } finally {
            setIsChanging(false);
        }
    };

    const handleLogoutAll = async () => {
        setIsLoggingOutAll(true);
        try {
            await api.post("/api/v1/auth/logout-all", {});
            toast.show({
                type: "info",
                title: "Sesi Diakhiri",
                message: "Semua sesi perangkat lain telah dihentikan.",
            });
            await logout();
        } catch {
            toast.show({
                type: "error",
                title: "Gagal",
                message: "Gagal mengakhiri semua sesi.",
            });
        } finally {
            setIsLoggingOutAll(false);
        }
    };

    return (
        <div className="space-y-6 max-w-4xl mx-auto">
            <div>
                <h1 className="text-xl font-bold text-gray-900">Pengaturan Akun</h1>
                <p className="text-xs text-gray-500 mt-0.5">
                    Kelola profil pengguna, kata sandi, dan keamanan sesi Anda.
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Profile Card */}
                <Card>
                    <CardHeader>
                        <div className="flex items-center gap-2">
                            <User className="w-4 h-4 text-indigo-600" />
                            <CardTitle>Profil Pengguna</CardTitle>
                        </div>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div>
                            <span className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider block">
                                Nama Lengkap
                            </span>
                            <p className="text-sm font-semibold text-gray-900 mt-0.5">
                                {user?.full_name || "-"}
                            </p>
                        </div>

                        <div>
                            <span className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider block">
                                Email
                            </span>
                            <p className="text-sm font-medium text-gray-900 mt-0.5">{user?.email}</p>
                        </div>

                        <div>
                            <span className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider block mb-1">
                                Role & Wewenang
                            </span>
                            <div className="flex flex-wrap gap-1.5">
                                {user?.roles?.map((r) => (
                                    <span
                                        key={r.id}
                                        className="inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-200"
                                    >
                                        <Shield className="w-3 h-3" />
                                        {r.name}
                                    </span>
                                ))}
                            </div>
                        </div>

                        <div className="pt-4 border-t border-gray-100">
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={handleLogoutAll}
                                isLoading={isLoggingOutAll}
                                className="text-rose-600 border-rose-200 hover:bg-rose-50"
                            >
                                <LogOut className="w-3.5 h-3.5 mr-1" />
                                Keluar dari Semua Perangkat
                            </Button>
                        </div>
                    </CardContent>
                </Card>

                {/* Change Password Card */}
                <Card>
                    <CardHeader>
                        <div className="flex items-center gap-2">
                            <KeyRound className="w-4 h-4 text-indigo-600" />
                            <CardTitle>Ubah Kata Sandi</CardTitle>
                        </div>
                    </CardHeader>
                    <CardContent>
                        <form onSubmit={handlePasswordChange} className="space-y-3.5">
                            <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Kata Sandi Lama
                                </label>
                                <Input
                                    type="password"
                                    required
                                    value={oldPassword}
                                    onChange={(e) => setOldPassword(e.target.value)}
                                    placeholder="••••••••"
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Kata Sandi Baru
                                </label>
                                <Input
                                    type="password"
                                    required
                                    value={newPassword}
                                    onChange={(e) => setNewPassword(e.target.value)}
                                    placeholder="Minimal 8 karakter"
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Konfirmasi Kata Sandi Baru
                                </label>
                                <Input
                                    type="password"
                                    required
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    placeholder="Ulangi kata sandi baru"
                                />
                            </div>

                            <Button
                                type="submit"
                                variant="primary"
                                size="sm"
                                className="w-full mt-2"
                                isLoading={isChanging}
                            >
                                Simpan Kata Sandi Baru
                            </Button>
                        </form>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
