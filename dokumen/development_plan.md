# Rencana Pengembangan (Development Plan) SupplierHub

Dokumen ini berisi panduan dan urutan pengerjaan (_roadmap_) untuk melanjutkan proyek SupplierHub berdasarkan PRD yang sudah disetujui. Dengan menggunakan panduan ini, Anda dapat memberikan instruksi bertahap (_step-by-step_) kepada AI agar pengerjaan lebih terstruktur dan minim error.

---

## Fase 1: Persiapan dan Koneksi Database (Backend)
**Fokus:** Menyiapkan skema database dan koneksi ke aplikasi Golang.
*   [ ] **1.1 Setup Database:** Mengatur file `config/database.go` untuk terhubung ke database (PostgreSQL/MySQL).
*   [ ] **1.2 Pembuatan Model/Struct:** Membuat definisi struct Golang di dalam folder `models/` untuk tabel `Users`, `Items`, `Orders`, dan `Finance_Logs` sesuai skema di PRD.
*   [ ] **1.3 Auto-Migrasi/DDL:** Membuat script untuk menjalankan DDL (migrasi tabel) saat aplikasi pertama kali dijalankan.

**💡 Prompt yang bisa Anda gunakan:**
> _"Tolong kerjakan Fase 1. Buatkan file koneksi database di `config/` dan buatkan model struct di `models/` sesuai dengan skema SQL yang ada di `prd-backend.md`."_

--6

## Fase 2: Implementasi Endpoint API (Backend Core)
**Fokus:** Membuat fungsionalitas CRUD dasar untuk Sistem Manajemen dan Transaksi.
*   [ ] **2.1 Autentikasi & Middleware:** Memastikan JWT Auth dan Middleware Role (UMKM, Supplier, Admin) di `middlewares/` sudah siap digunakan oleh routes.
*   [ ] **2.2 API Manajemen Bahan Baku:** Membuat `controllers/item_controller.go` untuk `POST /api/items` (tambah stok), `GET /api/items` (lihat stok), dll.
*   [ ] **2.3 API Order & Konfirmasi:** Membuat `controllers/order_controller.go` untuk `POST /api/orders` (UMKM membuat pesanan) dan `POST /api/orders/{order_id}/confirm` (Supplier menyetujui, kalkulasi fee 3%, grand total).
*   [ ] **2.4 Setup Routes:** Mendaftarkan semua endpoint di `routes/`.

**💡 Prompt yang bisa Anda gunakan:**
> _"Tolong kerjakan Fase 2. Buatkan controller untuk manajemen bahan baku (`/api/items`) dan pemesanan (`/api/orders`), lalu daftarkan di dalam folder `routes/`."_

---

## Fase 3: Integrasi Algoritma Inti (Backend Logic)
**Fokus:** Memasukkan logika algoritma khusus ke dalam Controller/Service sesuai PRD.
*   [ ] **3.1 Knuth-Morris-Pratt (KMP):** Menerapkan algoritma KMP untuk fitur pencarian nama bahan baku di endpoint `GET /api/items`.
*   [ ] **3.2 Quick Sort:** Mengimplementasikan sorting (berdasarkan harga/stok) menggunakan algoritma Quick Sort.
*   [ ] **3.3 Binary Search:** Menerapkan validasi ketersediaan `item_id` sebelum pemrosesan `POST /api/orders` menggunakan Binary Search pada ID yang disortir.

**💡 Prompt yang bisa Anda gunakan:**
> _"Tolong kerjakan Fase 3. Implementasikan fungsi KMP untuk fitur pencarian bahan baku dan Quick Sort untuk mengurutkan harga pada endpoint item."_

---

## Fase 4: Pembuatan Mock Server (Sistem Eksternal)
**Fokus:** Membuat server tiruan untuk SmartBank dan LogistiKita sesuai `prd-mock-server.md`.
*   [ ] **4.1 Inisialisasi Mock Server:** Membuat direktori `mock-server` dan inisialisasi project Express.js (Node.js).
*   [ ] **4.2 Mock SmartBank:** Membuat endpoint `POST /mock/smartbank/pay` yang merespon delay 10 detik lalu menembak Webhook.
*   [ ] **4.3 Mock LogistiKita:** Membuat endpoint `POST /mock/logistikita/send` yang membalas resi dummy.
*   [ ] **4.4 API Webhook Payment:** Menyelesaikan endpoint `POST /api/webhook/payment` di backend Golang untuk menerima panggilan dari mock server.

**💡 Prompt yang bisa Anda gunakan:**
> _"Tolong kerjakan Fase 4. Buatkan folder baru `mock-server` menggunakan Express.js yang isinya endpoint SmartBank dan LogistiKita sesuai PRD Mock Server."_

---

## Fase 5: Integrasi API dengan Frontend
**Fokus:** Menyambungkan UI (HTML/Vanilla JS) dengan Backend Golang dan Mock Server.
*   [ ] **5.1 Fetch API Manajemen Stok:** Menghubungkan form di `supplier_produk_saya.html` dengan `POST /api/items`.
*   [ ] **5.2 Fetch API Order:** Menghubungkan form pesan UMKM di `umkm.html` / `umkm_wishlist.html` ke backend `POST /api/orders`.
*   [ ] **5.3 Fetch API Konfirmasi & Bayar:** Menghubungkan fungsi setuju pesanan di `supplier_daftar_pesanan.html` ke `POST /api/orders/{order_id}/confirm`.
*   [ ] **5.4 Indikator Status & Alert:** Mengatur tampilan Loading Skeleton, Alert, atau Toast untuk memperjelas _User Experience_.

**💡 Prompt yang bisa Anda gunakan:**
> _"Tolong kerjakan Fase 5. Buatkan file JS untuk halaman `supplier_produk_saya.html` agar bisa mengambil dan mengirim data (fetch API) ke backend Golang."_

---

## Fase 6: Finalisasi dan Pengujian End-to-End
**Fokus:** Merapikan dan menguji seluruh flow ekosistem (Testing).
*   [ ] **6.1 Testing Skenario Lengkap:** Testing login UMKM -> pesan barang -> Supplier konfirmasi -> Bank simulasi webhook 10 detik -> UMKM cek status lunas -> Pengiriman dikonfirmasi.
*   [ ] **6.2 Perbaikan Bug (Bugfixing):** Merapikan error handling dan validasi respons.
*   [ ] **6.3 Penulisan Dokumentasi Akhir / README:** Memperbarui petunjuk instalasi dan cara menjalankan _Backend_, _Frontend_, dan _Mock Server_ secara bersamaan.

**💡 Prompt yang bisa Anda gunakan:**
> _"Mari kita lakukan Fase 6. Tolong uji coba integrasi end-to-end dari frontend UMKM order barang sampai webhook pembayaran selesai, lalu bantu saya fix jika ada error."_
