package web

import (
	"embed"
	"io/fs"
)

//go:embed assets
var Assets embed.FS

// GetFS 返回 assets 子目录的文件系统
// 前端构建产物放在 assets/ 目录中
func GetFS() (fs.FS, error) {
	return fs.Sub(Assets, "assets")
}
