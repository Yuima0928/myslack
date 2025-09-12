set -euo pipefail

# ルートを現在ディレクトリに
ROOT="$(pwd)"
echo "[*] Project root: $ROOT"

# ディレクトリ作成
mkdir -p backend/go/cmd/api
mkdir -p backend/go/internal/{config,db,model,jwt,ws}
mkdir -p backend/go/internal/http/{handlers,middleware}
mkdir -p docker

########################
# go.mod
########################
cat > backend/go/go.mod <<'EOF'
module github.com/example/slackgo

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.1
	golang.org/x/crypto v0.28.0
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.7
)
EOF

########################
# config.go（env読み込み）
########################
cat > backend/go/internal/config/config.go <<'EOF'
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
EOF

########################
# model.go（GORMモデル）
########################
cat > backend/go/internal/model/model.go <<'EOF'
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	DisplayName  *string
	CreatedAt    time.Time
}

type Workspace struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time
}

type Channel struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	IsPrivate   bool
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

type Message struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"`
	ChannelID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	Text        *string
	CreatedAt   time.Time
	EditedAt    *time.Time
}

func EnableExtensions(db *gorm.DB) error {
	return db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error
}
EOF

########################
# db.go（GORM初期化）
########################
cat > backend/go/internal/db/db.go <<'EOF'
package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/example/slackgo/backend/go/internal/model"
)

func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := model.EnableExtensions(gdb); err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(&model.User{}, &model.Workspace{}, &model.Channel{}, &model.Message{}); err != nil {
		return nil, err
	}
	return gdb, nil
}
EOF

########################
# jwt.go（JWTユーティリティ）
########################
cat > backend/go/internal/jwt/jwt.go <<'EOF'
package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Maker struct {
	secret []byte
	ttl    time.Duration
}

func New(secret string, ttl time.Duration) *Maker {
	return &Maker{secret: []byte(secret), ttl: ttl}
}

func (m *Maker) Create(sub string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": sub,
		"iat": now.Unix(),
		"exp": now.Add(m.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Maker) Parse(tokenStr string) (jwt.MapClaims, error) {
	tk, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if c, ok := tk.Claims.(jwt.MapClaims); ok && tk.Valid {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
EOF

########################
# ws/hub.go（WSハブ）
########################
cat > backend/go/internal/ws/hub.go <<'EOF'
package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu       sync.RWMutex
	channels map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{channels: map[string]map[*websocket.Conn]struct{}{}}
}

func (h *Hub) Join(channel string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = map[*websocket.Conn]struct{}{}
	}
	h.channels[channel][conn] = struct{}{}
}

func (h *Hub) Leave(channel string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.channels[channel], conn)
}

func (h *Hub) Broadcast(channel string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.channels[channel] {
		_ = c.WriteMessage(websocket.TextMessage, payload)
	}
}
EOF

########################
# middleware/jwt.go
########################
cat > backend/go/internal/http/middleware/jwt.go <<'EOF'
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/example/slackgo/backend/go/internal/jwt"
)

func JWT(m *jwtutil.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Missing bearer"})
			return
		}
		raw := strings.TrimPrefix(auth, "Bearer ")
		claims, err := m.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token"})
			return
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid sub"})
			return
		}
		c.Set("user_id", sub)
		c.Next()
	}
}
EOF

########################
# handlers/auth.go
########################
cat > backend/go/internal/http/handlers/auth.go <<'EOF'
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/example/slackgo/backend/go/internal/jwt"
	"github.com/example/slackgo/backend/go/internal/model"
)

type AuthHandler struct {
	db  *gorm.DB
	jwt *jwtutil.Maker
}

func NewAuthHandler(db *gorm.DB, jm *jwtutil.Maker) *AuthHandler {
	return &AuthHandler{db: db, jwt: jm}
}

type SignUpIn struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=4"`
	DisplayName string `json:"display_name"`
}
type TokenOut struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var in SignUpIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	var exist model.User
	if err := h.db.Where("email = ?", in.Email).First(&exist).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"detail": "Email already registered"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	u := model.User{Email: in.Email, PasswordHash: string(hash)}
	if in.DisplayName != "" {
		u.DisplayName = &in.DisplayName
	}
	if err := h.db.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "create user failed"})
		return
	}
	t, _ := h.jwt.Create(u.ID.String())
	c.JSON(http.StatusOK, TokenOut{AccessToken: t, TokenType: "bearer"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in SignUpIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", in.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid credentials"})
		return
	}
	t, _ := h.jwt.Create(u.ID.String())
	c.JSON(http.StatusOK, TokenOut{AccessToken: t, TokenType: "bearer"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.GetString("user_id")})
}
EOF

########################
# handlers/messages.go
########################
cat > backend/go/internal/http/handlers/messages.go <<'EOF'
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/slackgo/backend/go/internal/model"
	"github.com/example/slackgo/backend/go/internal/ws"
)

type MessagesHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewMessagesHandler(db *gorm.DB, hub *ws.Hub) *MessagesHandler {
	return &MessagesHandler{db: db, hub: hub}
}

type MsgCreateIn struct {
	WorkspaceID uuid.UUID `json:"workspace_id" binding:"required"`
	ChannelID   uuid.UUID `json:"channel_id" binding:"required"`
	Text        string    `json:"text" binding:"required,min=1"`
}
type MsgOut struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	UserID      uuid.UUID `json:"user_id"`
	Text        string    `json:"text"`
}

func (h *MessagesHandler) Create(c *gin.Context) {
	var in MsgCreateIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	uid := uuid.MustParse(c.GetString("user_id"))
	text := in.Text
	m := model.Message{
		WorkspaceID: in.WorkspaceID,
		ChannelID:   in.ChannelID,
		UserID:      &uid,
		Text:        &text,
	}
	if err := h.db.Create(&m).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "insert failed"})
		return
	}
	out := MsgOut{ID: m.ID, WorkspaceID: m.WorkspaceID, ChannelID: m.ChannelID, UserID: *m.UserID, Text: *m.Text}
	c.JSON(http.StatusOK, out)

	ev := map[string]any{"type": "message_created", "message": out}
	b, _ := json.Marshal(ev)
	h.hub.Broadcast(m.ChannelID.String(), b)
}

func (h *MessagesHandler) List(c *gin.Context) {
	chID := c.Query("channel_id")
	if chID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "channel_id is required"})
		return
	}
	var rows []model.Message
	if err := h.db.Where("channel_id = ?", chID).
		Order("created_at DESC").Limit(50).Offset(0).Find(&rows).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "list failed"})
		return
	}
	out := make([]MsgOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, MsgOut{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ChannelID: r.ChannelID, UserID: *r.UserID, Text: deref(r.Text),
		})
	}
	c.JSON(http.StatusOK, out)
}

func deref(s *string) string { if s==nil { return ""}; return *s }
EOF

########################
# handlers/health.go
########################
cat > backend/go/internal/http/handlers/health.go <<'EOF'
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
EOF

########################
# router.go（Ginルーター）
########################
cat > backend/go/internal/http/router.go <<'EOF'
package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/example/slackgo/backend/go/internal/http/handlers"
	"github.com/example/slackgo/backend/go/internal/http/middleware"
	"github.com/example/slackgo/backend/go/internal/ws"
)

func NewRouter(
	auth *handlers.AuthHandler,
	msg  *handlers.MessagesHandler,
	jwtMw gin.HandlerFunc,
	hub *ws.Hub,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", handlers.Health)

	r.POST("/auth/signup", auth.SignUp)
	r.POST("/auth/login", auth.Login)
	r.GET("/auth/me", jwtMw, auth.Me)

	msgGroup := r.Group("/messages")
	msgGroup.Use(jwtMw)
	msgGroup.POST("", msg.Create)
	r.GET("/messages", msg.List)

	upgrader := websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }
	r.GET("/ws", func(c *gin.Context) {
		channel := c.Query("channel_id")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil { return }
		hub.Join(channel, conn)
		go func() {
			defer func(){ hub.Leave(channel, conn); conn.Close() }()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil { return }
				hub.Broadcast(channel, msg)
			}
		}()
	})

	return r
}
EOF

########################
# main.go（起動）
########################
cat > backend/go/cmd/api/main.go <<'EOF'
package main

import (
	"log"
	"os"
	"time"

	"github.com/example/slackgo/backend/go/internal/config"
	"github.com/example/slackgo/backend/go/internal/db"
	httpapi "github.com/example/slackgo/backend/go/internal/http"
	"github.com/example/slackgo/backend/go/internal/http/handlers"
	"github.com/example/slackgo/backend/go/internal/http/middleware"
	"github.com/example/slackgo/backend/go/internal/jwt"
	"github.com/example/slackgo/backend/go/internal/ws"
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
EOF

########################
# Dockerfile.go
########################
cat > docker/Dockerfile.go <<'EOF'
FROM golang:1.22 AS build
WORKDIR /src
COPY backend/go/go.mod backend/go/go.sum ./
RUN go mod download
COPY backend/go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/api ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /
EXPOSE 8000
COPY --from=build /app/api /api
ENTRYPOINT ["/api"]
EOF

########################
# docker-compose.yml
########################
cat > docker-compose.yml <<'EOF'
version: "3.9"
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: appdb
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d appdb"]
      interval: 3s
      timeout: 3s
      retries: 30
    volumes:
      - pgdata:/var/lib/postgresql/data

  api:
    build:
      context: .
      dockerfile: docker/Dockerfile.go
    environment:
      DB_URL: postgresql://app:app@postgres:5432/appdb
      JWT_SECRET: devjwtsecret_change_me
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8000:8000"

volumes:
  pgdata:
EOF

echo "[*] Files generated."

echo "[*] Building & starting containers..."
docker compose up -d --build

echo "[*] Waiting a few seconds for API to start..."
sleep 3

echo '[*] Health check:'
curl -s http://localhost:8000/health || true
echo
