// internal/http/handlers/files.go
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
	"slackgo/internal/storage" // ★ 追加
)

type FilesHandler struct {
	db *gorm.DB
	s3 *storage.S3Deps
}

func NewFilesHandler(db *gorm.DB, s3deps *storage.S3Deps) *FilesHandler {
	return &FilesHandler{db: db, s3: s3deps}
}

func (h *FilesHandler) buildKey(wsID, chID, fileID, filename string) string {
	safe := regexp.MustCompile(`[^\w.\-]`).ReplaceAllString(filename, "_")
	return path.Join(h.s3.Prefix, "ws", wsID, "ch", chID, fmt.Sprintf("%s_%s", fileID, safe))
}

// POST /workspaces/:ws_id/channels/:channel_id/files/sign-upload
func (h *FilesHandler) SignUpload(c *gin.Context) {
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
	if err := c.BindJSON(&body); err != nil || body.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}

	fileID := uuid.New().String()
	key := h.buildKey(wsID, chID, fileID, body.Filename)

	presigned, err := h.s3.Presign.PresignPutObject(
		c, // context
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

// POST /files/complete
func (h *FilesHandler) Complete(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var body struct {
		StorageKey  string  `json:"storage_key"`
		ETag        string  `json:"etag"`
		SHA256Hex   *string `json:"sha256_hex"`
		Filename    string  `json:"filename"`
		ContentType string  `json:"content_type"`
		SizeBytes   int64   `json:"size_bytes"`
		WorkspaceID string  `json:"workspace_id"`
		ChannelID   string  `json:"channel_id"`
	}
	if err := c.BindJSON(&body); err != nil || body.StorageKey == "" || body.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "bad body"})
		return
	}

	rec := model.File{
		ID:          uuid.New(),
		WorkspaceID: uuid.MustParse(body.WorkspaceID),
		ChannelID:   uuid.MustParse(body.ChannelID),
		UploaderID:  uuid.MustParse(uid),
		Filename:    body.Filename,
		ContentType: strPtr(body.ContentType),
		SizeBytes:   int64Ptr(body.SizeBytes),
		ETag:        strPtr(body.ETag),
		SHA256Hex:   body.SHA256Hex,
		StorageKey:  body.StorageKey,
		IsImage:     strings.HasPrefix(strings.ToLower(body.ContentType), "image/"),
		CreatedAt:   time.Now(),
	}
	if err := h.db.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "db insert failed"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *FilesHandler) canReadFile(uid uuid.UUID, f *model.File) (bool, error) {
	// チャンネル情報を取得
	var ch model.Channel
	if err := h.db.Select("id, workspace_id, is_private").
		First(&ch, "id = ?", f.ChannelID).Error; err != nil {
		return false, err
	}

	if ch.IsPrivate {
		// プライベートはチャンネルメンバーのみ
		var n int64
		if err := h.db.Model(&model.ChannelMember{}).
			Where("channel_id = ? AND user_id = ?", ch.ID, uid).
			Count(&n).Error; err != nil {
			return false, err
		}
		return n > 0, nil
	}

	// パブリックはワークスペースメンバーならOK
	var n int64
	if err := h.db.Model(&model.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", ch.WorkspaceID, uid).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// GET /files/:file_id/url?disposition=inline|attachment
func (h *FilesHandler) GetDownloadURL(c *gin.Context) {
	uidStr := c.GetString("user_id")
	if uidStr == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(uidStr)
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

	// ★ 権限チェックをここで
	ok, err := h.canReadFile(uid, &f)
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
