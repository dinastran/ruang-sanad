package queries

import (
	"context"
	"database/sql"

	"github.com/maulanashalihin/laju-go/app/models"
)

type productOrderScanner interface {
	Scan(dest ...interface{}) error
}

func scanProductOrder(s productOrderScanner) (models.ProductOrder, error) {
	var out models.ProductOrder
	err := s.Scan(
		&out.ID,
		&out.OrderNo,
		&out.Tanggal,
		&out.NamaPembeli,
		&out.NoWA,
		&out.Sumber,
		&out.Status,
		&out.Catatan,
		&out.DicatatOleh,
		&out.DibatalkanOleh,
		&out.CreatedAt,
	)
	return out, err
}

func (q *Queries) CreateProductOrder(ctx context.Context, tanggal, namaPembeli, noWA, sumber, catatan string, userID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO product_orders
			(order_no, tanggal, nama_pembeli, no_wa, sumber, status, catatan, dicatat_oleh)
		VALUES (NULL, ?, ?, ?, ?, 'aktif', ?, ?)`,
		tanggal, namaPembeli, noWA, sumber, catatan, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) UpdateProductOrderNumber(ctx context.Context, id int64, orderNo string) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE product_orders
		SET order_no = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, orderNo, id)
	return err
}

func (q *Queries) CreateProductOrderItem(ctx context.Context, orderID, productID int64, batchID sql.NullInt64, qty int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO product_order_items (order_id, produk_id, produk_batch_id, qty)
		VALUES (?, ?, ?, ?)`,
		orderID, productID, nullableInt64(batchID), qty,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) GetProductOrder(ctx context.Context, id int64) (models.ProductOrder, error) {
	return scanProductOrder(q.db.QueryRowContext(ctx, `
		SELECT po.id, COALESCE(po.order_no, ''), strftime('%Y-%m-%d', po.tanggal),
		       po.nama_pembeli, po.no_wa, po.sumber, po.status, po.catatan,
		       COALESCE(creator.name, ''), COALESCE(canceller.name, ''),
		       strftime('%Y-%m-%d %H:%M', po.created_at)
		FROM product_orders po
		LEFT JOIN users creator ON creator.id = po.dicatat_oleh
		LEFT JOIN users canceller ON canceller.id = po.dibatalkan_oleh
		WHERE po.id = ?`, id))
}

func (q *Queries) ListProductOrders(ctx context.Context, limit int64) ([]models.ProductOrder, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT po.id, COALESCE(po.order_no, ''), strftime('%Y-%m-%d', po.tanggal),
		       po.nama_pembeli, po.no_wa, po.sumber, po.status, po.catatan,
		       COALESCE(creator.name, ''), COALESCE(canceller.name, ''),
		       strftime('%Y-%m-%d %H:%M', po.created_at)
		FROM product_orders po
		LEFT JOIN users creator ON creator.id = po.dicatat_oleh
		LEFT JOIN users canceller ON canceller.id = po.dibatalkan_oleh
		ORDER BY po.tanggal DESC, po.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductOrder, 0)
	for rows.Next() {
		item, err := scanProductOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ListProductOrderItems(ctx context.Context, orderID int64) ([]models.ProductOrderItem, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT poi.id, poi.order_id, poi.produk_id, poi.produk_batch_id,
		       p.nama, COALESCE(pb.nama, ''), p.kategori, p.track_stok, poi.qty
		FROM product_order_items poi
		JOIN produk p ON p.id = poi.produk_id
		LEFT JOIN produk_batch pb ON pb.id = poi.produk_batch_id
		WHERE poi.order_id = ?
		ORDER BY poi.id ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductOrderItem, 0)
	for rows.Next() {
		var item models.ProductOrderItem
		var batchID sql.NullInt64
		var track int64
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProdukID, &batchID,
			&item.ProdukNama, &item.BatchNama, &item.Kategori, &track, &item.Qty,
		); err != nil {
			return nil, err
		}
		if batchID.Valid {
			item.ProdukBatchID = batchID.Int64
		}
		item.TrackStok = track == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) CancelProductOrder(ctx context.Context, id, userID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE product_orders
		SET status = 'dibatalkan',
		    dibatalkan_at = CURRENT_TIMESTAMP,
		    dibatalkan_oleh = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'aktif'`, userID, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
