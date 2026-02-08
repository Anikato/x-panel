package constant

const (
	StatusEnable  = "Enable"
	StatusDisable = "Disable"
	StatusSuccess = "Success"
	StatusFailed  = "Failed"

	DateTimeLayout = "2006-01-02 15:04:05"
	DateLayout     = "2006-01-02"

	// JWT
	JWTHeaderKey   = "Authorization"
	JWTTokenPrefix = "Bearer "
	JWTIssuer      = "xpanel"

	// Default settings
	DefaultLanguage       = "zh"
	DefaultSessionTimeout = 86400 // 24h in seconds
	DefaultPanelName      = "X-Panel"
)
