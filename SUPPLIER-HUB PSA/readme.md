# SupplierHub - Product Requirements Document (PRD) Overview

## 1. Pendahuluan

**SupplierHub** (Kelompok 4) adalah aplikasi B2B (Business-to-Business) yang dirancang untuk menjadi jembatan ekosistem antara UMKM dan Supplier bahan baku. Aplikasi ini memfasilitasi pemesanan bahan baku oleh UMKM, manajemen stok dan harga oleh Supplier, serta terintegrasi langsung dengan ekosistem luar seperti sistem pembayaran (SmartBank) dan logistik (LogistiKita) melalui API Gateway.

Dokumentasi ini dibuat untuk merancang pengembangan aplikasi yang berpedoman pada `TugasBesar_Plan.xlsx` dengan aturan pengerjaan dan keuangan yang ketat, termasuk pemotongan _fee_ layanan sebesar 3%.

## 2. Struktur PRD (Product Requirements Document)

Sesuai dengan rencana pengembangan, dokumentasi teknis ini dibagi menjadi beberapa bagian untuk mempermudah pengerjaan secara modular:

1. **`readme.md`** (Dokumen ini) - Overview sistem, Arsitektur Makro, dan Workflow Bisnis.
2. **`prd-frontend.md`** - Spesifikasi UI/UX, routing, state management (HTML, TailwindCSS, Vanilla JS).
3. **`prd-backend.md`** - Spesifikasi arsitektur backend, Contract API, struktur Clean Code Golang, dan Database Relasional (SQL).
4. **`prd-mock-server.md`** - Spesifikasi Mock API Server untuk testing Gateway, Bank, dan Logistik (simulasi ekosistem pihak ketiga).

---

## 3. Workflow Sistem (Alur Bisnis Utama)

Skenario transaksi yang dikelola oleh SupplierHub adalah sebagai berikut:

1. **Pemesanan (UMKM):** UMKM melakukan request pemesanan bahan baku melalui UI SupplierHub.
2. **Konfirmasi (Supplier):** Supplier menerima notifikasi pesanan, lalu mengonfirmasi ketersediaan stok bahan dan menetapkan harga final pesanan.
3. **Kalkulasi Biaya:** Sistem SupplierHub menghitung total pesanan dengan menambahkan _Fee Supplier_ sebesar **3%** (aturan keuangan ekosistem).
4. **Permintaan Pembayaran:** SupplierHub mengirimkan _Payment Request_ secara terpusat ke **SmartBank** melalui **API Gateway**. Aplikasi tidak menahan saldo uang, melainkan meneruskan tagihan.
5. **Penyelesaian Transaksi:** SmartBank memproses pembayaran. Jika berhasil, SmartBank mengirimkan notifikasi status sukses ke SupplierHub (via Gateway).
6. **Pengiriman Logistik:** SupplierHub meng-update status pesanan menjadi Lunas (Paid) dan memicu panggilan API ke **LogistiKita** untuk memulai proses pengiriman dari Supplier ke UMKM.

### Diagram Workflow

```mermaid
sequenceDiagram
    autonumber
    actor UMKM
    actor Supplier
    participant Frontend as Frontend (Web)
    participant Backend as Backend (Golang)
    participant DB as SQL Database
    participant GW as API Gateway
    participant Bank as SmartBank (Mock)
    participant Log as LogistiKita (Mock)

    UMKM->>Frontend: Buat Pesanan Bahan Baku
    Frontend->>Backend: POST /api/order
    Backend->>DB: Simpan Order (Status: Pending)
    Backend-->>Frontend: Respon Order Sukses

    Supplier->>Frontend: Lihat Daftar Pesanan
    Supplier->>Frontend: Konfirmasi Stok & Harga
    Frontend->>Backend: POST /api/order/confirm
    Backend->>Backend: Kalkulasi Total + Fee 3%
    Backend->>DB: Update Status Order (Awaiting Payment)

    Backend->>GW: POST /payment/request (Forward ke SmartBank)
    GW->>Bank: Kirim Tagihan
    Bank-->>GW: Respons (Tagihan Dibuat)
    GW-->>Backend: Respons Sukses
    Backend-->>Frontend: Notifikasi Menunggu Pembayaran

    Note over UMKM, Bank: UMKM melakukan pembayaran di SmartBank...
    Bank->>GW: Webhook: Payment Success
    GW->>Backend: POST /api/webhook/payment
    Backend->>DB: Update Status Order (Paid)

    Backend->>GW: POST /logistic/send (Forward ke LogistiKita)
    GW->>Log: Request Pengiriman Barang
    Log-->>GW: Resi Pengiriman
    GW-->>Backend: Status Pengiriman
    Backend->>DB: Update Status Order (Shipped)
    Backend-->>Frontend: Update Dashboard Status Selesai
```

---

## 4. Arsitektur Sistem

Arsitektur aplikasi akan dipisahkan menjadi komponen Frontend, Backend, dan Mock Server. Backend akan ditulis dalam Golang dengan pola _Clean Architecture_ / MVC. Aplikasi ini terhubung dengan sistem eksternal menggunakan API Gateway sebagai pintu masuk/keluar terpusat.

```mermaid
graph TD
    subgraph "Klien UMKM / Supplier"
        UI[Frontend Web UI<br>HTML, Tailwind, JS]
    end

    subgraph "SupplierHub System"
        subgraph "Backend Golang"
            API_Auth[Auth Service]
            API_Product[Product & Stock Service]
            API_Order[Order Service]
            API_Finance[Finance/Fee Service]
        end
        DB[(Database SQL<br>PostgreSQL/MySQL)]
    end

    subgraph "External Ecosystem (Mocked/Real)"
        GW{API Gateway}
        Bank[SmartBank API]
        Logistik[LogistiKita API]
    end

    %% Relasi Internal
    UI -->|HTTP REST| API_Auth
    UI -->|HTTP REST| API_Product
    UI -->|HTTP REST| API_Order

    API_Auth <--> DB
    API_Product <--> DB
    API_Order <--> DB
    API_Finance <--> DB

    API_Order --> API_Finance

    %% Relasi Eksternal
    API_Order -->|Payment Request| GW
    GW -->|Forward| Bank

    Bank -->|Webhook / Callback| GW
    GW -->|Update Status| API_Order

    API_Order -->|Request Delivery| GW
    GW -->|Forward| Logistik
```

---

## 5. Ringkasan Teknologi

- **Frontend**: HTML5, TailwindCSS, Vanilla JavaScript (Fetch API untuk integrasi Backend).
- **Backend**: Golang (Gin/Fiber/Standar HTTP) dengan arsitektur Clean Code.
- **Database**: Relational Database menggunakan SQL (PostgreSQL atau MySQL).
- **Mock Server**: Node.js/Python/Go sederhana untuk simulasi SmartBank, LogistiKita, dan API Gateway.

---

**Status Dokumen:** ✅ Selesai
_Langkah selanjutnya adalah pembuatan **`prd-frontend.md`**, **`prd-backend.md`**, dan **`prd-mock-server.md`** sesuai alur kerja bertahap._
