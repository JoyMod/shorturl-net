package model

import (
	"time"
)

// ShortLink 短链接模型
type ShortLink struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	ShortCode   string    `gorm:"size:10;uniqueIndex;not null" json:"short_code"`
	OriginalURL string    `gorm:"type:text;not null" json:"original_url"`
	ClickCount  int64     `gorm:"default:0" json:"click_count"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ShortLink) TableName() string {
	return "short_links"
}
