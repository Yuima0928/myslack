package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"slackgo/internal/http/handlers"
	"slackgo/internal/ws"
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
