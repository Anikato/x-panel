package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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

// MoveFile 移动/复制文件（全部异步执行，返回任务 ID 或瞬时完成）
func (a *FileAPI) MoveFile(c *gin.Context) {
	var req dto.FileMoveReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()

	// 判断是否可以瞬时完成（同分区单文件 rename）
	if !req.IsCopy && len(req.SrcPaths) == 1 {
		srcClean := filepath.Clean(req.SrcPaths[0])
		dstClean := filepath.Join(filepath.Clean(req.DstPath), filepath.Base(srcClean))
		if err := os.Rename(srcClean, dstClean); err == nil {
			// 同分区 rename：瞬时完成，直接返回成功
			helper.SuccessWithOutData(c)
			return
		}
		// rename 失败（跨设备等原因）→ 走异步任务
	}

	// 其余情况（多文件、跨分区、复制）→ 异步执行（带进度）
	opType := "移动"
	if req.IsCopy {
		opType = "复制"
	}
	taskName := fmt.Sprintf("%s %d 个文件到 %s", opType, len(req.SrcPaths), filepath.Base(req.DstPath))

	// 预先统计总字节数（用于进度百分比）
	var totalBytes int64
	for _, src := range req.SrcPaths {
		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if info.IsDir() {
			totalBytes += service.CalcDirBytes(src)
		} else {
			totalBytes += info.Size()
		}
	}

	task := service.StartFileTaskWithProgress("move", taskName, totalBytes, func(tracker *service.ProgressTracker) error {
		return svc.MoveWithTracker(req, tracker)
	})
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
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

// CompressFile 压缩（异步执行）
func (a *FileAPI) CompressFile(c *gin.Context) {
	var req dto.FileCompressReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	taskName := fmt.Sprintf("压缩 %d 个文件 → %s", len(req.Paths), req.Name)
	task := service.StartFileTask("compress", taskName, func() error {
		return svc.Compress(req)
	})
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
}

// DecompressFile 解压（异步执行）
func (a *FileAPI) DecompressFile(c *gin.Context) {
	var req dto.FileDecompressReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	taskName := fmt.Sprintf("解压 %s → %s", filepath.Base(req.Path), filepath.Base(req.Dst))
	task := service.StartFileTask("decompress", taskName, func() error {
		return svc.Decompress(req)
	})
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
}

// ListArchive 压缩包内容预览
func (a *FileAPI) ListArchive(c *gin.Context) {
	var req dto.FileArchiveListReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	data, err := svc.ListArchive(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// GetFileTaskStatus 查询单个文件操作任务状态
func (a *FileAPI) GetFileTaskStatus(c *gin.Context) {
	taskID := c.Query("id")
	if taskID == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "task id is required")
		return
	}
	task := service.GetFileTask(taskID)
	if task == nil {
		helper.ErrorWithDetail(c, http.StatusNotFound, "task not found")
		return
	}
	helper.SuccessWithData(c, task)
}

// ListFileTasks 获取所有文件操作任务列表
func (a *FileAPI) ListFileTasks(c *gin.Context) {
	tasks := service.ListFileTasks()
	helper.SuccessWithData(c, tasks)
}

// CancelFileTask 取消支持取消的文件后台任务
func (a *FileAPI) CancelFileTask(c *gin.Context) {
	taskID := c.Query("id")
	if taskID == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "task id is required")
		return
	}
	if err := service.CancelFileTask(taskID); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessWithOutData(c)
}

// CheckConflict 检查移动/复制目标冲突
func (a *FileAPI) CheckConflict(c *gin.Context) {
	var req dto.FileMoveReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	conflicts := svc.CheckConflict(req.SrcPaths, req.DstPath)
	helper.SuccessWithData(c, map[string]interface{}{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// PreflightUpload validates exact relative paths and reports upload conflicts.
func (a *FileAPI) PreflightUpload(c *gin.Context) {
	var req dto.FileUploadPreflightReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	result, err := service.NewIFileService().PreflightUpload(req)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessWithData(c, result)
}

// WgetFile 远程下载文件（异步执行，返回 taskID）
func (a *FileAPI) WgetFile(c *gin.Context) {
	var req dto.FileWgetReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	svc := service.NewIFileService()
	ctx, cancel := context.WithCancel(context.Background())
	taskName := fmt.Sprintf("远程下载 %s", req.URL)
	task := service.StartFileTaskWithProgress("download", taskName, 0, func(tracker *service.ProgressTracker) error {
		return svc.WgetWithTracker(ctx, req, tracker)
	})
	service.RegisterFileTaskCancel(task.ID, cancel)
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
}

// UploadFile 上传文件（流式写盘，支持大文件）
func (a *FileAPI) UploadFile(c *gin.Context) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "failed to parse multipart: "+err.Error())
		return
	}

	var targetPath, relativePath string
	overwrite := true
	batch := false
	svc := service.NewIFileService()

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			helper.ErrorWithDetail(c, http.StatusBadRequest, "multipart error: "+err.Error())
			return
		}

		fieldName := part.FormName()
		if fieldName != "file" {
			value, readErr := io.ReadAll(io.LimitReader(part, 64*1024))
			part.Close()
			if readErr != nil {
				helper.ErrorWithDetail(c, http.StatusBadRequest, "failed to read multipart field: "+readErr.Error())
				return
			}
			switch fieldName {
			case "path":
				targetPath = string(value)
			case "relativePath":
				relativePath = string(value)
			case "overwrite":
				overwrite, err = strconv.ParseBool(string(value))
			case "batch":
				batch, err = strconv.ParseBool(string(value))
			}
			if err != nil {
				helper.ErrorWithDetail(c, http.StatusBadRequest, "invalid "+fieldName+": "+err.Error())
				return
			}
			continue
		}

		if targetPath == "" {
			part.Close()
			helper.ErrorWithDetail(c, http.StatusBadRequest, "path must be provided before file")
			return
		}
		if relativePath == "" {
			relativePath = part.FileName()
		}
		savedPath, saveErr := svc.SaveUpload(targetPath, relativePath, overwrite, part)
		part.Close()
		if saveErr != nil {
			switch {
			case errors.Is(saveErr, service.ErrUploadConflict):
				helper.ErrorWithDetail(c, http.StatusConflict, saveErr.Error())
			case errors.Is(saveErr, service.ErrInvalidUploadPath):
				helper.ErrorWithDetail(c, http.StatusBadRequest, saveErr.Error())
			default:
				helper.ErrorWithDetail(c, http.StatusInternalServerError, "upload failed: "+saveErr.Error())
			}
			return
		}

		if !batch {
			service.CreateNotification(dto.NotificationCreate{
				Type:      "success",
				Event:     "file.upload.completed",
				Title:     fmt.Sprintf("文件「%s」上传完成", filepath.Base(savedPath)),
				Content:   filepath.Join(filepath.Clean(targetPath), filepath.FromSlash(savedPath)),
				Source:    "file",
				TargetURL: "/host/files",
			})
		}
		helper.SuccessWithOutData(c)
		return
	}

	helper.ErrorWithDetail(c, http.StatusBadRequest, "file is required")
}

// UploadFileChunk streams one verified chunk into a root-confined temporary file.
func (a *FileAPI) UploadFileChunk(c *gin.Context) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "failed to parse multipart: "+err.Error())
		return
	}
	var req dto.FileUploadChunkReq
	for {
		part, partErr := mr.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			helper.ErrorWithDetail(c, http.StatusBadRequest, "multipart error: "+partErr.Error())
			return
		}
		fieldName := part.FormName()
		if fieldName != "file" {
			value, readErr := io.ReadAll(io.LimitReader(part, 64*1024))
			part.Close()
			if readErr != nil {
				helper.ErrorWithDetail(c, http.StatusBadRequest, "failed to read multipart field: "+readErr.Error())
				return
			}
			if err := assignUploadChunkField(&req, fieldName, string(value)); err != nil {
				helper.ErrorWithDetail(c, http.StatusBadRequest, "invalid "+fieldName+": "+err.Error())
				return
			}
			continue
		}

		if err := service.NewIFileService().SaveUploadChunk(req, part); err != nil {
			part.Close()
			handleFileUploadError(c, err)
			return
		}
		part.Close()
		helper.SuccessWithOutData(c)
		return
	}
	helper.ErrorWithDetail(c, http.StatusBadRequest, "file is required")
}

func assignUploadChunkField(req *dto.FileUploadChunkReq, name, value string) error {
	var err error
	switch name {
	case "path":
		req.TargetPath = value
	case "relativePath":
		req.RelativePath = value
	case "uploadID":
		req.UploadID = value
	case "checksum":
		req.Checksum = value
	case "chunkIndex":
		req.ChunkIndex, err = strconv.Atoi(value)
	case "chunkCount":
		req.ChunkCount, err = strconv.Atoi(value)
	case "totalSize":
		req.TotalSize, err = strconv.ParseInt(value, 10, 64)
	}
	return err
}

// CompleteFileChunks validates and atomically publishes a chunked upload.
func (a *FileAPI) CompleteFileChunks(c *gin.Context) {
	var req dto.FileUploadChunkCompleteReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	savedPath, err := service.NewIFileService().CompleteUploadChunks(req)
	if err != nil {
		handleFileUploadError(c, err)
		return
	}
	if !req.Batch {
		service.CreateNotification(dto.NotificationCreate{
			Type:      "success",
			Event:     "file.upload.completed",
			Title:     fmt.Sprintf("文件「%s」上传完成", filepath.Base(savedPath)),
			Content:   filepath.Join(filepath.Clean(req.TargetPath), filepath.FromSlash(savedPath)),
			Source:    "file",
			TargetURL: "/host/files",
		})
	}
	helper.SuccessWithOutData(c)
}

// AbortFileChunks removes an unfinished chunked upload if it still exists.
func (a *FileAPI) AbortFileChunks(c *gin.Context) {
	var req dto.FileUploadChunkAbortReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIFileService().AbortUploadChunks(req); err != nil {
		handleFileUploadError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func handleFileUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUploadConflict):
		helper.ErrorWithDetail(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidUploadPath),
		errors.Is(err, service.ErrInvalidUploadMetadata),
		errors.Is(err, service.ErrUploadChecksum),
		errors.Is(err, service.ErrUploadChunkOrder),
		errors.Is(err, service.ErrUploadSizeMismatch):
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
	default:
		helper.ErrorWithDetail(c, http.StatusInternalServerError, "upload failed: "+err.Error())
	}
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
