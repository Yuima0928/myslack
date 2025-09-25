package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"slackgo/internal/auth"
	"slackgo/internal/http/handlers"
	"slackgo/internal/http/middleware"
	"slackgo/internal/model"
	"slackgo/internal/storage"
	"slackgo/internal/ws"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func ensureUserFromSub(db *gorm.DB, sub, email, name string) (uuid.UUID, error) {
	var u model.User
	if err := db.Where("external_id = ?", sub).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			u = model.User{
				ExternalID:  strPtrOrNil(sub),
				Email:       strPtrOrNil(email),
				DisplayName: strPtrOrNil(name),
			}
			if err := db.Create(&u).Error; err != nil {
				return uuid.Nil, err
			}
		} else {
			return uuid.Nil, err
		}
	}
	return u.ID, nil
}

func canReadChannel(db *gorm.DB, userID uuid.UUID, channelID string) (bool, error) {
	var ch struct {
		ID          uuid.UUID
		WorkspaceID uuid.UUID
		IsPrivate   bool
	}
	if err := db.
		Table("channels").
		Select("id, workspace_id, is_private").
		Where("id = ?", channelID).
		Limit(1).
		Scan(&ch).Error; err != nil {
		return false, err
	}
	if ch.ID == uuid.Nil {
		return false, nil
	}
	if ch.IsPrivate {
		var n int64
		if err := db.Table("channel_members").
			Where("channel_id = ? AND user_id = ?", ch.ID, userID).
			Count(&n).Error; err != nil {
			return false, err
		}
		return n > 0, nil
	}
	var n int64
	if err := db.Table("workspace_members").
		Where("workspace_id = ? AND user_id = ?", ch.WorkspaceID, userID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

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

	// WebSocket（そのまま）
	upgrader := websocket.Upgrader{
		// 本番は許可オリジンを絞ること
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "http://localhost:5173"
		},
		// こちらは「サーバが採用してよいプロトコル」のリスト
		// bearer を明示しておく（トークン自体は採用しない）
		Subprotocols: []string{"bearer"},
	}

	// httpapi.NewRouter 内の /ws ハンドラを差し替え
	r.GET("/ws", func(c *gin.Context) {
		channel := c.Query("channel_id")
		if channel == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		raw := c.Request.Header.Get("Sec-WebSocket-Protocol")
		if raw == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		parts := strings.Split(raw, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		var token string
		for _, p := range parts {
			if strings.EqualFold(p, "bearer") {
				continue
			}
			if p != "" {
				token = p
				break
			}
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		uid, err := ensureUserFromSub(db, claims.Sub, claims.Email, claims.Name)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// 接続時は「読めるか」だけチェック（private: channel member / public: WS member）
		ok, err := canReadChannel(db, uid, channel)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		hub.Join(channel, conn)
		go func() {
			defer func() { hub.Leave(channel, conn); conn.Close() }()
			// 上りは受け取っても何もしない（破棄）
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
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
