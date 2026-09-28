package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type ProductOrderHandler struct {
	service        *services.ProductOrderService
	productService *services.ProductCRMService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewProductOrderHandler(service *services.ProductOrderService, productService *services.ProductCRMService, store *session.Store, inertiaService *services.InertiaService) *ProductOrderHandler {
	return &ProductOrderHandler{
		service:        service,
		productService: productService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func (h *ProductOrderHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)

	products, err := h.productService.Products()
	if err != nil {
		return h.inertiaService.Render(c, "app/ProductCRM", fiber.Map{
			"user": user,
			"tab":  "orders",
			"error": "Gagal memuat produk",
		})
	}
	orders, err := h.service.List(100)
	if err != nil {
		return h.inertiaService.Render(c, "app/ProductCRM", fiber.Map{
			"user":     user,
			"products": products,
			"tab":      "orders",
			"error":    "Gagal memuat order manual: " + err.Error(),
		})
	}

	return h.inertiaService.Render(c, "app/ProductCRM", fiber.Map{
		"user":     user,
		"products": products,
		"orders":   orders,
		"tab":      "orders",
	})
}

func (h *ProductOrderHandler) Create(c *fiber.Ctx) error {
	var req models.CreateProductOrderRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data order tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm/orders")
	}
	if _, err := h.service.Create(req, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal menyimpan order: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Order berhasil dicatat dan stok diperbarui")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm/orders")
}

func (h *ProductOrderHandler) Cancel(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		h.store.Flash(c, "error", "Order tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm/orders")
	}
	if err := h.service.Cancel(id, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal membatalkan order: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Order dibatalkan dan stok dikembalikan")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm/orders")
}
