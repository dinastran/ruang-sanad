package handlers

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestSantriListReturnURL(t *testing.T) {
	cases := map[string]string{
		"":                                       "/app/santri",
		"/app/santri":                            "/app/santri",
		"/app/santri?status=cuti&page=2":         "/app/santri?status=cuti&page=2",
		"https://evil.example/app/santri?page=2": "/app/santri",
		"//evil.example/app/santri?page=2":       "/app/santri",
		"/app/kelas?page=2":                      "/app/santri",
	}
	for returnTo, want := range cases {
		app := fiber.New()
		var got string
		app.Get("/", func(c *fiber.Ctx) error {
			got = santriListReturnURL(c)
			return nil
		})
		_, err := app.Test(httptest.NewRequest("GET", "/?return_to="+url.QueryEscape(returnTo), nil))
		require.NoError(t, err)
		require.Equal(t, want, got, returnTo)
	}
}
