package response

import codes "github.com/scc749/nimbus-blog-api/internal/controller/http/bizcode"

const (
	Success = codes.Success

	ErrorParam        = codes.ErrorParam
	ErrorParamMissing = codes.ErrorParamMissing
	ErrorParamFormat  = codes.ErrorParamFormat
	ErrorDataNotFound = codes.ErrorDataNotFound

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

// 管理员模块错误码 (11xx)
const (
	// 数据相关 (1101-1119)
	ErrorAdminNotFound = "1101" // 管理员不存在

	// 认证授权 (1120-1139)
	ErrorAdminPasswordWrong     = "1120" // 管理员密码错误
	ErrorAdminOTPWrong          = "1121" // OTP 验证码错误
	ErrorAdminRecoveryCodeWrong = "1122" // 恢复码错误
	ErrorAdminSessionMissing    = "1123" // 管理员会话缺失
	ErrorAdminTwoFANotEnabled   = "1124" // 管理员未开启二次验证

	// 业务逻辑 (1140-1159)
	ErrorAdminTwoFASetupFailed           = "1140" // 二次验证设置失败
	ErrorAdminRecoveryCodesPersistFailed = "1141" // 恢复码持久化失败
	ErrorAdminTwoFASetupNotFound         = "1142" // 二次验证配置不存在或已过期

	// 操作相关 (1160-1179)
	ErrorAdminPasswordChangeFailed = "1160" // 管理员修改密码失败
	ErrorAdminDisable2FAFailed     = "1161" // 禁用二次验证失败
	ErrorAdminUpdateProfileFailed  = "1162" // 管理员修改个人信息失败
)

// 文件模块 (12xx)
const (
	// 数据相关 (1201-1219)
	ErrorUploadTypeUnsupported  = "1201" // 不支持的上传类型
	ErrorContentTypeUnsupported = "1202" // 不支持的内容类型

	// 操作相关 (1260-1279)
	ErrorGenerateUploadURLFailed = "1260" // 生成上传链接失败
	ErrorDeleteObjectFailed      = "1261" // 删除对象失败
	ErrorSaveFileMetaFailed      = "1262" // 保存文件元数据失败
	ErrorBindResourceFailed      = "1263" // 绑定资源失败
	ErrorListFilesFailed         = "1264" // 查询文件列表失败
)

// 用户模块 (13xx)
const (
	// 操作相关 (1360-1379)
	ErrorListUsersFailed        = "1360" // 获取用户列表失败
	ErrorUpdateUserStatusFailed = "1361" // 更新用户状态失败
)

// 内容模块 (14xx)
const (
	// 数据相关 (1401-1419)
	ErrorPostNotFound = "1401" // 文章不存在

	// 操作相关 (1460-1479)
	ErrorCreatePostFailed     = "1460" // 创建文章失败
	ErrorUpdatePostFailed     = "1461" // 更新文章失败
	ErrorDeletePostFailed     = "1462" // 删除文章失败
	ErrorUpdatePostTagsFailed = "1463" // 更新文章标签失败
	ErrorCreateCategoryFailed = "1464" // 创建分类失败
	ErrorUpdateCategoryFailed = "1465" // 更新分类失败
	ErrorDeleteCategoryFailed = "1466" // 删除分类失败
	ErrorCreateTagFailed      = "1467" // 创建标签失败
	ErrorUpdateTagFailed      = "1468" // 更新标签失败
	ErrorDeleteTagFailed      = "1469" // 删除标签失败
	ErrorGenerateSlugFailed   = "1470" // 生成 Slug 失败
)

// 评论模块 (15xx)
const (
	// 操作相关 (1560-1579)
	ErrorListCommentsFailed        = "1560" // 获取评论列表失败
	ErrorUpdateCommentStatusFailed = "1561" // 更新评论状态失败
	ErrorDeleteCommentFailed       = "1562" // 删除评论失败
)

// 反馈模块 (16xx)
const (
	// 操作相关 (1660-1679)
	ErrorListFeedbacksFailed        = "1660" // 获取反馈列表失败
	ErrorGetFeedbackFailed          = "1661" // 获取反馈详情失败
	ErrorUpdateFeedbackStatusFailed = "1662" // 更新反馈状态失败
	ErrorDeleteFeedbackFailed       = "1663" // 删除反馈失败
)

// 友情链接模块 (17xx)
const (
	// 操作相关 (1760-1779)
	ErrorListLinksFailed  = "1760" // 获取友链列表失败
	ErrorCreateLinkFailed = "1761" // 创建友链失败
	ErrorUpdateLinkFailed = "1762" // 更新友链失败
	ErrorDeleteLinkFailed = "1763" // 删除友链失败
)

// 设置模块 (18xx)
const (
	// 操作相关 (1860-1879)
	ErrorListSettingsFailed  = "1860" // 获取设置列表失败
	ErrorGetSettingFailed    = "1861" // 获取设置详情失败
	ErrorUpsertSettingFailed = "1862" // 更新设置失败
)

// 通知模块 (19xx)
const (
	// 操作相关 (1960-1979)
	ErrorSendNotificationFailed = "1960" // 发送通知失败
)
