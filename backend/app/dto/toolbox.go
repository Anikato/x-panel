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
	HostsAllow string `json:"hostsAllow"`
	HostsDeny  string `json:"hostsDeny"`
}

type SambaShareCreate struct {
	Name       string `json:"name" validate:"required"`
	Path       string `json:"path" validate:"required"`
	Comment    string `json:"comment"`
	Writable   bool   `json:"writable"`
	GuestOK    bool   `json:"guestOK"`
	ValidUsers string `json:"validUsers"`
	HostsAllow string `json:"hostsAllow"`
	HostsDeny  string `json:"hostsDeny"`
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
	HostsAllow string `json:"hostsAllow"`
	HostsDeny  string `json:"hostsDeny"`
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
	MinProtocol string `json:"minProtocol"`
	MaxProtocol string `json:"maxProtocol"`
	HostsAllow  string `json:"hostsAllow"`
	HostsDeny   string `json:"hostsDeny"`
}

type SambaConnection struct {
	PID        string `json:"pid"`
	Username   string `json:"username"`
	Group      string `json:"group"`
	Machine    string `json:"machine"`
	Protocol   string `json:"protocol"`
	Encryption string `json:"encryption"`
}

type SambaShareUsage struct {
	Service     string `json:"service"`
	PID         string `json:"pid"`
	Machine     string `json:"machine"`
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

// ===== Fail2ban =====

type Fail2banJail struct {
	Name        string   `json:"name"`
	Enabled     bool     `json:"enabled"`
	Port        string   `json:"port"`
	Filter      string   `json:"filter"`
	LogPath     string   `json:"logPath"`
	MaxRetry    int      `json:"maxRetry"`
	FindTime    string   `json:"findTime"`
	BanTime     string   `json:"banTime"`
	Action      string   `json:"action"`
	BannedIPs   []string `json:"bannedIPs"`
	BannedCount int      `json:"bannedCount"`
}

type Fail2banJailUpdate struct {
	Name     string `json:"name" validate:"required"`
	Enabled  bool   `json:"enabled"`
	Port     string `json:"port"`
	MaxRetry int    `json:"maxRetry"`
	FindTime string `json:"findTime"`
	BanTime  string `json:"banTime"`
	Action   string `json:"action"`
}

type Fail2banSSHConfig struct {
	Enabled  bool   `json:"enabled"`
	Port     string `json:"port"`
	MaxRetry int    `json:"maxRetry" validate:"min=1,max=100"`
	FindTime string `json:"findTime" validate:"required"`
	BanTime  string `json:"banTime" validate:"required"`
}

type Fail2banBannedIP struct {
	IP          string `json:"ip"`
	Jail        string `json:"jail"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	Region      string `json:"region"`
	BannedAt    string `json:"bannedAt"`
}

type Fail2banBanReq struct {
	IP   string `json:"ip" validate:"required"`
	Jail string `json:"jail"`
}

type Fail2banUnbanReq struct {
	IP   string `json:"ip" validate:"required"`
	Jail string `json:"jail" validate:"required"`
}

// ===== IP Location =====

type IPBatchLookupReq struct {
	IPs []string `json:"ips" validate:"required,min=1"`
}

// ===== Systemd Service Manager =====

type SystemdServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"loadState"`
	ActiveState string `json:"activeState"`
	SubState    string `json:"subState"`
	Enabled     bool   `json:"enabled"`
	IsPanel     bool   `json:"isPanel"`
}

type SystemdServiceDetail struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	LoadState    string `json:"loadState"`
	ActiveState  string `json:"activeState"`
	SubState     string `json:"subState"`
	Enabled      bool   `json:"enabled"`
	IsPanel      bool   `json:"isPanel"`
	MainPID      int    `json:"mainPID"`
	ExecStart    string `json:"execStart"`
	WorkingDir   string `json:"workingDir"`
	User         string `json:"user"`
	Restart      string `json:"restart"`
	RestartSec   string `json:"restartSec"`
	Environment  string `json:"environment"`
	MemoryCurrent string `json:"memoryCurrent"`
	CPUUsage     string `json:"cpuUsage"`
	StartedAt    string `json:"startedAt"`
	UnitFile     string `json:"unitFile"`
	UnitContent  string `json:"unitContent"`
}

type SystemdServiceCreate struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	ExecStart   string `json:"execStart" validate:"required"`
	WorkingDir  string `json:"workingDir"`
	User        string `json:"user"`
	Restart     string `json:"restart"`
	RestartSec  int    `json:"restartSec"`
	Environment string `json:"environment"`
	AfterTarget string `json:"afterTarget"`
	AutoStart   bool   `json:"autoStart"`
}

type SystemdServiceUpdate struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	ExecStart   string `json:"execStart" validate:"required"`
	WorkingDir  string `json:"workingDir"`
	User        string `json:"user"`
	Restart     string `json:"restart"`
	RestartSec  int    `json:"restartSec"`
	Environment string `json:"environment"`
	AfterTarget string `json:"afterTarget"`
}

type SystemdServiceOperate struct {
	Name      string `json:"name" validate:"required"`
	Operation string `json:"operation" validate:"required,oneof=start stop restart enable disable"`
}

type SystemdServiceDelete struct {
	Name string `json:"name" validate:"required"`
}

type SystemdServiceLogReq struct {
	Name  string `json:"name" validate:"required"`
	Lines int    `json:"lines"`
}
