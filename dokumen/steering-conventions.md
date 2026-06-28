# Steering Document: Coding Conventions

## 1. Konvensi Struktur Kode
- `backend/` untuk semua kode server.
- `backend/config/` untuk inisialisasi database, OAuth, dan konfigurasi aplikasi.
- `backend/controllers/` untuk logika handler HTTP dan business logic endpoint.
- `backend/middlewares/` untuk middleware JWT dan otorisasi.
- `backend/models/` untuk definisi entitas dan hooks Gorm.
- `backend/routes/` untuk peta route dan grup route berdasarkan role.

## 2. Konvensi Penamaan
- Gunakan `snake_case` untuk file Go yang menggambarkan peran modul: `auth_controller.go`, `database.go`, `routes.go`.
- Gunakan `PascalCase` untuk nama fungsi dan tipe diekspor di Go: `SetupRoutes`, `ConnectDatabase`, `Register`, `Login`, `RoleSupplier`.
- Gunakan `camelCase` untuk variabel lokal di JS dan input JSON.
- Gunakan `json` tags pada struct model untuk konsistensi response JSON.
- Gunakan `binding` tags pada DTO untuk validasi input Gin.

## 3. Konvensi Go
- Semua kode Go berada di package yang jelas dan ekspor hanya fungsi atau tipe yang perlu digunakan di luar package.
- Tangani error segera setelah pemanggilan fungsi dan kembalikan respons jika terjadi error.
- Gunakan `gorm` hooks seperti `BeforeCreate` untuk menghasilkan UUID secara otomatis.
- Gunakan konstanta tipe khusus untuk value domain penting, misalnya `Role`, `OrderStatus`.
- Tulis komentar ringkas pada fungsi publik dan block penting dengan Bahasa Indonesia agar konsisten dengan kode existing.
- Definisikan default konfigurasi dan fallback environment di `config`.

## 4. Konvensi API dan Route
- Gunakan group route berdasarkan domain dan role:
  - `/api/auth/*`
  - `/api/user/*`
  - `/api/supplier/*`
  - `/api/admin/*`
- Terapkan middleware otorisasi JWT global setelah route public auth.
- Masukkan validasi role di route group dengan middleware `RequireRole("user")`, `RequireRole("supplier")`, `RequireRole("admin")`.
- Gunakan status HTTP yang tepat: `201 Created` untuk pendaftaran sukses, `400 Bad Request` untuk validasi input, `401 Unauthorized`/`403 Forbidden` untuk masalah otentikasi/otorisasi.

## 5. Konvensi Frontend
- Simpan data sesi di `localStorage` dengan kunci yang konsisten, misalnya `user_session`.
- Pisahkan fungsi utilitas frontend seperti `loginUser()`, `checkAuth()`, `logoutUser()`, dan `getRoleConfig()`.
- Gunakan `fetch()` dengan header `Content-Type: application/json` untuk permintaan API JSON.
- Tangani respons API dengan memeriksa `response.ok` dan menampilkan pesan kesalahan yang jelas.
- Pastikan halaman dashboard role-based memeriksa autentikasi sebelum render.

## 6. Konvensi Dokumentasi
- Simpan dokumen steering di `dokumen/` untuk kemudahan referensi.
- Gunakan Bahasa Indonesia untuk deskripsi umum dan instruksi internal yang relevan dengan tim.
- Gunakan dokumen `readme.md` sebagai overview umum, dan pisahkan detil teknis ke `prd-*` atau `steering-*` jika perlu.

## 7. Praktik Keamanan Dasar
- Simpan secret JWT dalam environment variable `JWT_SECRET`, dan jangan commit ke repository.
- Jangan distribusikan password asli dalam response API.
- Validasi semua input terutama pada endpoint `Register` dan `Login`.
- Pastikan file upload hanya disimpan setelah validasi, dan folder `uploads` disajikan statis dengan benar.
