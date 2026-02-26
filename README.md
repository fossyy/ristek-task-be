# Ristek Task Backend

> **📄 Catatan Deployment:** Cara deployment menggunakan **Docker Compose** sudah disertakan di dalam **submisi PDF**.

---

## Prasyarat

Pastikan sudah terinstall:

- [Go 1.26+](https://go.dev/dl/) (disarankan memakai versi latest)
- [PostgreSQL](https://www.postgresql.org/download/) (berjalan secara lokal atau remote)

---

## Cara Menjalankan Secara Lokal

### 1. Clone Repository

```bash
git clone https://github.com/fossyy/ristek-task-be
cd ristek-task-be
```

### 2. Buat Database PostgreSQL

Buat database baru di PostgreSQL kamu:

```sql
CREATE DATABASE ristek;
```

### 3. Konfigurasi Environment

Buat file `.env` di root project:

```env
DATABASE_URL=postgresql://<user>:<password>@localhost:5432/<dbname>?sslmode=disable
PORT=8080
```

Contoh:

```env
DATABASE_URL=postgresql://postgres:password@localhost:5432/ristek?sslmode=disable
PORT=8080
```

> **Catatan:** Variabel `ADDRESS` bersifat opsional (default: `0.0.0.0`).

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Jalankan Aplikasi

```bash
go run main.go
```

Aplikasi akan berjalan di `http://localhost:8080`.

> **Migrasi otomatis:** Saat aplikasi pertama kali dijalankan, skema database akan dibuat secara otomatis menggunakan `golang-migrate`. Tidak perlu menjalankan migrasi secara manual.

---

## Dokumentasi API (Swagger)

Setelah aplikasi berjalan, buka browser dan akses:

```
http://localhost:8080/swagger/index.html
```

---

## Environment Variables

| Variable       | Wajib | Default     | Keterangan                            |
|----------------|-------|-------------|---------------------------------------|
| `DATABASE_URL` | ✅    | —           | PostgreSQL connection string          |
| `PORT`         | ❌    | `8080`      | Port HTTP server                      |
| `ADDRESS`      | ❌    | `0.0.0.0`   | Bind address HTTP server              |
