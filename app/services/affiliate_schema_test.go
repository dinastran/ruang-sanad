package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestAffiliateSchemaRoundTrip(t *testing.T) {
	db, q := setupSantriDatabase(t)
	ctx := t.Context()
	now := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)

	a, err := q.CreateAffiliate(ctx, queries.CreateAffiliateParams{
		Kode: "UST-AHMAD", Tipe: "eksternal", Nama: "Ahmad", NoWa: "6281200000000", Status: "aktif", CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	_, err = q.CreateAffiliate(ctx, queries.CreateAffiliateParams{Kode: "UST-AHMAD", Tipe: "eksternal", Nama: "X", Status: "aktif", CreatedAt: now, UpdatedAt: now})
	require.Error(t, err, "kode unik")
	got, err := q.GetAffiliateByKode(ctx, "UST-AHMAD")
	require.NoError(t, err)
	require.Equal(t, a.ID, got.ID)

	st, err := q.GetAffiliateSettings(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 60, st.MasaBerlakuHari)
	require.EqualValues(t, 14, st.MasaTahanHari)
	require.True(t, st.KomponenUjrohID.Valid, "komponen ujroh 'Komisi Affiliate' di-seed")

	_, err = q.UpsertAffiliateRule(ctx, queries.UpsertAffiliateRuleParams{Scope: "global", Jenis: "persen", Nilai: 1000, Pola: "sekali", IsAktif: 1, UpdatedAt: now})
	require.NoError(t, err)
	_, err = q.UpsertAffiliateRule(ctx, queries.UpsertAffiliateRuleParams{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali", IsAktif: 1, UpdatedAt: now})
	require.NoError(t, err, "upsert global menimpa")
	rules, err := q.ListAffiliateRules(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, "flat", rules[0].Jenis)

	require.NoError(t, q.BumpAffiliateClick(ctx, queries.BumpAffiliateClickParams{AffiliateID: a.ID, Tanggal: "2026-11-19"}))
	require.NoError(t, q.BumpAffiliateClick(ctx, queries.BumpAffiliateClickParams{AffiliateID: a.ID, Tanggal: "2026-11-19"}))

	lead, err := q.CreateLead(ctx, queries.CreateLeadParams{WaChatID: "1@s.whatsapp.net", Phone: "6281", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)
	n, err := q.InsertLeadAffiliate(ctx, queries.InsertLeadAffiliateParams{LeadID: lead.ID, AffiliateID: a.ID, Sumber: "cookie", DikunciAt: now})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	n, err = q.InsertLeadAffiliate(ctx, queries.InsertLeadAffiliateParams{LeadID: lead.ID, AffiliateID: a.ID, Sumber: "manual", DikunciAt: now})
	require.NoError(t, err)
	require.Zero(t, n, "first-click: insert kedua diabaikan")

	c, err := q.InsertAffiliateCommission(ctx, queries.InsertAffiliateCommissionParams{
		AffiliateID: a.ID, LeadID: lead.ID, Sumber: "pendaftaran", SumberID: lead.ID, BulanKe: 1,
		Basis: 150000, Jenis: "persen", Nilai: 1000, Pola: "sekali", Jumlah: 15000, Status: "pending", CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, c)
	c2, err := q.InsertAffiliateCommission(ctx, queries.InsertAffiliateCommissionParams{
		AffiliateID: a.ID, LeadID: lead.ID, Sumber: "pendaftaran", SumberID: lead.ID, BulanKe: 1, Status: "pending", CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.Zero(t, c2, "unik per sumber")

	_, err = q.CreateAffiliateInvite(ctx, queries.CreateAffiliateInviteParams{AffiliateID: a.ID, TokenHash: "h", ExpiresAt: now.Add(72 * time.Hour), CreatedAt: now})
	require.NoError(t, err)
	_ = db
	_ = sql.ErrNoRows
}
