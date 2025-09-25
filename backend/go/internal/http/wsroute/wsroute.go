package wsroute

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"slackgo/internal/auth"
	"slackgo/internal/ws"
)

type Deps struct {
	DB            *gorm.DB
	Hub           *ws.Hub
	Verifier      *auth.Verifier
	AllowedOrigin string
}

func Register(r *gin.Engine, d Deps) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == d.AllowedOrigin
		},
		Subprotocols: []string{"bearer"},
	}

	r.GET("/ws", func(c *gin.Context) {
		channel := c.Query("channel_id")
		if channel == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// Sec-WebSocket-Protocol: "bearer, <JWT>" からトークンを抜く
		raw := c.Request.Header.Get("Sec-WebSocket-Protocol")
		if raw == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		var token string
		parts := strings.Split(raw, ",")
		for i := range parts {
			p := strings.TrimSpace(parts[i])
			if strings.EqualFold(p, "bearer") || p == "" {
				continue
			}
			token = p
			break
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// JWT 検証 → ユーザー確定（JIT作成）
		claims, err := d.Verifier.Verify(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		uid, err := ensureUserFromSub(d.DB, claims.Sub, claims.Email, claims.Name)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// 接続権限（read可否）確認
		ok, err := canReadChannel(d.DB, uid, channel)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Upgrade → Hubへ参加
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		d.Hub.Join(channel, conn)

		// 上りは受け捨て（いまはサーバからの配信専用）
		go func() {
			defer func() { d.Hub.Leave(channel, conn); conn.Close() }()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	})
}
