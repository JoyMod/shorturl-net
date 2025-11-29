package database

import (
	"fmt"
	"shorturl-platform/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 使用清晰的参数名
func InitMySQL(host string, port int, user, password, dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbName)

	connection, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 自动迁移表
	err = connection.AutoMigrate(
		&model.ShortLink{},
		&model.User{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %v", err)
	}

	return connection, nil
}
