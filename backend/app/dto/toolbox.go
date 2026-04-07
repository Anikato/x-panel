package dto

// ===== 通用服务状态 =====

type ServiceStatus struct {
	IsInstalled bool   `json:"isInstalled"`
	IsRunning   bool   `json:"isRunning"`
	Version     string `json:"version"`
	AutoStart   bool   `json:"autoStart"`
}

type ServiceOperate struct {
	Operation string `json:"operation" validate:"required,oneof=start stop restart enable disable"`
}

// ===== Samba =====

type SambaShare struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Comment    string `json:"comment"`
	Writable   bool   `json:"writable"`
	GuestOK    bool   `json:"guestOK"`
	Browseable bool   `json:"browseable"`
	ValidUsers string `json:"validUsers"`
}

type SambaShareCreate struct {
	Name       string `json:"name" validate:"required"`
	Path       string `json:"path" validate:"required"`
	Comment    string `json:"comment"`
	Writable   bool   `json:"writable"`
	GuestOK    bool   `json:"guestOK"`
	ValidUsers string `json:"validUsers"`
	CreateDir  bool   `json:"createDir"`
}

type SambaShareUpdate struct {
	OrigName   string `json:"origName" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Path       string `json:"path" validate:"required"`
	Comment    string `json:"comment"`
	Writable   bool   `json:"writable"`
	GuestOK    bool   `json:"guestOK"`
	ValidUsers string `json:"validUsers"`
}

type SambaShareDelete struct {
	Name string `json:"name" validate:"required"`
}

type SambaUser struct {
	Username string `json:"username"`
	Flags    string `json:"flags"`
}

type SambaUserCreate struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type SambaUserDelete struct {
	Username string `json:"username" validate:"required"`
}

type SambaPasswordUpdate struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type SambaUserToggle struct {
	Username string `json:"username" validate:"required"`
	Enabled  bool   `json:"enabled"`
}

type SambaGlobalConfig struct {
	Workgroup   string `json:"workgroup"`
	ServerName  string `json:"serverName"`
	Security    string `json:"security"`
	MapToGuest  string `json:"mapToGuest"`
	LogLevel    string `json:"logLevel"`
	MaxLogSize  string `json:"maxLogSize"`
	Interfaces  string `json:"interfaces"`
}

type SambaConnection struct {
	PID       string `json:"pid"`
	Username  string `json:"username"`
	Group     string `json:"group"`
	Machine   string `json:"machine"`
	Protocol  string `json:"protocol"`
	Encryption string `json:"encryption"`
}

type SambaShareUsage struct {
	Service   string `json:"service"`
	PID       string `json:"pid"`
	Machine   string `json:"machine"`
	ConnectedAt string `json:"connectedAt"`
	Encryption  string `json:"encryption"`
}

type SambaConnections struct {
	Processes []SambaConnection `json:"processes"`
	Shares    []SambaShareUsage `json:"shares"`
}

// ===== NFS =====

type NfsExport struct {
	Path    string      `json:"path"`
	Clients []NfsClient `json:"clients"`
	Comment string      `json:"comment"`
}

type NfsClient struct {
	Host    string `json:"host" validate:"required"`
	Options string `json:"options" validate:"required"`
}

type NfsExportCreate struct {
	Path      string      `json:"path" validate:"required"`
	Clients   []NfsClient `json:"clients" validate:"required,min=1"`
	Comment   string      `json:"comment"`
	CreateDir bool        `json:"createDir"`
}

type NfsExportUpdate struct {
	OrigPath string      `json:"origPath" validate:"required"`
	Path     string      `json:"path" validate:"required"`
	Clients  []NfsClient `json:"clients" validate:"required,min=1"`
	Comment  string      `json:"comment"`
}

type NfsExportDelete struct {
	Path string `json:"path" validate:"required"`
}

type NfsConnectionInfo struct {
	Hostname string `json:"hostname"`
	DirPath  string `json:"dirPath"`
}

type NfsConnections struct {
	ActiveExports []string            `json:"activeExports"`
	Clients       []NfsConnectionInfo `json:"clients"`
}
