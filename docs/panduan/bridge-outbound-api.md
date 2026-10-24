# Kontrak API bridge: `POST /api/outbound` (Fase D)

Dokumen ini untuk PR pendamping di repo GOWA
`ruangsanadindonesia-art/go-whatsapp-web-multidevice`, folder `integrations/hermes-cs-bridge/`
(PRD `docs/prd/prd-crm-leads-cs-ai.md`, Lampiran B bagian 3).
Sisi Dashboard sudah selesai: klien di `app/services/cs_bridge.go`, worker di `app/services/wa_outbox.go`.

## Kenapa lewat bridge

Pesan yang dikirim lewat API GOWA tidak memicu webhook ke bridge. Kalau Dashboard mengirim langsung dari device `cs-ai`,
CS AI kehilangan konteks saat calon santri membalas. Karena itu semua pesan keluar dari nomor CS lewat bridge (PRD D6).

## Request

```
POST http://172.17.0.1:8650/api/outbound
Authorization: Bearer <OUTBOUND_TOKEN>
Content-Type: application/json

{"chat_id":"6281234567890@s.whatsapp.net","text":"Assalamu'alaikum Kak ...","reference_id":123,"purpose":"followup"}
```

| Field | Isi |
|---|---|
| `chat_id` | JID chat lead (sama dengan `chat_id` di event ingest) |
| `text` | Teks final yang sudah disetujui manusia. Maks 1000 karakter |
| `reference_id` | `lead_followups.id` untuk `purpose=followup`; `0` untuk tujuan lain |
| `purpose` | `followup` atau `payment_verified` (konfirmasi pembayaran) |

`OUTBOUND_TOKEN` di `.env` bridge harus sama dengan `CS_BRIDGE_OUTBOUND_TOKEN` di `.env` Dashboard.
Token ini **berbeda** dari token ingest (arah berlawanan). Minimal 32 byte acak; bandingkan *constant-time*.

## Perilaku wajib

1. Token salah → `401`. Endpoint hanya listen di alamat bridge yang sudah ada (tidak publik).
2. 🔒 Chat sedang dijeda (admin menangani langsung) → `409 {"error":"paused"}`. Dashboard mengulang nanti.
3. 🔒 Teks memuat `[[` → `400 {"error":"marker"}`. Tidak ada marker di pesan outbound.
4. Kirim lewat GOWA device `cs-ai`.
5. Simpan "pesan outbound terakhir" per chat di `state.json` dan sisipkan ke konteks prompt berikutnya, mis.
   `Follow-up terakhir yang dikirim admin: "<teks>" (2026-10-25 11:00 WIB)`, supaya CS AI paham saat lead membalas.
6. Emit event `message.out` ke Dashboard (antrean lokal yang sama dengan Fase A) dengan
   `sender=followup`, `reference_id` dan `purpose` dari request:
   ```json
   {"event_id":"evt_…","type":"message.out","occurred_at":"…","chat_id":"…","phone":"…","wa_name":"…",
    "data":{"wa_message_id":"3EB0…","body":"<teks>","sender":"followup","reference_id":123,"purpose":"followup"}}
   ```
   Dashboard hanya menandai follow-up terkirim bila `purpose` kosong atau `followup`.
7. Balas `200 {"wa_message_id":"3EB0…"}`.

## Cara Dashboard menafsirkan balasan

| Balasan | Perlakuan |
|---|---|
| 2xx + `wa_message_id` | Terkirim |
| 409, 429, 5xx, timeout, koneksi gagal | Sementara: dicoba lagi setelah 1 menit, lalu 5 menit; gagal ke-3 → "Gagal kirim" |
| 400, 401, 403, 404, lainnya | Permanen: langsung "Gagal kirim" (tombol Coba lagi di halaman Follow-up) |

Dashboard mengirim paling banyak satu pesan per menit dan menghormati jeda minimum, batas harian, dan jam kirim dari
pengaturan CRM. Bridge tidak perlu membatasi ulang, tapi boleh menambah pengaman sendiri.

## Test minimal di repo bridge

- token salah → 401
- chat dijeda → 409
- teks dengan `[[FORM]]` → 400
- sukses → pesan terkirim lewat GOWA, event `message.out` (sender `followup`, `reference_id`, `purpose`) masuk antrean, konteks tersisip di prompt berikutnya
