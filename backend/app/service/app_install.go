package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/global"
	"xpanel/utils/cmd"
	"xpanel/utils/docker"
	"gopkg.in/yaml.v3"
)

type IAppInstallService interface {
	// 应用安装
	Install(req dto.AppInstallReq) error
	
	// 已安装应用查询
	PageInstalled(req dto.AppInstallSearchReq) (int64, []dto.AppInstallDTO, error)
	GetInstalled(id uint) (*dto.AppInstallDTO, error)
	
	// 应用操作
	Start(installID uint) error
	Stop(installID uint) error
	Restart(installID uint) error
	Uninstall(req dto.AppUninstallReq) error
	
	// 应用更新
	Update(req dto.AppUpdateReq) error
	
	// 容器日志
	GetLogs(installID uint, lines int) (string, error)
}

type AppInstallService struct {
	appRepo        repo.IAppRepo
	appDetailRepo  repo.IAppDetailRepo
	appInstallRepo repo.IAppInstallRepo
}

func NewIAppInstallService() IAppInstallService {
	return &AppInstallService{
		appRepo:        repo.NewIAppRepo(),
		appDetailRepo:  repo.NewIAppDetailRepo(),
		appInstallRepo: repo.NewIAppInstallRepo(),
	}
}

// Install 安装应用
func (s *AppInstallService) Install(req dto.AppInstallReq) error {
	ctx := context.Background()

	// 1. 检查名称是否已存在
	existing, _ := s.appInstallRepo.GetFirst(repo.WithByName(req.Name))
	if existing.ID > 0 {
		return buserr.New("ErrAppNameExist")
	}

	// 2. 获取应用详情
	appDetail, err := s.appDetailRepo.GetFirst(repo.WithByID(req.AppDetailID))
	if err != nil {
		return err
	}

	app, err := s.appRepo.GetFirst(repo.WithByID(appDetail.AppID))
	if err != nil {
		return err
	}

	// 3. 分配端口
	httpPort, httpsPort, err := s.allocatePorts(req.Params)
	if err != nil {
		return err
	}

	// 4. 生成容器名称
	containerName := fmt.Sprintf("xpanel-%s-%s", app.Key, req.Name)
	serviceName := req.Name

	// 5. 创建安装目录
	installDir := filepath.Join(global.CONF.System.DataDir, "apps", app.Key, req.Name)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return buserr.WithDetail("ErrCreateDir", err.Error(), err)
	}

	// 6. 生成环境变量
	envVars := s.generateEnvVars(req.Params, containerName, httpPort, httpsPort)

	// 7. 处理 docker-compose.yml
	composeContent, err := s.processDockerCompose(appDetail.DockerCompose, serviceName, envVars)
	if err != nil {
		return err
	}

	// 8. 写入文件
	if err := s.writeInstallFiles(installDir, envVars, composeContent); err != nil {
		return err
	}

	// 9. 创建安装记录
	install := &model.AppInstall{
		Name:          req.Name,
		AppID:         app.ID,
		AppDetailID:   appDetail.ID,
		Version:       appDetail.Version,
		Param:         s.paramsToJSON(req.Params),
		Env:           s.envToJSON(envVars),
		DockerCompose: composeContent,
		Status:        "installing",
		ContainerName: containerName,
		ServiceName:   serviceName,
		HttpPort:      httpPort,
		HttpsPort:     httpsPort,
	}

	if err := s.appInstallRepo.Create(ctx, install); err != nil {
		return err
	}

	// 10. 启动容器（异步）
	go func() {
		if err := s.startDockerCompose(installDir); err != nil {
			install.Status = "error"
			install.Message = err.Error()
		} else {
			install.Status = "running"
			install.Message = ""
		}
		s.appInstallRepo.Save(context.Background(), install)
	}()

	return nil
}

// allocatePorts 分配端口
func (s *AppInstallService) allocatePorts(params map[string]interface{}) (int, int, error) {
	var httpPort, httpsPort int

	// 检查用户指定的端口
	if port, ok := params["PANEL_APP_PORT_HTTP"]; ok {
		httpPort = int(port.(float64))
		if err := s.checkPortAvailable(httpPort); err != nil {
			return 0, 0, err
		}
	}

	if port, ok := params["PANEL_APP_PORT_HTTPS"]; ok {
		httpsPort = int(port.(float64))
		if err := s.checkPortAvailable(httpsPort); err != nil {
			return 0, 0, err
		}
	}

	// 如果没有指定，自动分配
	if httpPort == 0 {
		httpPort, _ = s.findAvailablePort(8000, 9000)
	}

	return httpPort, httpsPort, nil
}

// checkPortAvailable 检查端口是否可用
func (s *AppInstallService) checkPortAvailable(port int) error {
	// 检查是否已被其他应用使用
	existing, _ := s.appInstallRepo.GetFirst(s.appInstallRepo.WithPort(port))
	if existing.ID > 0 {
		return buserr.WithDetail("ErrPortInUse", strconv.Itoa(port), nil)
	}

	// TODO: 检查系统端口占用
	return nil
}

// findAvailablePort 查找可用端口
func (s *AppInstallService) findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		if err := s.checkPortAvailable(port); err == nil {
			return port, nil
		}
	}
	return 0, buserr.New("ErrNoAvailablePort")
}

// generateEnvVars 生成环境变量
func (s *AppInstallService) generateEnvVars(params map[string]interface{}, containerName string, httpPort, httpsPort int) map[string]string {
	envVars := make(map[string]string)

	// 添加面板标准变量
	envVars["CONTAINER_NAME"] = containerName
	envVars["PANEL_APP_PORT_HTTP"] = strconv.Itoa(httpPort)
	if httpsPort > 0 {
		envVars["PANEL_APP_PORT_HTTPS"] = strconv.Itoa(httpsPort)
	}

	// 添加用户参数
	for key, value := range params {
		envVars[key] = fmt.Sprintf("%v", value)
	}

	return envVars
}

// processDockerCompose 处理 docker-compose.yml
func (s *AppInstallService) processDockerCompose(composeContent, serviceName string, envVars map[string]string) (string, error) {
	var composeMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeContent), &composeMap); err != nil {
		return "", buserr.WithDetail("ErrParseCompose", err.Error(), err)
	}

	// 替换环境变量
	services, ok := composeMap["services"].(map[string]interface{})
	if !ok {
		return "", buserr.New("ErrInvalidCompose")
	}

	// 处理服务配置
	for _, svcConfig := range services {
		if svcMap, ok := svcConfig.(map[string]interface{}); ok {
			// 设置容器名称
			svcMap["container_name"] = envVars["CONTAINER_NAME"]

			// 设置网络
			if _, hasNetworks := svcMap["networks"]; !hasNetworks {
				svcMap["networks"] = []string{"xpanel-network"}
			}

			// 设置重启策略
			if _, hasRestart := svcMap["restart"]; !hasRestart {
				svcMap["restart"] = "unless-stopped"
			}
		}
		break // 只处理第一个服务
	}

	// 确保网络存在
	if _, hasNetworks := composeMap["networks"]; !hasNetworks {
		composeMap["networks"] = map[string]interface{}{
			"xpanel-network": map[string]interface{}{
				"external": true,
			},
		}
	}

	// 转回 YAML
	composeByte, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", err
	}

	return string(composeByte), nil
}

// writeInstallFiles 写入安装文件
func (s *AppInstallService) writeInstallFiles(installDir string, envVars map[string]string, composeContent string) error {
	// 写入 .env 文件
	envFile := filepath.Join(installDir, ".env")
	envContent := ""
	for key, value := range envVars {
		envContent += fmt.Sprintf("%s=%s\n", key, value)
	}
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		return err
	}

	// 写入 docker-compose.yml
	composeFile := filepath.Join(installDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		return err
	}

	// 创建数据目录
	dataDir := filepath.Join(installDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	return nil
}

// startDockerCompose 启动 docker-compose
func (s *AppInstallService) startDockerCompose(installDir string) error {
	composeFile := filepath.Join(installDir, "docker-compose.yml")
	
	// 确保 xpanel-network 存在
	if err := docker.EnsureNetwork("xpanel-network"); err != nil {
		return err
	}

	// docker-compose up -d
	output, err := cmd.ExecWithTimeoutAndOutput(300*time.Second, "docker-compose", "-f", composeFile, "up", "-d")
	if err != nil {
		return buserr.WithDetail("ErrDockerComposeUp", output, err)
	}

	return nil
}

// PageInstalled 分页查询已安装应用
func (s *AppInstallService) PageInstalled(req dto.AppInstallSearchReq) (int64, []dto.AppInstallDTO, error) {
	var opts []repo.DBOption

	if req.Name != "" {
		opts = append(opts, repo.WithByLikeName(req.Name))
	}

	if req.Type != "" {
		// TODO: 按应用类型筛选
	}

	total, installs, err := s.appInstallRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	var installDTOs []dto.AppInstallDTO
	for _, install := range installs {
		installDTO := s.convertInstallToDTO(install)
		installDTOs = append(installDTOs, installDTO)
	}

	return total, installDTOs, nil
}

// GetInstalled 获取已安装应用详情
func (s *AppInstallService) GetInstalled(id uint) (*dto.AppInstallDTO, error) {
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(id))
	if err != nil {
		return nil, err
	}

	installDTO := s.convertInstallToDTO(install)
	return &installDTO, nil
}

// Start 启动应用
func (s *AppInstallService) Start(installID uint) error {
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(installID))
	if err != nil {
		return err
	}

	installDir := s.getInstallDir(install)
	composeFile := filepath.Join(installDir, "docker-compose.yml")

	output, err := cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "start")
	if err != nil {
		return buserr.WithDetail("ErrDockerComposeStart", output, err)
	}

	install.Status = "running"
	return s.appInstallRepo.Save(context.Background(), &install)
}

// Stop 停止应用
func (s *AppInstallService) Stop(installID uint) error {
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(installID))
	if err != nil {
		return err
	}

	installDir := s.getInstallDir(install)
	composeFile := filepath.Join(installDir, "docker-compose.yml")

	output, err := cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "stop")
	if err != nil {
		return buserr.WithDetail("ErrDockerComposeStop", output, err)
	}

	install.Status = "stopped"
	return s.appInstallRepo.Save(context.Background(), &install)
}

// Restart 重启应用
func (s *AppInstallService) Restart(installID uint) error {
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(installID))
	if err != nil {
		return err
	}

	installDir := s.getInstallDir(install)
	composeFile := filepath.Join(installDir, "docker-compose.yml")

	output, err := cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "restart")
	if err != nil {
		return buserr.WithDetail("ErrDockerComposeRestart", output, err)
	}

	install.Status = "running"
	return s.appInstallRepo.Save(context.Background(), &install)
}

// Uninstall 卸载应用
func (s *AppInstallService) Uninstall(req dto.AppUninstallReq) error {
	ctx := context.Background()

	install, err := s.appInstallRepo.GetFirst(repo.WithByID(req.InstallID))
	if err != nil {
		return err
	}

	installDir := s.getInstallDir(install)
	composeFile := filepath.Join(installDir, "docker-compose.yml")

	// 停止并删除容器
	if !req.ForceDelete {
		output, err := cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "down")
		if err != nil {
			return buserr.WithDetail("ErrDockerComposeDown", output, err)
		}
	}

	// 删除数据
	if req.DeleteData {
		if err := os.RemoveAll(installDir); err != nil {
			global.LOG.Errorf("Failed to remove install dir: %v", err)
		}
	}

	// 删除数据库记录
	return s.appInstallRepo.Delete(ctx, &install)
}

// Update 更新应用
func (s *AppInstallService) Update(req dto.AppUpdateReq) error {
	// TODO: 实现应用更新逻辑
	return buserr.New("ErrNotImplemented")
}

// GetLogs 获取容器日志
func (s *AppInstallService) GetLogs(installID uint, lines int) (string, error) {
	// 获取安装记录
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(installID))
	if err != nil {
		return "", err
	}

	if install.ContainerName == "" {
		return "", buserr.New("ErrContainerNotFound")
	}

	// 使用 docker logs 命令获取日志
	args := []string{"logs", "--tail", strconv.Itoa(lines), install.ContainerName}
	output, err := cmd.ExecWithTimeoutAndOutput(10*time.Second, "docker", args...)
	
	if err != nil {
		return "", buserr.WithDetail("ErrGetContainerLogs", err.Error(), err)
	}

	return output, nil
}

// 辅助方法

func (s *AppInstallService) getInstallDir(install model.AppInstall) string {
	return filepath.Join(global.CONF.System.DataDir, "apps", install.App.Key, install.Name)
}

func (s *AppInstallService) paramsToJSON(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	return string(data)
}

func (s *AppInstallService) envToJSON(env map[string]string) string {
	data, _ := json.Marshal(env)
	return string(data)
}

func (s *AppInstallService) convertInstallToDTO(install model.AppInstall) dto.AppInstallDTO {
	return dto.AppInstallDTO{
		ID:            install.ID,
		Name:          install.Name,
		AppID:         install.AppID,
		AppKey:        install.App.Key,
		AppName:       install.App.Name,
		AppIcon:       install.App.Icon,
		Version:       install.Version,
		Status:        install.Status,
		Message:       install.Message,
		ContainerName: install.ContainerName,
		HttpPort:      install.HttpPort,
		HttpsPort:     install.HttpsPort,
		WebUI:         install.WebUI,
		InstalledAt:   install.InstalledAt.Format("2006-01-02 15:04:05"),
	}
}
