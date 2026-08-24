package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:80;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	DisplayName  string    `gorm:"size:120;not null" json:"display_name"`
	Role         string    `gorm:"size:40;not null;index" json:"role"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ActorID      uint      `gorm:"not null;index" json:"actor_id"`
	ActorName    string    `gorm:"size:120;not null" json:"actor_name"`
	ActorRole    string    `gorm:"size:40;not null" json:"actor_role"`
	RequestID    string    `gorm:"size:80;not null;index" json:"request_id"`
	EntityType   string    `gorm:"size:60;not null;index" json:"entity_type"`
	EntityID     uint      `gorm:"not null;index" json:"entity_id"`
	Action       string    `gorm:"size:80;not null;index" json:"action"`
	BeforeJSON   string    `gorm:"type:text;not null" json:"before_json"`
	AfterJSON    string    `gorm:"type:text;not null" json:"after_json"`
	MetadataJSON string    `gorm:"type:text;not null" json:"metadata_json"`
	CreatedAt    time.Time `gorm:"not null;index" json:"created_at"`
}
