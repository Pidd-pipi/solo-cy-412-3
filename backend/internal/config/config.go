package config

import "os"

type Config struct {
	Port, DBDriver, DSN, JWTSecret string
	RateLimit                      int
}

func Load() Config {
	c := Config{Port: getenv("APP_PORT", "8080"), DBDriver: getenv("DB_DRIVER", "sqlite"), DSN: getenv("DB_DSN", "smartestate.db"), JWTSecret: getenv("JWT_SECRET", "dev-secret-change-me"), RateLimit: RateLimitPerMinute()}
	return c
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
