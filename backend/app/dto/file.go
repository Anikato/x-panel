package dto

// FileInfo 文件信息
type FileInfo struct {
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	Mode      string     `json:"mode"`
	ModeNum   string     `json:"modeNum"`
	ModTime   string     `json:"modTime"`
	IsDir     bool       `json:"isDir"`
	IsSymlink bool       `json:"isSymlink"`
	LinkPath  string     `json:"linkPath,omitempty"`
	User      string     `json:"user"`
	Group     string     `json:"group"`
	Uid       string     `json:"uid"`
	Gid       string     `json:"gid"`
	Path      string     `json:"path"`
	Extension string     `json:"extension,omitempty"`
	MimeType  string     `json:"mimeType,omitempty"`
	Items     []FileInfo `json:"items,omitempty"`
}

// FileSearchReq 文件列表请求
type FileSearchReq struct {
	Path       string `json:"path" binding:"required"`
	ShowHidden bool   `json:"showHidden"`
	Search     string `json:"search"`
	SortBy     string `json:"sortBy"`
	SortOrder  string `json:"sortOrder"`
}

// FileCreateReq 创建文件/目录
type FileCreateReq struct {
	Path  string `json:"path" binding:"required"`
	IsDir bool   `json:"isDir"`
	Mode  string `json:"mode"`
}

// FileDeleteReq 删除文件/目录
type FileDeleteReq struct {
	Path  string `json:"path" binding:"required"`
	Force bool   `json:"force"`
}

// FileBatchDeleteReq 批量删除
type FileBatchDeleteReq struct {
	Paths []string `json:"paths" binding:"required"`
}

// FileRenameReq 重命名
type FileRenameReq struct {
	OldName string `json:"oldName" binding:"required"`
	NewName string `json:"newName" binding:"required"`
}

// FileMoveReq 移动/复制
type FileMoveReq struct {
	SrcPaths []string `json:"srcPaths" binding:"required"`
	DstPath  string   `json:"dstPath" binding:"required"`
	IsCopy   bool     `json:"isCopy"`
	Cover    bool     `json:"cover"`
}

// FileContentReq 获取文件内容
type FileContentReq struct {
	Path string `json:"path" binding:"required"`
}

// FileContentResp 文件内容响应
type FileContentResp struct {
	Content string `json:"content"`
	Path    string `json:"path"`
	Name    string `json:"name"`
}

// FileSaveReq 保存文件内容
type FileSaveReq struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

// FileModeReq 修改权限
type FileModeReq struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
	Sub  bool   `json:"sub"`
}

// FileChownReq 修改所有者
type FileChownReq struct {
	Path  string `json:"path" binding:"required"`
	User  string `json:"user" binding:"required"`
	Group string `json:"group" binding:"required"`
	Sub   bool   `json:"sub"`
}

// FileCompressReq 压缩请求
type FileCompressReq struct {
	Paths []string `json:"paths" binding:"required"`
	Dst   string   `json:"dst" binding:"required"`
	Name  string   `json:"name" binding:"required"`
	Type  string   `json:"type"`
}

// FileDecompressReq 解压请求
type FileDecompressReq struct {
	Path string `json:"path" binding:"required"`
	Dst  string `json:"dst" binding:"required"`
}

// FileTreeReq 文件树请求
type FileTreeReq struct {
	Path string `json:"path" binding:"required"`
}

// FileTreeNode 文件树节点
type FileTreeNode struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	IsDir    bool           `json:"isDir"`
	Children []FileTreeNode `json:"children,omitempty"`
}

// UserInfo 系统用户信息
type UserInfo struct {
	Username string `json:"username"`
	Group    string `json:"group"`
}

// UserGroupResp 用户和组列表响应
type UserGroupResp struct {
	Users  []UserInfo `json:"users"`
	Groups []string   `json:"groups"`
}

// DirSizeReq 目录大小请求
type DirSizeReq struct {
	Path string `json:"path" binding:"required"`
}

// DirSizeResp 目录大小响应
type DirSizeResp struct {
	Size int64 `json:"size"`
}
