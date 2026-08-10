package dto

// NezhaAgentConfigUpdate is the panel request for configuring the bundled Nezha Agent.
// Pointer fields allow partial updates: omitted or empty ClientSecret means leave unchanged.
type NezhaAgentConfigUpdate struct {
	DashboardURL            *string `json:"dashboardUrl"`
	ClientSecret            *string `json:"clientSecret"`
	RemoteOperationsEnabled *bool   `json:"remoteOperationsEnabled"`
	EnableAndStart          bool    `json:"enableAndStart"`
}

// NezhaAgentOperateRequest is the panel request for lifecycle operations on the
// bundled Nezha Agent. Allowed operations match the service Operate contract.
type NezhaAgentOperateRequest struct {
	Operation string `json:"operation" binding:"required,oneof=start stop restart enable disable"`
}

// NezhaAgentConflict describes an external Agent installation or unit that may
// conflict with the bundled xpanel-nezha-agent.
// Kind is one of: "unit", "process", "directory".
// Detail is the unit name, executable path, or directory path.
// Detection is read-only and never takes over external installs.
type NezhaAgentConflict struct {
	Kind    string `json:"kind"`
	Detail  string `json:"detail"`
	Message string `json:"message"`
}

// NezhaAgentStatus is the panel read model for the bundled Nezha Agent.
// AgentSecret is never returned; only SecretConfigured is exposed.
type NezhaAgentStatus struct {
	ComponentAvailable      bool                 `json:"componentAvailable"`
	Configured              bool                 `json:"configured"`
	ConfigHealthy           bool                 `json:"configHealthy"`
	ConfigError             string               `json:"configError"`
	Active                  bool                 `json:"active"`
	ServiceState            string               `json:"serviceState"`
	Enabled                 bool                 `json:"enabled"`
	DesiredEnabled          bool                 `json:"desiredEnabled"`
	Drift                   bool                 `json:"drift"`
	Version                 string               `json:"version"`
	UUID                    string               `json:"uuid"`
	DashboardURL            string               `json:"dashboardUrl"`
	Server                  string               `json:"server"`
	TLS                     bool                 `json:"tls"`
	InsecureTLS             bool                 `json:"insecureTls"`
	SecretConfigured        bool                 `json:"secretConfigured"`
	RemoteOperationsEnabled bool                 `json:"remoteOperationsEnabled"`
	PermissionsWarning      string               `json:"permissionsWarning"`
	ServiceError            string               `json:"serviceError"`
	Conflicts               []NezhaAgentConflict `json:"conflicts"`
}
