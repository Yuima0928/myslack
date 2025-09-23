package main

import (
	"context"
	"log"
	"os"
	"time"

	_ "slackgo/docs"
	"slackgo/internal/config"
	"slackgo/internal/db"
	httpapi "slackgo/internal/http"
	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	jwtutil "slackgo/internal/jwt"
	"slackgo/internal/storage"
	"slackgo/internal/ws"
)

// @title       MySlack API (Go)
// @version     0.1
// @description Minimal Slack-like backend with Gin + GORM + WS
// @BasePath    /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		log.Fatal(err)
	}
	if err := db.RunMigrations(sqlDB); err != nil {
		log.Fatal(err)
	}

	// ★ ここで S3 依存を初期化（MinIO でも AWS でも OK）
	s3deps, err := storage.NewS3Deps(context.Background(), cfg)
	if err != nil {
		log.Fatalf("init s3 deps failed: %v", err)
	}
	if s3deps.Bucket == "" {
		log.Fatal("S3_BUCKET が未設定です")
	}

	jm := jwtutil.New(cfg.JWTSecret, 7*24*time.Hour)
	hub := ws.NewHub()

	auth := handlers.NewAuthHandler(gdb, jm)
	msg := handlers.NewMessagesHandler(gdb, hub)

	ch := handlers.NewChannelsHandler(gdb)
	wsh := handlers.NewWorkspacesHandler(gdb)

	domain := "myslack-yuima.us.auth0.com"
	aud := "https://api.myslack.local"
	if domain == "" || aud == "" {
		log.Fatal("AUTH0_DOMAIN / AUTH0_AUDIENCE が未設定です")
	}

	router := httpapi.NewRouter(auth, msg, ch, wsh, middleware.JWTAuth0(gdb, middleware.Auth0Config{
		Domain:   domain,
		Audience: aud,
	}), hub, gdb, s3deps)

	log.Printf("listening on %s", cfg.BindAddr)
	if err := router.Run(cfg.BindAddr); err != nil {
		log.Fatal(err)
	}
	_ = os.Setenv // keep import
}
