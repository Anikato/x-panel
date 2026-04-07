package constant

// 错误码定义：业务错误使用 i18n key
const (
	ErrInternalServer  = "ErrInternalServer"
	ErrInvalidParams   = "ErrInvalidParams"
	ErrRecordNotFound  = "ErrRecordNotFound"
	ErrRecordExist     = "ErrRecordExist"
	ErrCaptchaInvalid  = "ErrCaptchaInvalid"
	ErrAuth            = "ErrAuth"
	ErrTokenInvalid    = "ErrTokenInvalid"
	ErrTokenExpired    = "ErrTokenExpired"
	ErrNotLogin        = "ErrNotLogin"
	ErrPasswordWrong   = "ErrPasswordWrong"
	ErrUserNotFound    = "ErrUserNotFound"
	ErrInitialPassword = "ErrInitialPassword"

	// 文件管理
	ErrFileNotExist        = "ErrFileNotExist"
	ErrFileNotDir          = "ErrFileNotDir"
	ErrFileIsDir           = "ErrFileIsDir"
	ErrFileTooLarge        = "ErrFileTooLarge"
	ErrFileDeleteProtected = "ErrFileDeleteProtected"
	ErrFileInvalidChar     = "ErrFileInvalidChar"
	ErrFileChown           = "ErrFileChown"
	ErrFileCompress        = "ErrFileCompress"
	ErrFileDecompress      = "ErrFileDecompress"
	ErrCmdNotFound         = "ErrCmdNotFound"

	// SSL 证书
	ErrSSLAcmeRegister = "ErrSSLAcmeRegister"
	ErrSSLApply        = "ErrSSLApply"
	ErrSSLRenew        = "ErrSSLRenew"

	// Nginx
	ErrNginxNotInstalled    = "ErrNginxNotInstalled"
	ErrNginxAlreadyRunning  = "ErrNginxAlreadyRunning"
	ErrNginxAlreadyInstalled = "ErrNginxAlreadyInstalled"
	ErrNginxNotRunning      = "ErrNginxNotRunning"
	ErrNginxConfigTest      = "ErrNginxConfigTest"
	ErrNginxInstall         = "ErrNginxInstall"
	ErrNginxBuildDeps       = "ErrNginxBuildDeps"
	ErrNginxHasSites        = "ErrNginxHasSites"

	// Website
	ErrWebsiteDomainExist = "ErrWebsiteDomainExist"
	ErrWebsiteApplyConfig = "ErrWebsiteApplyConfig"
	ErrWebsiteNotFound    = "ErrWebsiteNotFound"

	// 升级
	ErrUpgradeInProgress = "ErrUpgradeInProgress"

	// GOST
	ErrGostNotInstalled     = "ErrGostNotInstalled"
	ErrGostAlreadyInstalled = "ErrGostAlreadyInstalled"
	ErrGostAPIUnavailable   = "ErrGostAPIUnavailable"
	ErrGostNameExist        = "ErrGostNameExist"
)
