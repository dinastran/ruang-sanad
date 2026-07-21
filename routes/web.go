package routes

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/handlers"
	"github.com/maulanashalihin/laju-go/app/middlewares"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type Handlers struct {
	Public        *handlers.PublicHandler
	Auth          *handlers.AuthHandler
	App           *handlers.AppHandler
	Upload        *handlers.UploadHandler
	PasswordReset *handlers.PasswordResetHandler
	Santri        *handlers.SantriHandler
	Kelas         *handlers.KelasHandler
	Master        *handlers.MasterHandler
	Import        *handlers.ImportHandler
	Laporan       *handlers.LaporanHandler
	Admin         *handlers.AdminHandler
	Dashboard     *handlers.DashboardHandler
}

func SetupRoutes(app *fiber.App, h Handlers, store *session.Store, userService *services.UserService, mailerService *services.MailerService, csrfMiddleware *middlewares.CSRFMiddleware) {
	setupStaticRoutes(app)
	setupPublicRoutes(app, h.Public)
	setupAuthRoutes(app, h.Auth, h.PasswordReset, store, mailerService, csrfMiddleware)
	setupAppRoutes(app, h, store, userService, csrfMiddleware)
	setupAdminRoutes(app, h, store, userService, csrfMiddleware)
}

func setupStaticRoutes(app *fiber.App) {
	app.Static("/dist", "./dist", fiber.Static{
		CacheDuration: 365 * 24 * time.Hour,
		MaxAge:        31536000,
		Compress:      true,
	})
	app.Static("/assets", "./dist/assets", fiber.Static{
		CacheDuration: 365 * 24 * time.Hour,
		MaxAge:        31536000,
		Compress:      true,
	})
	app.Static("/public", "./public", fiber.Static{
		CacheDuration: 1 * time.Hour,
		MaxAge:        3600,
	})
	app.Static("/storage", "./storage", fiber.Static{
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400,
	})
}

func setupPublicRoutes(app *fiber.App, handler *handlers.PublicHandler) {
	app.Get("/", handler.Index)
	app.Get("/about", handler.About)
}

func setupAuthRoutes(app *fiber.App, authHandler *handlers.AuthHandler, passwordResetHandler *handlers.PasswordResetHandler, store *session.Store, mailerService *services.MailerService, csrfMiddleware *middlewares.CSRFMiddleware) {
	app.Get("/login", middlewares.Guest(store), authHandler.ShowLoginForm)
	app.Post("/login", middlewares.Guest(store), authHandler.Login, middlewares.AuthRateLimit.Limit())
	app.Get("/register", middlewares.Guest(store), authHandler.ShowRegisterForm)
	app.Post("/register", middlewares.Guest(store), authHandler.Register, middlewares.AuthRateLimit.Limit())
	app.Get("/auth/google", authHandler.GoogleLogin)
	app.Get("/auth/google/callback", authHandler.GoogleCallback)
	app.Post("/logout", middlewares.AuthRequired(store), csrfMiddleware.Protect(), authHandler.Logout)
	app.Get("/forgot-password", passwordResetHandler.ShowForgotPasswordForm)
	app.Post("/forgot-password", passwordResetHandler.SendResetLink, middlewares.PasswordResetRateLimit.Limit())
	app.Get("/reset-password/:token", passwordResetHandler.ShowResetPasswordForm)
	app.Post("/reset-password/:token", passwordResetHandler.ResetPassword)
}

func setupAppRoutes(app *fiber.App, h Handlers, store *session.Store, userService *services.UserService, csrfMiddleware *middlewares.CSRFMiddleware) {
	protected := app.Group("/app", middlewares.AuthRequired(store))
	protected.Use(csrfMiddleware.Protect())

	// Dashboard (all roles)
	protected.Get("/", h.Dashboard.Index)

	// Profile
	protected.Get("/profile", h.App.Profile)
	protected.Put("/profile", h.App.UpdateProfile)
	protected.Put("/profile/password", h.App.UpdatePassword)

	// Upload
	protected.Get("/upload", h.App.UploadTest)
	protected.Post("/upload", h.Upload.AvatarUpload)

	// TUS routes
	authMiddleware := middlewares.AuthRequired(store)
	h.Upload.RegisterTUSRoutes(app, authMiddleware)

	// Role middleware is attached PER-ROUTE (not via empty-prefix groups).
	// Fiber mounts an empty-prefix Group's middleware at the parent path, so it
	// leaks onto every later /app/* route — which previously blocked each role
	// from routes owned by another role group. Route-level middleware is scoped
	// to exactly one route and does not leak.
	csRole := middlewares.RoleRequired(store, userService, "cs", "super_admin")
	akRole := middlewares.RoleRequired(store, userService, "admin_kelas", "super_admin")
	kuRole := middlewares.RoleRequired(store, userService, "keuangan", "super_admin")
	// Data Santri is viewable (read-only) by admin_kelas as well; the mutating
	// routes below stay CS-only so admin_kelas can look but not edit.
	santriViewRole := middlewares.RoleRequired(store, userService, "cs", "admin_kelas", "super_admin")

	// CS routes (cs + super_admin); list & detail also viewable by admin_kelas
	protected.Get("/santri", santriViewRole, h.Santri.Index)
	protected.Get("/santri/new", csRole, h.Santri.New)
	protected.Post("/santri", csRole, h.Santri.Store)
	protected.Get("/santri/:id", santriViewRole, h.Santri.Show)
	protected.Put("/santri/:id/cs", csRole, h.Santri.UpdateCS)

	// Admin Kelas routes (admin_kelas + super_admin)
	protected.Get("/perlu-dilengkapi", akRole, h.Santri.PerluDilengkapi)
	protected.Put("/santri/:id/kelas-data", akRole, h.Santri.UpdateAdminKelas)
	protected.Post("/santri/:id/pindah", akRole, h.Santri.PindahKelas)
	protected.Get("/kelas", akRole, h.Kelas.Index)
	protected.Get("/kelas/:id", akRole, h.Kelas.Show)
	protected.Put("/kelas/:id/guru", akRole, h.Kelas.AssignGuru)
	protected.Put("/kelas/:id/status", akRole, h.Kelas.SetAktif)
	protected.Delete("/kelas/:id", akRole, h.Kelas.Delete)

	// Keuangan routes (keuangan + super_admin)
	protected.Get("/keuangan", kuRole, h.Santri.Keuangan)
	protected.Put("/santri/:id/keuangan", kuRole, h.Santri.UpdateKeuangan)
	protected.Get("/laporan/keuangan", kuRole, h.Laporan.Keuangan)

	// Master data API (all authenticated)
	api := protected.Group("/api")
	api.Get("/master/angkatan", h.Master.Angkatan)
	api.Get("/master/level", h.Master.Level)
	api.Get("/master/jadwal", h.Master.Jadwal)
	api.Get("/master/guru", h.Master.Guru)
	api.Get("/master/kode-kelas", h.Master.KodeKelas)

	// Master data management (admin_kelas + super_admin)
	protected.Get("/master", akRole, h.Master.Index)
	protected.Post("/master/angkatan", akRole, h.Master.CreateAngkatan)
	protected.Put("/master/angkatan/:id", akRole, h.Master.UpdateAngkatan)
	protected.Delete("/master/angkatan/:id", akRole, h.Master.DeleteAngkatan)
	protected.Post("/master/level", akRole, h.Master.CreateLevel)
	protected.Put("/master/level/:id", akRole, h.Master.UpdateLevel)
	protected.Delete("/master/level/:id", akRole, h.Master.DeleteLevel)
	protected.Post("/master/jadwal", akRole, h.Master.CreateJadwal)
	protected.Put("/master/jadwal/:id", akRole, h.Master.UpdateJadwal)
	protected.Delete("/master/jadwal/:id", akRole, h.Master.DeleteJadwal)
	protected.Post("/master/guru", akRole, h.Master.CreateGuru)
	protected.Put("/master/guru/:id", akRole, h.Master.UpdateGuru)
	protected.Delete("/master/guru/:id", akRole, h.Master.DeleteGuru)
	protected.Post("/master/kode-kelas", akRole, h.Master.CreateKodeKelas)
	protected.Put("/master/kode-kelas/:id", akRole, h.Master.UpdateKodeKelas)
	protected.Delete("/master/kode-kelas/:id", akRole, h.Master.DeleteKodeKelas)
}

func setupAdminRoutes(app *fiber.App, h Handlers, store *session.Store, userService *services.UserService, csrfMiddleware *middlewares.CSRFMiddleware) {
	sa := app.Group("/admin", middlewares.RoleRequired(store, userService, "super_admin"))
	sa.Use(csrfMiddleware.Protect())
	sa.Get("/", h.Admin.Dashboard)
	sa.Get("/users", h.Admin.Users)
	sa.Put("/users/:id/role", h.Admin.UpdateUserRole)
	sa.Get("/import", h.Import.Show)
	sa.Post("/import", h.Import.Upload)
	sa.Get("/import/template", h.Import.Template)
}

func SetupCSRFMiddleware(secret string, secure bool) *middlewares.CSRFMiddleware {
	config := middlewares.DefaultCSRFConfig(secret)
	config.Secure = secure
	config.SameSite = "Lax"
	return middlewares.NewCSRFMiddleware(config)
}

func SetupMailerService(querier *queries.Querier, smtpHost string, smtpPort int, smtpUser, smtpPass, fromEmail, fromName, appURL string) *services.MailerService {
	return services.NewMailerService(querier, smtpHost, smtpPort, smtpUser, smtpPass, fromEmail, fromName, appURL)
}

func SetupPasswordResetHandler(
	mailerService *services.MailerService,
	userService *services.UserService,
	store *session.Store,
	inertiaService *services.InertiaService,
) *handlers.PasswordResetHandler {
	return handlers.NewPasswordResetHandler(
		mailerService,
		userService,
		store,
		inertiaService,
	)
}

func GetAppURL(appPort string, appEnv string) string {
	if appEnv == "production" {
		return "https://yourdomain.com"
	}
	return fmt.Sprintf("http://localhost:%s", appPort)
}
