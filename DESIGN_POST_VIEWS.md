# 文章阅读量统计 — 设计文档

## 一、背景与现状

### 1.1 数据库层

已有两套存储结构：

**`posts` 表** — 冗余计数字段：

```sql
views INTEGER DEFAULT 0 NOT NULL
```

**`post_views` 表** — 访问明细：

```sql
CREATE TABLE IF NOT EXISTS post_views (
    id         BIGSERIAL PRIMARY KEY,
    post_id    BIGINT    NOT NULL,
    ip_address INET      NOT NULL,
    user_agent TEXT,
    referer    VARCHAR(500),
    viewed_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE
);
```

Gen 已生成 `model.PostView` 和 `query.PostView`，业务层已通过 `repo/viewbuffer` 落地使用；`posts.views` 由缓冲刷盘增量更新（存在秒级延迟）。

### 1.2 参考模式

| 维度 | 点赞（PostLike） | 阅读量（PostView） |
|------|-----------------|-------------------|
| 触发频率 | 低（主动操作） | 高（每次访问） |
| 精确性要求 | 高（不可重复） | 容忍秒级延迟 |
| 去重逻辑 | 用户级唯一（Toggle） | IP + 时间窗口 |
| 性能敏感度 | 低 | 高（热门文章并发） |

点赞使用同步事务模式。阅读量由于高频触发、低精确要求，需要异步缓冲。

---

## 二、设计目标

1. **零请求开销**：文章详情请求路径不增加任何数据库操作
2. **防刷去重**：同一 IP + 同一文章在时间窗口内只计一次
3. **数据可分析**：明细记录落库（`post_views`），支持后续分析
4. **严格分层**：缓冲策略是 Repo 层实现细节，UseCase / Controller 完全无感知
5. **优雅关闭**：通过 Wire cleanup 链自动触发刷盘

---

## 三、架构分层

```
┌─────────────────────────────────────────────────┐
│  Controller 层                                   │
│                                                 │
│  getPost handler                                │
│      ↓ 调用 UseCase                              │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│  UseCase 层（纯业务逻辑）                          │
│                                                 │
│  content.RecordView(ctx, postID, ip, ua, ref)   │
│      ↓ 调用 PostViewRepo.Record()               │
│                                                 │
│  ※ 不知道也不关心 Record 是同步还是异步             │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│  Repo 层（数据访问 — 缓冲实现）                     │
│                                                 │
│  PostViewRepo 接口 ← bufferedPostViewRepo 实现   │
│      │                                          │
│      ├── Record(): 内存去重 → channel 投递        │
│      │                                          │
│      └── 后台 goroutine（flush loop）             │
│              ├── 每 30s 或 buffer 满时            │
│              ├── batch INSERT post_views         │
│              └── UPDATE posts.views += delta     │
│                                                 │
│  ※ 缓冲、去重、批量写入全部在 Repo 层内部完成       │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
                   PostgreSQL
```

**关键决策：缓冲逻辑封装在 Repo 实现中。**

参照已有的 `repo/notification/` 模式（组合 `NotificationRepo` + `ssehub.Hub`），缓冲写入是数据访问策略，属于 Repo 层职责。UseCase 只看到 `PostViewRepo.Record()` 这一个接口方法。

---

## 四、各层详细设计

### 4.1 Entity 层

定义 [post_view.go](internal/entity/post_view.go#L1-L12)：

```go
package entity

import "time"

type PostView struct {
    ID        int64
    PostID    int64
    IPAddress string
    UserAgent *string
    Referer   *string
    ViewedAt  time.Time
}
```

纯结构体，无 tag，遵循 Entity 规范。

### 4.2 Repo 层

#### 4.2.1 接口定义（[contracts.go](internal/repo/contracts.go#L121-L130)）

```go
PostViewRepo interface {
    Record(ctx context.Context, pv entity.PostView) error
}
```

单方法接口。调用方（UseCase）只知道"记录一次浏览"，不感知底层是同步写还是异步缓冲。

#### 4.2.2 缓冲实现（[post_view_buffered.go](internal/repo/viewbuffer/post_view_buffered.go#L1-L166)）

目录选择 `repo/viewbuffer/`，与 `repo/notification/`、`repo/cache/`、`repo/storage/` 同级，表示一种特定的数据访问策略。

```go
package viewbuffer

import (
    "context"
    "time"

    "github.com/scc749/nimbus-blog-api/internal/entity"
    "github.com/scc749/nimbus-blog-api/internal/repo"
    "github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
    "github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
    "github.com/scc749/nimbus-blog-api/pkg/logger"
    "gorm.io/gorm"
)

const (
    _chanSize      = 4096
    _flushInterval = 30 * time.Second
    _dedupeWindow  = 30 * time.Minute
    _maxBatchSize  = 500
    _batchInsert   = 100
)

var _ repo.PostViewRepo = (*bufferedPostViewRepo)(nil)

type bufferedPostViewRepo struct {
    query  *query.Query
    logger logger.Interface
    ch     chan viewEvent
    cancel context.CancelFunc
    done   chan struct{}
}

type viewEvent struct {
    postID    int64
    ip        string
    userAgent *string
    referer   *string
    timestamp time.Time
}

type dedupeKey struct {
    postID int64
    ip     string
}
```

**构造函数**（返回 cleanup，接入 Wire 生命周期）：

```go
func New(db *gorm.DB, l logger.Interface) (repo.PostViewRepo, func()) {
    ctx, cancel := context.WithCancel(context.Background())
    r := &bufferedPostViewRepo{
        query:  query.Use(db),
        logger: l,
        ch:     make(chan viewEvent, _chanSize),
        cancel: cancel,
        done:   make(chan struct{}),
    }
    go r.flushLoop(ctx)
    cleanup := func() {
        r.cancel()
        <-r.done
    }
    return r, cleanup
}
```

**Record 方法**（非阻塞投递）：

```go
func (r *bufferedPostViewRepo) Record(_ context.Context, pv entity.PostView) error {
    select {
    case r.ch <- viewEvent{
        postID:    pv.PostID,
        ip:        pv.IPAddress,
        userAgent: pv.UserAgent,
        referer:   pv.Referer,
        timestamp: pv.ViewedAt,
    }:
    default:
    }
    return nil
}
```

> `Record` 接受 `context.Context` 以符合 Repo 接口规范，但实际不使用（纯内存操作）。返回 `nil`，统计失败不应影响业务。

**flushLoop**（后台协程）：

```go
func (r *bufferedPostViewRepo) flushLoop(ctx context.Context) {
    defer close(r.done)

    ticker := time.NewTicker(_flushInterval)
    defer ticker.Stop()

    seen := make(map[dedupeKey]time.Time)
    var pending []viewEvent
    deltas := make(map[int64]int32)

    flush := func() {
        if len(pending) == 0 {
            return
        }
        r.writeBatch(pending, deltas)
        pending = pending[:0]
        for k := range deltas {
            delete(deltas, k)
        }
    }

    for {
        select {
        case ev := <-r.ch:
            key := dedupeKey{postID: ev.postID, ip: ev.ip}
            if last, ok := seen[key]; ok && time.Since(last) < _dedupeWindow {
                continue
            }
            seen[key] = ev.timestamp
            pending = append(pending, ev)
            deltas[ev.postID]++
            if len(pending) >= _maxBatchSize {
                flush()
            }

        case <-ticker.C:
            flush()
            now := time.Now()
            for k, t := range seen {
                if now.Sub(t) > _dedupeWindow {
                    delete(seen, k)
                }
            }

        case <-ctx.Done():
        drain:
            for {
                select {
                case ev := <-r.ch:
                    key := dedupeKey{postID: ev.postID, ip: ev.ip}
                    if last, ok := seen[key]; ok && time.Since(last) < _dedupeWindow {
                        continue
                    }
                    seen[key] = ev.timestamp
                    pending = append(pending, ev)
                    deltas[ev.postID]++
                default:
                    break drain
                }
            }
            flush()
            return
        }
    }
}
```

**writeBatch**（实际 DB 写入）：

```go
func (r *bufferedPostViewRepo) writeBatch(events []viewEvent, deltas map[int64]int32) {
    ctx := context.Background()

    models := make([]*model.PostView, len(events))
    for i, ev := range events {
        models[i] = &model.PostView{
            PostID:    ev.postID,
            IPAddress: ev.ip,
            UserAgent: ev.userAgent,
            Referer:   ev.referer,
            ViewedAt:  ev.timestamp,
        }
    }
    if err := r.query.PostView.WithContext(ctx).CreateInBatches(models, _batchInsert); err != nil {
        r.logger.Error(err, "viewbuffer - writeBatch - CreateInBatches")
    }

    p := r.query.Post
    for postID, delta := range deltas {
        if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Views.Add(delta)); err != nil {
            r.logger.Error(err, "viewbuffer - writeBatch - IncrementViews")
        }
    }
}
```

> 直接使用 `r.query.Post` 更新 `posts.views`，与 `PostLikeRepo` 在事务中同时操作 `PostLike` 和 `Post` 表的模式一致。

### 4.3 UseCase 层

#### 4.3.1 接口新增（[contracts.go](internal/usecase/contracts.go#L79-L104)）

`Content` 接口 Public 部分新增一个方法：

```go
type Content interface {
    // ...existing methods...

    // Public
    RecordView(ctx context.Context, postID int64, ip, userAgent, referer string)
}
```

> 无返回值 error。统计是尽力而为（best-effort），不应让调用方处理统计失败。这与 `Notifier.Send` 返回 error 不同，因为通知失败需要上层感知，但浏览统计失败可以静默。

#### 4.3.2 实现（[content.go:RecordView](internal/usecase/content/content.go#L444-L459)）

```go
type useCase struct {
    // ...existing fields...
    postViews repo.PostViewRepo  // 新增
}

func (u *useCase) RecordView(ctx context.Context, postID int64, ip, userAgent, referer string) {
    var ua, ref *string
    if userAgent != "" {
        ua = &userAgent
    }
    if referer != "" {
        ref = &referer
    }
    _ = u.postViews.Record(ctx, entity.PostView{
        PostID:    postID,
        IPAddress: ip,
        UserAgent: ua,
        Referer:   ref,
        ViewedAt:  time.Now(),
    })
}
```

UseCase 职责仅限于：构造 entity → 调用 Repo。不知道底层是否缓冲。

### 4.4 Controller 层

#### 触发点（[content.go:getPost](internal/controller/http/v1/content.go#L104-L174)）

```go
func (r *V1) getPost(ctx fiber.Ctx) error {
    // ...existing logic: GetPublicPostBySlug → build response...

    // 记录浏览（fire-and-forget）
    r.content.RecordView(ctx.Context(), post.ID, ctx.IP(), ctx.Get("User-Agent"), ctx.Get("Referer"))

    // ...return response（post.Views 来自 DB 字段，最多延迟 30 秒）...
}
```

不新增路由，不等待结果，不影响响应延迟。

### 4.5 DI 层（[wire.go](internal/app/wire.go#L180-L430)）

```go
// ViewBuffer Repo（浏览量缓冲）。
func NewPostViewRepo(pg *postgres.Postgres, l logger.Interface) (repo.PostViewRepo, func()) {
	return viewbuffer.New(pg.DB, l)
}
```

返回 `(repo.PostViewRepo, func())`，Wire 自动将 `func()` 加入 cleanup 链。

`NewContentUseCase` 新增参数：

```go
// Content UseCase。
func NewContentUseCase(translationAPI repo.TranslationWebAPI, llmAPI repo.LLMWebAPI, adminRepo repo.AdminRepo, postRepo repo.PostRepo, tagRepo repo.TagRepo, categoryRepo repo.CategoryRepo, postLikeRepo repo.PostLikeRepo, fileRepo repo.FileRepo, postViewRepo repo.PostViewRepo) usecase.Content {
	return content.New(translationAPI, llmAPI, adminRepo, postRepo, tagRepo, categoryRepo, postLikeRepo, fileRepo, postViewRepo, content.NewCalculator())
}
```

ProviderSet 新增：

```go
// ProviderSet Wire ProviderSet。
var ProviderSet = wire.NewSet(
	// App 应用容器。
	NewAppInfo,
	NewLogger,
	NewApp,
	// Infrastructure 基础设施。
	NewPostgres,
	NewRedis,
	NewMinioClient,
	// RepoPersistence Postgres Repo。
	NewAdminRepo,
	NewUserRepo,
	NewPostRepo,
	NewTagRepo,
	NewCategoryRepo,
	NewCommentRepo,
	NewPostLikeRepo,
	NewCommentLikeRepo,
	NewFeedbackRepo,
	NewLinkRepo,
	NewSiteSettingRepo,
	NewFileRepo,
	NewNotificationRepo,
	NewRefreshTokenBlacklistRepo,
	// RepoViewBuffer 浏览量缓冲。
	NewPostViewRepo,
	// RepoCache Redis Repo。
	NewCaptchaStore,
	NewEmailCodeStore,
	NewRefreshTokenStore,
	NewAdminTwoFASetupStore,
	// RepoStorage MinIO Repo。
	NewObjectStore,
	// RepoMessaging SMTP Repo。
	NewEmailSender,
	// RepoWebAPI 外部 API。
	NewTranslationWebAPI,
	NewLLMWebAPI,
	// UseCaseAuth 认证用例。
	NewTokenSigner,
	NewAdminAuthUseCase,
	NewUserAuthUseCase,
	NewAuthUseCase,
	// UseCaseCaptcha 验证码用例。
	NewCaptchaGenerator,
	NewCaptchaUseCase,
	// UseCaseEmail 邮件用例。
	NewEmailUseCase,
	// UseCaseFile 文件用例。
	NewFileUseCase,
	// UseCaseUser 用户用例。
	NewUserUseCase,
	// UseCaseContent 内容用例。
	NewContentUseCase,
	// UseCaseComment 评论用例。
	NewCommentUseCase,
	// UseCaseFeedback 反馈用例。
	NewFeedbackUseCase,
	// UseCaseLink 友链用例。
	NewLinkUseCase,
	// UseCaseSetting 设置用例。
	NewSettingUseCase,
	// Pkg 基础包。
	NewSSEHub,
	// RepoNotifier 通知推送。
	NewNotifier,
	// UseCaseNotification 通知用例。
	NewNotificationUseCase,
	// HTTP HTTP Server。
	SetupHTTPServer,
)
```

### 4.6 应用生命周期

**`app.go` 无需任何改动。**

Wire 的 cleanup 机制会自动处理：

```
进程收到 SIGTERM
    → app.go: defer cleanup()
        → Wire cleanup chain 逆序执行
            → NewPostViewRepo 的 cleanup: cancel() + <-done（触发刷盘并等待完成）
            → NewRedis 的 cleanup: rdb.Close()
            → NewPostgres 的 cleanup: pg.Close()
```

> 关键：`NewPostViewRepo` 在 ProviderSet 中排在 `NewPostgres` 之后注册，所以 cleanup 时 PostViewRepo 先执行刷盘，Postgres 后关闭，保证刷盘时数据库连接仍然可用。

---

## 五、去重策略

### 5.1 规则

内存 `map[dedupeKey]time.Time`，`dedupeKey = {postID, ip}`。

同一 `{postID, IP}` 在 30 分钟窗口内只计一次。超过窗口后重新计数。

### 5.2 定期清理

每次 flush tick 时，遍历 `seen` map，删除距今超过 `_dedupeWindow` 的条目。

### 5.3 内存估算

| 场景 | 独立 IP | 文章数 | map 条目上限 | 内存 |
|------|--------|--------|------------|------|
| 小型博客 | ~1K | ~100 | ~10K | ~1 MB |
| 中型博客 | ~10K | ~1K | ~100K | ~10 MB |

完全可控，无需引入 Redis。

---

## 六、数据一致性

| 场景 | 行为 | 影响 |
|------|------|------|
| 正常运行 | 每 30 秒刷盘 | `posts.views` 最多延迟 30 秒 |
| 优雅退出（SIGTERM） | Wire cleanup → 排空 channel + flush | 几乎无损失 |
| 崩溃（SIGKILL） | 缓冲区丢失 | 最多丢失 30 秒浏览记录 |
| `views` 与 COUNT 不一致 | 允许 | 必要时可离线重算 |

---

## 七、文件清单

| 层 | 文件 | 说明 |
|----|------|------|
| Entity | [post_view.go](internal/entity/post_view.go#L1-L12) | `PostView` 领域实体 |
| Repo 接口 | [contracts.go](internal/repo/contracts.go#L121-L130) | `PostViewRepo` 接口 |
| Repo 实现 | [post_view_buffered.go](internal/repo/viewbuffer/post_view_buffered.go#L1-L166) | 缓冲实现（channel + 去重 + 批量 flush） |
| UseCase 接口 | [contracts.go](internal/usecase/contracts.go#L79-L104) | `Content.RecordView` 方法 |
| UseCase 实现 | [content.go](internal/usecase/content/content.go#L444-L459) | 注入 `PostViewRepo` 并调用 `Record` |
| Controller | [content.go](internal/controller/http/v1/content.go#L104-L174) | `getPost` 触发 `RecordView` |
| DI | [wire.go](internal/app/wire.go#L180-L430) | Provider：`NewPostViewRepo` 并注入 `NewContentUseCase` |
| DI | [wire_gen.go](internal/app/wire_gen.go#L1-L40) | Wire 生成产物 |

---

## 八、与现有架构的对齐

| 架构规范 | 本设计的遵循方式 |
|----------|---------------|
| Controller → UseCase → Repo 数据流 | `getPost` → `content.RecordView` → `postViews.Record` |
| Entity 纯结构体、无 tag | `entity.PostView` |
| Repo 接口集中在 `contracts.go` | `PostViewRepo` 定义在 `repo/contracts.go` |
| UseCase 纯业务逻辑，不管理生命周期 | `RecordView` 只做 entity 构造 + Repo 调用 |
| 缓冲是 Repo 实现细节 | 参照 `repo/notification/`（组合 Repo + Hub），`repo/viewbuffer/` 封装缓冲策略 |
| Wire cleanup 管理生命周期 | 参照 `NewPostgres`/`NewRedis` 返回 `(T, func())`，PostViewRepo 同理 |
| `App` 只持有基础设施 | `App` 结构体不变，不引入 UseCase 依赖 |
| Gen API 优先 | `CreateInBatches` / `UpdateSimple(Views.Add())` |
| 文件命名规范 | `post_view_buffered.go`，`repo/viewbuffer/` 与 `repo/cache/`、`repo/notification/` 同级 |
| Repo 可操作多表 | 参照 `PostLikeRepo` 同时操作 `PostLike` + `Post`，`bufferedPostViewRepo` 同时操作 `PostView` + `Post` |
