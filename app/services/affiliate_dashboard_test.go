package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestMyAffiliateDashboard(t *testing.T) {
	f, a, _ := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "persen", Nilai: 1000, Pola: "sekali"})
	_, err := f.aff.TrackClick(t.Context(), "AHMAD", "9.9.9.9", "ua", "")
	require.NoError(t, err)
	f.reconcile(t)
	d, err := f.aff.Dashboard(t.Context(), a)
	require.NoError(t, err)
	require.Equal(t, "AHMAD", d.Kode)
	require.EqualValues(t, 1, d.Klik30)
	require.EqualValues(t, 1, d.Daftar30)
	require.EqualValues(t, 1, d.Santri)
	require.EqualValues(t, 15000, d.KomisiPending)
	require.Len(t, d.Rujukan, 1)
	require.Equal(t, "Fulanah", d.Rujukan[0].NamaDepan, "hanya nama depan")
	require.Equal(t, "terverifikasi", d.Rujukan[0].Status)
	require.NotNil(t, d.Aturan)
	require.Equal(t, "10%", d.Aturan.Label)
	require.NotEmpty(t, d.Tujuan, "daftar link pendaftaran aktif untuk dibagikan")
}

func TestAdminAffiliateRowsAndLedger(t *testing.T) {
	f, a, _ := paidAffiliateLead(t, models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 50000, Pola: "sekali"})
	f.reconcile(t)
	f.clock = f.clock.Add(15 * 24 * time.Hour)
	f.reconcile(t)
	rows, err := f.aff.AdminRows(t.Context())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, a, rows[0].ID)
	require.EqualValues(t, 1, rows[0].Daftar)
	require.EqualValues(t, 1, rows[0].Santri)
	require.EqualValues(t, 50000, rows[0].KomisiSiap)
	ledger, err := f.aff.Ledger(t.Context(), "siap")
	require.NoError(t, err)
	require.Len(t, ledger, 1)
	require.EqualValues(t, 50000, ledger[0].Jumlah)
	require.Equal(t, "Fulanah", ledger[0].LeadNama)
}
