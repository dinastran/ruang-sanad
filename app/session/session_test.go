package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maulanashalihin/laju-go/app/cache"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const (
	uaChrome130Android = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36"
	uaChrome131Android = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36"
	uaFirefoxWindows   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0"
)

const ttl30 = 30 * 24 * time.Hour

type sesiTest struct {
	db    *sql.DB
	q     *queries.Querier
	store *Store
	app   *fiber.App
}

func siapkan(t *testing.T) *sesiTest {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec("INSERT INTO users (id, email, name, role) VALUES (1, 'a@test.local', 'A', 'user')")
	require.NoError(t, err)
	q := queries.NewQuerier(db)
	st := &sesiTest{db: db, q: q, store: New(q, cache.NewSessionCache(), ttl30)}
	st.app = fiber.New()
	st.app.Get("/", func(c *fiber.Ctx) error {
		sess, err := st.store.Get(c)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"uid": sess.Get("user_id")})
	})
	return st
}

func (st *sesiTest) buatSesi(t *testing.T, id, ua string, expires time.Time) {
	t.Helper()
	data, _ := json.Marshal(SessionData{UserID: 1, Email: "a@test.local", Role: "user", UserAgent: ua})
	require.NoError(t, st.q.CreateSession(context.Background(), &queries.Session{ID: id, UserID: 1, Data: string(data), ExpiresAt: expires}))
}

// minta melakukan navigasi halaman (dicek UA) dengan cookie sesi.
func (st *sesiTest) minta(t *testing.T, id, ua string) (uid any, setCookie string) {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", "session_id="+id)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html")
	resp, err := st.app.Test(req, -1)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["uid"], resp.Header.Get("Set-Cookie")
}

func (st *sesiTest) kedaluwarsa(t *testing.T, id string) time.Time {
	t.Helper()
	s, err := st.q.GetSessionByID(context.Background(), id)
	require.NoError(t, err)
	return s.ExpiresAt
}

func TestSlidingMemperpanjangSesiYangDipakai(t *testing.T) {
	st := siapkan(t)
	st.buatSesi(t, "s1", uaChrome130Android, time.Now().Add(10*24*time.Hour))
	uid, cookie := st.minta(t, "s1", uaChrome130Android)
	require.EqualValues(t, 1, uid)
	require.WithinDuration(t, time.Now().Add(ttl30), st.kedaluwarsa(t, "s1"), time.Minute)
	require.Contains(t, cookie, "session_id=s1")
	require.Contains(t, strings.ToLower(cookie), "max-age=259")
}

func TestSlidingTidakMenulisUlangSesiYangMasihBaru(t *testing.T) {
	st := siapkan(t)
	awal := time.Now().Add(ttl30 - 2*time.Hour).Truncate(time.Second)
	st.buatSesi(t, "s2", uaChrome130Android, awal)
	uid, cookie := st.minta(t, "s2", uaChrome130Android)
	require.EqualValues(t, 1, uid)
	require.WithinDuration(t, awal, st.kedaluwarsa(t, "s2"), time.Second, "diperpanjang maksimal sekali per hari")
	require.NotContains(t, cookie, "session_id=")
}

func TestUpdateVersiBrowserTidakMembuatLogout(t *testing.T) {
	st := siapkan(t)
	st.buatSesi(t, "s3", uaChrome130Android, time.Now().Add(ttl30))
	for i := 0; i < 2; i++ { // jalur DB lalu jalur cache
		uid, _ := st.minta(t, "s3", uaChrome131Android)
		require.EqualValues(t, 1, uid, "permintaan ke-%d", i+1)
	}
}

func TestBrowserBerbedaDitolak(t *testing.T) {
	st := siapkan(t)
	st.buatSesi(t, "s4", uaChrome130Android, time.Now().Add(ttl30))
	uid, _ := st.minta(t, "s4", uaChrome130Android) // isi cache
	require.EqualValues(t, 1, uid)
	uid, _ = st.minta(t, "s4", uaFirefoxWindows)
	require.Nil(t, uid)
	_, err := st.q.GetSessionByID(context.Background(), "s4")
	require.ErrorIs(t, err, queries.ErrSessionNotFound)
}

func TestFamiliUserAgent(t *testing.T) {
	sama := [][2]string{
		{uaChrome130Android, uaChrome131Android},
		// Chrome Android "situs desktop"
		{uaChrome130Android, "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
		// iPhone Safari update iOS
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1"},
		// iPad "situs desktop" melaporkan Macintosh
		{"Mozilla/5.0 (iPad; CPU OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0"},
	}
	for _, p := range sama {
		require.Equal(t, famili(p[0]), famili(p[1]), "%s\n%s", p[0], p[1])
	}
	beda := [][2]string{
		{uaChrome130Android, uaFirefoxWindows},
		{uaChrome130Android, "Mozilla/5.0 (Linux; Android 10; SM-A515F) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/26.0 Chrome/122.0.0.0 Mobile Safari/537.36"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36", uaChrome131Android},
	}
	for _, p := range beda {
		require.NotEqual(t, famili(p[0]), famili(p[1]), "%s\n%s", p[0], p[1])
	}
}

func TestTujuanLoginHanyaPathAplikasi(t *testing.T) {
	aman := map[string]string{
		"/app":                    "/app",
		"/app/kelas/7?tab=santri": "/app/kelas/7?tab=santri",
		"/admin/users":            "/admin/users",
		"/app/tasbih":             "/app/tasbih",
	}
	for in, want := range aman {
		require.Equal(t, want, tujuanAman(in), in)
	}
	for _, in := range []string{"", "//evil.com", "https://evil.com/app", "/login", "/logout", "/apps", "/app\\evil", "/app/../login", "javascript:alert(1)", "/app/" + strings.Repeat("a", 600)} {
		require.Equal(t, "", tujuanAman(in), in)
	}
}
