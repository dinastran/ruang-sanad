package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestApproveExternalCreatesAccountAndInvite(t *testing.T) {
	f := setupAffiliateFixture(t)
	ctx := t.Context()
	id, err := f.aff.Register(ctx, models.AffiliateSignup{Nama: "Dewi Lestari", NoWA: "081200000009", Email: "Dewi@Example.com"})
	require.NoError(t, err)

	inv, err := f.aff.Approve(ctx, id, f.adminID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(inv.Path, "/affiliate/aktivasi/"))
	wa := inv.WaURL("https://app.test")
	require.True(t, strings.HasPrefix(wa, "https://wa.me/6281200000009?text="))
	require.Contains(t, wa, "https%3A%2F%2Fapp.test%2Faffiliate%2Faktivasi%2F")
	a, err := f.querier.GetAffiliate(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "aktif", a.Status)
	require.True(t, a.UserID.Valid)
	var role, email string
	require.NoError(t, f.db.QueryRow(`SELECT role, email FROM users WHERE id = ?`, a.UserID.Int64).Scan(&role, &email))
	require.Equal(t, "affiliate", role)
	require.Equal(t, "dewi@example.com", email)

	token := strings.TrimPrefix(inv.Path, "/affiliate/aktivasi/")
	require.Error(t, f.aff.Activate(ctx, token, "pendek"), "password minimal 8")
	require.NoError(t, f.aff.Activate(ctx, token, "rahasia-123"))
	require.Error(t, f.aff.Activate(ctx, token, "rahasia-456"), "token sekali pakai")
	var hash string
	require.NoError(t, f.db.QueryRow(`SELECT password FROM users WHERE id = ?`, a.UserID.Int64).Scan(&hash))
	require.True(t, CheckPassword("rahasia-123", hash))
}

func TestActivateExpiredToken(t *testing.T) {
	f := setupAffiliateFixture(t)
	id, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Eka Putri", NoWA: "081200000010", Email: "eka@x.com"})
	require.NoError(t, err)
	inv, err := f.aff.Approve(t.Context(), id, f.adminID)
	require.NoError(t, err)
	f.clock = f.clock.Add(73 * time.Hour)
	err = f.aff.Activate(t.Context(), strings.TrimPrefix(inv.Path, "/affiliate/aktivasi/"), "rahasia-123")
	require.True(t, errors.Is(err, ErrUndanganTidakValid))
}

func TestApproveEmailOfStaffRefused(t *testing.T) {
	f := setupAffiliateFixture(t)
	_, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Staf Ganda", NoWA: "081200000011", Email: "cs@example.com"})
	var fe *FieldError
	require.True(t, errors.As(err, &fe), "email akun staf ditolak sejak pendaftaran")
	require.Equal(t, "email", fe.Field)
}

func TestFixApproveRefusesExistingAccount(t *testing.T) {
	f := setupAffiliateFixture(t)
	_, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('korban@x.com', 'Korban', 'user')`)
	require.NoError(t, err)
	_, err = f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Penyerang", NoWA: "081200000012", Email: "korban@x.com"})
	var fe *FieldError
	require.True(t, errors.As(err, &fe), "email akun yang sudah ada ditolak saat daftar")
	require.Equal(t, "email", fe.Field)

	id, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Penyerang", NoWA: "081200000012", Email: "baru@x.com"})
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('baru@x.com', 'Pemilik Asli', 'user')`)
	require.NoError(t, err)
	_, err = f.aff.Approve(t.Context(), id, f.adminID)
	require.Error(t, err, "akun yang muncul setelah daftar juga tidak diambil alih")
	var role string
	require.NoError(t, f.db.QueryRow(`SELECT role FROM users WHERE email = 'baru@x.com'`).Scan(&role))
	require.Equal(t, "user", role)
}

func TestFixInviteOnlyExternal(t *testing.T) {
	f := setupAffiliateFixture(t)
	res, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('g@x.com', 'Guru', 'guru')`)
	require.NoError(t, err)
	uid, _ := res.LastInsertId()
	_, err = f.db.Exec(`INSERT INTO guru (nama, user_id) VALUES ('Hasan', ?)`, uid)
	require.NoError(t, err)
	_, err = f.aff.EnsureGuruAffiliates(t.Context())
	require.NoError(t, err)
	a, err := f.aff.ForUser(t.Context(), uid)
	require.NoError(t, err)
	_, err = f.aff.NewInvite(t.Context(), a.ID)
	require.Error(t, err, "tidak boleh membuat link set-password untuk akun guru")
}

func TestFixNewInviteRevokesOld(t *testing.T) {
	f := setupAffiliateFixture(t)
	id, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Eka Putri", NoWA: "081200000010", Email: "eka@x.com"})
	require.NoError(t, err)
	inv1, err := f.aff.Approve(t.Context(), id, f.adminID)
	require.NoError(t, err)
	_, err = f.aff.NewInvite(t.Context(), id)
	require.NoError(t, err)
	err = f.aff.Activate(t.Context(), strings.TrimPrefix(inv1.Path, "/affiliate/aktivasi/"), "rahasia-123")
	require.True(t, errors.Is(err, ErrUndanganTidakValid), "link lama tidak berlaku setelah kirim ulang")

	inv3, err := f.aff.NewInvite(t.Context(), id)
	require.NoError(t, err)
	require.NoError(t, f.aff.SetStatus(t.Context(), id, "nonaktif", f.adminID))
	err = f.aff.Activate(t.Context(), strings.TrimPrefix(inv3.Path, "/affiliate/aktivasi/"), "rahasia-123")
	require.True(t, errors.Is(err, ErrUndanganTidakValid), "affiliate nonaktif tidak bisa aktivasi")
}

func TestRegisterRequiresEmail(t *testing.T) {
	f := setupAffiliateFixture(t)
	_, err := f.aff.Register(t.Context(), models.AffiliateSignup{Nama: "Tanpa Email", NoWA: "081200000013"})
	var fe *FieldError
	require.True(t, errors.As(err, &fe))
	require.Equal(t, "email", fe.Field)
}

func TestAffiliateForUser(t *testing.T) {
	f := setupAffiliateFixture(t)
	res, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('g@x.com', 'Guru', 'koordinator_guru')`)
	require.NoError(t, err)
	uid, _ := res.LastInsertId()
	_, err = f.db.Exec(`INSERT INTO guru (nama, user_id) VALUES ('Hasan', ?)`, uid)
	require.NoError(t, err)
	_, err = f.aff.EnsureGuruAffiliates(t.Context())
	require.NoError(t, err)
	a, err := f.aff.ForUser(t.Context(), uid)
	require.NoError(t, err)
	require.Equal(t, "guru", a.Tipe)
	_, err = f.aff.ForUser(t.Context(), f.csID)
	require.True(t, errors.Is(err, ErrAffiliateTidakAda))
}
