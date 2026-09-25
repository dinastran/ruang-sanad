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
	Public                      *handlers.PublicHandler
	Auth                        *handlers.AuthHandler
	App                         *handlers.AppHandler
	Upload                      *handlers.UploadHandler
	PasswordReset               *handlers.PasswordResetHandler
	Santri                      *handlers.SantriHandler
	SantriAnalytics             *handlers.SantriAnalyticsHandler
	Kelas                       *handlers.KelasHandler
	Master                      *handlers.MasterHandler
	Import                      *handlers.ImportHandler
	Laporan                     *handlers.LaporanHandler
	Admin                       *handlers.AdminHandler
	Dashboard                   *handlers.DashboardHandler
	Guru                        *handlers.GuruHandler
	Pertemuan                   *handlers.PertemuanHandler
	JadwalPertemuan             *handlers.JadwalPertemuanHandler
	Riayah                      *handlers.RiayahHandler
	RiayahSantri                *handlers.RiayahSantriHandler
	KoordinatorGuru             *handlers.KoordinatorGuruHandler
	KoordinatorGuruDirectory    *handlers.KoordinatorGuruDirectoryHandler
	KoordinatorMonitoring       *handlers.KoordinatorMonitoringHandler
	KoordinatorFeatures         *handlers.KoordinatorFeaturesHandler
	KoordinatorAttendanceExport *handlers.KoordinatorAttendanceExportHandler
	Notifications               *handlers.NotificationHandler
	TSI                         *handlers.TSIHandler
	Tagihan                     *handlers.TagihanHandler
	ProductCRM                  *handlers.ProductCRMHandler
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
		// Compress disabled on purpose: Fiber's Static compression writes cached
		// .fiber.gz/.br files next to the assets, which fails when dist is a
		// read-only deploy dir (systemd ProtectSystem=strict) → 404 on gzip
		// requests. Cloudflare (and any reverse proxy) compresses at the edge.
	})
	app.Static("/assets", "./dist/assets", fiber.Static{
		CacheDuration: 365 * 24 * time.Hour,
		MaxAge:        31536000,
		// Compress disabled on purpose: Fiber's Static compression writes cached
		// .fiber.gz/.br files next to the assets, which fails when dist is a
		// read-only deploy dir (systemd ProtectSystem=strict) → 404 on gzip
		// requests. Cloudflare (and any reverse proxy) compresses at the edge.
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
	protected.Put("/notifications/:id/read", h.Notifications.MarkRead)

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
	kuReadRole := middlewares.RoleRequired(store, userService, "keuangan", "super_admin", "admin")
	superAdminRole := middlewares.RoleRequired(store, userService, "super_admin")
	// Data Santri is viewable (read-only) by admin_kelas as well; the mutating
	// routes below stay CS-only so admin_kelas can look but not edit.
	santriViewRole := middlewares.RoleRequired(store, userService, "cs", "admin_kelas", "super_admin")

	// Santri analytics is management-only and read-only.
	protected.Get("/analitik-santri", superAdminRole, h.SantriAnalytics.Index)

	// CS routes (cs + super_admin); list & detail also viewable by admin_kelas
	protected.Get("/santri", santriViewRole, h.Santri.Index)
	protected.Get("/santri/new", csRole, h.Santri.New)
	protected.Post("/santri", csRole, h.Santri.Store)
	protected.Get("/santri/:id", santriViewRole, h.Santri.Show)
	protected.Put("/santri/:id/cs", csRole, h.Santri.UpdateCS)
	protected.Put("/santri/:id/registration-identity", superAdminRole, h.Santri.CorrectRegistrationIdentity)
	protected.Delete("/santri/:id", superAdminRole, h.Santri.Delete)

	// Admin Kelas routes (admin_kelas + super_admin)
	protected.Get("/perlu-dilengkapi", akRole, h.Santri.PerluDilengkapi)
	protected.Put("/santri/:id/kelas-data", akRole, h.Santri.UpdateAdminKelas)
	protected.Post("/santri/:id/voice-note", akRole, middlewares.UploadRateLimit.Limit(), h.Santri.UploadVoiceNote)
	protected.Get("/santri/:id/voice-note/audio", akRole, h.Santri.ServeVoiceNote)
	protected.Post("/santri/:id/pindah", akRole, h.Santri.PindahKelas)
	protected.Get("/kelas", akRole, h.Kelas.Index)
	protected.Get("/kelas/:id", akRole, h.Kelas.Show)
	protected.Put("/kelas/:id/guru", akRole, h.Kelas.AssignGuru)
	protected.Put("/kelas/:id/pertemuan-terakhir", akRole, h.Kelas.SetPertemuanTerakhir)
	protected.Put("/kelas/:id/santri/:santriId/status", akRole, h.Kelas.UbahStatusSantri)
	protected.Put("/kelas/:id/status", akRole, h.Kelas.SetAktif)
	protected.Put("/kelas/:id/materi-individual", akRole, h.Kelas.SetMateriIndividual)
	protected.Put("/kelas/:id/level", akRole, h.Kelas.GantiLevel)
	protected.Put("/kelas/:id/jadwal", akRole, h.Kelas.GantiJadwal)
	protected.Post("/kelas/:id/ganti-level-santri", akRole, h.Kelas.GantiLevelSantri)
	protected.Delete("/kelas/:id", akRole, h.Kelas.Delete)

	// Produk, stok opname, dan CRM Mahasantri (admin_kelas + super_admin)
	protected.Get("/produk-crm", akRole, h.ProductCRM.Index)
	protected.Post("/produk-crm/produk", akRole, h.ProductCRM.CreateProduct)
	protected.Put("/produk-crm/produk/:id", akRole, h.ProductCRM.UpdateProduct)
	protected.Post("/produk-crm/produk/:id/batch", akRole, h.ProductCRM.CreateBatch)
	protected.Put("/produk-crm/batch/:id", akRole, h.ProductCRM.UpdateBatch)
	protected.Post("/produk-crm/stok", akRole, h.ProductCRM.AddStock)
	protected.Post("/produk-crm/opname", akRole, h.ProductCRM.Opname)
	protected.Post("/produk-crm/mahasantri/:santriID/produk", akRole, h.ProductCRM.AssignProduct)
	protected.Post("/produk-crm/mahasantri-produk/:id/batal", akRole, h.ProductCRM.CancelAssignment)

	// Keuangan routes (keuangan + super_admin)
	protected.Get("/keuangan", kuReadRole, h.Tagihan.Dashboard)
	protected.Get("/keuangan/tagihan", kuReadRole, h.Tagihan.List)
	protected.Get("/keuangan/tagihan/:id", kuReadRole, h.Tagihan.Detail)
	protected.Put("/keuangan/tagihan/:id/lunas", kuRole, h.Tagihan.MarkLunas)
	protected.Put("/keuangan/tagihan/:id/batal", kuRole, h.Tagihan.Batal)
	protected.Post("/keuangan/tagihan/:id/follow-up", kuReadRole, h.Tagihan.FollowUp)
	protected.Post("/keuangan/tagihan/sync", kuRole, h.Tagihan.Sync)
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
	protected.Post("/master/kode-kelas", akRole, h.Master.CreateKodeKelas)
	protected.Put("/master/kode-kelas/:id", akRole, h.Master.UpdateKodeKelas)
	protected.Delete("/master/kode-kelas/:id", akRole, h.Master.DeleteKodeKelas)

	// Guru & Koordinator Guru routes
	guruWorkflowRole := middlewares.RoleRequired(store, userService, "guru", "admin_kelas", "super_admin")
	guruPersonalReadRole := middlewares.RoleRequired(store, userService, "guru", "super_admin")
	guruWriteRole := middlewares.RoleRequired(store, userService, "guru")
	koordinatorRole := middlewares.RoleRequired(store, userService, "koordinator_guru", "super_admin")
	protected.Get("/guru", guruWorkflowRole, h.Guru.Dashboard)
	protected.Get("/guru/kelas", guruWorkflowRole, h.Guru.KelasSaya)
	protected.Get("/guru/kelas/:id", guruWorkflowRole, h.Guru.DetailKelas)
	protected.Get("/guru/kelas/:id/pertemuan/mulai", guruWorkflowRole, h.Pertemuan.FormMulai)
	protected.Post("/guru/kelas/:id/pertemuan/mulai", guruWorkflowRole, h.Pertemuan.Mulai)
	protected.Get("/guru/kelas/:id/pertemuan/:pid", guruWorkflowRole, h.Pertemuan.Detail)
	protected.Get("/guru/kelas/:id/pertemuan/:pid/selesai", guruWorkflowRole, h.Pertemuan.FormSelesai)
	protected.Post("/guru/kelas/:id/pertemuan/:pid/selesai", guruWorkflowRole, h.Pertemuan.Selesai)
	protected.Get("/guru/jadwal-pertemuan", guruWorkflowRole, h.JadwalPertemuan.Index)
	protected.Post("/guru/jadwal-pertemuan", guruWorkflowRole, h.JadwalPertemuan.Create)
	protected.Post("/guru/kelas/:id/jadwal-pertemuan/:jid/reschedule", guruWorkflowRole, h.JadwalPertemuan.Reschedule)
	protected.Post("/guru/kelas/:id/jadwal-pertemuan/:jid/badal", guruWorkflowRole, h.JadwalPertemuan.Badal)
	protected.Post("/guru/kelas/:id/jadwal-pertemuan/:jid/batal", guruWorkflowRole, h.JadwalPertemuan.Cancel)
	protected.Post("/guru/kelas/:id/jadwal-pertemuan/:jid/mulai", guruWorkflowRole, h.JadwalPertemuan.Start)
	protected.Get("/guru/kelas/:id/rekap", guruWorkflowRole, h.Pertemuan.RekapAbsensi)
	protected.Get("/guru/kelas/:id/riwayat-pertemuan", guruWorkflowRole, h.Pertemuan.Riwayat)
	protected.Put("/guru/kelas/:id/absensi/:aid", guruWorkflowRole, h.Pertemuan.EditAbsensi)
	protected.Get("/guru/kelas/:id/santri/:sid/riayah", guruWorkflowRole, h.Riayah.List)
	protected.Post("/guru/kelas/:id/santri/:sid/riayah", guruWorkflowRole, h.Riayah.Create)
	protected.Put("/guru/kelas/:id/santri/:sid/riayah/:rid", guruWorkflowRole, h.Riayah.Update)
	protected.Delete("/guru/kelas/:id/santri/:sid/riayah/:rid", guruWorkflowRole, h.Riayah.Delete)
	protected.Get("/guru/kelas/:id/santri/:sid/wa", guruWorkflowRole, h.Riayah.WALink)
	protected.Get("/guru/kelas/:id/wa", guruWorkflowRole, h.Riayah.BroadcastWA)
	protected.Post("/guru/tilawah", guruWriteRole, h.Guru.TilawahCheckin)
	protected.Delete("/guru/tilawah", guruWriteRole, h.Guru.TilawahUncheck)
	protected.Get("/guru/tsi", guruPersonalReadRole, h.TSI.Saya)

	// Riayah santri — koordinator guru hanya membaca.
	riayahReadRole := middlewares.RoleRequired(store, userService, "guru", "admin_kelas", "super_admin", "koordinator_guru")
	riayahWriteRole := middlewares.RoleRequired(store, userService, "guru", "admin_kelas", "super_admin")
	protected.Get("/guru/riayah", riayahReadRole, h.RiayahSantri.Index)
	protected.Get("/guru/santri/:sid", riayahReadRole, h.RiayahSantri.Profil)
	protected.Post("/guru/santri/:sid/catatan", riayahWriteRole, h.RiayahSantri.CatatanCreate)
	protected.Delete("/guru/santri/:sid/catatan/:rid", riayahWriteRole, h.RiayahSantri.CatatanDelete)

	// Koordinator Guru — dashboard & direktori guru
	protected.Get("/koordinator-guru", koordinatorRole, h.KoordinatorGuru.Dashboard)
	protected.Get("/koordinator-guru/guru", koordinatorRole, h.KoordinatorGuruDirectory.List)
	protected.Post("/koordinator-guru/guru", koordinatorRole, h.KoordinatorGuruDirectory.Create)
	protected.Get("/koordinator-guru/guru/:id", koordinatorRole, h.KoordinatorGuruDirectory.Detail)
	protected.Put("/koordinator-guru/guru/:id", koordinatorRole, h.KoordinatorGuruDirectory.Update)
	protected.Put("/koordinator-guru/guru/:id/kompetensi", koordinatorRole, h.KoordinatorFeatures.KompetensiSave)
	protected.Post("/koordinator-guru/guru/:id/riayah", koordinatorRole, h.KoordinatorFeatures.RiayahGuruCreate)
	protected.Put("/koordinator-guru/guru/:id/link", superAdminRole, h.KoordinatorGuruDirectory.LinkUser)
	protected.Delete("/koordinator-guru/guru/:id/link", superAdminRole, h.KoordinatorGuruDirectory.Unlink)

	// Koordinator Guru — monitoring operasional kelas
	protected.Get("/koordinator-guru/monitoring-kelas", koordinatorRole, h.KoordinatorMonitoring.Index)
	protected.Post("/koordinator-guru/monitoring-kelas/:kelasID/schedules/:scheduleID/reminder", koordinatorRole, h.KoordinatorMonitoring.SendReminder)
	protected.Post("/koordinator-guru/monitoring-kelas/:kelasID/schedules/:scheduleID/notes", koordinatorRole, h.KoordinatorMonitoring.AddNote)
	protected.Post("/koordinator-guru/monitoring-kelas/:kelasID/schedules/:scheduleID/reschedule", koordinatorRole, h.KoordinatorMonitoring.Reschedule)
	protected.Post("/koordinator-guru/monitoring-kelas/:kelasID/schedules/:scheduleID/substitute", koordinatorRole, h.KoordinatorMonitoring.AssignSubstitute)
	protected.Post("/koordinator-guru/monitoring-kelas/:kelasID/schedules/:scheduleID/cancel", koordinatorRole, h.KoordinatorMonitoring.Cancel)

	// Koordinator Guru — pembinaan
	protected.Get("/koordinator-guru/pembinaan", koordinatorRole, h.KoordinatorFeatures.PembinaanList)
	protected.Post("/koordinator-guru/pembinaan", koordinatorRole, h.KoordinatorFeatures.PembinaanCreate)
	protected.Put("/koordinator-guru/pembinaan/:id", koordinatorRole, h.KoordinatorFeatures.PembinaanUpdate)
	protected.Delete("/koordinator-guru/pembinaan/:id", koordinatorRole, h.KoordinatorFeatures.PembinaanDelete)
	protected.Get("/koordinator-guru/pembinaan/:id/absen", koordinatorRole, h.KoordinatorFeatures.PembinaanAbsenPage)
	protected.Post("/koordinator-guru/pembinaan/:id/absen", koordinatorRole, h.KoordinatorFeatures.PembinaanAbsenSave)
	protected.Get("/koordinator-guru/pembinaan/:id/absen/export", koordinatorRole, h.KoordinatorAttendanceExport.PembinaanExcel)

	// Koordinator Guru — rapat
	protected.Get("/koordinator-guru/rapat", koordinatorRole, h.KoordinatorFeatures.RapatList)
	protected.Post("/koordinator-guru/rapat", koordinatorRole, h.KoordinatorFeatures.RapatCreate)
	protected.Put("/koordinator-guru/rapat/:id", koordinatorRole, h.KoordinatorFeatures.RapatUpdate)
	protected.Delete("/koordinator-guru/rapat/:id", koordinatorRole, h.KoordinatorFeatures.RapatDelete)
	protected.Get("/koordinator-guru/rapat/:id/absen", koordinatorRole, h.KoordinatorFeatures.RapatAbsenPage)
	protected.Post("/koordinator-guru/rapat/:id/absen", koordinatorRole, h.KoordinatorFeatures.RapatAbsenSave)
	protected.Get("/koordinator-guru/rapat/:id/absen/export", koordinatorRole, h.KoordinatorAttendanceExport.RapatExcel)
	protected.Get("/koordinator-guru/riwayat-absensi", koordinatorRole, h.KoordinatorFeatures.RiwayatAbsensi)

	// Koordinator Guru — kunjungan kelas
	protected.Get("/koordinator-guru/kunjungan", koordinatorRole, h.KoordinatorFeatures.KunjunganList)
	protected.Post("/koordinator-guru/kunjungan", koordinatorRole, h.KoordinatorFeatures.KunjunganCreate)
	protected.Put("/koordinator-guru/kunjungan/:id", koordinatorRole, h.KoordinatorFeatures.KunjunganUpdate)
	protected.Delete("/koordinator-guru/kunjungan/:id", koordinatorRole, h.KoordinatorFeatures.KunjunganDelete)

	// Koordinator Guru — Kalam Bersanad
	protected.Get("/koordinator-guru/kalam", koordinatorRole, h.KoordinatorFeatures.KalamList)
	protected.Post("/koordinator-guru/kalam", koordinatorRole, h.KoordinatorFeatures.KalamCreate)
	protected.Put("/koordinator-guru/kalam/:id", koordinatorRole, h.KoordinatorFeatures.KalamUpdate)
	protected.Delete("/koordinator-guru/kalam/:id", koordinatorRole, h.KoordinatorFeatures.KalamDelete)
	protected.Get("/koordinator-guru/kalam/:id/share", koordinatorRole, h.KoordinatorFeatures.KalamSharePage)
	protected.Post("/koordinator-guru/kalam/:id/share", koordinatorRole, h.KoordinatorFeatures.KalamShareToggle)

	// Koordinator Guru — template WA
	protected.Get("/koordinator-guru/wa-template", koordinatorRole, h.KoordinatorFeatures.WaTemplateList)
	protected.Post("/koordinator-guru/wa-template", koordinatorRole, h.KoordinatorFeatures.WaTemplateCreate)
	protected.Put("/koordinator-guru/wa-template/:id", koordinatorRole, h.KoordinatorFeatures.WaTemplateUpdate)
	protected.Delete("/koordinator-guru/wa-template/:id", koordinatorRole, h.KoordinatorFeatures.WaTemplateDelete)

	// Koordinator Guru — todo
	protected.Get("/koordinator-guru/todo", koordinatorRole, h.KoordinatorFeatures.TodoList)
	protected.Post("/koordinator-guru/todo", koordinatorRole, h.KoordinatorFeatures.TodoCreate)
	protected.Put("/koordinator-guru/todo/:id", koordinatorRole, h.KoordinatorFeatures.TodoUpdate)
	protected.Put("/koordinator-guru/todo/:id/status", koordinatorRole, h.KoordinatorFeatures.TodoStatus)
	protected.Delete("/koordinator-guru/todo/:id", koordinatorRole, h.KoordinatorFeatures.TodoDelete)

	// Koordinator Guru — TSI (penilaian & rekap)
	protected.Get("/koordinator-guru/tsi", koordinatorRole, h.TSI.Rekap)
	protected.Get("/koordinator-guru/tsi/:id", koordinatorRole, h.TSI.Penilaian)
	protected.Post("/koordinator-guru/tsi/:id", koordinatorRole, h.TSI.SaveNilai)
	protected.Post("/koordinator-guru/tsi/:id/finalize", koordinatorRole, h.TSI.Finalize)
	protected.Post("/koordinator-guru/tsi/:id/reopen", koordinatorRole, h.TSI.Reopen)
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
