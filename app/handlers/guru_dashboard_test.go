package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/middlewares"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/maulanashalihin/laju-go/app/services"
	"github.com/maulanashalihin/laju-go/app/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestGuruDashboardRedirectsTerminate(t *testing.T) {
	for _, scenario := range []string{"missing guru", "inactive guru", "dashboard query failure", "healthy guru"} {
		t.Run(scenario, func(t *testing.T) {
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(t, err)
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			require.NoError(t, goose.SetDialect("sqlite3"))
			require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
			result, err := db.Exec(`INSERT INTO users (email, name, password, role) VALUES ('teacher@example.com', 'Teacher', ?, 'guru')`, hashPW(t, "password123"))
			require.NoError(t, err)
			userID, err := result.LastInsertId()
			require.NoError(t, err)
			if scenario != "missing guru" {
				active := 1
				if scenario == "inactive guru" {
					active = 0
				}
				_, err = db.Exec(`INSERT INTO guru (nama, user_id, is_aktif) VALUES ('Teacher', ?, ?)`, userID, active)
				require.NoError(t, err)
			}
			if scenario == "dashboard query failure" {
				_, err = db.Exec(`DROP TABLE kelas`)
				require.NoError(t, err)
			}

			q := queries.NewQuerier(db)
			store := session.New(q, nil, time.Hour)
			inertia := services.NewInertiaService(services.NewAssetService("", "", false), store)
			auth := NewAuthHandler(services.NewAuthService(q, services.AuthServiceConfig{SessionSecret: "test-secret-32-chars-long-for-testing!!"}), store, inertia)
			guru := NewGuruHandler(services.NewGuruService(q), nil, nil, store, inertia)
			dashboard := NewDashboardHandler(nil, nil, nil, store, inertia)
			profile := NewAppHandler(services.NewUserService(q), store, inertia)
			app := fiber.New()
			app.Get("/login", middlewares.Guest(store), auth.ShowLoginForm)
			app.Post("/login", middlewares.Guest(store), auth.Login)
			protected := app.Group("/app", middlewares.AuthRequired(store))
			protected.Get("/", dashboard.Index)
			protected.Get("/profile", profile.Profile)
			protected.Get("/guru", middlewares.RoleRequired(store, services.NewUserService(q), "guru", "super_admin"), guru.Dashboard)

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"teacher@example.com","password":"password123"}`))
			req.Header.Set("Content-Type", "application/json")
			login, err := app.Test(req)
			require.NoError(t, err)
			login.Body.Close()
			require.Equal(t, http.StatusSeeOther, login.StatusCode)
			require.Equal(t, "/app/guru", login.Header.Get("Location"))
			var cookie *http.Cookie
			for _, c := range login.Cookies() {
				if c.Name == "session_id" {
					cookie = c
				}
			}
			require.NotNil(t, cookie)

			for _, start := range []string{login.Header.Get("Location"), "/app", "/login"} {
				for _, xhr := range []bool{false, true} {
					path := start
					visited := map[string]bool{}
					for {
						require.False(t, visited[path], "redirect cycle from %s (Inertia=%v): revisited %s", start, xhr, path)
						visited[path] = true
						req := httptest.NewRequest(http.MethodGet, path, nil)
						req.AddCookie(cookie)
						if xhr {
							req.Header.Set("X-Inertia", "true")
						}
						resp, err := app.Test(req)
						require.NoError(t, err)
						location := resp.Header.Get("Location")
						if location != "" {
							resp.Body.Close()
							require.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, resp.StatusCode)
							path = location
							continue
						}
						require.Equal(t, http.StatusOK, resp.StatusCode)
						component := "app/Profile"
						if scenario == "healthy guru" {
							require.Equal(t, "/app/guru", path)
							component = "guru/Dashboard"
						} else {
							require.Equal(t, "/app/profile", path)
						}
						if xhr {
							var page struct {
								Component string `json:"component"`
							}
							require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
							require.Equal(t, component, page.Component)
						}
						resp.Body.Close()
						break
					}
				}
			}
		})
	}
}
