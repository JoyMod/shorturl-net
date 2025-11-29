package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型 (标准定义)
type User struct {
	gorm.Model
	Username     string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email        string `gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string `gorm:"type:varchar(255);not null"` // GORM 默认会映射到 password_hash
	Role         string `gorm:"type:varchar(20);default:'user'"`
	IsActive     bool   `gorm:"default:true"`
	LastLogin    *time.Time
}

// SetPassword 加密并设置密码
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword 校验密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
