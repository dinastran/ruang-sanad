package handlers

import (
	"bytes"
	"database/sql"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestLaporanKeuanganExportExcel(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO santri (nama, status) VALUES ('Ahmad', 'aktif')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tagihan (santri_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, status, tanggal_bayar, metode) VALUES (1, 1, 4, 150000, '2026-03-02', 'lunas', '2026-03-05', 'Tunai')`)
	require.NoError(t, err)

	h := NewLaporanHandler(services.NewLaporanService(queries.NewQuerier(db), nil, nil), nil, nil)
	app := fiber.New()
	app.Get("/export", h.Export)

	resp, err := app.Test(httptest.NewRequest("GET", "/export?dari=2026-03-01&sampai=2026-03-31", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get(fiber.HeaderContentDisposition), "laporan-keuangan-2026-03-01_2026-03-31.xlsx")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, []string{"Ringkasan", "Pembayaran SPP", "Pengeluaran", "Per Santri", "Tidak Lanjut"}, f.GetSheetList())
	nominal, err := f.GetCellValue("Pembayaran SPP", "G2")
	require.NoError(t, err)
	require.Equal(t, "150,000", nominal)

	resp, err = app.Test(httptest.NewRequest("GET", "/export?dari=2026-03-31&sampai=2026-03-01", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
