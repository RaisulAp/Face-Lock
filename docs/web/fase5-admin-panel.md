# Dokumentasi Arsitektur Frontend — FaceClock Web Admin Panel (Fase 5)

## 1. Ikhtisar & Teknologi

Aplikasi web FaceClock dibangun sebagai Single Page Application (SPA) modern yang tangguh untuk administrasi, monitoring operasional presensi, dan tata kelola biometrik wajah.

### Stack Teknologi
- **Framework UI:** React 19 + TypeScript (~6.0)
- **Build Tool & Bundler:** Vite 8
- **Styling:** Tailwind CSS v4
- **State Management & Server Cache:** TanStack Query v5 (`@tanstack/react-query`)
- **Perutean (Routing):** React Router v7 (`react-router-dom`)
- **Peta Geofencing:** Leaflet 1.9 + OpenStreetMap tiles
- **Ikonografi:** Lucide React
- **Standar Kode:** TypeScript Strict Mode, `verbatimModuleSyntax`, `noUnusedLocals`, `noUnusedParameters`

---

## 2. Matriks Rute & Hak Akses (Route × Permission Matrix)

Seluruh navigasi dan akses halaman diatur secara ketat berdasarkan permissions yang diemban pengguna, bukan berdasarkan nama peran (Role Name):

| Path | Halaman / Komponen | Permission Wajib | Deskripsi Fungsional |
|---|---|---|---|
| `/login` | `LoginPage` | *Public* | Layar autentikasi kredensial (email & password) |
| `/` | `DashboardPage` | Terotentikasi | Ringkasan metrik harian, aktivitas presensi terbaru, status inferensi |
| `/attendances` | `AttendanceListPage` | `attendance.read_all` | Daftar seluruh transaksi kehadiran dengan penyaringan terperinci |
| `/attendances/:id` | `AttendanceDetailPage` | `attendance.read_all` | Pratinjau detail presensi, komparasi foto wajah, geofence map |
| `/attendances/pending` | `PendingQueuePage` | `attendance.review` | Antrian presensi yang memerlukan persetujuan manual (Aturan B11) |
| `/reports` | `ReportsPage` | `attendance.export` | Laporan agregat (#71) & unduhan streaming file CSV ber-BOM (#72) |
| `/employees` | `EmployeesPage` | `employee.read` | Direktori master data karyawan dan status biometrik |
| `/employees/new` | `EmployeeFormPage` | `employee.create` | Pendaftaran data karyawan baru |
| `/employees/:id` | `EmployeeDetailPage` | `employee.read` | Profil lengkap karyawan, riwayat consent, dan status enrollment |
| `/employees/:id/edit` | `EmployeeFormPage` | `employee.update` | Perubahan data profil karyawan |
| `/employees/:id/face` | `EmployeeFacePage` | `face.read_any` | Galeri referensi template biometrik wajah karyawan |
| `/consents` | `ConsentsPage` | `consent.read` | Monitoring persetujuan biometrik UU PDP No. 27/2022 |
| `/locations` | `LocationsPage` | `location.read` | Manajemen titik koordinat kantor dan radius geofence |
| `/settings` | `SettingsPage` | `settings.read` | Konfigurasi sistem dan live probe ambang inferensi (#73) |
| `/face/reindex` | `FaceReindexPage` | `face.reindex` | Rekalkulasi embedding wajah latar belakang untuk model baru |
| `/users` | `UsersPage` | `user.read` | Manajemen akun login, peran, dan reset password |
| `/roles` | `RolesPage` | `role.read` | Tata kelola peran sistem dan matriks hak akses |
| `/permissions` | `PermissionsPage` | `permission.read` | Katalog transparan seluruh hak akses dalam sistem |
| `/security/attempts` | `AttemptsPage` | `attempt.read` | Audit log percobaan presensi dan analisis anomali |
| `/audit/logs` | `AuditLogsPage` | `audit.read` | Jejak audit kepatuhan atas perubahan data |
| `/403` | `ForbiddenPage` | Terotentikasi | Tampilan ramah penolakan izin akses |
| `*` | `NotFoundPage` | *Public* | Tampilan halaman tidak ditemukan (empty state tenang) |

---

## 3. Siklus Hidup Token & Sinkronisasi Multi-Tab (Keputusan D23)

Untuk memitigasi risiko serangan XSS sekaligus mencegah kegagalan fatal *token reuse* saat rotasi token, FaceClock mengimplementasikan sistem tiga lapis:

```
[Tab 1 Browser]                            [Tab 2 Browser]
      │                                          │
401 Unauthorized                            401 Unauthorized
      │                                          │
Request Lock 'faceclock-refresh'           Request Lock 'faceclock-refresh'
(Acquired)                                  (Waiting in Queue...)
      │                                          │
POST /auth/refresh                               │
      │                                          │
Simpan New Refresh Token                         │
Set New Access Token di Memori                   │
Broadcast 'tokens-rotated' ─────────────────────►│
Release Lock                                (Acquired)
                                            Cek Token: Sudah Baru!
                                            Gunakan token baru tanpa re-call API.
```

1. **In-Memory Access Token:** Access token disimpan hanya dalam memori JavaScript runtime modul. Jika tab ditutup, token otomatis hilang dari memori.
2. **Web Locks Single-Flight:** Saat beberapa komponen mengirim query bersamaan dan access token kedaluwarsa, pemanggilan refresh dibungkus oleh `navigator.locks.request('faceclock-refresh')`. Hanya tepat satu request refresh yang dikirimkan ke server.
3. **BroadcastChannel Synchronization:** Menggunakan `new BroadcastChannel('faceclock-auth')` untuk memberi sinyal rotasi ke tab lain yang sedang terbuka tanpa menyertakan nilai token di payload channel.

---

## 4. Penanganan Media & Penghapusan Objek Memori (Keputusan D14)

Foto wajah presensi dan referensi dilindungi autentikasi dan tidak boleh bocor ke cache publik atau memicu kebocoran memori browser:

- Komponen `<AuthImage src="/api/v1/attendances/:id/photo" />` memanfaatkan hook `useAuthBlob` berbasis TanStack Query untuk mengunduh konten biner gambar (`Blob`).
- Di dalam lifecycle komponen:
  ```ts
  useEffect(() => {
    if (data?.blob) {
      const url = URL.createObjectURL(data.blob);
      setObjectUrl(url);
      return () => {
        URL.revokeObjectURL(url); // Membersihkan referensi memori seketika saat unmount
      };
    }
  }, [data?.blob]);
  ```
- Penanganan status HTTP 410 (`Gone`): Ketika foto absensi telah dibersihkan oleh worker retensi data, komponen menampilkan status informatif netral tanpa merusak tampilan data teks transaksi presensi.

---

## 5. Pengunduhan Laporan & Streaming CSV (Keputusan D20)

Ekspor data presensi dirancang untuk menangani dataset besar tanpa memicu crash kehabisan memori (OOM) pada server maupun browser:

- Server mengalirkan data baris demi baris diawali dengan Byte Order Mark UTF-8 (`\xEF\xBB\xBF`) agar langsung kompatibel dengan Microsoft Excel di sistem operasi Windows.
- Fungsi client `downloadFile(blob, filename)` membuat elemen anchor sintetis `<a download="...">`, memicu klik unduh, dan segera melepaskan object URL terkait.
- Jika query menghasilkan jumlah baris di atas `attendance.export_max_rows` (100.000 baris), server mengembalikan HTTP 422 dengan pesan edukatif menyarankan penyempitan filter tanggal.

---

## 6. Penegakan Aturan Bisnis Khusus di UI

### Aturan B11 (Supervisor Self-Review Guard)
Supervisor atau admin tidak boleh menyetujui atau menolak absensi milik dirinya sendiri. Pada `ReviewDrawer.tsx`:
```tsx
const isSelfReview = user?.id && user.id === attendance.user_id;

{isSelfReview ? (
  <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
    Aturan B11: Anda tidak dapat menyetujui absensi milik Anda sendiri.
  </div>
) : (
  <div className="flex gap-2">
    <Button variant="danger" onClick={handleReject}>Tolak</Button>
    <Button variant="primary" onClick={handleApprove}>Setujui</Button>
  </div>
)}
```

### Aturan B01 / REV-PERM-01 (Pencegahan Eskalasi Hak Istimewa)
Pengguna tidak diizinkan memberikan hak akses (permission) ke peran lain melebihi hak akses yang dimiliki oleh dirinya sendiri. Pada `RolesPage.tsx`, daftar checkbox izin dibatasi secara ketat berdasarkan izin yang aktif pada pengguna yang sedang login.
