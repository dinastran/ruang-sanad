# PRD — Integrasi WhatsApp Dashboard Ruang Sanad, GoWA MCP, Claude, dan Hermes

| Metadata | Keterangan |
|---|---|
| Status | **PROPOSED — READY FOR REVIEW**, belum implementasi |
| Tanggal | 28 Oktober 2026 |
| Owner dokumen | Dashboard Ruang Sanad — dinastran/ruang-sanad |
| Repo pendamping | ruangsanadindonesia-art/go-whatsapp-web-multidevice |
| PRD implementasi GoWA | [GoWA MCP Agent Gateway — PR #14 (draft)](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/pull/14) |
| UI GoWA saat audit | aldinokemal/gowa-ui (separate release asset) |
| Baseline Dashboard main | 65fc33e79e1819fef01a495c2fcfd98bd272dc0f |
| Baseline GoWA main | 3031d9a4163f5013963b7d52e8e442f65bd48890 |
| Evidence | Targeted source review, **bukan** verifikasi runtime |
| Dokumen rujukan | [PRD CRM CS AI existing](./prd-crm-leads-cs-ai.md), [MCP Dashboard](../panduan/akses-ai-agent-mcp.md), [kontrak CS Bridge](../panduan/bridge-outbound-api.md), [pilot Hermes](../hermes-auto-followup-pilot.md) |

**Status istilah:** **EXISTING CODE** = ditemukan di branch main saat audit. **EXISTING PRD** = direncanakan pada dokumen lama, tidak otomatis sudah berjalan. **PROPOSED** = disyaratkan oleh dokumen ini tetapi belum terbukti diimplementasikan. Dokumen ini sendiri tidak menyetujui perubahan data, permission produksi, merge, ataupun deployment.

## 1. Ringkasan dan keputusan desain

Ruang Sanad membutuhkan integrasi WhatsApp yang mendukung dua jalur berbeda tanpa bertabrakan.

1. **Dashboard-driven business messaging:** follow-up calon santri oleh CS, tindak lanjut tagihan Keuangan, komunikasi kelas oleh Admin Kelas, dan komunikasi guru oleh Koordinator Guru. Dashboard bertanggung jawab atas identitas bisnis, role, aturan validitas, eligibility, approval, scheduling bisnis, serta status proses.
2. **Direct AI-agent messaging:** Claude Web/Desktop/Code dan Hermes terhubung ke MCP GoWA untuk melakukan operasi WhatsApp secara natural language, termasuk membaca dan mengirim pesan teks, image/video/document, grup, dan scheduling. GoWA bertanggung jawab atas identitas agent, permission per device/action/target, approval grup, media transport, serta audit teknis.

**Source of truth:** Dashboard untuk keputusan dan state bisnis; GoWA untuk transport WhatsApp dan state direct-send/scheduled send GoWA; Hermes CS Bridge untuk konteks percakapan CS AI.

**Aturan yang tidak boleh dilanggar:** seluruh pesan dari device CS yang termasuk workflow lead/closing melewati Hermes CS Bridge. Direct MCP tidak boleh menjadi jalan pintas yang melewati approval, opt-out, dan aturan follow-up CRM.

**Semua device tersedia untuk sistem** berarti setiap device dapat dipetakan kepada client berizin, **bukan** setiap agent bisa mengendalikan semua device tanpa pembatasan.

## 2. Fakta existing dan gap

| Area | Bukti | Kondisi |
|---|---|---|
| Menu Dashboard | [IntegrasiWhatsApp.svelte](../../frontend/src/pages/app/IntegrasiWhatsApp.svelte), [handler](../../app/handlers/integrasi_whatsapp.go) | **EXISTING CODE**. Super Admin memantau bridge/GoWA dan mengatur pemetaan paket, saklar, jendela waktu, jeda, cap, label uji, template pembayaran |
| Dashboard worker | [wa_outbox.go](../../app/services/wa_outbox.go) | **EXISTING CODE**. Worker saat ini mengirim channel bridge_cs dengan guard, claim, retry dan failed/uncertain handling |
| Bridge client | [cs_bridge.go](../../app/services/cs_bridge.go) | **EXISTING CODE**. POST /api/outbound, personal text, purpose followup/payment_verified, response WA message ID |
| MCP Dashboard | [panduan Akses AI](../panduan/akses-ai-agent-mcp.md) | **EXISTING CODE/guide**. Role-aware, data kontak teredaksi, token tersendiri; dapat mengusulkan follow-up CRM dengan permission terbatas |
| Finance/Admin/Koordinator ke GoWA | [PRD CRM](./prd-crm-leads-cs-ai.md) | **EXISTING PRD**, channel gowa:<device> direncanakan; bukan bukti business worker sudah mengirim |
| MCP GoWA | [MCP tools](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/blob/main/src/ui/mcp/server.go) | **EXISTING CODE**. Enam tool konsolidasi: whatsapp_send, whatsapp_schedule, whatsapp_message, whatsapp_chat, whatsapp_group, whatsapp_app |
| Transport media | [MCP send](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/blob/main/src/ui/mcp/send.go) | **EXISTING CODE**. Text, image/video/audio/document dan format lain; media MCP memakai URL yang diambil server |
| OAuth MCP GoWA | [docs OAuth](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/blob/main/docs/mcp-oauth.md) dan [credential validator](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/blob/main/src/cmd/mcp_oauth.go) | **EXISTING CODE**, OAuth disabled by default. Akun yang dibatasi berdasarkan device ditolak mengakses MCP |
| Scheduler GoWA | [REST schedules](https://github.com/ruangsanadindonesia-art/go-whatsapp-web-multidevice/blob/main/src/ui/rest/schedule.go) | **EXISTING CODE**. List/get/pause/resume/cancel; shared usecase dengan MCP |
| UI GoWA | [Chats](https://github.com/aldinokemal/gowa-ui/blob/main/src/pages/chats.tsx), [Scheduled](https://github.com/aldinokemal/gowa-ui/blob/main/src/pages/scheduled.tsx) | **EXISTING SOURCE**. Implementasi UI berada di repo/release terpisah; versi produksi **NOT VERIFIED** |

### Keputusan existing yang tetap berlaku

- **D5:** Dashboard memakai akun API GoWA yang dikunci ke device non-CS lewat APP_BASIC_AUTH_DEVICES.
- **D6:** Semua pesan keluar dari nomor/device CS AI untuk proses lead/closing melalui CS Bridge.
- **D7:** Agent menganalisis data CRM dan mengusulkan follow-up lewat MCP Dashboard, bukan langsung memakai MCP GoWA untuk mengirim follow-up lead.
- Follow-up lead CS dibuat pending_approval, disetujui CS/Super Admin, lalu dikirim lewat wa_outbox → bridge → GoWA.
- MCP Dashboard tidak membocorkan phone, email, atau wa_chat_id pada DTO analisis lead.

Keputusan D7 tetap mengikat **workflow CRM**. Direct MCP GoWA diperbolehkan untuk kebutuhan komunikasi non-CRM yang lolos policy; dokumen baru tidak mencabut D7.

## 3. Objective, metrik, dan non-goals

### Goals

- Operator bisnis dapat menyiapkan komunikasi WhatsApp dari record Dashboard yang sah.
- Claude/Hermes dapat memakai seluruh kategori tool GoWA sesuai permission granular.
- Pesan personal operasional dapat dikirim otomatis hanya bila non-business-flow, client/device/target diizinkan, dan policy lolos.
- Pengiriman grup dari agent selalu memerlukan **persetujuan manusia** sebelum dikirim.
- Pengiriman image/video/document dari URL ataupun attachment AI didukung melalui alur yang aman.
- Ada satu owner scheduler per pesan, tidak ada duplikasi dispatch antar-sistem.
- Chats/Scheduled GoWA dan audit pengiriman dapat ditelusuri; workflow CS tetap stabil.

### Metrics MVP

- 0 unauthorized device/action sends pada negative tests.
- 0 direct-send bypass ke device CS AI untuk workflow lead.
- 0 group send sebelum approval.
- 0 duplicate send pada idempotency/replay tests.
- 100% request test punya correlation ID, status, actor dan device yang dapat diaudit.
- Setiap pesan business test memiliki kaitan ke record bisnis dan message_id WA setelah sukses.

### Non-goals

Tidak mengganti engine WhatsApp, tidak mengganti CS Bridge, tidak membuat GoWA menjadi database tagihan/santri, tidak memberi full PII pada MCP Dashboard, tidak mengaktifkan mass broadcast tanpa aturan opt-in, tidak mengizinkan AI approve sendiri, dan tidak mengerjakan production deployment di PRD ini.

## 4. Aktor dan permission

| Aktor | Scope bisnis | Aksi | Batas penting |
|---|---|---|---|
| Owner / Super Admin | Integrasi lintas fungsi | Atur mapping device, policy, review audit, approval administratif | Semua action tetap diaudit; tidak boleh bypass workflow CS tanpa desain terpisah |
| Keuangan | Tagihan santri | Review/kirim/jadwalkan pengingat pembayaran dari record sah | Device keuangan; harus periksa pembayaran terbaru |
| Admin Kelas | Kelas, santri, pertemuan | Pengingat/perubahan jadwal/komunikasi layanan | Device admin-kelas; membership & status kelas diperiksa |
| Koordinator Guru | Guru, pertemuan | Komunikasi terkait guru dan tindak lanjut pertemuan | Device koord-guru; cek data dan ownership |
| CS / Super Admin | Lead, closing | Approve follow-up pada Dashboard | Semua send memakai CS Bridge |
| Claude operator | Direct WhatsApp ops | Baca/kirim/schedule sesuai izin | Tanpa akses lintas device; tidak mengubah aturan bisnis |
| Hermes operator/cron | Direct WhatsApp ops dan workflow terdelegasi | Sama dengan scope yang diizinkan | Profile non-CS terpisah; cron tidak mendapat approval otomatis |
| Hermes CS AI | Lead AI via bridge | Respon dan konteks CS existing | Device CS AI tidak dapat dieksekusi bebas oleh MCP operator |

### Policy granular wajib

Evaluasi pada **setiap** operasi: authenticated principal + client + target device + tool + sub-action + target recipient/group + purpose + schedule owner + risk level.

- Device permission tidak boleh meningkat dengan mengubah device_id atau header.
- Per-action server enforcement diperlukan: whatsapp_group.info tidak setara dengan remove_participants, whatsapp_app.status tidak setara dengan logout.
- Deny by default, least privilege, revoke segera, audit lengkap.
- Akses source WhatsApp chat/contacts lewat GoWA MCP lebih sensitif dibanding redacted Dashboard MCP. Pisahkan permission READ_CHAT, READ_CONTACT, SEND_PERSONAL, SEND_GROUP, MANAGE_GROUP, MANAGE_DEVICE dan MANAGE_SCHEDULE.
- Akun/device CS Bridge tidak boleh diberikan direct-send melalui token general MCP.
- Login manusia untuk Claude dapat memakai OAuth MCP; Hermes mendapat identity/profile terpisah. Akun dengan APP_BASIC_AUTH_DEVICES saat ini tidak boleh langsung dilepas pembatasannya agar MCP berfungsi.


## 5. Arsitektur dan routing definitif

    Dashboard UI / Dashboard MCP
               |
       Dashboard business service
       [role, eligibility, approval]
               |
          wa_outbox (business)
               |
       purpose/device router
          /          \
       CS lead      Non-CS business
          |              |
      CS Bridge       GoWA REST
          |              |
          +-------+------+
                  |
             GoWA usecases   <--- GoWA MCP policy <--- Claude / Hermes
                  |                   |
           WhatsApp devices       approval grup
                  |
            Personal / Group

### Routing matrix

| Kasus | Owner aturan | Owner schedule | Jalur | Approval |
|---|---|---|---|---|
| Follow-up lead CS | Dashboard CRM | Dashboard wa_outbox | Dashboard → CS Bridge → GoWA cs-ai | CS/Super Admin existing |
| Konfirmasi pembayaran lead CS | Dashboard CRM | Dashboard wa_outbox | Dashboard → CS Bridge → GoWA cs-ai | Checkbox/manual step existing; tidak disamakan dengan follow-up toggle |
| Tindak lanjut tagihan santri | Dashboard Keuangan | Dashboard wa_outbox | Dashboard → REST GoWA keuangan | **Default review staf** sampai automasi diapprove |
| Komunikasi santri/kelas | Dashboard Admin Kelas | Dashboard wa_outbox | Dashboard → REST GoWA admin-kelas | **Default review staf** |
| Komunikasi guru | Dashboard Koordinator Guru | Dashboard wa_outbox | Dashboard → REST GoWA koord-guru | **Default review staf** |
| Claude/Hermes manual ke pribadi non-CRM | GoWA | GoWA jika scheduled | MCP GoWA → non-CS device | Boleh otomatis jika scope/policy lolos |
| Claude/Hermes manual ke grup | GoWA | GoWA | MCP → approval durable → GoWA | **Wajib approval manusia** |
| AI ingin follow-up lead dari data Dashboard | Dashboard CRM | Dashboard | Dashboard MCP mengusulkan → approval CS → bridge | Tidak boleh raw MCP send |
| AI ingin kirim tagihan dari data Dashboard | Dashboard Keuangan | Dashboard | Dashboard MCP business intent → role review → GoWA | Default review staf |

**Routing tidak boleh ditentukan hanya dari format nomor.** Permintaan terkait lead_ref, tagihan, santri, kelas, atau guru merupakan business intent Dashboard sekalipun agent mengetahui nomor WhatsApp penerima.

## 6. Alur bisnis end-to-end

### FLOW-01: Finance mengirim reminder tagihan

Actor: Keuangan. Trigger: tagihan yang memerlukan tindak lanjut di Dashboard.

1. Operator memilih record tagihan/santri yang memang diizinkan (bukan memasukkan nomor raw dari model).
2. Server membaca status terbaru, nominal tagihan yang benar, tanggal jatuh tempo dan aturan kontak/opt-out.
3. Sistem menghasilkan draft pesan dan menampilkan preview, device keuangan dan penerima masked.
4. Staf melakukan review/approval (default desain) dan menjadwalkan atau kirim.
5. Dashboard menyimpan intent pada wa_outbox channel gowa:keuangan; worker memeriksa ulang apakah tagihan masih perlu di-follow-up.
6. Worker memanggil GoWA REST dengan scoped service credential; GoWA memberi WA message ID dan audit; Dashboard menyimpan hasil.

Failure penting: tagihan sudah lunas/nomor tidak valid/device offline/pesan status tidak pasti. Tagihan lunas tidak boleh menerima reminder. Timeout ambigu masuk status UNCERTAIN, bukan retry otomatis.

Success: satu pesan WA terkirim dari device Finance dengan relasi ke tagihan, waktu, dan status pada Dashboard.

### FLOW-02: Admin Kelas mengingatkan atau mengubah jadwal kelas

Actor: Admin Kelas. Trigger: pertemuan/jadwal/santri yang berada dalam scope kerja.

- Server memvalidasi record kelas, membership, status pertemuan dan waktu terbaru.
- Preview menggunakan data current, bukan narasi AI yang mungkin outdated.
- Setelah review sesuai policy, outbox mengirim via device admin-kelas.
- Perubahan/pembatalan kelas sebelum due membatalkan atau memperbarui draft yang sudah usang, dan memerlukan review baru.
- Idempotency mencegah reminder berulang untuk event yang sama.

### FLOW-03: Koordinator Guru menindaklanjuti guru

Actor: Koordinator Guru. Trigger: pembatalan kelas, perubahan jadwal, atau data absensi guru.

- Server memeriksa hubungan guru-pertemuan dan apakah tindakan masih relevan.
- AI boleh menyusun draft; temuan AI bukan bukti pelanggaran guru.
- Review staf default, lalu kirim dari device koord-guru.
- Konteks kejadian dan hasil pengiriman muncul di alur operasional yang bersangkutan.

### FLOW-04: Claude/Hermes direct personal send

Actor: Claude/Hermes terautentikasi. Trigger: perintah eksplisit mengirim pesan/media ke nomor pribadi untuk komunikasi umum non-CRM.

- Pilih tool dan device, resolve target, validasi target, scope dan media.
- Jika request menyentuh record bisnis Dashboard atau device CS Bridge, **jangan** dispatch direct; arahkan ke business workflow yang sah.
- Jika lolos, kirim melalui GoWA usecase, catat request ID, WA message ID, origin, principal dan status.
- Hasil sukses adalah accepted/sent-to-WhatsApp, bukan otomatis delivered/read.

### FLOW-05: Claude/Hermes direct group send

Actor: Claude/Hermes. Trigger: permintaan mengirim pesan/media ke grup.

- Resolve group JID, membership dan permitted device.
- Freeze payload dan penerima; simpan PENDING_APPROVAL.
- Approver manusia berwenang melihat device, grup, caption, media, schedule dan agent peminta.
- Reject/expire tidak mengirim. Approved hanya berlaku untuk immutable payload/target tertentu.
- Dispatcher recheck group membership, principal policy, media validity, approval dan idempotency lalu mengirim satu kali.
- Semua outcome dapat dilihat dari halaman Approvals dan AI Activity.

### FLOW-06: AI meminta scheduled group message

- Direct scheduled personal dimiliki GoWA; direct scheduled grup juga dimiliki GoWA.
- Buat schedule dengan target dan payload namun **jangan** auto-send ke grup tanpa approval yang berlaku untuk occurrence itu.
- Recurrence tidak mengubah aturan approval; setiap occurrence harus punya approval sah, atau dilewati/ditahan tanpa kirim.
- Dashboard business scheduling tetap milik Dashboard; **jangan** membuat duplikat job di GoWA scheduler.

### FLOW-07: Lead CS yang sudah ada

- Hermes CRM / CS mengusulkan follow-up via MCP Dashboard.
- CS/Super Admin approve di /app/leads/followups.
- Dashboard worker mengambil approved follow-up, mengecek jadwal/opt-out/stage/window/cap.
- Dashboard mengirim via Hermes CS Bridge POST /api/outbound.
- Bridge mengirim dari device cs-ai, menambahkan konteks percakapan Hermes, dan mengirim event message.out kembali.
- **Tidak ada** MCP GoWA direct send ke device cs-ai dalam proses tersebut.


## 7. Functional requirements

| ID | Requirement (semua PROPOSED kecuali disebut existing) | Prioritas |
|---|---|---|
| FR-01 | Satu logical send mempunyai request_id, idempotency_key, origin, purpose, principal, device, recipient dan owner_system | P0 |
| FR-02 | Policy engine server-side mengevaluasi device, tool dan sub-action pada setiap call serta saat due dispatch | P0 |
| FR-03 | OAuth/auth GoWA mendukung agent/user identity granular tanpa memberi akses lintas device; restricted Basic Auth tidak diperluas otomatis | P0 |
| FR-04 | Direct personal text/image/video/audio/document dapat dieksekusi sesuai scope dan target | P0 |
| FR-05 | Group send selalu PENDING_APPROVAL sebelum eksekusi, walaupun peminta super_admin melalui AI | P0 |
| FR-06 | Persetujuan ditautkan ke device, recipient, payload hash, media hash, jadwal, occurrence dan masa berlaku | P0 |
| FR-07 | Ubah target/isi/media/jadwal => persetujuan invalid dan perlu review ulang | P0 |
| FR-08 | Approval manusia tidak dapat dilakukan oleh principal agent melalui tool yang sama | P0 |
| FR-09 | Dashboard Finance/Admin Kelas/Koordinator mengirim memakai entity business, bukan nomor yang diambil mentah dari AI | P0 |
| FR-10 | Extend channel wa_outbox untuk non-CS, sementara bridge_cs existing tetap utuh | P0 |
| FR-11 | CS AI device dibatasi dari direct send operator/MCP, termasuk lewat device override | P0 |
| FR-12 | Outbound retries tidak menggandakan pesan; ambiguous timeout/restart => UNCERTAIN dan rekonsiliasi | P0 |
| FR-13 | Scheduler ownership tunggal dan business state diperiksa kembali sebelum dispatch | P0 |
| FR-14 | Media URL divalidasi terhadap SSRF, redirect, konten, MIME dan size limits | P0 |
| FR-15 | Lampiran Claude/Hermes dapat memakai attachment adapter atau secure temporary-upload fallback | P0 |
| FR-16 | Audit lintas asal/aktor/device/ID request/status/approval tanpa menyimpan secret | P0 |
| FR-17 | GoWA Chats dan Scheduled tetap memakai storage existing; kegiatan AI punya label origin di activity log | P1 |
| FR-18 | Halaman admin GoWA bisa meninjau, menolak dan menyetujui group send yang diizinkan | P0 |
| FR-19 | Dashboard menu integrasi tetap super_admin; UI role action ditanam pada halaman proses bisnis masing-masing | P1 |
| FR-20 | Feature switches dan scoped revocation memungkinkan operator menghentikan integrasi baru tanpa memutus CS existing | P0 |
| FR-21 | MCP Dashboard tetap membatasi/meredaksi kontak dan tidak mempunyai tool raw-send yang membypass workflow | P0 |
| FR-22 | Error, schedule status, delivery status dan approval state dibedakan; 2xx bukan bukti pesan dibaca | P0 |

## 8. Kontrak data dan API yang diusulkan

> Semua kontrak di bagian 8 adalah **PROPOSED**. Nama field/endpoint final menunggu schema/API audit; jangan mengklaim sudah tersedia.

### 8.1 Conceptual SendIntent

| Field | Keterangan |
|---|---|
| request_id | UUID stabil untuk correlation lintas Dashboard/GoWA |
| origin | dashboard_finance / dashboard_admin_kelas / dashboard_koordinator / dashboard_cs / claude / hermes / gowa_ui / external_api |
| purpose | lead_followup / payment_verified / tagihan / reminder_kelas / tindak_lanjut_guru / direct_message / group_message |
| principal_id / client_id | Dibentuk server dari authentication, bukan dipercayai dari body request |
| device_id | Device yang diizinkan policy |
| target_ref | Business entity ID untuk Dashboard atau JID tervalidasi untuk GoWA direct |
| target_type | personal / group |
| payload_type | text/image/video/audio/document dan jenis existing lain |
| payload_ref / payload_hash | Snapshot immutable yang menjadi dasar approval dan deduplication |
| media_ref | Ref media staging privat jika ada |
| scheduled_at / timezone / recurrence | Owner scheduler, UTC storage, timezone IANA eksplisit |
| owner_system | dashboard atau gowa |
| idempotency_key / occurrence_id | Unik untuk satu logical execution |
| approval_ref | ID persetujuan valid, bila diperlukan |
| wa_message_id | Terisi setelah GoWA mengembalikan ID pesan |
| status / last_error | State yang dapat diaudit tanpa secret |

### 8.2 Channel contract

- Dashboard → device cs-ai: **existing CS Bridge**, bukan GOWA MCP/raw REST. Format existing hanya text personal dan purpose followup/payment_verified.
- Dashboard → device keuangan/admin-kelas/koord-guru: **proposed** business dispatcher melalui REST GoWA dengan account/service credential yang dibatasi device.
- Claude/Hermes → GoWA: remote MCP OAuth dan scoped principal. Service agent tidak memakai token MCP Dashboard rsai_* untuk akses GoWA.
- GoWA → Dashboard: update status via mekanisme event/poll scoped yang didesain dan diuji; tidak menambah webhook publik tanpa auth.
- Metadata origin yang dikirim client tidak boleh menjadi bukti kepercayaan; ditentukan server berdasarkan auth context.

### 8.3 Status standar

- DRAFT, PENDING_APPROVAL, APPROVED, QUEUED, SCHEDULED, SENDING, SENT, FAILED, CANCELLED, REJECTED, EXPIRED, UNCERTAIN.
- SENDING -> UNCERTAIN bila timeout sesudah kemungkinan pengiriman ke upstream; jangan otomatis mengulang tanpa rekonsiliasi.
- SENT berarti GoWA/WhatsApp menerima operasi pengiriman dan memberikan ID, bukan delivered/read.
- Job recurring memiliki status per occurrence, bukan satu approval permanen untuk semua kiriman grup.
- Error code minimal: UNAUTHORIZED, FORBIDDEN, DEVICE_DENIED, ACTION_DENIED, TARGET_DENIED, APPROVAL_REQUIRED, APPROVAL_EXPIRED, STALE_BUSINESS_STATE, MEDIA_INVALID, RATE_LIMITED, UPSTREAM_UNAVAILABLE, SEND_UNCERTAIN.

### 8.4 Persistensi konseptual

**Dashboard**: reuse wa_outbox, lead_followups, dan tabel bisnis existing. Pertimbangkan penambahan additive fields pada migration baru: origin, principal, device, request_id, target entity, approval reference, media reference, idempotency per occurrence, external WA message ID, status sinkronisasi dan hasil error. Jangan membuat tabel antrean kedua jika perluasan wa_outbox cukup.

**GoWA**: reuse message/chat storage dan scheduled_send. Tambahkan durable request/approval/audit metadata untuk MCP direct, media staging TTL, principal/action attribution dan state rekonsiliasi. Satu storage tidak menjadi source of truth bisnis pihak lain.

**Reconciliation**: message ID + request ID + source; event id unik; retries dan replay harus idempotent. Jika WhatsApp send sukses tetapi penulisan async chat storage gagal, activity transport tetap menunjukkan send sukses dengan histori UI belum tersimpan, bukan mengaku gagal kirim.


## 9. Approval dan scheduling detail

### 9.1 Approval state machine

    DRAFT -> PENDING_APPROVAL -> APPROVED -> QUEUED -> SENDING -> SENT
               |                   |                   |
               +-> REJECTED        +-> EXPIRED          +-> UNCERTAIN
               +-> CANCELLED       +-> CANCELLED        +-> FAILED

Approver melihat preview minimal: peminta, asal Claude/Hermes/Dashboard, device, grup/nomor tujuan, jumlah penerima bila relevan, jenis media, caption/teks, thumbnail/reference aman, waktu kirim/timezone, repeat pattern, serta payload hash. Approval harus dilakukan pada trusted server UI oleh identitas manusia dengan role berwenang. Tidak boleh ada action MCP bagi agent untuk self-approve.

- Perubahan payload/target/device/schedule membutuhkan approval ulang.
- Approval tidak otomatis berarti SENT; worker re-check saat dispatch.
- Jika approval expired saat schedule due, status ditahan dan tidak mengirim.
- Untuk grup recurring, setiap occurrence memerlukan approval yang sah; approve seluruh pola tanpa batas waktu tidak diperbolehkan pada MVP.

### 9.2 Scheduler ownership

| Owner | Jenis job | Tempat tersimpan | Aksi operasional |
|---|---|---|---|
| Dashboard | CRM follow-up/payment_verified dan seluruh business message Finance/Admin Kelas/Koordinator | wa_outbox dan domain tables | Kelola pada Dashboard dan periksa ulang state bisnis sebelum due |
| GoWA | Manual Claude/Hermes non-business personal/group dan scheduling direct | GoWA ScheduledSend existing + approval fields baru | Kelola melalui GoWA Scheduled/MCP sesuai izin |

Satu logical message tidak boleh disimpan sebagai dua scheduled job yang dapat mengirim sendiri-sendiri. Dashboard dispatch ketika due dengan REST immediate, tanpa memanggil scheduler GoWA untuk job yang sama. Gunakan UTC untuk storage dan Asia/Jakarta atau timezone IANA yang eksplisit untuk preview dan aturan jam.

## 10. Media URL dan upload dari AI

### 10.1 URL-based media

- GoWA sudah punya handler MCP image_url, video_url, file_url dan audio_url; jangan menduplikasi codec/transport.
- **PROPOSED security**: cegah SSRF, akses IP internal/loopback/metadata, DNS rebinding, protokol non-HTTP(S), redirect ke alamat terlarang, file terlalu besar, mismatch tipe/MIME, decompression bombs, dan long-running fetch.
- Media harus di-validate sebelum approval dan revalidate/bekukan konten saat dispatch agar URL berubah tidak mengubah payload approved.

### 10.2 Direct attachment from Claude/Hermes

- Keinginan pengguna: file dapat dilampirkan di lingkungan AI atau berasal dari URL.
- **Tidak diasumsikan** klien Claude/Hermes otomatis meneruskan file biner attachment ke GoWA MCP. Kemampuan transfer client harus diuji satu per satu.
- Jika tidak tersedia, sediakan secure upload adapter/page untuk memperoleh media_ref temporer. Jangan membuat raw local path dari agent menjadi path server.
- Storage privat, access scoped, type/content inspection, per-file size, signed one-time reference/TTL, private fetch, dan cleanup.
- File sementara dihapus setelah masa berlaku/retensi yang ditetapkan; audit merekam identitas media dan hash, bukan seluruh konten.

## 11. UI dan ownership

### Dashboard: menu existing harus tetap dapat digunakan

- /app/integrasi-whatsapp: halaman **super_admin** untuk connection health, device mapping, saklar follow-up existing, status outbox, link ke audit dan policy summary. Jangan memindahkan approval CS ke sini.
- Finance: tombol/draft/riwayat pengiriman dari halaman tagihan yang sah.
- Admin Kelas: tindakan dari santri/kelas/pertemuan terkait.
- Koordinator Guru: tindakan dari guru/pertemuan terkait.
- /app/leads/followups: tetap untuk approval CRM CS existing.
- MCP Dashboard Akses AI: tetap memisahkan token dan permission business analytics dari auth GoWA.

### GoWA UI

- Existing UI Chats dan Scheduled adalah tempat melihat histori/jadwal dari GoWA yang berhasil masuk storage.
- Tambahkan halaman **AI Activity**, **Approvals**, dan setting principal/device/permission; tampilkan origin Claude, Hermes, Dashboard, UI dan hasil action.
- **Peringatan ownership:** source GoWA UI berada di repo aldinokemal/gowa-ui dan diambil sebagai GitHub release asset. Penambahan UI memerlukan strategi owned fork/build dan konfigurasi release pin secara eksplisit; jangan mengasumsikan perubahan backend GoWA akan otomatis mengubah UI produksi.
- Untuk Dashboard business message, status bisnis tetap terlihat di Dashboard meskipun histori chat GoWA sudah tersedia.

## 12. Keamanan, privasi, reliability dan operasi

1. **Auth:** HTTPS, OAuth untuk human-facing MCP, service credentials scoped untuk Dashboard; jangan reuse rsai_* token Dashboard untuk GoWA.
2. **Granular authorization:** server-side per principal, device, tool/sub-action, recipient, scope dan waktu eksekusi. Jangan mengandalkan anotasi MCP readOnly/destructive atau UI disabled.
3. **CS isolation:** tidak ada raw send langsung dari MCP/REST Dashboard ke device cs-ai untuk workflow lead; gunakan bridge yang mempertahankan konteks dan event.
4. **Privacy:** MCP Dashboard tetap menolak pengiriman kontak, email, dan wa_chat_id. GoWA direct contact read hanya untuk principal yang punya izin; no prompt-log leakage.
5. **Opt-out:** tindak lanjut bisnis dilarang jika do-not-contact, status tidak layak, atau nomor tidak sah; revalidate tepat sebelum send.
6. **Idempotency/retry:** request_id/occurrence_id unik, dedup pada dispatcher, tidak blind retry setelah timeout yang statusnya ambigu.
7. **Rate limit:** per principal/device/recipient/target dan kill switch; jangan mengasumsikan cap CRM berlaku untuk semua non-CS atau sebaliknya.
8. **Approval:** group send menunggu manusia dan tidak dapat diloloskan lewat raw REST/tool lain; semua entrypoint yang bisa mengirim harus memakai enforcement yang setara.
9. **Media:** private staging, TTL, SSRF protection, no secret/media path leak.
10. **Audit:** identity, action, origin, permission result, device, request ID, approver, outcome, WA message ID jika ada; batasi penyimpanan isi konten sensitif.
11. **Observability:** queued, pending, approved, sent, failed, uncertain, approval latency, auth denials, media errors, age event CS, device disconnect dan outbox backlog.
12. **Production:** deployment Cloudflare/reverse proxy/TLS memerlukan review tersendiri; perubahan credential, WAF dan SSL tidak dilakukan tanpa approval.
13. **Volume:** penggunaan WhatsApp Web untuk automasi volume tinggi berisiko kena pembatasan akun; pertimbangkan WhatsApp Business Platform resmi bila skala meningkat.

## 13. Acceptance criteria

| ID | Uji | Hasil minimal |
|---|---|---|
| AC-01 | Claude remote MCP OAuth | Connect dengan principal teridentifikasi dan tool yang diizinkan |
| AC-02 | Hermes remote MCP | Profile terpisah dari CS, permission sesuai grant |
| AC-03 | Device scope | Override device_id/header tidak dapat akses device terlarang |
| AC-04 | Action scope | Read-only/sender tidak bisa logout device atau mengeluarkan anggota |
| AC-05 | Personal text | Pesan uji terkirim sekali, origin dan WA message ID tercatat |
| AC-06 | Media URL | Image/video/PDF/audio yang valid terkirim ke penerima uji |
| AC-07 | Attachment AI | Upload/staging atau fallback sukses; TTL cleanup terbukti |
| AC-08 | Group approval | Tidak ada pengiriman sebelum approve; reject/expire tidak mengirim |
| AC-09 | Approval tamper | Ubah device/tujuan/teks/media/schedule menginvalidasi approval |
| AC-10 | Group recurring | Occurrence tanpa approval valid tidak pernah terkirim |
| AC-11 | Finance business | Draft tagihan eligible ke device Finance, status tercatat |
| AC-12 | Finance stale | Tagihan lunas pada waktu due tidak dikirimi reminder |
| AC-13 | Admin Kelas | Hanya kelas/santri yang diizinkan; perubahan jadwal membatalkan draft usang |
| AC-14 | Koordinator Guru | Hanya guru/pertemuan dalam scope yang dapat ditindaklanjuti |
| AC-15 | CS regression | Follow-up CRM via bridge, event + prompt context Hermes tetap sinkron |
| AC-16 | Dashboard MCP privacy | Redaksi kontak tetap PASS, tidak muncul raw send bypass |
| AC-17 | Single scheduler | Satu business schedule hanya satu owner, tidak double dispatch |
| AC-18 | Replay/timeout | Idempotency PASS, UNCERTAIN tidak blind retry |
| AC-19 | UI | Chats/Scheduled existing terbaca; AI Activity/Approvals terbuka sesuai role |
| AC-20 | Audit | Semua tindakan yang diuji dapat ditelusuri tanpa token/secret |
| AC-21 | Negative security | Expired permission, invalid group approval, SSRF, oversized media ditolak |
| AC-22 | Deployment smoke | Hanya setelah approval, uji satu nomor dan grup yang dikelola tim; bukan lead riil tanpa consent |

**Verification rule:** PASS hanya jika ada direct evidence dari test yang benar-benar dijalankan. Build PASS tidak berarti production deployed/healthy; request HTTP 200 tidak membuktikan delivery/read WhatsApp.

## 14. Implementasi bertahap, ownership, gate

| Fase | Scope | Repo dominan | Output |
|---|---|---|---|
| 0 — Targeted technical audit | Exact schema wa_outbox, device mapping, auth principal, GoWA send/schedule entrypoints, CS bridge, UI deployment source, media upload capability tiap client | Dashboard + GoWA + UI | File/symbol map, kontrak existing, gap/risk |
| 1 — Permission foundation | MCP principal identity, per-device/action policy, CS isolation, auth/revoke | GoWA | Tests negative security PASS |
| 2 — Safe direct messaging | Personal direct, group approval durable, idempotency, dispatcher & audit | GoWA | Test direct send dan approval uji |
| 3 — Media | URL hardening + secure upload/fallback, cleanup | GoWA + adapters | Media/security tests |
| 4 — Dashboard business workflow | Perluas wa_outbox non-CS, role checks, approve/review, stale-state checks, reconciliation | Dashboard, GoWA bila perlu | Finance/Admin Kelas/Koordinator tests |
| 5 — UI operation | AI Activity, Approvals, status, owned fork/release pin UI, Dashboard integration health | GoWA UI + Dashboard | Browser E2E + UI evidence |
| 6 — Production rollout | Migration additive, backup/rollback, staged pilot, smoketest, observability | Ops lintas repo | Terpisah, perlu approval deploy |

**Coding rules Dashboard:** handler → service → sqlc-generated queries → SQLite; migration existing tidak diedit; satu PR focused per fase/repo; tidak ada generic dependency upgrade. **Merge dan deployment adalah approval terpisah**. CS Bridge yang aktif harus tetap jalan ketika pilot non-CS diuji.

**Rollback:** scoped feature flags, nonaktifkan worker/integrasi baru tanpa mematikan bridge, audit queue dan UNCERTAIN, gunakan backup dan rollback build terverifikasi, jangan hapus history/approval tanpa izin. Pesan yang sudah terkirim tidak dapat dibatalkan dengan rollback kode.

## 15. Decisions, pending approvals, dan checkpoint

### LOCKED dari diskusi pengguna

- Akses platform ke semua device; izin berbeda per agent.
- Semua kategori tools GoWA dengan granular control.
- Direct personal send boleh otomatis jika diizinkan.
- Group send memerlukan approval manusia.
- URL media dan attachment langsung/fallback.
- Kegiatan direct agent dan scheduling di GoWA UI, dengan activity metadata tambahan.
- Integrasi Dashboard tidak boleh merusak flow bisnis dan CS existing.
- Belum ada approval perubahan code aplikasi, merge, credential, atau production.

### OPEN DECISIONS (bukan asumsi final)

| ID | Pertanyaan material | Default desain yang aman |
|---|---|---|
| OD-01 | Untuk bisnis Keuangan, Admin Kelas, Koordinator: boleh auto-send atau selalu disetujui staf? | Review/approval staf dahulu; otomatis hanya nanti setelah policy+template disetujui |
| OD-02 | Siapa approver grup direct selain Super Admin? | Hanya Super Admin pada MVP; delegasi membutuhkan aturan baru |
| OD-03 | Siapa pemilik build/fork GoWA UI? | Owned pinned build; belum mengubah upstream |
| OD-04 | Daily cap, quiet hours, dan recipient allowlist khusus non-CS/agent? | Tentukan lewat audit/pilot per device dan role, tidak reuse cap CRM sembarangan |
| OD-05 | Retensi audit dan temporary media? | Tentukan sebelum produksi sesuai kebijakan perlindungan data |
| OD-06 | Perluasan CS Bridge agar mendukung media/grup? | Tidak termasuk MVP; perlu kontrak dan dampak pada konteks CS terpisah |

Ketidakpastian ini **tidak menghalangi penulisan PRD**, tetapi keputusan yang relevan harus dikunci sebelum implementasi/aktivasi fitur tersebut.

### Checkpoint

- **STAGE:** PRD / planning.
- **COMPLETED:** targeted inspection Dashboard main, GoWA main, UI upstream, existing flow/PRD; blueprint routing, auth, workflow, acceptance dan rollout.
- **CHANGED:** dokumen Markdown ini pada branch docs khusus.
- **NOT VERIFIED:** runtime production, versi actual UI, pengiriman real, OAuth Claude/Hermes, approval baru.
- **NEXT ACTION:** review PRD dan OD-01/OD-02; lanjut fase 0 audit teknis yang lebih terarah. Tidak mengedit source code atau deploy berdasarkan PRD ini saja.

