package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

// InitLogger 初始化 zap 日志记录器
func InitLogger() {
	// 配置日志写入位置
	writeSyncer := getLogWriter()
	// 配置编码器
	encoder := getEncoder()
	// 设置核心
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	// 创建 Logger
	Logger = zap.New(core, zap.AddCaller())
	Sugar = Logger.Sugar()

	// 将全局的 zap logger 替换为我们配置好的 logger
	zap.ReplaceGlobals(Logger)
}

// getEncoder 设置日志编码格式
func getEncoder() zapcore.Encoder {
	// 使用 zap 提供的默认生产环境编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	// 自定义时间编码器
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 自定义日志级别编码器，使其大写并带颜色
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// 返回一个在控制台和文件都适用的 JSON 编码器
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getLogWriter 指定日志写入位置 (文件和控制台)
func getLogWriter() zapcore.WriteSyncer {
	// 使用 lumberjack 实现日志切割和归档
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./logs/app.log", // 日志文件路径
		MaxSize:    10,               // 每个日志文件的最大尺寸，单位为 MB
		MaxBackups: 5,                // 保留的旧日志文件的最大数量
		MaxAge:     30,               // 保留的旧日志文件的最大天数
		Compress:   false,            // 是否压缩旧日志文件
	}
	// 同时将日志输出到文件和控制台
	return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lumberJackLogger))
}
