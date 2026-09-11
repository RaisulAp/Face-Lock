import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../../lib/api";
import { useToast } from "../../../components/ui/Toast";
import type { FaceQualityStatusResponse, SettingItem } from "../../../types/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Select } from "../../../components/ui/Select";
import { Badge } from "../../../components/ui/Badge";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import {
    Sliders,
    ShieldCheck,
    MapPin,
    Clock,
    Lock,
    Activity,
    AlertTriangle,
    RefreshCw,
    Save,
    CheckCircle2,
} from "lucide-react";

export function SettingsPage() {
    const toast = useToast();

    // 1. Fetch all settings (Endpoint #28)
    const {
        data: settingsList,
        isLoading: isSettingsLoading,
        refetch: refetchSettings,
    } = useQuery<SettingItem[]>({
        queryKey: ["settings", "all"],
        queryFn: () => api.get<SettingItem[]>("/api/v1/settings"),
    });

    // 2. Fetch Face Quality Status (Endpoint #73, K-02)
    const {
        data: qualityStatus,
        isLoading: isQualityLoading,
        refetch: refetchQuality,
        isFetching: isQualityFetching,
    } = useQuery<FaceQualityStatusResponse>({
        queryKey: ["settings", "face-quality-status"],
        queryFn: () => api.get<FaceQualityStatusResponse>("/api/v1/settings/face-quality-status"),
    });

    // Local form state map: key -> string/number/boolean
    const [formValues, setFormValues] = useState<Record<string, unknown>>({});
    const [originalValues, setOriginalValues] = useState<Record<string, unknown>>({});
    const [savingKey, setSavingKey] = useState<string | null>(null);

    // Consequence Dialog State
    const [pendingUpdate, setPendingUpdate] = useState<{
        key: string;
        value: unknown;
        warningMessage: string;
    } | null>(null);

    useEffect(() => {
        if (settingsList) {
            const map: Record<string, unknown> = {};
            settingsList.forEach((s) => {
                map[s.key] = s.value;
            });
            setFormValues(map);
            setOriginalValues(map);
        }
    }, [settingsList]);

    const handleChange = (key: string, val: unknown) => {
        setFormValues((prev) => ({ ...prev, [key]: val }));
    };

    const checkConsequenceAndSave = (key: string, newValue: unknown) => {
        const origVal = originalValues[key];

        // Consequence rules:
        if (key === "face.similarity_threshold" && Number(newValue) < Number(origVal)) {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Menurunkan ambang kemiripan wajah membuat sistem lebih mudah menerima wajah yang mirip. Angka saat ini berasal dari kalibrasi dataset. Pastikan Anda memiliki data evaluasi sebelum mengubah ini.",
            });
            return;
        }

        if (key === "attendance.outside_geofence_policy" && newValue === "pending_review") {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Absensi dari luar radius kantor akan tetap tercatat dan masuk antrian persetujuan. Jumlah antrian review supervisor akan bertambah banyak.",
            });
            return;
        }

        if (key === "attendance.require_face_for_checkout" && (newValue === false || newValue === "false")) {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Check-out tidak lagi diverifikasi biometrik wajah. Siapa pun yang memegang akun bisa mencatat jam pulang tanpa pencocokan kamera.",
            });
            return;
        }

        if (key === "attendance.allow_fallback_without_enrollment" && (newValue === true || newValue === "true")) {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Karyawan yang belum mendaftarkan wajah bisa absen lewat jalur persetujuan manual supervisor.",
            });
            return;
        }

        if (key === "attendance.geofence_enabled" && (newValue === false || newValue === "false")) {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Lokasi kantor tidak lagi diperiksa. Karyawan dapat melakukan absensi dari mana saja tanpa validasi jarak GPS.",
            });
            return;
        }

        if (key === "attendance.timezone" && newValue !== origVal) {
            setPendingUpdate({
                key,
                value: newValue,
                warningMessage:
                    "Absensi lama tetap memakai tanggal kerja yang sudah tersimpan. Perubahan timezone hanya berlaku untuk pencatatan absensi baru.",
            });
            return;
        }

        // Direct save if no special consequence warning
        performUpdate(key, newValue);
    };

    const performUpdate = async (key: string, value: unknown) => {
        setSavingKey(key);
        try {
            await api.put(`/api/v1/settings/${key}`, { value });
            toast.show({
                type: "success",
                title: "Pengaturan Disimpan",
                message: `Kunci ${key} berhasil diperbarui.`,
            });
            setOriginalValues((prev) => ({ ...prev, [key]: value }));
            setPendingUpdate(null);
            refetchSettings();
        } catch {
            toast.show({
                type: "error",
                title: "Gagal Menyimpan",
                message: `Terjadi kesalahan saat menyimpan pengaturan ${key}.`,
            });
        } finally {
            setSavingKey(null);
        }
    };

    if (isSettingsLoading) {
        return (
            <div className="flex h-64 items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent" />
            </div>
        );
    }

    return (
        <div className="space-y-8 max-w-5xl mx-auto pb-12">
            {/* Header */}
            <div>
                <div className="flex items-center gap-2">
                    <Sliders className="w-5 h-5 text-indigo-600" />
                    <h1 className="text-xl font-bold text-gray-900">Pengaturan Sistem & Kebijakan</h1>
                </div>
                <p className="text-xs text-gray-500 mt-0.5">
                    Kelola parameter biometrik wajah, kebijakan geofence, jadwal kerja, dan retensi data.
                </p>
            </div>

            {/* KARTU 1: VERIFIKASI WAJAH */}
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <ShieldCheck className="w-4 h-4 text-indigo-600" />
                        <CardTitle>1. Parameter Verifikasi Wajah</CardTitle>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Ambang Kemiripan Wajah (Similarity Threshold)
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    step="0.01"
                                    min="0.4"
                                    max="0.99"
                                    value={Number(formValues["face.similarity_threshold"] ?? 0.7)}
                                    onChange={(e) => handleChange("face.similarity_threshold", Number(e.target.value))}
                                />
                                <Button
                                    variant="primary"
                                    size="sm"
                                    isLoading={savingKey === "face.similarity_threshold"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "face.similarity_threshold",
                                            formValues["face.similarity_threshold"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                            <span className="text-[11px] text-gray-400 mt-1 block">
                                Default: 0.70 (Cosine Similarity ArcFace)
                            </span>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Model Versi Aktif (Read-Only)
                            </label>
                            <Input
                                disabled
                                value={String(formValues["face.model_version"] ?? "buffalo_l")}
                                className="bg-gray-50 text-gray-500 font-mono"
                            />
                            <span className="text-[11px] text-gray-400 mt-1 block">
                                InsightFace buffalo_l (ResNet-50 512D)
                            </span>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-2">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Min. Skor Kualitas Foto
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    step="0.05"
                                    value={Number(formValues["face.min_quality_score"] ?? 0.6)}
                                    onChange={(e) => handleChange("face.min_quality_score", Number(e.target.value))}
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "face.min_quality_score"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "face.min_quality_score",
                                            formValues["face.min_quality_score"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Wajib Formulir Persetujuan (PDP)
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["face.consent_required"] ?? "true")}
                                    onChange={(e) =>
                                        handleChange("face.consent_required", e.target.value === "true")
                                    }
                                >
                                    <option value="true">Wajib (UU PDP)</option>
                                    <option value="false">Tidak Wajib</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "face.consent_required"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "face.consent_required",
                                            formValues["face.consent_required"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Pencegahan Foto Duplikat
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["face.duplicate_check_enabled"] ?? "true")}
                                    onChange={(e) =>
                                        handleChange("face.duplicate_check_enabled", e.target.value === "true")
                                    }
                                >
                                    <option value="true">Aktif</option>
                                    <option value="false">Nonaktif</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "face.duplicate_check_enabled"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "face.duplicate_check_enabled",
                                            formValues["face.duplicate_check_enabled"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* KARTU 2: LOKASI & GEOFENCE */}
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <MapPin className="w-4 h-4 text-indigo-600" />
                        <CardTitle>2. Kebijakan Lokasi & Geofence (Decision D18)</CardTitle>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Validasi Geofence Aktif
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["attendance.geofence_enabled"] ?? "true")}
                                    onChange={(e) =>
                                        handleChange("attendance.geofence_enabled", e.target.value === "true")
                                    }
                                >
                                    <option value="true">Diperiksa (Aktif)</option>
                                    <option value="false">Diabaikan (Bebas Lokasi)</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.geofence_enabled"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.geofence_enabled",
                                            formValues["attendance.geofence_enabled"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Kebijakan Luar Geofence (D18)
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["attendance.outside_geofence_policy"] ?? "reject")}
                                    onChange={(e) =>
                                        handleChange("attendance.outside_geofence_policy", e.target.value)
                                    }
                                >
                                    <option value="reject">Tolak Langsung (Reject)</option>
                                    <option value="pending_review">Masuk Antrian Review (Pending)</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.outside_geofence_policy"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.outside_geofence_policy",
                                            formValues["attendance.outside_geofence_policy"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Toleransi Akurasi GPS Maksimal (Meter)
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    value={Number(formValues["attendance.max_gps_accuracy_meter"] ?? 100)}
                                    onChange={(e) =>
                                        handleChange("attendance.max_gps_accuracy_meter", Number(e.target.value))
                                    }
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.max_gps_accuracy_meter"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.max_gps_accuracy_meter",
                                            formValues["attendance.max_gps_accuracy_meter"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* KARTU 3: ATURAN ABSENSI & WAKTU KERJA */}
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4 text-indigo-600" />
                        <CardTitle>3. Jadwal Kerja & Aturan Absensi (Decision D19)</CardTitle>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Zona Waktu (Timezone)
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["attendance.timezone"] ?? "Asia/Jakarta")}
                                    onChange={(e) => handleChange("attendance.timezone", e.target.value)}
                                >
                                    <option value="Asia/Jakarta">Asia/Jakarta (WIB, UTC+7)</option>
                                    <option value="Asia/Makassar">Asia/Makassar (WITA, UTC+8)</option>
                                    <option value="Asia/Jayapura">Asia/Jayapura (WIT, UTC+9)</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.timezone"}
                                    onClick={() =>
                                        checkConsequenceAndSave("attendance.timezone", formValues["attendance.timezone"])
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Wajib Wajah Saat Checkout (D19)
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["attendance.require_face_for_checkout"] ?? "true")}
                                    onChange={(e) =>
                                        handleChange("attendance.require_face_for_checkout", e.target.value === "true")
                                    }
                                >
                                    <option value="true">Wajib Verifikasi Wajah</option>
                                    <option value="false">Bebas Foto (Tanpa Wajah)</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.require_face_for_checkout"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.require_face_for_checkout",
                                            formValues["attendance.require_face_for_checkout"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Izin Fallback Tanpa Wajah Terdaftar
                            </label>
                            <div className="flex items-center gap-2">
                                <Select
                                    value={String(formValues["attendance.allow_fallback_without_enrollment"] ?? "false")}
                                    onChange={(e) =>
                                        handleChange(
                                            "attendance.allow_fallback_without_enrollment",
                                            e.target.value === "true"
                                        )
                                    }
                                >
                                    <option value="false">Tolak (Wajib Terdaftar)</option>
                                    <option value="true">Izinkan Lewat Review Manual</option>
                                </Select>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.allow_fallback_without_enrollment"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.allow_fallback_without_enrollment",
                                            formValues["attendance.allow_fallback_without_enrollment"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* KARTU 4: KEAMANAN & RETENSI DATA */}
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Lock className="w-4 h-4 text-indigo-600" />
                        <CardTitle>4. Keamanan Percobaan & Retensi Data</CardTitle>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Maksimal Percobaan Gagal / Jam
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    value={Number(formValues["attendance.max_failed_attempts_per_hour"] ?? 5)}
                                    onChange={(e) =>
                                        handleChange(
                                            "attendance.max_failed_attempts_per_hour",
                                            Number(e.target.value)
                                        )
                                    }
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.max_failed_attempts_per_hour"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.max_failed_attempts_per_hour",
                                            formValues["attendance.max_failed_attempts_per_hour"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Masa Simpan Foto Absen (Hari)
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    value={Number(formValues["attendance.photo_retention_days"] ?? 90)}
                                    onChange={(e) =>
                                        handleChange("attendance.photo_retention_days", Number(e.target.value))
                                    }
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "attendance.photo_retention_days"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "attendance.photo_retention_days",
                                            formValues["attendance.photo_retention_days"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">
                                Retensi Wajah Karyawan Resign (Hari)
                            </label>
                            <div className="flex items-center gap-2">
                                <Input
                                    type="number"
                                    value={Number(formValues["face.retention_days_after_resign"] ?? 30)}
                                    onChange={(e) =>
                                        handleChange("face.retention_days_after_resign", Number(e.target.value))
                                    }
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    isLoading={savingKey === "face.retention_days_after_resign"}
                                    onClick={() =>
                                        checkConsequenceAndSave(
                                            "face.retention_days_after_resign",
                                            formValues["face.retention_days_after_resign"]
                                        )
                                    }
                                >
                                    <Save className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* KARTU 5: AMBANG KUALITAS WAJAH (READ-ONLY, RESOLUSI K-02) */}
            <Card className="border-indigo-200">
                <CardHeader>
                    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                        <div className="flex items-center gap-2">
                            <Activity className="w-4 h-4 text-indigo-600" />
                            <CardTitle>5. Ambang Kualitas Wajah (Inference Live Status #73)</CardTitle>
                        </div>

                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => refetchQuality()}
                            disabled={isQualityFetching}
                        >
                            <RefreshCw className={`w-3.5 h-3.5 mr-1 ${isQualityFetching ? "animate-spin" : ""}`} />
                            Periksa Ulang
                        </Button>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <p className="text-xs text-gray-500 leading-relaxed">
                        Ambang batas kualitas di bawah ini bersumber langsung dari kontainer inferensi AI.
                        Nilai di database (<em>app_settings</em>) dibandingkan secara live terhadap nilai aktif
                        di engine. Baris yang tidak sinkron menandakan container inference perlu disinkronkan.
                    </p>

                    {isQualityLoading ? (
                        <div className="py-8 text-center text-xs text-gray-400">
                            Memeriksa status container inferensi...
                        </div>
                    ) : !qualityStatus?.inference_ready ? (
                        <div className="p-3.5 bg-gray-50 rounded-xl border border-gray-200 flex items-center gap-3 text-xs text-gray-600">
                            <Badge variant="neutral" size="sm">
                                Tidak bisa diperiksa saat ini
                            </Badge>
                            <span>
                                Kontainer inferensi tidak merespons (<em>checked_at: null</em>). Pastikan service face-engine sedang berjalan.
                            </span>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full text-xs text-left">
                                <thead className="bg-gray-50/80 text-gray-600 border-b border-gray-200/80">
                                    <tr>
                                        <th className="px-3.5 py-2 font-semibold">Parameter Kualitas</th>
                                        <th className="px-3.5 py-2 font-semibold">Nilai di Database</th>
                                        <th className="px-3.5 py-2 font-semibold">Nilai Aktif Inference</th>
                                        <th className="px-3.5 py-2 font-semibold text-right">Status Sinkronisasi</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-100 font-mono">
                                    {qualityStatus.thresholds?.map((item) => {
                                        const isSynced = item.in_sync;
                                        return (
                                            <tr key={item.key} className={isSynced ? "" : "bg-amber-50/40"}>
                                                <td className="px-3.5 py-2.5 font-sans font-medium text-gray-800">
                                                    {item.key}
                                                </td>
                                                <td className="px-3.5 py-2.5 text-gray-700">{item.db_value}</td>
                                                <td className="px-3.5 py-2.5 font-bold text-gray-900">{item.inference_value}</td>
                                                <td className="px-3.5 py-2.5 text-right font-sans">
                                                    {isSynced ? (
                                                        <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700">
                                                            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                                                            Sinkron
                                                        </span>
                                                    ) : (
                                                        <div className="inline-flex flex-col items-end">
                                                            <Badge variant="warning" size="sm">
                                                                <AlertTriangle className="w-3 h-3 mr-1" />
                                                                Tidak Sinkron
                                                            </Badge>
                                                            <span className="text-[10px] text-amber-700 mt-0.5">
                                                                Nilai aktif adalah {item.inference_value}
                                                            </span>
                                                        </div>
                                                    )}
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Consequence Dialog */}
            <ConfirmDialog
                open={Boolean(pendingUpdate)}
                title="Konfirmasi Perubahan Kebijakan"
                description={pendingUpdate?.warningMessage ?? ""}
                confirmText="Tetap Simpan Perubahan"
                variant="warning"
                isLoading={Boolean(savingKey)}
                onConfirm={() => {
                    if (pendingUpdate) performUpdate(pendingUpdate.key, pendingUpdate.value);
                }}
                onClose={() => setPendingUpdate(null)}
            />
        </div>
    );
}
