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
	Name        string    `gorm:"not null"`
	IsPrivate   bool
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

type Message struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"`
	ChannelID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	Text        *string
	CreatedAt   time.Time
	EditedAt    *time.Time
}

func EnableExtensions(db *gorm.DB) error {
	return db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error
}
