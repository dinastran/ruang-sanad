package queries

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:?_pragma=journal_mode(WAL)")
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		password TEXT,
		avatar TEXT DEFAULT '',
		role TEXT NOT NULL DEFAULT 'user',
		google_id TEXT UNIQUE,
		email_verified BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		data TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, q *Querier, ctx context.Context, email, name string) *models.User {
	t.Helper()

	user := &models.User{
		Email: email,
		Name:  name,
		Password: sql.NullString{
			String: "$2a$10$dummyhashdummyhashdummyhashdummyhashdummyhashdu",
			Valid:  true,
		},
		Role: models.RoleUser,
	}
	err := q.CreateUser(ctx, user)
	require.NoError(t, err)
	require.True(t, user.ID > 0)

	return user
}

func TestCreateUserAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "alice@example.com", "Alice")

	got, err := q.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, got.Email)
	assert.Equal(t, user.Name, got.Name)
	assert.Equal(t, models.RoleUser, got.Role)
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	createTestUser(t, q, ctx, "dup@example.com", "First")

	err := q.CreateUser(ctx, &models.User{
		Email: "dup@example.com",
		Name:  "Second",
		Password: sql.NullString{
			String: "hashed", Valid: true,
		},
		Role: models.RoleUser,
	})
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	createTestUser(t, q, ctx, "bob@example.com", "Bob")

	got, err := q.GetUserByEmail(ctx, "bob@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Bob", got.Name)

	_, err = q.GetUserByEmail(ctx, "nobody@example.com")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestGetUserByGoogleID(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := &models.User{
		Email:         "google@example.com",
		Name:          "Google User",
		GoogleID:      sql.NullString{String: "google-123", Valid: true},
		Avatar:        "https://example.com/avatar.jpg",
		EmailVerified: true,
		Role:          models.RoleUser,
	}
	err := q.CreateUserWithGoogleID(ctx, user)
	require.NoError(t, err)

	got, err := q.GetUserByGoogleID(ctx, "google-123")
	require.NoError(t, err)
	assert.Equal(t, user.Name, got.Name)

	_, err = q.GetUserByGoogleID(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "update@example.com", "Original")

	user.Name = "Updated"
	user.Avatar = "/storage/avatar.jpg"
	err := q.UpdateUser(ctx, user)
	require.NoError(t, err)

	got, err := q.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, "/storage/avatar.jpg", got.Avatar)
}

func TestUpdateUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	err := q.UpdateUser(ctx, &models.User{ID: 999, Name: "Ghost"})
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "delete@example.com", "Delete Me")

	err := q.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	_, err = q.GetUserByID(ctx, user.ID)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestDeleteUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	err := q.DeleteUser(ctx, 999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestSetUserRoleAdmin(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "role@example.com", "Role Test")

	err := q.SetUserRoleAdmin(ctx, user.ID)
	require.NoError(t, err)

	got, err := q.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, got.Role)
}

func TestCreateAndGetSession(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "session@example.com", "Session User")

	sess := &Session{
		ID:        "test-session-id",
		UserID:    user.ID,
		Data:      fmt.Sprintf(`{"user_id":%d,"email":"session@example.com","role":"user"}`, user.ID),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := q.CreateSession(ctx, sess)
	require.NoError(t, err)

	got, err := q.GetSessionByID(ctx, "test-session-id")
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, sess.UserID, got.UserID)
}

func TestGetSessionExpired(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "expired@example.com", "Expired")

	sess := &Session{
		ID:        "expired-session",
		UserID:    user.ID,
		Data:      `{}`,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, q.CreateSession(ctx, sess))

	// GetSessionByID no longer checks expiry — expiry is handled by session.Store.Get()
	got, err := q.GetSessionByID(ctx, "expired-session")
	require.NoError(t, err)
	assert.Equal(t, "expired-session", got.ID)
	assert.Equal(t, user.ID, got.UserID)
}

func TestUpdateSession(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "session-update@example.com", "Update Session")

	sess := &Session{
		ID:        "update-session",
		UserID:    user.ID,
		Data:      `{}`,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, q.CreateSession(ctx, sess))

	sess.Data = `{"role":"admin"}`
	err := q.UpdateSession(ctx, sess)
	require.NoError(t, err)

	got, err := q.GetSessionByID(ctx, "update-session")
	require.NoError(t, err)
	assert.Contains(t, got.Data, "admin")
}

func TestGetSessionsByUserID(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "multi-session@example.com", "Multi")

	for i := 0; i < 3; i++ {
		sess := &Session{
			ID:        fmt.Sprintf("multi-session-%d", i),
			UserID:    user.ID,
			Data:      `{}`,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, q.CreateSession(ctx, sess))
	}

	sessions, err := q.GetSessionsByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestDeleteSessionsByUserID(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "delete-sessions@example.com", "Delete")

	sess := &Session{
		ID:        "to-delete",
		UserID:    user.ID,
		Data:      `{}`,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, q.CreateSession(ctx, sess))

	require.NoError(t, q.DeleteSessionsByUserID(ctx, user.ID))

	sessions, err := q.GetSessionsByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestDeleteExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := createTestUser(t, q, ctx, "expired-cleanup@example.com", "Cleanup")

	sess1 := &Session{
		ID:        "expired-1",
		UserID:    user.ID,
		Data:      `{}`,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, q.CreateSession(ctx, sess1))

	sess2 := &Session{
		ID:        "active-1",
		UserID:    user.ID,
		Data:      `{}`,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, q.CreateSession(ctx, sess2))

	require.NoError(t, q.DeleteExpiredSessions(ctx))

	sessions, err := q.GetSessionsByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "active-1", sessions[0].ID)
}

func TestIsDuplicateEmail(t *testing.T) {
	assert.True(t, isDuplicateEmail(fmt.Errorf("UNIQUE constraint failed: users.email")))
	assert.False(t, isDuplicateEmail(nil))
	assert.False(t, isDuplicateEmail(fmt.Errorf("some other error")))
}

func TestNullStringToString(t *testing.T) {
	assert.Equal(t, "hello", nullStringToString(sql.NullString{String: "hello", Valid: true}))
	assert.Equal(t, "", nullStringToString(sql.NullString{String: "", Valid: false}))
}

func TestCreateUserWithGoogleID(t *testing.T) {
	db := setupTestDB(t)
	q := NewQuerier(db)
	ctx := context.Background()

	user := &models.User{
		Email:         "google-new@example.com",
		Name:          "Google New",
		GoogleID:      sql.NullString{String: "google-new-456", Valid: true},
		Avatar:        "https://example.com/pic.jpg",
		EmailVerified: true,
		Role:          models.RoleUser,
	}
	err := q.CreateUserWithGoogleID(ctx, user)
	require.NoError(t, err)
	assert.True(t, user.ID > 0)

	got, err := q.GetUserByGoogleID(ctx, "google-new-456")
	require.NoError(t, err)
	assert.Equal(t, "Google New", got.Name)
	assert.Equal(t, "https://example.com/pic.jpg", got.Avatar)
	assert.True(t, got.EmailVerified)
}

func TestUserToModel(t *testing.T) {
	now := time.Now()
	qUser := User{
		ID:            1,
		Email:         "test@example.com",
		Name:          "Test",
		Password:      sql.NullString{String: "hashed", Valid: true},
		Avatar:        sql.NullString{String: "/storage/avatar.jpg", Valid: true},
		Role:          "user",
		GoogleID:      sql.NullString{String: "g-1", Valid: true},
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mUser := toModelUser(qUser)
	assert.Equal(t, qUser.ID, mUser.ID)
	assert.Equal(t, qUser.Email, mUser.Email)
	assert.Equal(t, "/storage/avatar.jpg", mUser.Avatar)
	assert.Equal(t, "user", string(mUser.Role))
	assert.True(t, mUser.EmailVerified)
}
