package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/ws"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	auth *handlers.AuthHandler,
	msg *handlers.MessagesHandler,
	ch *handlers.ChannelsHandler,
	wsH *handlers.WorkspacesHandler,
	jwtMw gin.HandlerFunc,
	hub *ws.Hub,
	db *gorm.DB,
) *gin.Engine {
	r := gin.Default()

	// ★ CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", handlers.Health)

	// 認証
	r.POST("/auth/signup", auth.SignUp)
	r.POST("/auth/login", auth.Login)
	r.GET("/auth/me", jwtMw, auth.Me)

	api := r.Group("/")
	api.Use(jwtMw)

	// Workspaces
	api.POST("/workspaces", wsH.Create) // 作成者=owner
	api.GET("/workspaces", wsH.ListMine)
	api.GET("/workspaces/:ws_id/members", middleware.RequireWorkspaceMember(db), wsH.ListMembers)

	// Channels under workspace
	wsGroup := api.Group("/workspaces/:ws_id")
	wsGroup.Use(middleware.RequireWorkspaceMember(db))
	// 作成はowner限定にしたい場合は RequireWorkspaceOwner にする
	wsGroup.POST("/channels", middleware.RequireWorkspaceOwner(db), ch.Create)
	wsGroup.GET("/channels", ch.ListByWorkspace)

	// Channel members（owner限定）
	chGroup := api.Group("/channels/:channel_id")
	chGroup.Use(middleware.RequireChannelOwner(db))
	chGroup.POST("/members", ch.AddMember)

	// Messages（メンバーのみ）
	msgs := api.Group("/channels/:channel_id/messages")
	msgs.GET("", middleware.RequireChannelReadable(db), msg.List)
	msgs.POST("", middleware.RequireChannelWritable(db), msg.Create)

	// WebSocket（そのまま）
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

	// Swagger UI
	r.GET("/docs/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/docs/doc.json"), // ← UI が読む JSON の絶対パスを指定
	))

	return r
}
