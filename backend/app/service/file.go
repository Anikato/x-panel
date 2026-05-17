package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
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
	MoveWithTracker(req dto.FileMoveReq, tracker *ProgressTracker) error
	ChangeMode(req dto.FileModeReq) error
	ChangeOwner(req dto.FileChownReq) error
	Compress(req dto.FileCompressReq) error
	Decompress(req dto.FileDecompressReq) error
	ListArchive(req dto.FileArchiveListReq) (*dto.FileArchiveListResp, error)
	Wget(req dto.FileWgetReq) error
	WgetWithTracker(ctx context.Context, req dto.FileWgetReq, tracker *ProgressTracker) error
	GetFileTree(req dto.FileTreeReq) ([]dto.FileTreeNode, error)
	GetUsersAndGroups() (*dto.UserGroupResp, error)
	GetDirSize(req dto.DirSizeReq) (*dto.DirSizeResp, error)
	CheckConflict(srcPaths []string, dstPath string) []string
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

// sanitizeFindPattern 转义 find -iname 中的特殊通配符
func sanitizeFindPattern(s string) string {
	r := strings.NewReplacer(
		"[", "\\[", "]", "\\]",
		"?", "\\?", "*", "\\*",
	)
	return r.Replace(s)
}

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
	safeName := sanitizeFindPattern(search)
	args := []string{rootPath, "-maxdepth", "10", "-iname", fmt.Sprintf("*%s*", safeName)}
	if !showHidden {
		args = []string{rootPath, "-maxdepth", "10", "-not", "-path", "*/.*", "-iname", fmt.Sprintf("*%s*", safeName)}
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

	if isProtectedPath(oldPath) {
		return buserr.New(constant.ErrFileDeleteProtected)
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

// Move 移动或复制
func (s *FileService) Move(req dto.FileMoveReq) error {
	return s.MoveWithTracker(req, nil)
}

func (s *FileService) MoveWithTracker(req dto.FileMoveReq, tracker *ProgressTracker) error {
	dstDir := filepath.Clean(req.DstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}

	policy := req.ConflictPolicy
	if policy == "" {
		if req.Cover {
			policy = "overwrite"
		} else {
			policy = "overwrite"
		}
	}

	for _, src := range req.SrcPaths {
		srcClean := filepath.Clean(src)
		dstClean := filepath.Join(dstDir, filepath.Base(srcClean))

		if strings.HasPrefix(dstDir, srcClean+"/") || dstDir == srcClean {
			return buserr.WithDetail(constant.ErrInvalidParams, "cannot move to itself", nil)
		}

		if _, err := os.Stat(dstClean); err == nil {
			switch policy {
			case "skip":
				global.LOG.Infof("File conflict skipped: %s", dstClean)
				continue
			default:
				if err := os.RemoveAll(dstClean); err != nil {
					return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
				}
			}
		}

		if !req.IsCopy {
			// 同分区：瞬间 rename，无需进度
			if err := os.Rename(srcClean, dstClean); err == nil {
				global.LOG.Infof("File moved (rename): %s → %s", srcClean, dstClean)
				continue
			}
			// 跨分区：回退到流式复制后删除
			if err := copyPathStreaming(srcClean, dstClean, tracker); err != nil {
				return err
			}
			if err := os.RemoveAll(srcClean); err != nil {
				global.LOG.Warnf("Failed to remove source after cross-device move: %s", err.Error())
			}
			global.LOG.Infof("File moved (cross-device): %s → %s", srcClean, dstClean)
		} else {
			if err := copyPathStreaming(srcClean, dstClean, tracker); err != nil {
				return err
			}
			global.LOG.Infof("File copied: %s → %s", srcClean, dstClean)
		}
	}
	return nil
}

// ===================== 流式复制工具 =====================

// calcDirBytes 递归统计目录总字节数（用于进度计算）
func calcDirBytes(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// copyPathStreaming 流式复制单个文件或目录，tracker 可为 nil（不追踪进度）
func copyPathStreaming(src, dst string, tracker *ProgressTracker) error {
	info, err := os.Lstat(src)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if info.IsDir() {
		return copyDirStreaming(src, dst, tracker)
	}
	return copyFileStreaming(src, dst, info, tracker)
}

// copyDirStreaming 递归复制目录
func copyDirStreaming(src, dst string, tracker *ProgressTracker) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirStreaming(srcPath, dstPath, tracker); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
			}
			if tracker != nil {
				tracker.task.CurrentFile = entry.Name()
			}
			if err := copyFileStreaming(srcPath, dstPath, info, tracker); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFileStreaming 流式复制单个文件，每 chunk 后更新 tracker
func copyFileStreaming(src, dst string, info fs.FileInfo, tracker *ProgressTracker) error {
	in, err := os.Open(src)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	defer out.Close()

	if tracker == nil {
		_, err = io.Copy(out, in)
	} else {
		buf := make([]byte, 256*1024) // 256KB chunks
		for {
			n, err := in.Read(buf)
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					return buserr.WithDetail(constant.ErrInternalServer, werr.Error(), werr)
				}
				tracker.AddBytes(int64(n))
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
			}
		}
	}
	if err != nil && err != io.EOF {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

// CheckConflict 检查目标目录中是否存在冲突文件
func (s *FileService) CheckConflict(srcPaths []string, dstPath string) []string {
	dstDir := filepath.Clean(dstPath)
	var conflicts []string
	for _, src := range srcPaths {
		srcClean := filepath.Clean(src)
		dstClean := filepath.Join(dstDir, filepath.Base(srcClean))
		if _, err := os.Stat(dstClean); err == nil {
			conflicts = append(conflicts, filepath.Base(srcClean))
		}
	}
	return conflicts
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

	// 读取用户
	users, _, err := getValidUsers(groupMap)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	var groups []string
	for group := range groupMap {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	return &dto.UserGroupResp{
		Users:  users,
		Groups: groups,
	}, nil
}

// getValidGroups 读取 /etc/group 获取用户组，包含系统组，便于 chown 到 www-data 等服务用户。
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
		if groupName == "" {
			continue
		}
		groupMap[groupName] = true
	}
	return groupMap, scanner.Err()
}

// getValidUsers 读取 /etc/passwd 获取用户，包含系统用户，便于文件所有者修复。
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
		if username == "" {
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
			Uid:      parts[2],
			Gid:      gid,
			System:   username != "root" && uid < 1000,
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

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return buserr.WithDetail(constant.ErrFileCompress, err.Error(), err)
	}

	absDst, _ := filepath.Abs(dst)

	for _, p := range req.Paths {
		cleanP := filepath.Clean(p)
		absP, _ := filepath.Abs(cleanP)
		if strings.HasPrefix(absDst, absP+"/") {
			return buserr.WithDetail(constant.ErrFileCompress, "output file cannot be inside source directory", nil)
		}
	}

	compressType := req.Type
	if compressType == "" {
		compressType = "tar.gz"
	}
	excludes := normalizeCompressExcludes(req.Excludes)

	var cmd *exec.Cmd
	switch compressType {
	case "zip":
		if _, err := exec.LookPath("zip"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "zip (apt install zip)", nil)
		}

		workDir := filepath.Dir(filepath.Clean(req.Paths[0]))

		sameParent := true
		for _, p := range req.Paths[1:] {
			if filepath.Dir(filepath.Clean(p)) != workDir {
				sameParent = false
				break
			}
		}

		if sameParent {
			relPaths := make([]string, 0, len(req.Paths))
			for _, p := range req.Paths {
				relPaths = append(relPaths, filepath.Base(filepath.Clean(p)))
			}
			args := []string{"-r", absDst}
			args = append(args, relPaths...)
			args = appendZipExcludeArgs(args, excludes)
			cmd = exec.Command("zip", args...)
			cmd.Dir = workDir
		} else {
			tmpDir, err := os.MkdirTemp("", "xpanel-zip-*")
			if err != nil {
				return buserr.WithDetail(constant.ErrFileCompress, err.Error(), err)
			}
			defer os.RemoveAll(tmpDir)

			relPaths := make([]string, 0, len(req.Paths))
			for _, p := range req.Paths {
				cleanP := filepath.Clean(p)
				base := filepath.Base(cleanP)
				linkDst := filepath.Join(tmpDir, base)
				if err := os.Symlink(cleanP, linkDst); err != nil {
					return buserr.WithDetail(constant.ErrFileCompress, err.Error(), err)
				}
				relPaths = append(relPaths, base)
			}
			args := []string{"-r", "--symlinks", absDst}
			args = append(args, relPaths...)
			args = appendZipExcludeArgs(args, excludes)
			cmd = exec.Command("zip", args...)
			cmd.Dir = tmpDir
		}
	default: // tar.gz
		args := []string{"-czf", absDst}
		for _, rule := range excludes {
			args = append(args, "--exclude", rule)
		}
		for _, p := range req.Paths {
			cleanP := filepath.Clean(p)
			args = append(args, "-C", filepath.Dir(cleanP), filepath.Base(cleanP))
		}
		cmd = exec.Command("tar", args...)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return buserr.WithDetail(constant.ErrFileCompress, errMsg, err)
	}
	global.LOG.Infof("Files compressed to: %s", dst)
	return nil
}

func normalizeCompressExcludes(excludes []string) []string {
	if len(excludes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(excludes))
	result := make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		rule := strings.TrimSpace(exclude)
		if rule == "" {
			continue
		}
		if _, ok := seen[rule]; ok {
			continue
		}
		seen[rule] = struct{}{}
		result = append(result, rule)
	}
	return result
}

func appendZipExcludeArgs(args []string, excludes []string) []string {
	if len(excludes) == 0 {
		return args
	}

	args = append(args, "-x")
	for _, rule := range excludes {
		args = append(args, rule)
	}
	return args
}

// Decompress 解压
func (s *FileService) Decompress(req dto.FileDecompressReq) error {
	src := filepath.Clean(req.Path)
	dst := filepath.Clean(req.Dst)
	if req.ExtractToSameDir {
		dst = filepath.Join(dst, archiveBaseName(src))
	}
	conflictPolicy := normalizeConflictPolicy(req.ConflictPolicy)

	if err := os.MkdirAll(dst, 0755); err != nil {
		return buserr.WithDetail(constant.ErrFileDecompress, err.Error(), err)
	}

	var cmd *exec.Cmd
	archiveType := detectArchiveType(src)

	switch archiveType {
	case "gz", "bz2", "xz":
		return s.decompressSingleFile(src, dst, archiveType, conflictPolicy)
	}

	if err := validateArchiveEntries(src, archiveType); err != nil {
		return buserr.WithDetail(constant.ErrFileDecompress, err.Error(), err)
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(dst), ".xpanel-decompress-*")
	if err != nil {
		return buserr.WithDetail(constant.ErrFileDecompress, err.Error(), err)
	}
	defer os.RemoveAll(tmpDir)

	switch archiveType {
	case "zip":
		if _, err := exec.LookPath("unzip"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "unzip (apt install unzip)", nil)
		}
		cmd = exec.Command("unzip", "-o", src, "-d", tmpDir)
	case "7z":
		if _, err := exec.LookPath("7z"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "7z (apt install p7zip-full)", nil)
		}
		cmd = exec.Command("7z", "x", "-y", "-o"+tmpDir, src)
	case "rar":
		if _, err := exec.LookPath("unrar"); err != nil {
			return buserr.WithDetail(constant.ErrCmdNotFound, "unrar (apt install unrar)", nil)
		}
		cmd = exec.Command("unrar", "x", "-y", "-o+", src, tmpDir+"/")
	default: // tar, tar.gz, tar.bz2, tar.xz, tgz
		cmd = exec.Command("tar", "-xf", src, "-C", tmpDir)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return buserr.WithDetail(constant.ErrFileDecompress, errMsg, err)
	}

	if err := mergeExtractedFiles(tmpDir, dst, conflictPolicy); err != nil {
		return buserr.WithDetail(constant.ErrFileDecompress, err.Error(), err)
	}
	global.LOG.Infof("File decompressed: %s → %s", src, dst)
	return nil
}

func (s *FileService) decompressSingleFile(src, dst, archiveType, conflictPolicy string) error {
	baseName := strings.TrimSuffix(filepath.Base(src), "."+archiveType)
	dstFile := resolveConflictPath(filepath.Join(dst, baseName), conflictPolicy)
	if dstFile == "" {
		return nil
	}

	var cmd *exec.Cmd
	switch archiveType {
	case "gz":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("gunzip -c %s > %s", shellQuote(src), shellQuote(dstFile)))
	case "bz2":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("bunzip2 -c %s > %s", shellQuote(src), shellQuote(dstFile)))
	case "xz":
		cmd = exec.Command("sh", "-c", fmt.Sprintf("xz -dc %s > %s", shellQuote(src), shellQuote(dstFile)))
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return buserr.WithDetail(constant.ErrFileDecompress, errMsg, err)
	}
	global.LOG.Infof("File decompressed: %s → %s", src, dstFile)
	return nil
}

func (s *FileService) ListArchive(req dto.FileArchiveListReq) (*dto.FileArchiveListResp, error) {
	src := filepath.Clean(req.Path)
	archiveType := detectArchiveType(src)
	if archiveType == "gz" || archiveType == "bz2" || archiveType == "xz" {
		name := strings.TrimSuffix(filepath.Base(src), "."+archiveType)
		return &dto.FileArchiveListResp{
			Entries: []string{name},
			Total:   1,
		}, nil
	}

	entries, err := listArchiveEntries(src, archiveType)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrFileDecompress, err.Error(), err)
	}

	unsafeEntries := make([]string, 0)
	for _, entry := range entries {
		if !isSafeArchiveEntry(entry) {
			unsafeEntries = append(unsafeEntries, entry)
		}
	}

	const maxPreviewEntries = 300
	previewEntries := entries
	if len(previewEntries) > maxPreviewEntries {
		previewEntries = previewEntries[:maxPreviewEntries]
	}

	return &dto.FileArchiveListResp{
		Entries:       previewEntries,
		Total:         len(entries),
		UnsafeEntries: unsafeEntries,
	}, nil
}

func normalizeConflictPolicy(policy string) string {
	normalized := strings.ToLower(strings.TrimSpace(policy))
	switch normalized {
	case "skip", "rename":
		return normalized
	default:
		return "overwrite"
	}
}

func archiveBaseName(src string) string {
	name := filepath.Base(src)
	lower := strings.ToLower(name)
	for _, suffix := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tgz", ".tbz2", ".txz", ".zip", ".7z", ".rar", ".tar", ".gz", ".bz2", ".xz"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSuffix(name, name[len(name)-len(suffix):])
		}
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func validateArchiveEntries(src, archiveType string) error {
	entries, err := listArchiveEntries(src, archiveType)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !isSafeArchiveEntry(entry) {
			return fmt.Errorf("unsafe archive path: %s", entry)
		}
	}
	return nil
}

func listArchiveEntries(src, archiveType string) ([]string, error) {
	var cmd *exec.Cmd
	switch archiveType {
	case "zip":
		if _, err := exec.LookPath("unzip"); err != nil {
			return nil, buserr.WithDetail(constant.ErrCmdNotFound, "unzip (apt install unzip)", nil)
		}
		cmd = exec.Command("unzip", "-Z1", src)
	case "7z":
		if _, err := exec.LookPath("7z"); err != nil {
			return nil, buserr.WithDetail(constant.ErrCmdNotFound, "7z (apt install p7zip-full)", nil)
		}
		cmd = exec.Command("7z", "l", "-slt", src)
	case "rar":
		if _, err := exec.LookPath("unrar"); err != nil {
			return nil, buserr.WithDetail(constant.ErrCmdNotFound, "unrar (apt install unrar)", nil)
		}
		cmd = exec.Command("unrar", "lb", src)
	default:
		cmd = exec.Command("tar", "-tf", src)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	lines := strings.Split(string(output), "\n")
	entries := make([]string, 0, len(lines))
	for _, line := range lines {
		item := strings.TrimSpace(line)
		if item == "" {
			continue
		}
		if archiveType == "7z" {
			if !strings.HasPrefix(item, "Path = ") {
				continue
			}
			item = strings.TrimSpace(strings.TrimPrefix(item, "Path = "))
			if item == filepath.Base(src) {
				continue
			}
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func isSafeArchiveEntry(entry string) bool {
	entry = strings.ReplaceAll(entry, "\\", "/")
	if entry == "" || strings.Contains(entry, "\x00") || path.IsAbs(entry) {
		return false
	}
	clean := path.Clean(entry)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func mergeExtractedFiles(srcDir, dstDir, conflictPolicy string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := mergeExtractedPath(filepath.Join(srcDir, entry.Name()), filepath.Join(dstDir, entry.Name()), conflictPolicy); err != nil {
			return err
		}
	}
	return nil
}

func mergeExtractedPath(src, dst, conflictPolicy string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := os.Rename(src, dst); err == nil {
				return nil
			}
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		for _, entry := range entries {
			if err := mergeExtractedPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), conflictPolicy); err != nil {
				return err
			}
		}
		return os.Remove(src)
	}

	target := resolveConflictPath(dst, conflictPolicy)
	if target == "" {
		return nil
	}
	if conflictPolicy == "overwrite" {
		if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	if err := os.Rename(src, target); err == nil {
		return nil
	}
	if err := copyPathStreaming(src, target, nil); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func resolveConflictPath(dst, conflictPolicy string) string {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst
	}

	switch conflictPolicy {
	case "skip":
		return ""
	case "rename":
		return nextAvailablePath(dst)
	default:
		return dst
	}
}

func nextAvailablePath(dst string) string {
	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s(%d)%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// shellQuote 对路径进行安全引用
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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
		return "gz"
	}
	if strings.HasSuffix(lower, ".bz2") {
		return "bz2"
	}
	if strings.HasSuffix(lower, ".xz") {
		return "xz"
	}
	return "tar" // 默认按 tar 处理
}

// ===================== 远程下载 =====================

type progressReader struct {
	reader  io.Reader
	tracker *ProgressTracker
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.tracker != nil {
		r.tracker.AddBytes(int64(n))
	}
	return n, err
}

// Wget 下载远程文件。
func (s *FileService) Wget(req dto.FileWgetReq) error {
	return s.WgetWithTracker(context.Background(), req, nil)
}

// WgetWithTracker 流式下载远程文件并更新任务进度。
func (s *FileService) WgetWithTracker(ctx context.Context, req dto.FileWgetReq, tracker *ProgressTracker) error {
	dst := filepath.Clean(req.Path)
	info, err := os.Stat(dst)
	if os.IsNotExist(err) {
		return buserr.New(constant.ErrFileNotExist)
	}
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if !info.IsDir() {
		return buserr.New(constant.ErrFileNotDir)
	}

	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return buserr.WithDetail(constant.ErrInvalidParams, "仅支持 http/https 下载地址", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, "下载地址无效", err)
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return buserr.WithDetail(constant.ErrInternalServer, "下载请求失败: "+err.Error(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return buserr.WithDetail(constant.ErrInternalServer, fmt.Sprintf("下载失败，HTTP 状态码: %d", resp.StatusCode), nil)
	}

	fileName := downloadFileName(resp, parsedURL)
	if tracker != nil {
		tracker.SetCurrentFile(fileName)
		if resp.ContentLength > 0 {
			tracker.SetTotal(resp.ContentLength)
		}
	}

	targetPath := uniqueDownloadPath(dst, fileName)
	tmpPath := filepath.Join(dst, fmt.Sprintf(".%s.xpanel-download-%d", filepath.Base(targetPath), time.Now().UnixNano()))
	out, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, "创建下载临时文件失败: "+err.Error(), err)
	}

	copyErr := func() error {
		defer out.Close()
		reader := io.Reader(resp.Body)
		if tracker != nil {
			reader = &progressReader{reader: resp.Body, tracker: tracker}
		}
		buf := make([]byte, 32*1024)
		_, err := io.CopyBuffer(out, reader, buf)
		return err
	}()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return buserr.WithDetail(constant.ErrInternalServer, "写入下载文件失败: "+copyErr.Error(), copyErr)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return buserr.WithDetail(constant.ErrInternalServer, "保存下载文件失败: "+err.Error(), err)
	}
	global.LOG.Infof("File downloaded: %s → %s", req.URL, targetPath)
	return nil
}

func downloadFileName(resp *http.Response, fallbackURL *url.URL) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if name := strings.TrimSpace(params["filename"]); name != "" {
				return safeDownloadFileName(name)
			}
			if name := strings.TrimSpace(params["filename*"]); name != "" {
				return safeDownloadFileName(name)
			}
		}
	}
	if resp.Request != nil && resp.Request.URL != nil {
		if name := path.Base(resp.Request.URL.Path); name != "." && name != "/" {
			return safeDownloadFileName(name)
		}
	}
	if fallbackURL != nil {
		if name := path.Base(fallbackURL.Path); name != "." && name != "/" {
			return safeDownloadFileName(name)
		}
	}
	return fmt.Sprintf("download-%d", time.Now().Unix())
}

func safeDownloadFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return fmt.Sprintf("download-%d", time.Now().Unix())
	}
	name = strings.NewReplacer("\x00", "", "\n", "", "\r", "").Replace(name)
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("download-%d", time.Now().Unix())
	}
	return name
}

func uniqueDownloadPath(dir, fileName string) string {
	target := filepath.Join(dir, fileName)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target
	}
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
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
			ID:    fullPath,
			Name:  entry.Name(),
			Path:  fullPath,
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
