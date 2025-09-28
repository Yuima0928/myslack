package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"slackgo/internal/model"
	"slackgo/internal/storage"
)

type FilesHandler struct {
	db *gorm.DB
	s3 *storage.S3Deps
}

func NewFilesHandler(db *gorm.DB, s3deps *storage.S3Deps) *FilesHandler {
	return &FilesHandler{db: db, s3: s3deps}
}

// ========= 署名URL用のキー生成 =========

func (h *FilesHandler) safeName(filename string) string {
	return regexp.MustCompile(`[^\w.\-]`).ReplaceAllString(filename, "_")
}

// メッセージ添付（WS/CH配下）
func (h *FilesHandler) keyForMessage(wsID, chID, fileID, filename string) string {
	return path.Join(h.s3.Prefix, "ws", wsID, "ch", chID, fmt.Sprintf("%s_%s", fileID, h.safeName(filename)))
}

// アバター（user配下）
func (h *FilesHandler) keyForAvatar(userID, fileID, filename string) string {
	return path.Join(h.s3.Prefix, "avatars", userID, fmt.Sprintf("%s_%s", fileID, h.safeName(filename)))
}

// ========= サイン発行（メッセージ添付） =========
// POST /workspaces/:ws_id/channels/:channel_id/files/sign-upload
func (h *FilesHandler) SignUploadMessage(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	wsID := c.Param("ws_id")
	chID := c.Param("channel_id")

	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	if err := c.BindJSON(&body); err != nil || strings.TrimSpace(body.Filename) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}

	fileID := uuid.New().String()
	key := h.keyForMessage(wsID, chID, fileID, body.Filename)

	presigned, err := h.s3.Presign.PresignPutObject(
		c,
		&s3.PutObjectInput{
			Bucket:      aws.String(h.s3.Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(body.ContentType),
		},
		func(o *s3.PresignOptions) { o.Expires = h.s3.Expire },
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "presign failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url":  presigned.URL,
		"storage_key": key,
		"file_id":     fileID,
	})
}

// ========= サイン発行（アバター） =========
// POST /users/me/avatar/sign-upload
func (h *FilesHandler) SignUploadAvatar(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	if err := c.BindJSON(&body); err != nil || strings.TrimSpace(body.Filename) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}

	fileID := uuid.New().String()
	key := h.keyForAvatar(uid, fileID, body.Filename)

	presigned, err := h.s3.Presign.PresignPutObject(
		c,
		&s3.PutObjectInput{
			Bucket:      aws.String(h.s3.Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(body.ContentType),
		},
		func(o *s3.PresignOptions) { o.Expires = h.s3.Expire },
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "presign failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url":  presigned.URL,
		"storage_key": key,
		"file_id":     fileID,
	})
}

// ========= アップロード完了（どちらの用途も共通で登録） =========
// POST /files/complete
// 目的に応じて body.purpose を "message_attachment" or "avatar" で受ける
func (h *FilesHandler) Complete(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	uploaderID := uuid.MustParse(uid)

	var body struct {
		// 共通
		Purpose     string  `json:"purpose"` // "message_attachment" | "avatar"
		StorageKey  string  `json:"storage_key"`
		ETag        string  `json:"etag"`
		SHA256Hex   *string `json:"sha256_hex"`
		Filename    string  `json:"filename"`
		ContentType string  `json:"content_type"`
		SizeBytes   int64   `json:"size_bytes"`

		// message_attachment 用
		WorkspaceID *string `json:"workspace_id,omitempty"`
		ChannelID   *string `json:"channel_id,omitempty"`

		// avatar 用（省略可。省略時は自分）
		OwnerUserID *string `json:"owner_user_id,omitempty"`
	}
	if err := c.BindJSON(&body); err != nil ||
		strings.TrimSpace(body.StorageKey) == "" ||
		strings.TrimSpace(body.Filename) == "" ||
		strings.TrimSpace(body.Purpose) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}

	now := time.Now()
	rec := model.File{
		ID:          uuid.New(),
		Purpose:     body.Purpose,
		UploaderID:  uploaderID,
		Filename:    body.Filename,
		ContentType: strPtr(body.ContentType),
		SizeBytes:   int64Ptr(body.SizeBytes),
		ETag:        strPtr(body.ETag),
		SHA256Hex:   body.SHA256Hex,
		StorageKey:  body.StorageKey,
		IsImage:     strings.HasPrefix(strings.ToLower(body.ContentType), "image/"),
		CreatedAt:   now,
	}

	switch body.Purpose {
	case "message_attachment":
		// 必須: workspace_id, channel_id
		if body.WorkspaceID == nil || body.ChannelID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "workspace_id and channel_id required for message_attachment"})
			return
		}
		wsID, err1 := uuid.Parse(*body.WorkspaceID)
		chID, err2 := uuid.Parse(*body.ChannelID)
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid workspace_id or channel_id"})
			return
		}
		rec.WorkspaceID = &wsID
		rec.ChannelID = &chID

	case "avatar":
		// 省略時は自分
		var owner uuid.UUID
		if body.OwnerUserID != nil && strings.TrimSpace(*body.OwnerUserID) != "" {
			oid, err := uuid.Parse(*body.OwnerUserID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid owner_user_id"})
				return
			}
			owner = oid
		} else {
			owner = uploaderID
		}
		rec.OwnerUserID = &owner

	default:
		c.JSON(http.StatusBadRequest, gin.H{"detail": "unknown purpose"})
		return
	}

	if err := h.db.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "db insert failed"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// ========= 権限チェック =========

func (h *FilesHandler) canReadFile(requester uuid.UUID, f *model.File) (bool, error) {
	switch f.Purpose {
	case "avatar":
		// ここは「アプリ内の認証済みユーザーなら誰でもOK」にする。
		// もっと厳密にしたいなら、ワークスペース所属関係などをここで見てもよい。
		return true, nil

	case "message_attachment":
		// チャンネル公開範囲に従う
		if f.ChannelID == nil {
			return false, nil
		}
		var ch model.Channel
		if err := h.db.Select("id, workspace_id, is_private").
			First(&ch, "id = ?", *f.ChannelID).Error; err != nil {
			return false, err
		}

		if ch.IsPrivate {
			var n int64
			if err := h.db.Model(&model.ChannelMember{}).
				Where("channel_id = ? AND user_id = ?", ch.ID, requester).
				Count(&n).Error; err != nil {
				return false, err
			}
			return n > 0, nil
		}
		var n int64
		if err := h.db.Model(&model.WorkspaceMember{}).
			Where("workspace_id = ? AND user_id = ?", ch.WorkspaceID, requester).
			Count(&n).Error; err != nil {
			return false, err
		}
		return n > 0, nil
	default:
		return false, nil
	}
}

// ========= ダウンロードURL（署名GET） =========
// GET /files/:file_id/url?disposition=inline|attachment
func (h *FilesHandler) GetDownloadURL(c *gin.Context) {
	uidStr := c.GetString("user_id")
	if uidStr == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	requester, err := uuid.Parse(uidStr)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	fileID := c.Param("file_id")
	var f model.File
	if err := h.db.First(&f, "id = ?", fileID).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	ok, err := h.canReadFile(requester, &f)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	disp := c.DefaultQuery("disposition", "attachment")
	ct := "application/octet-stream"
	if f.ContentType != nil && *f.ContentType != "" {
		ct = *f.ContentType
	}

	cd := fmt.Sprintf(`%s; filename="%s"`, disp, url.PathEscape(f.Filename))
	presigned, err := h.s3.Presign.PresignGetObject(
		c,
		&s3.GetObjectInput{
			Bucket:                     aws.String(h.s3.Bucket),
			Key:                        aws.String(f.StorageKey),
			ResponseContentDisposition: aws.String(cd),
			ResponseContentType:        aws.String(ct),
		},
		func(o *s3.PresignOptions) { o.Expires = h.s3.Expire },
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "presign failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        presigned.URL,
		"expires_at": time.Now().Add(h.s3.Expire),
	})
}

func strPtr(s string) *string { return &s }
func int64Ptr(n int64) *int64 { return &n }
