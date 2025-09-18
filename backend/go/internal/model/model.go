package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email        *string   `gorm:"uniqueIndex;omitempty"                                  json:"email"`
	PasswordHash string    `gorm:"not null"                                             json:"-"`
	DisplayName  *string   `json:"display_name,omitempty"`
	ExternalID   *string   `json:"external_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Workspace struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `gorm:"not null"                                       json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Channel struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"                       json:"workspace_id"`
	Workspace   Workspace  `gorm:"foreignKey:WorkspaceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Name        string     `gorm:"not null"                                                                            json:"name"`
	IsPrivate   bool       `json:"is_private"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"                                   json:"created_by,omitempty"`
	Creator     *User      `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Message struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"                                                json:"id"`
	WorkspaceID  uuid.UUID  `gorm:"type:uuid;not null;index"                                                                      json:"workspace_id"`
	Workspace    Workspace  `gorm:"foreignKey:WorkspaceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"             json:"-"`
	ChannelID    uuid.UUID  `gorm:"type:uuid;not null;index"                                                                      json:"channel_id"`
	Channel      Channel    `gorm:"foreignKey:ChannelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"               json:"-"`
	UserID       *uuid.UUID `gorm:"type:uuid"                                                                                     json:"user_id,omitempty"`
	User         *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"                 json:"-"`
	Text         *string    `json:"text,omitempty"`
	ParentID     *uuid.UUID `gorm:"type:uuid;index"                                                                               json:"parent_id,omitempty"`
	ThreadRootID *uuid.UUID `gorm:"type:uuid;index"                                                                               json:"thread_root_id,omitempty"`
	Parent       *Message   `gorm:"foreignKey:ParentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"               json:"-"`
	ThreadRoot   *Message   `gorm:"foreignKey:ThreadRootID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"           json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
}

type WorkspaceMember struct {
	UserID      uuid.UUID `gorm:"type:uuid;not null;index:idx_ws_member,unique" json:"user_id"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_ws_member,unique" json:"workspace_id"`
	Role        string    `gorm:"not null;default:member"                       json:"role"`
	CreatedAt   time.Time `json:"created_at"`

	User      User      `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"          json:"-"`
	Workspace Workspace `gorm:"constraint:OnDelete:CASCADE;foreignKey:WorkspaceID;references:ID"     json:"-"`
}

type ChannelMember struct {
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_ch_member,unique" json:"user_id"`
	ChannelID uuid.UUID `gorm:"type:uuid;not null;index:idx_ch_member,unique" json:"channel_id"`
	Role      string    `gorm:"not null;default:member"                       json:"role"`
	CreatedAt time.Time `json:"created_at"`

	User    User    `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"      json:"-"`
	Channel Channel `gorm:"constraint:OnDelete:CASCADE;foreignKey:ChannelID;references:ID"   json:"-"`
}

func EnableExtensions(db *gorm.DB) error {
	return db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error
}
