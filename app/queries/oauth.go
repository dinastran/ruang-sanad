package queries

// Handwritten queries for the OAuth 2.1 authorization server that lets
// Claude.ai / ChatGPT connect to /mcp. Same convention as ai_token.go.
import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

var (
	ErrOAuthClientNotFound = errors.New("oauth client tidak ditemukan")
	ErrOAuthCodeNotFound   = errors.New("kode otorisasi tidak ditemukan")
	ErrOAuthTokenNotFound  = errors.New("token oauth tidak ditemukan")
)

func (q *Queries) CreateOAuthClient(ctx context.Context, c models.OAuthClient) error {
	uris, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx, `INSERT INTO oauth_clients (client_id, client_secret_hash, client_name, redirect_uris, token_endpoint_auth_method)
	 VALUES (?, ?, ?, ?, ?)`, c.ClientID, c.ClientSecretHash, c.ClientName, string(uris), c.TokenEndpointAuthMethod)
	return err
}

func (q *Queries) GetOAuthClient(ctx context.Context, clientID string) (*models.OAuthClient, error) {
	var c models.OAuthClient
	var uris string
	err := q.db.QueryRowContext(ctx, `SELECT client_id, client_secret_hash, client_name, redirect_uris, token_endpoint_auth_method, created_at
	 FROM oauth_clients WHERE client_id = ?`, clientID).
		Scan(&c.ClientID, &c.ClientSecretHash, &c.ClientName, &uris, &c.TokenEndpointAuthMethod, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthClientNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(uris), &c.RedirectURIs); err != nil {
		return nil, err
	}
	return &c, nil
}

func (q *Queries) CreateOAuthAuthorizationCode(ctx context.Context, c models.OAuthAuthorizationCode) error {
	write := 0
	if c.CanWrite {
		write = 1
	}
	_, err := q.db.ExecContext(ctx, `INSERT INTO oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri, code_challenge, can_write, expires_at)
	 VALUES (?, ?, ?, ?, ?, ?, ?)`, c.CodeHash, c.ClientID, c.UserID, c.RedirectURI, c.CodeChallenge, write, c.ExpiresAt.UTC())
	return err
}

// ConsumeOAuthAuthorizationCode marks the code used and returns it. A code can
// be consumed once: a second call (replay) returns ErrOAuthCodeNotFound.
func (q *Queries) ConsumeOAuthAuthorizationCode(ctx context.Context, codeHash string, at time.Time) (*models.OAuthAuthorizationCode, error) {
	var c models.OAuthAuthorizationCode
	err := q.db.QueryRowContext(ctx, `UPDATE oauth_authorization_codes SET used_at = ?
	 WHERE code_hash = ? AND used_at IS NULL
	 RETURNING code_hash, client_id, user_id, redirect_uri, code_challenge, can_write, expires_at`, at.UTC(), codeHash).
		Scan(&c.CodeHash, &c.ClientID, &c.UserID, &c.RedirectURI, &c.CodeChallenge, &c.CanWrite, &c.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteExpiredOAuthAuthorizationCodes removes codes past their expiry.
func (q *Queries) DeleteExpiredOAuthAuthorizationCodes(ctx context.Context, before time.Time) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM oauth_authorization_codes WHERE expires_at < ?`, before.UTC())
	return err
}

// CreateOAuthGrant stores an OAuth connection as an ai_tokens row (jenis
// 'oauth'). tokenHash is a random value never handed out: these rows are not
// usable as personal tokens.
func (q *Queries) CreateOAuthGrant(ctx context.Context, userID int64, nama, tokenHash, clientID string, expiresAt time.Time, canWrite bool) (int64, error) {
	var id int64
	write := 0
	if canWrite {
		write = 1
	}
	err := q.db.QueryRowContext(ctx, `INSERT INTO ai_tokens (user_id, nama, token_hash, prefix, expires_at, can_write, jenis, oauth_client_id)
	 VALUES (?, ?, ?, 'oauth', ?, ?, 'oauth', ?) RETURNING id`, userID, nama, tokenHash, expiresAt.UTC(), write, clientID).Scan(&id)
	return id, err
}

func (q *Queries) ExtendOAuthGrant(ctx context.Context, grantID int64, expiresAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE ai_tokens SET expires_at = ? WHERE id = ? AND jenis = 'oauth'`, expiresAt.UTC(), grantID)
	return err
}

// RevokeOAuthGrant ends a connection and every token issued under it.
func (q *Queries) RevokeOAuthGrant(ctx context.Context, grantID int64, at time.Time) error {
	if _, err := q.db.ExecContext(ctx, `UPDATE ai_tokens SET revoked_at = ? WHERE id = ? AND jenis = 'oauth' AND revoked_at IS NULL`, at.UTC(), grantID); err != nil {
		return err
	}
	_, err := q.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE ai_token_id = ? AND revoked_at IS NULL`, at.UTC(), grantID)
	return err
}

func (q *Queries) CreateOAuthToken(ctx context.Context, grantID int64, jenis, tokenHash string, expiresAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO oauth_tokens (ai_token_id, jenis, token_hash, expires_at) VALUES (?, ?, ?, ?)`,
		grantID, jenis, tokenHash, expiresAt.UTC())
	return err
}

func (q *Queries) GetOAuthTokenAuthByHash(ctx context.Context, tokenHash string) (*models.OAuthTokenAuth, error) {
	var a models.OAuthTokenAuth
	var revoked, grantRevoked, lastUsed sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT o.id, o.jenis, o.expires_at, o.revoked_at,
	 t.id, t.oauth_client_id, t.expires_at, t.revoked_at, t.last_used_at, t.can_write,
	 u.id, u.name, u.role
	 FROM oauth_tokens o
	 JOIN ai_tokens t ON t.id = o.ai_token_id AND t.jenis = 'oauth'
	 JOIN users u ON u.id = t.user_id
	 WHERE o.token_hash = ?`, tokenHash).
		Scan(&a.TokenID, &a.Jenis, &a.ExpiresAt, &revoked,
			&a.GrantID, &a.GrantClientID, &a.GrantExpiresAt, &grantRevoked, &lastUsed, &a.CanWrite,
			&a.UserID, &a.UserName, &a.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	a.RevokedAt, a.GrantRevokedAt, a.LastUsedAt = nullTimePtr(revoked), nullTimePtr(grantRevoked), nullTimePtr(lastUsed)
	return &a, nil
}

// RevokeOAuthToken revokes one token; it reports whether this call revoked it
// (false when it was already revoked), which detects refresh token reuse.
func (q *Queries) RevokeOAuthToken(ctx context.Context, tokenID int64, at time.Time) (bool, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, at.UTC(), tokenID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// DeleteExpiredOAuthTokens prunes tokens that expired before the cutoff.
func (q *Queries) DeleteExpiredOAuthTokens(ctx context.Context, before time.Time) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE expires_at < ?`, before.UTC())
	return err
}
