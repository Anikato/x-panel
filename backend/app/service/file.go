package service

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type IFileService interface {
	ListFiles(req dto.FileSearchReq) (*dto.FileInfo, error)
	GetContent(req dto.FileContentReq) (*dto.FileContentResp, error)
	SaveContent(req dto.FileSaveReq) error
	Create(req dto.FileCreateReq) error
	Delete(req dto.FileDeleteReq) error
	BatchDelete(req dto.FileBatchDeleteReq) error
	Rename(req dto.FileRenameReq) error
	Move(req dto.FileMoveReq) error
	ChangeMode(req dto.FileModeReq) error
	ChangeOwner(req dto.FileChownReq) error
	Compress(req dto.FileCompressReq) error
	Decompress(req dto.FileDecompressReq) error
	Wget(req dto.FileWgetReq) error
	GetFileTree(req dto.FileTreeReq) ([]dto.FileTreeNode, error)
	GetUsersAndGroups() (*dto.UserGroupResp, error)
	GetDirSize(req dto.DirSizeReq) (*dto.DirSizeResp, error)
}

type FileService struct{}

func NewIFileService() IFileService {
	return &FileService{}
}

const maxReadSize = 10 * 1024 * 1024 // 10MB

// ===================== 路径安全 =====================

// protectedPaths 不可删除的系统关键路径
var protectedPaths = map[string]bool{
	"/":         true,
	"/root":     true,
	"/home":     true,
	"/etc":      true,
	"/usr":      true,
	"/var":      true,
	"/bin":      true,
	"/sbin":     true,
	"/lib":      true,
	"/lib64":    true,
	"/boot":     true,
	"/proc":     true,
	"/sys":      true,
	"/dev":      true,
	"/tmp":      true,
	"/run":      true,
	"/opt":      true,
	"/srv":      true,
	"/mnt":      true,
	"/media":    true,
	"/usr/bin":  true,
	"/usr/sbin": true,
	"/usr/lib":  true,
}

// isProtectedPath 检查是否为受保护路径
func isProtectedPath(path string) bool {
	cleanPath := filepath.Clean(path)
	return protectedPaths[cleanPath]
}

// invalidChars 文件名中不允许的字符
var invalidChars = []string{"\x00", "\n", "\r"}

// isInvalidChar 检查文件路径是否包含非法字符
func isInvalidChar(path string) bool {
	for _, ch := range invalidChars {
		if strings.Contains(path, ch) {
			return true
		}
	}
	// 检查路径分量中不允许的模式
	base := filepath.Base(path)
	if base == "." || base == ".." {
		return false // 这些是合法的
	}
	if strings.HasPrefix(base, " ") || strings.HasSuffix(base, " ") {
		return true // 前后空格不允许
	}
	return false
}

// ===================== 文件列表 =====================

// ListFiles 列出目录内容
func (s *FileService) ListFiles(req dto.FileSearchReq) (*dto.FileInfo, error) {
	cleanPath := filepath.Clean(req.Path)
	info, err := os.Lstat(cleanPath)
	if err != nil {
		return nil, buserr.New(constant.ErrFileNotExist)
	}
	if !info.IsDir() {
		return nil, buserr.New(constant.ErrFileNotDir)
	}

	var items []dto.FileInfo

	// 递归子目录搜索
	if req.Search != "" && req.ContainSub {
		items, err = searchRecursive(cleanPath, req.Search, req.ShowHidden)
		if err != nil {
			return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	} else {
		entries, err := os.ReadDir(cleanPath)
		if err != nil {
			return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}

		searchLower := strings.ToLower(req.Search)
		for _, entry := range entries {
			if !req.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			// 搜索过滤
			if searchLower != "" && !strings.Contains(strings.ToLower(entry.Name()), searchLower) {
				continue
			}
			fi, err := entry.Info()
			if err != nil {
				continue
			}
			fullPath := filepath.Join(cleanPath, entry.Name())
			items = append(items, buildFileInfo(fullPath, fi))
		}
	}

	// 排序：目录在前，文件在后
	sortBy := req.SortBy
	sortOrder := req.SortOrder
	if sortBy == "" {
		sortBy = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.Slice(items, func(i, j int) bool {
		// 目录始终在前
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		var less bool
		switch sortBy {
		case "size":
			less = items[i].Size < items[j].Size
		case "modTime":
			less = items[i].ModTime < items[j].ModTime
		default: // name
			less = strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	dirInfo := buildFileInfo(cleanPath, info)
	dirInfo.Items = items
	return &dirInfo, nil
}

// searchRecursive 使用 find 命令递归搜索子目录
func searchRecursive(rootPath, search string, showHidden bool) ([]dto.FileInfo, error) {
	args := []string{rootPath, "-iname", fmt.Sprintf("*%s*", search)}
	if !showHidden {
		// 排除隐藏文件和隐藏目录
		args = []string{rootPath, "-not", "-path", "*/.*", "-iname", fmt.Sprintf("*%s*", search)}
	}
	// 限制最多返回 1000 条结果，防止结果过多
	cmd := exec.Command("find", args...)
	output, err := cmd.Output()
	if err != nil {
		// find 命令可能因权限问题返回非零退出码，但仍有有效结果
		if len(output) == 0 {
			return nil, err
		}
	}

	var items []dto.FileInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == rootPath {
			continue // 跳过根目录本身
		}
		if count >= 1000 {
			break // 限制结果数量
		}
		fi, err := os.Lstat(line)
		if err != nil {
			continue
		}
		item := buildFileInfo(line, fi)
		// 递归搜索时显示相对路径作为名称
		relPath, _ := filepath.Rel(rootPath, line)
		if relPath != "" {
			item.Name = relPath
		}
		items = append(items, item)
		count++
	}
	return items, nil
}

// ===================== 文件内容 =====================

// GetContent 获取文件内容
func (s *FileService) GetContent(req dto.FileContentReq) (*dto.FileContentResp, error) {
	cleanPath := filepath.Clean(req.Path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, buserr.New(constant.ErrFileNotExist)
	}
	if info.IsDir() {
		return nil, buserr.New(constant.ErrFileIsDir)
	}
	if info.Size() > maxReadSize {
		return nil, buserr.New(constant.ErrFileTooLarge)
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	return &dto.FileContentResp{
		Content: string(content),
		Path:    cleanPath,
		Name:    filepath.Base(cleanPath),
	}, nil
}

// SaveContent 保存文件内容（保留原文件权限）
func (s *FileService) SaveContent(req dto.FileSaveReq) error {
	cleanPath := filepath.Clean(req.Path)

	// 读取原文件权限，如果文件存在则保留原权限
	var fileMode fs.FileMode = 0644
	if info, err := os.Stat(cleanPath); err == nil {
		fileMode = info.Mode()
	}

	if err := os.WriteFile(cleanPath, []byte(req.Content), fileMode); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("File saved: %s (mode preserved: %s)", cleanPath, fileMode)
	return nil
}

// ===================== 创建 =====================

// Create 创建文件或目录
func (s *FileService) Create(req dto.FileCreateReq) error {
	cleanPath := filepath.Clean(req.Path)

	if isInvalidChar(cleanPath) {
		return buserr.New(constant.ErrFileInvalidChar)
	}

	if _, err := os.Stat(cleanPath); err == nil {
		return buserr.New(constant.ErrRecordExist)
	}

	// 确定权限模式
	var mode fs.FileMode = 0755
	if req.Mode != "" {
		m, err := strconv.ParseUint(req.Mode, 8, 32)
		if err == nil {
			mode = fs.FileMode(m)
		}
	} else if !req.IsDir {
		// 文件默认继承父目录权限或使用 0644
		parentInfo, err := os.Stat(filepath.Dir(cleanPath))
		if err == nil {
			mode = parentInfo.Mode().Perm()
		} else {
			mode = 0644
		}
	}

	if req.IsDir {
		if err := os.MkdirAll(cleanPath, mode); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	} else {
		dir := filepath.Dir(cleanPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		f, err := os.OpenFile(cleanPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		f.Close()
	}
	global.LOG.Infof("File created: %s (isDir: %v, mode: %s)", cleanPath, req.IsDir, mode)
	return nil
}

// ===================== 删除 =====================

// Delete 删除文件或目录
func (s *FileService) Delete(req dto.FileDeleteReq) error {
	cleanPath := filepath.Clean(req.Path)
	if isProtectedPath(cleanPath) {
		return buserr.New(constant.ErrFileDeleteProtected)
	}
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}

	if err := os.RemoveAll(cleanPath); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("File deleted: %s", cleanPath)
	return nil
}

// BatchDelete 批量删除
func (s *FileService) BatchDelete(req dto.FileBatchDeleteReq) error {
	for _, p := range req.Paths {
		if err := s.Delete(dto.FileDeleteReq{Path: p}); err != nil {
			return err
		}
	}
	return nil
}

// ===================== 重命名 =====================

// Rename 重命名
func (s *FileService) Rename(req dto.FileRenameReq) error {
	oldPath := filepath.Clean(req.OldName)
	newPath := filepath.Clean(req.NewName)

	if isInvalidChar(newPath) {
		return buserr.New(constant.ErrFileInvalidChar)
	}

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}
	if _, err := os.Stat(newPath); err == nil {
		return buserr.New(constant.ErrRecordExist)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("File renamed: %s → %s", oldPath, newPath)
	return nil
}

// ===================== 移动/复制 =====================

// Move 移动或复制
func (s *FileService) Move(req dto.FileMoveReq) error {
	dstDir := filepath.Clean(req.DstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}

	for _, src := range req.SrcPaths {
		srcClean := filepath.Clean(src)
		dstClean := filepath.Join(dstDir, filepath.Base(srcClean))

		// 防止移动到自身内部
		if strings.HasPrefix(dstDir, srcClean+"/") || dstDir == srcClean {
			return buserr.WithDetail(constant.ErrInvalidParams, "cannot move to itself", nil)
		}

		// 目标已存在的冲突处理
		if _, err := os.Stat(dstClean); err == nil {
			if !req.Cover {
				return buserr.WithDetail(constant.ErrRecordExist, dstClean, nil)
			}
			// 覆盖模式：先删除目标
			if err := os.RemoveAll(dstClean); err != nil {
				return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
			}
		}

		if req.IsCopy {
			cmd := exec.Command("cp", "-rp", srcClean, dstClean)
			if output, err := cmd.CombinedOutput(); err != nil {
				return buserr.WithDetail(constant.ErrInternalServer, string(output), err)
			}
			global.LOG.Infof("File copied: %s → %s", srcClean, dstClean)
		} else {
			if err := os.Rename(srcClean, dstClean); err != nil {
				// 跨分区移动：先复制后删除
				cmd := exec.Command("cp", "-rp", srcClean, dstClean)
				if output, err2 := cmd.CombinedOutput(); err2 != nil {
					return buserr.WithDetail(constant.ErrInternalServer, string(output), err2)
				}
				if err2 := os.RemoveAll(srcClean); err2 != nil {
					global.LOG.Warnf("Failed to remove source after cross-device move: %s", err2.Error())
				}
			}
			global.LOG.Infof("File moved: %s → %s", srcClean, dstClean)
		}
	}
	return nil
}

// ===================== 权限修改 =====================

// ChangeMode 修改权限（支持递归）
func (s *FileService) ChangeMode(req dto.FileModeReq) error {
	cleanPath := filepath.Clean(req.Path)
	mode, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, "invalid mode", err)
	}
	fileMode := fs.FileMode(mode)

	if req.Sub {
		// 递归修改权限
		cmd := exec.Command("chmod", "-R", fmt.Sprintf("%04o", mode), cleanPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, string(output), err)
		}
	} else {
		if err := os.Chmod(cleanPath, fileMode); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}

	global.LOG.Infof("File mode changed: %s → %04o (recursive: %v)", cleanPath, mode, req.Sub)
	return nil
}

// ===================== 所有者修改 =====================

// ChangeOwner 修改文件所有者
func (s *FileService) ChangeOwner(req dto.FileChownReq) error {
	cleanPath := filepath.Clean(req.Path)
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}

	ownership := req.User + ":" + req.Group
	args := []string{ownership, cleanPath}
	if req.Sub {
		args = []string{"-R", ownership, cleanPath}
	}

	cmd := exec.Command("chown", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return buserr.WithDetail(constant.ErrFileChown, string(output), err)
	}

	global.LOG.Infof("File owner changed: %s → %s (recursive: %v)", cleanPath, ownership, req.Sub)
	return nil
}

// GetUsersAndGroups 获取系统可用用户和组列表
func (s *FileService) GetUsersAndGroups() (*dto.UserGroupResp, error) {
	// 读取有效用户组
	groupMap, err := getValidGroups()
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	// 读取有效用户
	users, groupSet, err := getValidUsers(groupMap)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	var groups []string
	for group := range groupSet {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	return &dto.UserGroupResp{
		Users:  users,
		Groups: groups,
	}, nil
}

// getValidGroups 读取 /etc/group 获取有效用户组
func getValidGroups() (map[string]bool, error) {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/group: %w", err)
	}
	defer f.Close()

	groupMap := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) < 3 {
			continue
		}
		groupName := parts[0]
		gid, _ := strconv.Atoi(parts[2])
		// root 和 GID >= 1000 的用户组
		if groupName == "root" || gid >= 1000 {
			groupMap[groupName] = true
		}
	}
	return groupMap, scanner.Err()
}

// getValidUsers 读取 /etc/passwd 获取有效用户
func getValidUsers(validGroups map[string]bool) ([]dto.UserInfo, map[string]struct{}, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open /etc/passwd: %w", err)
	}
	defer f.Close()

	var users []dto.UserInfo
	groupSet := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) < 4 {
			continue
		}
		username := parts[0]
		uid, _ := strconv.Atoi(parts[2])
		gid := parts[3]

		// 只要 root 和 UID >= 1000 的普通用户
		if username != "root" && uid < 1000 {
			continue
		}

		groupName := gid
		if g, err := user.LookupGroupId(gid); err == nil {
			groupName = g.Name
		}

		if !validGroups[groupName] {
			continue
		}

		users = append(users, dto.UserInfo{
			Username: username,
			Group:    groupName,
		})
		groupSet[groupName] = struct{}{}
	}
	return users, groupSet, scanner.Err()
}

// ===================== 压缩/解压 =====================

// Compress 压缩
func (s *FileService) Compress(req dto.FileCompressReq) error {
	dst := filepath.Join(filepath.Clean(req.Dst), req.Name)

	if isInvalidChar(dst) {
		return buserr.New(constant.ErrFileInvalidChar)
	}

	compressType := req.Type
	if compressType == "" {
		compressType = "tar.gz"
	}

	var cmd *exec.Cmd
	switch compressType {
	case "zip":
		// 使用相对路径压缩，避免解压时出现绝对路径
		relPaths := make([]string, 0, len(req.Paths))
		var workDir string
		for i, p := range req.Paths {
			cleanP := filepath.Clean(p)
			if i == 0 {
				workDir = filepath.Dir(cleanP)
			}
			relPaths = append(relPaths, filepath.Base(cleanP))
		}
		args := []string{"-r", dst}
		args = append(args, relPaths...)
		cmd = exec.Command("zip", args...)
		if workDir != "" {
			cmd.Dir = workDir
		}
	default: // tar.gz
		// 使用 -C dir basename 模式，确保压缩包内为相对路径
		args := []string{"-czf", dst}
		for _, p := range req.Paths {
			cleanP := filepath.Clean(p)
			args = append(args, "-C", filepath.Dir(cleanP), filepath.Base(cleanP))
		}
		cmd = exec.Command("tar", args...)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, string(output), err)
	}
	global.LOG.Infof("Files compressed to: %s", dst)
	return nil
}

// Decompress 解压
func (s *FileService) Decompress(req dto.FileDecompressReq) error {
	src := filepath.Clean(req.Path)
	dst := filepath.Clean(req.Dst)

	if err := os.MkdirAll(dst, 0755); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	var cmd *exec.Cmd
	archiveType := detectArchiveType(src)

	switch archiveType {
	case "zip":
		cmd = exec.Command("unzip", "-o", src, "-d", dst)
	case "7z":
		if _, err := exec.LookPath("7z"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "7z", nil)
		}
		cmd = exec.Command("7z", "x", "-y", "-o"+dst, src)
	case "rar":
		if _, err := exec.LookPath("unrar"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "unrar", nil)
		}
		cmd = exec.Command("unrar", "x", "-y", "-o+", src, dst+"/")
	default: // tar, tar.gz, tar.bz2, tar.xz, tgz
		cmd = exec.Command("tar", "-xf", src, "-C", dst)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, string(output), err)
	}
	global.LOG.Infof("File decompressed: %s → %s", src, dst)
	return nil
}

// detectArchiveType 检测压缩文件类型
func detectArchiveType(path string) string {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return "tar.gz"
	}
	if strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2") {
		return "tar.bz2"
	}
	if strings.HasSuffix(lower, ".tar.xz") || strings.HasSuffix(lower, ".txz") {
		return "tar.xz"
	}
	if strings.HasSuffix(lower, ".tar") {
		return "tar"
	}
	if strings.HasSuffix(lower, ".zip") {
		return "zip"
	}
	if strings.HasSuffix(lower, ".7z") {
		return "7z"
	}
	if strings.HasSuffix(lower, ".rar") {
		return "rar"
	}
	if strings.HasSuffix(lower, ".gz") {
		return "tar.gz" // 单独的 .gz 也用 tar 处理
	}
	if strings.HasSuffix(lower, ".bz2") {
		return "tar.bz2"
	}
	if strings.HasSuffix(lower, ".xz") {
		return "tar.xz"
	}
	return "tar" // 默认按 tar 处理
}

// ===================== 远程下载 =====================

// Wget 使用 wget 下载远程文件
func (s *FileService) Wget(req dto.FileWgetReq) error {
	dst := filepath.Clean(req.Path)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}

	cmd := exec.Command("wget", "-q", "-P", dst, req.URL)
	if output, err := cmd.CombinedOutput(); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, string(output), err)
	}
	global.LOG.Infof("File downloaded via wget: %s → %s", req.URL, dst)
	return nil
}

// ===================== 文件树 =====================

// GetFileTree 获取目录树（用于路径选择器）
func (s *FileService) GetFileTree(req dto.FileTreeReq) ([]dto.FileTreeNode, error) {
	cleanPath := filepath.Clean(req.Path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, buserr.New(constant.ErrFileNotExist)
	}
	if !info.IsDir() {
		return nil, buserr.New(constant.ErrFileNotDir)
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	var nodes []dto.FileTreeNode
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(cleanPath, entry.Name())
		node := dto.FileTreeNode{
			ID:   fullPath,
			Name: entry.Name(),
			Path: fullPath,
			IsDir: true,
		}
		// 浅层：只检查是否有子目录
		subEntries, err := os.ReadDir(fullPath)
		if err == nil {
			for _, sub := range subEntries {
				if sub.IsDir() && !strings.HasPrefix(sub.Name(), ".") {
					node.Children = []dto.FileTreeNode{} // 标记为可展开
					break
				}
			}
		}
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})

	return nodes, nil
}

// ===================== 目录大小 =====================

// GetDirSize 计算目录大小
func (s *FileService) GetDirSize(req dto.DirSizeReq) (*dto.DirSizeResp, error) {
	cleanPath := filepath.Clean(req.Path)
	if cleanPath == "/proc" || cleanPath == "/sys" || cleanPath == "/dev" {
		return &dto.DirSizeResp{Size: 0}, nil
	}

	cmd := exec.Command("du", "-sb", cleanPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, string(output), err)
	}

	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return &dto.DirSizeResp{Size: 0}, nil
	}
	size, _ := strconv.ParseInt(parts[0], 10, 64)
	return &dto.DirSizeResp{Size: size}, nil
}

// ===================== 辅助函数 =====================

// buildFileInfo 构建文件信息
func buildFileInfo(fullPath string, fi os.FileInfo) dto.FileInfo {
	info := dto.FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    fi.Mode().String(),
		ModTime: fi.ModTime().Format(time.RFC3339),
		IsDir:   fi.IsDir(),
		Path:    fullPath,
	}

	// 权限八进制字符串
	info.ModeNum = fmt.Sprintf("%04o", fi.Mode().Perm())

	// 文件扩展名
	if !fi.IsDir() {
		ext := filepath.Ext(fi.Name())
		if ext != "" {
			info.Extension = strings.TrimPrefix(ext, ".")
		}
	}

	// 检查是否为符号链接
	if fi.Mode()&os.ModeSymlink != 0 {
		info.IsSymlink = true
		if target, err := os.Readlink(fullPath); err == nil {
			info.LinkPath = target
		}
	}

	// 获取文件所有者信息（Unix 系统）
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		uid := fmt.Sprintf("%d", stat.Uid)
		gid := fmt.Sprintf("%d", stat.Gid)
		info.Uid = uid
		info.Gid = gid
		if u, err := user.LookupId(uid); err == nil {
			info.User = u.Username
		} else {
			info.User = uid
		}
		if g, err := user.LookupGroupId(gid); err == nil {
			info.Group = g.Name
		} else {
			info.Group = gid
		}
	}
	return info
}
