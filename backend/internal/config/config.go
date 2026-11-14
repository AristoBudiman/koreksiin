package config

import (
	"os"
	"strconv"
)

type Config struct {
	BackendPort          string
	DatabaseURL          string
	RedisURL             string
	ScoringQueue         string
	TrainingQueue        string
	JWTSecret            string
	JWTExpiresHours      int
	AdminName            string
	AdminEmail           string
	AdminPassword        string
	StorageDir           string
	MinTrainSamples      int
	DefaultTHigh         float64
	DefaultTLow          float64
	DefaultSSem          float64
	DefaultSLex          float64
	MidtransServerKey    string
	MidtransClientKey    string
	MidtransIsProduction bool
	PublicBaseURL        string
	TunnelMetricsURL     string
}

func Load() *Config {
	return &Config{
		BackendPort:          getEnv("BACKEND_PORT", "8000"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://koreksiin:koreksiin_secret_db_pass@postgres:5432/koreksiin_db?sslmode=disable"),
		RedisURL:             getEnv("REDIS_URL", "redis://redis:6379/0"),
		ScoringQueue:         getEnv("SCORING_QUEUE", "queue:scoring"),
		TrainingQueue:        getEnv("TRAINING_QUEUE", "queue:training"),
		JWTSecret:            getEnv("JWT_SECRET", "koreksiin_super_secret_jwt_key_2026_local_dev"),
		JWTExpiresHours:      getEnvInt("JWT_EXPIRES_HOURS", 72),
		AdminName:            getEnv("ADMIN_NAME", "Administrator Koreksi.in"),
		AdminEmail:           getEnv("ADMIN_EMAIL", "admin@koreksi.in"),
		AdminPassword:        getEnv("ADMIN_PASSWORD", "AdminKoreksiin2026!"),
		StorageDir:           getEnv("STORAGE_DIR", "/storage"),
		MinTrainSamples:      getEnvInt("MIN_TRAIN_SAMPLES", 20),
		DefaultTHigh:         getEnvFloat("DEFAULT_T_HIGH", 0.60),
		DefaultTLow:          getEnvFloat("DEFAULT_T_LOW", 0.45),
		DefaultSSem:          getEnvFloat("DEFAULT_S_SEM", 0.92),
		DefaultSLex:          getEnvFloat("DEFAULT_S_LEX", 0.60),
		MidtransServerKey:    getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey:    getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransIsProduction: getEnv("MIDTRANS_IS_PRODUCTION", "false") == "true",
		PublicBaseURL:        getEnv("PUBLIC_BASE_URL", ""),
		TunnelMetricsURL:     getEnv("TUNNEL_METRICS_URL", "http://tunnel:2000/quicktunnel"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

