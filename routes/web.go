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
}

func SetupRoutes(app *fiber.App, handlers Handlers, store *session.Store, userService *services.UserService, mailerService *services.MailerService, csrfMiddleware *middlewares.CSRFMiddleware) {
	// Setup static file serving
	setupStaticRoutes(app)

	// Setup public routes
	setupPublicRoutes(app, handlers.Public)

	// Setup auth routes
	setupAuthRoutes(app, handlers.Auth, handlers.PasswordReset, store, mailerService, csrfMiddleware)

	// Setup app routes (protected)
	setupAppRoutes(app, handlers.App, handlers.Upload, store, userService, csrfMiddleware)
}

func setupStaticRoutes(app *fiber.App) {
	// Static assets with aggressive caching — hashed filenames from Vite are immutable
	// Compress: true caches compressed (brotli/gzip) versions in memory, minimizing CPU reuse.
	app.Static("/dist", "./dist", fiber.Static{
		CacheDuration: 365 * 24 * time.Hour,
		MaxAge:        31536000, // 1 year in seconds
		Compress:      true,
	})
	app.Static("/assets", "./dist/assets", fiber.Static{
		CacheDuration: 365 * 24 * time.Hour,
		MaxAge:        31536000,
		Compress:      true,
	})
	// Public assets (non-hashed, short cache)
	app.Static("/public", "./public", fiber.Static{
		CacheDuration: 1 * time.Hour,
		MaxAge:        3600,
	})
	// Uploaded files (avatars, completed uploads — moderate cache)
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
	// Login routes (with Guest middleware)
	app.Get("/login", middlewares.Guest(store), authHandler.ShowLoginForm)
	app.Post("/login", middlewares.Guest(store), authHandler.Login, middlewares.AuthRateLimit.Limit())

	// Register routes (with Guest middleware)
	app.Get("/register", middlewares.Guest(store), authHandler.ShowRegisterForm)
	app.Post("/register", middlewares.Guest(store), authHandler.Register, middlewares.AuthRateLimit.Limit())

	// OAuth routes
	app.Get("/auth/google", authHandler.GoogleLogin)
	app.Get("/auth/google/callback", authHandler.GoogleCallback)

	// Logout (requires auth + CSRF protection)
	app.Post("/logout", middlewares.AuthRequired(store), csrfMiddleware.Protect(), authHandler.Logout)

	// Password reset routes
	app.Get("/forgot-password", passwordResetHandler.ShowForgotPasswordForm)
	app.Post("/forgot-password", passwordResetHandler.SendResetLink, middlewares.PasswordResetRateLimit.Limit())
	app.Get("/reset-password/:token", passwordResetHandler.ShowResetPasswordForm)
	app.Post("/reset-password/:token", passwordResetHandler.ResetPassword)
}

func setupAppRoutes(app *fiber.App, appHandler *handlers.AppHandler, uploadHandler *handlers.UploadHandler, store *session.Store, userService *services.UserService, csrfMiddleware *middlewares.CSRFMiddleware) {
	// Protected app routes with CSRF protection
	protected := app.Group("/app", middlewares.AuthRequired(store))
	protected.Use(csrfMiddleware.Protect())

	// Dashboard
	protected.Get("/", appHandler.Dashboard)

	// Profile
	protected.Get("/profile", appHandler.Profile)
	protected.Put("/profile", appHandler.UpdateProfile)
	protected.Put("/profile/password", appHandler.UpdatePassword)

	// Upload Test page
	protected.Get("/upload", appHandler.UploadTest)

	// Avatar upload (legacy multipart, for Profile page)
	protected.Post("/upload", uploadHandler.AvatarUpload)

	// TUS resumable upload protocol endpoints — directly on app.
	// AuthRequired applied internally by RegisterTUSRoutes via /tus prefix middleware.
	// BasePath = /tus/files/ so Location URLs correctly include /tus/ prefix.
	authMiddleware := middlewares.AuthRequired(store)
	uploadHandler.RegisterTUSRoutes(app, authMiddleware)

	// Admin-only routes
	admin := app.Group("/admin", middlewares.AdminRequired(store, userService))
	admin.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Admin dashboard",
		})
	})
}

// SetupCSRFMiddleware sets up the CSRF middleware
func SetupCSRFMiddleware(secret string, secure bool) *middlewares.CSRFMiddleware {
	config := middlewares.DefaultCSRFConfig(secret)
	config.Secure = secure
	config.SameSite = "Lax"
	return middlewares.NewCSRFMiddleware(config)
}

// SetupMailerService sets up the mailer service
func SetupMailerService(querier *queries.Querier, smtpHost string, smtpPort int, smtpUser, smtpPass, fromEmail, fromName, appURL string) *services.MailerService {
	return services.NewMailerService(querier, smtpHost, smtpPort, smtpUser, smtpPass, fromEmail, fromName, appURL)
}

// SetupPasswordResetHandler sets up the password reset handler
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

// GetAppURL returns the application URL based on environment
func GetAppURL(appPort string, appEnv string) string {
	if appEnv == "production" {
		return "https://yourdomain.com"
	}
	return fmt.Sprintf("http://localhost:%s", appPort)
}
