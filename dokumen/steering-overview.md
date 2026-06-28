# Steering: Project Overview

## Identitas Produk

SupplierHub adalah aplikasi B2B untuk mempertemukan UMKM pembeli bahan baku dengan Supplier penyedia stok. Sistem menyediakan katalog bahan, pemesanan, manajemen produk/stok, dashboard per role, dan fondasi integrasi pembayaran serta logistik.

Project ini memiliki tiga aktor utama:

- `user`: UMKM atau pembeli.
- `supplier`: penjual atau penyedia bahan baku.
- `admin`: pengelola platform dan verifikasi ekosistem.

## Tujuan Bisnis

- Mempermudah UMKM mencari dan memesan bahan baku dari banyak Supplier.
- Membantu Supplier mengelola produk, stok, harga, dan pesanan masuk.
- Memberi Admin visibilitas terhadap supplier, stok, transaksi, dan pendapatan platform.
- Menjaga aturan fee layanan platform sebesar 3 persen dari nilai dasar pesanan.
- Menyiapkan jalur integrasi eksternal melalui SmartBank, LogistiKita, dan API Gateway sesuai PRD.

## Alur Utama

1. Pengguna mendaftar atau login.
2. UMKM melihat katalog produk Supplier.
3. UMKM mencari produk dengan query pencarian dan dapat membuat pesanan.
4. Sistem menghitung `total_base_price`, `system_fee` sebesar 3 persen, dan `grand_total`.
5. Supplier mengelola produk dan melihat daftar pesanan.
6. Admin memonitor statistik, supplier, log, stok, dan keuangan.

Alur pembayaran SmartBank, webhook payment, dan pengiriman LogistiKita sudah dijelaskan di PRD, tetapi belum terlihat sebagai implementasi backend lengkap di route aktif saat scan ini.

## Struktur Workspace

- `index.html`: landing page SupplierHub.
- `Login/login.html`: halaman login, registrasi UMKM, registrasi Supplier, dan Google OAuth popup flow.
- `auth.js`: helper sesi frontend, login, logout, dan konfigurasi dashboard per role.
- `dashboard/`: halaman statis untuk role UMKM, Supplier, dan Admin.
- `assets/`: aset visual, termasuk logo.
- `backend/`: aplikasi API Go.
- `backend/main.go`: entrypoint server Gin, CORS, static uploads, route setup.
- `backend/routes/routes.go`: definisi endpoint `/api`.
- `backend/controllers/`: controller auth, UMKM, supplier, dan admin.
- `backend/models/models.go`: model GORM untuk `User`, `Product`, `Order`, dan `Log`.
- `backend/config/`: koneksi database dan Google OAuth.
- `backend/middlewares/`: middleware JWT dan role authorization.
- `dokumen/`: PRD, development plan, workflow dot files, dan steering files.

## Endpoint Aktif

Endpoint publik:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/google`
- `GET /api/auth/google/callback`

Endpoint dengan JWT dan role `user`:

- `GET /api/user/stats`
- `GET /api/user/orders`
- `GET /api/user/products`
- `POST /api/user/orders`

Endpoint dengan JWT dan role `supplier`:

- `GET /api/supplier/stats`
- `GET /api/supplier/products`
- `POST /api/supplier/products`
- `GET /api/supplier/orders`
- `PUT /api/supplier/orders/:id`

Endpoint dengan JWT dan role `admin`:

- `GET /api/admin/stats`
- `GET /api/admin/suppliers`
- `PUT /api/admin/suppliers/:id/verify`
- `GET /api/admin/logs`

Catatan: PRD lama masih menyebut `/api/items` dan `/api/orders`; implementasi saat ini memakai group role seperti `/api/user/products`, `/api/user/orders`, dan `/api/supplier/products`.

## Domain Model

- `User`: identitas UMKM, Supplier, atau Admin. Field penting: `business_name`, `email`, `password_hash`, `role`, `address`, `category`, `region`, `document_url`, `status`.
- `Product`: produk Supplier. Field penting: `supplier_id`, `name`, `category`, `price`, `stock`, `description`, `location`, `image_url`.
- `Order`: pesanan UMKM ke Supplier. Field penting: `umkm_id`, `supplier_id`, `product_id`, `quantity`, `total_base_price`, `system_fee`, `grand_total`, `status`.
- `Log`: audit log untuk aktivitas sistem/admin.

## Aturan Produk Yang Penting

- Role yang valid di backend adalah `user`, `supplier`, dan `admin`.
- Supplier wajib mengunggah dokumen legalitas saat registrasi normal.
- Status user default adalah `pending`; admin default dibuat sebagai `active`.
- Status order yang tersedia: `pending`, `paid`, `processing`, `shipped`, `completed`, `cancelled`.
- Fee sistem dihitung `total_base_price * 0.03`.
- Produk memakai UUID string sebagai ID.
- Upload dokumen disimpan di `uploads/documents`; upload gambar produk disimpan di `uploads/`.

## Algoritma Yang Sudah Ada

Controller UMKM berisi implementasi algoritma sesuai PRD:

- KMP untuk pencarian nama produk.
- Quick Sort untuk sorting harga produk ascending atau descending.
- Binary Search untuk validasi ID produk sebelum membuat order.

## Status Implementasi Saat Ini

- Backend API dasar sudah tersedia dengan Gin, GORM, JWT, dan middleware role.
- Database memakai AutoMigrate untuk model utama.
- Frontend dashboard sebagian sudah terhubung ke API backend via `fetch`.
- Beberapa data dashboard admin/supplier masih berupa response statis.
- Mock server SmartBank/LogistiKita belum terlihat di workspace.
- Webhook pembayaran dan request logistik belum terlihat di route aktif.
- Ada potensi gap auth yang perlu diperhatikan saat melanjutkan: frontend utama menyimpan sesi di `localStorage.user_session`, tetapi beberapa halaman lama masih mengambil `localStorage.token`; middleware juga perlu konsisten dengan claim JWT yang diterbitkan controller login.
