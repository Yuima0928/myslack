package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/ws"
)

func NewRouter(
	auth *handlers.AuthHandler,
	msg *handlers.MessagesHandler,
	ch *handlers.ChannelsHandler,
	jwtMw gin.HandlerFunc,
	hub *ws.Hub,
	db *gorm.DB, // ← 追加
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", handlers.Health)

	// 認証まわり
	r.POST("/auth/signup", auth.SignUp)
	r.POST("/auth/login", auth.Login)
	r.GET("/auth/me", jwtMw, auth.Me)

	// 認証必須グループ
	api := r.Group("/")
	api.Use(jwtMw)

	// メッセージ（チャンネルメンバーのみ）
	msgs := api.Group("/channels/:channel_id/messages")
	msgs.Use(middleware.RequireChannelMember(db))
	msgs.POST("", msg.Create)
	msgs.GET("", msg.List)

	chans := api.Group("/channels")
	chans.Use(middleware.RequireWorkspaceMember(db))
	chans.POST("", ch.Create)

	// WebSocket
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	r.GET("/ws", func(c *gin.Context) {
		channel := c.Query("channel_id")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		hub.Join(channel, conn)
		go func() {
			defer func() { hub.Leave(channel, conn); conn.Close() }()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				hub.Broadcast(channel, msg)
			}
		}()
	})

	return r
}
