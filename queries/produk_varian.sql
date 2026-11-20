-- name: CreateProdukVarian :one
INSERT INTO produk_varian (produk_id, nama, harga, paket_kode, urutan, is_aktif, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateProdukVarian :exec
UPDATE produk_varian SET nama = ?, harga = ?, paket_kode = ?, urutan = ?, is_aktif = ?, updated_at = ?
WHERE id = ?;

-- name: GetProdukVarian :one
SELECT * FROM produk_varian WHERE id = ?;

-- name: ListProdukVarianByProduk :many
SELECT * FROM produk_varian WHERE produk_id = ? ORDER BY urutan, id;

-- name: ListAllProdukVarian :many
SELECT * FROM produk_varian ORDER BY produk_id, urutan, id;
