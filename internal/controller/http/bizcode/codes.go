package bizcode

// 业务状态码定义（AABB 四位字符串格式）
// AA: 模块编号 (00=全局)
// BB: 具体错误，按类型分组:
//
//	01-19: 数据相关错误
//	20-39: 认证授权错误
//	40-59: 业务逻辑错误
//	60-79: 系统错误/操作相关
//	80-99: 预留扩展
const (
	// 成功
	Success = "0000"

	// 全局错误码 - 数据相关 (0001-0019)
	ErrorParam         = "0001" // 参数错误
	ErrorParamMissing  = "0002" // 参数缺失
	ErrorParamFormat   = "0003" // 参数格式错误
	ErrorDataNotFound  = "0004" // 数据不存在
	ErrorInvalidParams = "0006" // 无效参数

	// 全局错误码 - 认证授权 (0020-0039)
	ErrorUnauthorized     = "0020" // 未授权
	ErrorTokenInvalid     = "0021" // Token无效
	ErrorTokenExpired     = "0022" // Token已过期
	ErrorPermissionDenied = "0023" // 权限不足
	ErrorLoginRequired    = "0024" // 请先登录

	// 全局错误码 - 系统错误 (0060-0079)
	ErrorSystem          = "0060" // 系统错误
	ErrorDatabase        = "0061" // 数据库错误
	ErrorCache           = "0062" // 缓存错误
	ErrorThirdParty      = "0064" // 第三方服务错误
	ErrorConfigNotLoaded = "0065" // 配置未加载
)
