import { useState, useEffect } from "react";
import { Shield, AlertCircle } from "lucide-react";
import type { Role } from "../../../types/api";
import { Modal } from "../../../components/ui/Modal";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

interface RoleFormModalProps {
    open: boolean;
    onClose: () => void;
    roleToEdit: Role | null; // null for create mode
    onSubmit: (data: {
        name?: string;
        display_name: string;
        description: string;
    }) => Promise<void>;
    isSubmitting: boolean;
}

export function RoleFormModal({
    open,
    onClose,
    roleToEdit,
    onSubmit,
    isSubmitting,
}: RoleFormModalProps) {
    const isEdit = !!roleToEdit;

    const [name, setName] = useState("");
    const [displayName, setDisplayName] = useState("");
    const [description, setDescription] = useState("");
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (roleToEdit) {
            setName(roleToEdit.name);
            setDisplayName(roleToEdit.display_name || roleToEdit.name);
            setDescription(roleToEdit.description || "");
        } else {
            setName("");
            setDisplayName("");
            setDescription("");
        }
        setError(null);
    }, [roleToEdit, open]);

    // Auto-generate slug name from display_name in create mode if name is touched/untouched
    const handleDisplayNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = e.target.value;
        setDisplayName(val);
        if (!isEdit && (!name || name === slugify(displayName))) {
            setName(slugify(val));
        }
    };

    const slugify = (text: string) => {
        return text
            .toLowerCase()
            .trim()
            .replace(/[^a-z0-9_]/g, "_")
            .replace(/_{2,}/g, "_");
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!displayName.trim()) {
            setError("Nama tampilan peran wajib diisi.");
            return;
        }

        if (!isEdit && !name.trim()) {
            setError("Kode pengenal peran (slug) wajib diisi.");
            return;
        }

        try {
            await onSubmit({
                ...(isEdit ? {} : { name: slugify(name) }),
                display_name: displayName.trim(),
                description: description.trim(),
            });
            onClose();
        } catch (err: unknown) {
            const msg =
                err instanceof Error ? err.message : "Gagal menyimpan data peran.";
            setError(msg);
        }
    };

    return (
        <Modal
            open={open}
            onClose={onClose}
            title={isEdit ? `Edit Peran: ${roleToEdit.display_name || roleToEdit.name}` : "Tambah Peran Baru"}
            description={
                isEdit
                    ? "Perbarui nama tampilan dan deskripsi peran pengguna."
                    : "Buat peran khusus baru untuk menentukan hak akses menu dan modul secara terperinci."
            }
            maxWidth="md"
        >
            <form onSubmit={handleSubmit} className="space-y-4">
                {error && (
                    <div className="p-3 bg-red-50 border border-red-200 text-red-700 rounded-lg text-xs flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0 text-red-500" />
                        <span>{error}</span>
                    </div>
                )}

                {/* Display Name */}
                <div>
                    <label className="block text-xs font-semibold text-gray-700 mb-1">
                        Nama Tampilan Peran <span className="text-red-500">*</span>
                    </label>
                    <Input
                        type="text"
                        required
                        placeholder="Contoh: Manager Operasional, Pengawas Cabang"
                        value={displayName}
                        onChange={handleDisplayNameChange}
                        className="text-xs"
                    />
                    <p className="text-[11px] text-gray-400 mt-1">
                        Nama yang mudah dipahami dan ditampilkan pada antarmuka.
                    </p>
                </div>

                {/* Slug Name */}
                <div>
                    <label className="block text-xs font-semibold text-gray-700 mb-1">
                        Kode Teknis Peran (Slug) <span className="text-red-500">*</span>
                    </label>
                    <Input
                        type="text"
                        required
                        disabled={isEdit}
                        placeholder="Contoh: manager_operasional"
                        value={name}
                        onChange={(e) => setName(slugify(e.target.value))}
                        className="font-mono text-xs disabled:bg-gray-100 disabled:cursor-not-allowed"
                    />
                    <p className="text-[11px] text-gray-400 mt-1">
                        {isEdit
                            ? "Kode teknis peran sistem bersifat permanen dan tidak dapat diubah."
                            : "Hanya huruf kecil, angka, dan garis bawah (_). Digunakan dalam evaluasi izin internal."}
                    </p>
                </div>

                {/* Description */}
                <div>
                    <label className="block text-xs font-semibold text-gray-700 mb-1">
                        Deskripsi Peran
                    </label>
                    <textarea
                        rows={3}
                        placeholder="Jelaskan cakupan tanggung jawab dan tujuan dari peran ini..."
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        className="w-full text-xs rounded-lg border-gray-300 shadow-2xs focus:border-indigo-500 focus:ring-indigo-500 p-2.5 border"
                    />
                </div>

                {isEdit && roleToEdit.is_system && (
                    <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg text-amber-800 text-[11px] flex items-start gap-2">
                        <Shield className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                        <span>
                            Peran ini ditandai sebagai <strong>Peran Sistem</strong> bawaan. Anda tetap dapat mengatur matriks izin modul dan deskripsi, tetapi peran tidak dapat dihapus.
                        </span>
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
                        {isEdit ? "Perbarui Peran" : "Buat Peran"}
                    </Button>
                </div>
            </form>
        </Modal>
    );
}
