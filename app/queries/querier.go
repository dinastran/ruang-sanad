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

// isDuplicateEmail checks if the error is a duplicate email error
func isDuplicateEmail(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed: users.email") ||
		strings.Contains(msg, "UNIQUE constraint failed: users.google_id")
}
