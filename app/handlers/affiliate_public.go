package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

// AffiliatePublicHandler serves affiliate links /r/:kode (no login).
type AffiliatePublicHandler struct {
	aff *services.AffiliateService
}

func NewAffiliatePublicHandler(aff *services.AffiliateService) *AffiliatePublicHandler {
	return &AffiliatePublicHandler{aff: aff}
}

func (h *AffiliatePublicHandler) Redirect(c *fiber.Ctx) error {
	ctx := c.UserContext()
	res, err := h.aff.TrackClick(ctx, c.Params("kode"), session.ClientIP(c), c.Get(fiber.HeaderUserAgent), c.Cookies(services.AffCookieName))
	if err != nil {
		return err
	}
	if res.Cookie != "" {
		c.Cookie(&fiber.Cookie{
			Name: services.AffCookieName, Value: res.Cookie, Path: "/", MaxAge: res.MaxAge,
			Expires: time.Now().Add(time.Duration(res.MaxAge) * time.Second), HTTPOnly: true,
			Secure: c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https", SameSite: "Lax",
		})
	}
	kode := ""
	if res.Found {
		kode = res.Kode
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Redirect(h.aff.Destination(ctx, c.Query("ke"), kode), fiber.StatusFound)
}
