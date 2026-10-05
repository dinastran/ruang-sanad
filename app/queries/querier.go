package queries

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrSessionNotFound       = errors.New("session not found")
	ErrPasswordResetNotFound = errors.New("password reset not found")
)

// Querier wraps the generated Queries with domain-level error handling
// and conversion to models.User where needed.
type Querier struct {
	*Queries
}

func NewQuerier(db DBTX) *Querier {
	return &Querier{
		Queries: New(db),
	}
}

// BeginTx exposes a transaction while keeping all SQL execution in generated
// queries. Services use the returned Querier via WithTx.
func (q *Querier) BeginTx(ctx context.Context) (*sql.Tx, error) {
	db, ok := q.Queries.db.(*sql.DB)
	if !ok {
		return nil, errors.New("database transactions are unavailable")
	}
	return db.BeginTx(ctx, nil)
}

func (q *Querier) WithTx(tx *sql.Tx) *Querier {
	return &Querier{Queries: q.Queries.WithTx(tx)}
}

// --- User helpers that convert queries.User -> models.User ---

func toModelUser(qUser User) *models.User {
	return &models.User{
		ID:            qUser.ID,
		Email:         qUser.Email,
		Name:          qUser.Name,
		Password:      qUser.Password,
		Avatar:        nullStringToString(qUser.Avatar),
		Role:          models.UserRole(qUser.Role),
		GoogleID:      qUser.GoogleID,
		EmailVerified: qUser.EmailVerified,
		CreatedAt:     qUser.CreatedAt,
		UpdatedAt:     qUser.UpdatedAt,
	}
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// ClearUserReferences removes legacy author references that predate ON DELETE SET NULL.
// Newer user foreign keys already cascade or null themselves at the schema level.
func (q *Querier) ClearUserReferences(ctx context.Context, userID int64) error {
	statements := []string{
		`UPDATE pertemuan SET dibuat_oleh = NULL WHERE dibuat_oleh = ?`,
		`UPDATE absensi SET dibuat_oleh = NULL WHERE dibuat_oleh = ?`,
		`UPDATE tsi_audit SET oleh = NULL WHERE oleh = ?`,
		`UPDATE jadwal_pertemuan SET dibuat_oleh = NULL WHERE dibuat_oleh = ?`,
	}
	for _, statement := range statements {
		if _, err := q.Queries.db.ExecContext(ctx, statement, userID); err != nil {
			return err
		}
	}
	return nil
}

// --- User operations ---

func (q *Querier) CreateUser(ctx context.Context, user *models.User) error {
	id, err := q.Queries.CreateUser(ctx, CreateUserParams{
		Email:     user.Email,
		Name:      user.Name,
		Password:  user.Password,
		Role:      string(user.Role),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		if isDuplicateEmail(err) {
			return ErrUserAlreadyExists
		}
		return err
	}
	user.ID = id
	return nil
}

func (q *Querier) CreateUserWithGoogleID(ctx context.Context, user *models.User) error {
	id, err := q.Queries.CreateUserWithGoogleID(ctx, CreateUserWithGoogleIDParams{
		Email:         user.Email,
		Name:          user.Name,
		GoogleID:      user.GoogleID,
		Avatar:        sql.NullString{String: user.Avatar, Valid: user.Avatar != ""},
		EmailVerified: user.EmailVerified,
		Role:          string(user.Role),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	if err != nil {
		if isDuplicateEmail(err) {
			return ErrUserAlreadyExists
		}
		return err
	}
	user.ID = id
	return nil
}

func (q *Querier) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	qUser, err := q.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toModelUser(qUser), nil
}

func (q *Querier) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	qUser, err := q.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toModelUser(qUser), nil
}

func (q *Querier) GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	qUser, err := q.Queries.GetUserByGoogleID(ctx, sql.NullString{String: googleID, Valid: googleID != ""})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toModelUser(qUser), nil
}

func (q *Querier) UpdateUser(ctx context.Context, user *models.User) error {
	rows, err := q.Queries.UpdateUser(ctx, UpdateUserParams{
		Name:          user.Name,
		Avatar:        sql.NullString{String: user.Avatar, Valid: user.Avatar != ""},
		EmailVerified: user.EmailVerified,
		UpdatedAt:     time.Now(),
		ID:            user.ID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (q *Querier) UpdateUserPassword(ctx context.Context, id int64, hashedPassword string) error {
	rows, err := q.Queries.UpdateUserPassword(ctx, UpdateUserPasswordParams{
		Password:  sql.NullString{String: hashedPassword, Valid: true},
		UpdatedAt: time.Now(),
		ID:        id,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (q *Querier) UpdateUserAvatar(ctx context.Context, id int64, avatarURL string) error {
	rows, err := q.Queries.UpdateUserAvatar(ctx, UpdateUserAvatarParams{
		Avatar:    sql.NullString{String: avatarURL, Valid: avatarURL != ""},
		UpdatedAt: time.Now(),
		ID:        id,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (q *Querier) DeleteUser(ctx context.Context, id int64) error {
	rows, err := q.Queries.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (q *Querier) SetUserRoleAdmin(ctx context.Context, id int64) error {
	rows, err := q.Queries.SetUserRoleAdmin(ctx, SetUserRoleAdminParams{
		Role:      string(models.RoleAdmin),
		UpdatedAt: time.Now(),
		ID:        id,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// --- Session operations ---

func (q *Querier) CreateSession(ctx context.Context, session *Session) error {
	return q.Queries.CreateSession(ctx, CreateSessionParams{
		ID:        session.ID,
		UserID:    session.UserID,
		Data:      session.Data,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

func (q *Querier) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	qSession, err := q.Queries.GetSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	return &qSession, nil
}

func (q *Querier) GetSessionsByUserID(ctx context.Context, userID int64) ([]*Session, error) {
	qSessions, err := q.Queries.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, s := range qSessions {
		if s.ExpiresAt.After(time.Now()) {
			sessions = append(sessions, &s)
		}
	}
	return sessions, nil
}

func (q *Querier) UpdateSession(ctx context.Context, session *Session) error {
	rows, err := q.Queries.UpdateSession(ctx, UpdateSessionParams{
		Data:      session.Data,
		ExpiresAt: session.ExpiresAt,
		UpdatedAt: time.Now(),
		ID:        session.ID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (q *Querier) DeleteSession(ctx context.Context, id string) error {
	rows, err := q.Queries.DeleteSession(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (q *Querier) DeleteSessionsByUserID(ctx context.Context, userID int64) error {
	return q.Queries.DeleteSessionsByUserID(ctx, userID)
}

func (q *Querier) DeleteExpiredSessions(ctx context.Context) error {
	return q.Queries.DeleteExpiredSessions(ctx, time.Now())
}

// --- Password Reset operations ---

func (q *Querier) CreatePasswordReset(ctx context.Context, token string, userID int64, email string, expiresAt time.Time) error {
	return q.Queries.CreatePasswordReset(ctx, CreatePasswordResetParams{
		Token:     token,
		UserID:    userID,
		Email:     email,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	})
}

func (q *Querier) GetPasswordReset(ctx context.Context, token string) (*PasswordReset, error) {
	pr, err := q.Queries.GetPasswordReset(ctx, GetPasswordResetParams{
		Token:     token,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPasswordResetNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (q *Querier) MarkPasswordResetUsed(ctx context.Context, token string) error {
	return q.Queries.MarkPasswordResetUsed(ctx, token)
}

func (q *Querier) DecodeSessionData(data string) (*models.SessionData, error) {
	var sessionData models.SessionData
	if err := sessionDataFromJSON(data, &sessionData); err != nil {
		return nil, err
	}
	return &sessionData, nil
}

func (q *Querier) EncodeSessionData(data *models.SessionData) (string, error) {
	return sessionDataToJSON(data)
}

// --- Import Log operations (hand-written, no sqlc queries yet) ---

const createImportLog = `INSERT INTO import_log (user_id, nama_file, total_baris, berhasil, gagal, catatan, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

func (q *Querier) CreateImportLog(ctx context.Context, userID int64, namaFile string, totalBaris, berhasil, gagal int64, catatan string) error {
	_, err := q.Queries.db.ExecContext(ctx, createImportLog, userID, namaFile, totalBaris, berhasil, gagal, catatan, time.Now())
	return err
}

const listImportLogs = `SELECT id, user_id, nama_file, total_baris, berhasil, gagal, catatan, created_at FROM import_log ORDER BY created_at DESC`

func (q *Querier) ListImportLogs(ctx context.Context) ([]ImportLog, error) {
	rows, err := q.Queries.db.QueryContext(ctx, listImportLogs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ImportLog
	for rows.Next() {
		var i ImportLog
		if err := rows.Scan(&i.ID, &i.UserID, &i.NamaFile, &i.TotalBaris, &i.Berhasil, &i.Gagal, &i.Catatan, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return items, rows.Err()
}

// --- Tagihan finance helpers (hand-written to keep the change scoped) ---

const syncOpenTagihanNominalForSantri = `
UPDATE tagihan
SET nominal = ?, updated_at = CURRENT_TIMESTAMP
WHERE santri_id = ?
  AND status = 'belum_bayar'
  AND NOT EXISTS (
      SELECT 1 FROM tagihan_nominal_override o WHERE o.tagihan_id = tagihan.id
  )`

func (q *Querier) SyncOpenTagihanNominalForSantri(ctx context.Context, santriID, nominal int64) error {
	_, err := q.Queries.db.ExecContext(ctx, syncOpenTagihanNominalForSantri, nominal, santriID)
	return err
}

const updateTagihanNominal = `
UPDATE tagihan
SET nominal = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'belum_bayar'`

func (q *Querier) UpdateTagihanNominal(ctx context.Context, tagihanID, nominal int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, updateTagihanNominal, nominal, tagihanID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const upsertTagihanNominalOverride = `
INSERT INTO tagihan_nominal_override (tagihan_id, nominal, updated_by)
VALUES (?, ?, ?)
ON CONFLICT(tagihan_id) DO UPDATE SET
    nominal = excluded.nominal,
    updated_by = excluded.updated_by,
    updated_at = CURRENT_TIMESTAMP`

func (q *Querier) UpsertTagihanNominalOverride(ctx context.Context, tagihanID, nominal, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, upsertTagihanNominalOverride, tagihanID, nominal, financeNullablePositiveInt64(userID))
	return err
}

const deleteTagihanNominalOverride = `
DELETE FROM tagihan_nominal_override WHERE tagihan_id = ?`

func (q *Querier) DeleteTagihanNominalOverride(ctx context.Context, tagihanID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, deleteTagihanNominalOverride, tagihanID)
	return err
}

const resetTagihanNominalToSantri = `
UPDATE tagihan
SET nominal = (SELECT nominal FROM santri WHERE santri.id = tagihan.santri_id),
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'belum_bayar'`

func (q *Querier) ResetTagihanNominalToSantri(ctx context.Context, tagihanID int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, resetTagihanNominalToSantri, tagihanID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Querier) ListTagihanNominalOverrideIDs(ctx context.Context, tagihanIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if len(tagihanIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(tagihanIDs))
	args := make([]interface{}, len(tagihanIDs))
	for i, id := range tagihanIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT tagihan_id FROM tagihan_nominal_override WHERE tagihan_id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := q.Queries.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

const hasTagihanNominalOverride = `
SELECT COUNT(*) FROM tagihan_nominal_override WHERE tagihan_id = ?`

func (q *Querier) HasTagihanNominalOverride(ctx context.Context, tagihanID int64) (bool, error) {
	var count int64
	if err := q.Queries.db.QueryRowContext(ctx, hasTagihanNominalOverride, tagihanID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

type TagihanFollowUpLogRow struct {
	ID           int64
	TagihanID    int64
	TemplateID   sql.NullInt64
	TemplateNama string
	MessageBody  string
	DikirimOleh  sql.NullInt64
	PetugasNama  string
	CreatedAt    time.Time
}

const createTagihanFollowUpLog = `
INSERT INTO tagihan_follow_up_log
(tagihan_id, template_id, template_nama, message_body, dikirim_oleh)
VALUES (?, ?, ?, ?, ?)`

func (q *Querier) CreateTagihanFollowUpLog(ctx context.Context, tagihanID int64, templateID sql.NullInt64, templateNama, messageBody string, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, createTagihanFollowUpLog,
		tagihanID, templateID, templateNama, messageBody, financeNullablePositiveInt64(userID),
	)
	return err
}

const listTagihanFollowUpLogs = `
SELECT l.id, l.tagihan_id, l.template_id, l.template_nama, l.message_body,
       l.dikirim_oleh, COALESCE(u.name, '') AS petugas_nama, l.created_at
FROM tagihan_follow_up_log l
LEFT JOIN users u ON u.id = l.dikirim_oleh
WHERE l.tagihan_id = ?
ORDER BY l.created_at DESC, l.id DESC`

func (q *Querier) ListTagihanFollowUpLogs(ctx context.Context, tagihanID int64) ([]TagihanFollowUpLogRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, listTagihanFollowUpLogs, tagihanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagihanFollowUpLogRow
	for rows.Next() {
		var row TagihanFollowUpLogRow
		if err := rows.Scan(
			&row.ID, &row.TagihanID, &row.TemplateID, &row.TemplateNama,
			&row.MessageBody, &row.DikirimOleh, &row.PetugasNama, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func financeNullablePositiveInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v > 0}
}

// --- Per-santri billing progress helpers ---

type SantriBillingProgressRow struct {
	SantriID        int64
	MeetingCount    int64
	LastBilledMonth int64
}

type SantriBillingMeetingRow struct {
	PertemuanID    int64
	KelasID        int64
	PertemuanKe    int64
	Tanggal        time.Time
	Frekuensi      string
	AngkatanKelas  string
}

func (q *Querier) GetOrCreateSantriBillingProgress(ctx context.Context, santriID int64) (SantriBillingProgressRow, error) {
	if _, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO santri_billing_progress (santri_id, meeting_count, last_billed_month)
VALUES (?, 0, 1)
ON CONFLICT(santri_id) DO NOTHING
`, santriID); err != nil {
		return SantriBillingProgressRow{}, err
	}
	var row SantriBillingProgressRow
	err := q.Queries.db.QueryRowContext(ctx, `
SELECT santri_id, meeting_count, last_billed_month
FROM santri_billing_progress
WHERE santri_id = ?
`, santriID).Scan(&row.SantriID, &row.MeetingCount, &row.LastBilledMonth)
	return row, err
}

func (q *Querier) UpdateSantriBillingProgress(ctx context.Context, santriID, meetingCount, lastBilledMonth int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `
UPDATE santri_billing_progress
SET meeting_count = ?, last_billed_month = ?, updated_at = CURRENT_TIMESTAMP
WHERE santri_id = ?
`, meetingCount, lastBilledMonth, santriID)
	return err
}

func (q *Querier) MarkSantriBillingMeetingProcessed(ctx context.Context, santriID, pertemuanID int64, frekuensi string) (bool, error) {
	result, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO santri_billing_meeting (santri_id, pertemuan_id, frekuensi)
VALUES (?, ?, ?)
ON CONFLICT(santri_id, pertemuan_id) DO NOTHING
`, santriID, pertemuanID, frekuensi)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (q *Querier) ListUnprocessedBillingMeetingsForSantri(ctx context.Context, santriID int64) ([]SantriBillingMeetingRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT p.id,
       p.kelas_id,
       p.pertemuan_ke,
       p.tanggal,
       COALESCE(NULLIF(k.frekuensi, ''), NULLIF(s.frekuensi, ''), '') AS frekuensi,
       COALESCE(NULLIF(k.angkatan, ''), s.angkatan_kelas, '') AS angkatan_kelas
FROM absensi a
JOIN pertemuan p ON p.id = a.pertemuan_id
JOIN santri s ON s.id = a.santri_id
LEFT JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN santri_billing_meeting bm
       ON bm.santri_id = a.santri_id AND bm.pertemuan_id = a.pertemuan_id
WHERE a.santri_id = ?
  AND p.status = 'selesai'
  AND bm.id IS NULL
ORDER BY p.tanggal, p.id, a.id
`, santriID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SantriBillingMeetingRow{}
	for rows.Next() {
		var row SantriBillingMeetingRow
		if err := rows.Scan(
			&row.PertemuanID,
			&row.KelasID,
			&row.PertemuanKe,
			&row.Tanggal,
			&row.Frekuensi,
			&row.AngkatanKelas,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Querier) GetLatestProcessedBillingMeetingForSantri(ctx context.Context, santriID int64) (SantriBillingMeetingRow, error) {
	var row SantriBillingMeetingRow
	err := q.Queries.db.QueryRowContext(ctx, `
SELECT p.id,
       p.kelas_id,
       p.pertemuan_ke,
       p.tanggal,
       bm.frekuensi,
       COALESCE(NULLIF(k.angkatan, ''), s.angkatan_kelas, '') AS angkatan_kelas
FROM santri_billing_meeting bm
JOIN pertemuan p ON p.id = bm.pertemuan_id
JOIN santri s ON s.id = bm.santri_id
LEFT JOIN kelas k ON k.id = p.kelas_id
WHERE bm.santri_id = ?
ORDER BY p.tanggal DESC, p.id DESC, bm.id DESC
LIMIT 1
`, santriID).Scan(
		&row.PertemuanID,
		&row.KelasID,
		&row.PertemuanKe,
		&row.Tanggal,
		&row.Frekuensi,
		&row.AngkatanKelas,
	)
	return row, err
}

// --- Santri Admin Kelas note helpers ---

type SantriAdminNoteRow struct {
	ID            int64
	SantriID      int64
	Catatan       string
	AuthorName    string
	UpdatedByName string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (q *Querier) ListSantriAdminNotes(ctx context.Context, santriID int64) ([]SantriAdminNoteRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT n.id, n.santri_id, n.catatan,
       COALESCE(author.name, '') AS author_name,
       COALESCE(editor.name, '') AS updated_by_name,
       n.created_at, n.updated_at
FROM santri_admin_note n
LEFT JOIN users author ON author.id = n.author_user_id
LEFT JOIN users editor ON editor.id = n.updated_by_user_id
WHERE n.santri_id = ?
ORDER BY n.created_at DESC, n.id DESC
`, santriID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SantriAdminNoteRow{}
	for rows.Next() {
		var row SantriAdminNoteRow
		if err := rows.Scan(
			&row.ID, &row.SantriID, &row.Catatan,
			&row.AuthorName, &row.UpdatedByName,
			&row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Querier) CreateSantriAdminNote(ctx context.Context, santriID, actorID int64, catatan string) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO santri_admin_note (santri_id, author_user_id, updated_by_user_id, catatan)
VALUES (?, ?, ?, ?)
`, santriID, actorID, actorID, catatan)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (q *Querier) UpdateSantriAdminNote(ctx context.Context, santriID, noteID, actorID int64, catatan string) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `
UPDATE santri_admin_note
SET catatan = ?, updated_by_user_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND santri_id = ?
`, catatan, actorID, noteID, santriID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// isDuplicateEmail checks if the error is a duplicate email error
func isDuplicateEmail(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed: users.email") ||
		strings.Contains(msg, "UNIQUE constraint failed: users.google_id")
}
