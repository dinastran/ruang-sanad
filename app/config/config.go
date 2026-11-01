package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort            string
	AppBindHost        string
	AppEnv             string
	DBPath             string
	SessionSecret      string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	// AppURL is the public base URL (e.g. https://dashboard.ruangsanad.com),
	// used for links in AI agent answers. Empty means derive from the request.
	AppURL string
	// CORS
	AllowedOrigins []string
	// Email configuration
	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
	FromEmail string
	FromName  string
	// Session
	SessionTTL time.Duration
	// CRM Leads / CS AI bridge (PRD CRM). Secrets live only in .env.
	CSBridgeIngestToken   string
	CSBridgeAllowedCIDRs  []string
	CSBridgeURL           string
	CSBridgeOutboundToken string
	GowaURL               string
	GowaUser              string
	GowaPass              string

	// Meta Conversions API (CAPI): sends the CRM funnel to Meta. Off until
	// the dataset (Pixel) ID and token are set. See docs/panduan/meta-ads-api.md.
	MetaCAPIDatasetID     string
	MetaCAPIToken         string
	MetaCAPITestEventCode string
	// Default landing page URL for event_source_url (sources may override).
	MetaCAPIEventSourceURL string
	MetaAPIVersion         string
	// Only set when the CAPI token belongs to the app with this secret
	// (e.g. a System User token); an Events Manager token may not.
	MetaCAPIAppSecret string

	// Meta Ads Insights (read-only): ad spend and delivery for the
	// Performa Iklan page. Off until the token and an ad account are set.
	MetaAdsToken     string
	MetaAdAccountIDs []string
	MetaAppSecret    string
}

var AppConfig *Config

func Load() *Config {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	AppConfig = &Config{
		AppPort:            getEnv("APP_PORT", "8080"),
		AppBindHost:        strings.TrimSpace(getEnv("APP_BIND_HOST", "")),
		AppEnv:             getEnv("APP_ENV", "development"),
		DBPath:             getEnv("DB_PATH", "./data/app.db"),
		SessionSecret:      getEnv("SESSION_SECRET", "change-this-in-production"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		AppURL:             strings.TrimRight(getEnv("APP_URL", ""), "/"),
		AllowedOrigins:     parseAllowedOrigins(),
		// Email configuration
		SMTPHost:  getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:  getEnvAsInt("SMTP_PORT", 587),
		SMTPUser:  getEnv("SMTP_USER", ""),
		SMTPPass:  getEnv("SMTP_PASS", ""),
		FromEmail: getEnv("FROM_EMAIL", "noreply@example.com"),
		FromName:  getEnv("FROM_NAME", "Laju"),
		// Session
		SessionTTL: getSessionTTL(),
		// CRM Leads / CS AI bridge
		CSBridgeIngestToken:   strings.TrimSpace(getEnv("CS_BRIDGE_INGEST_TOKEN", "")),
		CSBridgeAllowedCIDRs:  splitList(getEnv("CS_BRIDGE_ALLOWED_CIDRS", "127.0.0.1/32,::1/128,172.16.0.0/12")),
		CSBridgeURL:           strings.TrimRight(getEnv("CS_BRIDGE_URL", ""), "/"),
		CSBridgeOutboundToken: strings.TrimSpace(getEnv("CS_BRIDGE_OUTBOUND_TOKEN", "")),
		GowaURL:               strings.TrimRight(getEnv("GOWA_URL", ""), "/"),
		GowaUser:              getEnv("GOWA_USER", ""),
		GowaPass:              getEnv("GOWA_PASS", ""),

		MetaCAPIDatasetID:      strings.TrimSpace(getEnv("META_CAPI_DATASET_ID", "")),
		MetaCAPIToken:          strings.TrimSpace(getEnv("META_CAPI_TOKEN", "")),
		MetaCAPITestEventCode:  strings.TrimSpace(getEnv("META_CAPI_TEST_EVENT_CODE", "")),
		MetaCAPIEventSourceURL: strings.TrimSpace(getEnv("META_CAPI_EVENT_SOURCE_URL", "")),
		MetaAPIVersion:         strings.TrimSpace(getEnv("META_API_VERSION", "")),
		MetaCAPIAppSecret:      strings.TrimSpace(getEnv("META_CAPI_APP_SECRET", "")),

		MetaAdsToken:     strings.TrimSpace(getEnv("META_ADS_TOKEN", "")),
		MetaAdAccountIDs: splitList(getEnv("META_AD_ACCOUNT_IDS", "")),
		MetaAppSecret:    strings.TrimSpace(getEnv("META_APP_SECRET", "")),
	}

	return AppConfig
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("file:%s?_foreign_keys=on", c.DBPath)
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// parseAllowedOrigins parses ALLOWED_ORIGINS env var (comma-separated).
// Defaults to http://localhost:5173 in dev, empty (strict) in prod.
func parseAllowedOrigins() []string {
	val := os.Getenv("ALLOWED_ORIGINS")
	if val == "" {
		return []string{"http://localhost:5173"}
	}
	var origins []string
	for _, o := range strings.Split(val, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func splitList(val string) []string {
	var out []string
	for _, item := range strings.Split(val, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getSessionTTL() time.Duration {
	val := getEnv("SESSION_TTL", "720h")
	d, err := time.ParseDuration(val)
	if err != nil {
		return 720 * time.Hour // 30 days
	}
	return d
}

// ListenAddress preserves production's existing wildcard binding by default.
// Staging can opt into loopback without changing production networking.
func (c *Config) ListenAddress() string {
	if c.AppBindHost == "" {
		return ":" + c.AppPort
	}
	return net.JoinHostPort(c.AppBindHost, c.AppPort)
}
