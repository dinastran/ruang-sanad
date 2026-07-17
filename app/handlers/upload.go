package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/maulanashalihin/tusdfiber"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
)

// UploadHandler manages file uploads via the TUS resumable protocol.
type UploadHandler struct {
	store        *session.Store
	userService  *services.UserService
	TusHandler   *tusdfiber.Handler
	TUSBasePath  string
	completedDir string
}

// NewUploadHandler creates a new UploadHandler with a tusdfiber TUS handler.
func NewUploadHandler(store *session.Store, userService *services.UserService, uploadDir string) *UploadHandler {
	// ── Storage ──────────────────────────────────────────────
	fs := filestore.New(uploadDir)
	fl := filelocker.New(uploadDir)

	composer := tusdfiber.NewStoreComposer()
	fs.UseIn(composer.StoreComposer)
	fl.UseIn(composer.StoreComposer)

	completedDir := "storage/completed"
	if err := os.MkdirAll(completedDir, 0755); err != nil {
		slog.Error("failed to create completed dir", "error", err)
	}

	// ── TUS Handler ───────────────────────────────────────────
	handler, err := tusdfiber.NewHandler(tusdfiber.Config{
		StoreComposer:           composer,
		BasePath:                "/tus/files/",
		MaxSize:                 1024 * 1024 * 1024,
		DisableDownload:         false,
		DisableTermination:      false,
		DisableConcatenation:    true,
		NotifyCompleteUploads:   true,
		NotifyTerminatedUploads: false,
		NotifyCreatedUploads:    false,
	})
	if err != nil {
		slog.Error("failed to create TUS handler", "error", err)
		return nil
	}

	h := &UploadHandler{
		store:        store,
		userService:  userService,
		TusHandler:   handler,
		TUSBasePath:  "/tus/files/",
		completedDir: completedDir,
	}

	// Drain CompleteUploads channel and copy files to completed dir
	go h.processCompletedUploads()

	return h
}

// processCompletedUploads drains the CompleteUploads channel and copies
// completed uploads to storage/completed/<filename> for easy access.
func (h *UploadHandler) processCompletedUploads() {
	for event := range h.TusHandler.CompleteUploads {
		h.handleCompletedUpload(event)
	}
}

func (h *UploadHandler) handleCompletedUpload(event tusdfiber.HookEvent) {
	info := event.Upload

	// Get original filename from metadata (base64-decoded by tusdfiber)
	filename := info.MetaData["filename"]
	if filename == "" {
		filename = info.ID
	}

	// Get the filestore path (from .info Storage.Path)
	storePath := info.Storage["Path"]
	if storePath == "" {
		slog.Warn("completed upload: missing storage path", "id", info.ID)
		return
	}

	// Destination path
	destPath := filepath.Join(h.completedDir, filename)

	// Copy file
	srcFile, err := os.Open(storePath)
	if err != nil {
		slog.Error("completed upload: failed to open source", "id", info.ID, "error", err)
		return
	}
	defer srcFile.Close()

	// Remove existing file with same name (overwrite)
	os.Remove(destPath)

	dstFile, err := os.Create(destPath)
	if err != nil {
		slog.Error("completed upload: failed to create destination", "path", destPath, "error", err)
		return
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		slog.Error("completed upload: copy failed", "id", info.ID, "error", err)
		return
	}

	slog.Info("upload completed and saved",
		"id", info.ID,
		"filename", filename,
		"size", written,
		"url", "/storage/completed/"+filename,
	)
}

// RegisterTUSRoutes registers TUS protocol routes on the given Fiber app.
func (h *UploadHandler) RegisterTUSRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	app.Use("/tus", authMiddleware)
	for _, mw := range tusdfiber.DefaultMiddlewareStack(nil) {
		app.Use("/tus", mw)
	}
	h.TusHandler.Register(app)
}

// AvatarUpload handles the legacy multipart avatar upload (Profile page).
func (h *UploadHandler) AvatarUpload(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	userID := sess.Get("user_id")

	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse form"})
	}

	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	file := files[0]

	allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	contentType := file.Header.Get("Content-Type")
	isAllowed := false
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file type. Allowed: JPEG, PNG, GIF, WEBP"})
	}

	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File too large. Max size: 5MB"})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d_%d%s", userID.(int64), time.Now().UnixNano(), ext)
	uploadPath := filepath.Join("storage", "avatars", filename)

	if err := c.SaveFile(file, uploadPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	avatarURL := "/storage/avatars/" + filename

	if err := h.userService.UpdateAvatar(userID.(int64), avatarURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update avatar"})
	}

	sess.Set("avatar", avatarURL)
	sess.Save()

	return c.JSON(fiber.Map{
		"success": true,
		"url":     avatarURL,
		"message": "File uploaded successfully",
	})
}

// GetUploadsDir returns the directory where TUS uploads are stored on disk.
func (h *UploadHandler) GetUploadsDir() string {
	return "storage/uploads"
}
