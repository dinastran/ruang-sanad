package middlewares

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSuperAdminRoleGuard(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, role TEXT NOT NULL);
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			data TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)
	require.NoError(t, err)

	querier := queries.NewQuerier(db)
	store := session.New(querier, nil, time.Hour)
	userService := services.NewUserService(querier)
	app := fiber.New()
	app.Delete("/santri/:id", RoleRequired(store, userService, "super_admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Put("/santri/:id/registration-identity", RoleRequired(store, userService, "super_admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	roles := []struct {
		role       string
		wantStatus int
	}{
		{role: "super_admin", wantStatus: fiber.StatusNoContent},
		{role: "cs", wantStatus: fiber.StatusForbidden},
		{role: "admin_kelas", wantStatus: fiber.StatusForbidden},
	}

	for i, tt := range roles {
		t.Run(tt.role, func(t *testing.T) {
			userID := int64(i + 1)
			sessionID := fmt.Sprintf("session-%d", userID)
			now := time.Now()
			_, err := db.Exec("INSERT INTO users (id, role) VALUES (?, ?)", userID, tt.role)
			require.NoError(t, err)
			_, err = db.Exec(
				"INSERT INTO sessions (id, user_id, data, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
				sessionID,
				userID,
				fmt.Sprintf(`{"user_id":%d,"email":"test@example.com","role":"%s"}`, userID, tt.role),
				now.Add(time.Hour),
				now,
				now,
			)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodDelete, "/santri/1", nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode)

			correctionReq := httptest.NewRequest(http.MethodPut, "/santri/1/registration-identity", nil)
			correctionReq.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
			correctionResp, err := app.Test(correctionReq)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, correctionResp.StatusCode)
		})
	}
}
