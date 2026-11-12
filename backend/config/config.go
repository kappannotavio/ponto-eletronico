package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port               string
	DeviceAPIKey       string
	CORSAllowedOrigins []string
	Timezone           string
	SeedDemoData       bool
	DatabasePath       string
}

func Load() Config {
	seed, _ := strconv.ParseBool(env("SEED_DEMO_DATA", "false"))

	return Config{
		Port:               env("PORT", "8080"),
		DeviceAPIKey:       env("DEVICE_API_KEY", "dev-device-key"),
		CORSAllowedOrigins: strings.Split(env("CORS_ALLOWED_ORIGINS", "http://localhost:4200"), ","),
		Timezone:           env("APP_TIMEZONE", "America/Sao_Paulo"),
		SeedDemoData:       seed,
		DatabasePath:       env("DB_PATH", "ponto.db"),
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
