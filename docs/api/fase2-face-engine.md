# FaceClock API Specification — Fase 2: Face Engine Integration & Attendance Pipeline

Dokumen ini mendefinisikan arsitektur pemrosesan biometrik, engine rekognisi wajah, dan kontrak endpoint HTTP API Fase 2 untuk backend FaceClock (`apps/faceclock-api`).
Sesuai konvensi standar arsitektur sistem (`docs/adr/0002-inference-prior-art.md` dan `docs/adr/0003-api-conventions.md`), Fase 2 mengintegrasikan pipeline pengenalan wajah berbasis deep learning (InsightFace / ArcFace Buffalo_L) dengan penyimpanan vektor berkecepatan tinggi di PostgreSQL (`pgvector`), penyimpanan foto snapshot berbasis `storage.Store`, serta pengamanan sesi berbasis cookie HttpOnly dan perisai Anti-CSRF.

---

## 1. Arsitektur Pemrosesan Biometrik & Skema Vektor

### 1.1 Model & Embedding
- **Model Biometrik**: ArcFace dengan arsitektur backbone ResNet-50 / ResNet-100 (InsightFace `buffalo_l`).
- **Dimensi Embedding**: Vektor 512 dimensi bertipe data floating-point 32-bit (`[]float32` di Go, `vector(512)` di PostgreSQL).
- **Normalisasi $L_2$**: Semua vektor dinormalisasi unit Euclidean ($||\mathbf{v}||_2 = 1.0$) sebelum disimpan dan dibandingkan:
  $$\mathbf{v}_{\text{norm}} = \frac{\mathbf{v}}{||\mathbf{v}||_2} = \frac{\mathbf{v}}{\sqrt{\sum_{i=1}^{512} v_i^2}}$$

### 1.2 Metrik Jarak: Cosine Similarity vs. Euclidean Distance
Untuk dua vektor ter-normalisasi $L_2$ ($\mathbf{u}$ dan $\mathbf{v}$):
1. **Cosine Similarity**:
   $$\text{sim}(\mathbf{u}, \mathbf{v}) = \frac{\mathbf{u} \cdot \mathbf{v}}{||\mathbf{u}||_2 ||\mathbf{v}||_2} = \sum_{i=1}^{512} u_i v_i$$
2. **Euclidean Distance ($L_2$ Distance)**:
   $$d_E(\mathbf{u}, \mathbf{v}) = \sqrt{\sum_{i=1}^{512} (u_i - v_i)^2}$$
3. **Hubungan Matematis Presisi**:
   $$d_E^2 = ||\mathbf{u} - \mathbf{v}||^2 = ||\mathbf{u}||^2 + ||\mathbf{v}||^2 - 2(\mathbf{u} \cdot \mathbf{v}) = 1 + 1 - 2 \cdot \text{sim} = 2(1 - \text{sim})$$
   $$d_E = \sqrt{2(1 - \text{sim})}$$

### 1.3 Ambang Batas Verifikasi (Threshold Calibration)
- **Cosine Similarity Threshold**: $\ge 0.75$ (Ambang batas verifikasi presisi tinggi dengan False Acceptance Rate (FAR) $< 0.01\%$).
- **Euclidean Distance Threshold**: $\le 0.60 - 0.70$:
  $$d_E = \sqrt{2(1 - 0.75)} = \sqrt{0.50} \approx 0.7071$$
- **Operasi Database PostgreSQL `pgvector`**:
  - Operasi Jarak Cosine: `1 - (face_embedding <=> $1::vector)` menghasilkan skor kemiripan $\text{sim} \in [-1, 1]$.
  - Operasi Jarak Euklidean: `face_embedding <-> $1::vector` menghasilkan jarak $d_E \ge 0$.

---

## 2. Pipa Kualitas Citra (Face Quality Pipeline)

Sebelum ekstraksi fitur dilakukan, setiap citra yang dikirim melalui endpoint Fase 2 divalidasi dengan kriteria kualitas berikut:

| Parameter | Ambang Batas Minimum | Kode Kesalahan / Hint |
|---|---|---|
| **Deteksi Wajah** | Tepat 1 wajah | `no_face_detected` / `multiple_faces` |
| **Dimensi Bounding Box** | $\ge 80 \times 80$ piksel | `face_too_small` |
| **Sudut Yaw (Toleh)** | $-30^\circ \le \text{yaw} \le +30^\circ$ | `extreme_pose` |
| **Sudut Pitch (Angguk)** | $-30^\circ \le \text{pitch} \le +30^\circ$ | `extreme_pose` |
| **Sudut Roll (Miring)** | $-20^\circ \le \text{roll} \le +20^\circ$ | `extreme_pose` |
| **Ketajaman (Sharpness)** | Variance of Laplacian $\ge 100.0$ | `blurry_image` |
| **Tingkat Kecerahan** | $40 \le \text{mean brightness} \le 220$ | `bad_illumination` |

Citra yang gagal memenuhi kriteria di atas langsung ditolak dengan status HTTP 422 `FACE_NOT_USABLE` tanpa menyentuh komparasi database.

---

## 3. Database Schema & Migration (000011)

### 3.1 Kolom Biometrik pada Tabel `employees`
```sql
ALTER TABLE employees
    ADD COLUMN face_embedding vector(512) NULL,
    ADD COLUMN face_photo_path text NULL,
    ADD COLUMN face_enrolled_at timestamptz NULL,
    ADD COLUMN face_model_version text NULL;
```

### 3.2 Tabel `face_references` (Penyimpanan Multi-Foto Referensi)
```sql
CREATE TABLE face_references (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    photo_key text NOT NULL,
    embedding vector(512) NOT NULL,
    model_version text NOT NULL,
    quality_score double precision NOT NULL DEFAULT 1.0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);
```

### 3.3 Tabel `attendances` (Catatan Absensi & Verifikasi Wajah)
```sql
CREATE TABLE attendances (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    type text NOT NULL CHECK (type IN ('in', 'out')),
    status text NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'pending_review')),
    photo_key text NOT NULL,
    face_embedding vector(512) NULL,
    similarity_score double precision NULL,
    distance double precision NULL,
    model_version text NOT NULL,
    notes text NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);
```

---

## 4. Keamanan: Cookie HttpOnly & Perisai Anti-CSRF

Setiap endpoint biometrik dan absensi mewarisi standar keamanan FaceClock:
- **Autentikasi Cookie**: Membaca cookie `access_token` (`HttpOnly`, `SameSite=Lax`, `Path=/`). Fallback ke header `Authorization: Bearer <token>` untuk client non-browser.
- **Anti-CSRF Shield**: Setiap request mutasi (`POST`) wajib mengirimkan salah satu header:
  - `X-Requested-With: XMLHttpRequest`
  - `X-CSRF-Token: <token>`
- **RBAC Matrix**:
  - `face.enroll_any`: Mengizinkan admin/HR mendaftarkan wajah karyawan mana pun.
  - `face.enroll_self`: Mengizinkan karyawan mendaftarkan wajahnya sendiri (`id` endpoint harus sama dengan `employee_id` token).
  - `attendance.checkin`: Mengizinkan karyawan melakukan absensi masuk (clock-in) atau keluar (clock-out).

---

## 5. Spesifikasi Endpoint Fase 2

### 5.1 POST `/api/v1/employees/{id}/face-enroll`
- **Guard**: `face.enroll_any|face.enroll_self`
- **Anti-CSRF**: Wajib (`X-Requested-With: XMLHttpRequest` atau `X-CSRF-Token`)
- **Format Input yang Didukung**:
  1. `multipart/form-data`: Field `photo`, `image`, atau `file`.
  2. `application/json`: Payload berisi base64 image (`photo_base64`).
  3. `image/jpeg` atau `image/png`: Raw binary stream di body request.
- **Deskripsi**: Mendaftarkan template wajah referensi biometrik untuk karyawan yang bersangkutan. Memvalidasi kualitas citra, mengekstraksi vektor 512-d, menyimpan snapshot foto di storage backend, dan memperbarui record karyawan dalam sebuah transaksi database terpadu.

#### Request Contoh (JSON Base64):
```http
POST /api/v1/employees/e44485a0-1bd7-4e61-82ac-c7036e5d0a7a/face-enroll HTTP/1.1
Host: api.faceclock.local
Content-Type: application/json
X-Requested-With: XMLHttpRequest

{
  "photo_base64": "/9j/4AAQSkZJRgABAQEASABIAAD..."
}
```

#### Response 201 Created:
```json
{
  "data": {
    "employee_id": "e44485a0-1bd7-4e61-82ac-c7036e5d0a7a",
    "photo_path": "faces/enrollments/e44485a0-1bd7-4e61-82ac-c7036e5d0a7a/1789034471619599200.jpg",
    "model_version": "buffalo_l",
    "quality_score": 0.92,
    "enrolled_at": "2026-09-10T17:01:11.622277+07:00"
  }
}
```

#### Kemungkinan Error:
- `401 UNAUTHENTICATED`: Token tidak ada atau tidak valid.
- `403 FORBIDDEN`: Karyawan mencoba mendaftarkan wajah karyawan lain tanpa izin `face.enroll_any`.
- `422 FACE_NOT_USABLE`: Kualitas foto tidak mencukupi, wajah miring berlebih, atau tidak ada wajah terdeteksi.

---

### 5.2 POST `/api/v1/attendance/clock-in`
- **Guard**: `attendance.checkin`
- **Anti-CSRF**: Wajib
- **Deskripsi**: Melakukan absensi masuk karyawan menggunakan verifikasi biometrik wajah. Vektor wajah dihitung secara langsung dan dicocokkan dengan embedding referensi karyawan melalui operator `pgvector` Cosine distance (`<=>`) di PostgreSQL.

#### Request Contoh (JSON Base64):
```http
POST /api/v1/attendance/clock-in HTTP/1.1
Host: api.faceclock.local
Content-Type: application/json
X-Requested-With: XMLHttpRequest

{
  "photo_base64": "/9j/4AAQSkZJRgABAQEASABIAAD...",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "accuracy": 12.5,
  "device_info": "iPhone 15 Pro / Safari 17",
  "notes": "Shift Pagi"
}
```

#### Response 201 Created (Verifikasi Berhasil):
```json
{
  "data": {
    "id": "8ceb1946-e80f-4080-aa06-96f1cb7788d4",
    "employee_id": "e44485a0-1bd7-4e61-82ac-c7036e5d0a7a",
    "type": "in",
    "status": "success",
    "photo_key": "attendances/e44485a0-1bd7-4e61-82ac-c7036e5d0a7a/in/1789034471816558400.jpg",
    "similarity_score": 0.9412,
    "distance": 0.3429,
    "model_version": "buffalo_l",
    "notes": "Shift Pagi",
    "recorded_at": "2026-09-10T17:01:11.821631+07:00",
    "created_at": "2026-09-10T17:01:11.821631+07:00"
  }
}
```

#### Response 422 Unprocessable Entity (Wajah Tidak Cocok):
```json
{
  "error": {
    "code": "FACE_MISMATCH",
    "message": "face verification failed: similarity 0.4210 below required threshold 0.75"
  }
}
```
*Catatan Keamanan*: Pada kegagalan `FACE_MISMATCH`, data percobaan tetap direkam pada tabel `attendances` dengan `status = 'failed'` untuk kebutuhan audit investigasi fraud biometrik.

---

### 5.3 POST `/api/v1/attendance/clock-out`
- **Guard**: `attendance.checkin`
- **Anti-CSRF**: Wajib
- **Deskripsi**: Melakukan absensi pulang karyawan dengan pipeline verifikasi biometrik wajah dan penyimpanan bukti kehadiran snapshot.

#### Request Contoh (Multipart Form):
```http
POST /api/v1/attendance/clock-out HTTP/1.1
Host: api.faceclock.local
Content-Type: multipart/form-data; boundary=----WebKitFormBoundaryXyZ
X-Requested-With: XMLHttpRequest

------WebKitFormBoundaryXyZ
Content-Disposition: form-data; name="photo"; filename="clockout.jpg"
Content-Type: image/jpeg

<binary JPEG data>
------WebKitFormBoundaryXyZ
Content-Disposition: form-data; name="notes"

Selesai lembur
------WebKitFormBoundaryXyZ--
```

#### Response 201 Created:
```json
{
  "data": {
    "id": "0bf19ae6-16da-46c3-98ce-c59ccb124b44",
    "employee_id": "e44485a0-1bd7-4e61-82ac-c7036e5d0a7a",
    "type": "out",
    "status": "success",
    "photo_key": "attendances/e44485a0-1bd7-4e61-82ac-c7036e5d0a7a/out/1789034471831957500.jpg",
    "similarity_score": 0.9531,
    "distance": 0.3062,
    "model_version": "buffalo_l",
    "notes": "Selesai lembur",
    "recorded_at": "2026-09-10T17:01:11.832957+07:00",
    "created_at": "2026-09-10T17:01:11.832957+07:00"
  }
}
```

---

## 6. Katalog Kode Error Fase 2

| Kode Error | HTTP Status | Keterangan & Tindakan |
|---|---|---|
| `FACE_NOT_ENROLLED` | 422 | Karyawan belum melakukan pendaftaran wajah referensi. Minta karyawan atau admin melakukan enroll wajah terlebih dahulu. |
| `FACE_MISMATCH` | 422 | Skor kemiripan wajah berada di bawah ambang batas ($\text{similarity} < 0.75$). Upaya absensi dicatat sebagai kegagalan. |
| `FACE_NOT_USABLE` | 422 | Foto tidak memenuhi kriteria kualitas (tidak ada wajah, wajah terpotong, buram, pencahayaan buruk, atau lebih dari 1 orang). |
| `EMPLOYEE_INACTIVE` | 422 | Karyawan berstatus tidak aktif (`inactive`). Absensi ditolak. |
| `CSRF_HEADER_MISSING` | 403 | Permintaan mutasi POST tidak menyertakan header `X-Requested-With` atau `X-CSRF-Token`. |
| `FORBIDDEN` | 403 | Token tidak memiliki permission yang diwajibkan untuk operasi terkait. |
| `PAYLOAD_TOO_LARGE` | 413 | Ukuran berkas foto melebihi limit maksimum konfigurasi server (`MaxBodyBytes`). |
