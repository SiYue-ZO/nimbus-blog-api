# User 认证与权限（JWT + Refresh Cookie）设计文档

## 一、组件与边界

### 1.1 关键组件

- HTTP：Fiber v3
- UserAuth UseCase：注册、登录、刷新、改密、忘记密码、撤销 refresh
- TokenSigner：Access/Refresh JWT 的签发与解析
- Repo：
  - `UserRepo`：用户查询与密码更新
  - `RefreshTokenStore`：当前 refresh token（Redis）
  - `RefreshTokenBlacklistRepo`：已撤销 refresh token hash（Postgres）
- 外围服务：
  - Captcha：登录前置校验
  - Email：注册/忘记密码验证码校验

### 1.2 依赖关系

```
controller/http/v1
  -> usecase.UserAuth
     -> repo.UserRepo
     -> repo.RefreshTokenStore
     -> repo.RefreshTokenBlacklistRepo
     -> auth/user.TokenSigner

controller/http/middleware (UserJWT)
  -> auth/user.TokenSigner
  -> usecase.UserAuth
```

## 二、JWT 与 TokenSigner

### 2.1 token 类型

- Access Token（Header：`Authorization: Bearer <token>`）
- Refresh Token（HttpOnly Cookie：`refresh_token`）

### 2.2 Claims 结构

- Access claims：
  - `sub`：用户 ID（string）
  - `meta`：可选扩展字段
  - `iss/iat/nbf/exp`
- Refresh claims：
  - `sub`：用户 ID（string）
  - `type`：固定为 `"refresh"`（用于拒绝拿 access token 冒充 refresh）
  - `iss/iat/nbf/exp`

实现：[jwt.go](internal/usecase/auth/user/jwt.go#L1-L192)

### 2.3 TTL 与 issuer

- 默认 TTL：
  - access：15m
  - refresh：7d
- issuer 默认：`nimbus-blog-api`
- secrets 为空会直接报错，TokenSigner 无法初始化

## 三、Bearer 鉴权中间件

### 3.1 强制鉴权（需要登录）

- 中间件：[user_jwt.go](internal/controller/http/middleware/user_jwt.go#L1-L101)
- 行为：
  - 缺 `Authorization`：HTTP 401 + `ErrorLoginRequired`
  - 格式非 Bearer：HTTP 400 + `ErrorParamFormat`
  - access token 过期：HTTP 401 + `ErrorTokenExpired`
  - access token 无效：HTTP 401 + `ErrorTokenInvalid`
  - 缺 `refresh_token` Cookie：HTTP 401 + `ErrorTokenInvalid`
  - refresh 会话无效：HTTP 401 + `ErrorTokenInvalid`（通过 `UserAuth.ValidateSession` 校验）
  - 用户被禁用：HTTP 403 + `ErrorPermissionDenied`
  - 成功：把 claims 写入
    - `ctx.Locals("claims", claims)`
    - `ctx.SetContext(WithAccessClaims(ctx.Context(), claims))`

### 3.2 可选鉴权（匿名可访问）

- 中间件：同文件的 `NewOptionalUserJWTMiddleware`
- 行为：解析失败或 refresh 会话无效直接放行（视为匿名），不返回错误

## 四、注册

### 4.1 路由

- `POST /api/v1/auth/register`
- Handler：[v1/auth.go:register](internal/controller/http/v1/auth.go#L28-L137)

### 4.2 流程

1. 校验邮箱验证码：`Email.VerifyCode(email, code)`
2. 创建用户：`UserAuth.Register(username, email, password)`
3. 自动登录：`UserAuth.Login(email, password)`
4. 写 refresh cookie（HttpOnly）并在响应体返回 access token

当前 cookie 写入（register）：
- `HTTPOnly=true`
- `Secure=false`
- `SameSite=Strict`
- `Path=/`
- `Expires=now + pair.RefreshExpiresIn * time.Second`

说明：
- `Secure=false` 为当前实现行为（便于本地 http 调试）。生产环境应统一为 `Secure=true`，并确保站点全站 https，否则浏览器会拒绝写入/发送该 Cookie。

## 五、登录

### 5.1 路由

- `POST /api/v1/auth/login`
- Handler：[v1/auth.go:login](internal/controller/http/v1/auth.go#L152-L227)

### 5.2 流程

1. 校验验证码：`Captcha.Verify(captcha_id, captcha)`
2. 账号校验与签发：`UserAuth.Login(email, password)`
3. 写 refresh cookie（HttpOnly）并在响应体返回 access token

当前 cookie 写入（login）：
- `HTTPOnly=true`
- `Secure=true`
- `SameSite=Strict`
- `Path=/`
- `Expires=now + pair.RefreshExpiresIn * time.Second`

## 六、刷新令牌（refresh rotation）

### 6.1 路由

- `POST /api/v1/auth/refresh`
- Handler：[v1/auth.go:refresh](internal/controller/http/v1/auth.go#L240-L294)

### 6.2 流程

1. 读取 cookie：`refresh_token`，为空则 HTTP 400 + `ErrorParamMissing`
2. UseCase：`UserAuth.Refresh(refreshToken)`
   - 解析 refresh claims，验证 `type=refresh`
   - 校验用户存在且状态为 `active`
   - 校验 refresh token（两段式）：
     - 若 Redis 存在“当前 refresh token”，要求必须完全相等
     - 若 Redis 不存在，则查 DB 黑名单；命中则判 invalid
   - 签发新的 access + refresh
   - 写入 Redis 的当前 refresh
   - 将旧 refresh 的 `sha256(token)` 写入 DB 黑名单（expires_at 使用旧 token exp）
3. 回写 refresh cookie，并在响应体返回新的 access token（同时返回 refresh token）

UseCase 实现：[auth.go:Login/Refresh](internal/usecase/auth/user/auth.go#L93-L187)

## 七、退出登录

### 7.1 路由

- `POST /api/v1/auth/logout`
- 前置：Bearer 中间件鉴权（路由层挂载）
- Handler：[v1/auth.go:logout](internal/controller/http/v1/auth.go#L359-L384)

### 7.2 行为

- 调用 `UserAuth.RevokeUserRefreshToken` 撤销服务端 refresh 会话
- 写入过期 cookie 清除浏览器侧 `refresh_token`

## 八、服务端 refresh 状态

### 8.1 Redis：当前 refresh token（单用户单会话）

- key：`refresh_token:{userID}`
- value：refresh token 原文

实现：[refresh_token_store_redis.go](internal/repo/cache/refresh_token_store_redis.go#L1-L41)

### 8.2 Postgres：refresh 黑名单

- 存储：`sha256(refresh_token)` + `user_id` + `expires_at`
- 用途：在 Redis 缺失（例如重启、驱逐、切换环境）时，仍能拒绝已轮换/撤销的旧 token

实现：[refresh_token_blacklist_postgres_gen.go](internal/repo/persistence/refresh_token_blacklist_postgres_gen.go#L1-L41)

### 8.3 用户禁用后的会话失效

- 管理员更新用户状态为 disabled 时，调用 `UserAuth.RevokeUserRefreshToken`
- 撤销逻辑：
  - 读取 Redis 当前 refresh token
  - 写入 DB 黑名单
  - 删除 Redis 当前 refresh token
- 后续请求经过 Bearer 中间件时，通过 `ValidateSession` 校验 refresh 会话并拒绝无效会话/禁用账号

实现：[admin/user.go:updateUserStatus](internal/controller/http/admin/user.go#L84-L125)，[auth.go:Revoke/Validate](internal/usecase/auth/user/auth.go#L213-L291)

## 九、与时间单位相关的当前行为

- UseCase 返回的 `ExpiresIn/RefreshExpiresIn` 使用 `AccessTTL/RefreshTTL().Milliseconds()`（单位：毫秒）：
  - [auth.go:TokenPair](internal/usecase/auth/user/auth.go#L112-L131)
- Controller 写 cookie 的 `Expires` 以 `time.Duration(pair.RefreshExpiresIn) * time.Second` 计算：
  - register：[v1/auth.go:register cookie](internal/controller/http/v1/auth.go#L107-L115)
  - login：[v1/auth.go:login cookie](internal/controller/http/v1/auth.go#L211-L220)
  - refresh：[v1/auth.go:refresh cookie](internal/controller/http/v1/auth.go#L278-L286)

结论与建议：
- 当前实现的单位存在错位风险：如果 `RefreshExpiresIn` 以“毫秒”返回，但按“秒”写入 Cookie，会导致 Cookie 过期时间被放大（约 1000 倍）。
- 建议二选一统一：
  - 方案 A：UseCase 返回秒（`TTL().Seconds()`），Controller 继续按 `time.Second` 计算；
  - 方案 B：UseCase 继续返回毫秒（`TTL().Milliseconds()`），Controller 改为按 `time.Millisecond` 计算。
