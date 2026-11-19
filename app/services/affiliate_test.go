package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

type affiliateFixture struct {
	*registrationFixture
	aff *AffiliateService
}

func setupAffiliateFixture(t *testing.T) *affiliateFixture {
	t.Helper()
	rf := setupRegistrationFixture(t)
	aff := NewAffiliateService(rf.querier, rf.service, []byte("aff-secret"))
	aff.now = func() time.Time { return rf.clock }
	rf.reg.WithAffiliates(aff)
	return &affiliateFixture{registrationFixture: rf, aff: aff}
}

func (f *affiliateFixture) external(t *testing.T, nama, wa string) int64 {
	t.Helper()
	id, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: nama, NoWA: wa, Email: nama + "@example.com"})
	require.NoError(t, err)
	require.NoError(t, f.aff.SetStatus(t.Context(), id, "aktif", f.adminID))
	return id
}

func TestEnsureGuruAffiliates(t *testing.T) {
	f := setupAffiliateFixture(t)
	res, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('g1@x.com', 'Guru 1', 'guru'), ('g2@x.com', 'Guru 2', 'koordinator_guru')`)
	require.NoError(t, err)
	last, _ := res.LastInsertId()
	_, err = f.db.Exec(`INSERT INTO guru (nama, user_id) VALUES ('Ahmad Fauzi', ?), ('Ahmad Rifai', ?), ('Tanpa Akun', NULL)`, last-1, last)
	require.NoError(t, err)

	n, err := f.aff.EnsureGuruAffiliates(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, n, "hanya guru dengan akun login")
	n, err = f.aff.EnsureGuruAffiliates(t.Context())
	require.NoError(t, err)
	require.Zero(t, n, "idempoten")
	list, err := f.aff.List(t.Context())
	require.NoError(t, err)
	kode := map[string]bool{}
	for _, a := range list {
		require.Equal(t, "guru", a.Tipe)
		require.Equal(t, "aktif", a.Status)
		kode[a.Kode] = true
	}
	require.Len(t, kode, 2, "kode unik walau nama depan sama")
}

func TestRegisterExternalValidation(t *testing.T) {
	f := setupAffiliateFixture(t)
	_, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "A", NoWA: "0812345678"})
	require.Error(t, err)
	_, err = f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Budi", NoWA: "123"})
	require.Error(t, err)
	id, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Budi Santoso", NoWA: "0812-3456-7000", Email: "budi@x.com"})
	require.NoError(t, err)
	a, err := f.querier.GetAffiliate(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, "menunggu", a.Status)
	require.Equal(t, "6281234567000", a.NoWa)
	_, err = f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Budi Lagi", NoWA: "081234567000"})
	require.Error(t, err, "nomor sudah terdaftar")
}

func TestResolveRulePrecedence(t *testing.T) {
	f := setupAffiliateFixture(t)
	ctx := t.Context()
	a := f.external(t, "Citra", "081200000001")
	r, err := f.aff.ResolveRule(ctx, a, 1)
	require.NoError(t, err)
	require.Nil(t, r, "tanpa aturan = tanpa komisi")

	require.NoError(t, f.aff.SaveRule(ctx, models.AffiliateRuleRequest{Scope: "global", Jenis: "persen", Nilai: 1000, Pola: "sekali"}, f.adminID))
	require.NoError(t, f.aff.SaveRule(ctx, models.AffiliateRuleRequest{Scope: "program", TargetID: 1, Jenis: "flat", Nilai: 75000, Pola: "bulanan"}, f.adminID))
	r, err = f.aff.ResolveRule(ctx, a, 1)
	require.NoError(t, err)
	require.Equal(t, "program", r.Scope)
	r, err = f.aff.ResolveRule(ctx, a, 99)
	require.NoError(t, err)
	require.Equal(t, "global", r.Scope)
	require.NoError(t, f.aff.SaveRule(ctx, models.AffiliateRuleRequest{Scope: "affiliate", TargetID: a, Jenis: "persen", Nilai: 2000, Pola: "sekali"}, f.adminID))
	r, err = f.aff.ResolveRule(ctx, a, 1)
	require.NoError(t, err)
	require.Equal(t, "affiliate", r.Scope)

	rule, err := f.querier.GetAffiliateRule(ctx, queries.GetAffiliateRuleParams{Scope: "program", TargetID: 1})
	require.NoError(t, err)
	require.EqualValues(t, 6, rule.MaksBulan.Int64, "bulanan default 6 bulan")
}

func TestSaveRuleValidation(t *testing.T) {
	f := setupAffiliateFixture(t)
	ctx := t.Context()
	bad := []models.AffiliateRuleRequest{
		{Scope: "global", Jenis: "persen", Nilai: 10001, Pola: "sekali"},
		{Scope: "global", Jenis: "x", Nilai: 1, Pola: "sekali"},
		{Scope: "global", Jenis: "flat", Nilai: -1, Pola: "sekali"},
		{Scope: "program", TargetID: 999, Jenis: "flat", Nilai: 1, Pola: "sekali"},
		{Scope: "affiliate", TargetID: 999, Jenis: "flat", Nilai: 1, Pola: "sekali"},
		{Scope: "global", Jenis: "flat", Nilai: 1, Pola: "bulanan", MaksBulan: ptrInt64(0)},
	}
	for i, r := range bad {
		require.Error(t, f.aff.SaveRule(ctx, r, f.adminID), i)
	}
}

func TestAffiliateSettingsValidation(t *testing.T) {
	f := setupAffiliateFixture(t)
	ctx := t.Context()
	require.Error(t, f.aff.SaveSettings(ctx, models.AffiliateSettings{MasaBerlakuHari: 45, MasaTahanHari: 14}, f.adminID))
	require.NoError(t, f.aff.SaveSettings(ctx, models.AffiliateSettings{MasaBerlakuHari: 90, MasaTahanHari: 7}, f.adminID))
	st, err := f.aff.Settings(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 90, st.MasaBerlakuHari)
}

func ptrInt64(v int64) *int64 { return &v }

func TestSaveRuleSelamaAktif(t *testing.T) {
	f := setupAffiliateFixture(t)
	require.NoError(t, f.aff.SaveRule(t.Context(), models.AffiliateRuleRequest{Scope: "global", Jenis: "flat", Nilai: 1, Pola: "bulanan", MaksBulan: ptrInt64(-1)}, f.adminID))
	r, err := f.querier.GetAffiliateRule(t.Context(), queries.GetAffiliateRuleParams{Scope: "global"})
	require.NoError(t, err)
	require.False(t, r.MaksBulan.Valid, "-1 = selama aktif")
}
