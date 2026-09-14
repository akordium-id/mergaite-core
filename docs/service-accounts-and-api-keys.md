# Machine-to-Machine (M2M) API Keys & Service Accounts

Dokumentasi arsitektur dan panduan penggunaan **Machine-to-Machine (M2M) Authentication** di platform **Mergiate Core**.

---

## 1. Arsitektur & Model Entitas (Opsi B: Service Account + API Keys)

Sistem M2M dirancang untuk integrasi sistem-ke-sistem (seperti worker IoT/GPS telematics, daemon sinkronisasi ERP, webhook emitter, atau microservice eksternal) tanpa bergantung pada kredensial interaktif user (email/password).

```mermaid
graph TD
    subgraph "External Integrations"
        Daemon[GPS / Telematics Daemon]
        Connector[E-Commerce Sync Worker]
    end

    subgraph "Mergiate Core Delivery Layer"
        MW[Unified Auth Middleware]
        KeyVal[API Key Validator]
        JWTVal[User JWT TokenManager]
    end

    subgraph "Identity & Access Domain"
        SA[Service Account Aggregate]
        AK1[API Key 1 (Active)]
        AK2[API Key 2 (Active - Rotated)]
        Role[RBAC Role]
    end

    Daemon -->|X-API-Key: mrg_live_...| MW
    Connector -->|Authorization: Bearer mrg_live_...| MW

    MW -->|mrg_live_...| KeyVal
    KeyVal --> SA
    SA --> AK1
    SA --> AK2
    SA --> Role
```

### Entitas Utama
1. **`ServiceAccount`** (`service_accounts`):
   - Merepresentasikan identitas mesin/daemon dalam suatu tenant.
   - Dapat di-assign ke suatu `role_id` (mewarisi izin RBAC) atau memiliki nama dan deskripsi.
2. **`APIKey`** (`api_keys`):
   - Kredensial kriptografi milik Service Account.
   - Satu Service Account dapat memiliki beberapa API Key aktif untuk mendukung **zero-downtime key rotation**.
   - Menyimpan hash SHA-256 (`key_hash`) dan display prefix (`key_prefix`, contoh: `mrg_live_41eadb85...`). Plaintext secret key **hanya ditampilkan 1 kali** saat dibuat.
   - Mendukung pembatasan IP (`ip_allowlist` CIDR/IP) dan masa berlaku (`expires_at`).
   - Melacak riwayat penggunaan (`last_used_at`, `last_used_ip`).

---

## 2. Format & Keamanan Kunci

- **Format**: `mrg_live_<64_hex_chars>` (256 bits CSPRNG entropy).
- **Prefix**: `mrg_live_` memudahkan routing middleware dan scanning token leak di tools seperti GitGuardian / GitHub Secret Scanning.
- **Hashing**: SHA-256 hex digest di database. Tidak ada plaintext secret yang tersimpan di server.
- **Display Prefix**: 17 karakter pertama (`mrg_live_` + 8 hex char) untuk identifikasi di dashboard.

---

## 3. Cara Menggunakan M2M API Key

Klien eksternal dapat mengirimkan API key melalui salah satu dari dua cara:

### Cara A: Header `X-API-Key`
```bash
curl -X GET http://localhost:8088/api/v1/documents \
  -H "X-API-Key: mrg_live_41eadb851e073a830b18d66e5af2faf923a61112a1950ef07009df22ea7c6764"
```

### Cara B: Header `Authorization: Bearer`
```bash
curl -X GET http://localhost:8088/api/v1/documents \
  -H "Authorization: Bearer mrg_live_41eadb851e073a830b18d66e5af2faf923a61112a1950ef07009df22ea7c6764"
```

> [!NOTE]
> Header `X-Tenant-ID` **tidak wajib** disertakan jika menggunakan API key, karena tenant context otomatis diinjeksi oleh middleware dari data Service Account terkait.

---

## 4. REST Endpoints

Seluruh endpoint berikut membutuhkan izin `service_account:*`, `api_key:*`, atau `iam:manage` / `*`.

| Method | Path | Deskripsi |
|---|---|---|
| `POST` | `/api/v1/service-accounts` | Membuat Service Account baru |
| `GET` | `/api/v1/service-accounts` | Daftar Service Account di tenant aktif |
| `GET` | `/api/v1/service-accounts/{id}` | Detail Service Account |
| `PUT` | `/api/v1/service-accounts/{id}` | Update nama, deskripsi, role, atau status |
| `DELETE` | `/api/v1/service-accounts/{id}` | Hapus Service Account & cabut seluruh key-nya |
| `POST` | `/api/v1/service-accounts/{id}/keys` | Generate API Key baru (**one-time display `secret_key`**) |
| `GET` | `/api/v1/service-accounts/{id}/keys` | Daftar API Key aktif/revoked (masked) |
| `DELETE` | `/api/v1/service-accounts/{id}/keys/{keyId}` | Cabut / revoke API key secara instan |

---

## 5. Workflow: Zero-Downtime Key Rotation

1. **Generate Key Baru**:
   ```bash
   POST /api/v1/service-accounts/{sa_id}/keys
   ```
   Dapatkan `secret_key` baru (misal: `mrg_live_new_...`). Pada titik ini, key lama dan key baru sama-sama aktif.
2. **Deploy Key Baru** ke server/daemon target (update environment variable atau secrets manager).
3. **Verifikasi** bahwa traffic baru sudah masuk dengan key baru.
4. **Revoke Key Lama**:
   ```bash
   DELETE /api/v1/service-accounts/{sa_id}/keys/{old_key_id}
   ```
   Key lama seketika ditolak dengan `401 Unauthorized`. Rotasi selesai tanpa downtime 1 detik pun!
