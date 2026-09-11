import { Smartphone, Clock } from "lucide-react";

export function EmployeePlaceholderPage() {
    return (
        <div className="max-w-2xl mx-auto py-12 text-center">
            <div className="w-14 h-14 rounded-2xl bg-indigo-50 border border-indigo-200 text-indigo-600 flex items-center justify-center mx-auto mb-4">
                <Smartphone className="w-7 h-7" />
            </div>
            <h1 className="text-xl font-bold text-gray-900">Portal Karyawan (Self-Service)</h1>
            <p className="text-xs text-gray-600 mt-2 max-w-md mx-auto leading-relaxed">
                Fitur absensi mandiri, kamera verifikasi wajah real-time, dan riwayat absensi pribadi karyawan
                sedang dipersiapkan pada <strong>Fase 6</strong>.
            </p>

            <div className="mt-8 p-4 bg-white rounded-xl border border-gray-200/80 shadow-xs max-w-md mx-auto text-left">
                <div className="flex items-center gap-2 text-xs font-semibold text-gray-800 mb-2">
                    <Clock className="w-4 h-4 text-indigo-600" />
                    <span>Agenda Rilis Fase 6:</span>
                </div>
                <ul className="text-xs text-gray-600 space-y-1.5 list-disc list-inside">
                    <li>PWA Mobile-First dengan dukungan Liveness Detection</li>
                    <li>Geo-Fencing GPS client-side & auto-checkin</li>
                    <li>Notifikasi & status persetujuan kehadiran mandiri</li>
                </ul>
            </div>
        </div>
    );
}
