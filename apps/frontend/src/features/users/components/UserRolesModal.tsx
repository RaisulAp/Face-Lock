import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Lock, AlertCircle, Shield } from "lucide-react";
import type { User, Role } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Badge } from "../../../components/ui/Badge";

interface UserRolesModalProps {
    open: boolean;
    onClose: () => void;
    user: User | null;
    rolesList: Role[];
    onSave: (roleIds: string[]) => Promise<void>;
    isSubmitting: boolean;
    isSuperAdmin: boolean;
}

export function UserRolesModal({
    open,
    onClose,
    user,
    rolesList,
    onSave,
    isSubmitting,
    isSuperAdmin,
}: UserRolesModalProps) {
    const { t } = useTranslation(["user", "common"]);
    const [selectedRoleId, setSelectedRoleId] = useState<string>("");
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (user && open) {
            // Extract user role names
            const userRoleNames = (user.roles || []).map((r: any) =>
                typeof r === "string" ? r : r.name
            );

            // Find matching role in rolesList
            const matchingRole = rolesList.find((r) => userRoleNames.includes(r.name));

            if (matchingRole) {
                setSelectedRoleId(matchingRole.id);
            } else {
                // Default to employee or first available role
                const employeeRole = rolesList.find((r) => r.name === "employee");
                setSelectedRoleId(employeeRole ? employeeRole.id : rolesList[0]?.id || "");
            }
            setError(null);
        }
    }, [user, rolesList, open]);

    const handleSelectRole = (roleId: string, roleName: string) => {
        // Only super_admin can assign super_admin role
        if (roleName === "super_admin" && !isSuperAdmin) {
            return;
        }
        setSelectedRoleId(roleId);
    };

    const handleFormSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!selectedRoleId) {
            setError("Pengguna wajib memiliki tepat 1 peran akses sistem.");
            return;
        }

        try {
            await onSave([selectedRoleId]);
            onClose();
        } catch (err: unknown) {
            const msg =
                err instanceof Error ? err.message : "Gagal memperbarui peran pengguna.";
            setError(msg);
        }
    };

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={`${t("roles.title", { ns: "user" })}: ${user?.email}`}
            description="Tentukan peran akses sistem (RBAC) untuk akun ini. Setiap pengguna wajib memiliki tepat 1 peran akses sistem."
            maxWidth="lg"
        >
            <form onSubmit={handleFormSubmit} className="space-y-4">
                {error && (
                    <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-lg text-xs flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
                        <span>{error}</span>
                    </div>
                )}

                <div className="space-y-2.5 max-h-[60vh] overflow-y-auto pr-1">
                    {rolesList.map((r) => {
                        const isChecked = selectedRoleId === r.id;
                        const isSuperAdminRole = r.name === "super_admin";
                        const isDisabled = isSuperAdminRole && !isSuperAdmin;

                        return (
                            <label
                                key={r.id}
                                onClick={() => {
                                    if (!isDisabled) handleSelectRole(r.id, r.name);
                                }}
                                className={`flex items-start gap-3 p-3.5 rounded-xl border text-xs transition cursor-pointer ${isDisabled
                                    ? "bg-gray-50 border-gray-200 opacity-60 cursor-not-allowed"
                                    : isChecked
                                        ? "bg-indigo-50/80 border-indigo-300 ring-1 ring-indigo-500 shadow-2xs"
                                        : "bg-white border-gray-200 hover:border-gray-300 hover:bg-gray-50/50"
                                    }`}
                            >
                                <input
                                    type="radio"
                                    name="userRoleRadio"
                                    disabled={isDisabled}
                                    checked={isChecked}
                                    onChange={() => handleSelectRole(r.id, r.name)}
                                    onClick={(e) => e.stopPropagation()}
                                    className="mt-0.5 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                />

                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center justify-between gap-2">
                                        <div className="flex items-center gap-2">
                                            <Shield className="w-4 h-4 text-indigo-600" />
                                            <span className="font-bold text-gray-900 text-xs">
                                                {r.display_name || r.name}
                                            </span>
                                        </div>
                                        <div className="flex items-center gap-1.5">
                                            <Badge variant="neutral" size="sm">
                                                <Lock className="w-3 h-3 mr-1" />
                                                Sistem
                                            </Badge>
                                        </div>
                                    </div>

                                    <div className="font-mono text-[10px] text-gray-400 mt-0.5">
                                        {r.name}
                                    </div>

                                    <p className="text-[11px] text-gray-500 mt-1 leading-snug">
                                        {r.description || "Peran akses sistem baku."}
                                    </p>

                                    {isDisabled && (
                                        <div className="text-[10px] text-amber-700 bg-amber-50 px-2 py-1 rounded border border-amber-200 mt-2">
                                            Hanya Super Admin yang dapat memberikan peran Administrator Utama.
                                        </div>
                                    )}
                                </div>
                            </label>
                        );
                    })}
                </div>

                <div className="flex items-center justify-between pt-2 border-t border-gray-100">
                    <span className="text-xs text-gray-500">
                        1 peran akses terpilih
                    </span>
                    <div className="flex items-center gap-2">
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
                            Simpan Peran
                        </Button>
                    </div>
                </div>
            </form>
        </Modal>
    );
}
