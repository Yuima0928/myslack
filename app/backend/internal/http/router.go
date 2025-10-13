package httpapi

import (
	"os"
	"strings"
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

func readOriginsEnv(key, def string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = def
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func NewRouter(
	authH *handlers.AuthHandler,
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

	// ---- CORS 設定（ENVから）----
	allowOrigins := readOriginsEnv("CORS_ALLOW_ORIGINS", "http://localhost:5173")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", handlers.Health)

	r.GET("/auth/me", jwtMw, authH.Me)

	api := r.Group("/")
	api.Use(jwtMw)
	api.POST("/auth/bootstrap", authH.Bootstrap)

	usersH := handlers.NewUsersHandler(db, s3deps)
	api.GET("/users/me", usersH.GetMe)
	api.PUT("/users/me", usersH.UpdateMe)

	filesH := handlers.NewFilesHandler(db, s3deps)
	api.POST("/workspaces/:ws_id/channels/:channel_id/files/sign-upload",
		middleware.RequireWorkspaceMember(db), filesH.SignUploadMessage)
	api.POST("/users/me/avatar/sign-upload", filesH.SignUploadAvatar)
	api.POST("/files/complete", filesH.Complete)
	api.GET("/files/:file_id/url", filesH.GetDownloadURL)

	api.POST("/workspaces", wsH.Create)
	api.GET("/workspaces", wsH.ListMine)

	api.GET("/workspaces/:ws_id/members", middleware.RequireWorkspaceMember(db), wsH.ListMembers)
	api.POST("/workspaces/:ws_id/members", middleware.RequireWorkspaceMember(db), wsH.AddMember)

	api.GET("/users/search", wsH.SearchUsers)

	wsGroup := api.Group("/workspaces/:ws_id")
	wsGroup.Use(middleware.RequireWorkspaceMember(db))
	wsGroup.POST("/channels", ch.Create)
	wsGroup.GET("/channels", ch.ListByWorkspace)
	wsGroup.POST("/channels/:channel_id/join", ch.JoinSelf)

	api.GET("/channels/:channel_id/membership", middleware.RequireChannelReadable(db), ch.IsMember)

	chGroup := api.Group("/channels/:channel_id")
	chGroup.Use(middleware.RequireChannelMember(db))
	chGroup.POST("/members", ch.AddMember)
	chGroup.GET("/members/search", ch.SearchWorkspaceMembers)

	// Messages（メンバーのみ）
	msgs := api.Group("/channels/:channel_id/messages")
	// publicであればworkspace memberならば、privateであればchannel memberならば観覧できる
	msgs.GET("", middleware.RequireChannelReadable(db), msg.List)
	// public, privateともにチャンネルへの書き込みはチャンネルメンバーでなくてはならない
	msgs.POST("", middleware.RequireChannelWritable(db), msg.Create)

	// ---- WS AllowedOrigin も ENV から ----
	wsAllowed := readOriginsEnv("WS_ALLOWED_ORIGIN", "http://localhost:5173")
	firstWSOrigin := "http://localhost:5173"
	if len(wsAllowed) > 0 {
		firstWSOrigin = wsAllowed[0]
	}
	wsroute.Register(r, wsroute.Deps{
		DB:            db,
		Hub:           hub,
		Verifier:      verifier,
		AllowedOrigin: firstWSOrigin,
	})

	// Swagger
	r.GET("/docs/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/docs/doc.json"),
	))

	return r
}
