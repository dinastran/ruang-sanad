package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestRegistrationSchemaRoundTrip(t *testing.T) {
	db, q := setupSantriDatabase(t)
	ctx := t.Context()
	now := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)

	res, err := db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Tahsin Reguler', 'program', 150000)`)
	require.NoError(t, err)
	produkID, _ := res.LastInsertId()
	_, err = db.Exec(`UPDATE lead_programs SET produk_id = ? WHERE id = 1`, produkID)
	require.NoError(t, err)

	varian, err := q.CreateProdukVarian(ctx, queries.CreateProdukVarianParams{
		ProdukID: produkID, Nama: "2x / pekan", Harga: 350000, PaketKode: "2x", Urutan: 1, IsAktif: 1, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	var sourceID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM lead_sources ORDER BY id LIMIT 1`).Scan(&sourceID))
	linkParams := queries.CreateRegistrationLinkParams{
		Slug: "tahsin-reguler", Judul: "Daftar Tahsin", ProgramID: 1, LeadSourceID: sourceID,
		Mode: "short", AfterSubmit: "bayar", FieldConfig: "{}", IzinkanManual: 1, IsAktif: 1, CreatedAt: now, UpdatedAt: now,
	}
	link, err := q.CreateRegistrationLink(ctx, linkParams)
	require.NoError(t, err)
	_, err = q.CreateRegistrationLink(ctx, linkParams)
	require.Error(t, err, "slug must be unique")

	got, err := q.GetRegistrationLinkBySlug(ctx, "tahsin-reguler")
	require.NoError(t, err)
	require.Equal(t, link.ID, got.ID)

	day := "2026-11-19"
	for range 2 {
		require.NoError(t, q.BumpRegistrationStat(ctx, queries.BumpRegistrationStatParams{LinkID: link.ID, Tanggal: day, Klik: 1}))
	}
	stats, err := q.ListRegistrationStatsSince(ctx, "2026-11-01")
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.EqualValues(t, 2, stats[0].Klik)

	lead, err := q.CreateLead(ctx, queries.CreateLeadParams{WaChatID: "6281234567890@s.whatsapp.net", Phone: "6281234567890", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)
	byPhone, err := q.GetLeadByPhone(ctx, "6281234567890")
	require.NoError(t, err)
	require.Equal(t, lead.ID, byPhone.ID)

	reg, err := q.CreateLeadRegistration(ctx, queries.CreateLeadRegistrationParams{
		LeadID: lead.ID, LinkID: link.ID, Ref: "ref-abc", Payload: "{}", CreatedAt: now,
	})
	require.NoError(t, err)
	byRef, err := q.GetLeadRegistrationByRef(ctx, "ref-abc")
	require.NoError(t, err)
	require.Equal(t, reg.ID, byRef.ID)

	pay, err := q.CreatePayment(ctx, queries.CreatePaymentParams{
		LeadID: lead.ID, LinkID: sql.NullInt64{Int64: link.ID, Valid: true}, ProdukVarianID: sql.NullInt64{Int64: varian.ID, Valid: true},
		Metode: "manual", Amount: 350000, Status: "menunggu_verifikasi", BuktiPath: "x.png", CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	pays, err := q.ListPaymentsByLead(ctx, lead.ID)
	require.NoError(t, err)
	require.Len(t, pays, 1)
	require.Equal(t, pay.ID, pays[0].ID)

	settings, err := q.GetRegistrationSettings(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, settings.PaymentExpiryHours)
}
