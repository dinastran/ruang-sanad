# Follow-up Leads terjadwal (Fase D)

Follow-up Script 30 Hari dikirim lewat nomor CS **hanya setelah disetujui manusia**. Dasar: PRD `docs/prd/prd-crm-leads-cs-ai.md` bagian 10.

```
AI (job harian) / CS ──► usulan (menunggu persetujuan)
CS / super_admin di /app/leads/followups ──► disetujui
worker Dashboard (tiap 1 menit) ──► antrean wa_outbox ──► bridge POST /api/outbound ──► GOWA cs-ai
bridge ──► event message.out (sender followup) ──► transkrip lead
```

## Siapa boleh apa

| Aksi | cs | keuangan | super_admin |
|---|---|---|---|
| Buat follow-up manual di halaman lead | ✅ | ✅ | ✅ |
| Buka antrean, setujui, edit, ubah jadwal, batalkan, Coba lagi | ✅ | ❌ | ✅ |
| Saklar, jam kirim, jeda, batas harian, label uji, template konfirmasi | ❌ | ❌ | ✅ (Integrasi WhatsApp) |

## Aturan yang dicek (saat diusulkan, saat disetujui, dan tepat sebelum dikirim)

Daftar lengkap di `docs/panduan/crm-ai-agent.md`. Tambahan saat kirim:

| Kondisi saat dikirim | Hasil |
|---|---|
| Saklar follow-up mati | Menunggu (tidak dikirim, tidak dibatalkan) |
| Di luar jam kirim | Menunggu sampai jam kirim berikutnya |
| Batas harian tercapai / jeda minimum belum lewat | Menunggu |
| Lead membalas | Follow-up aktif dibatalkan otomatis saat pesan masuk |
| Lead minta berhenti / jangan hubungi | Dibatalkan |
| Lead hilang / sudah jadi santri | Dilewati |
| Sudah ada follow-up terkirim ke lead ini hari ini | Ditunda ke jam kirim besok |
| Label uji diisi dan lead tidak berlabel itu | Dilewati |
| Bridge gagal | Dicoba lagi 1 dan 5 menit kemudian; gagal ke-3 → **Gagal kirim** (tombol Coba lagi) |
| Server mati saat mengirim | Ditandai gagal dengan pesan "status tidak pasti"; cek WA CS sebelum Coba lagi |

## Menyalakan (super admin)

1. Pastikan PR bridge (`docs/panduan/bridge-outbound-api.md`) sudah terpasang.
2. Isi `.env` Dashboard: `CS_BRIDGE_URL=http://172.17.0.1:8650` dan `CS_BRIDGE_OUTBOUND_TOKEN=<32+ byte acak>`
   (sama dengan `OUTBOUND_TOKEN` di bridge). Restart Dashboard.
3. Uji bertahap (PRD 15.2): di **Integrasi WhatsApp**, isi "Uji bertahap: hanya kirim ke lead berlabel" dengan `uji-followup`
   (label ini dibuat oleh migrasi 0080), lalu pasang label itu di 3–5 lead.
4. Centang **Kirim follow-up yang sudah disetujui**. Default mati.
5. Setujui satu follow-up untuk lead berlabel uji, tunggu jadwalnya, cek: terkirim di WA CS, muncul di transkrip lead
   sebagai "Follow-up", dan balasan lead dijawab CS AI dengan konteks follow-up.
6. Setelah aman, kosongkan label uji.

Default: jam 08:00–20:00 WIB, jeda 45 detik, maks. 40 per hari. Worker mengirim paling banyak 1 pesan per menit.
🔒 Kalau opt-out atau keluhan naik, turunkan batas harian dan evaluasi template (PRD 13).

## Konfirmasi pembayaran (PRD 6.3)

Isi **Template konfirmasi pembayaran** di Integrasi WhatsApp, mis.
`Alhamdulillah, pembayaran [NOMINAL] sudah kami terima, Kak [NAMA]. Admin kelas akan menghubungi untuk penempatan kelas.`
Saat verifikasi, CS bisa mencentang "Kirim pesan konfirmasi" (default tidak dicentang). Konfirmasi lewat antrean dan bridge yang sama,
**tidak** terpengaruh saklar follow-up, jam kirim, dan batas harian, tapi tetap menghormati jeda minimum.
