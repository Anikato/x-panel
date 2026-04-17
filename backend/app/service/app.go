package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/global"
)

const (
	// 默认应用商店配置（兼容 1Panel）
	DefaultAppStoreRepo   = "https://github.com/1Panel-dev/appstore"
	DefaultAppStoreBranch = "main"
	DefaultAppStoreURL    = "https://resource.1panel.hk/appstore"
)

var (
	appStoreSyncMu  sync.Mutex
	appStoreSyncing bool
)

type IAppService interface {
	// 应用商店同步
	SyncAppStore(force bool) error
	
	// 应用查询
	PageApps(req dto.AppSearchReq) (int64, []dto.AppDTO, error)
	GetAppByKey(key string) (*dto.AppDTO, error)
	GetAppDetail(appID uint, version string) (*dto.AppDetailDTO, error)
	
	// 标签管理
	GetTags() ([]dto.TagDTO, error)
}

type AppService struct {
	appRepo       repo.IAppRepo
	appDetailRepo repo.IAppDetailRepo
	appTagRepo    repo.IAppTagRepo
	tagRepo       repo.ITagRepo
}

func NewIAppService() IAppService {
	return &AppService{
		appRepo:       repo.NewIAppRepo(),
		appDetailRepo: repo.NewIAppDetailRepo(),
		appTagRepo:    repo.NewIAppTagRepo(),
		tagRepo:       repo.NewITagRepo(),
	}
}

// SyncAppStore 从远程同步应用商店数据
func (s *AppService) SyncAppStore(force bool) error {
	appStoreSyncMu.Lock()
	if appStoreSyncing {
		appStoreSyncMu.Unlock()
		return buserr.New("ErrAppStoreSyncing")
	}
	appStoreSyncing = true
	appStoreSyncMu.Unlock()

	defer func() {
		appStoreSyncMu.Lock()
		appStoreSyncing = false
		appStoreSyncMu.Unlock()
	}()

	global.LOG.Info("[AppStore] Starting sync from remote")

	// 1. 下载应用列表
	appList, err := s.downloadAppList()
	if err != nil {
		return err
	}

	// 2. 下载标签列表
	tags, err := s.downloadTags()
	if err != nil {
		return err
	}

	// 3. 保存到数据库
	ctx := context.Background()
	
	// 保存标签
	if err := s.saveTags(ctx, tags); err != nil {
		return err
	}

	// 保存应用
	if err := s.saveApps(ctx, appList); err != nil {
		return err
	}

	global.LOG.Info("[AppStore] Sync completed successfully")
	return nil
}

// downloadAppList 下载应用列表
func (s *AppService) downloadAppList() (*AppListResponse, error) {
	// 下载 1panel.json
	url := fmt.Sprintf("%s/1panel.json", DefaultAppStoreURL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, buserr.WithDetail("ErrDownloadAppList", err.Error(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buserr.WithDetail("ErrDownloadAppList", fmt.Sprintf("status code: %d", resp.StatusCode), nil)
	}

	var appList AppListResponse
	if err := json.NewDecoder(resp.Body).Decode(&appList); err != nil {
		return nil, buserr.WithDetail("ErrParseAppList", err.Error(), err)
	}

	return &appList, nil
}

// downloadTags 下载标签列表
func (s *AppService) downloadTags() ([]TagData, error) {
	url := fmt.Sprintf("%s/tags.json", DefaultAppStoreURL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, buserr.WithDetail("ErrDownloadTags", err.Error(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buserr.WithDetail("ErrDownloadTags", fmt.Sprintf("status code: %d", resp.StatusCode), nil)
	}

	var tags []TagData
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, buserr.WithDetail("ErrParseTags", err.Error(), err)
	}

	return tags, nil
}

// saveTags 保存标签到数据库
func (s *AppService) saveTags(ctx context.Context, tags []TagData) error {
	for _, tagData := range tags {
		// 检查标签是否存在
		existingTag, err := s.tagRepo.GetFirst(repo.WithByKey(tagData.Key))
		if err == nil && existingTag.ID > 0 {
			// 更新
			existingTag.Name = tagData.Name
			existingTag.Sort = tagData.Sort
			if err := s.tagRepo.Save(ctx, &existingTag); err != nil {
				return err
			}
		} else {
			// 创建
			tag := &model.Tag{
				Key:  tagData.Key,
				Name: tagData.Name,
				Sort: tagData.Sort,
			}
			if err := s.tagRepo.Create(ctx, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

// saveApps 保存应用到数据库
func (s *AppService) saveApps(ctx context.Context, appList *AppListResponse) error {
	for _, appData := range appList.Apps {
		// 过滤掉需要 Runtime 的应用
		if s.shouldSkipApp(appData) {
			global.LOG.Infof("[AppStore] Skipping app %s (requires runtime)", appData.Key)
			continue
		}

		// 检查应用是否存在
		existingApp, err := s.appRepo.GetFirst(s.appRepo.WithKey(appData.Key))
		if err == nil && existingApp.ID > 0 {
			// 更新应用
			s.updateAppFromData(&existingApp, appData)
			if err := s.appRepo.Save(ctx, &existingApp); err != nil {
				return err
			}

			// 保存版本
			if err := s.saveAppVersions(ctx, existingApp.ID, appData); err != nil {
				return err
			}

			// 保存标签关联
			if err := s.saveAppTags(ctx, existingApp.ID, appData.Tags); err != nil {
				return err
			}
		} else {
			// 创建新应用
			app := s.createAppFromData(appData)
			if err := s.appRepo.Create(ctx, app); err != nil {
				return err
			}

			// 保存版本
			if err := s.saveAppVersions(ctx, app.ID, appData); err != nil {
				return err
			}

			// 保存标签关联
			if err := s.saveAppTags(ctx, app.ID, appData.Tags); err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldSkipApp 判断是否应该跳过该应用
func (s *AppService) shouldSkipApp(appData AppData) bool {
	// 跳过 Runtime 类型的应用
	if appData.Type == "runtime" || appData.Type == "php" || 
	   appData.Type == "node" || appData.Type == "python" || 
	   appData.Type == "java" || appData.Type == "go" {
		return true
	}

	// 检查是否有 Runtime 依赖
	if appData.Required != "" {
		var required map[string]interface{}
		if err := json.Unmarshal([]byte(appData.Required), &required); err == nil {
			if _, hasRuntime := required["runtime"]; hasRuntime {
				return true
			}
		}
	}

	return false
}

// createAppFromData 从远程数据创建应用模型
func (s *AppService) createAppFromData(data AppData) *model.App {
	architectures, _ := json.Marshal(data.Architectures)
	
	return &model.App{
		Name:                 data.Name,
		Key:                  data.Key,
		ShortDescZh:          data.ShortDescZh,
		ShortDescEn:          data.ShortDescEn,
		Description:          data.Description,
		Icon:                 data.Icon,
		Type:                 data.Type,
		Status:               "ready",
		Required:             data.Required,
		CrossVersionUpdate:   data.CrossVersionUpdate,
		LimitNum:             data.Limit,
		Website:              data.Website,
		Github:               data.Github,
		Document:             data.Document,
		Recommend:            data.Recommend,
		Resource:             "remote",
		Architectures:        string(architectures),
		MemoryRequired:       data.MemoryRequired,
		GpuSupport:           data.GpuSupport,
		RequiredPanelVersion: data.RequiredPanelVersion,
		BatchInstallSupport:  data.BatchInstallSupport,
	}
}

// updateAppFromData 从远程数据更新应用模型
func (s *AppService) updateAppFromData(app *model.App, data AppData) {
	architectures, _ := json.Marshal(data.Architectures)
	
	app.Name = data.Name
	app.ShortDescZh = data.ShortDescZh
	app.ShortDescEn = data.ShortDescEn
	app.Description = data.Description
	app.Icon = data.Icon
	app.Type = data.Type
	app.Required = data.Required
	app.CrossVersionUpdate = data.CrossVersionUpdate
	app.LimitNum = data.Limit
	app.Website = data.Website
	app.Github = data.Github
	app.Document = data.Document
	app.Recommend = data.Recommend
	app.Architectures = string(architectures)
	app.MemoryRequired = data.MemoryRequired
	app.GpuSupport = data.GpuSupport
	app.RequiredPanelVersion = data.RequiredPanelVersion
	app.BatchInstallSupport = data.BatchInstallSupport
}

// saveAppVersions 保存应用版本
func (s *AppService) saveAppVersions(ctx context.Context, appID uint, appData AppData) error {
	for _, versionData := range appData.Versions {
		// 检查版本是否存在
		existingDetail, err := s.appDetailRepo.GetFirst(
			s.appDetailRepo.WithAppID(appID),
			s.appDetailRepo.WithVersion(versionData.Version),
		)

		params, _ := json.Marshal(versionData.Params)

		if err == nil && existingDetail.ID > 0 {
			// 更新版本
			existingDetail.Params = string(params)
			existingDetail.DockerCompose = versionData.DockerCompose
			existingDetail.DownloadURL = versionData.DownloadURL
			if err := s.appDetailRepo.Save(ctx, &existingDetail); err != nil {
				return err
			}
		} else {
			// 创建新版本
			detail := &model.AppDetail{
				AppID:         appID,
				Version:       versionData.Version,
				Params:        string(params),
				DockerCompose: versionData.DockerCompose,
				Status:        "ready",
				DownloadURL:   versionData.DownloadURL,
			}
			if err := s.appDetailRepo.Create(ctx, detail); err != nil {
				return err
			}
		}
	}
	return nil
}

// saveAppTags 保存应用标签关联
func (s *AppService) saveAppTags(ctx context.Context, appID uint, tagKeys []string) error {
	// 删除旧的标签关联
	if err := s.appTagRepo.DeleteBy(ctx, s.appTagRepo.WithAppID(appID)); err != nil {
		return err
	}

	// 创建新的标签关联
	for _, tagKey := range tagKeys {
		appTag := &model.AppTag{
			AppID:  appID,
			TagKey: tagKey,
		}
		if err := s.appTagRepo.Create(ctx, appTag); err != nil {
			return err
		}
	}
	return nil
}

// PageApps 分页查询应用
func (s *AppService) PageApps(req dto.AppSearchReq) (int64, []dto.AppDTO, error) {
	var opts []repo.DBOption

	// 按推荐度排序
	opts = append(opts, s.appRepo.OrderByRecommend())

	// 名称搜索
	if req.Name != "" {
		opts = append(opts, repo.WithByLikeName(req.Name))
	}

	// 类型筛选
	if req.Type != "" {
		opts = append(opts, s.appRepo.WithType(req.Type))
	}

	// 标签筛选
	if len(req.Tags) > 0 {
		// TODO: 实现标签筛选
	}

	total, apps, err := s.appRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	// 转换为 DTO
	var appDTOs []dto.AppDTO
	for _, app := range apps {
		appDTO := s.convertAppToDTO(app)
		appDTOs = append(appDTOs, appDTO)
	}

	return total, appDTOs, nil
}

// GetAppByKey 根据 key 获取应用详情
func (s *AppService) GetAppByKey(key string) (*dto.AppDTO, error) {
	app, err := s.appRepo.GetFirst(s.appRepo.WithKey(key))
	if err != nil {
		return nil, err
	}

	appDTO := s.convertAppToDTO(app)

	// 获取版本列表
	details, err := s.appDetailRepo.GetBy(s.appDetailRepo.WithAppID(app.ID))
	if err != nil {
		return nil, err
	}

	for _, detail := range details {
		appDTO.Versions = append(appDTO.Versions, detail.Version)
	}

	// 获取标签
	appTags, err := s.appTagRepo.GetBy(s.appTagRepo.WithAppID(app.ID))
	if err != nil {
		return nil, err
	}

	for _, appTag := range appTags {
		appDTO.Tags = append(appDTO.Tags, appTag.TagKey)
	}

	return &appDTO, nil
}

// GetAppDetail 获取应用版本详情
func (s *AppService) GetAppDetail(appID uint, version string) (*dto.AppDetailDTO, error) {
	detail, err := s.appDetailRepo.GetFirst(
		s.appDetailRepo.WithAppID(appID),
		s.appDetailRepo.WithVersion(version),
	)
	if err != nil {
		return nil, err
	}

	// 解析参数
	var params map[string]interface{}
	if detail.Params != "" {
		if err := json.Unmarshal([]byte(detail.Params), &params); err != nil {
			return nil, err
		}
	}

	return &dto.AppDetailDTO{
		ID:            detail.ID,
		AppID:         detail.AppID,
		Version:       detail.Version,
		Params:        detail.Params,
		DockerCompose: detail.DockerCompose,
		Status:        detail.Status,
		DownloadURL:   detail.DownloadURL,
	}, nil
}

// GetTags 获取所有标签
func (s *AppService) GetTags() ([]dto.TagDTO, error) {
	tags, err := s.tagRepo.GetAll()
	if err != nil {
		return nil, err
	}

	var tagDTOs []dto.TagDTO
	for _, tag := range tags {
		tagDTOs = append(tagDTOs, dto.TagDTO{
			Key:  tag.Key,
			Name: tag.Name,
			Sort: tag.Sort,
		})
	}

	return tagDTOs, nil
}

// convertAppToDTO 转换应用模型为 DTO
func (s *AppService) convertAppToDTO(app model.App) dto.AppDTO {
	var architectures []string
	if app.Architectures != "" {
		json.Unmarshal([]byte(app.Architectures), &architectures)
	}

	return dto.AppDTO{
		ID:                   app.ID,
		Name:                 app.Name,
		Key:                  app.Key,
		ShortDescZh:          app.ShortDescZh,
		ShortDescEn:          app.ShortDescEn,
		Description:          app.Description,
		Icon:                 app.Icon,
		Type:                 app.Type,
		Status:               app.Status,
		CrossVersionUpdate:   app.CrossVersionUpdate,
		LimitNum:             app.LimitNum,
		Website:              app.Website,
		Github:               app.Github,
		Document:             app.Document,
		Recommend:            app.Recommend,
		Resource:             app.Resource,
		Architectures:        architectures,
		MemoryRequired:       app.MemoryRequired,
		GpuSupport:           app.GpuSupport,
		RequiredPanelVersion: app.RequiredPanelVersion,
		CreatedAt:            app.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// 远程数据结构
type AppListResponse struct {
	Apps []AppData `json:"apps"`
}

type AppData struct {
	Name                 string                 `json:"name"`
	Key                  string                 `json:"key"`
	ShortDescZh          string                 `json:"shortDescZh"`
	ShortDescEn          string                 `json:"shortDescEn"`
	Description          string                 `json:"description"`
	Icon                 string                 `json:"icon"`
	Type                 string                 `json:"type"`
	Required             string                 `json:"required"`
	CrossVersionUpdate   bool                   `json:"crossVersionUpdate"`
	Limit                int                    `json:"limit"`
	Website              string                 `json:"website"`
	Github               string                 `json:"github"`
	Document             string                 `json:"document"`
	Recommend            int                    `json:"recommend"`
	Architectures        []string               `json:"architectures"`
	MemoryRequired       int                    `json:"memoryRequired"`
	GpuSupport           bool                   `json:"gpuSupport"`
	RequiredPanelVersion string                 `json:"requiredPanelVersion"`
	BatchInstallSupport  bool                   `json:"batchInstallSupport"`
	Tags                 []string               `json:"tags"`
	Versions             []AppVersionData       `json:"versions"`
}

type AppVersionData struct {
	Version       string                 `json:"version"`
	Params        map[string]interface{} `json:"params"`
	DockerCompose string                 `json:"dockerCompose"`
	DownloadURL   string                 `json:"downloadUrl"`
}

type TagData struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}
