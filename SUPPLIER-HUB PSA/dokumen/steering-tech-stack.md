# Steering: Tech Stack

## Ringkasan

SupplierHub memakai frontend statis berbasis HTML, Tailwind CDN, dan Vanilla JavaScript. Backend memakai Go dengan Gin sebagai HTTP framework, GORM sebagai ORM, dan MySQL sebagai database aktif.

## Frontend

Teknologi utama:

- HTML5 untuk struktur halaman.
- TailwindCSS via CDN untuk styling utility-first.
- Vanilla JavaScript ES6 untuk interaksi, form handling, fetch API, dan render data.
- `auth.js` sebagai helper shared untuk login, sesi, logout, dan role config.
- `localStorage` untuk menyimpan sesi login, terutama key `user_session`.
- `sessionStorage` untuk flag UI sementara seperti `justLoggedOut`.
- Lucide Icons via CDN pada halaman login dan dashboard.
- SweetAlert2 via CDN untuk alert/toast di halaman tertentu.
- Font Awesome hanya terlihat pada landing page `index.html`.
- Google Fonts `Plus Jakarta Sans` pada halaman login.

Pola frontend saat ini:

- Halaman berada sebagai file `.html` mandiri.
- Script halaman umumnya inline di bagian bawah HTML.
- API dipanggil langsung dengan URL `http://localhost:8080`.
- JWT dikirim dengan header `Authorization: Bearer <token>`.
- Navigasi role diarahkan ke file dashboard berbeda: `umkm.html`, `supplier.html`, atau `admin.html`.

## Backend

Teknologi utama:

- Go module: `supplierhub-backend`.
- Go version pada `go.mod`: `1.26.2`.
- Gin `github.com/gin-gonic/gin` untuk routing dan controller HTTP.
- `github.com/gin-contrib/cors` untuk CORS.
- GORM `gorm.io/gorm` untuk ORM dan AutoMigrate.
- MySQL driver `gorm.io/driver/mysql` sebagai driver aktif di `config/database.go`.
- JWT `github.com/golang-jwt/jwt/v5` untuk token auth.
- Bcrypt `golang.org/x/crypto/bcrypt` untuk password hashing.
- Google OAuth `golang.org/x/oauth2` dan `golang.org/x/oauth2/google`.
- `github.com/joho/godotenv` untuk membaca `.env`.
- `github.com/google/uuid` untuk ID model.

Dependensi Postgres dan MongoDB ada di `go.mod`, tetapi implementasi aktif saat scan ini memakai MySQL.

## Database

Database aktif:

- MySQL.
- Default DSN development: `root:@tcp(127.0.0.1:3306)/supplierhub?charset=utf8mb4&parseTime=True&loc=Local`.
- Override DSN melalui environment variable `DATABASE_URL`.
- Skema dibuat lewat `AutoMigrate` untuk `User`, `Product`, `Order`, dan `Log`.

Seeder:

- Admin default dibuat jika email `admin@supplierhub.com` belum ada.
- Password default development: `admin123`.

## Environment Variables

Backend membaca environment berikut:

- `DATABASE_URL`: DSN database MySQL.
- `JWT_SECRET`: secret untuk signing JWT.
- `GOOGLE_CLIENT_ID`: Google OAuth client ID.
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret.
- `GOOGLE_REDIRECT_URL`: callback URL OAuth, contoh `http://localhost:8080/api/auth/google/callback`.

Catatan penting:

- Ada default JWT secret di kode untuk development. Jangan pakai default secret untuk production.
- Ada dua nilai fallback secret yang berbeda di middleware dan controller auth. Saat memperbaiki auth, samakan sumber secret dan format claims.

## Runtime Lokal

Backend:

```powershell
cd backend
go mod tidy
go run .
```

Server berjalan di:

```text
http://localhost:8080
```

Frontend:

- Karena frontend berupa HTML statis, halaman dapat dibuka langsung dari file browser.
- Jika browser membatasi akses tertentu, jalankan static server sederhana dari root project.
- Pastikan backend berjalan di `localhost:8080` karena URL API saat ini masih hardcoded.

## Uploads Dan Static Files

- Backend menyajikan folder `uploads` sebagai static files di path `/uploads`.
- Dokumen supplier disimpan di `uploads/documents`.
- Gambar produk disimpan di `uploads/` dan controller membuat URL `http://localhost:8080/uploads/<filename>`.

## Integrasi Eksternal

Rencana integrasi menurut PRD:

- SmartBank untuk payment request dan payment callback.
- LogistiKita untuk request pengiriman dan tracking.
- API Gateway sebagai jalur keluar/masuk ekosistem eksternal.

Status scan workspace:

- Konsep integrasi ada di PRD.
- Mock server belum ada di repo.
- Endpoint webhook payment belum terdaftar di routes aktif.

## Tooling Dan Testing

Belum terlihat konfigurasi khusus untuk:

- Test frontend.
- Test backend.
- Linter Go.
- Formatter frontend.
- Package manager frontend.
- CI/CD.

Gunakan tooling bawaan sampai project menambahkan standar eksplisit:

- `gofmt` untuk semua file Go.
- `go test ./...` dari folder `backend` ketika ada test atau setelah perubahan backend.
- Validasi manual browser untuk perubahan HTML/JS.
