# Storage

This guide covers file storage in Laju Go.

## Overview

Laju Go stores uploaded files on the local filesystem under the `storage/` directory.

```
storage/
└── avatars/                   # User avatar uploads
```

## Directory Structure

| Path | Purpose | Served At |
|------|---------|-----------|
| `dist/` | Built frontend assets (immutable, 1yr cache) | `/dist/*`, `/assets/*` |
| `storage/avatars/` | User uploaded avatars | `/storage/*` |
| `public/` | Static assets (favicon, images) | `/public/*` |

Static file serving is configured in `routes/web.go`:

```go
app.Static("/dist", "./dist", fiber.Static{CacheDuration: 365 * 24 * time.Hour, MaxAge: 31536000, Compress: true})
app.Static("/assets", "./dist/assets", fiber.Static{CacheDuration: 365 * 24 * time.Hour, MaxAge: 31536000, Compress: true})
app.Static("/public", "./public", fiber.Static{CacheDuration: 1 * time.Hour, MaxAge: 3600})
app.Static("/storage", "./storage", fiber.Static{CacheDuration: 24 * time.Hour, MaxAge: 86400})
```

## File Upload

### Handler (Actual Implementation)

```go
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

    // Validate file type (JPEG, PNG, GIF, WEBP)
    allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
    // ... validate content-type ...

    // Validate file size (max 5MB)
    if file.Size > 5*1024*1024 { ... }

    // Unique filename: {userID}_{timestamp}{ext}
    ext := filepath.Ext(file.Filename)
    filename := fmt.Sprintf("%d_%d%s", userID.(int64), time.Now().UnixNano(), ext)
    uploadPath := filepath.Join("storage", "avatars", filename)
    c.SaveFile(file, uploadPath)

    // Update user avatar in database
    avatarURL := "/storage/avatars/" + filename
    h.userService.UpdateAvatar(userID.(int64), avatarURL)

    return c.JSON(fiber.Map{"success": true, "url": avatarURL})
}
```

### Avatar Download (Auth Service)

Avatars from Google OAuth are downloaded directly in the auth service and saved to local storage. The `user.avatar` field contains the local path (`/storage/avatars/<googleID>.jpg`), which is served directly by Fiber's static file handler.

```go
// app/services/auth.go
func (s *AuthService) downloadAndSaveAvatar(ctx context.Context, pictureURL, googleID string) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, pictureURL, nil)
    resp, _ := http.DefaultClient.Do(req)
    defer resp.Body.Close()

    filename := googleID + ".jpg"
    os.MkdirAll("./storage/avatars", 0750)
    f, _ := os.Create(filepath.Join("./storage/avatars", filename))
    defer f.Close()
    io.Copy(f, resp.Body)

    return "/storage/avatars/" + filename, nil
}
```

The frontend uses `user.avatar` directly as an `<img src>` — no API proxy needed.

## Backups

Backup `storage/` along with the database:

```bash
tar -czf backup.tar.gz data/ storage/
```

## Next Steps

- [File Upload Guide](file-upload.md) — Detailed upload handling
- [Data Protection Guide](../guide/data-protection.md) — Backup strategies
