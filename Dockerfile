# ---- 1. 构建阶段 ----
# 使用官方的 Go 镜像作为构建环境
FROM golang:1.18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件，并下载依赖
# 这一步可以利用 Docker 的层缓存机制
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码到工作目录
COPY . .

# 编译应用程序
# -ldflags="-w -s" 用于减小二进制文件体积
# CGO_ENABLED=0 用于构建静态链接的二进制文件，使其不依赖系统的 C 库
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/shorturl-server ./cmd/server/main.go

# ---- 2. 运行阶段 ----
# 使用一个极简的基础镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/shorturl-server /app/shorturl-server

# 复制程序运行时需要的配置文件和静态资源
COPY configs/ /app/configs/
COPY web/ /app/web/

# 暴露服务端口
EXPOSE 8080

# 设置容器启动时执行的命令
CMD ["/app/shorturl-server"]
