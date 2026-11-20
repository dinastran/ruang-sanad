package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrUndanganTidakValid = errors.New("link aktivasi tidak valid atau sudah kedaluwarsa; minta link baru ke admin")

const inviteTTL = 72 * time.Hour

// Invite is the account activation link for an approved external affiliate.
type Invite struct {
	Path     string // /affiliate/aktivasi/<token>
	WaNumber string // affiliate's WhatsApp (62…)
	Email    string
}

// WaURL is the wa.me link the admin taps to send the activation link from
// their own phone; baseURL is the public app URL.
func (i Invite) WaURL(baseURL string) string {
	if i.WaNumber == "" {
		return ""
	}
	return waURL(i.WaNumber, "Assalamu'alaikum, pendaftaran affiliate Ruang Sanad Anda sudah disetujui. "+
		"Buat password untuk login lewat link ini (berlaku 3 hari): "+baseURL+i.Path)
}

func hashInviteToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// ForUser returns the active affiliate linked to a login (guru or external).
func (s *AffiliateService) ForUser(ctx context.Context, userID int64) (queries.Affiliate, error) {
	a, err := s.querier.GetAffiliateByUser(ctx, sql.NullInt64{Int64: userID, Valid: true})
	if errors.Is(err, sql.ErrNoRows) || (err == nil && a.Status != "aktif") {
		return a, ErrAffiliateTidakAda
	}
	return a, err
}

// Approve activates an external affiliate: creates (or reuses a plain
// "user") login with role affiliate and returns a 72-hour activation link.
func (s *AffiliateService) Approve(ctx context.Context, id, actorID int64) (Invite, error) {
	a, err := s.get(ctx, id)
	if err != nil {
		return Invite{}, err
	}
	if a.Tipe != "eksternal" {
		return Invite{}, errors.New("hanya affiliate eksternal yang perlu disetujui")
	}
	if a.Email == "" {
		return Invite{}, errors.New("affiliate belum punya email untuk login")
	}
	err = s.leads.withTx(ctx, func(q *queries.Querier) error {
		now := s.nowUTC()
		var userID int64
		u, err := q.GetUserByEmail(ctx, a.Email)
		switch {
		case errors.Is(err, queries.ErrUserNotFound):
			// Unusable random password until the affiliate sets one.
			pw, err := HashPassword(newRegistrationRef())
			if err != nil {
				return err
			}
			nu := &models.User{Email: a.Email, Name: a.Nama, Password: sql.NullString{String: pw, Valid: true}, Role: models.RoleAffiliate}
			if err := q.CreateUser(ctx, nu); err != nil {
				return err
			}
			userID = nu.ID
		case err != nil:
			return err
		default:
			// An account with this email appeared after the signup. Taking
			// it over would hand its password link to the registrant.
			return errors.New("email ini sudah dipakai akun lain (" + string(u.Role) + "); minta affiliate mendaftar ulang dengan email lain")
		}
		a.UserID = sql.NullInt64{Int64: userID, Valid: true}
		a.Status = "aktif"
		a.DisetujuiAt = sql.NullTime{Time: now, Valid: true}
		a.DisetujuiOleh = sql.NullInt64{Int64: actorID, Valid: actorID > 0}
		return q.UpdateAffiliate(ctx, queries.UpdateAffiliateParams{
			Kode: a.Kode, Nama: a.Nama, NoWa: a.NoWa, Email: a.Email, Status: a.Status, BankNama: a.BankNama,
			BankRekening: a.BankRekening, BankAtasNama: a.BankAtasNama, CatatanAdmin: a.CatatanAdmin, UserID: a.UserID,
			DisetujuiOleh: a.DisetujuiOleh, DisetujuiAt: a.DisetujuiAt, UpdatedAt: now, ID: a.ID,
		})
	})
	if err != nil {
		return Invite{}, err
	}
	return s.NewInvite(ctx, id)
}

// NewInvite issues a fresh activation link (also "kirim ulang").
func (s *AffiliateService) NewInvite(ctx context.Context, id int64) (Invite, error) {
	a, err := s.get(ctx, id)
	if err != nil {
		return Invite{}, err
	}
	if a.Tipe != "eksternal" {
		return Invite{}, errors.New("link aktivasi hanya untuk affiliate eksternal")
	}
	if !a.UserID.Valid || a.Status != "aktif" {
		return Invite{}, errors.New("affiliate belum disetujui atau tidak aktif")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return Invite{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	now := s.nowUTC()
	// Only the newest link works ("kirim ulang" revokes older ones).
	if err := s.querier.RevokeAffiliateInvites(ctx, queries.RevokeAffiliateInvitesParams{
		UsedAt: sql.NullTime{Time: now, Valid: true}, AffiliateID: a.ID,
	}); err != nil {
		return Invite{}, err
	}
	if _, err := s.querier.CreateAffiliateInvite(ctx, queries.CreateAffiliateInviteParams{
		AffiliateID: a.ID, TokenHash: hashInviteToken(token), ExpiresAt: now.Add(inviteTTL), CreatedAt: now,
	}); err != nil {
		return Invite{}, err
	}
	return Invite{Path: "/affiliate/aktivasi/" + token, WaNumber: a.NoWa, Email: a.Email}, nil
}

// Activate sets the password of the affiliate's login with a one-time token.
func (s *AffiliateService) Activate(ctx context.Context, token, password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return &FieldError{Field: "password", Pesan: "Password minimal 8 karakter"}
	}
	inv, err := s.querier.GetAffiliateInviteByHash(ctx, hashInviteToken(strings.TrimSpace(token)))
	if err != nil {
		return ErrUndanganTidakValid
	}
	a, err := s.get(ctx, inv.AffiliateID)
	if err != nil || !a.UserID.Valid || a.Status != "aktif" || a.Tipe != "eksternal" {
		return ErrUndanganTidakValid
	}
	if role, err := s.querier.GetUserRole(ctx, a.UserID.Int64); err != nil || role != string(models.RoleAffiliate) {
		return ErrUndanganTidakValid
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.leads.withTx(ctx, func(q *queries.Querier) error {
		now := s.nowUTC()
		n, err := q.UseAffiliateInvite(ctx, queries.UseAffiliateInviteParams{UsedAt: sql.NullTime{Time: now, Valid: true}, ID: inv.ID, ExpiresAt: now})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrUndanganTidakValid
		}
		return q.UpdateUserPassword(ctx, a.UserID.Int64, hash)
	})
}
