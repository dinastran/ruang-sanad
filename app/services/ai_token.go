package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

const (
	AITokenPrefix         = "rsai_"
	aiTokenDefaultHari    = 90
	aiTokenMaksHari       = 365
	aiTokenMaksAktif      = 5
	aiTokenNamaMaks       = 60
	aiTokenTouchInterval  = 5 * time.Minute
	aiAuditListDefaultMax = 50
)

var (
	ErrAITokenInvalid  = errors.New("token AI tidak valid, sudah dicabut, atau kedaluwarsa")
	ErrAIRoleDitolak   = errors.New("role Anda belum mendapat akses AI agent")
	ErrAITokenNotFound = errors.New("token tidak ditemukan")
)

// aiAllowedRoles are the staff roles that may connect an AI agent in phase 1.
// guru is excluded: its data is scoped to its own classes, which the tools do
// not model yet. "user" (no role assigned) is excluded.
var aiAllowedRoles = map[string]bool{
	"super_admin":      true,
	"admin":            true,
	"keuangan":         true,
	"cs":               true,
	"admin_kelas":      true,
	"koordinator_guru": true,
}

func AIRoleAllowed(role string) bool { return aiAllowedRoles[role] }

type AITokenService struct {
	querier *queries.Querier
	now     func() time.Time
}

func NewAITokenService(q *queries.Querier) *AITokenService {
	return &AITokenService{querier: q, now: time.Now}
}

func hashAIToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

type AITokenCreated struct {
	Token     models.AIToken `json:"token"`
	Plaintext string         `json:"plaintext"`
}

// Create issues a new token. The plaintext is returned once and never stored.
func (s *AITokenService) Create(userID int64, role, nama string, masaBerlakuHari int) (*AITokenCreated, error) {
	if !AIRoleAllowed(role) {
		return nil, ErrAIRoleDitolak
	}
	nama = strings.TrimSpace(nama)
	if nama == "" {
		return nil, errors.New("beri nama token, misalnya \"Claude Desktop\" atau \"Hermes\"")
	}
	if utf8.RuneCountInString(nama) > aiTokenNamaMaks {
		return nil, errors.New("nama token maksimal 60 karakter")
	}
	if masaBerlakuHari == 0 {
		masaBerlakuHari = aiTokenDefaultHari
	}
	if masaBerlakuHari < 1 || masaBerlakuHari > aiTokenMaksHari {
		return nil, errors.New("masa berlaku token 1–365 hari")
	}
	ctx := context.Background()
	now := s.now()
	aktif, err := s.querier.CountActiveAITokensByUser(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	if aktif >= aiTokenMaksAktif {
		return nil, errors.New("maksimal 5 token aktif. Cabut token lama yang tidak dipakai terlebih dahulu")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	plain := AITokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	expires := now.Add(time.Duration(masaBerlakuHari) * 24 * time.Hour)
	prefix := plain[:len(AITokenPrefix)+6]
	id, err := s.querier.CreateAIToken(ctx, userID, nama, hashAIToken(plain), prefix, expires)
	if err != nil {
		return nil, err
	}
	return &AITokenCreated{
		Token:     models.AIToken{ID: id, UserID: userID, Nama: nama, Prefix: prefix, ExpiresAt: expires.UTC(), CreatedAt: now.UTC()},
		Plaintext: plain,
	}, nil
}

func (s *AITokenService) List(userID int64) ([]models.AIToken, error) {
	return s.querier.ListAITokensByUser(context.Background(), userID)
}

func (s *AITokenService) Revoke(userID, tokenID int64) error {
	ok, err := s.querier.RevokeAIToken(context.Background(), tokenID, userID, s.now())
	if err != nil {
		return err
	}
	if !ok {
		return ErrAITokenNotFound
	}
	return nil
}

// Authenticate resolves a bearer token to its owner. The owner's current role
// is re-checked on every call, so demoting a user cuts AI access immediately.
func (s *AITokenService) Authenticate(ctx context.Context, plain string) (*models.AIPrincipal, error) {
	plain = strings.TrimSpace(plain)
	if !strings.HasPrefix(plain, AITokenPrefix) || len(plain) < len(AITokenPrefix)+20 {
		return nil, ErrAITokenInvalid
	}
	auth, err := s.querier.GetAITokenAuthByHash(ctx, hashAIToken(plain))
	if errors.Is(err, queries.ErrAITokenNotFound) {
		return nil, ErrAITokenInvalid
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	if auth.RevokedAt != nil || !auth.ExpiresAt.After(now) {
		return nil, ErrAITokenInvalid
	}
	if !AIRoleAllowed(auth.Role) {
		return nil, ErrAIRoleDitolak
	}
	// Record usage at most once per interval to avoid a write per request.
	if auth.LastUsedAt == nil || now.Sub(*auth.LastUsedAt) >= aiTokenTouchInterval {
		_ = s.querier.TouchAIToken(ctx, auth.TokenID, now)
	}
	return &models.AIPrincipal{UserID: auth.UserID, TokenID: auth.TokenID, UserName: auth.UserName, Role: auth.Role}, nil
}

// ListAudit returns the caller's own recent AI tool calls; super_admin sees
// every user's calls.
func (s *AITokenService) ListAudit(userID int64, role string) ([]models.AIAuditEntry, error) {
	scope := userID
	if role == "super_admin" {
		scope = 0
	}
	return s.querier.ListAIAudit(context.Background(), scope, aiAuditListDefaultMax)
}
