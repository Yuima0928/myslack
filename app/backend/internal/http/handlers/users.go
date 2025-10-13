package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"slackgo/internal/model"
	"slackgo/internal/storage"
)

type UsersHandler struct {
	db     *gorm.DB
	s3deps *storage.S3Deps
}

func NewUsersHandler(db *gorm.DB, s3deps *storage.S3Deps) *UsersHandler {
	return &UsersHandler{db: db, s3deps: s3deps}
}

type MeOut struct {
	ID           uuid.UUID  `json:"id"`
	Email        *string    `json:"email,omitempty"`
	DisplayName  *string    `json:"display_name,omitempty"`
	AvatarFileID *uuid.UUID `json:"avatar_file_id,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"` // 署名URL(オプション)
}

// GET /users/me
func (h *UsersHandler) GetMe(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	var u model.User
	if err := h.db.First(&u, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "user not found"})
		return
	}

	var avatarURL *string
	if u.AvatarFileID != nil {
		// files から storage_key / filename / content_type を引く
		var f struct {
			StorageKey  string
			Filename    string
			ContentType *string
		}
		if err := h.db.
			Table("files").
			Select("storage_key, filename, content_type").
			Where("id = ?", *u.AvatarFileID).
			Take(&f).Error; err == nil && f.StorageKey != "" {
			ct := ""
			if f.ContentType != nil {
				ct = *f.ContentType
			}
			if link, err := h.s3deps.SignGetURL(c, f.StorageKey, f.Filename, ct, "inline"); err == nil {
				avatarURL = &link
			}
		}
	}

	out := MeOut{
		ID:           u.ID,
		Email:        u.Email,
		DisplayName:  u.DisplayName,
		AvatarFileID: u.AvatarFileID,
		AvatarURL:    avatarURL,
	}
	c.JSON(http.StatusOK, out)
}

type UpdateMeIn struct {
	DisplayName  *string `json:"display_name"`           // null なら変更なし、"" にしたいなら空文字を送る
	AvatarFileID *string `json:"avatar_file_id_or_null"` // null: 変更なし, ""(空文字): 画像解除, UUID文字列: 設定
}

// PUT /users/me
func (h *UsersHandler) UpdateMe(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	var in UpdateMeIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}

	updates := map[string]any{}

	if in.DisplayName != nil {
		updates["display_name"] = in.DisplayName
	}

	if in.AvatarFileID != nil {
		if *in.AvatarFileID == "" {
			// 空文字なら解除
			updates["avatar_file_id"] = nil
		} else {
			// UUID妥当性だけチェック（存在チェックは任意）
			if fid, err := uuid.Parse(*in.AvatarFileID); err == nil {
				updates["avatar_file_id"] = &fid
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid avatar_file_id"})
				return
			}
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "no changes"})
		return
	}

	if err := h.db.Model(&model.User{}).
		Where("id = ?", uid).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "update failed"})
		return
	}

	c.Status(http.StatusNoContent)
}
