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
	"slackgo/internal/storage"
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
	s3deps *storage.S3Deps,
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

	r.GET("/auth/me", jwtMw, auth.Me)

	api := r.Group("/")
	api.Use(jwtMw)
	api.POST("/auth/bootstrap", auth.Bootstrap)

	filesH := handlers.NewFilesHandler(db, s3deps)

	// 署名発行（書き込み権限必須）
	api.POST("/workspaces/:ws_id/channels/:channel_id/files/sign-upload",
		middleware.RequireWorkspaceMember(db), filesH.SignUpload)

	// 完了報告
	api.POST("/files/complete", filesH.Complete)

	// ダウンロード用URL
	api.GET("/files/:file_id/url", filesH.GetDownloadURL)

	// Workspaces
	api.POST("/workspaces", wsH.Create) // 作成者=owner
	api.GET("/workspaces", wsH.ListMine)

	// workspaceのメンバーではないと、チャンネルのメンバーは見れない //これ今使っていない？
	api.GET("/workspaces/:ws_id/members", middleware.RequireWorkspaceMember(db), wsH.ListMembers)
	// workspaceに招待できるのは、workspaceのメンバーのみ
	api.POST("/workspaces/:ws_id/members", middleware.RequireWorkspaceMember(db), wsH.AddMember)

	// 追加: 登録ユーザー検索（認証必須でOK）
	api.GET("/users/search", wsH.SearchUsers)

	// Channels under workspace
	wsGroup := api.Group("/workspaces/:ws_id")
	wsGroup.Use(middleware.RequireWorkspaceMember(db))
	// 作成はowner限定にしたい場合は RequireWorkspaceOwner にする
	// workspaceのメンバーではないと、チャンネルを作成できない
	wsGroup.POST("/channels", ch.Create)
	// workspaceのメンバーではないと、チャンネルを観覧できない。ただしch.ListByWorkspaceの実装により、観覧できるチャンネルはpublic or privateでそのチャンネルメンバーの場合
	wsGroup.GET("/channels", ch.ListByWorkspace)
	wsGroup.POST("/channels/:channel_id/join", ch.JoinSelf)
	wsGroup.GET("/channels/:channel_id/membership", ch.IsMember)

	// Channel members
	chGroup := api.Group("/channels/:channel_id")
	chGroup.Use(middleware.RequireChannelMember(db))
	// チャンネルメンバーであればチャンネルへの追加ができる
	chGroup.POST("/members", ch.AddMember)
	chGroup.GET("/members/search", ch.SearchWorkspaceMembers)

	// Messages（メンバーのみ）
	msgs := api.Group("/channels/:channel_id/messages")
	// publicであればworkspace memberならば、privateであればchannel memberならば観覧できる
	msgs.GET("", middleware.RequireChannelReadable(db), msg.List)
	// public, privateともにチャンネルへの書き込みはチャンネルメンバーでなくてはならない
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
