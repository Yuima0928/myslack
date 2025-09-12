package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	DisplayName  *string
	CreatedAt    time.Time
}

type Workspace struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time
}

type Channel struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index"`
	// FK: channels.workspace_id -> workspaces.id
	Workspace Workspace `gorm:"foreignKey:WorkspaceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name      string    `gorm:"not null"`
	IsPrivate bool
	CreatedBy *uuid.UUID `gorm:"type:uuid"`
	// FK: channels.created_by -> users.id（作成者削除でNULL）
	Creator   *User `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt time.Time
}

type Message struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"`
	Workspace   Workspace  `gorm:"foreignKey:WorkspaceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ChannelID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Channel     Channel    `gorm:"foreignKey:ChannelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	User        *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Text        *string
	CreatedAt   time.Time
	EditedAt    *time.Time
}

type WorkspaceMember struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;primaryKey;index"`
	Role        string    `gorm:"type:text;not null;default:'member'"`

	// FK
	User      User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Workspace Workspace `gorm:"foreignKey:WorkspaceID;references:ID;constraint:OnDelete:CASCADE"`
}

type ChannelMember struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:uuid;primaryKey;index"`
	Role      string    `gorm:"type:text;not null;default:'member'"`

	// FK
	User    User    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Channel Channel `gorm:"foreignKey:ChannelID;references:ID;constraint:OnDelete:CASCADE"`
}

func EnableExtensions(db *gorm.DB) error {
	return db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error
}
