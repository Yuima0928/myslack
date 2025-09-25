// internal/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Config struct {
	DBURL         string
	JWTSecret     string
	BindAddr      string
	Auth0Domain   string
	Auth0Audience string

	// S3/MinIO 共通
	AWSRegion      string
	S3Bucket       string
	S3Prefix       string // 任意: "myslack" 等。""でもOK
	S3URLExpirySec int    // 署名URLの有効期限(秒)。例: 900=15分

	// MinIO / カスタムS3互換向け
	S3Endpoint       string // 例: "http://localhost:9000"（AWS利用なら空に）
	S3PublicEndpoint string
	S3AccessKey      string // MinIO: MINIO_ROOT_USER
	S3SecretKey      string // MinIO: MINIO_ROOT_PASSWORD
	S3UsePathStyle   bool   // MinIOは true 推奨（AWSは false が既定）
}

func Load() Config {
	c := Config{
		DBURL:         env("DB_URL", "postgresql://app:app@localhost:5432/appdb"),
		JWTSecret:     env("JWT_SECRET", "devjwtsecret_change_me"),
		BindAddr:      env("BIND_ADDR", ":8000"),
		Auth0Domain:   env("AUTH0_DOMAIN", ""),
		Auth0Audience: env("AUTH0_AUDIENCE", ""),

		AWSRegion:      env("AWS_REGION", "ap-northeast-1"),
		S3Bucket:       mustEnv("S3_BUCKET"), // 必須にしたいなら mustEnv。任意なら env(...,"")
		S3Prefix:       env("S3_PREFIX", "myslack"),
		S3URLExpirySec: envInt("S3_URL_EXPIRE_SEC", 900),

		S3Endpoint:       env("S3_ENDPOINT", ""),        // MinIO 例: "http://localhost:9000"
		S3PublicEndpoint: env("S3_PUBLIC_ENDPOINT", ""), // ← 追加
		S3AccessKey:      env("S3_ACCESS_KEY", ""),
		S3SecretKey:      env("S3_SECRET_KEY", ""),
		S3UsePathStyle:   envBool("S3_USE_PATH_STYLE", true), // MinIO既定true、AWSならfalseでもOK
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] %s is required", key)
	}
	return v
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err == nil {
			return n
		}
		log.Printf("[config] %s parse error, using default %d", key, def)
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if v == "1" || v == "true" || v == "TRUE" || v == "True" {
			return true
		}
		if v == "0" || v == "false" || v == "FALSE" || v == "False" {
			return false
		}
		log.Printf("[config] %s parse error, using default %v", key, def)
	}
	return def
}

var _ = time.Now
