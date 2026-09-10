# DONE — Fase 3: Biometric Consent (UU PDP No. 27/2022), Multi-Photo Face Enrollment, Reference Management & Background Model Reindexing

> Ditulis sesuai Protokol Handoff [00MasterPlan.md § 10.4](00MasterPlan.md#10-protokol-handoff-untuk-agent).  
> Melanjutkan fondasi dari [DONE-Fase-0.md](DONE-Fase-0.md), [DONE-Fase-1.md](DONE-Fase-1.md), dan kontrak inference Fase 2.  
> Dieksekusi secara tuntas dan mandiri pada 2026-09-10.

---

## 1. Apa yang Dibangun

Seluruh 8 tugas arsitektural Fase 3 untuk `apps/faceclock-api` telah diimplementasikan, dimigrasikan, dihubungkan ke routing, dan diverifikasi penuh melalui automated test harness.

### 1.1 Database Migrations (`migrations/`)
- **`000012_create_consent_documents.{up,down}.sql`** — Tabel master dokumen persetujuan biometrik berversi (`version`, `title`, `content_markdown`, `content_hash` sha256, `effective_date`, `is_active`). Seeder idempoten memuat versi awal `2026-09-v1`.
- **`000013_create_biometric_consents.{up,down}.sql`** — Tabel pencatatan riwayat persetujuan karyawan (`employee_id`, `document_version`, `status` IN ('granted','denied','revoked'), `granted_at`, `revoked_at`, `revocation_reason`, `ip_address`, `user_agent`), ditambah kolom baru pada `employees`: `attendance_mode text NOT NULL DEFAULT 'face' CHECK (attendance_mode IN ('face', 'manual'))` (**REV-DB-01**).
- **`000014_create_face_references.{up,down}.sql`** — Tabel penyimpanan referensi biometrik karyawan dengan embedding pgvector `vector(512)`, `model_version`, `pose_label` ('front','left_tilt','right_tilt'), `photo_key` (**REV-DB-02**), `photo_sha256`, `photo_bytes`, `photo_mime`, `photo_purged_at`, `is_active`, `deactivated_at`, `deactivated_reason`. Menerapkan **REV-CONV-03** (tanpa kolom `deleted_at`; siklus hidup memakai `is_active`, sedangkan penghapusan permanen adalah hard delete untuk kepatuhan PDP). Dilengkapi indeks lookup cepat B-Tree `(employee_id, is_active, model_version)` dan partial index `(is_active) WHERE is_active = true`.
- **`000015_create_face_enrollment_sessions.{up,down}.sql`** — Tabel manajemen sesi enrollment multi-foto (`employee_id`, `session_token_hash`, `target_photos_count`, `current_photos_count`, `model_version`, `status`, `expires_at`, timestamps).
- **`000016_create_face_enrollment_photos.{up,down}.sql`** — Tabel staging foto selama sesi berlangsung (`session_id`, `pose_label`, `photo_key` staging, `photo_sha256`, `photo_bytes`, `embedding` vector(512), `quality_score`, `quality_metrics` jsonb, timestamps).
- **`000017_create_face_reindex_tables.{up,down}.sql`** — Tabel orkestrasi pembaruan model embedding:
  - `face_reindex_jobs` (`from_model_version`, `to_model_version`, `status`, `total_references`, `processed_references`, `failed_references`, `started_at`, `finished_at`, `error_message`).
  - `face_reindex_items` (`job_id`, `source_reference_id`, `employee_id`, `status`, `new_embedding`, `new_photo_key`, `error_message`, `retry_count`, `processed_at`).
  - Penambahan permission baru **`face.reindex`** (**REV-PERM-01**) yang di-grant khusus ke `super_admin`.
  - Penambahan 7 key pengaturan `face.*` (**REV-SET-02**): `face.max_reference_photos`, `face.enrollment_session_ttl_minutes`, `face.min_quality_score`, `face.duplicate_check_enabled`, `face.duplicate_threshold`, `face.retention_days_after_resign`, `face.consent_required`.

### 1.2 Core Subsystems (`apps/faceclock-api/internal/`)

#### A. Subsystem Biometric Consent (`internal/consent/`)
- Kepatuhan penuh regulasi **UU PDP No. 27/2022**:
  - `GetActiveDocument`: Mengambil dokumen aktif beserta hash SHA-256 kontennya.
  - `GrantConsent`: Memverifikasi kecocokan versi dan hash dokumen, mencatat persetujuan eksplisit, dan mengaktifkan mode `attendance_mode = 'face'`.
  - `DenyConsent`: Mencatat penolakan persetujuan karyawan tanpa sanksi, secara otomatis mengatur `employees.attendance_mode = 'manual'` dan mengembalikan hint `attendance_mode_hint: "manual"`.
  - `RevokeConsent`: Mencabut persetujuan, menonaktifkan seluruh `face_references` aktif dengan alasan `deactivated_reason = 'consent_withdrawn'`, mengatur `employees.attendance_mode = 'manual'`, serta mengosongkan `face_enrolled_at`.
  - `RequireConsent` Middleware (`internal/httpx/middleware/consent.go` — **REV-MW-01**): Memastikan pengguna/karyawan memiliki persetujuan biometrik aktif sebelum dapat memproses atau mendaftarkan data biometrik, menolak dengan `403 CONSENT_REQUIRED`.

#### B. Storage Driver Abstraction (`internal/storage/`)
- Menerapkan interface `storage.Store` dengan dukungan dual-driver:
  - `LocalStore`: Menyimpan foto pada disk/volume lokal dengan proteksi direktori (digunakan untuk development dan testing).
  - `S3Store`: Terintegrasi dengan MinIO / Amazon S3 (**REV-INF-03**) menggunakan `aws-sdk-go-v2/service/s3`, mendukung Server-Side Encryption (SSE-S3), Bucket isolasi terpisah (`face` dan `attendance`), serta path style configuration.
  - Skema key deterministik dan bebas kebocoran identitas:
    - Staging: `face-staging/{session_id}/{photo_id}.jpg`
    - Permanen: `face/{employee_id}/{reference_id}.jpg`
  - **REV-INF-04**: `Store.SignedURL()` tidak diekspos untuk akses publik. Akses foto dialirkan via HTTP streaming privat berotentikasi.

#### C. Multi-Photo Enrollment Subsystem (`internal/face/enrollment/`)
- Menerapkan pola sesi enrollment multi-foto (3 foto terpisah: `front`, `left_tilt`, `right_tilt`):
  - Penolakan *Average Vector Fallback*: Menyimpan 3 vektor terpisah untuk mengakomodasi variasi sudut dan pencahayaan alami karyawan.
  - Evaluasi kualitas real-time per foto melalui integrasi inference service (`/embed`): ambang deteksi wajah, blur variance, kecerahan, dan batas sudut yaw/pitch.
  - Pencegahan foto identik dalam satu sesi (`409 DUPLICATE_PHOTO`).
  - Verifikasi kesamaan ketiga foto (*cross-photo similarity*) saat finalisasi ($\ge 0.65$).
  - Deteksi wajah duplikat 1:N antar-karyawan (**D16**) sebelum penyimpanan permanen (`409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE`).
  - Transaksi commit atomik: memindahkan file dari staging ke permanen storage, menyisipkan 3 baris ke `face_references`, memperbarui `employees.face_enrolled_at = NOW()`, dan membersihkan staging storage.
  - Pembatalan sesi (`DELETE /api/v1/face/enrollments/{session_id}`) membersihkan file di staging storage.

#### D. Face Reference Management Subsystem (`internal/face/reference/`)
- `ListReferences`: Menampilkan referensi wajah dengan custom metadata envelope (`active_count`, `inactive_count`, `model_version`, `model_versions_in_use`).
- `StreamPhoto`: Mengalirkan biner citra referensi (`image/jpeg`) dengan header keamanan `Cache-Control: private, no-store`. Mengembalikan `410 Gone` dengan error code `NOT_FOUND` (**REV-ERR-04**) jika foto telah dihapus oleh retensi.
- `DeactivateReference` / `ActivateReference`: Pengelolaan status aktif/inaktif foto referensi secara granular.
- `PurgeEmployeeFaceData`: Implementasi pemenuhan hak **Right to be Forgotten (UU PDP)**:
  - Melakukan **Hard Delete** pada seluruh baris `face_references` di database.
  - Menghapus seluruh file citra fisik dari storage (`Store.Delete`).
  - Mengosongkan status enrollment `employees.face_enrolled_at = NULL`.
  - Mencatat aksi pembersihan permanen ke audit log.

#### E. Face Model Reindex Subsystem & Background Worker (`internal/face/reindex/`)
- `StartJob`: Memvalidasi versi model target terhadap inference service, membuat entitas `face_reindex_jobs`, dan melakukan staging seluruh referensi aktif ke `face_reindex_items`.
- `ReindexWorker`: Daemon latar belakang asinkron:
  - Menggunakan **PostgreSQL Advisory Lock** (`20260903`) untuk koordinasi non-blocking antar-node/replika API.
  - Membaca item pending secara batch, mengunduh foto dari storage, mengirim batch embed ke inference service (`/embed-batch`), dan menyimpan embedding baru sebagai referensi inaktif bertanda `deactivated_reason = 'model_reindex'`.
  - Melakukan atomic swap per karyawan setelah seluruh referensinya selesai diproses: menonaktifkan referensi versi lama dan mengaktifkan referensi versi baru.
  - Secara otomatis memperbarui pengaturan `face.model_version` sistem ke model baru saat job selesai secara utuh.
- Mendukung pembatalan job (`CancelJob`) dan penjadwalan ulang item gagal (`RetryFailed`).

### 1.3 Routing & HTTP Endpoints (#32–#52)
Seluruh 21 endpoint Fase 3 telah didaftarkan di `internal/httpx/routes.go` lengkap dengan middleware pengamanan RBAC, `RequireConsent`, Anti-CSRF, dan ownership check:
- **Biometric Consent**:
  - `#32` `GET /api/v1/consent/document`
  - `#33` `GET /api/v1/employees/me/consent`
  - `#34` `POST /api/v1/employees/me/consent`
  - `#35` `GET /api/v1/employees/{id}/consent`
  - `#36` `POST /api/v1/employees/{id}/consent/revoke`
- **Multi-Photo Enrollment**:
  - `#37` `POST /api/v1/face/enrollments/start`
  - `#38` `POST /api/v1/face/enrollments/{session_id}/photos`
  - `#39` `GET /api/v1/face/enrollments/{session_id}/status`
  - `#40` `POST /api/v1/face/enrollments/{session_id}/finalize`
  - `#41` `DELETE /api/v1/face/enrollments/{session_id}`
- **Face References**:
  - `#42` `GET /api/v1/employees/me/face-references`
  - `#43` `GET /api/v1/employees/{id}/face-references`
  - `#44` `GET /api/v1/face-references/{id}/photo`
  - `#45` `POST /api/v1/face-references/{id}/deactivate`
  - `#46` `POST /api/v1/face-references/{id}/activate`
  - `#47` `DELETE /api/v1/employees/{id}/face-data`
- **Model Reindex**:
  - `#48` `POST /api/v1/face/reindex/start`
  - `#49` `GET /api/v1/face/reindex/jobs`
  - `#50` `GET /api/v1/face/reindex/jobs/{id}`
  - `#51` `POST /api/v1/face/reindex/jobs/{id}/cancel`
  - `#52` `POST /api/v1/face/reindex/jobs/{id}/retry-failed`

---

## 2. Keputusan Teknis Final (D13–D16)

| Keputusan | Pilihan Final | Alasan & Landasan Hukum/Arsitektural |
|---|---|---|
| **D13 — Penyimpanan Foto Referensi** | **MinIO (S3 API) di Production / LocalStore di Dev** | Data biometrik menuntut pembuktian kepatuhan kebijakan retensi dan enkripsi at-rest. MinIO menyediakan *bucket lifecycle rules* otomatis dan enkripsi SSE-S3. Dua bucket terpisah (`face` dan `attendance`) memisahkan data dengan profil retensi berbeda. `bytea` di Postgres ditolak karena ukuran DB membengkak drastis. |
| **D14 — Akses Foto Citra Biometrik** | **Streaming Biner Privat via API (`GET /face-references/{id}/photo`)** | Menggugurkan opsi *Signed URL* publik. Begitu signed URL diterbitkan, token tersebut rentan bocor ke log proxy atau riwayat browser tanpa validasi hak akses per-detik. Streaming melalui API menjamin pengecekan RBAC dan ownership pada setiap request. Header `Cache-Control: private, no-store` mencegah caching proxy publik. |
| **D15 — Alur Enrollment Wajah** | **Sesi Multi-Foto Bertahap (3 Foto Terpisah: Front, Left Tilt, Right Tilt)** | Satu request 3 foto sekaligus ditolak karena jika foto ke-3 gagal validasi kualitas, seluruh request gagal dan user harus mengulang dari awal. Sesi bertahap memberikan feedback kualitas seketika (*real-time hints*). Perataan vektor (*average vector*) ditolak keras; 3 embedding terpisah disimpan untuk menangkap variabilitas pose natural karyawan. |
| **D16 — Deteksi Wajah Duplikat Antar-Karyawan** | **Aktif Secara Default saat Finalisasi Enrollment (`face.duplicate_check_enabled = true`)** | Mencegah kecurangan paling fatal: satu oknum mendaftarkan wajahnya pada dua akun karyawan yang berbeda untuk melakukan absensi titipan. Pemindaian 1:N sequential scan eksak terhadap seluruh embedding aktif karyawan lain dilakukan saat commit finalisasi. Jika terdeteksi kemiripan di atas ambang batas (`face.duplicate_threshold`), server mengembalikan `409 FACE_BELONGS_TO_ANOTHER_EMPLOYEE` tanpa membocorkan identitas korban. |

---

## 3. Status Verifikasi & Ringkasan Pengujian

Pengujian integrasi penuh dieksekusi secara otomatis dan independen di lingkungan database nyata:

1. **`TestFaceEnrollmentAndPDPCompliance`**:
   - `consent_flow_lifecycle`: Verifikasi dokumen aktif, persetujuan eksplisit, pencatatan IP/User-Agent, dan penegakan `RequireConsent`.
   - `multi_photo_enrollment_lifecycle`: Alur lengkap sesi: start -> upload 3 foto (front, left_tilt, right_tilt) dengan validasi kualitas inference -> finalisasi -> 3 embedding tersimpan di `face_references` dan `employees.face_enrolled_at` terisi.
   - `enrollment_session_cancellation`: Pembatalan sesi di tengah jalan membersihkan seluruh file sementara di staging storage.
   - `consent_revocation_deactivates_face_references`: Pencabutan consent menonaktifkan seluruh referensi wajah (`deactivated_reason = 'consent_withdrawn'`) dan mengalihkan mode absensi ke `manual`.
   - `right_to_be_forgotten_hard_deletes_face_data`: Penghapusan data biometrik melakukan hard delete pada database, membersihkan storage fisik, dan mengosongkan status enrollment karyawan.
   - `reindex_job_lifecycle`: Pembuatan job reindex model, pemetaan item, pembatalan job, dan penjadwalan ulang item gagal.
   - `reindex_worker_execution`: Eksekusi daemon reindex di latar belakang dengan proteksi advisory lock, re-embedding batch foto, atomic swap per karyawan, dan pembaruan versi model sistem.
2. **`TestFaceEnrollmentAndAttendance_Integration`**:
   - Menjamin tidak ada regresi pada endpoint legacy attendance dan pendaftaran wajah Fase 1.
3. **Hasil Eksekusi Test**:
   - Seluruh paket di `apps/faceclock-api` (`auth`, `consent`, `employee`, `face/enrollment`, `face/reference`, `face/reindex`, `httpx`, `rbac`, `role`, `settings`, `storage`, `user`, `integration`) **LULUS 100% (PASS)** tanpa error atau kebocoran memori.
   - Zero compilation/lint errors (`No errors found`).

---

## 4. Handoff ke Fase 4 (Attendance Engine)

Fase 3 telah menyerahkan fondasi biometrik yang lengkap dan siap dikonsumsi langsung oleh Fase 4:
- Tabel `face_references` terisi embedding `vector(512)` aktif per karyawan, terindeks B-Tree `(employee_id, is_active, model_version)`.
- Kolom `employees.attendance_mode` siap memfilter karyawan berstatus `face` vs `manual`.
- Subsystem storage siap menerima bucket kedua `faceclock-attendance` (**REV-DB-03**).
- Middleware `RequireConsent` siap mengawal endpoint `#53` (Check-in) dan `#54` (Check-out).
- Katalog error telah mencakup `FACE_SERVICE_NOT_CONFIGURED` (**REV-ERR-06**) dan status `410 Gone` (**REV-ERR-04**).
