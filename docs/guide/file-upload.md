# File Upload

Laju Go supports file uploads for user avatars with validation and secure storage.

## Backend Handler

### Actual Implementation

```go
// app/handlers/upload.go
func (h *UploadHandler) Upload(c *fiber.Ctx) error {
    sess, _ := h.store.Get(c)
    userID := sess.Get("user_id")
    if userID == nil {
        return c.Status(401).JSON(fiber.Map{"error": "Not authenticated"})
    }

    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Failed to parse form"})
    }

    files := form.File["file"]
    if len(files) == 0 {
        return c.Status(400).JSON(fiber.Map{"error": "No file uploaded"})
    }

    file := files[0]

    // Validate file type
    allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
    // ... validate ...

    // Validate file size (max 5MB)
    if file.Size > 5*1024*1024 {
        return c.Status(400).JSON(...)
    }

    // Generate unique filename: {userID}_{timestamp}{ext}
    ext := filepath.Ext(file.Filename)
    filename := fmt.Sprintf("%d_%d%s", userID.(int64), time.Now().UnixNano(), ext)
    uploadPath := filepath.Join("storage", "avatars", filename)
    c.SaveFile(file, uploadPath)

    // Update user avatar in database
    avatarURL := "/storage/avatars/" + filename
    h.userService.UpdateAvatar(userID.(int64), avatarURL)

    return c.JSON(fiber.Map{
        "success": true,
        "url":     avatarURL,
    })
}
```

## Frontend Pattern

### Actual Implementation (Profile.svelte)

Ikuti pattern yang sudah ada di `Profile.svelte`:

```svelte
<script lang="ts">
    import { router } from "@inertiajs/svelte";

    function handleAvatarChange(event: Event) {
        const target = event.target as HTMLInputElement;
        const file = target.files?.[0];
        if (!file) return;

        const formData = new FormData();
        formData.append("file", file);

        // 1. Upload file ke server
        fetch("/app/upload", { method: "POST", body: formData })
            .then((r) => r.json())
            .then((data) => {
                if (!data.success) return;

                // 2. Simpan URL via Inertia — server re-render otomatis
                router.put("/app/profile", { avatar: data.url });
            });
    }
</script>

<label>
    <input type="file" accept="image/*" onchange={handleAvatarChange} class="hidden" />
    <Upload class="w-5 h-5" />
</label>
```

### Kenapa `fetch()` + `router.put()`, bukan `router.post()`?

| Approach | Problem |
|----------|---------|
| `router.post()` | Inertia JSON-serialize form data — **tidak bisa kirim file** (binary) |
| `fetch()` upload | Bisa kirim `FormData` dengan file binary |
| `router.put()` after upload | Simpan URL + trigger Inertia re-render |

## Route Setup

```go
// routes/web.go
protected := app.Group("/app", middlewares.AuthRequired(store))
protected.Use(csrfMiddleware.Protect())
protected.Post("/upload", uploadHandler.Upload)
protected.Put("/profile", appHandler.UpdateProfile)
```

## Storage

File disimpan di `storage/avatars/{userID}_{timestamp}{ext}`.

Served via static route:

```go
// routes/web.go
app.Static("/storage", "./storage", fiber.Static{
    CacheDuration: 24 * time.Hour,
    MaxAge:        86400,
})
```

## Security

- **Authentication** — `middlewares.AuthRequired` + session check
- **CSRF** — endpoint dilindungi CSRF middleware
- **File type validation** — only JPEG, PNG, GIF, WEBP
- **File size limit** — max 5MB
- **Unique filenames** — `{userID}_{timestamp}{ext}`, no overwrite

## Troubleshooting

### Upload gagal "Not authenticated"

Pastikan user sudah login dan session valid.

### Upload gagal "Failed to parse form"

Cek `BodyLimit` di Fiber config:

```go
app := fiber.New(fiber.Config{
    BodyLimit: 10 * 1024 * 1024, // 10MB
})
```

### File tidak muncul setelah upload

Cek `storage/avatars/` directory permissions:

```bash
mkdir -p storage/avatars
chmod 755 storage/avatars
```
