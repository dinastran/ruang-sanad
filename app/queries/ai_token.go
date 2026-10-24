package queries

// Handwritten queries for AI agent access tokens and the AI audit log.
// Following overview.go / santri_analytics.go: plain SQL owned by this package.
import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

var ErrAITokenNotFound = errors.New("token tidak ditemukan")

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func (q *Queries) CreateAIToken(ctx context.Context, userID int64, nama, tokenHash, prefix string, expiresAt time.Time, canWrite bool) (int64, error) {
	var id int64
	write := 0
	if canWrite {
		write = 1
	}
	err := q.db.QueryRowContext(ctx, `INSERT INTO ai_tokens (user_id, nama, token_hash, prefix, expires_at, can_write)
	 VALUES (?, ?, ?, ?, ?, ?) RETURNING id`, userID, nama, tokenHash, prefix, expiresAt.UTC(), write).Scan(&id)
	return id, err
}

func (q *Queries) ListAITokensByUser(ctx context.Context, userID int64) ([]models.AIToken, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, user_id, nama, prefix, expires_at, last_used_at, revoked_at, created_at, can_write
	 FROM ai_tokens WHERE user_id = ? ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIToken, 0)
	for rows.Next() {
		var t models.AIToken
		var lastUsed, revoked sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Nama, &t.Prefix, &t.ExpiresAt, &lastUsed, &revoked, &t.CreatedAt, &t.CanWrite); err != nil {
			return nil, err
		}
		t.LastUsedAt, t.RevokedAt = nullTimePtr(lastUsed), nullTimePtr(revoked)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (q *Queries) CountActiveAITokensByUser(ctx context.Context, userID int64, now time.Time) (int64, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT expires_at FROM ai_tokens WHERE user_id = ? AND revoked_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int64
	for rows.Next() {
		var exp time.Time
		if err := rows.Scan(&exp); err != nil {
			return 0, err
		}
		if exp.After(now) {
			n++
		}
	}
	return n, rows.Err()
}

func (q *Queries) GetAITokenAuthByHash(ctx context.Context, tokenHash string) (*models.AITokenAuth, error) {
	var a models.AITokenAuth
	var revoked, lastUsed sql.NullTime
	err := q.db.QueryRowContext(ctx, `SELECT t.id, t.user_id, u.name, u.role, t.expires_at, t.revoked_at, t.last_used_at, t.can_write
	 FROM ai_tokens t JOIN users u ON u.id = t.user_id WHERE t.token_hash = ?`, tokenHash).
		Scan(&a.TokenID, &a.UserID, &a.UserName, &a.Role, &a.ExpiresAt, &revoked, &lastUsed, &a.CanWrite)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAITokenNotFound
	}
	if err != nil {
		return nil, err
	}
	a.RevokedAt, a.LastUsedAt = nullTimePtr(revoked), nullTimePtr(lastUsed)
	return &a, nil
}

func (q *Queries) TouchAIToken(ctx context.Context, tokenID int64, at time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE ai_tokens SET last_used_at = ? WHERE id = ?`, at.UTC(), tokenID)
	return err
}

// RevokeAIToken only revokes a token owned by userID; it reports whether a
// row changed so callers can answer "not found" for someone else's token.
func (q *Queries) RevokeAIToken(ctx context.Context, tokenID, userID int64, at time.Time) (bool, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE ai_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`, at.UTC(), tokenID, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (q *Queries) InsertAIAudit(ctx context.Context, e models.AIAuditEntry) error {
	var tokenID interface{}
	if e.TokenID > 0 {
		tokenID = e.TokenID
	}
	_, err := q.db.ExecContext(ctx, `INSERT INTO ai_audit_log (user_id, token_id, tool, arguments, status, pesan, durasi_ms)
	 VALUES (?, ?, ?, ?, ?, ?, ?)`, e.UserID, tokenID, e.Tool, e.Arguments, e.Status, e.Pesan, e.DurasiMs)
	return err
}

// ListAIAudit returns the latest audit rows; userID 0 means all users.
func (q *Queries) ListAIAudit(ctx context.Context, userID int64, limit int64) ([]models.AIAuditEntry, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT a.id, COALESCE(a.user_id, 0), COALESCE(u.name, ''), COALESCE(a.token_id, 0),
	 a.tool, a.arguments, a.status, a.pesan, a.durasi_ms, a.created_at
	 FROM ai_audit_log a LEFT JOIN users u ON u.id = a.user_id
	 WHERE (? = 0 OR a.user_id = ?) ORDER BY a.created_at DESC, a.id DESC LIMIT ?`, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AIAuditEntry, 0)
	for rows.Next() {
		var e models.AIAuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.TokenID, &e.Tool, &e.Arguments, &e.Status, &e.Pesan, &e.DurasiMs, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
