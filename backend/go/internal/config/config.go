package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	DBURL     string
	JWTSecret string
	BindAddr  string
}

func Load() Config {
	c := Config{
		DBURL:     env("DB_URL", "postgresql://app:app@localhost:5432/appdb"),
		JWTSecret: env("JWT_SECRET", "devjwtsecret_change_me"),
		BindAddr:  env("BIND_ADDR", ":8000"),
	}
	return c
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Printf("[config] %s not set, using default", key)
	return def
}

var _ = time.Now
