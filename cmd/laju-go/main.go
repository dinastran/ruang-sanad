package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/maulanashalihin/laju-go/app/cache"
	"github.com/maulanashalihin/laju-go/app/config"
	"github.com/maulanashalihin/laju-go/app/handlers"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/maulanashalihin/laju-go/routes"
	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO → easy cross-compile)
)

var (
	Version = "v1.0.0"
	Commit  = "none"
)

func main() {
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("laju-go %s (commit: %s)\n", Version, Commit)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.AppEnv == "development" {
		logLevel = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if cfg.AppEnv == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
	slog.Info("starting", "version", Version, "commit", Commit, "env", cfg.AppEnv)

	// Initialize database
	db, err := initDatabase(cfg.DBPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db, "./migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize querier
	querier := queries.NewQuerier(db)

	// Initialize in-memory session cache
	sessionCache := cache.NewSessionCache()

	// Initialize session store with database + in-memory cache
	sessionStore := session.New(querier, sessionCache, cfg.SessionTTL)
	sessionStore.SetSecure(cfg.AppEnv == "production")

	// Start background cleanup for expired sessions and password reset tokens
	startBackgroundCleanup(querier)

	// Initialize services
	authService := services.NewAuthService(querier, services.AuthServiceConfig{
		SessionSecret:      cfg.SessionSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GoogleRedirectURL:  cfg.GoogleRedirectURL,
	})
	userService := services.NewUserService(querier)

	// Initialize Asset service (for production builds with hashed filenames)
	assetService := services.NewAssetService("./dist/.vite/manifest.json", ".vite-port", cfg.IsDevelopment())

	// Initialize Inertia service (auto-detects Vite from .vite-port)
	inertiaService := services.NewInertiaService(assetService, sessionStore)

	// Initialize master data via KelasEngineService
	kelasEngine := services.NewKelasEngineService(querier)

	// Initialize new services
	santriService := services.NewSantriService(querier, kelasEngine)
	kelasService := services.NewKelasService(querier)
	masterService := services.NewMasterService(querier)
	laporanService := services.NewLaporanService(querier, santriService, kelasService)
	importService := services.NewImportService(querier, santriService)
	guruService := services.NewGuruService(querier)
	tagihanService := services.NewTagihanService(querier)
	pertemuanService := services.NewPertemuanService(querier, tagihanService)
	jadwalPertemuanService := services.NewJadwalPertemuanService(querier)
	notificationService := services.NewNotificationService(querier)
	riayahService := services.NewRiayahService(querier)
	koordinatorService := services.NewKoordinatorService(querier)
	koordinatorMonitoringService := services.NewKoordinatorMonitoringService(querier, jadwalPertemuanService)
	tsiService := services.NewTSIService(querier)
	santriStatusService := services.NewSantriStatusService(querier)
	santriStatusService.StartAutoAktifCuti(time.Hour, func(job string, total int, err error) {
		if err != nil {
			slog.Error(job+" gagal", "error", err, "diproses", total)
		} else if total > 0 {
			slog.Info(job, "diproses", total)
		}
	})

	// Create storage directories
	if err := os.MkdirAll("storage/uploads", 0755); err != nil {
		slog.Error("failed to create uploads directory", "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll("storage/avatars", 0755); err != nil {
		slog.Error("failed to create avatars directory", "error", err)
		os.Exit(1)
	}
	if err := os.MkdirAll("data/voice-notes", 0755); err != nil {
		slog.Error("failed to create voice notes directory", "error", err)
		os.Exit(1)
	}

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(sessionStore, userService, "storage/uploads")
	santriHandler := handlers.NewSantriHandler(santriService, masterService, kelasService, sessionStore, inertiaService)
	kelasHandler := handlers.NewKelasHandler(kelasService, services.NewKelasPerubahanService(querier, kelasEngine), santriStatusService, masterService, sessionStore, inertiaService)
	masterHandler := handlers.NewMasterHandler(masterService, sessionStore, inertiaService)
	importHandler := handlers.NewImportHandler(importService, sessionStore, inertiaService)
	laporanHandler := handlers.NewLaporanHandler(laporanService, sessionStore, inertiaService)
	adminHandler := handlers.NewAdminHandler(userService, sessionStore, inertiaService)
	dashboardHandler := handlers.NewDashboardHandler(santriService, laporanService, kelasService, sessionStore, inertiaService)
	guruHandler := handlers.NewGuruHandler(guruService, pertemuanService, riayahService, sessionStore, inertiaService, notificationService)
	pertemuanHandler := handlers.NewPertemuanHandler(guruService, pertemuanService, jadwalPertemuanService, sessionStore, inertiaService)
	jadwalPertemuanHandler := handlers.NewJadwalPertemuanHandler(guruService, jadwalPertemuanService, sessionStore, inertiaService)
	riayahHandler := handlers.NewRiayahHandler(guruService, riayahService, sessionStore, inertiaService)
	koordinatorGuruHandler := handlers.NewKoordinatorGuruHandler(koordinatorService, sessionStore, inertiaService)
	koordinatorGuruDirectoryHandler := handlers.NewKoordinatorGuruDirectoryHandler(guruService, koordinatorService, riayahService, sessionStore, inertiaService)
	koordinatorMonitoringHandler := handlers.NewKoordinatorMonitoringHandler(koordinatorMonitoringService, koordinatorService, sessionStore, inertiaService)
	koordinatorFeaturesHandler := handlers.NewKoordinatorFeaturesHandler(koordinatorService, guruService, riayahService, sessionStore, inertiaService)
	koordinatorAttendanceExportHandler := handlers.NewKoordinatorAttendanceExportHandler(koordinatorService)
	notificationHandler := handlers.NewNotificationHandler(notificationService, sessionStore, inertiaService)
	tsiHandler := handlers.NewTSIHandler(tsiService, guruService, koordinatorService, sessionStore, inertiaService)
	tagihanHandler := handlers.NewTagihanHandler(tagihanService, sessionStore, inertiaService)

	routeHandlers := routes.Handlers{
		Public:                      handlers.NewPublicHandler(authService, userService, inertiaService, assetService),
		Auth:                        handlers.NewAuthHandler(authService, sessionStore, inertiaService),
		App:                         handlers.NewAppHandler(userService, sessionStore, inertiaService),
		Upload:                      uploadHandler,
		Santri:                      santriHandler,
		Kelas:                       kelasHandler,
		Master:                      masterHandler,
		Import:                      importHandler,
		Laporan:                     laporanHandler,
		Admin:                       adminHandler,
		Dashboard:                   dashboardHandler,
		Guru:                        guruHandler,
		Pertemuan:                   pertemuanHandler,
		JadwalPertemuan:             jadwalPertemuanHandler,
		Riayah:                      riayahHandler,
		KoordinatorGuru:             koordinatorGuruHandler,
		KoordinatorGuruDirectory:    koordinatorGuruDirectoryHandler,
		KoordinatorMonitoring:       koordinatorMonitoringHandler,
		KoordinatorFeatures:         koordinatorFeaturesHandler,
		KoordinatorAttendanceExport: koordinatorAttendanceExportHandler,
		Notifications:               notificationHandler,
		TSI:                         tsiHandler,
		Tagihan:                     tagihanHandler,
	}

	// Setup CSRF middleware (Secure cookies only in production with HTTPS)
	csrfMiddleware := routes.SetupCSRFMiddleware(cfg.SessionSecret, cfg.AppEnv == "production")

	// Setup mailer service (with DB-backed token storage)
	appURL := routes.GetAppURL(cfg.AppPort, cfg.AppEnv)
	mailerService := routes.SetupMailerService(
		querier,
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.FromEmail,
		cfg.FromName,
		appURL,
	)

	// Setup password reset handler
	passwordResetHandler := routes.SetupPasswordResetHandler(
		mailerService,
		userService,
		sessionStore,
		inertiaService,
	)
	routeHandlers.PasswordReset = passwordResetHandler

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Laju",
		ErrorHandler: customErrorHandler,
		// Required for TUS resumable upload — streams request body instead of buffering
		StreamRequestBody: true,
		// Read buffer for streaming large uploads
		ReadBufferSize: 64 * 1024,
		// Voice-note uploads accept files up to 20 MB plus multipart overhead.
		BodyLimit: 21 * 1024 * 1024,
	})

	// Global middleware
	// Logger: only in development (avoids string allocation per request in prod)
	if cfg.IsDevelopment() {
		app.Use(logger.New())
	}
	app.Use(recover.New())

	// Response compression (brotli > gzip, best speed for low CPU overhead)
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Inertia middleware: asset version checking, shared props, Vary header
	app.Use(inertiaService.Middleware())

	// CORS with explicit allowed origins (no AllowOriginsFunc in production)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.AllowedOrigins, ","),
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Inertia, X-Inertia-Version, X-Requested-With, Tus-Resumable, Upload-Length, Upload-Offset, Upload-Metadata, Upload-Draft-Interop-Version, Upload-Complete, Upload-Incomplete, Upload-Concat, Upload-Defer-Length",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": Version,
		})
	})

	// Setup routes (includes static file serving)
	routes.SetupRoutes(app, routeHandlers, sessionStore, userService, mailerService, csrfMiddleware)

	go func() {
		slog.Info("server listening", "port", cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

// initDatabase initializes the SQLite database with optimized settings
func initDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// SQLite permits a single writer. One shared connection prevents transient
	// "database is locked" errors when login persists a new session.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Optimize SQLite for production (WAL mode for better concurrency)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, err
	}

	// Balance between durability and performance
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		return nil, err
	}

	// Set cache size to 16MB (negative value = KB) - optimized for Vultr HF 1-2GB RAM
	if _, err := db.Exec("PRAGMA cache_size = -16000"); err != nil {
		return nil, err
	}

	// Enable memory-mapped I/O for NVMe performance (256MB)
	if _, err := db.Exec("PRAGMA mmap_size = 268435456"); err != nil {
		return nil, err
	}

	// Store temporary tables in memory for better performance
	if _, err := db.Exec("PRAGMA temp_store = MEMORY"); err != nil {
		return nil, err
	}

	// Set busy timeout to 5 seconds (wait for locks instead of failing immediately)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, err
	}

	// Set WAL autocheckpoint to 1000 pages (default, but explicit is better)
	if _, err := db.Exec("PRAGMA wal_autocheckpoint = 1000"); err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Log database optimization status
	logDatabaseOptimizations(db)

	return db, nil
}

// logDatabaseOptimizations logs the current SQLite optimization settings
func logDatabaseOptimizations(db *sql.DB) {
	var journalMode, synchronous string
	var cacheSize, busyTimeout, mmapSize, walAutocheckpoint int

	// Query current settings
	err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		slog.Warn("could not verify journal_mode", "error", err)
	}

	err = db.QueryRow("PRAGMA synchronous").Scan(&synchronous)
	if err != nil {
		slog.Warn("could not verify synchronous", "error", err)
	}

	err = db.QueryRow("PRAGMA cache_size").Scan(&cacheSize)
	if err != nil {
		slog.Warn("could not verify cache_size", "error", err)
	}

	err = db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		slog.Warn("could not verify busy_timeout", "error", err)
	}

	err = db.QueryRow("PRAGMA mmap_size").Scan(&mmapSize)
	if err != nil {
		slog.Warn("could not verify mmap_size", "error", err)
	}

	err = db.QueryRow("PRAGMA wal_autocheckpoint").Scan(&walAutocheckpoint)
	if err != nil {
		slog.Warn("could not verify wal_autocheckpoint", "error", err)
	}

	slog.Info("sqlite optimizations",
		"journal_mode", journalMode,
		"synchronous", synchronous,
		"cache_size_kb", cacheSize,
		"mmap_size_kb", mmapSize,
		"wal_autocheckpoint", walAutocheckpoint,
		"busy_timeout_ms", busyTimeout,
	)
}

// startBackgroundCleanup runs periodic cleanup of expired sessions and password reset tokens
func startBackgroundCleanup(querier *queries.Querier) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run cleanup immediately on startup
		if err := querier.DeleteExpiredSessions(context.Background()); err != nil {
			slog.Error("cleanup: failed to delete expired sessions", "error", err)
		} else {
			slog.Debug("cleanup: expired sessions deleted")
		}
		if err := querier.DeleteExpiredPasswordResets(context.Background()); err != nil {
			slog.Error("cleanup: failed to delete expired password resets", "error", err)
		} else {
			slog.Debug("cleanup: expired password resets deleted")
		}

		for range ticker.C {
			if err := querier.DeleteExpiredSessions(context.Background()); err != nil {
				slog.Error("cleanup: failed to delete expired sessions", "error", err)
			}
			if err := querier.DeleteExpiredPasswordResets(context.Background()); err != nil {
				slog.Error("cleanup: failed to delete expired password resets", "error", err)
			}
		}
	}()
	slog.Info("background cleanup started (interval: 1h)")
}

func runMigrations(db *sql.DB, migrationsDir string) error {
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite"); err != nil {
		return err
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	return nil
}

// customErrorHandler handles Fiber errors
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// For Inertia requests, return JSON
	if c.Get("X-Inertia") == "true" {
		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set Content-Type: application/json; charset=utf-8
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	// Return custom error page
	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
