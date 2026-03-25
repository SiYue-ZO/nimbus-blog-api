# Nimbus Blog API

Nimbus Blog 的后端服务，采用 Clean Architecture 分层设计，提供后台管理与公开接口两套 API。

## 概览

- 模块划分：Admin（后台）与 Public V1（公开接口）
- 目标：类型安全、分层清晰、接口稳定、可观测
- Swagger UI：配置开启后访问 `/swagger/*`

## 目录

- 功能模块
- 技术栈
- 架构与约定
- 快速开始（本地开发）
- API 概览（前缀与鉴权）
- Swagger
- 开发工具（迁移 / Gen / Wire / Keys）
- 测试与构建
- 规范参考

## 功能模块

- 认证与安全：管理员/用户登录、JWT、会话、2FA（TOTP）
- 内容管理：文章、分类、标签（CRUD + 分页/排序/搜索）
- 评论系统：列表、审核状态变更、点赞
- 反馈与友链：反馈工单、友链管理
- 文件服务：上传、URL 解析、与资源绑定
- 站点设置：键值化站点配置（公开/私有）
- 通知中心：站内通知（DB 持久化 + SSE 推送）
- 验证码与邮件：图形验证码、邮件验证码与发送

## 技术栈

- Web 框架：Fiber v3
- 数据库：PostgreSQL（GORM Gen）
- 缓存：Redis
- 对象存储：MinIO
- 日志：Zerolog
- 文档：swag + Swagger UI（Fiber v3 使用 `github.com/gofiber/contrib/v3/swaggo`）
- 校验：go-playground/validator v10（request DTO 校验）

## 架构与约定

- Clean Architecture：Controller → UseCase → Repo → Infrastructure
- 目录结构与约定、错误码规范、分页约定等详见 [CONVENTIONS.md](./CONVENTIONS.md)

## 快速开始（本地开发）

### 1) 准备依赖

- Go：版本以 [go.mod](./go.mod) 为准
- PostgreSQL
- Redis（Admin Session、验证码、refresh 会话等会用到）
- MinIO（文件直传与对象访问）

### 1.1) 使用 Docker 启动依赖（可选）

下面示例用于本地开发快速拉起依赖服务；生产环境请使用更严格的密码、网络与存储策略。

```bash
# PostgreSQL
docker pull postgres
docker run -d --name nimbus-postgres \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=myp455w0rd \
  -e POSTGRES_DB=blog_db \
  -p 5432:5432 postgres

# Redis
docker pull redis
docker run -d --name nimbus-redis \
  -p 6379:6379 redis

# MinIO（示例版本）
docker pull minio/minio:RELEASE.2025-04-22T22-12-26Z
docker run -d --name nimbus-minio \
  -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=admin" \
  -e "MINIO_ROOT_PASSWORD=admin123456" \
  minio/minio:RELEASE.2025-04-22T22-12-26Z server /data --console-address ":9001"
```

MinIO 初始化建议：
- 在 MinIO 控制台创建 bucket：`nimbus-files`（或与配置文件保持一致）
- 控制台默认地址：`http://localhost:9001`

### 2) 配置

项目运行时会从**工作目录**读取 `config.yaml`（`config.NewConfig()` 直接加载 `config.yaml`），因此建议始终在仓库根目录执行命令。

- 配置文件：[config.yaml](./config.yaml)
- 配置结构体：[config/config.go](./config/config.go)

文件存储对外访问地址（`file_storage.public_base_url`）：
- 本地 MinIO：`http://localhost:9000`（默认）
- 使用本地 MinIO 之外的其他服务：改为你的自定义存储域名（CDN/对象存储网关域名）

建议在本地/测试环境也替换以下敏感项（不要把真实密钥提交到仓库）：
- `postgres.password`
- `redis.password`
- `minio.access_key` / `minio.secret_key`
- `smtp.password`
- `jwt.access_secret` / `jwt.refresh_secret`
- `twofa.encryption_key`
- `openai.api_key`

### 3) 数据库迁移

迁移程序入口：`cmd/migrate/main.go`（基于 golang-migrate）。

```bash
go run ./cmd/migrate -dir ./migrations -action up
```

也可以直接运行（默认：`-dir migrations -action up`）：

```bash
go run ./cmd/migrate
```

常用动作：
- `-action up|down|steps|force|version|drop`
- `-steps N`：用于 `down/steps`
- `-to VERSION`：用于 `force`

### 4) 启动服务

```bash
go run ./cmd/app
```

健康检查：
- `GET /healthz`

## API 概览

### BasePath

Swagger 注解定义 BasePath 为 `/api`（见 [cmd/app/main.go](./cmd/app/main.go)）。

### 路由前缀

- Admin：`/api/admin/*`（Session Cookie 鉴权）
- Public V1：`/api/v1/*`（部分接口公开，部分接口 Bearer JWT 鉴权）

### 鉴权与 Cookie

- Admin Session Cookie：`fiber_session`（由 Fiber session 默认名决定）
- User refresh cookie：`refresh_token`（HttpOnly）
- User access token：`Authorization: Bearer <token>`

更完整的设计说明：
- 用户认证（JWT + refresh）：[DESIGN_USER_AUTH.md](./DESIGN_USER_AUTH.md)
- 管理端认证（Session + 2FA）：[DESIGN_ADMIN_AUTH.md](./DESIGN_ADMIN_AUTH.md)

## Swagger

### 开关

`config.yaml` 中：

```yaml
swagger:
  enabled: true
```

开启后可访问：
- `GET /swagger/*`

### 生成文档

项目引入 docs 包（`cmd/app/main.go` 中 `_ "github.com/scc749/nimbus-blog-api/docs"`），因此需要在更新注解后重新生成：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/app/main.go -d . -o docs --parseInternal --parseDependency
```

说明：不建议加 `--parseGoList=false`，否则在部分 Go 版本下可能导致 `json.RawMessage` 等类型解析失败，从而出现 schema 缺失。

## 开发工具

### 依赖注入（Wire）

项目使用 Google Wire 进行依赖注入，代码生成文件为 `internal/app/wire_gen.go`。

```bash
go install github.com/google/wire/cmd/wire@latest
cd internal/app
wire
```

说明：
- 修改 `internal/app/wire.go` 的 provider 后，需要重新执行 `wire` 生成 `wire_gen.go`
- 通常仅在调整依赖注入关系时需要执行；日常开发不必频繁运行

### 代码生成（GORM Gen）

GORM Gen 入口：`cmd/gen/main.go`，生成目标目录：
- `internal/repo/persistence/gen/model`
- `internal/repo/persistence/gen/query`

```bash
go run ./cmd/gen
```

说明：
- Gen 依赖可用的 Postgres 连接与最新的 schema，建议在迁移完成后执行

### 密钥生成工具

密钥生成入口：`cmd/keys/main.go`，用于生成 JWT 与 TwoFA 加密密钥。

```bash
go run ./cmd/keys -yaml
```

常用参数：
- `-access-bytes`（默认 32）
- `-refresh-bytes`（默认 64）
- `-key-bytes`（默认 32）
- `-yaml=true|false`（默认 true）

## 测试与构建

```bash
go test ./...
```

（Swagger 格式校验测试位于 `internal/controller/http/swagger_test.go`）

```bash
go build -o dist/nimbus-blog-api ./cmd/app
go build -o dist/migrate ./cmd/migrate
```

## 规范参考

- 代码设计规范：[CONVENTIONS.md](./CONVENTIONS.md)
- 浏览量缓冲设计：[DESIGN_POST_VIEWS.md](./DESIGN_POST_VIEWS.md)
- 通知模块设计：[DESIGN_NOTIFICATION.md](./DESIGN_NOTIFICATION.md)
- 用户认证设计：[DESIGN_USER_AUTH.md](./DESIGN_USER_AUTH.md)
- 管理端认证设计：[DESIGN_ADMIN_AUTH.md](./DESIGN_ADMIN_AUTH.md)
