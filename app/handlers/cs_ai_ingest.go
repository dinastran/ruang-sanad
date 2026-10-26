package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
)

const csAIIngestMaxBody = 256 * 1024

// CsAiIngestHandler receives lead events from hermes-cs-bridge
// (POST /integrations/cs-ai/events). It sits outside /app: no session or
// CSRF; the bridge authenticates with a bearer token from .env.
type CsAiIngestHandler struct {
	leads     *services.LeadService
	jadwal    *services.JadwalPenawaranService
	tokenHash [32]byte
	enabled   bool
	allowed   []*net.IPNet
}

func NewCsAiIngestHandler(leads *services.LeadService, token string, allowedCIDRs []string) *CsAiIngestHandler {
	h := &CsAiIngestHandler{leads: leads, enabled: token != ""}
	h.tokenHash = sha256.Sum256([]byte(token))
	for _, cidr := range allowedCIDRs {
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("CS_BRIDGE_ALLOWED_CIDRS: entry diabaikan", "cidr", cidr, "error", err)
			continue
		}
		h.allowed = append(h.allowed, network)
	}
	return h
}

// WithJadwal enables GET /integrations/cs-ai/jadwal.
func (h *CsAiIngestHandler) WithJadwal(j *services.JadwalPenawaranService) *CsAiIngestHandler {
	h.jadwal = j
	return h
}

// Jadwal lists the open class schedules with remaining seats per gender, for
// the CS AI prompt. Guard runs first (token + direct address).
func (h *CsAiIngestHandler) Jadwal(c *fiber.Ctx) error {
	if h.jadwal == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	list, err := h.jadwal.Open(c.UserContext())
	if err != nil {
		slog.Error("cs-ai jadwal gagal", "error", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "gagal membaca jadwal"})
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(fiber.Map{"jadwal": list})
}

func (h *CsAiIngestHandler) ipAllowed(ip net.IP) bool {
	for _, network := range h.allowed {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (h *CsAiIngestHandler) validToken(header string) bool {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return false
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return subtle.ConstantTimeCompare(sum[:], h.tokenHash[:]) == 1
}

// Guard authenticates the bridge before anything else (including the rate
// limiter), so unauthenticated traffic cannot use up the bridge's quota.
func (h *CsAiIngestHandler) Guard(c *fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	return c.Next()
}

func (h *CsAiIngestHandler) Events(c *fiber.Ctx) error {
	if err := h.authorize(c); err != nil {
		return err
	}
	return h.events(c)
}

func (h *CsAiIngestHandler) authorize(c *fiber.Ctx) error {
	if !h.enabled {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	// Use the direct connection address, never forwarded headers. A request
	// that came through the public reverse proxy carries X-Forwarded-For /
	// X-Real-IP even though its connection is from loopback, so it is refused:
	// the bridge always calls the dashboard directly.
	if !h.ipAllowed(c.Context().RemoteIP()) || c.Get("X-Forwarded-For") != "" || c.Get("X-Real-IP") != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	if !h.validToken(c.Get(fiber.HeaderAuthorization)) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return nil
}

func (h *CsAiIngestHandler) events(c *fiber.Ctx) error {
	if c.Request().Header.ContentLength() > csAIIngestMaxBody {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "body maksimal 256 KB"})
	}
	body := c.Body()
	if len(body) > csAIIngestMaxBody {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "body maksimal 256 KB"})
	}
	var req models.LeadIngestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "JSON tidak valid"})
	}
	if len(req.Events) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "events kosong"})
	}
	if len(req.Events) > services.MaxLeadIngestEvents {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "maksimal 100 event per request"})
	}
	result, err := h.leads.IngestEvents(c.UserContext(), req.Events)
	if err != nil {
		slog.Error("cs-ai ingest gagal", "error", err, "accepted", result.Accepted)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "gagal menyimpan event, kirim ulang"})
	}
	if len(result.Rejected) > 0 {
		slog.Warn("cs-ai ingest: event ditolak", "jumlah", len(result.Rejected), "contoh", result.Rejected[0].Error)
	}
	return c.JSON(result)
}
