package response

import codes "github.com/scc749/nimbus-blog-api/internal/controller/http/bizcode"

// 通用业务状态码重导出
const (
	Success = codes.Success

	ErrorParam         = codes.ErrorParam
	ErrorParamMissing  = codes.ErrorParamMissing
	ErrorParamFormat   = codes.ErrorParamFormat
	ErrorDataNotFound  = codes.ErrorDataNotFound
	ErrorInvalidParams = codes.ErrorInvalidParams

	ErrorUnauthorized     = codes.ErrorUnauthorized
	ErrorTokenInvalid     = codes.ErrorTokenInvalid
	ErrorTokenExpired     = codes.ErrorTokenExpired
	ErrorPermissionDenied = codes.ErrorPermissionDenied
	ErrorLoginRequired    = codes.ErrorLoginRequired

	ErrorSystem          = codes.ErrorSystem
	ErrorDatabase        = codes.ErrorDatabase
	ErrorCache           = codes.ErrorCache
	ErrorThirdParty      = codes.ErrorThirdParty
	ErrorConfigNotLoaded = codes.ErrorConfigNotLoaded
)

// 认证模块 (01xx)
const (
	// 认证授权 (0120-0139)
	ErrorPasswordWrong    = "0120" // 密码错误
	ErrorVerificationCode = "0121" // 验证码错误
)

// 文件模块 (02xx)
const (
	// 操作相关 (0260-0279)
	ErrorGetFileURLFailed = "0260" // 获取文件 URL 失败
)

// 用户模块 (03xx)
const (
	// 数据相关 (0301-0319)
	ErrorUserNotFound = "0301" // 用户不存在
	ErrorEmailExists  = "0302" // 邮箱已存在
)

// 内容模块 (04xx)
const (
	// 数据相关 (0401-0419)
	ErrorPostNotFound = "0401" // 文章不存在

	// 操作相关 (0460-0479)
	ErrorListPostsFailed      = "0460" // 获取文章列表失败
	ErrorGetPostFailed        = "0461" // 获取文章详情失败
	ErrorLikePostFailed       = "0462" // 点赞文章失败
	ErrorUnlikePostFailed     = "0463" // 取消点赞文章失败
	ErrorListCategoriesFailed = "0464" // 获取分类列表失败
	ErrorListTagsFailed       = "0465" // 获取标签列表失败
)

// 评论模块 (05xx)
const (
	// 操作相关 (0560-0579)
	ErrorListCommentsFailed  = "0560" // 获取评论列表失败
	ErrorSubmitCommentFailed = "0561" // 提交评论失败
	ErrorLikeCommentFailed   = "0562" // 点赞评论失败
	ErrorUnlikeCommentFailed = "0563" // 取消点赞评论失败
	ErrorDeleteCommentFailed = "0564" // 删除评论失败
)

// 反馈模块 (06xx)
const (
	// 操作相关 (0660-0679)
	ErrorSubmitFeedbackFailed = "0660" // 提交反馈失败
)

// 友情链接模块 (07xx)
const (
	// 操作相关 (0760-0779)
	ErrorListLinksFailed = "0760" // 获取友链列表失败
)

// 设置模块 (08xx)
const (
	// 操作相关 (0860-0879)
	ErrorListSettingsFailed = "0860" // 获取公开设置列表失败
)

// 通知模块 (09xx)
const (
	// 操作相关 (0960-0979)
	ErrorListNotificationsFailed  = "0960" // 获取通知列表失败
	ErrorGetUnreadCountFailed     = "0961" // 获取未读数量失败
	ErrorMarkReadFailed           = "0962" // 标记已读失败
	ErrorMarkAllReadFailed        = "0963" // 全部已读失败
	ErrorDeleteNotificationFailed = "0964" // 删除通知失败
)
