<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="frontend/public/brand/koreksiin-logo-gelap.svg" />
    <source media="(prefers-color-scheme: light)" srcset="frontend/public/brand/koreksiin-logo.svg" />
    <img alt="Koreksi.in Logo" src="frontend/public/brand/koreksiin-logo.svg" width="320" />
  </picture>
</p>

# Koreksi.in — Platform SaaS Penilaian Esai Otomatis Berbasis AI

**Koreksi.in** adalah platform *Software-as-a-Service* (SaaS) penilaian jawaban esai Bahasa Indonesia secara otomatis untuk membantu guru dan pengajar menilai ratusan jawaban siswa secara objektif, terstruktur per kriteria rubrik, dan konsisten. Seluruh pemrosesan kecerdasan buatan berjalan 100% secara lokal di atas CPU menggunakan model *open-source* tanpa ketergantungan pada layanan AI berbayar eksternal.

---

## Daftar Isi

1. [Gambaran Arsitektur](#1-gambaran-arsitektur)
2. [Prasyarat Sistem](#2-prasyarat-sistem)
3. [Konfigurasi Environment (.env)](#3-konfigurasi-environment-env)
4. [Cara Menjalankan Aplikasi](#4-cara-menjalankan-aplikasi)
5. [Cara Mendaftar Akun Guru & Masuk sebagai Admin](#5-cara-mendaftar-akun-guru--masuk-sebagai-admin)
6. [Konfigurasi Tunnel & Webhook Midtrans Sandbox](#6-konfigurasi-tunnel--webhook-midtrans-sandbox)
7. [Cara Membayar di Simulator Midtrans Sandbox v2](#7-cara-membayar-di-simulator-midtrans-sandbox-v2)
8. [Menjalankan Skrip Kalibrasi & Ablation 5-Fold CV](#8-menjalankan-skrip-kalibrasi--ablation-5-fold-cv)
9. [Skenario Demo Presentasi (5–7 Menit)](#9-skenario-demo-presentasi-57-menit)
10. [Troubleshooting Singkat](#10-troubleshooting-singkat)

---

## 1. Gambaran Arsitektur

Koreksi.in dikemas menggunakan **Docker Compose** dan terdiri atas layanan-layanan berikut:

| Layanan (`service`) | Teknologi | Peran Utama |
| :--- | :--- | :--- |
| **`frontend`** | Vue 3, Vite, Tailwind CSS, Pinia, Nginx (`:8080`) | Antarmuka pengguna (Landing, Auth, Ruang Kerja Guru, Panel Administrator) serta *reverse proxy* `/api/*` ke `backend:8000`. |
| **`backend`** | Go 1.25 (Gin, `pgx/v5`, `go-redis/v9`, `excelize/v2`) | REST API utama, autentikasi JWT, manajemen tugas & rubrik, impor/ekspor Excel, integrasi Midtrans Snap & verifikasi webhook SHA512, serta manajemen kuota langganan. |
| **`worker`** | Python 3.11, PyTorch CPU, `sentence-transformers` | Mengonsumsi antrean `queue:scoring` dari Redis untuk menilai esai siswa, menghitung cakupan kriteria rubrik, menyusun umpan balik otomatis, dan mendeteksi kemiripan jawaban antar-siswa. |
| **`trainer`** | Python 3.11, PyTorch CPU, `peft` (LoRA) | Layanan terpisah (image yang sama dengan `worker`, dibatasi dengan `TRAINER_THREADS`) yang khusus mengonsumsi antrean `queue:training` agar proses pelatihan model custom tidak menghambat antrean penilaian esai. |
| **`postgres`** | PostgreSQL 16 Alpine (`koreksiin_pg_data`) | Basis data relasional utama (17 tabel dengan migrasi otomatis saat `backend` dimulai). |
| **`redis`** | Redis 7 Alpine (`koreksiin_redis_data`) | Antrean pekerjaan asinkron (`queue:scoring` dan `queue:training`) serta pemantauan detak jantung (*heartbeat*) *worker*. |
| **`tunnel`** | `cloudflare/cloudflared` (Profil `tunnel`) | Membuat *Quick Tunnel* publik (`*.trycloudflare.com`) yang meneruskan webhook pembayaran dari Midtrans Sandbox ke container aplikasi lokal. |

### Alur Penilaian Esai & Model AI

1. **Model Bawaan (`LazarusNLP/all-indobert-base-v4`)**:
   - Jawaban siswa dipecah menjadi potongan kalimat/klausa dan di-encode menggunakan *mean pooling* menjadi vektor berdimensi 768.
   - Setiap poin kunci pada kriteria rubrik dicocokkan dengan potongan jawaban siswa menggunakan *cosine similarity* maksimum ($s$).
   - Skor poin kunci diskalakan ulang menggunakan ambang bawah ($T_{\text{LOW}}$) dan ambang atas ($T_{\text{HIGH}}$):
     $$s' = \text{clip}\left(\frac{s - T_{\text{LOW}}}{T_{\text{HIGH}} - T_{\text{LOW}}}, 0, 1\right)$$
   - Cakupan kriteria $C_i$ adalah rata-rata $s'$ dari seluruh poin kunci pada kriteria ke-$i$, dan nilai total dihitung secara proporsional terhadap bobot kriteria serta `max_score` tugas.
   - Umpan balik otomatis secara spesifik menyebutkan poin-poin kunci yang belum tercakup di jawaban siswa.
2. **Deteksi Kemiripan Ganda**:
   - Membandingkan seluruh pasangan jawaban siswa pada tugas yang sama menggunakan kombinasi kemiripan semantik (*cosine similarity* vektor dokumen $\ge S_{\text{SEM}}$) atau kemiripan leksikal (*Jaccard token overlap* $\ge S_{\text{LEX}}$).
3. **Model Custom Guru (Mode Kalibrasi & Mode Fine-Tune Mendalam LoRA)**:
   - Setiap revisi nilai yang dilakukan guru pada tugas mana pun (maupun unggahan Excel sampel latih) disimpan di tabel `training_samples` yang terhubung ke tugas (`assignment_id`).
   - Guru dapat memilih satu atau beberapa tugas sebagai sumber data latih untuk model custom.
   - Menggunakan arsitektur **Differentiable Key-Point Matcher** di atas **8 fitur agregat berukuran tetap** (sehingga model custom dapat digunakan lintas tugas dengan jumlah kriteria rubrik yang berbeda).
   - Menyediakan dua mode pelatihan:
     - **Mode Kalibrasi (Default)**: Melatih *head* pencocok kunci yang *differentiable* di atas 8 fitur agregat tanpa mengubah bobot dasar IndoBERT (~5–10 detik di CPU).
     - **Mode Fine-tune Mendalam (LoRA)**: Melatih kedua pendekatan (Kalibrasi dan LoRA `r=8` pada `query` & `value` `LazarusNLP/all-indobert-base-v4` dengan `lr=1e-4` tanpa pergeseran *dropout*), membandingkan MAE *5-Fold Cross-Validation*, menyimpan keduanya sebagai versi terpisah, dan merekomendasikan versi terbaik.
   - Seluruh metrik evaluasi (`MAE` dan korelasi `Pearson`) dihitung menggunakan **5-Fold Cross-Validation (`seed=42`)** pada data uji yang tidak dilihat saat pelatihan, kemudian model final dilatih ulang menggunakan 100% sampel.

---

## 2. Prasyarat Sistem

- **Docker Engine** & **Docker Compose v2** (atau Docker Desktop untuk Windows/macOS/Linux).
- **RAM**: Minimal 4 GB ruang bebas untuk container (disarankan 8 GB).
- **Koneksi Internet**: Diperlukan saat pertama kali menjalankan container `worker` untuk mengunduh model `LazarusNLP/all-indobert-base-v4` (~440 MB, disimpan permanen di volume Docker `koreksiin_hf_cache`) serta saat menggunakan pembayaran Midtrans Sandbox dan Cloudflare Quick Tunnel.
- **Akun Midtrans (Mode Sandbox)**: Untuk memperoleh *Server Key* dan *Client Key* Sandbox serta mengatur URL notifikasi webhook.

---

## 3. Konfigurasi Environment (`.env`)

Salin file `.env.example` menjadi `.env` di direktori akar proyek:

```bash
# Linux / macOS
cp .env.example .env

# Windows PowerShell
Copy-Item .env.example .env
```

Buka file `.env` dan sesuaikan variabel berikut (jangan pernah melakukan *commit* file `.env` ke repositori Git):

```ini
# Port aplikasi (Frontend Nginx + Proxy API)
APP_PORT=8080

# Konfigurasi PostgreSQL & Redis
POSTGRES_USER=koreksiin
POSTGRES_PASSWORD=ganti_password_db_anda
POSTGRES_DB=koreksiin_db
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
REDIS_ADDR=redis:6379

# Keamanan JWT & Seed Akun Administrator
JWT_SECRET=ganti_dengan_secret_acak_minimal_32_karakter
ADMIN_EMAIL=admin@koreksi.in
ADMIN_PASSWORD=AdminKoreksi2026!
ADMIN_NAME=Administrator Koreksi.in

# Konfigurasi AI & Ambang Default (Hasil Kalibrasi all-indobert-base-v4)
HF_MODEL_NAME=LazarusNLP/all-indobert-base-v4
DEFAULT_THRESHOLD_HIGH=0.60
DEFAULT_THRESHOLD_LOW=0.45
DEFAULT_SIMILARITY_SEMANTIC=0.92
DEFAULT_SIMILARITY_LEXICAL=0.60
MIN_TRAIN_SAMPLES=20
TRAINER_THREADS=2

# Konfigurasi Midtrans Sandbox (Isi dengan Key Sandbox dari Dashboard Midtrans Anda)
MIDTRANS_ENV=sandbox
MIDTRANS_IS_PRODUCTION=false
MIDTRANS_SERVER_KEY=SB-Mid-server-XXXXXXXXXXXXXXXXXXXXXXXX
MIDTRANS_CLIENT_KEY=SB-Mid-client-XXXXXXXXXXXXXXXXXXXXXXXX

# Opsional: URL publik tetap (kosongkan bila memakai Cloudflare Quick Tunnel otomatis)
PUBLIC_BASE_URL=
```

---

## 4. Cara Menjalankan Aplikasi

Untuk menjalankan seluruh layanan aplikasi beserta Cloudflare Quick Tunnel secara bersamaan:

```bash
docker compose --profile tunnel up --build -d
```

Apabila Anda hanya ingin menjalankan aplikasi lokal tanpa tunnel webhook:

```bash
docker compose up --build -d
```

Setelah container berstatus *Started* / *Healthy*, buka peramban dan akses:

- **Aplikasi Web Koreksi.in**: `http://localhost:8080`
- **Endpoint Pemeriksaan Kesehatan API**: `http://localhost:8080/api/health`

> [!NOTE]
> Saat pertama kali menjalankan penilaian esai, container `worker` akan mengunduh bobot model `LazarusNLP/all-indobert-base-v4` dari Hugging Face ke dalam volume Docker `koreksiin_hf_cache`. Proses ini hanya terjadi satu kali pada saat pemakaian perdana.

---

## 5. Cara Mendaftar Akun Guru & Masuk sebagai Admin

### Mendaftar Akun Guru Baru
1. Buka `http://localhost:8080/daftar` (atau klik tombol **Daftar Guru** di halaman beranda).
2. Isi **Nama Lengkap & Gelar**, **Alamat Email Aktif**, **Sekolah / Instansi**, dan **Kata Sandi** (minimal 6 karakter).
3. Klik **Daftar & Aktifkan Paket Gratis**.
4. Sistem secara otomatis mengaktifkan **Paket Gratis** (kuota bawaan direset setiap 30 hari) dan mengarahkan Anda ke **Ruang Kerja Guru** (`/guru`).

### Masuk sebagai Administrator
1. Buka `http://localhost:8080/masuk`.
2. Masuk menggunakan kredensial `ADMIN_EMAIL` dan `ADMIN_PASSWORD` yang diatur pada file `.env` (default: `admin@koreksi.in` / `AdminKoreksi2026!`).
3. Anda akan diarahkan ke **Panel Administrator** (`/admin`).

---

## 6. Konfigurasi Tunnel & Webhook Midtrans Sandbox (Otomatis)

Koreksi.in dilengkapi mekanisme **pengiriman URL notifikasi webhook otomatis per transaksi** menggunakan header `X-Override-Notification` dari Midtrans Snap, sehingga **Anda tidak perlu mengubah atau memasukkan ulang Payment Notification URL secara manual di Dashboard Midtrans Sandbox setiap kali URL tunnel berubah**:

1. Jalankan aplikasi dengan menyertakan profil `tunnel`:
   ```bash
   docker compose --profile tunnel up -d
   ```
2. Service `tunnel` (`cloudflare/cloudflared`) berjalan dengan endpoint metrik internal di jaringan Docker (`--metrics 0.0.0.0:2000 --url http://frontend:80`).
3. Setiap kali guru membuat transaksi Snap baru (`POST /api/payments/checkout`), `backend` mengambil hostname tunnel aktif dari `http://tunnel:2000/quicktunnel` (di-*cache* selama 60 detik) dan mengirimkan header:
   ```text
   X-Override-Notification: https://<hostname>.trycloudflare.com/api/payments/midtrans/notification
   ```
   langsung ke Midtrans Snap.
4. **Urutan Fallback URL Notifikasi (`ResolveNotificationURL`)**:
   - **Prioritas 1 (`PUBLIC_BASE_URL`)**: Jika variabel `PUBLIC_BASE_URL` di `.env` diisi (misalnya saat di-*deploy* di domain permanen), backend memakai `<PUBLIC_BASE_URL>/api/payments/midtrans/notification`.
   - **Prioritas 2 (Cloudflare Quick Tunnel Otomatis)**: Jika `PUBLIC_BASE_URL` kosong, backend mengambil URL aktif dari `http://tunnel:2000/quicktunnel`.
   - **Prioritas 3 (Default Dashboard Midtrans)**: Jika container `tunnel` tidak sedang berjalan atau tidak dapat diakses, backend mencatat peringatan di log dan tidak mengirim header `X-Override-Notification` (sehingga Midtrans menggunakan URL bawaan di Dashboard Midtrans).
5. **Rekonsiliasi Pembayaran Otomatis & Manual**:
   - Apabila webhook sempat tertunda atau gagal masuk karena kendala jaringan, saat guru membuka halaman **Langganan & Kuota** (`/guru/langganan`) atau mengeklik tombol **Cek status pembayaran**, `backend` otomatis memeriksa transaksi yang masih `pending` ke **Midtrans Status API (`GET /v2/{order_id}/status`)** dan memproses perubahan status (`settlement`, penambahan kuota, dan masa aktif) secara idempoten.
6. Anda juga dapat melihat **URL Notifikasi Webhook Midtrans Aktif** saat ini secara *read-only* di halaman **Dashboard Administrator** (`/admin`).

---

## 7. Cara Membayar di Simulator Midtrans Sandbox v2

Pilihan metode pembayaran yang tampil di Midtrans Snap mengikuti pengaturan **Payment methods** di Dashboard Midtrans Sandbox Anda (QRIS, GoPay, ShopeePay, Virtual Account Bank, Gerai Retail Indomaret/Alfamart, serta Kartu Kredit/Debit).

1. Masuk sebagai **Guru** di `http://localhost:8080/masuk`, lalu buka menu **Langganan & Kuota** (`/guru/langganan`).
2. Pilih paket berbayar (misalnya **Paket Guru Bulanan — Rp 49.000** atau **Add-on Kuota +150 Esai — Rp 20.000**) dan klik **Beli Paket Ini**.
3. Jendela **Midtrans Snap** akan terbuka (atau Anda dapat mengeklik tombol **Bayar (Snap)** / **Link Snap** pada tabel Riwayat Transaksi).

### A. Menguji Pembayaran via QRIS di Simulator v2
1. Pada jendela Snap, pilih metode pembayaran **QRIS** (atau **GoPay**) hingga kode QR ditampilkan.
2. Klik kanan pada gambar kode QR dan pilih **Copy Image Address** (Salin Alamat Gambar).
3. Buka halaman **Simulator QRIS Midtrans Sandbox v2** di tab baru:
   `https://simulator.sandbox.midtrans.com/v2/qris/index`
4. Tempel URL gambar QR pada kolom *QR Code Image URL*, klik **Scan QR**, lalu klik **Pay** hingga muncul keterangan pembayaran berhasil (**PAID**).

### B. Menguji Pembayaran via Virtual Account Bank di Simulator v2
1. Pada jendela Snap, pilih **Virtual Account** lalu pilih bank yang diinginkan (misalnya **BCA**, **BNI**, **BRI**, **Mandiri Bill**, **Permata**, atau **CIMB**) hingga nomor Virtual Account ditampilkan.
2. Klik tombol **Copy** (Salin) pada **Virtual account number** (untuk Mandiri Bill, catat *Company Code / Biller Code* dan *Bill Key*). Selama belum dibayar, transaksi akan tercatat di riwayat dengan status **Menunggu Pembayaran (`pending`)** beserta nama bank Virtual Account-nya.
3. Buka **Simulator Midtrans Sandbox v2** untuk Virtual Account:
   - Halaman utama simulator v2 (pilih menu **Virtual Account** lalu pilih bank): `https://simulator.sandbox.midtrans.com/`
   - Atau langsung ke halaman bank terkait (contoh BCA VA: `https://simulator.sandbox.midtrans.com/bca/va/index`, BNI VA: `https://simulator.sandbox.midtrans.com/bni/va/index`, BRI VA: `https://simulator.sandbox.midtrans.com/bri/va/index`, Permata VA: `https://simulator.sandbox.midtrans.com/permata/va/index`, Mandiri Bill: `https://simulator.sandbox.midtrans.com/mandiri/bill/index`).
4. Tempel **Virtual Account Number** pada kolom yang tersedia, klik **Inquire**, lalu klik **Pay** hingga muncul status **PAID / Successful**.

### C. Verifikasi Status & Kuota
- Kembali ke halaman **Langganan & Kuota** (`/guru/langganan`):
  - Setelah webhook diterima melalui URL tunnel otomatis (`X-Override-Notification`) dan signature SHA512 tervalidasi (atau saat Anda mengeklik tombol **Cek status pembayaran**), status transaksi berubah menjadi **Lunas (Settlement)** dan menampilkan metode pembayaran yang dipakai (misalnya `QRIS`, `BCA Virtual Account`, `BNI Virtual Account`, `Indomaret`, dll.).
  - Kuota esai dan masa aktif paket Anda langsung bertambah.
  - Apabila Virtual Account atau kode pembayaran gerai retail melewati batas waktu tanpa dibayar, status akhirnya akan otomatis berubah menjadi **Kedaluwarsa (`expire`)** tanpa menambah kuota.

---

## 8. Menjalankan Skrip Kalibrasi & Ablation 5-Fold CV

Repositori ini menyertakan dua skrip pengujian ilmiah di direktori `worker/scripts/` yang menggunakan dataset sampel 30 siswa pada `samples/contoh_fotosintesis_30_siswa.xlsx`:

### 1. Skrip Kalibrasi Ambang (`calibrate_sample.py`)
Menghitung distribusi *cosine similarity* antara pasangan poin kunci dan potongan kalimat siswa yang cocok vs. tidak cocok, distribusi nilai per kelompok siswa (lengkap, sebagian, melenceng), serta kemiripan semantik dan leksikal antar-siswa:

```bash
docker compose exec worker python scripts/calibrate_sample.py
```

### 2. Skrip Evaluasi & Ablation 5-Fold Cross-Validation (`ablation_cv.py`)
Menjalankan evaluasi *5-Fold Cross-Validation (`seed=42`)* untuk membandingkan (a) Model Bawaan (`all-indobert-base-v4`), (b) Mode Kalibrasi (`DifferentiableKeyMatcherHead` di atas 8 fitur agregat tanpa LoRA), dan (c) Mode Fine-tune Mendalam (`LoRA + Head`):

```bash
docker compose exec trainer python scripts/ablation_cv.py
```

---

## 9. Skenario Demo Presentasi (5–7 Menit)

Berikut urutan langkah demo terpadu berdurasi sekitar **5–7 menit** untuk mempresentasikan seluruh fitur Koreksi.in dari awal hingga akhir:

### Langkah 1 — Pendaftaran Guru & Kuota Gratis (30 detik)
1. Buka `http://localhost:8080`, tunjukkan tampilan beranda dan fitur utama, lalu klik **Daftar Guru** (`/daftar`).
2. Daftarkan akun guru baru (misalnya `Ibu Ratna Sari, S.Pd.` — `SMA Negeri 1 Surakarta`).
3. Tunjukkan bahwa guru langsung masuk ke **Ruang Kerja Guru** (`/guru`) dengan **Paket Gratis** yang aktif secara otomatis.

### Langkah 2 — Membuat Tugas Esai & Rubrik Fotosintesis (1 menit)
1. Klik **Buat Tugas Baru** (`/guru/tugas/baru`).
2. Isi judul tugas **Uji Pemahaman Konsep Fotosintesis**, mata pelajaran **Biologi Kelas X**, nilai maksimum `100`, dan soal esai dari sheet `Soal dan Rubrik` pada file `samples/contoh_fotosintesis_30_siswa.xlsx`.
3. Tambahkan 3 kriteria rubrik beserta bobotnya (`Definisi Proses Fotosintesis` 35%, `Bahan Baku dan Hasil` 35%, `Tempat Terjadinya dan Peran Klorofil` 30%) dengan poin kunci berupa kalimat utuh per baris sesuai sheet `Soal dan Rubrik`, lalu simpan tugas.

### Langkah 3 — Unggah Jawaban 30 Siswa & Penilaian Otomatis (1 menit)
1. Pada halaman detail tugas, unggah file `samples/contoh_fotosintesis_30_siswa.xlsx` melalui zona unggah Excel (30 jawaban siswa langsung masuk).
2. Klik **Nilai Semua Jawaban**. Tunjukkan bilah progres penilaian dan pemotongan kuota esai secara *real-time*.

### Langkah 4 — Analisis Nilai, Feedback Poin Kunci, Kemiripan, & Revisi Guru (1,5 menit)
1. Tunjukkan tabel hasil penilaian:
   - Siswa 1–10 (jawaban lengkap) memperoleh nilai tinggi (rata-rata hasil uji pada data contoh: **99,17**).
   - Siswa 27–30 (jawaban melenceng) memperoleh nilai **0,00**.
2. Klik salah satu siswa dengan jawaban sebagian (misalnya siswa nomor 11–26) untuk membuka rincian kriteria (`Terpenuhi`, `Sebagian`, `Tidak Ada`) dan tunjukkan **feedback otomatis** yang secara spesifik menyebutkan poin kunci yang belum dijelaskan siswa.
3. Tunjukkan panel **Peringatan Kemiripan Jawaban** yang mendeteksi pasangan **Bagas Setiawan** dan **Dimas Arya Putra** (kemiripan semantik `0.9469` $\ge 0.92$).
4. Ubah nilai salah satu siswa melalui fitur **Revisi Nilai Guru** dan simpan — jelaskan bahwa setiap revisi otomatis tersimpan sebagai sampel latih di `training_samples`. Klik **Ekspor Hasil Excel** untuk mengunduh rekap nilai.

### Langkah 5 — Pelatihan Model Custom "Biologi Kelas 10" (1,5 menit)
1. Buka menu **Model Custom** (`/guru/model-custom`), buat model baru bernama **Biologi Kelas 10**, dan centang tugas **Fotosintesis** sebagai sumber data latih.
2. Unggah sheet `Jawaban Siswa` (kolom `nama_siswa`, `jawaban`, `nilai`) dari `samples/contoh_fotosintesis_30_siswa.xlsx` melalui tombol **Unggah Excel Sampel Latih** (30 sampel latih), lalu klik **Latih Model Sekarang** (mode default **Kalibrasi** selesai dalam ~6 detik).
3. Tunjukkan kartu **Riwayat Versi & Metrik Evaluasi (5-Fold Cross-Validation)**. Sebagai **hasil uji pada data contoh**:
   - **MAE 5-Fold CV** turun dari **15,68** (Model Bawaan) menjadi **6,40** (Model Custom mode Kalibrasi).
   - **Korelasi Pearson 5-Fold CV** meningkat dari **0,8656** menjadi **0,9577**, memperbaiki penilaian kelompok jawaban "sebagian" agar sesuai standar koreksi guru.

### Langkah 6 — Pembayaran Paket via QRIS Midtrans Sandbox & Panel Admin (1,5 menit)
1. Buka menu **Langganan & Kuota** (`/guru/langganan`), pilih **Paket Guru Bulanan (Rp 49.000)**, dan klik **Beli Paket Ini** (backend otomatis menyertakan URL webhook tunnel aktif lewat `X-Override-Notification` tanpa perlu mengatur ulang URL di Dashboard Midtrans).
2. Pilih metode **QRIS** pada Midtrans Snap, salin alamat gambar QR, lalu bayar melalui `https://simulator.sandbox.midtrans.com/v2/qris/index`.
3. Tunjukkan bahwa status transaksi langsung berubah menjadi **Lunas (Settlement)** dan kuota bertambah **+300 esai** berkat verifikasi webhook Midtrans (atau klik tombol **Cek status pembayaran** untuk rekonsiliasi instan ke Midtrans Status API).
4. Keluar dari akun guru dan masuk sebagai **Administrator** (`admin@koreksi.in`) di `/admin` untuk menampilkan statistik platform, informasi *read-only* **URL Notifikasi Webhook Midtrans Aktif**, manajemen status akun guru, log transaksi & verifikasi signature webhook Midtrans, pengaturan paket, serta pengaturan ambang AI.

---

## 10. Troubleshooting Singkat

1. **Penilaian Pertama Kali Memakan Waktu Lebih Lama**:
   - Saat pekerjaan penilaian pertama dijalankan, container `worker` mengunduh model `LazarusNLP/all-indobert-base-v4` (~440 MB) ke volume `koreksiin_hf_cache`. Pantau progres unduhan dengan `docker compose logs -f worker`. Penilaian berikutnya berjalan instan dalam hitungan detik.
2. **Webhook Midtrans Terlambat atau Tidak Masuk Setelah Membayar di Simulator**:
   - Klik tombol **Cek status pembayaran** di halaman **Langganan & Kuota** (`/guru/langganan`) atau muat ulang halaman tersebut — backend akan langsung memeriksa transaksi `pending` ke Midtrans Status API dan mengaktifkan paket serta kuota Anda secara idempoten.
   - Pastikan container `tunnel` sedang berjalan (`docker compose --profile tunnel up -d`) dan periksa URL notifikasi aktif di halaman **Dashboard Administrator** (`/admin`).
3. **Mereset Database Tanpa Menghapus Cache Model AI**:
   - Jangan gunakan `docker compose down -v` jika tidak ingin mengunduh ulang model Hugging Face. Cukup hapus volume PostgreSQL saja:
     ```bash
     docker compose down
     docker volume rm koreksiin_pg_data
     docker compose --profile tunnel up -d
     ```
