package shortcode

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// Charset 包含用于生成短码的所有字符
	Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// CodeLength 是生成的短码的长度
	CodeLength = 7
	// ChannelBufferSize 是短码通道的缓冲区大小
	ChannelBufferSize = 1000
	// MinFillThreshold 是触发补充的最小阈值
	MinFillThreshold = 100
)

// Generator 负责生成和提供唯一的短码
type Generator struct {
	db        *gorm.DB
	codeChan  chan string
	mu        sync.Mutex
	isFilling bool
	stopChan  chan struct{}
	logger    *zap.SugaredLogger
}

// NewGenerator 创建一个新的短码生成器实例
func NewGenerator(db *gorm.DB, logger *zap.SugaredLogger) *Generator {
	return &Generator{
		db:       db,
		codeChan: make(chan string, ChannelBufferSize),
		stopChan: make(chan struct{}),
		logger:   logger.Named("shortcode_generator"),
	}
}

// Start 启动后台短码生成和补充任务
func (g *Generator) Start() {
	g.logger.Info("启动短码生成器...")
	go g.fillChannel() // 初始填充
	go g.monitorAndRefill()
}

// Stop 停止短码生成器
func (g *Generator) Stop() {
	g.logger.Info("正在停止短码生成器...")
	close(g.stopChan)
}

// GetCode 从通道中获取一个唯一的短码
func (g *Generator) GetCode() string {
	return <-g.codeChan
}

// monitorAndRefill 监视通道的填充水平并根据需要进行补充
func (g *Generator) monitorAndRefill() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if len(g.codeChan) < MinFillThreshold {
				g.fillChannel()
			}
		case <-g.stopChan:
			g.logger.Info("已停止监控和补充任务。")
			return
		}
	}
}

// fillChannel 是一个后台 goroutine，用于生成短码并填充通道
func (g *Generator) fillChannel() {
	g.mu.Lock()
	if g.isFilling {
		g.mu.Unlock()
		return
	}
	g.isFilling = true
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		g.isFilling = false
		g.mu.Unlock()
	}()

	g.logger.Infof("通道中剩余 %d 个短码，开始补充...", len(g.codeChan))
	for len(g.codeChan) < ChannelBufferSize {
		select {
		case <-g.stopChan:
			g.logger.Info("填充任务已中断。")
			return
		default:
			code, err := g.generateUniqueCode()
			if err != nil {
				g.logger.Errorf("生成唯一短码时出错: %v", err)
				time.Sleep(100 * time.Millisecond) // 避免在错误情况下快速循环
				continue
			}
			if code != "" {
				g.codeChan <- code
			}
		}
	}
	g.logger.Infof("短码通道已填满，现有 %d 个。", len(g.codeChan))
}

// generateUniqueCode 生成一个在数据库中唯一的短码
func (g *Generator) generateUniqueCode() (string, error) {
	for i := 0; i < 10; i++ { // 尝试最多10次
		code, err := g.generateRandomString(CodeLength)
		if err != nil {
			return "", err
		}
		if !g.isCodeExist(code) {
			return code, nil
		}
	}
	g.logger.Warn("已尝试10次生成短码，但均存在冲突。")
	return "", nil // 返回空字符串表示需要重试
}

// generateRandomString 使用加密安全的随机数生成器生成一个给定长度的字符串
func (g *Generator) generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(Charset))))
		if err != nil {
			return "", err
		}
		b[i] = Charset[num.Int64()]
	}
	return string(b), nil
}

// isCodeExist 检查给定的短码是否已在数据库中存在
func (g *Generator) isCodeExist(code string) bool {
	var count int64
	// 使用 unscoped 可以在包含软删除的表上进行查询
	if err := g.db.Unscoped().Table("short_links").Where("short_code = ?", code).Count(&count).Error; err != nil {
		g.logger.Errorf("查询数据库时出错: %v", err)
		// 在不确定的情况下，保守地认为它存在以避免冲突
		return true
	}
	return count > 0
}
