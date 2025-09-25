package wsroute

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"slackgo/internal/model"
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
