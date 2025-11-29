package model

import (
	"time"
)

type ClickRecord struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	ShortLinkID uint      `gorm:"not null;index" json:"short_link_id"`
	IPAddress   string    `gorm:"size:45" json:"ip_address"`
	UserAgent   string    `gorm:"type:text" json:"user_agent"`
	Referer     string    `gorm:"type:text" json:"referer"`
	Country     string    `gorm:"size:100" json:"country"`
	City        string    `gorm:"size:100" json:"city"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ClickRecord) TableName() string {
	return "click_records"
}
