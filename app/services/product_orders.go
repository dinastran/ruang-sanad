package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrStokOrderTidakCukup = errors.New("stok produk tidak mencukupi")

type ProductOrderService struct {
	querier *queries.Querier
}

func NewProductOrderService(querier *queries.Querier) *ProductOrderService {
	return &ProductOrderService{querier: querier}
}

func validProductOrderSource(value string) bool {
	switch value {
	case "whatsapp", "instagram", "marketplace", "offline", "lainnya":
		return true
	default:
		return false
	}
}

func validateProductOrderBatch(ctx context.Context, q *queries.Querier, product models.Product, batchID int64) (sql.NullInt64, error) {
	activeBatches, err := q.CountActiveProdukBatches(ctx, product.ID)
	if err != nil {
		return sql.NullInt64{}, err
	}
	if activeBatches > 0 && batchID <= 0 {
		return sql.NullInt64{}, errors.New("batch/edisi wajib dipilih untuk produk ini")
	}
	if batchID <= 0 {
		return sql.NullInt64{}, nil
	}
	batch, err := q.GetProdukBatch(ctx, batchID)
	if err != nil {
		return sql.NullInt64{}, errors.New("batch/edisi tidak ditemukan")
	}
	if batch.ProdukID != product.ID {
		return sql.NullInt64{}, errors.New("batch/edisi tidak sesuai dengan produk")
	}
	if !batch.IsAktif {
		return sql.NullInt64{}, errors.New("batch/edisi sudah tidak aktif")
	}
	return nullBatch(batchID), nil
}

func (s *ProductOrderService) Create(req models.CreateProductOrderRequest, userID int64) (int64, error) {
	req.NamaPembeli = strings.TrimSpace(req.NamaPembeli)
	req.NoWA = strings.TrimSpace(req.NoWA)
	req.Sumber = strings.ToLower(strings.TrimSpace(req.Sumber))
	req.Catatan = strings.TrimSpace(req.Catatan)

	if !validDate(req.Tanggal) {
		return 0, errors.New("tanggal order wajib berformat YYYY-MM-DD")
	}
	if req.NamaPembeli == "" {
		return 0, errors.New("nama pembeli wajib diisi")
	}
	if !validProductOrderSource(req.Sumber) {
		return 0, errors.New("sumber order tidak valid")
	}
	if len(req.Items) == 0 {
		return 0, errors.New("minimal satu produk harus dipilih")
	}

	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	type validatedItem struct {
		request models.CreateProductOrderItemRequest
		product models.Product
		batch   sql.NullInt64
	}
	validated := make([]validatedItem, 0, len(req.Items))
	reservedStock := make(map[string]int64)

	for _, item := range req.Items {
		if item.ProdukID <= 0 {
			return 0, errors.New("produk wajib dipilih")
		}
		if item.Qty <= 0 {
			return 0, errors.New("qty order harus lebih dari nol")
		}
		product, err := q.GetProduk(ctx, item.ProdukID)
		if err != nil {
			return 0, errors.New("produk tidak ditemukan")
		}
		if !product.IsAktif {
			return 0, ErrProdukTidakAktif
		}
		batch, err := validateProductOrderBatch(ctx, q, product, item.ProdukBatchID)
		if err != nil {
			return 0, err
		}
		if product.TrackStok {
			key := fmt.Sprintf("%d:%d", product.ID, item.ProdukBatchID)
			stock, err := q.CurrentProdukStock(ctx, product.ID, batch)
			if err != nil {
				return 0, err
			}
			needed := reservedStock[key] + item.Qty
			if stock < needed {
				return 0, fmt.Errorf("%w: %s tersedia %d, dibutuhkan %d", ErrStokOrderTidakCukup, product.Nama, stock, needed)
			}
			reservedStock[key] = needed
		}
		validated = append(validated, validatedItem{request: item, product: product, batch: batch})
	}

	orderID, err := q.CreateProductOrder(ctx, req.Tanggal, req.NamaPembeli, req.NoWA, req.Sumber, req.Catatan, userID)
	if err != nil {
		return 0, err
	}
	orderNo := fmt.Sprintf("ORD-%06d", orderID)
	if err := q.UpdateProductOrderNumber(ctx, orderID, orderNo); err != nil {
		return 0, err
	}

	for _, item := range validated {
		itemID, err := q.CreateProductOrderItem(ctx, orderID, item.product.ID, item.batch, item.request.Qty)
		if err != nil {
			return 0, err
		}
		if item.product.TrackStok {
			note := fmt.Sprintf("Order manual %s - %s", orderNo, req.NamaPembeli)
			if _, err := q.InsertStockMutation(
				ctx,
				item.product.ID,
				item.batch,
				"pembelian",
				-item.request.Qty,
				"product_order_item",
				nullRef(itemID),
				note,
				userID,
			); err != nil {
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

func (s *ProductOrderService) List(limit int64) ([]models.ProductOrder, error) {
	ctx := context.Background()
	orders, err := s.querier.ListProductOrders(ctx, limit)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		items, err := s.querier.ListProductOrderItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}
	return orders, nil
}

func (s *ProductOrderService) Cancel(id, userID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	order, err := q.GetProductOrder(ctx, id)
	if err != nil {
		return err
	}
	if order.Status != "aktif" {
		return errors.New("order sudah dibatalkan")
	}
	items, err := q.ListProductOrderItems(ctx, id)
	if err != nil {
		return err
	}
	affected, err := q.CancelProductOrder(ctx, id, userID)
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("order sudah berubah, muat ulang data")
	}

	for _, item := range items {
		if !item.TrackStok {
			continue
		}
		batch := nullBatch(item.ProdukBatchID)
		note := fmt.Sprintf("Pembatalan order %s - %s", order.OrderNo, order.NamaPembeli)
		if _, err := q.InsertStockMutation(
			ctx,
			item.ProdukID,
			batch,
			"pembatalan",
			item.Qty,
			"product_order_item",
			nullRef(item.ID),
			note,
			userID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
