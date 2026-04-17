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

	// HAProxy
	ErrHAProxyNotInstalled     = "ErrHAProxyNotInstalled"
	ErrHAProxyAlreadyInstalled = "ErrHAProxyAlreadyInstalled"
	ErrHAProxyCheckFailed      = "ErrHAProxyCheckFailed"
	ErrHAProxyReloadFailed     = "ErrHAProxyReloadFailed"
	ErrHAProxyNameExist        = "ErrHAProxyNameExist"
	ErrHAProxyPortInUse        = "ErrHAProxyPortInUse"
	ErrHAProxyBackendHasRefs   = "ErrHAProxyBackendHasRefs"
	ErrHAProxySocketFailed     = "ErrHAProxySocketFailed"

	// 应用中心
	ErrAppStoreSyncing    = "ErrAppStoreSyncing"
	ErrDownloadAppList    = "ErrDownloadAppList"
	ErrParseAppList       = "ErrParseAppList"
	ErrDownloadTags       = "ErrDownloadTags"
	ErrParseTags          = "ErrParseTags"
	ErrAppNameExist       = "ErrAppNameExist"
	ErrPortInUse          = "ErrPortInUse"
	ErrNoAvailablePort    = "ErrNoAvailablePort"
	ErrCreateDir          = "ErrCreateDir"
	ErrParseCompose       = "ErrParseCompose"
	ErrInvalidCompose     = "ErrInvalidCompose"
	ErrDockerComposeUp    = "ErrDockerComposeUp"
	ErrDockerComposeStart = "ErrDockerComposeStart"
	ErrDockerComposeStop  = "ErrDockerComposeStop"
	ErrDockerComposeRestart = "ErrDockerComposeRestart"
	ErrDockerComposeDown  = "ErrDockerComposeDown"
	ErrNotImplemented     = "ErrNotImplemented"
	ErrCreateBackupDir    = "ErrCreateBackupDir"
	ErrBackupFailed       = "ErrBackupFailed"
	ErrRestoreFailed      = "ErrRestoreFailed"
	ErrContainerNotFound  = "ErrContainerNotFound"
	ErrGetContainerLogs   = "ErrGetContainerLogs"
	ErrBackupFileNotFound = "ErrBackupFileNotFound"
	ErrOpenBackup         = "ErrOpenBackup"
	ErrDecompressBackup   = "ErrDecompressBackup"
	ErrReadBackup         = "ErrReadBackup"
	ErrReadMeta           = "ErrReadMeta"
	ErrParseMeta          = "ErrParseMeta"
	ErrReadCompose        = "ErrReadCompose"
	ErrCopyData           = "ErrCopyData"
	ErrWriteFile          = "ErrWriteFile"
	ErrGenerateCompose    = "ErrGenerateCompose"
	ErrStartContainer     = "ErrStartContainer"
	// 导入功能新增错误码
	ErrInvalidBackupFormat = "ErrInvalidBackupFormat"
	ErrCreateApp          = "ErrCreateApp"
	ErrCreateAppDetail    = "ErrCreateAppDetail"
	ErrImportTaskRunning  = "ErrImportTaskRunning"
	ErrImportTaskNotFound = "ErrImportTaskNotFound"
	ErrCreateImportTask   = "ErrCreateImportTask"
)
