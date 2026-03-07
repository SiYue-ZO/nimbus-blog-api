# Admin 认证与权限（Session + 2FA）设计文档

## 一、组件与边界

### 1.1 关键组件

- HTTP：Fiber v3 + session middleware
- Session 存储：Redis-backed session store
- AdminAuth UseCase：账号验证、密码修改、2FA 管理
- Repo：
  - `AdminRepo`：管理员账号与 2FA 持久化
  - `AdminTwoFASetupStore`：2FA setup 缓存（Redis）

### 1.2 依赖关系

```
controller/http/admin
  -> usecase.AdminAuth
     -> repo.AdminRepo
     -> repo.AdminTwoFASetupStore

controller/http/middleware (AdminSession)
  -> session.Store (Redis storage)
```

## 二、Session 模型

### 2.1 Session Cookie

- Cookie 名：`fiber_session`（Fiber session 默认名；Swagger 里也按此声明）
- 服务端存储：Redis（由 `session.NewStore(session.Config{Storage: redisstore.New(...)})` 初始化）
- Cookie 策略（初始化时）：`HTTPOnly=true`、`Secure=true`、`SameSite=Strict`、`IdleTimeout=24h`

说明：
- `Secure=true` 表示 Cookie 仅在 https 下写入/发送；本地若使用纯 http 访问管理端接口，需要通过部署层（反代/本地证书）提供 https，或调整 session store 的 CookieSecure 配置以匹配环境。

相关实现：
- Session 初始化：[router.go](internal/controller/http/router.go#L42-L58)
- AdminSession 中间件：[admin_session.go](internal/controller/http/middleware/admin_session.go#L1-L27)

示例（Session 初始化片段）：

```go
// internal/controller/http/router.go
rs := redisstore.New(redisstore.Config{
    Host: cfg.Redis.Host, Port: cfg.Redis.Port, Password: cfg.Redis.Password, Database: cfg.Redis.DB,
})
store := session.NewStore(session.Config{
    Storage: rs, CookieHTTPOnly: true, CookieSecure: true, CookieSameSite: "Strict", IdleTimeout: 24 * time.Hour,
})
```

### 2.2 Session 字段

- `admin_id`：字符串形式的管理员 ID
- `recovery_login`：bool，标记本次登录是否使用了 recovery code（仅用于标识，不参与鉴权）

## 三、鉴权入口（AdminSession 中间件）

### 3.1 校验逻辑

- `store.Get(ctx)` 获取 session
- 必须存在 `admin_id`
- 成功则写入 `ctx.Locals("admin_id", sess.Get("admin_id"))`

失败响应：
- HTTP 401
- biz code：`admin/response.ErrorAdminSessionMissing`（`"1123"`）

## 四、登录流程

### 4.1 路由与请求

- `POST /api/admin/auth/login`
- 请求 DTO：[request/auth.go](internal/controller/http/admin/request/auth.go#L1-L36)
  - `username/password`
  - 可选：`otp_code`、`recovery_code`
- Handler：[admin/auth.go:login](internal/controller/http/admin/auth.go#L29-L111)

### 4.2 流程

1. 校验用户名密码：`AdminAuth.Login(username, password)`
2. 若 `MustResetPassword=true`：直接返回 `RequiresReset=true`，不写 Session
3. 若已启用 2FA（`TwoFactorSecret != nil && != ""`）：
   - 有 `otp_code`：`ValidateTOTP(adminID, otp_code)`，失败返回 401
   - 有 `recovery_code`：`VerifyAndUseRecoveryCode(adminID, recovery_code)`，失败返回 401
   - 两者都没有：返回 `OTPRequired=true`，不写 Session
4. 写入 Session：
   - `admin_id = strconv.FormatInt(admin.ID, 10)`
   - 若使用了 `recovery_code`：`recovery_login=true`
   - `IdleTimeout=24h`
5. `sess.Save()` 后返回 `RequiresReset=false`、`OTPRequired=false`

### 4.3 错误映射（登录）

- `ErrAdminNotFound` → HTTP 404 + `ErrorAdminNotFound`（`"1101"`）
- `ErrPasswordWrong` → HTTP 401 + `ErrorAdminPasswordWrong`（`"1120"`）
- `ErrRepo` → HTTP 500 + `ErrorDatabase`（`"0061"`）
- OTP 错误 → HTTP 401 + `ErrorAdminOTPWrong`（`"1121"`）
- Recovery code 错误 → HTTP 401 + `ErrorAdminRecoveryCodeWrong`（`"1122"`）

## 五、密码修改

### 5.1 强制重置密码（reset）

- `POST /api/admin/auth/reset`
- 逻辑：先用 `Login(username, old_password)` 验证，再 `ChangePassword(ClearResetFlag=true)`
- Handler：[admin/auth.go:resetPassword](internal/controller/http/admin/auth.go#L124-L187)

### 5.2 在线修改密码（change)

- `PUT /api/admin/auth/password`
- 前置：AdminSession 中间件（从 locals 取 `admin_id`）
- Handler：[admin/auth.go:changePassword](internal/controller/http/admin/auth.go#L201-L254)

## 六、2FA（TOTP + setup 缓存 + recovery codes）

### 6.1 存储策略

- 数据库：`admins.two_factor_secret` 保存“加密后的 secret”
  - 加密实现：[crypto.go](internal/usecase/auth/admin/crypto.go#L1-L54)
- setup 缓存：Redis 保存 `{admin_id, secret}`，TTL 10 分钟
  - key：`admin_2fa_setup:{setup_id}`
  - 实现：[admin_twofa_setup_store_redis.go](internal/repo/cache/admin_twofa_setup_store_redis.go#L1-L56)

### 6.2 启用 2FA：setup → verify

#### (1) setup

- `POST /api/admin/auth/2fa/setup`
- 行为：生成 `secret` 与二维码（base64），写入 setup 缓存并返回 `setup_id`
- Handler：[admin/auth.go:twoFASetup](internal/controller/http/admin/auth.go#L367-L423)

#### (2) verify

- `POST /api/admin/auth/2fa/verify`
- 行为：
  - 从 setup 缓存取回 secret，校验 OTP
  - 校验通过后把加密后的 secret 写入 DB
  - 生成 8 个 recovery codes，并以 bcrypt hash 形式写入 DB
  - 删除 setup 缓存
  - 销毁当前 AdminSession（要求重新登录）
- Handler：[admin/auth.go:twoFAVerify](internal/controller/http/admin/auth.go#L424-L490)

### 6.3 禁用 2FA

- `POST /api/admin/auth/2fa/disable`
- 必须提供 `code` 或 `recovery_code` 之一；验证通过后清空 DB 中的 `two_factor_secret`
- Handler：[admin/auth.go:twoFADisable](internal/controller/http/admin/auth.go#L491-L550)

### 6.4 重置 recovery codes

- `POST /api/admin/auth/2fa/recovery/reset`
- 必须提供 `code` 或 `recovery_code` 之一；验证通过后重新生成并覆盖 recovery codes
- Handler：[admin/auth.go:twoFARecoveryReset](internal/controller/http/admin/auth.go#L564-L626)

## 七、登出

- `POST /api/admin/auth/logout`
- 行为：尝试 `sess.Destroy()`（若 session 不存在也返回成功）
- Handler：[admin/auth.go:logout](internal/controller/http/admin/auth.go#L255-L266)
