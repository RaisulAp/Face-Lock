# FaceClock API Specification — Fase 4: Attendance Engine, Geofencing, Biometric Verification & Review Workflow

Dokumen ini mendefinisikan kontrak 18 endpoint HTTP API Fase 4 (Endpoints #53–#70) untuk backend FaceClock (`apps/faceclock-api`).
Sesuai konvensi standar arsitektur sistem (`docs/adr/0003-api-conventions.md`):
- Berakar di prefix `/api/v1`
- Menggunakan JSON envelope standar: `{ "data": ... }` untuk sukses, dan `{ "error": { "code": "...", "message": "...", "details": [...] } }` untuk gagal
- Mengirimkan header korelasi `X-Request-ID` di setiap response
- Menegakkan otorisasi berbasis Role-Based Access Control (RBAC) dengan prinsip **default-deny**
- Mempertahankan arsitektur keamanan **HttpOnly Cookie-Based Auth + Anti-CSRF Shield**
- Menegakkan **Anti-Score Leakage (Rule B7 / D16)**: Karyawan biasa TIDAK PERNAH menerima skor biometrik (`matched_similarity` atau `threshold_used`), melainkan hanya status kualitatif (`hints`)
- Menegakkan **Anti-Replay Foto (Rule B4 / § 2.7c)**: Pencegahan duplikasi foto via SHA-256 hash detection dalam rentang waktu yang dapat dikonfigurasi (`attendance.duplicate_photo_window_days`)
- Menegakkan **Anti-Self Review (Rule B11)**: Reviewer tidak dapat menyetujui atau menolak absensi milik dirinya sendiri
- Menegakkan **Sliding-Window Failed Attempts Rate Limiting (Rule B12)**: Pembatasan percobaan gagal per karyawan per jam
- Menyediakan daemon latar belakang pembersihan foto harian (**Retention Worker**) dengan proteksi PostgreSQL Advisory Lock `20260904`.

---

## Daftar Isi
1. [Prinsip Desain & Aturan Bisnis Fase 4](#1-prinsip-desain--aturan-bisnis-fase-4)
2. [Arsitektur Anti-Score Leakage (Rule B7 / D16)](#2-arsitektur-anti-score-leakage-rule-b7--d16)
3. [Geofencing Murni (WGS-84 Haversine)](#3-geofencing-murni-wgs-84-haversine)
4. [Subsystem Presensi Karyawan (#53–#58)](#4-subsystem-presensi-karyawan-5358)
5. [Subsystem Monitoring & Review Admin (#59–#65)](#5-subsystem-monitoring--review-admin-5965)
6. [Subsystem Master Lokasi Kantor (#66–#70)](#6-subsystem-master-lokasi-kantor-6670)
7. [Daemon Background Retention Worker](#7-daemon-background-retention-worker)
8. [Katalog Kode Error Fase 4](#8-katalog-kode-error-fase-4)

---

## 1. Prinsip Desain & Aturan Bisnis Fase 4

### 1.1 Integritas Domain Absensi (`attendances`)
Tabel `attendances` menegakkan 6 *domain check constraints* setingkat database:
1. `attendances_type_chk`: `type IN ('check_in', 'check_out')`
2. `attendances_status_chk`: `status IN ('approved', 'rejected', 'pending_review')`
3. `attendances_method_chk`: `method IN ('face_verified', 'fallback_manual', 'fallback_offline', 'system_timeout')`
4. `attendances_fallback_chk`: Ketika `method = 'fallback_manual'`, kolom `fallback_reason` wajib terisi dan bernilai salah satu dari: `'camera_failure'`, `'outside_geofence'`, `'face_unrecognized'`, `'device_unsupported'`, `'system_timeout'`, `'other'`.
5. `attendances_review_chk`: Ketika `status IN ('approved', 'rejected')` dan `method = 'fallback_manual'`, kolom `reviewed_by` dan `reviewed_at` wajib terisi.
6. `attendances_hash_chk`: `photo_sha256 ~ '^[0-9a-f]{64}$'`

Selain itu, partial unique index menjamin tepat satu `check_in` dan satu `check_out` yang sah per karyawan per tanggal kerja (`work_date`):
- `attendances_one_checkin_per_day`: `(employee_id, work_date) WHERE type = 'check_in' AND status <> 'rejected'`
- `attendances_one_checkout_per_day`: `(employee_id, work_date) WHERE type = 'check_out' AND status <> 'rejected'`

### 1.2 Telemetri Percobaan (`attendance_attempts`)
Setiap upaya presensi (baik sukses maupun gagal pada tahap apa pun) dicatat ke dalam tabel telemetri `attendance_attempts` dengan salah satu dari 10 nilai `outcome` terstandarisasi:
- `'matched'`: Wajah terverifikasi dan memenuhi ambang batas kemiripan.
- `'below_threshold'`: Wajah terdeteksi namun skor kemiripan di bawah ambang batas.
- `'face_not_usable'`: Kualitas foto tidak memenuhi standar (tidak ada wajah, blur, multi-face).
- `'inference_unavailable'`: Layanan inferensi biometrik tidak dapat dihubungi.
- `'no_reference'`: Karyawan belum memiliki data referensi wajah terdaftar.
- `'geofence_rejected'`: Posisi GPS berada di luar radius kantor yang diizinkan.
- `'rule_rejected'`: Pelanggaran aturan jadwal (mis. sudah check-in, check-out terlalu cepat).
- `'duplicate_photo'`: Terdeteksi pengiriman ulang foto yang identik (anti-replay).
- `'rate_limited'`: Karyawan melebihi batas percobaan gagal dalam jendela 1 jam.
- `'manual_mode'`: Presensi berhasil dibuat melalui mode manual/fallback yang memerlukan review admin.

---

## 2. Arsitektur Anti-Score Leakage (Rule B7 / D16)

Untuk mencegah serangan *hill-climbing* atau penyesuaian wajah terpandu (*score-guided spoofing*), sistem memisahkan representasi data presensi ke dalam dua Data Transfer Object (DTO) yang tegas:

### 2.1 `EmployeeAttendanceDTO` (Response Karyawan)
Dipakai pada:
- `POST /api/v1/attendance/clock-in`
- `POST /api/v1/attendance/clock-out`
- `GET /api/v1/attendance/my/today`
- `GET /api/v1/attendance/my/history`

**Kolom Dikecualikan**: `matched_similarity`, `threshold_used`, `photo_sha256`.
**Kolom Disediakan**: `id`, `work_date`, `type`, `status`, `method`, `server_timestamp`, `is_late`, `late_minutes`, `is_early_leave`, `early_leave_minutes`, `office_location_id`, `office_name`, `geofence_status`, `distance_meter`, `fallback_reason`, `notes`, `hints`.

### 2.2 `AdminAttendanceDTO` (Response Admin / Supervisor)
Dipakai pada:
- `GET /api/v1/attendance/records`
- `GET /api/v1/attendance/records/{id}`

Menampilkan seluruh data `EmployeeAttendanceDTO` ditambah: `employee_number`, `full_name`, `matched_similarity`, `threshold_used`, `photo_sha256`, `reviewed_by`, `reviewer_name`, `reviewed_at`, `review_notes`.

---

## 3. Geofencing Murni (WGS-84 Haversine)

Sistem menggunakan pustaka murni internal (`internal/geo`) yang mengimplementasikan formula Great-Circle Distance Haversine:
$$a = \sin^2\left(\frac{\Delta\phi}{2}\right) + \cos(\phi_1)\cos(\phi_2)\sin^2\left(\frac{\Delta\lambda}{2}\right)$$
$$c = 2 \cdot \text{atan2}\left(\sqrt{a}, \sqrt{1-a}\right)$$
$$d = R \cdot c \quad (R = 6.371.000\text{ meter})$$

### Aturan Evaluasi Radius (`geo.Evaluate`):
- `status = "inside"`: Jarak titik pengguna ke pusat kantor $\le$ `radius_meter` kantor.
- `status = "outside"`: Jarak titik pengguna ke seluruh kantor aktif $>$ `radius_meter`.
- `status = "unavailable"`: Nilai latitude / longitude bernilai null atau koordinat tidak valid (mis. lat $< -90$ atau lat $> 90$).
- Pengguna dievaluasi terhadap seluruh kantor aktif non-terhapus (`is_active = true AND deleted_at IS NULL`), dan kantor terdekat (`nearest_office`) otomatis dipilih.

---

## 4. Subsystem Presensi Karyawan (#53–#58)

### 4.1 POST `/api/v1/attendance/clock-in` (#53) & POST `/api/v1/attendances/clock-in` (#53b)
- **Permission**: `attendance.checkin`
- **Request Form-Data** atau **JSON**:
```json
{
  "photo_base64": "...",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "accuracy": 12.5,
  "client_time": "2026-09-10T08:00:00Z",
  "notes": "Hadir tepat waktu",
  "device_info": "Pixel 8 / Chrome 120",
  "allow_fallback": false,
  "fallback_reason": null,
  "fallback_note": null
}
```
- **Response 201 Created**:
```json
{
  "data": {
    "id": "7b7e6822-0df0-4b5c-b17b-d2c49df5d539",
    "employee_id": "c1f7b04a-4d22-4467-bc22-b5eef9eb7e1a",
    "work_date": "2026-09-10",
    "type": "check_in",
    "status": "approved",
    "method": "face_verified",
    "server_timestamp": "2026-09-10T01:00:15Z",
    "is_late": false,
    "late_minutes": 0,
    "is_early_leave": false,
    "early_leave_minutes": 0,
    "office_location_id": "90e0e6c5-e51c-4b5f-a365-c7e6c4ea5768",
    "office_name": "Kantor Pusat",
    "geofence_status": "inside",
    "distance_meter": 24.5,
    "fallback_reason": null,
    "fallback_note": null,
    "notes": "Hadir tepat waktu",
    "hints": []
  }
}
```

### 4.2 POST `/api/v1/attendance/clock-out` (#54) & POST `/api/v1/attendances/clock-out` (#54b)
- **Permission**: `attendance.checkin`
- **Aturan B6**: Wajib memiliki `check_in` yang sah pada `work_date` bersangkutan, dan jarak waktu minimal dari check-in harus memenuhi `attendance_checkout_min_interval_minutes` (default 5 menit).

### 4.3 GET `/api/v1/attendance/context` (#55)
- **Permission**: `attendance.checkin`
- Mengambil informasi kontekstual sebelum karyawan melakukan presensi:
```json
{
  "data": {
    "server_time": "2026-09-10T01:05:00Z",
    "work_date": "2026-09-10",
    "has_checked_in": true,
    "has_checked_out": false,
    "check_in_time": "2026-09-10T01:00:15Z",
    "check_out_time": null,
    "can_check_in": false,
    "can_check_out": true,
    "recommended_action": "check_out",
    "active_offices": [
      {
        "id": "90e0e6c5-e51c-4b5f-a365-c7e6c4ea5768",
        "name": "Kantor Pusat",
        "latitude": -6.2088,
        "longitude": 106.8456,
        "radius_meter": 100
      }
    ],
    "geofence_policy": "reject",
    "fallback_enabled": true
  }
}
```

### 4.4 GET `/api/v1/attendance/my/today` (#56)
- **Permission**: `attendance.read_self`
- Mengembalikan daftar presensi karyawan hari ini (`check_in` dan/atau `check_out`) dalam format `EmployeeAttendanceDTO`.

### 4.5 GET `/api/v1/attendance/my/history` (#57)
- **Permission**: `attendance.read_self`
- **Query Params**: `start_date`, `end_date`, `page`, `per_page`.
- Mengembalikan riwayat presensi karyawan dengan paginasi metadata terstandarisasi.

### 4.6 GET `/api/v1/attendance/my/summary` (#58)
- **Permission**: `attendance.read_self`
- **Query Params**: `month` (`YYYY-MM`).
- Ringkasan statistik bulanan: total kehadiran, terlambat, pulang cepat, pending review, dan izin.

---

## 5. Subsystem Monitoring & Review Admin (#59–#65)

### 5.1 GET `/api/v1/attendance/records` (#59)
- **Permission**: `attendance.read_all` atau `attendance.read_team`
- **Query Params**: `work_date`, `start_date`, `end_date`, `status`, `type`, `department`, `page`, `per_page`.
- Mengembalikan daftar presensi seluruh karyawan dalam format `AdminAttendanceDTO` (termasuk skor kemiripan dan hash foto).

### 5.2 GET `/api/v1/attendance/records/{id}` (#60)
- **Permission**: `attendance.read_all` atau `attendance.read_team`
- Mengembalikan detail spesifik absensi beserta histori peninjauan.

### 5.3 GET `/api/v1/attendance/records/{id}/photo` (#61)
- **Permission**: `attendance.read_all` atau `attendance.read_team`
- Mengalirkan file foto absensi via HTTP stream (`image/jpeg`). Mengembalikan 404 jika foto telah dipurging oleh retention worker.

### 5.4 POST `/api/v1/attendance/records/{id}/approve` (#62)
- **Permission**: `attendance.review` atau `attendance.approve`
- **Anti-Self Review (Rule B11)**: Ditolak dengan 403 `FORBIDDEN` jika `reviewed_by == employee.user_id`.
- **Request Body**:
```json
{
  "notes": "Disetujui setelah konfirmasi kamera ponsel bermasalah"
}
```
- **Response 200 OK**: Objek absensi dengan `status = "approved"`.

### 5.5 POST `/api/v1/attendance/records/{id}/reject` (#63)
- **Permission**: `attendance.review` atau `attendance.approve`
- **Anti-Self Review (Rule B11)**: Ditolak dengan 403 `FORBIDDEN` jika `reviewed_by == employee.user_id`.
- Mengubah status absensi menjadi `"rejected"`.

### 5.6 POST `/api/v1/attendance/records/bulk-review` (#64)
- **Permission**: `attendance.review` atau `attendance.approve`
- Menyetujui atau menolak banyak absensi pending secara atomik dengan melewati rekaman milik sendiri.
- **Request Body**:
```json
{
  "action": "approve",
  "attendance_ids": [
    "7b7e6822-0df0-4b5c-b17b-d2c49df5d539",
    "8c8f7933-1ef1-5c6d-c28c-e3d50ef6e640"
  ],
  "notes": "Bulk approval persetujuan supervisor"
}
```

### 5.7 GET `/api/v1/attendance/stats` (#65)
- **Permission**: `attendance.read_all`
- **Query Params**: `work_date` (default hari ini).
- Menghasilkan statistik kehadiran perusahaan: total karyawan, hadir tepat waktu, terlambat, pending review, tidak hadir.

---

## 6. Subsystem Master Lokasi Kantor (#66–#70)

### 6.1 Endpoints Master Lokasi
- `GET /api/v1/locations` (#66): Daftar seluruh lokasi kantor aktif.
- `POST /api/v1/locations` (#67): Menambah lokasi kantor baru (`location.manage`).
- `GET /api/v1/locations/{id}` (#68): Detail lokasi kantor.
- `PUT /api/v1/locations/{id}` (#69): Memperbarui koordinat, radius, atau status aktif (`location.manage`).
- `DELETE /api/v1/locations/{id}` (#70): Soft-delete lokasi kantor (`location.manage`).

### 6.2 Skema Payload Kantor (#67, #69)
```json
{
  "name": "Kantor Cabang Bandung",
  "address": "Jl. Asia Afrika No. 10, Bandung",
  "latitude": -6.921472,
  "longitude": 107.607584,
  "radius_meter": 75,
  "is_active": true
}
```

---

## 7. Daemon Background Retention Worker

Untuk kepatuhan hukum retensi data dan efisiensi penyimpanan:
- **Scheduler**: Berjalan setiap 24 jam sekali (default pada tengah malam UTC).
- **PostgreSQL Advisory Lock**: Mengunci key integer `20260904` agar hanya 1 instance backend yang menjalankan proses pembersihan dalam arsitektur multi-replica.
- **Mekanisme Purge**:
  1. Mencari rekaman absensi dengan `server_timestamp < NOW() - make_interval(days => attendance_retention_days)`.
  2. Menghapus objek foto fisik dari Object Storage (`Store.Delete`).
  3. Mengatur kolom `photo_path = NULL` pada tabel `attendances`.
  4. Menjaga nilai `photo_sha256` tetap ada untuk mencegah pelanggaran anti-replay historis.
  5. Mencatat log audit sistem atas jumlah objek yang dibersihkan.

---

## 8. Katalog Kode Error Fase 4

| HTTP Status | Error Code | Keterangan |
|---|---|---|
| 400 | `BAD_REQUEST` | Payload JSON/Form-data tidak valid atau koordinat cacat. |
| 401 | `UNAUTHENTICATED` | Token sesi JWT / HttpOnly cookie tidak valid atau kedaluwarsa. |
| 403 | `FORBIDDEN` | Role tidak memiliki permission yang dipersyaratkan. |
| 403 | `SELF_REVIEW_NOT_ALLOWED` | Reviewer mencoba menyetujui/menolak absensinya sendiri (Rule B11). |
| 404 | `NOT_FOUND` | Data absensi, foto, atau lokasi kantor tidak ditemukan. |
| 409 | `ALREADY_CHECKED_IN` | Karyawan sudah melakukan check-in pada tanggal kerja hari ini. |
| 409 | `ALREADY_CHECKED_OUT` | Karyawan sudah melakukan check-out pada tanggal kerja hari ini. |
| 409 | `CHECKOUT_WITHOUT_CHECKIN` | Check-out dicoba sebelum check-in berhasil dicatat. |
| 409 | `CHECKOUT_TOO_SOON` | Check-out dicoba dalam rentang waktu terlalu singkat pasca check-in (< 5 menit). |
| 409 | `DUPLICATE_PHOTO` | Foto identik telah digunakan sebelumnya oleh karyawan ini (Rule B4). |
| 422 | `FACE_NOT_ENROLLED` | Karyawan belum mendaftarkan referensi wajah biometrik. |
| 422 | `FACE_MISMATCH` | Wajah tidak cocok dengan referensi biometrik karyawan. |
| 422 | `FACE_NOT_USABLE` | Kualitas foto wajah buruk (blur, gelap, tidak ada wajah, multi-wajah). |
| 422 | `OUTSIDE_GEOFENCE` | Pengguna berada di luar radius kantor dan policy diset `reject`. |
| 422 | `LOCATION_REQUIRED` | Koordinat GPS wajib dilampirkan namun tidak dikirimkan. |
| 422 | `CLIENT_CLOCK_SKEW` | Waktu pada perangkat pengguna berselisih signifikan dengan server. |
| 429 | `RATE_LIMITED` | Terlalu banyak percobaan presensi yang gagal dalam 1 jam terakhir (Rule B12). |
| 500 | `INTERNAL_ERROR` | Terjadi kesalahan internal basis data atau object storage. |
