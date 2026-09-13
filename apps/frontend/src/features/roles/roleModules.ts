import type { Permission } from "../../types/api";

export interface PermissionDefinition {
    name: string;
    label: string;
    description: string;
}

export interface ModuleDefinition {
    key: string;
    title: string;
    description: string;
    color: {
        bg: string;
        text: string;
        border: string;
        badge: string;
        light: string;
    };
    permissions: PermissionDefinition[];
}

export const MODULE_DEFINITIONS: ModuleDefinition[] = [
    {
        key: "attendance",
        title: "Presensi & Absensi",
        description: "Pencatatan kehadiran karyawan, verifikasi review foto & geofence, serta ekspor laporan rekap.",
        color: {
            bg: "bg-emerald-50",
            text: "text-emerald-700",
            border: "border-emerald-200",
            badge: "bg-emerald-100 text-emerald-800",
            light: "hover:bg-emerald-50/50",
        },
        permissions: [
            {
                name: "attendance.checkin",
                label: "Presensi / Check-in & Check-out",
                description: "Melakukan pencatatan kehadiran mandiri melalui portal atau kamera",
            },
            {
                name: "attendance.read_self",
                label: "Lihat Riwayat Presensi Sendiri",
                description: "Melihat riwayat kehadiran dan jam kerja milik akun sendiri",
            },
            {
                name: "attendance.read_all",
                label: "Lihat Seluruh Data Absensi",
                description: "Melihat data kehadiran dan status absensi semua karyawan perusahaan",
            },
            {
                name: "attendance.approve",
                label: "Verifikasi Antrian Review",
                description: "Menyetujui (Approve) atau menolak (Reject) absensi dengan status pending review",
            },
            {
                name: "attendance.export",
                label: "Ekspor Laporan Rekap",
                description: "Mengunduh rekapitulasi kehadiran dan data keterlambatan dalam format CSV/Excel",
            },
        ],
    },
    {
        key: "employee",
        title: "Karyawan & Pegawai",
        description: "Manajemen data induk karyawan, NIK, penempatan jabatan, dan status kepegawaian.",
        color: {
            bg: "bg-blue-50",
            text: "text-blue-700",
            border: "border-blue-200",
            badge: "bg-blue-100 text-blue-800",
            light: "hover:bg-blue-50/50",
        },
        permissions: [
            {
                name: "employee.read",
                label: "Lihat Semua Karyawan",
                description: "Melihat daftar lengkap dan detail profil seluruh karyawan",
            },
            {
                name: "employee.read_self",
                label: "Lihat Profil Sendiri",
                description: "Melihat informasi kepegawaian milik diri sendiri",
            },
            {
                name: "employee.create",
                label: "Tambah Karyawan Baru",
                description: "Mendaftarkan karyawan baru ke dalam sistem kepegawaian",
            },
            {
                name: "employee.update",
                label: "Ubah Data Karyawan",
                description: "Memperbarui identitas, kontak, jabatan, atau departemen karyawan",
            },
            {
                name: "employee.delete",
                label: "Hapus / Nonaktifkan Karyawan",
                description: "Menghapus (soft-delete) atau menonaktifkan data karyawan dari sistem",
            },
        ],
    },
    {
        key: "face",
        title: "Biometrik Wajah (AI)",
        description: "Pendaftaran foto referensi biometrik wajah, verifikasi kualitas, dan regenerasi embedding model AI.",
        color: {
            bg: "bg-violet-50",
            text: "text-violet-700",
            border: "border-violet-200",
            badge: "bg-violet-100 text-violet-800",
            light: "hover:bg-violet-50/50",
        },
        permissions: [
            {
                name: "face.enroll_self",
                label: "Pendaftaran Wajah Sendiri",
                description: "Mengambil foto wajah referensi untuk akun diri sendiri",
            },
            {
                name: "face.read_self",
                label: "Lihat Foto Wajah Sendiri",
                description: "Melihat status dan galeri foto referensi wajah diri sendiri",
            },
            {
                name: "face.enroll_any",
                label: "Pendaftaran Wajah Siapapun",
                description: "Mengambil dan mengunggah foto wajah referensi karyawan lain (Admin / Operator)",
            },
            {
                name: "face.read_any",
                label: "Lihat Galeri Wajah Seluruh Karyawan",
                description: "Melihat riwayat dan galeri foto referensi biometrik semua pegawai",
            },
            {
                name: "face.delete_any",
                label: "Hapus Foto Referensi Wajah",
                description: "Menonaktifkan atau menghapus foto biometrik wajah karyawan",
            },
            {
                name: "face.reindex",
                label: "Jalankan Reindex Model Wajah",
                description: "Memicu proses batch regenerasi vektor embedding saat upgrade model AI",
            },
        ],
    },
    {
        key: "location",
        title: "Lokasi Kantor & Geofence",
        description: "Pengaturan titik koordinat GPS kantor dan radius geofencing tempat presensi sah.",
        color: {
            bg: "bg-amber-50",
            text: "text-amber-700",
            border: "border-amber-200",
            badge: "bg-amber-100 text-amber-800",
            light: "hover:bg-amber-50/50",
        },
        permissions: [
            {
                name: "location.read",
                label: "Lihat Lokasi Kantor",
                description: "Melihat daftar titik lokasi kantor dan radius geofence",
            },
            {
                name: "location.create",
                label: "Tambah Lokasi Kantor",
                description: "Menambahkan kantor cabang atau lokasi penugasan baru",
            },
            {
                name: "location.update",
                label: "Ubah Lokasi & Radius",
                description: "Memperbarui titik latitude, longitude, dan radius toleransi jarak",
            },
            {
                name: "location.delete",
                label: "Hapus Lokasi Kantor",
                description: "Menghapus lokasi kantor dari daftar geofence presensi",
            },
        ],
    },
    {
        key: "user",
        title: "Pengguna Sistem & Akun",
        description: "Pengelolaan akun login (email, kata sandi, status keaktifan, dan penguncian).",
        color: {
            bg: "bg-indigo-50",
            text: "text-indigo-700",
            border: "border-indigo-200",
            badge: "bg-indigo-100 text-indigo-800",
            light: "hover:bg-indigo-50/50",
        },
        permissions: [
            {
                name: "user.read",
                label: "Lihat Daftar Pengguna",
                description: "Melihat seluruh akun login yang terdaftar dalam sistem FaceClock",
            },
            {
                name: "user.create",
                label: "Tambah Pengguna Baru",
                description: "Mendaftarkan akun login baru untuk pegawai atau staf admin",
            },
            {
                name: "user.update",
                label: "Ubah Data & Status Pengguna",
                description: "Memperbarui email atau mengaktifkan/menonaktifkan akun pengguna",
            },
            {
                name: "user.delete",
                label: "Hapus Akun Pengguna",
                description: "Menonaktifkan permanen (soft-delete) akun pengguna dari sistem",
            },
            {
                name: "user.assign_role",
                label: "Atur Peran Pengguna",
                description: "Menetapkan atau mengganti peran akses (roles) milik pengguna",
            },
            {
                name: "user.reset_password",
                label: "Reset Kata Sandi Pengguna",
                description: "Mereset kata sandi pengguna lain atau membuatkan kata sandi sementara",
            },
        ],
    },
    {
        key: "role",
        title: "Peran & Hak Akses (RBAC)",
        description: "Pengaturan kelompok hak akses pengguna, pembuatan peran baru, dan katalog izin.",
        color: {
            bg: "bg-purple-50",
            text: "text-purple-700",
            border: "border-purple-200",
            badge: "bg-purple-100 text-purple-800",
            light: "hover:bg-purple-50/50",
        },
        permissions: [
            {
                name: "role.read",
                label: "Lihat Daftar Peran",
                description: "Melihat daftar peran sistem dan peran kustom beserta jumlah pengguna",
            },
            {
                name: "role.create",
                label: "Buat Peran Baru",
                description: "Membuat kelompok peran kustom baru dengan wewenang khusus",
            },
            {
                name: "role.update",
                label: "Ubah Informasi Peran",
                description: "Mengubah nama tampilan atau keterangan deskripsi peran",
            },
            {
                name: "role.delete",
                label: "Hapus Peran",
                description: "Menghapus peran kustom yang sudah tidak digunakan",
            },
            {
                name: "role.assign_permission",
                label: "Atur Matriks Izin Peran",
                description: "Menentukan izin apa saja yang aktif untuk suatu peran",
            },
            {
                name: "permission.read",
                label: "Lihat Katalog Izin",
                description: "Melihat daftar referensi wewenang (permission) bawaan sistem",
            },
        ],
    },
    {
        key: "settings",
        title: "Pengaturan Sistem",
        description: "Konfigurasi parameter ambang toleransi wajah, waktu sesi, dan kebijakan aplikasi.",
        color: {
            bg: "bg-slate-50",
            text: "text-slate-700",
            border: "border-slate-200",
            badge: "bg-slate-100 text-slate-800",
            light: "hover:bg-slate-50/50",
        },
        permissions: [
            {
                name: "settings.read",
                label: "Baca Pengaturan Sistem",
                description: "Melihat konfigurasi threshold biometrik dan parameter operasional",
            },
            {
                name: "settings.update",
                label: "Ubah Pengaturan Sistem",
                description: "Memperbarui konfigurasi toleransi AI, kuota foto, atau pengaturan sistem",
            },
        ],
    },
    {
        key: "audit",
        title: "Jejak Audit & Keamanan",
        description: "Catatan riwayat aktivitas operasional penting dan audit kepatuhan data.",
        color: {
            bg: "bg-rose-50",
            text: "text-rose-700",
            border: "border-rose-200",
            badge: "bg-rose-100 text-rose-800",
            light: "hover:bg-rose-50/50",
        },
        permissions: [
            {
                name: "audit.read",
                label: "Lihat Jejak Audit (Audit Log)",
                description: "Melihat catatan aktivitas login, perubahan data penting, dan log keamanan",
            },
        ],
    },
];

export interface GroupedModulePermissions {
    module: ModuleDefinition;
    permissions: (Permission & { friendlyLabel?: string; friendlyDesc?: string })[];
}

export function groupPermissionsByDefinedModules(allPerms: Permission[]): GroupedModulePermissions[] {
    const permMap = new Map<string, Permission>();
    allPerms.forEach((p) => permMap.set(p.name, p));

    const result: GroupedModulePermissions[] = [];

    MODULE_DEFINITIONS.forEach((mod) => {
        const modPerms: (Permission & { friendlyLabel?: string; friendlyDesc?: string })[] = [];

        mod.permissions.forEach((def) => {
            const found = permMap.get(def.name);
            if (found) {
                modPerms.push({
                    ...found,
                    friendlyLabel: def.label,
                    friendlyDesc: def.description || found.description,
                });
            }
        });

        if (modPerms.length > 0) {
            result.push({
                module: mod,
                permissions: modPerms,
            });
        }
    });

    // Handle any orphan permissions not in predefined list
    const knownNames = new Set(
        MODULE_DEFINITIONS.flatMap((m) => m.permissions.map((p) => p.name))
    );
    const orphans = allPerms.filter((p) => !knownNames.has(p.name));

    if (orphans.length > 0) {
        result.push({
            module: {
                key: "other",
                title: "Izin Lainnya",
                description: "Izin sistem tambahan yang belum diklasifikasikan ke modul khusus.",
                color: {
                    bg: "bg-gray-50",
                    text: "text-gray-700",
                    border: "border-gray-200",
                    badge: "bg-gray-100 text-gray-800",
                    light: "hover:bg-gray-50/50",
                },
                permissions: orphans.map((o) => ({
                    name: o.name,
                    label: o.name,
                    description: o.description || "",
                })),
            },
            permissions: orphans.map((o) => ({
                ...o,
                friendlyLabel: o.name,
                friendlyDesc: o.description,
            })),
        });
    }

    return result;
}

export interface PresetTemplate {
    id: string;
    name: string;
    description: string;
    pattern: (permName: string) => boolean;
}

export const ROLE_PRESETS: PresetTemplate[] = [
    {
        id: "full",
        name: "Akses Penuh (Administrator)",
        description: "Centang seluruh izin di semua modul.",
        pattern: () => true,
    },
    {
        id: "hr_manager",
        name: "HR / Manajer Personalia",
        description: "Modul karyawan lengkap, persetujuan absensi, laporan, geofence, dan audit.",
        pattern: (p) =>
            p.startsWith("employee.") ||
            p.startsWith("attendance.") ||
            p === "location.read" ||
            p === "face.read_any" ||
            p === "face.enroll_any" ||
            p === "audit.read",
    },
    {
        id: "supervisor",
        name: "Supervisor Cabang / Lapangan",
        description: "Pantau absensi tim, approve antrian review, ekspor laporan, dan lihat lokasi.",
        pattern: (p) =>
            p === "attendance.read_all" ||
            p === "attendance.approve" ||
            p === "attendance.export" ||
            p === "attendance.checkin" ||
            p === "attendance.read_self" ||
            p === "employee.read" ||
            p === "location.read",
    },
    {
        id: "employee",
        name: "Pegawai Mandiri (Karyawan)",
        description: "Hanya akses presensi mandiri, riwayat pribadi, profil sendiri, dan foto sendiri.",
        pattern: (p) =>
            p === "attendance.checkin" ||
            p === "attendance.read_self" ||
            p === "employee.read_self" ||
            p === "face.enroll_self" ||
            p === "face.read_self",
    },
    {
        id: "auditor",
        name: "Auditor / Hanya Lihat (Read-Only)",
        description: "Hanya izin membaca/melihat data di seluruh modul tanpa hak modifikasi.",
        pattern: (p) =>
            p.endsWith(".read") ||
            p.endsWith(".read_all") ||
            p.endsWith(".read_self"),
    },
];
