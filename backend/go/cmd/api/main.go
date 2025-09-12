package main

import (
	"log"
	"os"
	"time"

	"slackgo/internal/config"
	"slackgo/internal/db"
	httpapi "slackgo/internal/http"
	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/jwt"
	"slackgo/internal/ws"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	jm := jwtutil.New(cfg.JWTSecret, 7*24*time.Hour)
	hub := ws.NewHub()

	auth := handlers.NewAuthHandler(gdb, jm)
	msg := handlers.NewMessagesHandler(gdb, hub)
	router := httpapi.NewRouter(auth, msg, middleware.JWT(jm), hub)

	log.Printf("listening on %s", cfg.BindAddr)
	if err := router.Run(cfg.BindAddr); err != nil {
		log.Fatal(err)
	}
	_ = os.Setenv // keep import
}
