package model

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	Id         int            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int            `json:"user_id" gorm:"index;not null"`
	MaskId     string         `json:"mask_id" gorm:"type:varchar(64);index"`
	Content    string         `json:"content" gorm:"type:text;not null"`
	ImageUrl   string         `json:"image_url" gorm:"type:varchar(512)"`
	ParentId   *int           `json:"parent_id" gorm:"index"`
	RootId     *int           `json:"root_id" gorm:"index"`
	LikeCount  int            `json:"like_count" gorm:"default:0"`
	ReplyCount int            `json:"reply_count" gorm:"default:0"`
	Status     int            `json:"status" gorm:"type:int;default:1"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserId;references:Id"`
}

type Like struct {
	Id        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int       `json:"user_id" gorm:"index;not null"`
	PostId    int       `json:"post_id" gorm:"index;not null"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSafe struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
}

type CommunityMask struct {
	Id           string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Name         string `json:"name" gorm:"type:varchar(128);not null"`
	Avatar       string `json:"avatar" gorm:"type:varchar(64)"`
	SystemPrompt string `json:"system_prompt" gorm:"type:text"`
	Model        string `json:"model" gorm:"type:varchar(64);default:'deepseek-chat'"`
	Lang         string `json:"lang" gorm:"type:varchar(8);default:'cn'"`
	IsActive     bool   `json:"is_active" gorm:"default:true"`
}
