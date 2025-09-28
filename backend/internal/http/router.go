package httpapi

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"slackgo/internal/auth"
	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/http/wsroute"
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
	verifier *auth.Verifier,
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

	usersH := handlers.NewUsersHandler(db, s3deps)
	api.GET("/users/me", usersH.GetMe)
	api.PUT("/users/me", usersH.UpdateMe)

	filesH := handlers.NewFilesHandler(db, s3deps)

	/// 署名発行
	api.POST("/workspaces/:ws_id/channels/:channel_id/files/sign-upload",
		middleware.RequireWorkspaceMember(db), filesH.SignUploadMessage)

	// アバター用署名発行
	api.POST("/users/me/avatar/sign-upload", filesH.SignUploadAvatar)

	// 完了報告
	api.POST("/files/complete", filesH.Complete)

	// ダウンロードURL
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

	api.GET("/channels/:channel_id/membership", middleware.RequireChannelReadable(db), ch.IsMember)

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

	wsroute.Register(r, wsroute.Deps{
		DB:            db,
		Hub:           hub,
		Verifier:      verifier,
		AllowedOrigin: "http://localhost:5173", // 本番は適切に絞る
	})

	// Swagger UI
	r.GET("/docs/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/docs/doc.json"), // ← UI が読む JSON の絶対パスを指定
	))

	return r
}
