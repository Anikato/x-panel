package dto

// --- Host ---

type HostCreate struct {
	GroupID     uint   `json:"groupID"`
	Name        string `json:"name" binding:"required"`
	Addr        string `json:"addr" binding:"required"`
	Port        int    `json:"port" binding:"required"`
	User        string `json:"user" binding:"required"`
	AuthMode    string `json:"authMode" binding:"required,oneof=password key"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
	PassPhrase  string `json:"passPhrase"`
	Description string `json:"description"`
}

type HostUpdate struct {
	ID          uint   `json:"id" binding:"required"`
	GroupID     uint   `json:"groupID"`
	Name        string `json:"name" binding:"required"`
	Addr        string `json:"addr" binding:"required"`
	Port        int    `json:"port" binding:"required"`
	User        string `json:"user" binding:"required"`
	AuthMode    string `json:"authMode" binding:"required,oneof=password key"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
	PassPhrase  string `json:"passPhrase"`
	Description string `json:"description"`
}

type HostInfo struct {
	ID          uint   `json:"id"`
	GroupID     uint   `json:"groupID"`
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	AuthMode    string `json:"authMode"`
	Description string `json:"description"`
	GroupName   string `json:"groupName"`
}

type HostTree struct {
	ID       uint       `json:"id"`
	Label    string     `json:"label"`
	Children []HostTree `json:"children,omitempty"`
}

type SearchHostReq struct {
	PageInfo
	Info    string `json:"info"`
	GroupID uint   `json:"groupID"`
}

// --- Command ---

type CommandCreate struct {
	GroupID uint   `json:"groupID"`
	Name    string `json:"name" binding:"required"`
	Command string `json:"command" binding:"required"`
}

type CommandUpdate struct {
	ID      uint   `json:"id" binding:"required"`
	GroupID uint   `json:"groupID"`
	Name    string `json:"name" binding:"required"`
	Command string `json:"command" binding:"required"`
}

type CommandInfo struct {
	ID      uint   `json:"id"`
	GroupID uint   `json:"groupID"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

type CommandTree struct {
	ID       uint          `json:"id"`
	Label    string        `json:"label"`
	Value    string        `json:"value"`
	Children []CommandInfo `json:"children,omitempty"`
}

type SearchCommandReq struct {
	PageInfo
	Info    string `json:"info"`
	GroupID uint   `json:"groupID"`
}

// --- Group ---

type GroupCreate struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=host command"`
}

type GroupUpdate struct {
	ID   uint   `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type GroupInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type GroupSearch struct {
	Type string `json:"type" binding:"required,oneof=host command"`
}
