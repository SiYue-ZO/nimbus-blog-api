# Nimbus Blog API 代码设计规范

文档目的：统一后端代码设计与实践，确保与当前实现一致  
适用读者：后端/全栈开发人员、运维工程师  
覆盖范围：架构、目录、命名、接口契约、错误处理、路由、Repo、部署与运维  
更新策略：以代码为准，发现不一致优先更新文档

## 目录（阅读顺序）
- 项目架构（层次与依赖方向）
- 目录结构规范（项目地图）
- 命名规范（包/文件/结构体/方法）
- 枚举常量（DB ENUM 对齐）
- 接口契约规范（UseCase/Repo 合同）
- Entity 规范（领域模型）
- DTO 规范（Request/Response/Output）
- 错误处理规范（哨兵错误/映射/包装）
- 业务状态码规范（AABB 分段）
- HTTP 响应规范（Envelope/Page）
- Controller Handler 规范（解析/校验/映射/调用/响应）
- 路由注册规范（领域路由组织与分组）
- Repo 实现规范（Gen 优先/多表操作）
- UseCase 实现规范（结构/依赖/边界）
- Pkg 封装规范（Functional Options）
- 配置规范（Viper + mapstructure）
- 依赖注入（Wire）
- 接口文档（Swagger）规范
- 部署与运行环境约定
- 数据库迁移执行规范
- 容器网络与主机名约定
- 密钥生成工具

## 一、项目架构

采用 **Clean Architecture（整洁架构）**，分层如下：

| 层级 | 目录 | 职责 | 依赖方向 |
|------|------|------|---------|
| Entry Point | `cmd/` | 程序入口（app/gen/keys/migrate） | 向内 |
| Controller | `internal/controller/http/` | HTTP 路由、请求解析、响应构建 | 依赖 usecase |
| Use Case | `internal/usecase/` | 业务逻辑、接口定义 | 依赖 entity |
| Repository | `internal/repo/` | 数据访问抽象与实现 | 依赖 entity |
| Entity | `internal/entity/` | 纯领域模型 | 无依赖 |
| Pkg | `pkg/` | 可复用基础设施组件 | 无内部依赖 |
| Config | `config/` | 配置加载 | 无内部依赖 |
| DI | `internal/app/` | Wire 依赖注入、应用生命周期 | 组装所有层 |

**数据流方向：** `Controller → UseCase Interface → Repo Interface → Persistence/Cache/Storage`

**依赖约束：**
- Controller 只能依赖 UseCase（接口），不直接依赖 Repo 实现
- Repo 只能向下依赖基础设施（DB/Redis/MinIO/外部 API），不依赖 UseCase
- **UseCase 同层不互调：** 不允许 `usecase.A` 注入/调用 `usecase.B`（同一层横向依赖）；跨领域协作通过抽象到 `repo`（如 `repo.Notifier`）或在当前 UseCase 内实现必要的组合逻辑
- 允许在同一 UseCase 实现内部使用私有 helper（函数或方法）组织逻辑（例如 `validateXxx`、`setXxx`、`blacklistXxx`），这不属于“UseCase 同层互调”

---

## 二、目录结构规范

```
cmd/
  app/main.go                  # 主程序入口
  gen/main.go                  # GORM Gen 代码生成
  keys/main.go                 # 密钥生成工具（生成 JWT 与 TwoFA 加密密钥）
  migrate/main.go              # 数据库迁移

config/
  config.go                    # 配置结构体 + Viper 加载

docs/                          # Swagger 文档（swag init 输出 + 注册包）
  docs.go
  swagger.json
  swagger.yaml

internal/
  app/                         # Wire DI + 应用生命周期
    app.go / wire.go / wire_gen.go

  entity/                      # 领域实体（纯结构体）
    {entity_name}.go

  usecase/                     # 业务逻辑层
    contracts.go               # 所有 UseCase 接口定义
    input/                     # UseCase 入参 DTO
    output/                    # UseCase 出参 DTO
    {domain}/                  # 具体 UseCase 实现（如 auth/admin、auth/user、content、comment 等）
      {domain}.go

  repo/                        # 数据访问层
    contracts.go               # 所有 Repository 接口定义
    persistence/               # Postgres 实现（GORM Gen）
      {entity}_postgres_gen.go
      gen/model/               # 自动生成的 Model
      gen/query/               # 自动生成的 Query
    cache/                     # Redis 缓存实现
    storage/                   # MinIO 对象存储实现
    messaging/                 # SMTP 邮件实现
    webapi/                    # 外部 API 实现（LLM、翻译）
    notification/              # Notifier 实现
    viewbuffer/                # 文章浏览量异步缓冲写入实现

  controller/http/             # HTTP 控制器层
    router.go                  # 主路由注册
    swagger_test.go            # Swagger 文档格式校验测试
    admin/                     # Admin API 模块
      controller.go            # 控制器结构体
      router.go                # 路由注册函数
      {domain}.go              # 各领域 Handler
      request/                 # 请求 DTO（带 json+validate 标签）
      response/                # 响应 DTO（带 json 标签）+ 业务码
    v1/                        # Public API 模块（同 admin 结构）
    shared/                    # 共享：响应信封、分页、工具函数
    bizcode/                   # 全局业务状态码定义
    middleware/                # 中间件

pkg/                           # 可复用基础设施包
  httpserver/                  # Fiber HTTP Server 封装
  logger/                      # Zerolog 日志封装
  postgres/                    # GORM Postgres 连接封装
  redis/                       # Redis 连接封装
  minio/                       # MinIO 客户端封装
  ssehub/                      # SSE Hub（内存级客户端连接管理 + 事件推送）

migrations/                    # SQL 迁移文件
dist/                          # 可执行产物（本地/CI 构建输出）
  nimbus-blog-api
  migrate
```

---

## 三、命名规范

### 包命名

- 全小写，单词组合不加下划线：`usecase`、`bizcode`、`shared`
- 内部包统一放在 `internal/` 下
- import 别名允许按语义使用（例如 `internal/controller/http/shared` 常用别名 `sharedresp`），但不作为包名规范本身

### 文件命名

- 全小写 + 下划线分隔：`admin_postgres_gen.go`、`site_setting.go`、`admin_session.go`
- 按领域划分文件，每个文件对应一个领域概念
- 接口契约文件统一命名 `contracts.go`

### 结构体命名

| 类型 | 规则 | 示例 |
|------|------|------|
| Entity | PascalCase 单数 | `Post`、`User`、`SiteSetting` |
| UseCase 实现 | 统一 `useCase`（未导出） | `useCase` |
| Repo 实现 | `{entity}Repo`（未导出） | `adminRepo`、`captchaRedisStore` |
| Controller | 以 API 模块命名 | `Admin`、`V1` |
| Request DTO | 按操作命名 | `request.Login`、`request.CreatePost` |
| Response DTO | 按用途命名 | `response.PostSummary`、`response.PostDetail` |
| Input DTO | 按操作命名 | `input.CreatePost`、`input.ListPosts` |
| Output DTO | 按用途命名 | `output.PostDetail`、`output.ListResult[T]` |

### 接口命名

| 类型 | 规则 | 示例 |
|------|------|------|
| UseCase 接口 | 以领域名命名 | `Content`、`AdminAuth`、`Comment` |
| Repo 接口 | `{Entity}Repo` / `{Purpose}Store` | `AdminRepo`、`CaptchaStore`、`ObjectStore` |
| Pkg 接口 | `Interface` | `logger.Interface` |

接口编译检查：`var _ Interface = (*Logger)(nil)`

### 方法命名

| 类型 | 规则 | 示例 |
|------|------|------|
| UseCase 方法 | 动词开头（导出） | `Login`、`CreatePost`、`ListUsers` |
| Repo 方法 | CRUD 动词（导出） | `Create`、`GetByID`、`ListAll`、`Delete` |
| Handler 方法 | camelCase（未导出） | `login`、`listPosts`、`createCategory` |
| 路由注册 | `New{Domain}Routes` | `NewAuthRoutes`、`NewContentRoutes` |

### 变量命名

| 类型 | 规则 | 示例 |
|------|------|------|
| 哨兵错误 | `Err` 前缀 | `ErrAdminNotFound`、`ErrRepo` |
| 业务码常量 | `Error` 前缀 | `ErrorParamFormat`、`ErrorAdminNotFound` |
| 包级默认值 | 下划线前缀 | `_defaultPage`、`_defaultAddress` |

---

## 四、枚举常量

- 所有与数据库 ENUM 同名的业务常量统一在 `internal/entity` 定义并在代码中引用，不得在业务代码里直接使用字面字符串。
- 常量分布：
  - 用户状态：`UserStatusActive`、`UserStatusDisabled`
  - 文章状态：`PostStatusDraft`、`PostStatusPublished`、`PostStatusArchived`
  - 评论状态：`CommentStatusPending`、`CommentStatusApproved`、`CommentStatusRejected`、`CommentStatusSpam`
  - 反馈类型：`FeedbackTypeGeneral`、`FeedbackTypeBug`、`FeedbackTypeFeature`、`FeedbackTypeUI`
  - 反馈状态：`FeedbackStatusPending`、`FeedbackStatusProcessing`、`FeedbackStatusResolved`、`FeedbackStatusClosed`
  - 站点设置类型：`SettingTypeString`、`SettingTypeNumber`、`SettingTypeBoolean`、`SettingTypeJSON`
  - 友链状态：`LinkStatusActive`、`LinkStatusInactive`
  - 文件用途：`FileUsagePostCover`、`FileUsagePostContent`、`FileUsageAvatar`
  - 通知类型：`NotificationTypeCommentReply`、`NotificationTypeCommentApproved`、`NotificationTypeAdminMessage`

> 请求校验的 struct tag（如 `validate:"oneof=..."`）保留字面值以配合校验器；业务逻辑与持久化层严格使用上述常量。

---

## 五、接口契约规范

### 前后端协作与自托管约定
- 前端构建阶段不依赖后端：前端根 Layout 元数据使用静态默认值，后端无需在构建时提供可达性。
- 运行时通过同域 `/api/*` 访问后端：部署层用反向代理将 `/{prefix}` 转发到后端（本项目前缀：`/api/admin`、`/api/v1`）。
- 前端请求默认携带 Cookie（`credentials: include`），确保 Session 与权限校验在同域场景下生效。
- 公开数据走 `v1`，管理端走 `admin`；必要时使用短缓存或不缓存策略由前端页面级控制（例如 `fetch(..., { cache: "no-store" })`）。

### UseCase 接口（`internal/usecase/contracts.go`）

**基本规则：**

- 所有 UseCase 接口集中定义在一个文件中
- 每个方法第一个参数必须是 `context.Context`
- 入参使用 `input.*` 结构体（复杂参数）或原始类型（简单参数如 `id int64`）
- 出参使用 `output.*` 结构体或泛型 `output.ListResult[T]` / `output.AllResult[T]`
- 返回模式：`(结果, error)` 或 `error`
- 不使用命名返回值
- 参数名不得与导入的包名冲突（如避免用 `input` 作参数名）

**ID 类型：**

- 所有实体 ID 统一使用 `int64`（对应数据库 `BIGSERIAL`/`BIGINT`），不使用 `string`

### 路径 ID → 请求体 ID 统一处理模式

- 适用范围：所有 Admin 更新接口形如 `PUT /api/admin/{resource}/{id}` 或 `PUT .../{id}/status`
- 请求体 DTO：包含 `id int64` 字段，校验规则为 `validate:"required,gte=1"`
- Handler 流程：
  1. 从路径读取 `id` 并解析为 `int64`
  2. `ctx.Bind().Body(&body)`
  3. 将路径 `id` 赋值给 `body.ID`
  4. 执行 `r.validate.Struct(body)`
  5. 调用 UseCase，`id` 参数使用解析值或 `body.ID`（二者一致）
- 设计动机：保持 DTO 校验一致性（必填 ID），同时让前端不必在请求体重复携带路径 ID；Swagger 文档将正确展示请求体包含 ID 字段

**返回值一致性：**

- 分页列表统一返回 `*output.ListResult[T]`
- 全量列表统一返回 `*output.AllResult[T]`
- 创建操作统一返回 `(int64, error)`（返回新记录 ID）

**接口声明顺序：**

1. `Auth`（门面）→ `AdminAuth` → `UserAuth`
2. `Captcha` → `Email`（辅助服务）
3. `File`（对象存储 + 文件元数据管理）
4. `User` → `Content` → `Comment` → `Feedback` → `Link` → `Setting` → `Notification`（业务领域）

**方法分组与排序：**

- 同一接口内用 `// Admin` 和 `// Public` 注释分隔管理端和公共端方法
- Admin 组内按 `List → Get → Create → Update → Delete` 顺序排列
- 多子实体接口（如 Content）按子实体分块：Post → Category → Tag，每块内保持同一排序
- 工具方法（如 `GenerateSlug`）放在 Admin 组末尾

```go
type Content interface {
    // Admin
    ListPosts(ctx context.Context, params input.ListPosts) (*output.ListResult[output.PostSummary], error)
    GetPostByID(ctx context.Context, id int64) (*output.PostDetail, error)
    CreatePost(ctx context.Context, params input.CreatePost) (int64, error)
    UpdatePost(ctx context.Context, params input.UpdatePost) error
    DeletePost(ctx context.Context, id int64) error
    GenerateSlug(ctx context.Context, title string) (string, error)

    // Public
    ListPublicPosts(ctx context.Context, params input.ListPublicPosts, userID *int64) (*output.ListResult[output.PostSummary], error)
    GetPublicPostBySlug(ctx context.Context, slug string, userID *int64) (*output.PostDetail, error) // UseCase 层校验 status=published
    GetAllPublicCategories(ctx context.Context) (*output.AllResult[output.CategoryDetail], error)
}
```

> **Public 端查询安全规则：** `GetPublicPostBySlug` 在 Repo 取到文章后校验 `status == "published"`，非 published 返回 `ErrNotFound`，防止草稿/归档文章通过 slug 泄露。

### Slug 生成规则

- 提示词：生成slug: [ %s ] → 英文小写连字符，核心关键词
- 生成流程：
  - 首选调用 TranslationWebAPI 将标题翻译为英文并进行 slugify（小写 + 连字符）
  - 翻译失败则调用 LLMWebAPI，使用上述提示词生成文本：
    - 先用正则 `\b[a-z0-9]+(?:-[a-z0-9]+)*\b` 抽取第一个符合 slug 形态的片段
    - 未命中则回退对 LLM 输出进行 slugify
  - 仍失败返回 `ErrSlugGenerate`

### Repository 接口（`internal/repo/contracts.go`）

- 所有 Repo 接口集中定义在一个文件中，使用 `type ( ... )` 分组声明
- 参数和返回值使用 `entity.*` 结构体
- 方法签名第一个参数必须是 `context.Context`

```go
type PostRepo interface {
    Create(ctx context.Context, p entity.Post) (int64, error)
    Update(ctx context.Context, p entity.Post) error
    Delete(ctx context.Context, id int64) error
    GetByID(ctx context.Context, id int64) (*entity.Post, error)
}
```

**Persistence Repo 总览：**

| Repo 接口 | 对应表 | 说明 |
|-----------|--------|------|
| `AdminRepo` | admins | 管理员 |
| `UserRepo` | users | 用户 |
| `PostRepo` | posts | 文章 |
| `TagRepo` | tags | 标签 |
| `CategoryRepo` | categories | 分类 |
| `CommentRepo` | comments | 评论 |
| `PostLikeRepo` | post_likes | 文章点赞（Toggle/Remove/HasLiked） |
| `PostViewRepo` | post_views | 文章浏览记录（异步缓冲写入，由 `viewbuffer` 实现） |
| `CommentLikeRepo` | comment_likes | 评论点赞（Toggle/Remove/HasLiked） |
| `FeedbackRepo` | feedbacks | 反馈 |
| `LinkRepo` | links | 友链 |
| `SiteSettingRepo` | site_settings | 站点设置 |
| `RefreshTokenBlacklistRepo` | refresh_token_blacklist | 刷新令牌黑名单 |
| `RefreshTokenStore` | — | 刷新令牌当前值存储（Redis），提供 `Set`/`Get`/`Delete` |
| `FileRepo` | files | 文件元数据 |
| `NotificationRepo` | notifications | 通知 CRUD |
| `Notifier` | — | 通知发送（DB 持久化 + SSE 推送） |

### Link 领域契约（Admin / Public）
- 用途：管理友情链接记录（名称、URL、Logo、描述、排序、状态），前台公开展示 active 友链
- UseCase 接口（`internal/usecase/contracts.go`）：
  - Admin：
    - `ListLinks(ctx, params input.ListLinks) (*output.ListResult[output.LinkDetail], error)`
    - `CreateLink(ctx, params input.CreateLink) (int64, error)`
    - `UpdateLink(ctx, params input.UpdateLink) error`
    - `DeleteLink(ctx, id int64) error`
  - Public：
    - `GetAllPublicLinks(ctx context.Context) (*output.AllResult[output.LinkDetail], error)`（仅返回 `status=active`）
- Admin 路由（REST）：
  - `GET /api/admin/links`（分页，支持 `keyword/sort_by/order`；默认 `sort_order asc`）
  - `POST /api/admin/links`（创建）
  - `PUT /api/admin/links/{id}`（更新）
  - `DELETE /api/admin/links/{id}`（删除）
- Public 路由：
  - `GET /api/v1/links`（全量，`status=active`）
- 入参 DTO（`internal/usecase/input/`）：
  - `ListLinks`：`page, page_size, keyword?, sort_by?, order?`
  - `CreateLink`：`name, url, description?, logo?, sort_order?, status`
  - `UpdateLink`：`id, name?, url?, description?, logo?, sort_order?, status?`（Controller 在 `ctx.Bind().Body(&body)` 后将路径参数 `id` 赋值给 `body.ID`，再执行校验）
- 字段与枚举：
  - `status` 使用常量：`LinkStatusActive` / `LinkStatusInactive`
  - `logo` 与 `description` 可空；Controller 层将空字符串归一化为 `nil`
  - `sort_order` 为整型，默认 `0`，用于前台展示排序
- 前端协作：
  - Admin 前端调用上述 Admin 路由进行 CRUD；更新弹窗提交完整 `UpdateLink` 字段
  - 前台展示页面通过 `GET /api/v1/links` 获取公开友链；Logo 路径按前端规范使用同域解析或文件服务 `/api/v1/files/{key}`

### 缓存与写缓冲目录规范

- 缓存（`internal/repo/cache/`）：
  - 目的：面向会话/验证码等临时数据的 Key-Value 存储（通常带 TTL）
  - 典型实现：`CaptchaStore`、`EmailCodeStore`、`RefreshTokenStore`（Redis）
  - 接口形态：尽量保持简洁的 `Set`/`Get`/`Delete`，不承载复杂业务规则
  - 使用约束：除 `RefreshTokenStore` 作为 refresh 会话权威外，缓存不作为权威数据源；失效策略由 UseCase 明确掌控，避免“隐式缓存”导致状态不一致
  - 适用场景：一次性验证码、临时令牌当前值、限流计数等短生命周期数据

- 写缓冲（`internal/repo/viewbuffer/`）：
  - 目的：高频、低价值但需要累计的写操作（如浏览量）进行批量合并与定期落库
  - 典型实现：文章浏览量缓冲写入（`post_view_buffered.go`）
  - 接口形态：对外仍暴露领域 Repo 接口（如 `PostViewRepo`），内部以缓冲队列 + 定时 Flush 实现
  - 使用约束：保证最终一致性，不用于强一致、用户敏感的场景（如余额、权限）
  - 适用场景：计数器、埋点、统计型数据的聚合写入

- 放置准则：
  - 临时 Key-Value + TTL → 放入 `cache/`
  - 累计计数/聚合写入 → 放入 `viewbuffer/`
  - 不将“站内通知”或“外部邮件”实现混入上述目录，分别归属 `notification/` 与 `messaging/`

### 字段清空语义（featured_image）

- 责任边界：
  - Controller 层负责将前端传入的空字符串归一化为 nil，表达“显式清空”意图
  - Repo 层负责将实体中的 nil 字段持久化为数据库 NULL（例如 `featured_image`）
  - 未提供该字段表示“不修改”，Repo 层不更新该列
- 协作约定（Admin 更新文章）：
  - 前端移除封面时发送 `featured_image: ""`
  - Controller 解析后将 `""` 归一化为 `nil`
  - Repo 在 `Update` 中对 `nil` 执行 `UpdateSimple(FeaturedImage.Null())`，写入 NULL
  - 读取文章时若为 NULL，前端视为“无封面”

### 通知触发策略

> **所有评论相关通知统一在 `UpdateCommentStatus`（状态变更为 `approved`）时触发，`SubmitComment` 不发送任何通知。**

| 触发点 | 通知类型 | 接收人 | 说明 |
|--------|----------|--------|------|
| `UpdateCommentStatus(approved)` | `comment_approved` | 评论者 | 评论审核通过后通知评论者 |
| `UpdateCommentStatus(approved)` | `comment_reply` | 被回复者 | 回复评论审核通过后通知父评论作者（跳过自回复） |

`UpdateCommentStatus` 支持 `approved`/`rejected`/`spam` 三种状态，仅 `approved` 触发通知。

**设计原则：** 保证用户收到通知时，对应内容一定已可见（status=approved）。

#### 通知数据模型（meta JSONB）

- `notifications.meta` 为 JSONB 扩展字段，用于存放通知的关联信息与前端跳转信息（例如 `post_id`、`post_slug`、`comment_id`、`target_url` 等）。
- meta 字段的 key 统一使用 `internal/entity/notification_meta.go` 中的常量，避免散落魔法字符串。
- 推荐写入 `target_url`，使通知列表渲染与跳转无需额外查表。

#### 通知摘要长度控制

- 摘要截断在 `repo.Notifier.Send` 内部执行，评论 UseCase 传入完整内容
- 通知的 `content` 字段保存已截断的摘要，读取 REST 与 SSE 推送统一返回该摘要
- 默认最大长度 100 字符（Unicode 安全截断，尾部追加 `...`）
- 不在 `usecase.Comment` 层进行截断，保持领域职责边界清晰

#### 通知模块与邮箱消息的边界

- 站内通知：使用 `repo.Notifier`，职责为持久化到 `notifications` 表并通过 `pkg/ssehub` 推送到在线客户端
- 邮箱消息：属于外部通信通道，归属 `usecase.Email` + `internal/repo/messaging`（如 `email_smtp.go`），可用于登录验证码、密码重置等不属于站内通知的场景
- 目录规范：
  - 站内通知实现放在 `internal/repo/notification/`
  - 邮件发送实现放在 `internal/repo/messaging/`
- 联动策略：如需在某类站内通知同时发送邮件，由上层 UseCase 协调分别调用 `repo.Notifier.Send` 与 `usecase.Email`，无需合并目录或改变接口边界

### 评论业务规则

**提交校验（`SubmitComment` UseCase 层）：**

- `parent_id` 非空时校验：父评论存在、属于同一篇文章（`parent.PostID == params.PostID`）、且状态为 `approved`，否则返回 `ErrInvalidParent`
- `Content` 字段 Request DTO 限制：`validate:"required,min=1,max=5000"`

**删除权限（`DeleteOwnComment`）：**

- 校验 `comment.UserID == userID`，不匹配返回 `ErrForbidden`，Handler 映射为 `403 Forbidden`

### 文件存储规范（MinIO Object Key）

**路径规则：** `{分类}/{uuid}.ext`，**不在路径中嵌入 resource_id**。

| upload_type | Object Key 格式 | 示例 |
|-------------|-----------------|------|
| `avatar` | `avatars/{uuid}.ext` | `avatars/a1b2c3d4.jpg` |
| `post_cover` | `posts/covers/{uuid}.ext` | `posts/covers/e5f6g7h8.png` |
| `post_content` | `posts/content/{uuid}.ext` | `posts/content/i9j0k1l2.webp` |

**设计原则：**

- 路径仅区分**用途分类**，不绑定具体资源 ID
- 文件与资源的绑定关系由 `files` 表的 `resource_id` 字段管理
- 新建文章时先上传文件（此时无文章 ID），文章创建后自动绑定
- 更新文章时先按用途解绑旧文件（`ClearResourceIDByResourceAndUsage`），再重新绑定当前引用的文件
- `files` 表的 `usage` 字段对应 DB 枚举 `file_usage`（`post_cover`, `post_content`, `avatar`）
- 不提供“全量解绑” API；仅允许按用途解绑，避免不同实体 ID 数值重合导致跨资源污染

**头像上传与设置更新（Admin Settings）：**
- 管理员在「站点设置 → 博主信息」上传头像，前端调用 `POST /admin/files/upload-url`（`upload_type=avatar`）获得预签名 URL 与 `object_key`
- 浏览器用 `PUT` 直传到 MinIO（同域 `/minio/...` 代理），UseCase `SaveMeta` 已写入 `files` 表并标记 `usage=avatar`
- 上传成功后，前端使用 `PUT /admin/settings/profile.avatar` 将设置值更新为该 `object_key`
- 前台展示通过 `resolveImageURL(profile.avatar)` 支持本地静态路径或 MinIO key
- 绑定策略：生成上传 URL 时，如 `upload_type=avatar` 且存在管理员会话，自动将 `files.resource_id` 设为当前管理员 ID；在 `upsert profile.avatar` 时，后端先按 `usage=avatar` + `resource_id=admin_id` 清空旧绑定（`ClearResourceIDByResourceAndUsage`），再将新 `object_key` 绑定到当前管理员 ID（以 `/author.png` 开头的默认本地路径或以 `/`/`http` 开头的值不参与绑定）

**文章文件绑定流程（`bindPostFiles`）：**

Content usecase 的 `CreatePost` / `UpdatePost` 执行成功后自动调用 `bindPostFiles`：

1. `ClearResourceIDByResourceAndUsage(ctx, postID, "post_cover"|"post_content")` — 按用途清除该文章已绑定文件的 `resource_id`（置 NULL）
2. 收集封面图 object key（`FeaturedImage` 字段，直接是 object key）
3. 用正则 `/api/v1/files/([^\s)"']+)` 从文章 markdown 内容中提取所有内容图片的 object key
4. 逐一调用 `UpdateResourceID(ctx, key, postID)` 重新绑定

该流程保证文件绑定始终与文章当前实际引用一致：更换封面、删除内容图片、新增图片后，旧的未引用文件自动解绑。

---

## 六、Entity 规范

- 纯结构体，**不带任何 struct tag**（无 json/gorm/db 标签）
- 可选字段使用指针类型：`*string`、`*time.Time`、`*int64`
- 统一包含 `CreatedAt time.Time` 和 `UpdatedAt time.Time` 时间戳
- 不包含业务逻辑方法
- 仅允许依赖 Go 标准库（如 `time`、`encoding/json`），禁止依赖 `internal/` 任何包与第三方库

```go
type Post struct {
    ID            int64
    Title         string
    Slug          string
    Excerpt       *string      // 可选字段用指针
    Content       string
    FeaturedImage *string
    AuthorID      int64
    CategoryID    int64
    Status        string
    IsFeatured    bool
    PublishedAt   *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

---

## 七、DTO 规范

### UseCase Input（`internal/usecase/input/`）

- 纯结构体，**不带 struct tag**
- 可选字段用指针类型
- 通过嵌入 `PageParams`、`KeywordParams`、`SortParams` 组合分页/搜索能力
- 过滤参数使用类型别名：`StringFilterParam`、`IntFilterParam`、`BoolFilterParam`

```go
type ListPosts struct {
    PageParams
    Keyword    *KeywordParams
    Sort       *SortParams
    CategoryID IntFilterParam
    Status     StringFilterParam
}
```

### UseCase Output（`internal/usecase/output/`）

- 带 `json` tag（snake_case 格式）
- 使用 Base 结构体嵌入实现继承：`BasePost → PostSummary → PostDetail`
- 关联信息使用独立结构体：`AuthorInfo`（嵌入 `PostSummary` / `PostDetail`，包含 `id` + `nickname` + `specialization`）、`LikeInfo`（`Liked *bool` + `Likes int32`，嵌入文章/评论 Output DTO）
- 分页结果使用泛型：`ListResult[T]`、`AllResult[T]`

```go
type ListResult[T any] struct {
    Items    []T   `json:"items"`
    Page     int   `json:"page"`
    PageSize int   `json:"page_size"`
    Total    int64 `json:"total"`
}
```

### Controller Request（`internal/controller/http/{module}/request/`）

- 带 `json` + `validate` tag
- 仅接收前端传入的字段
- 所有字段必须携带合适的校验规则（长度、枚举、格式等），不允许裸 `required`

```go
type Login struct {
    Username string `json:"username" validate:"required,min=3,max=32"`
    Password string `json:"password" validate:"required,min=1,max=20"`
}
```

**校验规则约定：**

| 字段类型 | 规则 | 说明 |
|----------|------|------|
| 密码（新建/修改） | `min=8,max=20` | 统一 8-20 字符 |
| 密码（登录/旧密码） | `min=1,max=20` | 允许任意已有密码，仅限长度 |
| 字符串字段 | 必须携带 `max=N` | 防止超长输入 |
| 可选字符串 | `omitempty,max=N` | 可选但有上限 |
| 枚举字段 | `required,oneof=a b c` | 与数据库 ENUM 一致 |
| int64 ID | `required,gte=1` | 确保正整数（`required` 对数字仅校验非零） |
| 可选 URL | `omitempty,http_url` 或 `omitempty,max=N` | 限定协议或长度 |
| OTP 验证码 | `len=6,numeric` | 6位纯数字 |
| 数组元素 | `omitempty,dive,gte=1` | 每个元素独立校验 |

### Controller Response（`internal/controller/http/{module}/response/`）

- 带 `json` tag（snake_case 格式）
- 与 UseCase Output 字段可能不同（按 API 需求裁剪）
- 同目录下 `codes.go` 定义模块级业务码

---

## 八、错误处理规范

### 哨兵错误（UseCase 层）

每个 UseCase 包定义自己的哨兵错误：

```go
var (
    ErrRepo          = errors.New("repo")
    ErrNotFound      = errors.New("not found")
    ErrForbidden     = errors.New("forbidden")
    ErrInvalidParent = errors.New("invalid parent comment")
    ErrPasswordWrong = errors.New("password wrong")
    ErrUserDisabled  = errors.New("user disabled")
)
```

Handler 层需对权限类哨兵错误精确映射 HTTP 状态码（如 `ErrForbidden → 403`、`ErrInvalidParent → 400`），避免统一返回 500。

### 错误包装

UseCase 层对下游错误统一用 `%w` 包装哨兵错误（如 `ErrRepo`），保留可解包性与错误归类能力：

```go
return nil, fmt.Errorf("%w: %v", ErrRepo, err)
```

### GORM 错误转换

Repo 返回的 `gorm.ErrRecordNotFound` 在 UseCase 层转换为领域哨兵错误：

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, ErrAdminNotFound
}
```

### Controller 层错误映射

使用 `errors.Is` + `switch` 将哨兵错误映射为 HTTP 状态码 + 业务码：

```go
switch {
case errors.Is(err, authUC.ErrAdminNotFound):
    httpCode = http.StatusNotFound
    bizCode  = response.ErrorAdminNotFound
    msg      = "admin not found"
case errors.Is(err, authUC.ErrRepo):
    httpCode = http.StatusInternalServerError
    bizCode  = response.ErrorDatabase
    msg      = "database error"
}
return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
```

---

## 九、业务状态码规范

采用 **AABB 四位字符串** 格式（`string` 类型），Admin 和 V1 使用**独立的模块编号体系**，所有码值统一为 4 位零填充字符串：

- **AA**：模块编号
- **BB**：错误分组

### BB 分组规则（Admin 和 V1 通用）

| BB 范围 | 类型 | 起始编号 |
|---------|------|---------|
| 00 | 成功 | AA00 |
| 01-19 | 数据相关错误（不存在、类型不支持等） | AA01 |
| 20-39 | 认证授权错误（密码错误、会话缺失等） | AA20 |
| 40-59 | 业务逻辑错误（流程限制、校验失败等） | AA40 |
| 60-79 | 操作/系统错误（CRUD 失败、调用失败等） | AA60 |
| 80-99 | 预留扩展 | AA80 |

**编号规则：**

- 所有码值使用 **`string` 类型**，统一 4 位零填充（如 `"0001"`、`"0120"`、`"1401"`）
- 每个 BB 子区间内，从子区间首位（x0 或 x1）开始，**连续递增，不留空号**
- 不需要的分组可以整组省略（如某模块无认证相关错误则不定义 20-39 段）
- 每个 `const` 块内按分组顺序排列，每组前加注释标明范围

### 全局码（`bizcode/codes.go`）

| 码值 | 常量 | 含义 |
|------|------|------|
| `"0000"` | `Success` | 成功 |
| `"0001"` | `ErrorParam` | 参数错误 |
| `"0002"` | `ErrorParamMissing` | 参数缺失 |
| `"0003"` | `ErrorParamFormat` | 参数格式错误 |
| `"0004"` | `ErrorDataNotFound` | 数据不存在 |
| `"0006"` | `ErrorInvalidParams` | 无效参数 |
| `"0020"` | `ErrorUnauthorized` | 未授权 |
| `"0021"` | `ErrorTokenInvalid` | Token 无效 |
| `"0022"` | `ErrorTokenExpired` | Token 已过期 |
| `"0023"` | `ErrorPermissionDenied` | 权限不足 |
| `"0024"` | `ErrorLoginRequired` | 请先登录 |
| `"0060"` | `ErrorSystem` | 系统错误 |
| `"0061"` | `ErrorDatabase` | 数据库错误 |
| `"0062"` | `ErrorCache` | 缓存错误 |
| `"0064"` | `ErrorThirdParty` | 第三方服务错误 |
| `"0065"` | `ErrorConfigNotLoaded` | 配置未加载 |

### Admin 模块码（`admin/response/codes.go`）

| AA | 模块 | 使用的 BB 分组及起始码 |
|----|------|----------------------|
| 11 | 管理员 | 数据 `"1101"` / 认证 `"1120"` / 业务 `"1140"` / 操作 `"1160"` |
| 12 | 文件 | 数据 `"1201"`-`"1202"` / 操作 `"1260"`-`"1264"` |
| 13 | 用户 | 操作 `"1360"` |
| 14 | 内容 | 数据 `"1401"` / 操作 `"1460"` |
| 15 | 评论 | 操作 `"1560"` |
| 16 | 反馈 | 操作 `"1660"` |
| 17 | 友链 | 操作 `"1760"` |
| 18 | 设置 | 操作 `"1860"` |
| 19 | 通知 | 操作 `"1960"` |

### V1 模块码（`v1/response/codes.go`）

V1 模块编号 01-09，统一使用 4 位零填充字符串格式 `"AABB"`：

| AA | 模块 | 使用的 BB 分组及起始码 |
|----|------|----------------------|
| 01 | 认证 | 认证 `"0120"` |
| 02 | 文件 | 操作 `"0260"` |
| 03 | 用户 | 数据 `"0301"` |
| 04 | 内容 | 数据 `"0401"` / 操作 `"0460"` |
| 05 | 评论 | 操作 `"0560"` |
| 06 | 反馈 | 操作 `"0660"` |
| 07 | 友链 | 操作 `"0760"` |
| 08 | 设置 | 操作 `"0860"` |
| 09 | 通知 | 操作 `"0960"` |

### 组织方式

- 全局码定义在 `bizcode/codes.go`
- 模块码定义在各模块 `response/codes.go` 中
- 每个模块的 `codes.go` 先 re-export 全局码，再定义本模块私有码
- 每个 `const` 块按 BB 分组顺序排列，每组前加注释标明范围

```go
// 示例：内容模块 (14xx)
const (
    // 数据相关 (1401-1419)
    ErrorPostNotFound = "1401" // 文章不存在

    // 操作相关 (1460-1479)
    ErrorCreatePostFailed     = "1460" // 创建文章失败
    ErrorUpdatePostFailed     = "1461" // 更新文章失败
    ErrorDeletePostFailed     = "1462" // 删除文章失败
    ErrorUpdatePostTagsFailed = "1463" // 更新文章标签失败
    ErrorCreateCategoryFailed = "1464" // 创建分类失败
    ...
    ErrorGenerateSlugFailed   = "1470" // 生成 Slug 失败
)
```

---

## 十、HTTP 响应规范

### 统一响应信封

```go
type Envelope struct {
    Code    string      `json:"code"`           // "0000" = 成功
    Message string      `json:"message"`        // "ok" 或错误描述
    Data    interface{} `json:"data,omitempty"`
}
```

- 成功：`sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))` → HTTP 200
- 失败：`sharedresp.WriteError(ctx, httpCode, bizCode, msg)` → 对应 HTTP 状态码
- 使用 Functional Options：`WithData`、`WithMsg`、`WithCode`

### 分页响应

```go
type Page[T any] struct {
    List []T `json:"list"`
    PageMeta
}

type PageMeta struct {
    CurrentPage int   `json:"current_page"`
    PageSize    int   `json:"page_size"`
    TotalItems  int64 `json:"total_items"`
    TotalPages  int   `json:"total_pages"`
}
```

构建方式：`sharedresp.NewPage(list, page, pageSize, total)`

### 分页查询参数命名（URL Query Parameters）

统一使用 **snake_case**，与 JSON 响应格式保持一致：

| 参数 | 说明 | 示例 |
|------|------|------|
| `page` | 页码（默认 1） | `?page=2` |
| `page_size` | 每页数量（默认 10，最大 100） | `?page_size=15` |
| `sort_by` | 排序字段（需在 `WithAllowedSortBy` 白名单中） | `?sort_by=created_at` |
| `order` | 排序方向：`asc` / `desc`（默认 `desc`） | `?order=asc` |
| `keyword` | 关键词搜索 | `?keyword=test` |
| `filter.*` | 过滤参数 | `?filter.status=published` |

**各实体可排序字段：**

| 实体 | 允许的 `sort_by` 值 | 默认排序 |
|------|---------------------|----------|
| Post | `created_at`, `updated_at`, `views`, `likes` | `created_at desc` |
| Category | `name`, `created_at` | `created_at desc` |
| Tag | `name`, `created_at` | `created_at desc` |
| Comment | `created_at` | `created_at desc` |
| User | `created_at` | `created_at desc` |
| Feedback | `created_at` | `created_at desc` |
| Link | `sort_order`, `created_at`, `name` | `sort_order asc, created_at desc` |
| File | `created_at`, `file_size` | `created_at desc` |
| Notification | `created_at` | `created_at desc` |

---

## 十一、Controller Handler 规范

标准流程（5 步）：

1. **解析请求**：`ctx.Bind().Body(&body)` 或 `ctx.Params("id")` / `sharedresp.ParsePageQueryWithOptions(ctx, ...)`
2. **参数校验**：`r.validate.Struct(body)`
3. **构造入参**：手动映射 `request.*` → `input.*`
4. **调用 UseCase**：`r.content.CreatePost(ctx.Context(), params)`
5. **构造响应**：手动映射 `output.*` → `response.*`，返回 `WriteSuccess` / `WriteError`

### 日志格式

```go
r.logger.Error(err, "http - admin - auth - login - usecase")
//                     层      模块    领域   方法名    步骤
```

格式：`"http - {module} - {domain} - {handler} - {step}"`

- **module**：`admin` / `v1`
- **domain**：`auth` / `content` / `user` / `comment` / `feedback` / `link` / `setting` / `file` / `notification`
- **handler**：camelCase 方法名，如 `login`、`listPosts`、`twoFASetup`
- **step**：`parse body` / `validate body` / `usecase` / `otp validate` / `session get` 等

### Handler 内部变量命名规范

| 场景 | 变量名 | 示例 |
|------|--------|------|
| 请求体 | `body` | `var body request.CreatePost` |
| 分页查询 | `pq` | `pq := sharedresp.ParsePageQueryWithOptions(ctx, ...)` |
| UseCase 返回结果 | `result` | `result, err := r.content.ListPosts(...)` |
| 响应列表 | `list` | `list := make([]response.PostSummary, 0, len(result.Items))` |
| URL 路径参数（原始字符串） | `id` / `key` / `slug` | `id := ctx.Params("id")`、`slug := ctx.Params("slug")` |
| URL 路径参数（解析后） | `nid` | `nid, err := strconv.ParseInt(id, 10, 64)` |
| 从 JWT claims 获取的用户 ID | `uid` | `uid, err := claims.UserIDInt()` |
| 从 session 获取的 admin ID | `aid` | `aid, _ := strconv.ParseInt(idStr, 10, 64)` |
| 循环变量 | 单字母缩写 | `p`(post)、`c`(category/comment)、`t`(tag)、`u`(user)、`f`(feedback)、`l`(link)、`s`(setting) |
| 错误映射三元组 | `httpCode` / `bizCode` / `msg` | 用于 `switch` 错误映射块 |

### ID 解析统一模式

GET（详情）/ DELETE / PUT（更新）统一从 URL 路径参数取 ID：

```go
id := ctx.Params("id")
if id == "" {
    return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
}
nid, err := strconv.ParseInt(id, 10, 64)
if err != nil {
    return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
}
```

字符串标识符（如 settings 的 key）：

```go
key := ctx.Params("key")
if key == "" {
    return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing key")
}
```

### PUT 更新 Handler 标准流程

ID 从 URL 取，业务字段从 body 取，两者合并构造 input：

```go
func (r *Admin) updatePost(ctx fiber.Ctx) error {
    // 1. 从 URL 解析 ID
    id := ctx.Params("id")
    nid, err := strconv.ParseInt(id, 10, 64)
    // 2. 解析 body
    var body request.UpdatePost
    if err := ctx.Bind().Body(&body); err != nil {
        return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
    }
    // 3. 校验
    if err := r.validate.Struct(body); err != nil {
        return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
    }
    // 4. 合并调用 — ID 来自 URL，其余来自 body
    r.content.UpdatePost(ctx.Context(), input.UpdatePost{
        ID:    nid,
        Title: body.Title,
        // ...
    })
}
```

### 用户禁用状态的额外步骤

Admin 端在执行 `PUT /users/:id/status` 将 `status=disabled` 时，应先更新状态以阻断刷新续签，然后尽力撤销刷新会话：

- 先调用 `User.UpdateStatus(ctx, id, status)` 更新用户状态（确保 refresh 立即在 UseCase 层被拒绝）
- 再调用 `UserAuth.RevokeUserRefreshToken(ctx, userID)` 尝试撤销刷新会话（写入 `refresh_token_blacklist`，并删除 Redis 中当前刷新令牌键）
- 撤销失败记录日志即可，不影响禁用结果（禁用已生效，refresh 也会被拒绝）
- 撤销会删除 Redis refresh 会话键，使该用户现有 access token 在后续鉴权时立即失效（`NewUserJWTMiddleware` 会校验会话）

### 分页响应构建

分页参数统一取自 UseCase 返回的 `result`，而非 controller 层的 `pq`：

```go
sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)
```

### V1 JWT Claims 提取模式

V1 handler 中提取当前登录用户 ID 的标准写法：

```go
claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
if !ok || claims == nil {
    return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
}
uid, err := claims.UserIDInt()
if err != nil {
    return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
}
```

- `authUC` 导入路径：`"github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"`
- 统一使用 `uid` 变量名存储解析后的用户 ID

### 点赞状态：`LikeInfo` 与 `getLikeInfo`

与 `AuthorInfo` / `getAuthorInfo` 同一模式：UseCase 定义 `getLikeInfo` 逐项查询，`Liked *bool`（`nil` = 未登录，`true`/`false` = 已登录）。
V1 Response 同名 `LikeInfo`（`liked *bool` + `likes int32`），直接映射，无冗余独立 `Likes` 字段。Admin Response 保留独立 `Likes int32`，通过 `p.Like.Likes` 读取。

### V1 Handler 层过滤模式

当 UseCase 未提供精确的公共端方法时，可在 Handler 层过滤。例如公开设置接口复用 Admin 的全量查询方法，在 Handler 中过滤 `IsPublic`：

```go
func (r *V1) listSettings(ctx fiber.Ctx) error {
    result, err := r.setting.GetAllSiteSettings(ctx.Context())
    // ...
    list := make([]response.SiteSettingDetail, 0)
    for _, s := range result.Items {
        if !s.IsPublic {
            continue
        }
        list = append(list, response.SiteSettingDetail{...})
    }
    return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}
```

---

## 十二、路由注册规范

每个领域一个 `New{Domain}Routes` 函数，接收路由组 + 依赖，内部构造 Controller 并注册路由：

```go
func NewContentRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, content usecase.Content) {
    r := &Admin{logger: l, validate: validator.New(...), sess: store, content: content}
    contentAuthGroup := apiAdminGroup.Group("/content", middleware.NewAdminSessionMiddleware(store))
    {
        contentAuthGroup.Get("/posts", r.listPosts)
        contentAuthGroup.Get("/posts/:id", r.getPost)
        contentAuthGroup.Post("/posts", r.createPost)
        contentAuthGroup.Put("/posts/:id", r.updatePost)
        contentAuthGroup.Delete("/posts/:id", r.deletePost)
    }
}
```

### RESTful 路由规范

**资源 CRUD 路由：**

| 方法 | 路径 | 用途 | ID 来源 |
|------|------|------|---------|
| GET | `/resources` | 列表 | — |
| GET | `/resources/:id` | 详情 | URL 路径参数 |
| POST | `/resources` | 创建 | — |
| PUT | `/resources/:id` | 更新 | URL 路径参数 |
| DELETE | `/resources/:id` | 删除 | URL 路径参数 |

**关键规则：**

- 更新（PUT）路由**必须**在路径中包含资源标识符（`:id` 或 `:key`），ID 从 URL 解析，不从 body 中取
- 路径参数命名：数值主键用 `:id`，字符串键用 `:key`

**子资源状态变更：**

| 方法 | 路径 | 用途 |
|------|------|------|
| PUT | `/resources/:id/status` | 变更资源状态（approve、ban 等） |

状态变更统一使用 `PUT /:id/status`，不使用 `POST /:id/动词` 形式。保持 users / comments / feedbacks 风格一致。

**动作型端点（非 CRUD）：**

认证、文件上传等非资源操作使用 `POST + 动作路径`：

```
POST /auth/login
POST /auth/logout
POST /auth/reset
POST /auth/2fa/setup
POST /files/upload-url
```

**请求体解析与空请求体规则：**
- 所有 `POST`/`PUT` 端点按照 JSON 解析：body 解析失败或缺失时返回 `400`，业务码使用参数格式错误码（`"0003"`），message 为 `invalid request body`。
- 对于无必填字段但需要 JSON 的端点，前端必须发送**空对象** `{}`（而非完全空的 body）。示例：`POST /auth/2fa/setup` 请求体为 `{}`。
- Admin 2FA 采用“两阶段启用”：
  - `POST /auth/2fa/setup`：仅生成密钥与二维码并写入缓存（返回 `setup_id`），**不会**写入数据库
  - `POST /auth/2fa/verify`：请求体必须包含 `setup_id` + `code`（OTP），后端从缓存取密钥校验；校验通过后写入数据库、生成恢复码并**销毁 AdminSession**（要求重新登录）
- 当 2FA 已启用且需要关闭/重置恢复码时，使用 `POST /auth/2fa/disable` 或 `POST /auth/2fa/recovery/reset`，请求体包含可选字段：`code`（OTP）或 `recovery_code`（恢复码），后端 `validate` 使用 `omitempty` 约束。

**非数值标识符资源（如 settings）：**

使用字符串标识符 `:key`，GET 和 PUT 路径对称：

```
GET  /settings/:key    → 按 key 查询
PUT  /settings/:key    → 按 key upsert
```

**Admin 用户路由签名（需注入 UserAuth 用于撤销会话）：**

```go
func NewUserRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, auth usecase.UserAuth, user usecase.User)
```

### Admin 路由总览

```
Auth（公开）
  POST   /auth/login
  POST   /auth/reset
  POST   /auth/logout

Auth（需鉴权）
  GET    /auth/profile
  PUT    /auth/profile
  PUT    /auth/password
  POST   /auth/2fa/setup
  POST   /auth/2fa/verify
  POST   /auth/2fa/disable
  POST   /auth/2fa/recovery/reset

Files
  GET    /files
  POST   /files/upload-url
  DELETE /files/*

Users
  GET    /users
  PUT    /users/:id/status

Content
  GET    /content/posts
  GET    /content/posts/:id
  POST   /content/posts
  PUT    /content/posts/:id
  DELETE /content/posts/:id
  GET    /content/categories
  POST   /content/categories
  PUT    /content/categories/:id
  DELETE /content/categories/:id
  GET    /content/tags
  POST   /content/tags
  PUT    /content/tags/:id
  DELETE /content/tags/:id
  POST   /content/generate-slug

Comments
  GET    /comments
  PUT    /comments/:id/status
  DELETE /comments/:id

Feedbacks
  GET    /feedbacks
  GET    /feedbacks/:id
  PUT    /feedbacks/:id/status
  DELETE /feedbacks/:id

Links
  GET    /links
  POST   /links
  PUT    /links/:id
  DELETE /links/:id

Settings
  GET    /settings
  GET    /settings/:key
  PUT    /settings/:key

Notifications
  POST   /notifications
```

**Admin.Auth Profile 响应：**

- GET /auth/profile 返回 AdminProfile，字段：
  - nickname
  - specialization
  - twofa_enabled（根据管理员 TwoFactorSecret 是否非空判定）

### V1（Public API）路由规范

V1 路由与 Admin 共享 RESTful 风格，但有以下区别：

**鉴权中间件分级：**

V1 根据接口性质使用三种鉴权层级：

| 层级 | 中间件 | 适用场景 |
|------|--------|----------|
| 公开 | 无中间件 | 分类、标签、链接、设置等无用户上下文的只读接口 |
| 可选鉴权 | `NewOptionalUserJWTMiddleware` | 文章列表/详情、评论列表（登录用户可获取 `like` 点赞状态） |
| 强制鉴权 | `NewUserJWTMiddleware` | 点赞、评论、个人资料等需要用户身份的写操作 |
（不再使用独立的“活跃用户”中间件。禁用用户的拦截策略见下文说明）

**路由分组与命名：**
- 同一资源前缀按 `Public → Optional → Auth` 顺序注册，避免前缀中间件误伤更具体的 public 子路径
- group 变量命名统一使用：`{domain}PublicGroup`、`{domain}OptionalGroup`、`{domain}AuthGroup`
- 当同一前缀下既有 public 路由又有强制鉴权 group（例如通知 SSE），public 路由必须注册在 auth group 之前

禁用用户拦截策略：
- refresh 会话定义：Redis `refresh_token:{userID}` 保存该用户“当前 refresh token”（字符串）；存在即视为会话有效
- `NewUserJWTMiddleware` 在验签通过后，读取 `refresh_token` Cookie，并调用 `UserAuth.ValidateSession(ctx, userID, refreshToken)` 校验 refresh 是否仍有效；无效则返回 `401 ErrorTokenInvalid`
- 刷新流程在 UseCase 层检查 `user.Status != "active"` 并返回 `ErrUserDisabled`，确保禁用用户无法续签刷新令牌
- 退出登录会撤销 refresh 会话（删除 Redis key），使 access token 立即失效
- 启用 Redis refreshStore 时：
  - 若 `refresh_token:{userID}` 存在，则必须与 Cookie 中 refresh token 一致，否则返回 `ErrTokenInvalid`
  - 若 `refresh_token:{userID}` 不存在（如 Redis 重启清空），回退查询 `refresh_token_blacklist`：未拉黑则视为有效并用该 refresh token 补写 Redis 当前值（自愈）
  - 若用户状态非 active 则优先返回 `ErrUserDisabled`
- 未启用 refreshStore（例如未配置 Redis）时：中间件仍会校验 refresh JWT 与用户状态，但不会执行“当前值一致性”校验

Admin 禁用触发的即时会话撤销：

- 当管理端将用户状态置为 `disabled` 时，应尽力撤销该用户的刷新令牌会话
- 撤销包含两步：写入 `refresh_token_blacklist`（保存刷新令牌哈希与过期时间），并删除 Redis 中该用户的当前刷新令牌键
- 该步骤在 Admin Handler 中调用 `User.UpdateStatus` 后再调用 `UserAuth.RevokeUserRefreshToken` 完成（撤销失败记录日志即可）

```go
contentPublicGroup := apiV1Group.Group("/content")
{
    contentPublicGroup.Get("/categories", r.listCategories)
    contentPublicGroup.Get("/tags", r.listTags)
}

contentOptionalGroup := apiV1Group.Group("/content", middleware.NewOptionalUserJWTMiddleware(signer, auth))
{
    contentOptionalGroup.Get("/posts", r.listPosts)
    contentOptionalGroup.Get("/posts/:slug", r.getPost)
}

contentAuthGroup := apiV1Group.Group("/content", middleware.NewUserJWTMiddleware(signer, auth))
{
    contentAuthGroup.Post("/posts/:id/likes", r.togglePostLike)
    contentAuthGroup.Delete("/posts/:id/likes", r.removePostLike)
}
```

通知 SSE（EventSource 不支持自定义 `Authorization` Header，使用 Query Token 鉴权）：

```go
notificationPublicGroup := apiV1Group.Group("/notifications")
{
    notificationPublicGroup.Get("/stream", r.streamNotifications)
}

notificationAuthGroup := apiV1Group.Group("/notifications", middleware.NewUserJWTMiddleware(signer, auth))
{
    notificationAuthGroup.Get("/", r.listNotifications)
    notificationAuthGroup.Get("/unread", r.getUnreadCount)
}
```

> **⚠️ 重要：禁止使用空前缀 Group 挂载鉴权中间件**
>
> `apiV1Group.Group("", middleware.NewUserJWTMiddleware(signer, auth))` 会导致 JWT 中间件泄漏到同级所有路由（包括无需认证的 `/settings` 等），
> 因为 Fiber 将空前缀 Group 的中间件应用到父级路径下的全部请求。
>
> **正确做法：** 始终使用明确的前缀（如 `/content`、`/comments`），或拆分为多个有前缀的 Group。

**子资源路由：**

点赞和评论作为文章的子资源，路径挂在 `/content/posts/:id/` 下：

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/content/posts/:id/likes` | 切换点赞（Toggle） |
| DELETE | `/content/posts/:id/likes` | 取消点赞 |
| GET | `/content/posts/:id/comments` | 文章评论列表 |
| POST | `/content/posts/:id/comments` | 提交评论 |

> 注意：文章详情使用 `:slug`（`GET /content/posts/:slug`），而子资源操作使用 `:id`（数值主键）。
> 两者不冲突，因为 Fiber 按路径段数区分（3 段 vs 4 段）。

**路由注册签名：**

V1 不依赖 session，鉴权通过 JWT `signer` 参数传递：

```go
// 无鉴权
func NewLinkRoutes(apiV1Group fiber.Router, l logger.Interface, link usecase.Link)

// 带鉴权
func NewAuthRoutes(apiV1Group fiber.Router, l logger.Interface, c usecase.Captcha, e usecase.Email, signer authUC.TokenSigner, auth usecase.UserAuth)
func NewUserRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, user usecase.User)
func NewContentRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, content usecase.Content)
func NewCommentRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, comment usecase.Comment)
func NewNotificationRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, notification usecase.Notification)
```

- 以上签名用于路由层注入依赖：凡是挂载了用户 JWT 中间件的 group，必须可拿到 `auth usecase` 以完成 refresh 会话校验

### V1 路由总览

```
Captcha
  GET    /captcha/generate

Email
  POST   /email/send-code

Auth（公开）
  POST   /auth/register
  POST   /auth/login
  POST   /auth/refresh
  POST   /auth/forgot

Auth（需鉴权）
  POST   /auth/logout

Files（公开）
  GET    /files/*

User（需鉴权）
  GET    /user/me
  PUT    /user/profile
  PUT    /user/password

Content（公开）
  GET    /content/categories
  GET    /content/tags

Content（可选鉴权）
  GET    /content/posts
  GET    /content/posts/:slug

Content（需鉴权）
  POST   /content/posts/:id/likes
  DELETE /content/posts/:id/likes

Comments（可选鉴权）
  GET    /content/posts/:id/comments

Comments（需鉴权）
  POST   /content/posts/:id/comments
  POST   /comments/:id/likes
  DELETE /comments/:id/likes
  DELETE /comments/:id

Feedbacks
  POST   /feedbacks

Links
  GET    /links

Settings
  GET    /settings

Notifications（需鉴权）
  GET    /notifications
  GET    /notifications/unread
  PUT    /notifications/:id/read
  PUT    /notifications/read-all
  DELETE /notifications/:id

Notifications（SSE，Query Token 鉴权）
  GET    /notifications/stream
```

---

## 十三、Repo 实现规范

### 基本结构

- 结构体**仅持有** `query *query.Query`（GORM Gen 生成），**不持有** `db *gorm.DB`
- 构造函数接收 `*gorm.DB`，通过 `query.Use(db)` 创建 Gen 实例，返回接口类型
- 所有数据库操作**统一使用 Gen 类型安全 API**，不直接使用原始 GORM

```go
type postRepo struct {
    query *query.Query
}

func NewPostRepo(db *gorm.DB) repo.PostRepo {
    return &postRepo{query: query.Use(db)}
}
```

### Model ↔ Entity 映射

- 提供 `toEntity{Name}` 和 `toModel{Name}` 私有函数完成双向映射
- 注意处理 `model`（GORM 生成，值类型）与 `entity`（领域模型，可选字段用指针）之间的类型差异

```go
func toEntityPost(mp *model.Post) *entity.Post {
    p := &entity.Post{
        ID:    mp.ID,
        Title: mp.Title,
        // ...
    }
    if mp.Excerpt != "" {
        p.Excerpt = &mp.Excerpt   // model string → entity *string
    }
    return p
}
```

### Gen 类型安全查询

所有查询通过 `r.query.{Table}` 获取表对象，使用 `.WithContext(ctx)` 进入查询链：

```go
func (r *postRepo) GetByID(ctx context.Context, id int64) (*entity.Post, error) {
    p := r.query.Post
    mp, err := p.WithContext(ctx).Where(p.ID.Eq(id)).First()
    if err != nil {
        return nil, err
    }
    return toEntityPost(mp), nil
}
```

### 动态条件构建

使用局部变量 `do` 链式追加可选条件：

```go
p := r.query.Post
do := p.WithContext(ctx)

if status != nil {
    do = do.Where(p.Status.Eq(*status))
}
if isFeatured != nil {
    do = do.Where(p.IsFeatured.Is(*isFeatured))
}

total, err := do.Count()
rows, err := do.Order(p.CreatedAt.Desc()).Offset(offset).Limit(limit).Find()
```

### 原始 SQL 表达式（field.NewUnsafeFieldRaw）

Gen 类型 API 无法表达的 PostgreSQL 特有语法（如 `ILIKE`），使用 `field.NewUnsafeFieldRaw()` 创建原始字段表达式，传入 `Where()` / `Order()`：

> **注意**：
> - **不要**使用 `gen.Cond(clause.Expr{...})`，gen v0.3.27 的 `exprToCondition` 仅支持 JSON 相关表达式，`clause.Expr` 会导致运行时错误 `unsupported Expression clause.Expr to converted to Condition`。
> - **不要**使用 `Clauses(clause.Where{...})` 或 `Clauses(clause.OrderBy{...})`，gen 的安全检查会 ban 掉 WHERE / ORDER BY 等子句，导致运行时错误 `clause WHERE is banned`。
> - `field.NewUnsafeFieldRaw` 返回 `field.Field`（实现 `field.Expr` / `gen.Condition` 接口），可安全用于 `Where()` 和 `Order()`。

```go
import "gorm.io/gen/field"

// 单字段 ILIKE 搜索（分类名）
do = do.Where(field.NewUnsafeFieldRaw("name ILIKE ?", "%"+*keyword+"%"))

// 多字段 ILIKE 搜索（用户名 + 邮箱）
do = do.Where(field.NewUnsafeFieldRaw("name ILIKE ? OR email ILIKE ?", "%"+*keyword+"%", "%"+*keyword+"%"))

// 多字段 ILIKE 搜索（文章标题 + 摘要 + 正文）
kw := "%" + *keyword + "%"
do = do.Where(field.NewUnsafeFieldRaw("title ILIKE ? OR COALESCE(excerpt, '') ILIKE ? OR content ILIKE ?", kw, kw, kw))
```

#### 关键词搜索策略

| 搜索层 | 方式 | 说明 |
|--------|------|------|
| WHERE 过滤 | `ILIKE '%keyword%'` | 精确子串匹配，不会误匹配 |
| ORDER BY 排序 | `similarity(...) DESC` | pg_trgm 相似度排序，匹配度高的排前面 |

> **不要**在 WHERE 中使用 pg_trgm 的 `%` 运算符。`%` 基于三元组（trigram）计算模糊相似度，默认阈值仅 0.3，对短关键词（如 1~3 字符）区分度极差，容易产生大量误匹配。`%` 适合 ORDER BY 排序场景，不适合 WHERE 精确过滤。

### 排序

**静态排序** — 使用 Gen 字段的 `.Desc()` / `.Asc()` 方法：

```go
do = do.Order(p.CreatedAt.Desc())
do = do.Order(p.IsFeatured.Desc(), p.PublishedAt.Desc())   // 多字段排序
```

**动态排序** — 使用 `GetFieldByName` 按字段名动态排序：

```go
orderField, ok := p.GetFieldByName(*sortBy)
if ok {
    if order != nil && strings.EqualFold(*order, "asc") {
        do = do.Order(orderField)
    } else {
        do = do.Order(orderField.Desc())
    }
}
```

**原始 SQL 排序**（如 pg_trgm 相似度）— 使用 `field.NewUnsafeFieldRaw().Desc()`：

```go
do = do.Order(field.NewUnsafeFieldRaw("similarity(title || ' ' || COALESCE(excerpt, '') || ' ' || content, ?)", *keyword).Desc())
```

### 关联子查询

使用 `gen.Exists()` + 关联子查询替代 `WHERE id IN (SELECT ...)`：

```go
// 按 tag 过滤文章：WHERE EXISTS (SELECT 1 FROM post_tags WHERE tag_id = ? AND post_id = posts.id)
pt := r.query.PostTag
subQ := pt.WithContext(ctx).Where(pt.TagID.Eq(int64(*tagID)), pt.PostID.EqCol(p.ID))
do = do.Where(gen.Exists(subQ))
```

### 事务

使用 Gen 原生事务 `r.query.Transaction()`，回调中通过 `tx` 访问所有表：

```go
err := r.query.Transaction(func(tx *query.Query) error {
    pl := tx.PostLike
    p := tx.Post

    // 创建记录
    if err := pl.WithContext(ctx).Create(&model.PostLike{...}); err != nil {
        return err
    }
    // 原子递增
    if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Add(1)); err != nil {
        return err
    }
    return nil
})
```

### 原子更新（UpdateSimple）

使用字段的 `.Add()` / `.Sub()` 方法进行原子递增/递减，避免竞态条件：

```go
// likes = likes + 1
p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Add(1))

// likes = likes - 1
p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Sub(1))
```

### Upsert（ON CONFLICT）

使用 `Clauses(clause.OnConflict{...})` 实现 PostgreSQL 的 `INSERT ... ON CONFLICT DO UPDATE`：

```go
s.WithContext(ctx).Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "setting_key"}},
    DoUpdates: clause.AssignmentColumns([]string{"setting_value", "description", "updated_at"}),
}).Create(ms)
```

### CRUD 方法命名

| 操作 | 方法名 | 返回值 |
|------|--------|--------|
| 创建 | `Create` | `(int64, error)` |
| 查询单条 | `GetByID` / `GetBySlug` / `GetByEmail` | `(*entity.T, error)` |
| 查询列表（分页） | `List` | `([]*entity.T, int64, error)` |
| 查询列表（全量） | `ListAll` / `ListAllPublic` | `([]*entity.T, error)` |
| 更新 | `Update` / `UpdateStatus` | `error` |
| 删除 | `Delete` | `error` |
| 关联操作 | `SetTags` / `Toggle` / `Remove` | 按需 |
| 单项存在性检查 | `HasLiked` | `(bool, error)`（检查用户是否已点赞指定资源） |
| 批量解绑（按用途） | `ClearResourceIDByResourceAndUsage` | `error`（按 usage 解绑指定 resource_id 的文件） |

### Gen API 使用优先级

| 优先级 | 方式 | 适用场景 |
|--------|------|----------|
| 1（首选） | Gen 类型安全 API | 等值/范围/IN/NULL 条件、标准排序、分页、CRUD |
| 2 | `field.NewUnsafeFieldRaw()` 用于 `Where()` | ILIKE 等 PostgreSQL 特有 WHERE 条件 |
| 3 | `field.NewUnsafeFieldRaw().Desc()` 用于 `Order()` | similarity() 等原始 SQL 排序表达式 |
| 4 | `Clauses(clause.OnConflict{})` | Upsert 操作 |

**禁止**直接使用 `r.db` 或 `gorm.DB` 进行查询。所有数据访问必须经过 `r.query.{Table}` 的 Gen API。

---

## 十四、UseCase 实现规范

- 未导出结构体 `useCase`，持有 Repo 接口作为依赖
- 构造函数 `New(...)` 返回对应 UseCase 接口
- 每个包定义自己的哨兵错误
- 辅助接口/服务（如 `ReadTimeCalculator`、`TOTP`、`Encryptor`）在同包中定义

```go
type useCase struct {
    translationWebAPI  repo.TranslationWebAPI
    llmWebAPI          repo.LLMWebAPI
    admins             repo.AdminRepo
    posts              repo.PostRepo
    tags               repo.TagRepo
    categories         repo.CategoryRepo
    postLikes          repo.PostLikeRepo
    files              repo.FileRepo
    postViews          repo.PostViewRepo
    readTimeCalculator ReadTimeCalculator
}

func New(
    translationWebAPI repo.TranslationWebAPI,
    llmWebAPI repo.LLMWebAPI,
    admins repo.AdminRepo,
    posts repo.PostRepo,
    tags repo.TagRepo,
    categories repo.CategoryRepo,
    postLikes repo.PostLikeRepo,
    files repo.FileRepo,
    postViews repo.PostViewRepo,
    readTimeCalculator ReadTimeCalculator,
) usecase.Content {
    return &useCase{
        translationWebAPI:  translationWebAPI,
        llmWebAPI:          llmWebAPI,
        admins:             admins,
        posts:              posts,
        tags:               tags,
        categories:         categories,
        postLikes:          postLikes,
        files:              files,
        postViews:          postViews,
        readTimeCalculator: readTimeCalculator,
    }
}
```

---

## 十五、Pkg 封装规范

- 使用 **Functional Options** 模式配置：`New(logger, ...Option)`
- 包级默认常量以下划线前缀：`_defaultAddress`、`_defaultReadTimeout`
- 导出 `Interface` 用于抽象，具体实现未导出

```go
const (
    _defaultAddress      = ":8080"
    _defaultReadTimeout  = 5 * time.Second
    _defaultWriteTimeout = 5 * time.Second
)

func New(l logger.Interface, opts ...Option) *Server {
    s := &Server{ address: _defaultAddress, ... }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

---

## 十六、配置规范

- 使用 Viper + `mapstructure` tag 加载 `config.yaml`
- 配置结构体使用 `type ( ... )` 分组声明
- 子配置按服务拆分：`Postgres`、`Redis`、`MinIO`、`JWT`、`SMTP` 等

```go
type (
    Config struct {
        App      App      `mapstructure:"app"`
        Log      Log      `mapstructure:"log"`
        HTTP     HTTP     `mapstructure:"http"`
        Postgres Postgres `mapstructure:"postgres"`
        Redis    Redis    `mapstructure:"redis"`
        // ...
    }

    App struct {
        Name    string `mapstructure:"name"`
        Version string `mapstructure:"version"`
    }
)
```

---

## 十七、依赖注入（Wire）

使用 **Google Wire** 进行编译期依赖注入，所有组装逻辑集中在 `internal/app/wire.go`。

### 文件职责

| 文件 | 说明 |
|------|------|
| `wire.go` | Provider 函数定义 + ProviderSet + Injector（带 `//go:build wireinject` 标签） |
| `wire_gen.go` | 在 `internal/app/` 目录下执行 `wire` 命令自动生成（**禁止手动编辑**） |
| `app.go` | 应用生命周期（启动、信号监听、优雅关闭） |

### App 结构体

`App` 持有应用运行所需的核心依赖，`AppInfo` 使用值类型（非指针）：

```go
type App struct {
    Info       AppInfo
    Logger     logger.Interface
    Postgres   *postgres.Postgres
    Redis      *redis.Redis
    HTTPServer *httpserver.Server
}

type AppInfo struct {
    Name    string
    Version string
}
```

### Provider 函数分类与命名

按层级组织，每类使用 `// ─── 分类标题 ───` 分隔符：

| 分类 | 命名规则 | 参数规则 | 示例 |
|------|----------|----------|------|
| Infrastructure | `New{Component}` | 仅接收 `cfg *config.Config` | `NewLogger`、`NewPostgres`、`NewRedis`、`NewMinioClient` |
| Repo: Persistence | `New{Entity}Repo` | 接收 `pg *postgres.Postgres`，内部提取 `pg.DB` | `NewAdminRepo`、`NewPostRepo`、`NewFileRepo` |
| Repo: ViewBuffer | `New{Entity}Repo` | 接收 `pg` + `l`，返回 `(repo.Interface, func())` | `NewPostViewRepo`（返回 cleanup 用于优雅关闭） |
| Repo: Cache | `New{Purpose}Store` | 接收 `r *redis.Redis` | `NewCaptchaStore`、`NewRefreshTokenStore` |
| Repo: Storage/Messaging/WebAPI | `New{Purpose}` | 按需接收 `cfg` 或具体依赖 | `NewObjectStore`、`NewEmailSender`、`NewLLMWebAPI` |
| UseCase | `New{Domain}UseCase` | 仅接收实际需要的依赖，**不传入未使用的参数** | `NewContentUseCase`、`NewUserUseCase` |
| HTTP | `SetupHTTPServer` | 接收 `cfg`、`l` 和所有 UseCase 接口 | — |

### Provider 函数参数命名

**禁止传入未使用的参数**（如 `_ *config.Config`、`_ logger.Interface`），只声明函数实际需要的依赖。

参数命名约定：

| 类型 | 变量名 | 说明 |
|------|--------|------|
| `*config.Config` | `cfg` | 全局统一 |
| `logger.Interface` | `l` | 全局统一 |
| `*postgres.Postgres` | `pg` | Repo 构造器参数 |
| `*redis.Redis` | `r` | Cache 构造器参数 |
| `*minioSDK.Client` | `cli` | Storage 构造器参数 |
| `repo.XxxRepo` | `xxxRepo` | UseCase 构造器中的 Repo 参数，使用 `camelCase` 全称 |
| `repo.XxxStore` | `xxxStore` | 如 `refreshStore`、`codeStore` |
| `repo.XxxWebAPI` | 描述性名称 | 如 `translationAPI`、`llmAPI` |
| `usecase.Xxx` | `xxxUC` | SetupHTTPServer 中的 UseCase 参数（`auth` 除外） |
| `authuser.TokenSigner` | `signer` | JWT 签名器 |

### Provider 函数模式

**简单转发型** — Repo 构造器只做类型适配：

```go
func NewPostRepo(pg *postgres.Postgres) repo.PostRepo {
    return persistence.NewPostRepo(pg.DB)
}
```

**配置提取型** — 从 `cfg` 提取参数传递给底层构造器：

```go
func NewAdminAuthUseCase(cfg *config.Config, adminRepo repo.AdminRepo, twoFASetupStore repo.AdminTwoFASetupStore) usecase.AdminAuth {
    totpCfg := authadmin.TOTPConfig{QRWidth: cfg.TwoFA.QRWidth, QRHeight: cfg.TwoFA.QRHeight}
    enc := authadmin.NewEncryptorFromSecret(cfg.TwoFA.EncryptionKey)
    return authadmin.New(adminRepo, twoFASetupStore, cfg.App.Name, authadmin.NewTOTPProviderWithConfig(totpCfg), enc)
}
```

**内联辅助依赖型** — 将内部辅助组件（如 `ReadTimeCalculator`）直接在 Provider 内创建，不暴露为独立 Provider：

```go
func NewContentUseCase(translationAPI repo.TranslationWebAPI, llmAPI repo.LLMWebAPI, adminRepo repo.AdminRepo, postRepo repo.PostRepo, tagRepo repo.TagRepo, categoryRepo repo.CategoryRepo, postLikeRepo repo.PostLikeRepo, fileRepo repo.FileRepo, postViewRepo repo.PostViewRepo) usecase.Content {
    return content.New(translationAPI, llmAPI, adminRepo, postRepo, tagRepo, categoryRepo, postLikeRepo, fileRepo, postViewRepo, content.NewCalculator())
}
```

### ProviderSet 组织

使用单一扁平 `ProviderSet`，按层级分块并用注释分隔：

```go
var ProviderSet = wire.NewSet(
    // App
    NewAppInfo, NewLogger, NewApp,
    // Infrastructure
    NewPostgres, NewRedis, NewMinioClient,
    // Repo: Persistence
    NewAdminRepo, NewUserRepo, NewPostRepo, NewFileRepo, ...
    // Repo: ViewBuffer
    NewPostViewRepo,
    // Repo: Cache
    NewCaptchaStore, NewEmailCodeStore, NewRefreshTokenStore, NewAdminTwoFASetupStore,
    // Repo: Storage
    NewObjectStore,
    // Repo: Messaging
    NewEmailSender,
    // Repo: WebAPI
    NewTranslationWebAPI, NewLLMWebAPI,
    // UseCase: Auth
    NewTokenSigner, NewAdminAuthUseCase, NewUserAuthUseCase, NewAuthUseCase,
    // UseCase: Captcha ~ Setting
    ...
    // HTTP
    SetupHTTPServer,
)
```

### Injector 函数

```go
func InitializeApp(cfg *config.Config) (*App, func(), error) {
    wire.Build(ProviderSet)
    return nil, nil, nil
}
```

- 入参：`*config.Config`（由 `cmd/app/main.go` 加载并传入）
- 返回：`*App` + `cleanup func()` + `error`
- cleanup 由 Wire 自动组合所有带 cleanup 的 Provider（如 Postgres、Redis）

### import 别名约定

| 别名 | 包路径 | 说明 |
|------|--------|------|
| `httpctrl` | `internal/controller/http` | 避免与 `net/http` 冲突 |
| `minioPkg` | `pkg/minio` | 避免与 `minio-go/v7` 冲突 |
| `minioSDK` | `github.com/minio/minio-go/v7` | MinIO SDK |
| `reponotif` | `internal/repo/notification` | 避免与 `usecase/notification` 冲突 |
| `authfacade` | `internal/usecase/auth` | Auth 门面 |
| `authadmin` | `internal/usecase/auth/admin` | Admin 认证 |
| `authuser` | `internal/usecase/auth/user` | User 认证 |
| `useruc` | `internal/usecase/user` | 避免与 `authuser` 冲突 |
| 其他 usecase | 无别名 | `captcha`、`comment`、`content` 等直接使用包名 |
 
## 十八、接口文档（Swagger）规范
 
### 集成方式
 
 - 使用 Fiber v3 + gofiber/contrib/v3/swaggo 集成 Swagger UI
 - 路由：当配置开启时，GET `/swagger/*` 提供文档页面
 - 生成文档：在项目根目录执行
 
 ```bash
 swag init -g cmd/app/main.go -o docs
 ```
 
 - 配置开关：`config.yaml` 中 `swagger.enabled`（true 开启 / false 关闭）
 
### 分页响应的文档类型
 
 - 为 `Page[T]` 定义具名类型别名，便于文档展示与注解引用
 - 位置：
   - `internal/controller/http/admin/response/pages.go`
   - `internal/controller/http/v1/response/pages.go`
 - 示例：
 
 ```go
 // admin
 type CommentDetailPage = shared.Page[CommentDetail]
 type TagDetailPage     = shared.Page[TagDetail]
 
 // v1
 type PostSummaryPage        = shared.Page[PostSummary]
 type NotificationDetailPage = shared.Page[NotificationDetail]
 ```
 
 - 注解引用规范：在 `@Success` 或嵌套 `Envelope` 的 `data` 字段中使用具名别名
 - 推荐写法：
 
 ```go
 // @Success 200 {object} response.Envelope{data=response.CommentDetailPage}
 ```
 
 说明：类型别名仅影响文档生成与类型可读性，运行时结构与序列化不变

---

## 十九、部署与运行环境约定

- 基础镜像与系统包：镜像需安装 `tzdata` 与 `ca-certificates`，确保时区与证书链正常；推荐使用 `alpine:3.20`。
- 运行用户：以非特权用户运行（如 `nobody`），避免以 `root` 运行服务。
- 时区一致性：数据库连接与应用配置的时区需一致（如 `Asia/Shanghai`），避免时间计算与显示偏差。

## 二十、数据库迁移执行规范

- 迁移程序入口：`cmd/migrate/main.go`，支持 `up/down/steps/force/version/drop` 等操作。
- 示例：`./migrate -dir ./migrations -action up`；容器场景下需挂载配置与迁移目录。
- 约定：应用启动不自动执行迁移；部署流程中显式执行迁移步骤，保证版本可控。

## 二十一、容器网络与主机名约定

- 自定义网络：创建专用网络（如 `nimbus-net`），服务间通过网络名互通。
- 主机名策略：在 `config.yaml` 中为 Postgres/Redis/MinIO 配置服务名（如 `postgres`、`redis`、`minio-server`），避免使用 `localhost`。
- 反向代理：前端通过同域 `/api/*` 访问后端，代理层转发至后端容器服务。

## 二十二、密钥生成工具

- 位置：`cmd/keys/main.go`
- 功能：生成 JWT `access_secret`、`refresh_secret` 与 TwoFA `encryption_key`。
- 使用：`go run ./cmd/keys -yaml` 输出 YAML 片段，直接粘贴到 `config.yaml`。
