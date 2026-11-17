package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
)

// PWAHandler melayani service worker aplikasi Tasbih di root (/sw.js) agar
// scope-nya mencakup seluruh situs.
type PWAHandler struct {
	pwa *services.PWAService
}

func NewPWAHandler(pwa *services.PWAService) *PWAHandler {
	return &PWAHandler{pwa: pwa}
}

func (h *PWAHandler) ServiceWorker(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("Service-Worker-Allowed", "/")
	return c.SendString(h.pwa.ServiceWorker())
}
