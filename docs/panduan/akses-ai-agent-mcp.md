# Akses AI Agent (MCP) — Tahap 1

Dashboard Ruang Sanad bisa dibaca oleh AI agent (Claude, Hermes, dan client MCP lain) lewat endpoint **`/mcp`**.
Aksesnya **hanya membaca**, mengikuti **role** pemilik token, dan **tidak pernah** mengirim nomor HP, email, voice note, atau domisili.

## Ringkasan teknis

| Bagian | Isi |
|---|---|
| Endpoint | `POST https://<domain>/mcp` (MCP Streamable HTTP, JSON, tanpa SSE, stateless) |
| Login | Header `Authorization: Bearer rsai_…` (token pribadi dari menu **Akses AI**) |
| Hak akses | Role user dibaca ulang dari database di setiap request. Role diturunkan → akses langsung hilang |
| Data kontak | Tidak di-query sama sekali, ditambah sensor otomatis untuk nomor HP/email/link WhatsApp di teks catatan |
| Audit | Setiap pemanggilan tool tercatat di tabel `ai_audit_log` (siapa, tool apa, argumen, berhasil/gagal, durasi). Isi jawaban tidak disimpan |
| Batas | 120 request/menit per token, token maks. 5 aktif per user, masa berlaku 1–365 hari |

Kode utama:

- `app/handlers/mcp.go` — protokol MCP (initialize, tools/list, tools/call, ping)
- `app/handlers/ai_akses.go` + `frontend/src/pages/app/AksesAI.svelte` — halaman buat/cabut token
- `app/services/ai_agent.go`, `ai_agent_tools.go` — daftar tool, pembatasan role, ambang "perlu perhatian"
- `app/services/ai_redact.go` — sensor data kontak
- `app/services/ai_token.go` — token (hanya hash SHA-256 yang disimpan)
- `app/queries/ai_agent.go`, `ai_token.go` — query read-only
- `migrations/0065_create_ai_tokens_table.sql`, `0066_create_ai_audit_log_table.sql`

## Setting saat deploy (sekali saja)

1. Deploy seperti biasa. Migrasi 0065 dan 0066 jalan otomatis saat aplikasi start.
2. Tambahkan di `.env` server (disarankan):
   ```
   APP_URL=https://dashboard.ruangsanad.com
   ```
   Dipakai untuk link balik ke dashboard di jawaban AI. Kalau kosong, alamat diambil dari request.
3. Kalau memakai Cloudflare: pastikan **Bot Fight Mode / WAF** tidak menantang (challenge) request `POST /mcp`.
   AI agent bukan browser, jadi tidak bisa menyelesaikan challenge. Bila perlu buat rule *Skip* untuk path `/mcp`.

## Setting per staf

1. Login dashboard → menu **Administrasi → Akses AI**.
2. Isi nama token (mis. "Claude Desktop laptop"), pilih masa berlaku, klik **Buat token**. Salin token — hanya tampil sekali.
3. Pasang di aplikasi AI (contoh juga tersedia di halaman Akses AI, token sudah terisi):

**Claude Desktop** — Settings → Developer → Edit Config (`claude_desktop_config.json`), butuh Node.js:
```json
{
  "mcpServers": {
    "ruang-sanad": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://dashboard.ruangsanad.com/mcp", "--header", "Authorization:${RS_AUTH}"],
      "env": { "RS_AUTH": "Bearer rsai_TOKEN_ANDA" }
    }
  }
}
```

**Claude Code**:
```
claude mcp add --transport http ruang-sanad https://dashboard.ruangsanad.com/mcp \
  --header "Authorization: Bearer rsai_TOKEN_ANDA"
```

**Hermes Agent** (`~/.hermes/config.yaml`; format bisa berbeda antar versi Hermes):
```yaml
mcp_servers:
  ruang_sanad:
    url: "https://dashboard.ruangsanad.com/mcp"
    headers:
      Authorization: "Bearer rsai_TOKEN_ANDA"
```

**claude.ai (web) dan ChatGPT**: belum bisa di tahap 1. Keduanya meminta login OAuth untuk custom connector → tahap 2.

## Tool dan role

| Tool | Isi | Role |
|---|---|---|
| `ringkasan_kondisi` | Prioritas bulan ini, status santri, program, operasional, riayah, produk, keuangan; untuk super admin juga kelas & guru yang perlu perhatian | super_admin, admin, keuangan |
| `santri_perlu_tindak_lanjut` | Santri aktif berpenanda riayah (+ tunggakan bila role boleh lihat keuangan), dengan alasan & bukti | super_admin, admin_kelas, koordinator_guru |
| `cari_santri` | Cari nama / ID mahasantri | + cs |
| `detail_santri` | Profil, riwayat status; riayah & timeline (role riayah), catatan admin (admin kelas), tagihan (keuangan) | + cs (profil saja) |
| `tren_program` | Per level / tipe / frekuensi / angkatan / produk per bulan | super_admin |
| `rekap_tunggakan` | Piutang, aging, kolektibilitas, piutang per kelas/angkatan, prioritas follow-up | super_admin, admin, keuangan |
| `kelas_perlu_perhatian`, `detail_kelas` | Masalah kelas + bukti | super_admin, admin_kelas, koordinator_guru |
| `guru_perlu_perhatian`, `detail_guru` | Masalah guru + bukti (TSI, riayah, kunjungan, pembinaan/rapat) | super_admin, koordinator_guru |
| `cari_catatan` | Cari teks di catatan dashboard; sumber dibatasi per role | super_admin, admin_kelas, koordinator_guru |
| `daftar_santri` | Daftar lengkap santri + nama, berhalaman, filter bebas (status, tanpa kelas aktif, belum lengkap, kelas, guru, level, tipe, angkatan, tanggal daftar) | super_admin, admin_kelas, koordinator_guru, cs |
| `daftar_kelas` | Semua kelas + metrik (opsional termasuk nonaktif) | super_admin, admin_kelas, koordinator_guru |
| `daftar_guru` | Semua guru + metrik (tanpa kontak) | super_admin, koordinator_guru |
| `daftar_tagihan` | Daftar tagihan per santri, filter status/terlambat/kelas/angkatan/tanggal tagih, berhalaman | super_admin, admin, keuangan |
| `daftar_lead`, `detail_lead`, `laporan_funnel` | CRM leads (calon santri dari WhatsApp CS) tanpa data kontak, transkrip disensor | super_admin, keuangan, cs |
| `simpan_analisis_lead`, `usulkan_followup`, `tambah_label_lead` | **Tulis terbatas** CRM, hanya token dengan izin tulis (dibuat super_admin). Lihat `docs/panduan/crm-ai-agent.md` | super_admin, keuangan, cs |

Setiap prioritas di `ringkasan_kondisi` membawa field `rincian` yang menyebut tool + argumen untuk melihat nama-nama di balik angkanya
(mis. "Santri aktif belum punya kelas aktif" → `daftar_santri {status: aktif, tanpa_kelas_aktif: true}`).

Role `guru` dan `user` belum mendapat akses (data guru dibatasi ke kelasnya sendiri; belum dimodelkan di tahap 1).

## Ambang "perlu perhatian" (bisa diubah)

Diatur di `DefaultAIAmbang` (`app/services/ai_agent.go`). Penanda santri memakai ambang riayah yang sudah ada (`app/services/riayah_santri.go`) supaya AI sama dengan halaman Riayah Santri.

| Ambang | Nilai awal |
|---|---|
| Kehadiran kelas 30 hari minimal | 70% (minimal 5 absensi tercatat) |
| Kelas tanpa pertemuan selesai | 14 hari |
| Porsi santri berpenanda di kelas/guru | ≥ 30% (minimal 2 santri) |
| Cakupan riayah guru minimal | 50% |
| Nilai TSI minimal | 70 ("Perlu Pembinaan") |
| Kehadiran pembinaan + rapat guru (90 hari) | 75% |
| Tunggakan dianggap prioritas | telat ≥ 30 hari |

## Keterbatasan yang perlu diketahui

- **Data kosong ≠ kondisi baik/buruk.** Kalau absensi/pembayaran tidak diinput, AI membacanya sebagai penurunan.
- **Tren program memakai level/tipe santri saat ini**, bukan riwayat perpindahan.
- **Pencarian catatan** memakai pencocokan kata (semua kata harus ada), bukan pencarian makna.
- **Rekomendasi AI adalah interpretasi.** Hasil tool selalu menyertakan link dashboard untuk dicek.
- Nama santri/guru tetap terkirim ke penyedia AI (Anthropic/OpenAI/dll.) — pastikan sesuai kebijakan privasi lembaga (UU PDP).

## Cabut akses

- Staf: menu **Akses AI** → **Cabut** pada token.
- Super admin: ubah role user (mis. ke `user`) → semua tokennya langsung tidak berlaku.
