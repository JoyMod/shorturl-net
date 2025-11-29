package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Options struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// 创建Redis客户端
func NewRedisClient(opts *Options) (*redis.Client, error) {
	if opts.Host == "" {
		return nil, nil
	}

	address := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: opts.Password,
		DB:       opts.DB,
		PoolSize: 20,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %v", err)
	}

	return client, nil
}
