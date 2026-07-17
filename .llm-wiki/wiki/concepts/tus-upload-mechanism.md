# TUS Upload Mechanism

Laju Go uses [tusdfiber](https://github.com/maulanashalihin/tusdfiber) for resumable file uploads via the TUS protocol, integrated with Go Fiber natively.

## Architecture

```
Browser (tus-js-client)         Go Fiber Server
        │                             │
        │  POST /tus/files            │
        │  Upload-Length: <size>      │
        ├────────────────────────────►│ tusdfiber.PostFile()
        │  201 Created                │ filestore.NewUpload()
        │  Location: /tus/files/<id>  │
        │◄────────────────────────────┤
        │                             │
        │  PATCH /tus/files/<id>      │
        │  Upload-Offset: 0           │
        │  [chunk data]               │
        ├────────────────────────────►│ tusdfiber.PatchFile()
        │  204 No Content             │ filestore.WriteChunk()
        │  Upload-Offset: <offset>    │
        │◄────────────────────────────┤
        │  (repeat PATCH for chunks)  │
        │                             │
        │  (on final chunk:           │
        │   offset == size)           │
        │                             ├──► processCompletedUploads()
        │                                 │ copy file from
        │                                 │ storage/uploads/<id>
        │                                 │ to storage/completed/<name>
        │                                 │
        │                                 ▼
        │                          /storage/completed/<file>
        │                          (served via app.Static)
```

## Storage Layout

```
storage/
├── uploads/              ← tusd filestore (internal format)
│   ├── <upload-id>       ← raw file data
│   └── <upload-id>.info  ← metadata (size, filename, etc.)
├── completed/            ← post-processed files (original names)
│   └── <original-name>   ← accessible via /storage/completed/<name>
└── avatars/              ← legacy avatar uploads (multipart)
    └── <user>_<ts>.<ext>
```

## Key Endpoints

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/tus/files` | POST | Create upload | Yes |
| `/tus/files` | OPTIONS | Protocol discovery | Yes (skips OPTIONS) |
| `/tus/files/:id` | HEAD | Get offset/info | Yes |
| `/tus/files/:id` | PATCH | Upload chunk | Yes |
| `/tus/files/:id` | GET | Download file | Yes |
| `/tus/files/:id` | DELETE | Terminate upload | Yes |
| `/storage/completed/<name>` | GET | Download completed file | No (public) |
| `/app/upload` | POST | Avatar upload (legacy) | Yes + CSRF |

## Configuration Requirements

1. **`StreamRequestBody: true`** — Required in `fiber.Config` for streaming large upload bodies.
2. **Notification channels** — `NotifyCreatedUploads/CompleteUploads` must be `false` OR their channels must be drained in a goroutine. Channels are unbuffered, sending blocks until read.
3. **BasePath vs Group** — When registering TUS routes directly on `app` (not a group), `BasePath` must include the full prefix so `Location` URLs are correct (e.g. `/tus/files/`).

## Implementation Files

- `app/handlers/upload.go` — TUS handler setup, post-processing, avatar upload
- `routes/web.go` — Route registration for `/tus/*` and `/app/*`
- `frontend/src/pages/app/UploadTest.svelte` — Upload test page with drag-drop
- `cmd/laju-go/main.go` — `StreamRequestBody` config, storage dirs

## Post-Processing Strategy

When an upload completes via TUS (`NotifyCompleteUploads` event), ada 2 skenario:

### Skenario A: Langsung Copy ke `completed/` (sekarang)

Cocok untuk file yang langsung bisa di-download tanpa proses tambahan (dokumen, gambar, zip, dll).

```
storage/uploads/<id>   ← raw file dari TUS
       │
       ▼ copy
storage/completed/<filename>   ← akses via /storage/completed/<name>
```

**Keuntungan:**

- URL clean pake nama asli
- Gak perlu auth (via `app.Static`)
- Langsung bisa diakses

### Skenario B: Proses Lanjutan (transcode, compress, dsb)

Cocok untuk file yang perlu diolah dulu sebelum bisa diakses (video → HLS, image → thumbnail, dsb).

```
storage/uploads/<id>   ← raw file dari TUS (jangan di-copy ke completed)
       │
       ▼ (CompleteUploads event langsung trigger processing)
FFmpeg / ImageMagick / etc.
       │
       ▼
storage/hls/<id>/          ← hasil processing
├── 1080p/
│   ├── segment-001.ts
│   └── playlist.m3u8
├── 720p/
│   └── ...
└── master.m3u8
```

**Keuntungan:**

- Hemat IO — gak perlu copy dulu baru proses
- File asli tetap di `uploads/` — bisa dihapus setelah selesai processing
- Hasil processing disimpan di folder terpisah dengan struktur sendiri

### Kapan Pilih A vs B

| Skenario | A (completed) | B (proses langsung) |
|----------|:-------------:|:-------------------:|
| Dokumen/PDF/Gambar | ✅ | ❌ |
| Video → HLS | ❌ | ✅ |
| ZIP → extract & simpan | ❌ | ✅ |
| File besar → compress | ❌ | ✅ |

## Upload Policy: TUS vs Multipart

| Upload | Mekanisme | Ukuran Maks | Endpoint | Resumable | CSRF | Kebutuhan |
|--------|-----------|-------------|----------|-----------|------|-----------|
| **Avatar** (Profile) | Multipart POST | 5MB | `POST /app/upload` | ❌ | ✅ | Update DB + sync session |
| **File besar** (UploadTest) | TUS chunked | 1GB | `POST /tus/files` | ✅ | ❌ | Simpan file doang |

**Keputusan desain:** Dipisah karena:

1. **Overhead TUS gak sebanding** — Avatar 100KB butuh 3 request (POST, HEAD, PATCH), multipart cukup 1 POST
2. **Kebutuhan beda** — Avatar harus update DB `user.avatar` + sync session, butuh CSRF. TUS cuma simpan file
3. **Resumable cuma berguna buat besar** — 100KB gagal tinggal upload ulang, 500MB baru rugi kalau gak resumable

## Post-Upload Processing

When an upload completes via TUS:

1. `tusdfiber` sends event to `CompleteUploads` channel
2. Goroutine in `processCompletedUploads()` reads the event
3. File is copied from `storage/uploads/<id>` to `storage/completed/<filename>`
4. Filename is extracted from `Upload-Metadata` header (base64-decoded)

This allows direct download via `/storage/completed/<filename>` without authentication. For production, this copy step can be removed and downloads handled via the TUS GET endpoint with auth.
