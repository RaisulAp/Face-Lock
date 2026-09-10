# FaceClock API Specification — Fase 3: Enrollment Wajah, Biometric Consent (UU PDP No. 27/2022) & Reindexing Model

Dokumen ini mendefinisikan kontrak 21 endpoint HTTP API Fase 3 (Endpoints #32–#52) untuk backend FaceClock (`apps/faceclock-api`).
Sesuai konvensi standar arsitektur sistem (`docs/adr/0003-api-conventions.md`):
- Berakar di prefix `/api/v1`
- Menggunakan JSON envelope standar: `{ "data": ... }` untuk sukses (atau custom metadata envelope terotorisasi), dan `{ "error": { "code": "...", "message": "...", "details": [...] } }` untuk gagal
- Mengirimkan header korelasi `X-Request-ID` di setiap response
- Menegakkan otorisasi berbasis Role-Based Access Control (RBAC) dengan prinsip **default-deny** dan middleware `RequireConsent`
- Mengikuti hukum privasi data biometrik **UU PDP No. 27/2022** (Persetujuan eksplisit, hak pencabutan persetujuan, fallback mode manual, dan *Right to be Forgotten*).
- Mencegah *Public Photo Leakage*: file foto disimpan di object storage (Local / S3 MinIO) dan hanya dialirkan via HTTP streaming terotentikasi.

---

## Daftar Isi
1. [Arsitektur Kepatuhan Regulasi UU PDP No. 27/2022](#1-arsitektur-kepatuhan-regulasi-uu-pdp-no-272022)
2. [Arsitektur Multi-Photo Enrollment & Reindexing](#2-arsitektur-multi-photo-enrollment--reindexing)
3. [Subsystem Biometric Consent (#32–#36)](#3-subsystem-biometric-consent)
4. [Subsystem Multi-Photo Face Enrollment (#37–#41)](#4-subsystem-multi-photo-face-enrollment)
5. [Subsystem Face References Management (#42–#47)](#5-subsystem-face-references-management)
6. [Subsystem Face Model Reindexing (#48–#52)](#6-subsystem-face-model-reindexing)
7. [Katalog Kode Error Fase 3](#7-katalog-kode-error-fase-3)

---

## 1. Arsitektur Kepatuhan Regulasi UU PDP No. 27/2022

Sistem FaceClock memperlakukan data biometrik wajah (embedding 512D dan foto wajah) sebagai **Data Pribadi yang Bersifat Spesifik** sesuai Pasal 4 UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi (UU PDP):

1. **Persetujuan Eksplisit (Explicit Consent - Pasal 20 & 22)**:
   - Sesi enrollment wajah tidak dapat dimulai tanpa persetujuan eksplisit karyawan atas dokumen consent aktif (`2026-09-v1`).
   - Setiap persetujuan mengunci `document_version`, `document_hash` (SHA-256), `ip_address`, dan `user_agent`.
2. **Hak Penolakan Persetujuan & Fallback Operasional (Pasal 24 & 40)**:
   - Karyawan berhak menolak atau mencabut persetujuan biometrik.
   - Karyawan yang menolak/mencabut consent dialihkan secara otomatis ke `employees.attendance_mode = 'manual'`.
   - Sistem **TIDAK MENGHUKUM** penolakan: karyawan dapat melakukan absensi manual tanpa pemindaian wajah.
3. **Pencabutan Persetujuan (Withdrawal of Consent - Pasal 40)**:
   - Karyawan atau Admin dapat mencabut persetujuan biometrik (`POST /api/v1/employees/{id}/consent/revoke`).
   - Pencabutan secara otomatis menonaktifkan seluruh referensi wajah aktif dengan alasan `deactivated_reason = 'consent_withdrawn'`, mengatur `face_enrolled_at = NULL`, dan mencatat riwayat penarikan consent di `biometric_consents`.
4. **Hak atas Penghapusan / Right to be Forgotten (Pasal 43)**:
   - Penghapusan data biometrik (`DELETE /api/v1/employees/{id}/face-data`) melakukan **HARD DELETE** pada tabel `face_references`, menghapus seluruh objek foto dari storage (`Store.Delete`), dan mengatur `employees.face_enrolled_at = NULL`.
   - Mengikuti **REV-CONV-03**: tabel `face_references` tidak memiliki soft-delete (`deleted_at`) karena menyimpan embedding pada baris soft-deleted melanggar prinsip penghapusan PDP.

---

## 2. Arsitektur Multi-Photo Enrollment & Reindexing

### 2.1 Multi-Photo Enrollment (3 Foto Terpisah)
- **Anti-Average Vector Fallback**: Sistem menolak perataan vektor (*average vector*). Tiga sudut foto (Depan, Serong Kiri, Serong Kanan) disimpan sebagai 3 baris embedding terpisah di `face_references` untuk menangkap variasi pose alami karyawan.
- **Sesi Enrollment Transaksional**:
  - `POST /api/v1/face/enrollments/start` menginisialisasi sesi di `face_enrollment_sessions` dengan TTL (default 15 menit).
  - Foto diunggah satu per satu ke staging storage (`face-staging/{session_id}/{photo_id}.jpg`).
  - Setiap foto divalidasi kualitasnya oleh inference service (`/embed`): deteksi wajah tunggal, blur variance, brightness, yaw/pitch bounding.
  - Sesi difinalisasi dengan memverifikasi *cross-photo similarity* antar 3 foto ($\ge 0.65$), pemindaian duplikasi 1:N terhadap seluruh database referensi karyawan lain, transfer file dari staging ke permanen (`face/{employee_id}/{ref_id}.jpg`), dan penyimpanan 3 embedding ke `face_references`.

### 2.2 Reindexing Model Otomatis & Non-Blocking
- Ketika arsitektur/model embedding ditingkatkan (mis. dari `buffalo_l` ke versi baru), seluruh referensi lama harus dihitung ulang.
- Sistem menyediakan job runner asinkron:
  - Job dimulai via `POST /api/v1/face/reindex/start` dengan permission `face.reindex` (`super_admin` only).
  - Daemon latar belakang (`ReindexWorker`) berjalan dengan proteksi **PostgreSQL Advisory Lock** (`20260903`) untuk mencegah race condition antar replika instance backend.
  - Setiap foto referensi aktif di-embed ulang menggunakan model baru dan disimpan sebagai referensi staging (`is_active = false`, `deactivated_reason = 'model_reindex'`).
  - Setelah seluruh referensi karyawan berhasil di-embed ulang, aktivasi swap dilakukan secara transaksional: referensi model lama dinonaktifkan, referensi model baru diaktifkan, dan versi model sistem diperbarui.

---

## 3. Subsystem Biometric Consent

### 3.1 GET `/api/v1/consent/document`
- **Guard**: Terbuka / Terautentikasi (semua user)
- **Deskripsi**: Mengambil dokumen persetujuan biometrik aktif yang berlaku menurut regulasi perusahaan & UU PDP.
- **Response 200 OK**:
```json
{
  "data": {
    "version": "2026-09-v1",
    "effective_date": "2026-09-01T00:00:00Z",
    "title": "Persetujuan Pemrosesan Data Biometrik Pengenalan Wajah",
    "content_markdown": "# Persetujuan Pemrosesan Data Biometrik...",
    "document_hash": "6411d334ff9d0ec0e76839359eefaa5744cb446cb3eeaa634c03b8e4e9f7833a",
    "required": true
  }
}
```

### 3.2 GET `/api/v1/employees/me/consent`
- **Guard**: `employee.read_self`
- **Deskripsi**: Mengambil riwayat persetujuan biometrik karyawan yang sedang login.
- **Response 200 OK**:
```json
{
  "data": {
    "status": "granted",
    "attendance_mode": "face",
    "consent": {
      "id": "188b48ef-c5ef-4d37-83cb-4f35cf5723b7",
      "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
      "document_version": "2026-09-v1",
      "status": "granted",
      "granted_at": "2026-09-10T08:00:00Z",
      "revoked_at": null,
      "attendance_mode": "face"
    },
    "current_document_version": "2026-09-v1",
    "is_current_version": true
  }
}
```

### 3.3 POST `/api/v1/employees/me/consent`
- **Guard**: `employee.read_self`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Memberikan persetujuan (`agreed: true`) atau penolakan (`agreed: false`) atas pemrosesan data biometrik.
- **Request Body**:
```json
{
  "document_version": "2026-09-v1",
  "document_hash": "6411d334ff9d0ec0e76839359eefaa5744cb446cb3eeaa634c03b8e4e9f7833a",
  "agreed": true
}
```
- **Response 200 OK (Persetujuan Diberikan)**:
```json
{
  "data": {
    "consent_id": "188b48ef-c5ef-4d37-83cb-4f35cf5723b7",
    "status": "granted",
    "attendance_mode": "face",
    "granted_at": "2026-09-10T08:00:00Z",
    "document_version": "2026-09-v1"
  }
}
```
- **Response 200 OK (Persetujuan Ditolak)**:
```json
{
  "data": {
    "consent_id": "188b48ef-c5ef-4d37-83cb-4f35cf5723b7",
    "status": "denied",
    "attendance_mode": "manual",
    "denied_at": "2026-09-10T08:00:00Z",
    "attendance_mode_hint": "manual"
  }
}
```

### 3.4 GET `/api/v1/employees/{id}/consent`
- **Guard**: `employee.read_all`
- **Deskripsi**: Melihat status dan riwayat persetujuan biometrik karyawan tertentu oleh Admin/HR.

### 3.5 POST `/api/v1/employees/{id}/consent/revoke`
- **Guard**: `employee.update` (atau `employee.read_self` jika ID milik sendiri)
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Mencabut persetujuan biometrik karyawan. Otomatis menonaktifkan seluruh referensi wajah aktif dengan alasan `consent_withdrawn` dan mengalihkan mode absensi ke `manual`.
- **Request Body**:
```json
{
  "revocation_reason": "Permintaan pencabutan persetujuan mandiri oleh karyawan."
}
```
- **Response 200 OK**:
```json
{
  "data": {
    "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
    "status": "revoked",
    "attendance_mode": "manual",
    "revoked_at": "2026-09-10T09:30:00Z",
    "deactivated_references_count": 3
  }
}
```

---

## 4. Subsystem Multi-Photo Face Enrollment

### 4.1 POST `/api/v1/face/enrollments/start`
- **Guard**: `face.enroll` + `RequireConsent`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Memulai sesi baru pendaftaran foto wajah karyawan (wajib 3 foto).
- **Request Body**:
```json
{
  "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58"
}
```
- **Response 201 Created**:
```json
{
  "data": {
    "session_id": "993a4b08-333e-4b72-b7e3-1ca73eb7f632",
    "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
    "target_photos_count": 3,
    "current_photos_count": 0,
    "expires_at": "2026-09-10T08:15:00Z",
    "status": "in_progress"
  }
}
```

### 4.2 POST `/api/v1/face/enrollments/{session_id}/photos`
- **Guard**: `face.enroll`
- **Content-Type**: `multipart/form-data`
- **Deskripsi**: Mengunggah 1 foto wajah (`photo`, image/jpeg) dengan pose label (`front`, `left_tilt`, `right_tilt`). Wajah divalidasi kualitasnya oleh inference service.
- **Response 201 Created**:
```json
{
  "data": {
    "photo_id": "778bb1d8-04b1-4f01-831d-b8d29b1eebfa",
    "pose_label": "front",
    "quality_score": 0.942,
    "quality_metrics": {
      "blur_var": 320.5,
      "brightness": 140.2,
      "face_ratio": 0.45,
      "yaw": -1.2,
      "pitch": 0.8
    },
    "current_photos_count": 1,
    "remaining_photos_count": 2
  }
}
```

### 4.3 GET `/api/v1/face/enrollments/{session_id}/status`
- **Guard**: `face.enroll`
- **Deskripsi**: Mengecek progres sesi enrollment yang sedang berlangsung dan rincian foto yang telah terunggah.

### 4.4 POST `/api/v1/face/enrollments/{session_id}/finalize`
- **Guard**: `face.enroll`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Memvalidasi kesamaan ketiga foto wajah ($\ge 0.65$), memeriksa pencegahan duplikasi terhadap seluruh karyawan lain di database, memindahkan file dari staging ke permanen, menyimpan 3 embedding terpisah ke `face_references`, dan mengaktifkan karyawan (`face_enrolled_at = NOW()`).
- **Response 200 OK**:
```json
{
  "data": {
    "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
    "reference_ids": [
      "d0c2e9a7-8622-4a0b-a131-a8bb7d00f341",
      "e1d3f0b8-9733-5b1c-b242-b9cc8e11f452",
      "f2e4a1c9-0844-6c2d-c353-ca0d9f22f563"
    ],
    "photos_enrolled_count": 3,
    "model_version": "buffalo_l",
    "enrolled_at": "2026-09-10T08:05:00Z"
  }
}
```

### 4.5 DELETE `/api/v1/face/enrollments/{session_id}`
- **Guard**: `face.enroll`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Membatalkan sesi pendaftaran wajah dan membersihkan seluruh file sementara di staging storage.
- **Response 200 OK**:
```json
{
  "data": {
    "session_id": "993a4b08-333e-4b72-b7e3-1ca73eb7f632",
    "status": "cancelled",
    "cleaned_staging_photos_count": 2
  }
}
```

---

## 5. Subsystem Face References Management

### 5.1 GET `/api/v1/employees/me/face-references`
- **Guard**: `employee.read_self`
- **Deskripsi**: Menampilkan daftar metadata referensi wajah aktif milik karyawan yang sedang login.

### 5.2 GET `/api/v1/employees/{id}/face-references`
- **Guard**: `face.read_all` (atau `employee.read_self` jika ID milik sendiri)
- **Deskripsi**: Menampilkan referensi wajah karyawan dengan custom metadata envelope (statistik referensi aktif/inaktif dan versi model).
- **Response 200 OK**:
```json
{
  "data": [
    {
      "id": "d0c2e9a7-8622-4a0b-a131-a8bb7d00f341",
      "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
      "pose_label": "front",
      "model_version": "buffalo_l",
      "is_active": true,
      "created_at": "2026-09-10T08:05:00Z",
      "deactivated_at": null,
      "deactivated_reason": null,
      "photo_purged_at": null
    }
  ],
  "meta": {
    "active_count": 3,
    "inactive_count": 0,
    "model_version": "buffalo_l",
    "model_versions_in_use": ["buffalo_l"]
  }
}
```

### 5.3 GET `/api/v1/face-references/{id}/photo`
- **Guard**: `face.read_all` (atau `employee.read_self` untuk pemilik)
- **Deskripsi**: Streaming file citra wajah referensi asli secara biner (`image/jpeg`). Dilengkapi header proteksi `Cache-Control: private, no-store`. Mengembalikan `410 Gone` (dengan error code `NOT_FOUND` sesuai REV-ERR-04) bila foto telah dihapus oleh kebijakan retensi.

### 5.4 POST `/api/v1/face-references/{id}/deactivate`
- **Guard**: `face.manage`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Menonaktifkan satu foto referensi wajah tertentu. Jika seluruh referensi aktif habis, `employees.face_enrolled_at` otomatis direset ke `NULL`.

### 5.5 POST `/api/v1/face-references/{id}/activate`
- **Guard**: `face.manage`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Mengaktifkan kembali foto referensi wajah yang sebelumnya dinonaktifkan.

### 5.6 DELETE `/api/v1/employees/{id}/face-data` (Right to be Forgotten - UU PDP)
- **Guard**: `face.manage` (atau `super_admin`)
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Menghapus seluruh data biometrik karyawan secara permanen (*Hard Delete*): menghapus semua baris `face_references` dari database, menghapus semua file foto dari object storage, mereset `employees.face_enrolled_at = NULL`, dan mencatat kejadian ini di audit log.
- **Response 200 OK**:
```json
{
  "data": {
    "employee_id": "7649537f-ec73-451e-876e-ecff02a9eb58",
    "deleted_references_count": 3,
    "purged_storage_photos_count": 3,
    "face_enrolled_at": null,
    "status": "face_data_purged"
  }
}
```

---

## 6. Subsystem Face Model Reindexing

### 6.1 POST `/api/v1/face/reindex/start`
- **Guard**: `face.reindex` (`super_admin` only)
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Memulai proses reindexing model biometrik ke versi arsitektur embedding baru.
- **Request Body**:
```json
{
  "to_model_version": "buffalo_m",
  "batch_size": 25
}
```
- **Response 202 Accepted**:
```json
{
  "data": {
    "job_id": "52c80332-9df7-4d6d-8bc4-cf36db1a0301",
    "from_model_version": "buffalo_l",
    "to_model_version": "buffalo_m",
    "total_references": 30,
    "status": "pending",
    "started_at": "2026-09-10T10:00:00Z"
  }
}
```

### 6.2 GET `/api/v1/face/reindex/jobs`
- **Guard**: `face.reindex`
- **Deskripsi**: Menampilkan riwayat daftar seluruh pekerjaan reindex biometrik.

### 6.3 GET `/api/v1/face/reindex/jobs/{id}`
- **Guard**: `face.reindex`
- **Deskripsi**: Melihat detail progres pekerjaan reindex, jumlah referensi yang berhasil/gagal diproses, dan pesan error jika ada.

### 6.4 POST `/api/v1/face/reindex/jobs/{id}/cancel`
- **Guard**: `face.reindex`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Membatalkan pekerjaan reindexing yang sedang berjalan.

### 6.5 POST `/api/v1/face/reindex/jobs/{id}/retry-failed`
- **Guard**: `face.reindex`
- **CSRF**: Wajib header CSRF
- **Deskripsi**: Menjadwalkan ulang pemrosesan foto-foto referensi yang gagal di-embed pada pekerjaan sebelumnya.

---

## 7. Katalog Kode Error Fase 3

| HTTP Status | Error Code | Deskripsi Bisnis |
|---|---|---|
| `403 Forbidden` | `CONSENT_REQUIRED` | Karyawan belum menyetujui dokumen persetujuan biometrik aktif. |
| `409 Conflict` | `CONSENT_ALREADY_GRANTED` | Karyawan telah menyetujui versi dokumen consent yang sama sebelumnya. |
| `409 Conflict` | `CONSENT_VERSION_OUTDATED` | Persetujuan dikirim untuk versi dokumen yang sudah usang (*outdated*). |
| `422 Unprocessable` | `FACE_NOT_USABLE` | Citra wajah gagal melewati ambang kualitas (blur, terlalu gelap/terang, wajah miring). |
| `409 Conflict` | `DUPLICATE_PHOTO` | Foto yang diunggah terdeteksi identik/duplikat dengan foto yang sudah ada di sesi ini. |
| `422 Unprocessable` | `ENROLLMENT_INCOMPLETE` | Sesi difinalisasi sebelum memenuhi syarat 3 foto berkualitas baik. |
| `409 Conflict` | `ENROLLMENT_LIMIT_REACHED` | Jumlah foto yang diunggah dalam sesi telah mencapai batas maksimal (3 foto). |
| `409 Conflict` | `ENROLLMENT_SESSION_EXPIRED` | Sesi enrollment telah melewati batas waktu berlaku (*expired*). |
| `409 Conflict` | `ENROLLMENT_MODEL_CHANGED` | Versi model biometrik server berubah di tengah proses sesi enrollment. |
| `409 Conflict` | `FACE_BELONGS_TO_ANOTHER_EMPLOYEE` | Pemindaian 1:N mendeteksi kemiripan wajah tinggi dengan karyawan lain yang sudah terdaftar. |
| `409 Conflict` | `REINDEX_IN_PROGRESS` | Terdapat pekerjaan reindex model biometrik lain yang sedang berjalan aktif. |
| `422 Unprocessable` | `ATTENDANCE_MODE_MANUAL` | Operasi biometrik ditolak karena karyawan memilih mode absensi manual. |
| `410 Gone` | `NOT_FOUND` | Foto referensi telah dihapus secara permanen oleh kebijakan retensi sistem. |
| `503 Service Unavailable`| `FACE_SERVICE_NOT_CONFIGURED` | Versi model biometrik belum dikonfigurasi (`face.model_version = 'unset'`). |
