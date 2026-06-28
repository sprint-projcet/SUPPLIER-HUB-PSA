# Steering: Coding Conventions

## Prinsip Umum

- Ikuti struktur yang sudah ada sebelum menambah arsitektur baru.
- Pertahankan pemisahan folder backend: `config`, `controllers`, `middlewares`, `models`, dan `routes`.
- Pertahankan frontend sebagai HTML statis dan Vanilla JS kecuali ada keputusan eksplisit untuk migrasi framework.
- Gunakan istilah domain yang konsisten: UMKM untuk pembeli, Supplier untuk penjual, Admin untuk pengelola.
- Gunakan Bahasa Indonesia untuk copy UI dan pesan error yang dilihat user.
- Gunakan Bahasa Inggris seperlunya untuk nama teknis, identifier, endpoint, dan library.

## Konvensi Backend Go

- Jalankan `gofmt` pada semua perubahan Go.
- Package memakai nama folder: `controllers`, `models`, `routes`, `middlewares`, `config`.
- Controller menerima `*gin.Context` dan mengembalikan JSON dengan `c.JSON`.
- Gunakan status HTTP yang tepat:
  - `400` untuk input tidak valid.
  - `401` untuk belum login/token tidak valid.
  - `403` untuk role tidak diizinkan.
  - `404` untuk data tidak ditemukan.
  - `409` untuk konflik data seperti email sudah terdaftar.
  - `500` untuk error internal.
- Response error saat ini memakai bentuk `gin.H{"error": "..."};` pertahankan pola ini agar frontend mudah membaca error.
- Response sukses boleh memakai `message`, `status`, dan `data`; untuk endpoint baru pilih satu pola dan konsisten dalam satu modul.
- Gunakan `ShouldBindJSON` untuk body JSON.
- Gunakan `PostForm` dan `FormFile` untuk endpoint multipart/form-data.
- Jangan menyimpan password plaintext; selalu hash dengan bcrypt.
- Jangan kirim `password_hash` ke response JSON.
- Simpan role sebagai nilai `models.Role`: `user`, `supplier`, atau `admin`.
- Gunakan model GORM dan relation/preload ketika response perlu data terkait.
- Gunakan UUID string untuk entity utama yang mengikuti pola `User`, `Product`, dan `Order`.

## Konvensi Auth Dan Role

- Endpoint publik hanya auth/register/login/OAuth.
- Endpoint bisnis berada di bawah `/api` dan dilindungi `RequireAuth()`.
- Batasi group route dengan `RequireRole(...)`.
- Frontend mengirim JWT melalui:

```http
Authorization: Bearer <token>
```

- Sumber sesi frontend utama adalah `localStorage.user_session`.
- Saat memperbaiki atau menambah auth, samakan JWT claims antara controller login dan middleware. Pilih satu nama claim untuk user ID, lalu gunakan konsisten di semua controller.
- Hindari menambah key localStorage baru untuk token jika `user_session.token` sudah cukup.

## Konvensi Database Dan Model

- Tambahkan field model dengan tag GORM dan tag JSON yang eksplisit.
- Gunakan snake_case untuk field JSON, contoh `business_name`, `image_url`, `grand_total`.
- Gunakan enum string untuk status dan role agar mudah dibaca frontend.
- Jika menambah tabel baru, daftarkan ke `AutoMigrate` di `config/database.go`.
- Hindari query mentah jika GORM sudah cukup jelas.
- Untuk relasi yang ditampilkan ke frontend, gunakan `Preload` secara eksplisit agar response lengkap dan terkontrol.

## Konvensi API

- Pertahankan route aktif berbasis role:
  - `/api/user/...` untuk UMKM.
  - `/api/supplier/...` untuk Supplier.
  - `/api/admin/...` untuk Admin.
- Jika PRD menyebut `/api/items`, pastikan mapping-nya diselaraskan dengan implementasi `/api/*/products` sebelum menambah endpoint paralel.
- Untuk endpoint create/update yang menerima file, gunakan `multipart/form-data` dan jangan set header `Content-Type` manual di frontend.
- Untuk endpoint JSON, set header `Content-Type: application/json`.
- Gunakan query parameter sederhana untuk katalog:
  - `search` untuk pencarian.
  - `sort_by=price_asc` atau `sort_by=price_desc` untuk sorting harga.
- Tetap hitung fee 3 persen di backend, bukan frontend.

## Konvensi Frontend HTML/CSS

- Gunakan Tailwind utility classes seperti pola halaman yang sudah ada.
- Pertahankan palet utama emerald/teal `#02C39A` untuk identitas SupplierHub.
- Gunakan Lucide Icons untuk ikon dashboard dan tombol.
- Panggil `lucide.createIcons()` setelah render HTML dinamis yang berisi `data-lucide`.
- Gunakan layout dashboard yang konsisten: sidebar kiri, top bar, konten utama.
- Pertahankan responsivitas dengan class Tailwind seperti `lg:`, `md:`, `grid`, `flex`, dan `overflow-x-auto`.
- Jangan membuat style global besar baru jika utility Tailwind cukup.
- Hindari mengganti struktur semua dashboard sekaligus untuk perubahan kecil.

## Konvensi Frontend JavaScript

- Pakai Vanilla JS dan Fetch API.
- Untuk halaman dashboard, panggil `checkAuth('../Login/login.html')` saat `DOMContentLoaded` atau `window.onload`.
- Setelah auth, validasi role sebelum menampilkan konten.
- Simpan token dari `user_session.token` dan kirim sebagai Bearer token.
- Tangani loading state pada tombol submit dengan `disabled = true` dan teks proses.
- Tangani error API dengan membaca `data.error` atau `result.error`.
- Gunakan `Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' })` untuk harga Rupiah.
- Gunakan `encodeURIComponent` untuk query pencarian.
- Untuk render list dinamis, kosongkan container lalu render ulang dari data terbaru.
- Setelah render list yang memakai ikon Lucide, panggil ulang `lucide.createIcons()`.

## Konvensi Upload

- Supplier registration mengirim dokumen dengan field `document`.
- Product creation mengirim gambar dengan field `image`.
- Jangan set `Content-Type` secara manual saat memakai `FormData`; biarkan browser membuat boundary.
- Validasi keberadaan file supplier di backend karena supplier wajib mengunggah dokumen legalitas.

## Konvensi Dokumentasi

- Simpan dokumen project di `dokumen/`.
- PRD menjelaskan target produk; steering files menjelaskan kondisi dan arah implementasi saat ini.
- Jika implementasi berbeda dari PRD, dokumentasikan gap tersebut daripada diam-diam mengikuti asumsi lama.
- Gunakan contoh endpoint dan nama file yang benar-benar ada di workspace.

## Hal Yang Perlu Dihindari

- Jangan menambah framework frontend baru tanpa kebutuhan kuat.
- Jangan menyimpan token di banyak key localStorage yang berbeda.
- Jangan menghitung fee layanan hanya di frontend.
- Jangan membuat endpoint baru yang menduplikasi endpoint aktif tanpa migrasi route yang jelas.
- Jangan hardcode secret production di source code.
- Jangan menghapus perubahan user di worktree ketika hanya mengerjakan fitur/dokumen kecil.
