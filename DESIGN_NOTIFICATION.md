# 站内通知功能设计方案

## 一、功能概述

为用户提供站内通知能力，覆盖以下场景：

- **评论回复通知** — 有人回复了该用户的评论
- **评论审核通过** — 用户提交的评论被管理员审核通过
- **管理员通知** — 管理员主动给指定用户发送消息

实时推送采用 **SSE（Server-Sent Events）**。

---

## 二、架构约束

| 约束 | 说明 |
|------|------|
| **依赖方向** | Controller → UseCase → Repo → Entity，只允许向内依赖 |
| **UseCase 同层不互调** | UseCase 之间禁止相互依赖，通知发送下沉到 Repo 层 |
| **Pkg 无内部依赖** | `pkg/` 包不依赖 `internal/` 任何包 |
| **Repo 接口集中定义** | 所有 Repo 接口在 `repo/contracts.go` 中声明 |
| **UseCase 接口集中定义** | 所有 UseCase 接口在 `usecase/contracts.go` 中声明 |

### 依赖关系图

```
pkg/ssehub                    ← 纯基础设施（Event + Hub），无内部依赖
    ▲               ▲
    │               │
repo/persistence    usecase/notification
(Notifier 实现)     (Notification UseCase)
    ▲               ▲
    │               │
usecase/comment     controller/http/v1
(业务 UseCase)      (REST + SSE Handler)
```

**关键：** Comment UseCase 依赖 `repo.Notifier`（向下），不依赖 `usecase.Notification`（同层）。

---

## 三、数据库设计

### 3.1 ENUM 类型

```sql
CREATE TYPE notification_type AS ENUM (
    'comment_reply',
    'comment_approved',
    'admin_message'
);
```

### 3.2 notifications 表

```sql
CREATE TABLE notifications (
    id             BIGSERIAL    PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type           notification_type NOT NULL,
    title          VARCHAR(200) NOT NULL,
    content        TEXT         NOT NULL DEFAULT '',
    meta           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    is_read        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_unread
    ON notifications(user_id, is_read, created_at DESC);
```

| 字段 | 说明 |
|------|------|
| `user_id` | 通知接收者（users 外键） |
| `type` | 通知类型枚举 |
| `title` | 通知标题 |
| `content` | 通知正文（摘要） |
| `meta` | 通用扩展字段（JSONB），用于存放跳转/关联信息 |
| `is_read` | 已读标记 |

#### meta 建议字段

| key | 说明 |
|-----|------|
| `post_id` | 文章 ID（可选） |
| `post_slug` | 文章 slug（可选） |
| `comment_id` | 评论 ID（可选） |
| `parent_comment_id` | 父评论 ID（可选） |
| `target_url` | 前端可直接跳转的站内 URL（可选） |
| `source` | 来源标记（如 `"admin"`）（可选） |

---

## 四、Pkg 层 — SSE Hub

Hub 是 SSE 的核心基础设施，放在 `pkg/ssehub/` 下，无任何 `internal/` 依赖。Repo 层（`Notifier` 实现）和 UseCase 层（`Notification` 实现）均可合规引用。

### 4.1 文件

[`hub.go`](pkg/ssehub/hub.go#L1-L74)

### 4.2 实现

```go
// Package ssehub SSE Hub（内存级连接管理与事件推送）。
package ssehub

import "sync"

const _defaultBufferSize = 16

// Event SSE 事件。
type Event struct {
	Name string
	Data []byte
}

// Hub SSE Hub。
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[chan Event]struct{}
}

// New 创建 Hub。
func New() *Hub {
	return &Hub{clients: make(map[int64]map[chan Event]struct{})}
}

// Subscribe 订阅指定用户的事件通道。
func (h *Hub) Subscribe(userID int64) chan Event {
	ch := make(chan Event, _defaultBufferSize)
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan Event]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 取消订阅并关闭事件通道。
func (h *Hub) Unsubscribe(userID int64, ch chan Event) {
	h.mu.Lock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, ch)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

// Publish 向指定用户推送事件。
func (h *Hub) Publish(userID int64, event Event) {
	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	for ch := range conns {
		select {
		case ch <- event:
		default:
		}
	}
}

// Shutdown 关闭 Hub 并清理所有订阅。
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for uid, conns := range h.clients {
		for ch := range conns {
			close(ch)
		}
		delete(h.clients, uid)
	}
}
```

**设计要点：**

- 包级默认值用下划线前缀（`_defaultBufferSize`），符合 Pkg 封装规范
- 每用户多连接（多标签页场景），`map[chan]struct{}` 管理
- 带缓冲 channel，缓冲区满时静默丢弃，保证不阻塞调用方
- `Shutdown()` 用于优雅停机，可在 `app.go` 中调用

---

## 五、Entity 层

[`notification.go`](internal/entity/notification.go#L1-L29)

```go
package entity

import (
	"encoding/json"
	"time"
)

const (
	NotificationTypeCommentReply    = "comment_reply"
	NotificationTypeCommentApproved = "comment_approved"
	NotificationTypeAdminMessage    = "admin_message"
)

type Notification struct {
	ID        int64
	UserID    int64
	Type      string
	Title     string
	Content   string
	Meta      json.RawMessage
	IsRead    bool
	CreatedAt time.Time
}
```

遵循 Entity 规范：纯结构体、无 struct tag、可选字段用指针。

---

## 六、Repo 层

### 6.1 接口定义

接口定义见：[contracts.go](internal/repo/contracts.go#L169-L181)

```go
NotificationRepo interface {
	Create(ctx context.Context, n entity.Notification) (int64, error)
	List(ctx context.Context, offset, limit int, userID int64, isRead *bool, sortBy *string, order *string) ([]*entity.Notification, int64, error)
	MarkRead(ctx context.Context, id, userID int64) error
	MarkAllRead(ctx context.Context, userID int64) error
	CountUnread(ctx context.Context, userID int64) (int64, error)
	Delete(ctx context.Context, id, userID int64) error
}

Notifier interface {
	Send(ctx context.Context, n entity.Notification) error
}
```

**为什么 `Notifier` 是 Repo 接口而非 UseCase 接口？**

- Comment UseCase 需要在回复/审核时发送通知
- UseCase 同层不允许互调 → 不能依赖 `usecase.Notification`
- `Notifier.Send()` 本质是 "写入 DB + 推送侧效"，与 `EmailSender.Send()` 同级
- Comment UseCase 依赖 `repo.Notifier` = 向下依赖，完全合规

### 6.2 NotificationRepo 实现

[`notification_postgres_gen.go`](internal/repo/persistence/notification_postgres_gen.go#L1-L120)

```go
package persistence

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
)

type notificationRepo struct {
	query *query.Query
}

func NewNotificationRepo(db *gorm.DB) repo.NotificationRepo {
	return &notificationRepo{query: query.Use(db)}
}

func (r *notificationRepo) Create(ctx context.Context, en entity.Notification) (int64, error) {
	mn := toModelNotification(&en)
	if err := r.query.Notification.WithContext(ctx).Create(mn); err != nil {
		return 0, err
	}
	return mn.ID, nil
}

func (r *notificationRepo) List(ctx context.Context, offset, limit int, userID int64, isRead *bool, sortBy *string, order *string) ([]*entity.Notification, int64, error) {
	n := r.query.Notification
	do := n.WithContext(ctx).Where(n.UserID.Eq(userID))

	if isRead != nil {
		do = do.Where(n.IsRead.Is(*isRead))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := n.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(n.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	items := make([]*entity.Notification, len(rows))
	for i, mn := range rows {
		items[i] = toEntityNotification(mn)
	}
	return items, total, nil
}

func (r *notificationRepo) CountUnread(ctx context.Context, userID int64) (int64, error) {
	n := r.query.Notification
	return n.WithContext(ctx).Where(n.UserID.Eq(userID), n.IsRead.Is(false)).Count()
}

func (r *notificationRepo) MarkRead(ctx context.Context, id, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.ID.Eq(id), n.UserID.Eq(userID)).Update(n.IsRead, true)
	return err
}

func (r *notificationRepo) MarkAllRead(ctx context.Context, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.UserID.Eq(userID), n.IsRead.Is(false)).Update(n.IsRead, true)
	return err
}

func (r *notificationRepo) Delete(ctx context.Context, id, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.ID.Eq(id), n.UserID.Eq(userID)).Delete()
	return err
}

func toModelNotification(en *entity.Notification) *model.Notification {
	meta := en.Meta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &model.Notification{
		ID:        en.ID,
		UserID:    en.UserID,
		Type:      en.Type,
		Title:     en.Title,
		Content:   en.Content,
		Meta:      string(meta),
		IsRead:    en.IsRead,
		CreatedAt: en.CreatedAt,
	}
}

func toEntityNotification(mn *model.Notification) *entity.Notification {
	return &entity.Notification{
		ID:        mn.ID,
		UserID:    mn.UserID,
		Type:      mn.Type,
		Title:     mn.Title,
		Content:   mn.Content,
		Meta:      json.RawMessage(mn.Meta),
		IsRead:    mn.IsRead,
		CreatedAt: mn.CreatedAt,
	}
}
```

### 6.3 Notifier 实现

[`notifier.go`](internal/repo/notification/notifier.go#L1-L50)

```go
package notification

import (
    "context"
    "encoding/json"
    "unicode/utf8"

    "github.com/scc749/nimbus-blog-api/internal/entity"
    "github.com/scc749/nimbus-blog-api/internal/repo"
    "github.com/scc749/nimbus-blog-api/pkg/ssehub"
)

const _maxNotificationContentLen = 100

type notifier struct {
    notifications repo.NotificationRepo
    hub           *ssehub.Hub
}

func NewNotifier(notifications repo.NotificationRepo, hub *ssehub.Hub) repo.Notifier {
    return &notifier{notifications: notifications, hub: hub}
}

func truncate(s string, maxLen int) string {
    if utf8.RuneCountInString(s) <= maxLen {
        return s
    }
    runes := []rune(s)
    return string(runes[:maxLen]) + "..."
}

func (n *notifier) Send(ctx context.Context, notif entity.Notification) error {
    if utf8.RuneCountInString(notif.Content) > _maxNotificationContentLen {
        notif.Content = truncate(notif.Content, _maxNotificationContentLen)
    }
    id, err := n.notifications.Create(ctx, notif)
    if err != nil {
        return err
    }
    notif.ID = id

    data, _ := json.Marshal(notif)
    n.hub.Publish(notif.UserID, ssehub.Event{Name: "notification", Data: data})

    count, _ := n.notifications.CountUnread(ctx, notif.UserID)
    countData, _ := json.Marshal(map[string]int64{"count": count})
    n.hub.Publish(notif.UserID, ssehub.Event{Name: "unread_count", Data: countData})

    return nil
}
```

说明：
- `notification` 事件推送的 payload 为 `entity.Notification` 的 JSON；其中 `post_slug/comment_id/target_url` 等跳转信息保存在 `meta` 内（由前端解析或通过 REST 列表获取已展开字段）。
- `content` 会在 `repo.Notifier.Send` 内部截断（默认 100 字符，Unicode 安全截断，尾部追加 `...`），保证通知列表/推送不会携带超长正文。
- 放在 `internal/repo/notification/` 而非 `persistence/`，因为它组合了 DB 和 Hub 两种基础设施，不是纯 Persistence 实现。命名模式类似 `repo/messaging/`（SMTP）、`repo/storage/`（MinIO）。

---

## 七、UseCase 层

### 7.1 Input

[`input/notification.go`](internal/usecase/input/notification.go#L1-L14)

```go
package input

type ListNotifications struct {
    PageParams
    Sort   *SortParams
    UserID int64
    IsRead BoolFilterParam
}

type SendAdminNotification struct {
	UserID  int64
	Title   string
	Content string
}
```

> 无 `CreateNotification` Input — 通知创建由 `repo.Notifier.Send()` 直接接收 `entity.Notification`。

### 7.2 Output

[`output/notification.go`](internal/usecase/output/notification.go#L1-L19)

```go
package output

import (
	"encoding/json"
	"time"
)

type NotificationDetail struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Meta      json.RawMessage `json:"meta"`
	PostSlug  *string         `json:"post_slug,omitempty"`
	CommentID *int64          `json:"comment_id,omitempty"`
	TargetURL *string         `json:"target_url,omitempty"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}
```

### 7.3 UseCase 接口

接口定义见：[contracts.go](internal/usecase/contracts.go#L149-L163)

```go
type Notification interface {
	// V1
	ListMyNotifications(ctx context.Context, params input.ListNotifications) (*output.ListResult[output.NotificationDetail], error)
	GetUnreadCount(ctx context.Context, userID int64) (int64, error)
	MarkRead(ctx context.Context, id, userID int64) error
	MarkAllRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, id, userID int64) error

	// Admin
	SendAdminMessage(ctx context.Context, params input.SendAdminNotification) error

	// SSE
	Subscribe(userID int64) chan ssehub.Event
	Unsubscribe(userID int64, ch chan ssehub.Event)
}
```

**接口职责边界：**

| 方法 | 职责 |
|------|------|
| `List` / `Get` / `Mark` / `Delete` | 通知 REST API（供 V1 Controller 调用） |
| `Subscribe` / `Unsubscribe` | SSE 连接管理（供 V1 SSE Handler 调用） |
| `SendAdminMessage` | 管理端站内消息发送（供 Admin Controller 调用） |

> 注意：通知“投递”（DB 持久化 + SSE 推送）仍由 `repo.Notifier` 承担；UseCase 仅在需要时（如 Admin 模块）封装一层输入校验与组装逻辑，避免 Controller 直连 Repo。

### 7.4 UseCase 实现

[`notification/notification.go`](internal/usecase/notification/notification.go#L1-L156)

```go
package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
	"github.com/scc749/nimbus-blog-api/pkg/ssehub"
)

var ErrRepo = errors.New("repo")

type useCase struct {
	notifications repo.NotificationRepo
	notifier      repo.Notifier
	hub           *ssehub.Hub
}

func New(notifications repo.NotificationRepo, notifier repo.Notifier, hub *ssehub.Hub) usecase.Notification {
	return &useCase{notifications: notifications, notifier: notifier, hub: hub}
}

func (u *useCase) Subscribe(userID int64) chan ssehub.Event {
	return u.hub.Subscribe(userID)
}

func (u *useCase) Unsubscribe(userID int64, ch chan ssehub.Event) {
	u.hub.Unsubscribe(userID, ch)
}

func (u *useCase) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	count, err := u.notifications.CountUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return count, nil
}

func (u *useCase) ListMyNotifications(ctx context.Context, params input.ListNotifications) (*output.ListResult[output.NotificationDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var isRead *bool
	if params.IsRead != nil {
		isRead = (*bool)(params.IsRead)
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	rows, total, err := u.notifications.List(ctx, offset, params.PageSize, params.UserID, isRead, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.NotificationDetail, len(rows))
	for i, n := range rows {
		items[i] = toNotificationDetail(n)
	}

	return &output.ListResult[output.NotificationDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) MarkRead(ctx context.Context, id, userID int64) error {
	if err := u.notifications.MarkRead(ctx, id, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) MarkAllRead(ctx context.Context, userID int64) error {
	if err := u.notifications.MarkAllRead(ctx, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) DeleteNotification(ctx context.Context, id, userID int64) error {
	if err := u.notifications.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) SendAdminMessage(ctx context.Context, params input.SendAdminNotification) error {
	meta, _ := json.Marshal(map[string]string{entity.NotificationMetaSource: "admin"})
	return u.notifier.Send(ctx, entity.Notification{
		UserID:  params.UserID,
		Type:    entity.NotificationTypeAdminMessage,
		Title:   params.Title,
		Content: params.Content,
		Meta:    meta,
	})
}

func toNotificationDetail(n *entity.Notification) output.NotificationDetail {
	if n == nil {
		return output.NotificationDetail{}
	}

	meta := n.Meta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}

	d := output.NotificationDetail{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Content:   n.Content,
		Meta:      meta,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}

	if len(n.Meta) > 0 {
		var m struct {
			PostSlug  *string `json:"post_slug"`
			CommentID *int64  `json:"comment_id"`
			TargetURL *string `json:"target_url"`
		}
		if err := json.Unmarshal(n.Meta, &m); err == nil {
			d.PostSlug = m.PostSlug
			d.CommentID = m.CommentID
			d.TargetURL = m.TargetURL
		}
	}

	return d
}

func (u *useCase) publishUnreadCount(ctx context.Context, userID int64) {
	count, err := u.notifications.CountUnread(ctx, userID)
	if err != nil {
		return
	}
	data, err := json.Marshal(map[string]int64{"count": count})
	if err != nil {
		return
	}
	u.hub.Publish(userID, ssehub.Event{Name: "unread_count", Data: data})
}
```

---

## 八、Controller 层

### 8.1 路由

**V1（用户端，JWT Bearer 认证）：**

```
GET    /api/v1/notifications          → 通知列表
GET    /api/v1/notifications/unread   → 未读数量
GET    /api/v1/notifications/stream   → SSE 实时流（长连接）
PUT    /api/v1/notifications/:id/read → 标记单条已读
PUT    /api/v1/notifications/read-all → 全部已读
DELETE /api/v1/notifications/:id      → 删除单条
```

**Admin（管理端，Session Cookie 认证）：**

```
POST   /api/admin/notifications       → 给指定用户发送通知
```

### 8.2 路由注册

V1 注册见：[router.go](internal/controller/http/v1/router.go#L137-L153)

```go
func NewNotificationRoutes(apiV1Group fiber.Router, l logger.Interface, signer authUC.TokenSigner, auth usecase.UserAuth, notification usecase.Notification) {
	r := &V1{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), notification: notification, signer: signer}

	notificationPublicGroup := apiV1Group.Group("/notifications")
	{
		notificationPublicGroup.Get("/stream", r.streamNotifications)
	}

	notificationAuthGroup := apiV1Group.Group("/notifications", middleware.NewUserJWTMiddleware(signer, auth))
	{
		notificationAuthGroup.Get("/", r.listNotifications)
		notificationAuthGroup.Get("/unread", r.getUnreadCount)
		notificationAuthGroup.Put("/:id/read", r.markRead)
		notificationAuthGroup.Put("/read-all", r.markAllRead)
		notificationAuthGroup.Delete("/:id", r.deleteNotification)
	}
}
```

Admin 注册见：[router.go](internal/controller/http/admin/router.go#L123-L130)

```go
func NewNotificationRoutes(apiAdminGroup fiber.Router, l logger.Interface, store *session.Store, notify usecase.Notification) {
	r := &Admin{logger: l, validate: validator.New(validator.WithRequiredStructEnabled()), sess: store, notify: notify}

	notifAuthGroup := apiAdminGroup.Group("/notifications", middleware.NewAdminSessionMiddleware(store))
	{
		notifAuthGroup.Post("/", r.sendNotification)
	}
}
```

### 8.3 Request / Response DTO

**Admin Request：**

[`request/notification.go`](internal/controller/http/admin/request/notification.go#L1-L7)

```go
package request

type SendNotification struct {
	UserID  int64  `json:"user_id" validate:"required"`
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1"`
}
```

**V1 Response：**

[`v1/response/notification.go`](internal/controller/http/v1/response/notification.go#L1-L23)

```go
package response

import (
	"encoding/json"
	"time"
)

type NotificationDetail struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Meta      json.RawMessage `json:"meta"`
	PostSlug  *string         `json:"post_slug,omitempty"`
	CommentID *int64          `json:"comment_id,omitempty"`
	TargetURL *string         `json:"target_url,omitempty"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}

type UnreadCount struct {
	Count int64 `json:"count"`
}
```

### 8.4 业务状态码

**V1（`v1/response/codes.go`）— 通知模块 (9xx)：**

```go
// 通知模块 (09xx)
const (
    // 操作相关 (0960-0979)
    ErrorListNotificationsFailed  = "0960" // 获取通知列表失败
    ErrorGetUnreadCountFailed     = "0961" // 获取未读数量失败
    ErrorMarkReadFailed           = "0962" // 标记已读失败
    ErrorMarkAllReadFailed        = "0963" // 全部已读失败
    ErrorDeleteNotificationFailed = "0964" // 删除通知失败
)
```

**Admin（`admin/response/codes.go`）— 通知模块 (19xx)：**

```go
// 通知模块 (19xx)
const (
    // 操作相关 (1960-1979)
    ErrorSendNotificationFailed = "1960" // 发送通知失败
)
```

### 8.5 SSE Stream Handler

[`v1/notification.go:streamNotifications`](internal/controller/http/v1/notification.go#L218-L272)

```go
func (r *V1) streamNotifications(ctx fiber.Ctx) error {
    // EventSource 不支持自定义 Header，从 URL query 获取 token
    tokenStr := ctx.Query("token", "")
    if tokenStr == "" {
        return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "missing token")
    }
    claims, err := r.signer.ParseAccess(tokenStr)
    if err != nil {
        return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
    }
    uid, err := claims.UserIDInt()
    if err != nil {
        return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
    }

    // SSE 必需响应头
    ctx.Set("Content-Type", "text/event-stream")
    ctx.Set("Cache-Control", "no-cache")
    ctx.Set("Connection", "keep-alive")
    ctx.Set("X-Accel-Buffering", "no")

    ch := r.notification.Subscribe(uid)

    ctx.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
        defer r.notification.Unsubscribe(uid, ch)

        // 连接建立 → 立即推送当前未读数
        count, _ := r.notification.GetUnreadCount(context.Background(), uid)
        countData, _ := json.Marshal(map[string]int64{"count": count})
        fmt.Fprintf(w, "event: unread_count\ndata: %s\n\n", countData)
        w.Flush()

        // 30 秒心跳，防止代理超时断连
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case event, ok := <-ch:
                if !ok {
                    return
                }
                if event.Name != "" {
                    fmt.Fprintf(w, "event: %s\n", event.Name)
                }
                fmt.Fprintf(w, "data: %s\n\n", event.Data)
                if err := w.Flush(); err != nil {
                    return
                }
            case <-ticker.C:
                fmt.Fprintf(w, ": heartbeat\n\n")
                if err := w.Flush(); err != nil {
                    return
                }
            }
        }
    })
    return nil
}
```

**SSE 协议格式：**

```
event: unread_count
data: {"count":3}

event: notification
data: {"id":42,"type":"comment_reply","title":"你的评论收到了新回复",...}

: heartbeat
```

- `event:` — 事件类型，前端 `addEventListener(name)` 分别监听
- `data:` — JSON 载荷
- `: heartbeat` — SSE 注释行（冒号开头），客户端忽略，仅保活
- 每个事件以空行 `\n\n` 结尾

### 8.6 Admin Handler

```go
func (r *Admin) sendNotification(ctx fiber.Ctx) error {
    var body request.SendNotification
    if err := ctx.Bind().Body(&body); err != nil {
        return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
    }
    if err := r.validate.Struct(body); err != nil {
        return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
    }

    if err := r.notify.SendAdminMessage(ctx.Context(), input.SendAdminNotification{
        UserID:  body.UserID,
        Title:   body.Title,
        Content: body.Content,
    }); err != nil {
        r.logger.Error(err, "http - admin - notification - sendNotification - usecase")
        return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSendNotificationFailed, "send notification failed")
    }
    return sharedresp.WriteSuccess(ctx)
}
```

---

## 九、通知触发点

业务 UseCase 通过 `repo.Notifier`（向下依赖）发送通知。

### 9.1 Comment UseCase 改造

```go
// internal/usecase/comment/comment.go

type useCase struct {
    comments     repo.CommentRepo
    commentLikes repo.CommentLikeRepo
    users        repo.UserRepo
    posts        repo.PostRepo
    notifier     repo.Notifier // 新增
}

func New(comments repo.CommentRepo, commentLikes repo.CommentLikeRepo, users repo.UserRepo, posts repo.PostRepo, notifier repo.Notifier) usecase.Comment {
    return &useCase{comments: comments, commentLikes: commentLikes, users: users, posts: posts, notifier: notifier}
}
```

### 9.2 通知触发策略

> **设计原则：所有评论相关通知统一在审核通过时触发。**
>
> - `SubmitComment` 仅创建评论（status=pending），不发送任何通知。
> - `UpdateCommentStatus(approved)` 审核通过后同时发送：
>   1. `comment_approved` —— 通知评论者"你的评论已通过审核"。
>   2. `comment_reply`（仅回复评论）—— 通知被回复者"你的评论收到了新回复"。
>
> 这样保证用户收到通知后，一定能看到对应的评论内容。

### 9.3 审核通过后发送通知（在 UpdateCommentStatus 内触发）

```go
// 省略：状态更新、权限校验等逻辑
if status == entity.CommentStatusApproved {
    post, _ := u.posts.GetByID(ctx, c.PostID)
    postSlug := ""
    if post != nil {
        postSlug = post.Slug
    }
    targetURL := ""
    if postSlug != "" {
        targetURL = fmt.Sprintf("/post/%s#comment-%d", postSlug, c.ID)
    }

    metaApproved, _ := json.Marshal(map[string]interface{}{
        entity.NotificationMetaPostID:    c.PostID,
        entity.NotificationMetaPostSlug:  postSlug,
        entity.NotificationMetaCommentID: c.ID,
        entity.NotificationMetaTargetURL: targetURL,
    })
    _ = u.notifier.Send(ctx, entity.Notification{
        UserID:  c.UserID,
        Type:    entity.NotificationTypeCommentApproved,
        Title:   "你的评论已通过审核",
        Content: c.Content,
        Meta:    metaApproved,
    })

    if c.ParentID != nil {
        parent, _ := u.comments.GetByID(ctx, *c.ParentID)
        if parent != nil && parent.UserID != c.UserID {
            metaReply, _ := json.Marshal(map[string]interface{}{
                entity.NotificationMetaPostID:          c.PostID,
                entity.NotificationMetaPostSlug:        postSlug,
                entity.NotificationMetaCommentID:       c.ID,
                entity.NotificationMetaParentCommentID: *c.ParentID,
                entity.NotificationMetaTargetURL:       targetURL,
            })
            _ = u.notifier.Send(ctx, entity.Notification{
                UserID:  parent.UserID,
                Type:    entity.NotificationTypeCommentReply,
                Title:   "你的评论收到了新回复",
                Content: c.Content,
                Meta:    metaReply,
            })
        }
    }
}
```

---

## 十、Wire 依赖注入

实现见：[wire.go](internal/app/wire.go#L185-L438)

```go
func NewNotificationRepo(pg *postgres.Postgres) repo.NotificationRepo {
	return persistence.NewNotificationRepo(pg.DB)
}
```

修改现有 Provider：

```go
func NewCommentUseCase(commentRepo repo.CommentRepo, commentLikeRepo repo.CommentLikeRepo, userRepo repo.UserRepo, postRepo repo.PostRepo, notifier repo.Notifier) usecase.Comment {
	return comment.New(commentRepo, commentLikeRepo, userRepo, postRepo, notifier)
}
```

ProviderSet 新增：

```go
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

> Wire 保证 `ssehub.Hub` 为单例 — 同一实例注入到 `Notifier`（写端）和 `Notification UseCase`（读端）。

---

## 十一、SSE 实时推送详解

### 11.1 数据流

```
Comment UC ──► repo.Notifier.Send() ──► NotificationRepo.Create() (DB)
                                    ──► Hub.Publish() (内存)
                                              │
                                              ▼ chan ssehub.Event
                                        SSE Handler
                                              │
                                              ▼ text/event-stream
                                        Browser EventSource
```

### 11.2 SSE 事件类型

| 事件名 | 触发时机 | Data |
|--------|----------|------|
| `unread_count` | 连接建立 + 未读数变化（新通知、标记已读、全部已读、删除） | `{"count": 5}` |
| `notification` | 新通知到达 | `entity.Notification` JSON（跳转信息在 `meta` 内） |
| `: heartbeat` | 每 30 秒 | 无（SSE 注释行，仅保活） |

### 11.3 连接生命周期

```
浏览器                                        服务端
  │                                              │
  │── GET /notifications/stream?token=xxx ──────►│
  │                                              │── 验证 JWT
  │                                              │── hub.Subscribe(uid)
  │◄── event: unread_count ─────────────────────│
  │    data: {"count": 3}                        │
  │                                              │
  │    ... 等待 ...                               │
  │                                              │── hub.Publish(uid, event)
  │◄── event: notification ─────────────────────│
  │    data: {"id":42,...}                       │
  │◄── event: unread_count ─────────────────────│
  │    data: {"count": 4}                        │
  │                                              │
  │◄── : heartbeat ────────────────────────────│   (每 30s)
  │                                              │
  │── 断开（关闭页面 / 网络中断）──────────────►│
  │                                              │── w.Flush() 返回 err
  │                                              │── defer hub.Unsubscribe(uid, ch)
  │                                              │
  │── 自动重连（EventSource 内置）─────────────►│
  │                                              │── 重新 Subscribe
```

### 11.4 JWT 认证

`EventSource` API 不支持自定义 Header，SSE 端点通过 URL query 传 token：

```
GET /api/v1/notifications/stream?token=eyJhbGciOiJIUzI1NiIs...
```

> Access Token 有效期短（通常 15 分钟），URL 泄露风险可控。

### 11.5 异常处理

| 场景 | 处理 |
|------|------|
| 客户端断开 | `w.Flush()` 返回 error → handler return → `defer Unsubscribe()` |
| 慢客户端 | channel 缓冲区满 → `select default` 丢弃事件 |
| 浏览器不支持 SSE | 降级到 `GET /notifications/unread` 轮询 |
| 代理超时 | 30s 心跳 + `X-Accel-Buffering: no` |
| 服务端重启 | 连接断开 → `EventSource` 自动重连 → 重新 Subscribe |

---

## 十二、前端设计

### 12.1 SSE Hook

`hooks/useNotificationSSE.ts`

```ts
import { useEffect, useRef, useCallback } from "react";

interface Options {
    token: string | null;
    onUnreadCount?: (count: number) => void;
    onNotification?: (n: NotificationDetail) => void;
}

export function useNotificationSSE({ token, onUnreadCount, onNotification }: Options) {
    const esRef = useRef<EventSource | null>(null);

    const connect = useCallback(() => {
        if (!token) return;
        esRef.current?.close();

        const es = new EventSource(
            `${process.env.NEXT_PUBLIC_API_BASE}/api/v1/notifications/stream?token=${token}`
        );

        es.addEventListener("unread_count", (e) => {
            onUnreadCount?.(JSON.parse(e.data).count);
        });
        es.addEventListener("notification", (e) => {
            onNotification?.(JSON.parse(e.data));
        });

        esRef.current = es;
    }, [token, onUnreadCount, onNotification]);

    useEffect(() => {
        connect();
        return () => { esRef.current?.close(); esRef.current = null; };
    }, [connect]);
}
```

### 12.2 NotificationProvider

`contexts/NotificationContext.tsx`

```tsx
"use client";

import { createContext, useContext, useState, useCallback } from "react";
import { useNotificationSSE } from "@/hooks/useNotificationSSE";
import { useAuth } from "@/contexts/AuthContext";

interface NotificationContextType {
    unreadCount: number;
    latestNotifications: NotificationDetail[];
    refreshCount: () => void;
}

const NotificationContext = createContext<NotificationContextType>({
    unreadCount: 0, latestNotifications: [], refreshCount: () => {},
});

export function NotificationProvider({ children }: { children: React.ReactNode }) {
    const { token, isLoggedIn } = useAuth();
    const [unreadCount, setUnreadCount] = useState(0);
    const [latest, setLatest] = useState<NotificationDetail[]>([]);

    useNotificationSSE({
        token: isLoggedIn ? token : null,
        onUnreadCount: useCallback((c: number) => setUnreadCount(c), []),
        onNotification: useCallback((n: NotificationDetail) => {
            setLatest((prev) => [n, ...prev].slice(0, 5));
        }, []),
    });

    const refreshCount = useCallback(async () => {
        const res = await getUnreadCount();
        setUnreadCount(res.count);
    }, []);

    return (
        <NotificationContext.Provider value={{ unreadCount, latestNotifications: latest, refreshCount }}>
            {children}
        </NotificationContext.Provider>
    );
}

export const useNotifications = () => useContext(NotificationContext);
```

### 12.3 组件清单

| 组件 | 位置 | 说明 |
|------|------|------|
| `useNotificationSSE` | Hook | SSE 连接管理 |
| `NotificationProvider` | Context | 全局状态（未读数 + 最新通知） |
| `NotificationBell` | Navbar 右侧 | 铃铛 + Badge 未读数 |
| `NotificationDropdown` | 点击铃铛 | Popover 最近 5 条 + "查看全部" |
| `NotificationsPage` | `/notifications` | 完整列表 + 分页 + 筛选 |

### 12.4 HeroUI 组件

`Badge`、`Popover`、`Listbox`、`Button`、`Chip`、`Pagination`、`Tabs`

### 12.5 前端 API 层

`lib/api/v1/notification.ts` / `lib/api/admin/notification.ts`

```ts
// V1
export async function listNotifications(query?: ListNotificationsQuery): Promise<Page<NotificationDetail>> { /* ... */ }
export async function getUnreadCount(): Promise<{ count: number }> { /* ... */ }
export async function markRead(id: number): Promise<void> { /* ... */ }
export async function markAllRead(): Promise<void> { /* ... */ }
export async function deleteNotification(id: number): Promise<void> { /* ... */ }

// Admin
export async function sendNotification(body: SendNotificationBody): Promise<void> { /* ... */ }
```

---

## 十三、Nginx 配置

SSE 需要禁用代理缓冲，否则事件会被积攒到缓冲区满才发送：

```nginx
location /api/v1/notifications/stream {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 3600s;
    chunked_transfer_encoding off;
}
```

---
