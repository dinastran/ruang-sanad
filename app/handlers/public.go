package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/templates"
)

type PublicHandler struct {
	authService    *services.AuthService
	userService    *services.UserService
	inertiaService *services.InertiaService
	assetService   *services.AssetService
}

func NewPublicHandler(authService *services.AuthService, userService *services.UserService, inertiaService *services.InertiaService, assetService *services.AssetService) *PublicHandler {
	return &PublicHandler{
		authService:    authService,
		userService:    userService,
		inertiaService: inertiaService,
		assetService:   assetService,
	}
}

// Index renders the home page
func (h *PublicHandler) Index(c *fiber.Ctx) error {
	isDev := h.assetService.IsDevelopment()
	viteURL := ""
	if isDev {
		viteURL = h.assetService.GetViteServerURL()
	}
	mainCSS := h.assetService.GetMainCSS()

	c.Set("Content-Type", "text/html; charset=utf-8")
	return templates.LandingPage("Welcome to Laju", isDev, viteURL, mainCSS).Render(c.Context(), c.Response().BodyWriter())
}

// About renders the about page
func (h *PublicHandler) About(c *fiber.Ctx) error {
	return h.inertiaService.Render(c, "About", fiber.Map{
		"Title": "About Laju",
	})
}
