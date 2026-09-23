package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
)

type ProductCRMHandler struct {
	service        *services.ProductCRMService
	masterService  *services.MasterService
	store          *session.Store
	inertiaService *services.InertiaService
}

func NewProductCRMHandler(service *services.ProductCRMService, masterService *services.MasterService, store *session.Store, inertiaService *services.InertiaService) *ProductCRMHandler {
	return &ProductCRMHandler{
		service:        service,
		masterService:  masterService,
		store:          store,
		inertiaService: inertiaService,
	}
}

func productCRMIntQuery(c *fiber.Ctx, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key, "0"), 10, 64)
	return value
}

func (h *ProductCRMHandler) Index(c *fiber.Ctx) error {
	sess, _ := h.store.Get(c)
	user := sessionUser(sess)
	page := productCRMIntQuery(c, "page")
	if page <= 0 {
		page = 1
	}
	filters := models.ProductCRMFilters{
		Search:           c.Query("search"),
		Angkatan:         c.Query("angkatan"),
		Status:           c.Query("status"),
		HasProductID:     productCRMIntQuery(c, "has_product_id"),
		MissingProductID: productCRMIntQuery(c, "missing_product_id"),
		BatchID:          productCRMIntQuery(c, "batch_id"),
		Page:             page,
		Limit:            25,
	}

	products, err := h.service.Products()
	if err != nil {
		return h.inertiaService.Render(c, "app/ProductCRM", fiber.Map{"user": user, "error": "Gagal memuat produk"})
	}

	dashboardFilters := models.ProductCRMDashboardFilters{
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
		ProductID: productCRMIntQuery(c, "product_id"),
		BatchID:   productCRMIntQuery(c, "dashboard_batch_id"),
		Category:  c.Query("category"),
		Angkatan:  c.Query("angkatan"),
		Status:    c.Query("status"),
	}
	dashboard, normalizedDashboardFilters, dashboardErr := h.service.Dashboard(dashboardFilters)

	var drilldown *models.ProductCRMDrilldown
	var drilldownErr error
	drilldownMode := strings.TrimSpace(c.Query("drilldown"))
	if drilldownMode != "" {
		drillPage := productCRMIntQuery(c, "drill_page")
		if drillPage <= 0 {
			drillPage = 1
		}
		drilldown, _, drilldownErr = h.service.DashboardDrilldown(
			normalizedDashboardFilters,
			drilldownMode,
			c.Query("drill_period"),
			drillPage,
			20,
		)
	}

	crm, err := h.service.ListCRM(filters)
	if err != nil {
		return h.inertiaService.Render(c, "app/ProductCRM", fiber.Map{"user": user, "products": products, "error": "Gagal memuat CRM Mahasantri"})
	}
	mutations, _ := h.service.Mutations(100)
	angkatan, _ := h.masterService.ListAngkatan()

	props := fiber.Map{
		"user":      user,
		"products":  products,
		"crm":       crm.Data,
		"total":     crm.Total,
		"page":      page,
		"limit":     int64(25),
		"mutations": mutations,
		"angkatan":  angkatan,
		"tab":       c.Query("tab", "dashboard"),
		"filters": fiber.Map{
			"search":             filters.Search,
			"angkatan":           filters.Angkatan,
			"status":             filters.Status,
			"has_product_id":     filters.HasProductID,
			"missing_product_id": filters.MissingProductID,
			"batch_id":           filters.BatchID,
		},
		"dashboard": dashboard,
		"dashboard_filters": normalizedDashboardFilters,
		"drilldown": drilldown,
		"drilldown_mode": drilldownMode,
		"drilldown_period": c.Query("drill_period"),
	}
	if drilldownErr != nil {
		props["drilldown_error"] = "Gagal memuat data Mahasantri: " + drilldownErr.Error()
	}
	if dashboardErr != nil {
		props["dashboard_error"] = "Gagal memuat dashboard intelligence: " + dashboardErr.Error()
	}
	return h.inertiaService.Render(c, "app/ProductCRM", props)
}

func (h *ProductCRMHandler) CreateProduct(c *fiber.Ctx) error {
	var req models.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data produk tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	if err := h.service.CreateProduct(req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah produk: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Produk berhasil ditambahkan")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
}

func (h *ProductCRMHandler) UpdateProduct(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	var req models.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data produk tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	if err := h.service.UpdateProduct(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal memperbarui produk: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Produk berhasil diperbarui")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
}

func (h *ProductCRMHandler) CreateBatch(c *fiber.Ctx) error {
	productID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	var req models.CreateProductBatchRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data batch tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	if err := h.service.CreateBatch(productID, req); err != nil {
		h.store.Flash(c, "error", "Gagal menambah batch/edisi: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Batch/edisi berhasil ditambahkan")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
}

func (h *ProductCRMHandler) UpdateBatch(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	var req models.UpdateProductBatchRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data batch tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
	}
	if err := h.service.UpdateBatch(id, req); err != nil {
		h.store.Flash(c, "error", "Gagal memperbarui batch/edisi: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Batch/edisi berhasil diperbarui")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=produk")
}

func (h *ProductCRMHandler) AddStock(c *fiber.Ctx) error {
	var req models.StockEntryRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data stok tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=stok")
	}
	if err := h.service.AddStock(req, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal menambah stok: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Stok berhasil ditambahkan")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=stok")
}

func (h *ProductCRMHandler) Opname(c *fiber.Ctx) error {
	var req models.StockOpnameRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data opname tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=stok")
	}
	if err := h.service.Opname(req, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal menyimpan opname: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Stok opname berhasil dicatat")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=stok")
}

func (h *ProductCRMHandler) AssignProduct(c *fiber.Ctx) error {
	santriID, err := strconv.ParseInt(c.Params("santriID"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=crm")
	}
	var req models.AssignProductRequest
	if err := c.BodyParser(&req); err != nil {
		h.store.Flash(c, "error", "Data transaksi tidak valid")
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=crm")
	}
	if err := h.service.AssignProduct(santriID, req, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal mencatat produk: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Produk Mahasantri berhasil dicatat")
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=crm")
}

func (h *ProductCRMHandler) CancelAssignment(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return h.inertiaService.Redirect(c, "/app/produk-crm?tab=crm")
	}
	if err := h.service.CancelAssignment(id, c.Locals("user_id").(int64)); err != nil {
		h.store.Flash(c, "error", "Gagal membatalkan transaksi: "+err.Error())
	} else {
		h.store.Flash(c, "success", "Transaksi dibatalkan dan stok dikembalikan bila relevan")
	}
	returnTo := c.Query("return_to")
	if strings.HasPrefix(returnTo, "/app/") {
		return h.inertiaService.Redirect(c, returnTo)
	}
	return h.inertiaService.Redirect(c, "/app/produk-crm?tab=crm")
}
