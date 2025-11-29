# 短链接平台 (shorturl-net)

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

这是一个使用 Go 语言构建的高性能、高可用的短链接服务平台。它提供了完整的 API，支持短链接的创建、重定向、管理，并为高并发场景进行了特别优化。

## ✨ 功能特性

- **高性能**: 采用后台预生成短码和内存通道缓冲的策略，创建短链接接近纯内存操作，轻松应对高并发请求。
- **缓存优先**: 集成 Redis 缓存，优先从缓存中读取链接，极大加快重定向速度，降低数据库压力。
- **RESTful API**: 提供清晰、标准的 API，支持用户认证和管理功能。
- **交互式文档**: 内置 Swagger UI，API 文档清晰明了，支持在线调试。
- **可配置**: 核心参数（数据库、Redis、JWT、服务器端口等）均通过配置文件管理。
- **容器化支持**: 提供 Dockerfile，支持一键打包和部署。
- **独立的测试环境**: 提供基于内存数据库的集成测试，无需复杂配置即可验证核心功能。
- **结构清晰**: 遵循标准的 Go 项目布局，易于理解和二次开发。

## 🛠️ 技术栈

- **后端框架**: [Gin](https://github.com/gin-gonic/gin)
- **数据库**: [GORM](https://gorm.io/) (支持 MySQL, 测试中使用 SQLite)
- **缓存**: [go-redis](https://github.com/redis/go-redis)
- **日记**: [Zap](https://github.com/uber-go/zap)
- **配置**: [Viper](https://github.com/spf13/viper) (通过 `config.yaml` 实现)
- **API 文档**: [gin-swagger](https://github.com/swaggo/gin-swagger)

## 🚀 快速开始

### 1. 环境准备

- [Go](https://golang.org/dl/) (版本 >= 1.18)
- [MySQL](https://www.mysql.com/) (或使用 Docker)
- [Redis](https://redis.io/) (或使用 Docker)
- [Docker](https://www.docker.com/) (可选, 用于快速部署)

### 2. 克隆项目

```bash
git clone https://github.com/your-username/shorturl-net.git
cd shorturl-net
```

### 3. 配置

项目的所有配置都在 `configs/config.yaml` 文件中。请根据您的本地环境修改此文件。

```yaml
# configs/config.yaml
app:
  mode: "debug" # 开发模式, 生产环境请改为 "production"

server:
  port: 8080
  read_timeout: 10
  write_timeout: 10

database:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_mysql_password"
  name: "shorturl_db"

cache:
  host: "127.0.0.1"
  port: 6379
  password: ""
  db: 0

# ... 其他配置
```

**注意**: 请确保在 MySQL 中已创建名为 `shorturl_db` 的数据库。

### 4. 安装依赖并运行

```bash
# 安装 Go 依赖
go mod tidy

# 运行服务
go run ./cmd/server/main.go
```

服务启动后，您会看到类似以下的输出：

```
INFO    🚀 服务启动成功, 访问 http://localhost:8080
INFO    📚 Swagger 文档地址: http://localhost:8080/swagger/index.html
```

## 📚 API 文档

项目启动后，直接访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) 即可查看交互式的 API 文档。

## 🐳 使用 Docker 运行

我们提供了 `Dockerfile`，您可以轻松地将项目打包成 Docker 镜像。

1.  **构建镜像**:

    ```bash
    docker build -t shorturl-net:latest .
    ```

2.  **运行容器**:

    确保您的 `configs/config.yaml` 文件中的数据库和 Redis 地址对于 Docker 容器是可访问的 (例如，使用 Docker 内部网络地址或公网 IP，而不是 `localhost`)。

    ```bash
    docker run -p 8080:8080 -d --name shorturl-app shorturl-net:latest
    ```

## ✅ 测试

项目提供了独立的集成测试，无需连接到真实的 MySQL 或 Redis。

```bash
# 运行所有测试
go test -v ./...
```

更多测试细节，请参考 `TESTING.md` 文件。

## 🤝 如何贡献

我们非常欢迎社区的贡献！如果您想为这个项目做出贡献，请遵循以下步骤：

1.  **Fork** 本项目。
2.  创建一个新的分支 (`git checkout -b feature/YourFeature`)。
3.  提交您的代码 (`git commit -m 'Add some feature'`)。
4.  将您的分支推送到远程 (`git push origin feature/YourFeature`)。
5.  创建一个 **Pull Request**。

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。
