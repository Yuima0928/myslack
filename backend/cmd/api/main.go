package main

import (
	"context"
	"log"
	"os"

	_ "slackgo/docs"
	"slackgo/internal/config"
	"slackgo/internal/db"
	httpapi "slackgo/internal/http"
	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/storage"
	"slackgo/internal/ws"

	authpkg "slackgo/internal/auth" // ← パッケージを別名でインポート
)

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

	// S3 依存初期化
	s3deps, err := storage.NewS3Deps(context.Background(), cfg)
	if err != nil {
		log.Fatalf("init s3 deps failed: %v", err)
	}
	if s3deps.Bucket == "" {
		log.Fatal("S3_BUCKET が未設定です")
	}

	// Handlers
	authH := handlers.NewAuthHandler(gdb) // ← 変数名を authH に
	hub := ws.NewHub()
	msgH := handlers.NewMessagesHandler(gdb, hub)
	chH := handlers.NewChannelsHandler(gdb)
	wsH := handlers.NewWorkspacesHandler(gdb)

	// WebSocket でも使う共通JWT Verifier
	verifier, err := authpkg.NewVerifier(context.Background(), authpkg.Config{
		Domain:   cfg.Auth0Domain,
		Audience: cfg.Auth0Audience,
	})
	if err != nil {
		log.Fatal(err)
	}

	jwtMw := middleware.JWTAuth0(gdb, verifier)

	// ルータ作成（NewRouter の引数順はあなたの定義に合わせて）
	router := httpapi.NewRouter(authH, msgH, chH, wsH, jwtMw, hub, gdb, s3deps, verifier)

	log.Printf("listening on %s", cfg.BindAddr)
	if err := router.Run(cfg.BindAddr); err != nil {
		log.Fatal(err)
	}
	_ = os.Setenv // keep import
}
