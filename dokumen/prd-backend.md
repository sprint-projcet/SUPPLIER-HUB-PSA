# SupplierHub - Backend Product Requirements Document (PRD)

## 1. Pendahuluan

Dokumen ini menjelaskan spesifikasi sisi Backend untuk aplikasi **SupplierHub** (Kelompok 4). Backend bertugas memproses logika bisnis seperti manajemen pesanan, validasi stok, perhitungan biaya layanan (fee 3%), serta integrasi dengan sistem eksternal (API Gateway, SmartBank, LogistiKita).

## 2. Arsitektur & Teknologi

- **Bahasa Pemrograman**: Golang (Go 1.20+)
- **Framework**: Standar Pustaka `net/http` atau Framework ringan seperti `Gin` / `Fiber`.
- **Database**: SQL (PostgreSQL atau MySQL)
- **Desain Pola (Pattern)**: _Clean Architecture_ atau _MVC (Model-View-Controller)_ dengan pemisahan struktur folder yang jelas (`controllers/`, `models/`, `routes/`, `middlewares/`, `config/`).
- **Autentikasi**: JWT (JSON Web Token).

## 3. Skema Database (SQL Queries)

Berikut adalah struktur DDL (Data Definition Language) untuk menginisialisasi database `supplierhub_db`.

```sql
-- 1. Tabel Users
CREATE TABLE users (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('UMKM', 'Supplier', 'Admin')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabel Items (Bahan Baku dari Supplier)
CREATE TABLE items (
    id VARCHAR(50) PRIMARY KEY,
    supplier_id VARCHAR(50) REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    price DECIMAL(15, 2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Tabel Orders (Pesanan dari UMKM ke Supplier)
CREATE TABLE orders (
    id VARCHAR(50) PRIMARY KEY,
    umkm_id VARCHAR(50) REFERENCES users(id),
    supplier_id VARCHAR(50) REFERENCES users(id),
    item_id VARCHAR(50) REFERENCES items(id),
    quantity INT NOT NULL,
    total_base_price DECIMAL(15, 2) NOT NULL, -- qty * price
    system_fee DECIMAL(15, 2) NOT NULL DEFAULT 0, -- 3% dari total_base_price
    grand_total DECIMAL(15, 2) NOT NULL, -- total_base_price + system_fee
    status VARCHAR(50) NOT NULL DEFAULT 'Pending', -- Pending, Confirmed, Awaiting Payment, Paid, Shipped
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. Tabel Finance_Logs (Catatan Fee 3%)
CREATE TABLE finance_logs (
    id SERIAL PRIMARY KEY,
    order_id VARCHAR(50) REFERENCES orders(id),
    fee_amount DECIMAL(15, 2) NOT NULL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 4. Contract API (Spesifikasi Endpoint)

Seluruh permintaan API harus menyertakan header `Authorization: Bearer <Token>`. Semua response menggunakan format JSON.

### 4.1. Manajemen Bahan Baku (Supplier)

**`POST /api/items`**

- **Fungsi**: Supplier menambah stok bahan baku.
- **Request Body**:
  ```json
  {
    "name": "Tepung Terigu 1 Kg",
    "price": 12000,
    "stock": 50
  }
  ```
- **Response (200 OK)**:
  ```json
  {
    "status": "success",
    "data": { "item_id": "ITM-001", "name": "Tepung Terigu 1 Kg" }
  }
  ```

### 4.2. Pemesanan Bahan (UMKM)

**`POST /api/orders`**

- **Fungsi**: UMKM memesan barang ke Supplier.
- **Request Body**:
  ```json
  {
    "item_id": "ITM-001",
    "quantity": 10
  }
  ```
- **Response (200 OK)**: Status Order diset _Pending_.

### 4.3. Konfirmasi Pesanan & Kalkulasi Fee 3% (Supplier)

**`POST /api/orders/{order_id}/confirm`**

- **Fungsi**: Supplier mengecek stok, menyetujui, lalu sistem menghitung grand total (termasuk fee 3%) dan melempar ke SmartBank.
- **Logika Internal**:
  - `total_base` = `10 qty` \* `12000` = `120000`
  - `system_fee` = `3%` \* `120000` = `3600`
  - `grand_total` = `123600`
- **Response (200 OK)**: Status Order diset _Awaiting Payment_.
  ```json
  {
    "status": "success",
    "message": "Order Confirmed. Payment requested to SmartBank.",
    "data": {
      "grand_total": 123600,
      "fee_deducted": 3600
    }
  }
  ```

### 4.4. Webhook Pembayaran (Dari Gateway / SmartBank)

**`POST /api/webhook/payment`**

- **Fungsi**: Dipanggil oleh ekosistem luar saat UMKM lunas membayar.
- **Request Body**:
  ```json
  {
    "order_id": "ORD-123",
    "payment_status": "PAID"
  }
  ```
- **Proses**: Mengubah status order menjadi `Paid`, mencatat data ke `finance_logs`, dan memicu API ke LogistiKita.

## 5. Arsitektur Pemrosesan Backend

```mermaid
graph LR
    subgraph Routes
        R_Items[/api/items]
        R_Orders[/api/orders]
        R_Confirm[/api/orders/confirm]
    end

    subgraph Controllers
        C_Item[Item Controller]
        C_Order[Order Controller]
    end

    subgraph Middlewares
        M_Auth[Auth & Role Check]
        M_Log[Request Logger]
    end

    subgraph External Calls
        HTTP_Bank[SmartBank HTTP Client]
        HTTP_Log[LogistiKita HTTP Client]
    end

    M_Log --> M_Auth
    M_Auth --> R_Items
    M_Auth --> R_Orders
    M_Auth --> R_Confirm

    R_Items --> C_Item
    R_Orders --> C_Order
    R_Confirm --> C_Order

    C_Order -->|Calculate 3% Fee| HTTP_Bank
    C_Order -->|If Paid| HTTP_Log
```

## 6. Implementasi Algoritma Inti

Sistem backend mengintegrasikan 3 algoritma utama untuk memastikan performa yang cepat dan efisien dalam pemrosesan data katalog serta pesanan.

### 6.1. Knuth-Morris-Pratt (KMP) - Pencarian Teks

- **Penempatan**: Pada endpoint pencarian produk atau pencarian nama supplier (`GET /api/items?search={query}`).
- **Alur Pengaplikasian**: Ketika _request_ pencarian masuk, backend mengambil memori array data produk. Algoritma KMP akan langsung mencocokkan pola _query_ yang diinput oleh UMKM ke setiap teks nama bahan baku secara efisien tanpa _backtracking_ (menghemat komputasi dengan kompleksitas $O(n+m)$).
- **Cara Implementasi / Pseudo-code**:
  Di level _Service_ Golang, KMP memfilter _array of struct_ sebelum dikembalikan ke HTTP _Response_.
  ```go
  // Pseudo-code KMP di Golang
  func KMPMatch(text string, pattern string) bool {
      // 1. Buat array LPS (Longest Prefix Suffix) dari pattern
      // 2. Lakukan iterasi pada text dan pattern secara sekuensial
      // 3. Jika cocok seluruh pattern, return true
      // 4. Jika ada karakter tidak cocok, geser indeks pola berdasarkan tabel LPS
  }
  // Implementasi:
  // var filteredItems []Item
  // for _, item := range allItems {
  //     if KMPMatch(strings.ToLower(item.Name), query) {
  //         filteredItems = append(filteredItems, item)
  //     }
  // }
  ```

### 6.2. Quick Sort / Merge Sort - Pengurutan Data

- **Penempatan**: Pada mekanisme filter/sorting setelah data difilter oleh KMP (atau secara global) via `GET /api/items?sort_by=price_asc`.
- **Alur Pengaplikasian**: Hasil _array_ produk dari pencarian KMP akan langsung dioper ke dalam fungsi `QuickSort()`. Jika UMKM menekan "Harga Termurah", pivot diatur berdasarkan `integer` harga (_Price_) dan akan diurutkan secara sangat responsif di dalam memori internal server rata-rata $O(n \log n)$.
- **Cara Implementasi / Pseudo-code**:
  Dipanggil pada array `filteredItems` hasil KMP.
  ```go
  // Pseudo-code Quick Sort (Ascending by Price)
  func QuickSortPrice(items []Item, low, high int) {
      if low < high {
          pi := partition(items, low, high)
          QuickSortPrice(items, low, pi-1)
          QuickSortPrice(items, pi+1, high)
      }
  }
  func partition(items []Item, low, high int) int {
      pivot := items[high].Price
      i := low - 1
      for j := low; j < high; j++ {
          if items[j].Price <= pivot {
              i++
              items[i], items[j] = items[j], items[i] // Swap
          }
      }
      items[i+1], items[high] = items[high], items[i+1]
      return i + 1
  }
  ```

### 6.3. Binary Search - Validasi Eksistensi Terurut

- **Penempatan**: Berjalan sebagai _guard clause_ atau validasi ketika UMKM menekan tombol order di endpoint pemesanan (`POST /api/orders`).
- **Alur Pengaplikasian**: Sesuai Aturan No. 6: Validasi wajib. Sebelum backend melangkah ke kalkulasi berat "Biaya Layanan Supplier 3%" dan menembak Gateway SmartBank, backend akan memvalidasi apakah `item_id` benar-benar valid. Backend menggunakan Binary Search yang mencari di daftar ID cache yang sudah tersortir secara sangat cepat $O(\log n)$.
- **Cara Implementasi / Pseudo-code**:
  Backend menarik/menyiapkan array ID produk yang sudah di-_sort_ (misalnya dari Redis Cache/DB index), lalu mencari kecocokan `item_id`.
  ```go
  // Pseudo-code Binary Search untuk Validasi
  func IsItemValid(sortedIDs []string, targetID string) bool {
      low, high := 0, len(sortedIDs)-1
      for low <= high {
          mid := low + (high-low)/2
          if sortedIDs[mid] == targetID {
              return true // Item Valid
          } else if sortedIDs[mid] < targetID {
              low = mid + 1
          } else {
              high = mid - 1
          }
      }
      return false // Item Tidak Ditemukan / Invalid
  }
  ```

---

**Status Dokumen:** ✅ Selesai
