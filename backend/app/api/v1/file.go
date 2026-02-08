package v1

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type FileAPI struct{}

// ListFiles 列出目录内容
func (a *FileAPI) ListFiles(c *gin.Context) {
	var req dto.FileSearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	data, err := svc.ListFiles(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// GetFileContent 获取文件内容
func (a *FileAPI) GetFileContent(c *gin.Context) {
	var req dto.FileContentReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	data, err := svc.GetContent(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// SaveFileContent 保存文件内容
func (a *FileAPI) SaveFileContent(c *gin.Context) {
	var req dto.FileSaveReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.SaveContent(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// CreateFile 创建文件/目录
func (a *FileAPI) CreateFile(c *gin.Context) {
	var req dto.FileCreateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// DeleteFile 删除文件/目录
func (a *FileAPI) DeleteFile(c *gin.Context) {
	var req dto.FileDeleteReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Delete(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// BatchDeleteFile 批量删除
func (a *FileAPI) BatchDeleteFile(c *gin.Context) {
	var req dto.FileBatchDeleteReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.BatchDelete(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// RenameFile 重命名
func (a *FileAPI) RenameFile(c *gin.Context) {
	var req dto.FileRenameReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Rename(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// MoveFile 移动/复制文件
func (a *FileAPI) MoveFile(c *gin.Context) {
	var req dto.FileMoveReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Move(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// ChangeMode 修改文件权限
func (a *FileAPI) ChangeMode(c *gin.Context) {
	var req dto.FileModeReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.ChangeMode(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// ChangeOwner 修改文件所有者
func (a *FileAPI) ChangeOwner(c *gin.Context) {
	var req dto.FileChownReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.ChangeOwner(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// GetUsersAndGroups 获取系统用户和组列表
func (a *FileAPI) GetUsersAndGroups(c *gin.Context) {
	svc := service.NewIFileService()
	data, err := svc.GetUsersAndGroups()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// GetFileTree 获取目录树
func (a *FileAPI) GetFileTree(c *gin.Context) {
	var req dto.FileTreeReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	data, err := svc.GetFileTree(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// GetDirSize 获取目录大小
func (a *FileAPI) GetDirSize(c *gin.Context) {
	var req dto.DirSizeReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	data, err := svc.GetDirSize(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// CompressFile 压缩
func (a *FileAPI) CompressFile(c *gin.Context) {
	var req dto.FileCompressReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Compress(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// DecompressFile 解压
func (a *FileAPI) DecompressFile(c *gin.Context) {
	var req dto.FileDecompressReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	if err := svc.Decompress(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// UploadFile 上传文件
func (a *FileAPI) UploadFile(c *gin.Context) {
	dstPath := c.PostForm("path")
	if dstPath == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "path is required")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "file is required")
		return
	}

	dst := filepath.Join(filepath.Clean(dstPath), file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// DownloadFile 下载文件
func (a *FileAPI) DownloadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "path is required")
		return
	}

	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusNotFound, "file not found")
		return
	}
	if info.IsDir() {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "cannot download directory")
		return
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(cleanPath))
	c.Header("Content-Type", "application/octet-stream")
	io.Copy(c.Writer, f)
}
