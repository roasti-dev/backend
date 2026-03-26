package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/middleware"
)

type Config struct {
	ServerPort                    string
	Env                           log.Env
	Debug                         bool
	SecureCookies                 bool
	RateLimit                     middleware.RateLimitConfig
	DBPath                        string
	UploadsPath                   string
	AppVersion                    string
	AllowedOrigins                []string
	FirebaseProjectID             string
	FirebaseCredentialsJSONBase64 string
	FirebaseAPIKey                string
	FirebaseIdentityBaseURL       string
	FirebaseTokenBaseURL          string
}

// ConfigFromEnv builds a Config by reading all relevant environment variables.
func ConfigFromEnv(appVersion string) Config {
	appEnv := log.Env(envOrDefault("APP_ENV", string(log.EnvDevelopment)))
	if !appEnv.IsValid() {
		slog.Warn("unknown APP_ENV, falling back to development", "value", appEnv)
		appEnv = log.EnvDevelopment
	}

	var allowedOrigins []string
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		for o := range strings.SplitSeq(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}

	debug := os.Getenv("DEBUG") != ""

	rateLimitCfg := middleware.DefaultRateLimitConfig()
	rateLimitCfg.Enabled = !debug

	return Config{
		ServerPort:                    envOrDefault("SERVER_PORT", "9090"),
		Env:                           appEnv,
		Debug:         debug,
		SecureCookies: appEnv == log.EnvProduction,
		RateLimit:     rateLimitCfg,
		DBPath:                        envOrDefault("DATABASE_PATH", "data.db"),
		UploadsPath:                   envOrDefault("UPLOADS_PATH", "./uploads"),
		AppVersion:                    appVersion,
		AllowedOrigins:                allowedOrigins,
		FirebaseProjectID:             os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAPIKey:                os.Getenv("FIREBASE_API_KEY"),
		FirebaseCredentialsJSONBase64: os.Getenv("FIREBASE_CREDENTIALS_JSON_BASE64"),
		FirebaseIdentityBaseURL:       envOrDefault("FIREBASE_IDENTITY_BASE_URL", "https://identitytoolkit.googleapis.com/v1/accounts"),
		FirebaseTokenBaseURL:          envOrDefault("FIREBASE_TOKEN_BASE_URL", "https://securetoken.googleapis.com/v1/token"),
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
