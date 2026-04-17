package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/global"
	"gopkg.in/yaml.v3"
)

type IAppImportService interface {
	ImportFromBackup(req dto.AppImportReq) error
}

type AppImportService struct {
	appRepo        repo.IAppRepo
	appDetailRepo  repo.IAppDetailRepo
	appInstallRepo repo.IAppInstallRepo
	installService IAppInstallService
}

func NewIAppImportService() IAppImportService {
	return &AppImportService{
		appRepo:        repo.NewIAppRepo(),
		appDetailRepo:  repo.NewIAppDetailRepo(),
		appInstallRepo: repo.NewIAppInstallRepo(),
		installService: NewIAppInstallService(),
	}
}

// 1Panel 备份包元数据结构
type PanelBackupMeta struct {
	AppName    string                 `json:"appName"`
	AppKey     string                 `json:"appKey"`
	Version    string                 `json:"version"`
	Params     map[string]interface{} `json:"params"`
	Env        map[string]string      `json:"env"`
	HttpPort   int                    `json:"httpPort"`
	HttpsPort  int                    `json:"httpsPort"`
	BackupTime string                 `json:"backupTime"`
}

// ImportFromBackup 从备份导入应用
func (s *AppImportService) ImportFromBackup(req dto.AppImportReq) error {
	// 1. 验证备份文件存在和格式
	if _, err := os.Stat(req.BackupPath); os.IsNotExist(err) {
		return buserr.New("ErrBackupFileNotFound")
	}
	
	// 验证文件格式
	if !strings.HasSuffix(strings.ToLower(req.BackupPath), ".tar.gz") {
		return buserr.New("ErrInvalidBackupFormat")
	}

	// 2. 创建临时目录
	tempDir := filepath.Join(global.CONF.System.DataDir, "temp", "import", req.Name)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return buserr.WithDetail("ErrCreateDir", err.Error(), err)
	}
	
	// 改进的临时目录清理
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			global.LOG.Warnf("Failed to clean temp dir %s: %v", tempDir, err)
		}
	}()

	// 3. 解压备份包
	if err := s.extractBackup(req.BackupPath, tempDir); err != nil {
		return err
	}

	// 4. 读取元数据
	meta, err := s.readBackupMeta(tempDir)
	if err != nil {
		return err
	}

	// 5. 检查名称是否已存在
	existing, _ := s.appInstallRepo.GetFirst(repo.WithByName(req.Name))
	if existing.ID > 0 {
		return buserr.New("ErrAppNameExist")
	}

	// 6. 尝试从应用商店匹配应用
	var app model.App
	var appDetail model.AppDetail
	var createdApp bool // 标记是否创建了临时记录，用于回滚
	
	appKey := req.AppKey
	if appKey == "" {
		appKey = meta.AppKey
	}
	
	version := req.Version
	if version == "" {
		version = meta.Version
	}

	// 尝试从数据库查找应用
	if appKey != "" {
		app, _ = s.appRepo.GetFirst(repo.WithByKey(appKey))
		if app.ID > 0 && version != "" {
			appDetail, _ = s.appDetailRepo.GetFirst(
				repo.WithByAppID(app.ID),
				repo.WithByVersion(version),
			)
		}
	}

	// 7. 如果找不到应用，创建一个临时应用记录
	if app.ID == 0 {
		app = model.App{
			Name:        meta.AppName,
			Key:         appKey,
			Type:        "imported",
			Status:      "enable",
			Description: "Imported from 1Panel backup",
		}
		if err := s.appRepo.Create(context.Background(), &app); err != nil {
			return buserr.WithDetail("ErrCreateApp", err.Error(), err)
		}
		createdApp = true
	}

	// 8. 如果找不到版本详情，创建一个临时版本记录
	if appDetail.ID == 0 {
		// 读取 docker-compose.yml
		composeContent, err := s.readDockerCompose(tempDir)
		if err != nil {
			// 回滚创建的应用记录
			if createdApp {
				s.appRepo.DeleteBy(context.Background(), repo.WithByID(app.ID))
			}
			return err
		}

		appDetail = model.AppDetail{
			AppID:         app.ID,
			Version:       version,
			DockerCompose: composeContent,
			Status:        "enable",
		}
		if err := s.appDetailRepo.Create(context.Background(), &appDetail); err != nil {
			// 回滚创建的应用记录
			if createdApp {
				s.appRepo.DeleteBy(context.Background(), repo.WithByID(app.ID))
			}
			return buserr.WithDetail("ErrCreateAppDetail", err.Error(), err)
		}
	}

	// 9. 转换环境变量（1Panel → X-Panel）
	convertedEnv := s.convertEnvVars(meta.Env)
	
	// 10. 分配端口（如果原端口被占用）
	httpPort, httpsPort, err := s.allocatePortsForImport(meta.HttpPort, meta.HttpsPort)
	if err != nil {
		return err
	}

	// 11. 创建安装目录
	installDir := filepath.Join(global.CONF.System.DataDir, "apps", app.Key, req.Name)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return buserr.WithDetail("ErrCreateDir", err.Error(), err)
	}

	// 12. 复制数据文件
	dataDir := filepath.Join(tempDir, "data")
	if _, err := os.Stat(dataDir); err == nil {
		if err := s.copyDir(dataDir, installDir); err != nil {
			return buserr.WithDetail("ErrCopyData", err.Error(), err)
		}
	}

	// 13. 生成容器名称
	containerName := fmt.Sprintf("xpanel-%s-%s", app.Key, req.Name)

	// 14. 处理 docker-compose.yml
	composeContent, err := s.processImportedCompose(
		appDetail.DockerCompose,
		req.Name,
		convertedEnv,
		installDir,
		httpPort,
		httpsPort,
	)
	if err != nil {
		return err
	}

	// 15. 写入 docker-compose.yml
	composePath := filepath.Join(installDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return buserr.WithDetail("ErrWriteFile", err.Error(), err)
	}

	// 16. 写入 .env 文件
	envPath := filepath.Join(installDir, ".env")
	envContent := s.generateEnvFile(convertedEnv)
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return buserr.WithDetail("ErrWriteFile", err.Error(), err)
	}

	// 17. 创建安装记录
	paramsJSON, _ := json.Marshal(meta.Params)
	envJSON, _ := json.Marshal(convertedEnv)

	install := model.AppInstall{
		Name:          req.Name,
		AppID:         app.ID,
		AppDetailID:   appDetail.ID,
		Version:       version,
		Status:        "stopped",
		Param:         string(paramsJSON),
		Env:           string(envJSON),
		ContainerName: containerName,
		HttpPort:      httpPort,
		HttpsPort:     httpsPort,
	}

	if err := s.appInstallRepo.Create(context.Background(), &install); err != nil {
		return err
	}

	// 18. 启动容器
	if err := s.installService.Start(install.ID); err != nil {
		// 启动失败不回滚，让用户手动处理
		install.Status = "error"
		install.Message = err.Error()
		s.appInstallRepo.Save(context.Background(), &install)
		return buserr.WithDetail("ErrStartContainer", err.Error(), err)
	}

	return nil
}

// extractBackup 解压备份包
func (s *AppImportService) extractBackup(backupPath, destDir string) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return buserr.WithDetail("ErrOpenBackup", err.Error(), err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return buserr.WithDetail("ErrDecompressBackup", err.Error(), err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return buserr.WithDetail("ErrReadBackup", err.Error(), err)
		}

		// 安全检查：防止路径遍历攻击
		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			global.LOG.Warnf("Skipping dangerous path in backup: %s", header.Name)
			continue
		}

		target := filepath.Join(destDir, cleanName)

		// 二次检查：确保目标路径在 destDir 内
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			global.LOG.Warnf("Skipping path outside destination: %s", target)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// readBackupMeta 读取备份元数据
func (s *AppImportService) readBackupMeta(tempDir string) (*PanelBackupMeta, error) {
	metaPath := filepath.Join(tempDir, "app.json")
	
	// 如果没有 app.json，尝试从 .env 和 docker-compose.yml 推断
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return s.inferMetaFromFiles(tempDir)
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, buserr.WithDetail("ErrReadMeta", err.Error(), err)
	}

	var meta PanelBackupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, buserr.WithDetail("ErrParseMeta", err.Error(), err)
	}

	return &meta, nil
}

// inferMetaFromFiles 从文件推断元数据
func (s *AppImportService) inferMetaFromFiles(tempDir string) (*PanelBackupMeta, error) {
	meta := &PanelBackupMeta{
		Env:    make(map[string]string),
		Params: make(map[string]interface{}),
	}

	// 读取 .env 文件
	envPath := filepath.Join(tempDir, ".env")
	if data, err := os.ReadFile(envPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				meta.Env[key] = value

				// 尝试提取端口
				if strings.Contains(key, "PORT_HTTP") && !strings.Contains(key, "HTTPS") {
					if port, err := parseInt(value); err == nil {
						meta.HttpPort = port
					}
				}
				if strings.Contains(key, "PORT_HTTPS") {
					if port, err := parseInt(value); err == nil {
						meta.HttpsPort = port
					}
				}
			}
		}
	}

	// 从 docker-compose.yml 推断应用名称
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if data, err := os.ReadFile(composePath); err == nil {
		var compose map[string]interface{}
		if err := yaml.Unmarshal(data, &compose); err == nil {
			if services, ok := compose["services"].(map[string]interface{}); ok {
				for serviceName := range services {
					meta.AppName = serviceName
					meta.AppKey = serviceName
					break
				}
			}
		}
	}

	if meta.AppName == "" {
		meta.AppName = "imported-app"
		meta.AppKey = "imported-app"
	}

	meta.Version = "latest"

	return meta, nil
}

// readDockerCompose 读取 docker-compose.yml
func (s *AppImportService) readDockerCompose(tempDir string) (string, error) {
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	data, err := os.ReadFile(composePath)
	if err != nil {
		return "", buserr.WithDetail("ErrReadCompose", err.Error(), err)
	}
	return string(data), nil
}

// convertEnvVars 转换环境变量（1Panel → X-Panel）
func (s *AppImportService) convertEnvVars(env map[string]string) map[string]string {
	converted := make(map[string]string)
	
	for key, value := range env {
		// 替换 PANEL_ 前缀为 XPANEL_
		if strings.HasPrefix(key, "PANEL_") {
			newKey := "XPANEL_" + strings.TrimPrefix(key, "PANEL_")
			converted[newKey] = value
		} else {
			converted[key] = value
		}
	}
	
	return converted
}

// allocatePortsForImport 为导入的应用分配端口
func (s *AppImportService) allocatePortsForImport(preferredHttp, preferredHttps int) (int, int, error) {
	// 检查首选端口是否可用
	httpPort := preferredHttp
	httpsPort := preferredHttps

	// 获取所有已使用的端口
	installs, err := s.appInstallRepo.GetBy()
	if err != nil {
		return 0, 0, err
	}

	usedPorts := make(map[int]bool)
	for _, install := range installs {
		if install.HttpPort > 0 {
			usedPorts[install.HttpPort] = true
		}
		if install.HttpsPort > 0 {
			usedPorts[install.HttpsPort] = true
		}
	}

	// 检查系统端口是否实际可用
	isPortAvailable := func(port int) bool {
		if port <= 0 {
			return false
		}
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return false
		}
		ln.Close()
		return true
	}

	// 如果首选 HTTP 端口被占用或不可用，分配新端口
	if httpPort > 0 && (usedPorts[httpPort] || !isPortAvailable(httpPort)) {
		httpPort = 0
		for port := 8000; port <= 9000; port++ {
			if !usedPorts[port] && isPortAvailable(port) {
				httpPort = port
				usedPorts[port] = true // 标记为已使用
				break
			}
		}
		if httpPort == 0 {
			return 0, 0, buserr.New("ErrNoAvailablePort")
		}
	} else if httpPort > 0 {
		usedPorts[httpPort] = true // 标记首选端口为已使用
	}

	// 如果首选 HTTPS 端口被占用或不可用，分配新端口
	if httpsPort > 0 && (usedPorts[httpsPort] || !isPortAvailable(httpsPort)) {
		httpsPort = 0
		for port := 8000; port <= 9000; port++ {
			if !usedPorts[port] && isPortAvailable(port) {
				httpsPort = port
				break
			}
		}
		if httpsPort == 0 {
			return 0, 0, buserr.New("ErrNoAvailablePort")
		}
	}

	return httpPort, httpsPort, nil
}

// processImportedCompose 处理导入的 docker-compose.yml
func (s *AppImportService) processImportedCompose(
	composeContent string,
	serviceName string,
	env map[string]string,
	installDir string,
	httpPort, httpsPort int,
) (string, error) {
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeContent), &compose); err != nil {
		return "", buserr.WithDetail("ErrParseCompose", err.Error(), err)
	}

	// 更新服务配置
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceKey, service := range services {
			if svc, ok := service.(map[string]interface{}); ok {
				// 更新容器名称，避免冲突
				containerName := fmt.Sprintf("xpanel-%s-%s", serviceKey, serviceName)
				svc["container_name"] = containerName

				// 合并环境变量（保留原有的，添加新的）
				existingEnv := make(map[string]string)
				if envList, ok := svc["environment"].([]interface{}); ok {
					// 处理数组格式的环境变量
					for _, envItem := range envList {
						if envStr, ok := envItem.(string); ok {
							if parts := strings.SplitN(envStr, "=", 2); len(parts) == 2 {
								existingEnv[parts[0]] = parts[1]
							}
						}
					}
				} else if envMap, ok := svc["environment"].(map[string]interface{}); ok {
					// 处理对象格式的环境变量
					for k, v := range envMap {
						if vStr, ok := v.(string); ok {
							existingEnv[k] = vStr
						}
					}
				}

				// 合并新的环境变量
				for k, v := range env {
					existingEnv[k] = v
				}
				svc["environment"] = existingEnv

				// 更新 volumes（替换路径）
				if volumes, ok := svc["volumes"].([]interface{}); ok {
					newVolumes := make([]interface{}, 0)
					for _, vol := range volumes {
						if volStr, ok := vol.(string); ok {
							// 替换路径中的占位符
							volStr = strings.ReplaceAll(volStr, "${PANEL_APP_DATA_DIR}", installDir)
							volStr = strings.ReplaceAll(volStr, "${XPANEL_APP_DATA_DIR}", installDir)
							// 替换 1Panel 常见路径
							volStr = strings.ReplaceAll(volStr, "/opt/1panel/apps", installDir)
							newVolumes = append(newVolumes, volStr)
						}
					}
					svc["volumes"] = newVolumes
				}

				// 更新端口映射
				if ports, ok := svc["ports"].([]interface{}); ok {
					newPorts := make([]interface{}, 0)
					for _, port := range ports {
						if portStr, ok := port.(string); ok {
							newPortStr := s.replacePortMapping(portStr, httpPort, httpsPort)
							newPorts = append(newPorts, newPortStr)
						}
					}
					svc["ports"] = newPorts
				}

				// 添加网络
				svc["networks"] = []string{"xpanel-network"}
			}
		}
	}

	// 添加网络配置
	if compose["networks"] == nil {
		compose["networks"] = make(map[string]interface{})
	}
	networks := compose["networks"].(map[string]interface{})
	networks["xpanel-network"] = map[string]interface{}{
		"external": true,
	}

	// 转换回 YAML
	data, err := yaml.Marshal(compose)
	if err != nil {
		return "", buserr.WithDetail("ErrGenerateCompose", err.Error(), err)
	}

	return string(data), nil
}

// replacePortMapping 替换端口映射中的端口号
func (s *AppImportService) replacePortMapping(portStr string, httpPort, httpsPort int) string {
	// 处理 "8080:80" 格式的端口映射
	if strings.Contains(portStr, ":") {
		parts := strings.Split(portStr, ":")
		if len(parts) >= 2 {
			containerPort := parts[1]
			
			// 如果是 HTTP 端口（80, 8080 等）
			if containerPort == "80" || containerPort == "8080" {
				if httpPort > 0 {
					return fmt.Sprintf("%d:%s", httpPort, containerPort)
				}
			}
			// 如果是 HTTPS 端口（443, 8443 等）
			if containerPort == "443" || containerPort == "8443" {
				if httpsPort > 0 {
					return fmt.Sprintf("%d:%s", httpsPort, containerPort)
				}
			}
			
			// 其他端口保持原样，但可能需要检查冲突
			return portStr
		}
	}
	
	return portStr
}

// generateEnvFile 生成 .env 文件内容
func (s *AppImportService) generateEnvFile(env map[string]string) string {
	var lines []string
	for key, value := range env {
		// 转义特殊字符
		escapedValue := s.escapeEnvValue(value)
		lines = append(lines, fmt.Sprintf("%s=%s", key, escapedValue))
	}
	return strings.Join(lines, "\n")
}

// escapeEnvValue 转义环境变量值中的特殊字符
func (s *AppImportService) escapeEnvValue(value string) string {
	// 如果包含空格、制表符、换行符或引号，需要用双引号包围并转义内部引号
	if strings.ContainsAny(value, " \t\n\"'") {
		// 转义双引号和反斜杠
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return value
}

// copyDir 递归复制目录
func (s *AppImportService) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return s.copyFile(path, dstPath)
	})
}

// copyFile 复制文件
func (s *AppImportService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
