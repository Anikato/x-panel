# X-Panel 工作日志

> 记录每次开发会话的工作内容，便于追踪项目进展和上下文衔接。

---

## 2026-04-18 — Session：面板 HTTPS 绑定证书管理

### 完成内容

- [x] 后端：`CertificateService.ResolveCertFilePaths`；`SettingService.GetPanelSSL` / `UpdatePanelSSL`；`GET/POST /api/v1/settings/panel-ssl`；设置键 `PanelSSLCertificateID`；`constant` + `zh.yaml` 错误文案
- [x] 前端：面板设置「面板 HTTPS 证书」；证书下拉（仅 `ready`）；保存与重启提示

### 说明

- 保存后写入 `config.yaml` 的 `system.ssl`，需重启面板进程后 TLS 重新加载。

## 2026-04-18 — Session #80：应用中心编译修复 + 前后端全量编译通过

### 完成内容

**后端编译修复（全量通过）**：
- [x] 新增 `backend/utils/helper/helper.go` — API 响应工具包（CheckBindAndValidate、SuccessWithData、ErrorWithDetail 等）
- [x] 修复 `backend/app/repo/` 全系列文件 — `getDB()` → `getDb()` 统一函数名
- [x] 修复 `backend/app/repo/common.go` — 新增 `WithByAppID`、`WithByVersion`、`WithBySourceID`、`WithLikeName`、`WithLikeDomain`、`WithOrderBy` 等缺失的 DBOption 函数
- [x] 修复 `backend/app/repo/app_import_task.go` — 移除错误的 context 参数用法
- [x] 修复 `backend/app/repo/cert_sync.go` — 修正函数名
- [x] 修复 `backend/app/repo/gost.go` — 重写，统一使用 getDb(opts...)
- [x] 修复 `backend/app/repo/host.go` — 删除重复的 WithByType 声明
- [x] 修复 `backend/app/service/app_install.go` — 补充 `"time"` import，修正 cmd 函数名
- [x] 修复 `backend/app/service/app_backup.go` — 修正 ExecWithTimeoutAndOutput 参数顺序
- [x] 修复 `backend/app/service/app_import.go` — 修正字段名 Params→Param，修正方法名 Update→Save、GetList→GetBy，删除未使用变量
- [x] 修复 `backend/app/service/app_import_progress.go` — 删除未使用的 steps 变量，修正 WithOrderBy 参数
- [x] 修复 `backend/app/service/app.go` — 删除未使用的 import
- [x] 修复 `backend/app/api/v1/app.go` — dto.BatchDelete → dto.OperateByIDs
- [x] 修复 `backend/init/migration/migration.go` — 删除不存在的 model.AppBackup 引用
- [x] 修复 `backend/utils/docker/network.go` — cmd.Exec → cmd.ExecWithOutput

**前端编译修复（全量通过）**：
- [x] 修复 `frontend/src/api/modules/app.ts` — 修正 import 路径和 ResPage→PageResult
- [x] 修复 `frontend/src/store/modules/global.ts` — persist.pick → persist.paths（pinia-plugin-persistedstate v3 API）
- [x] 修复 `frontend/src/i18n/zh.ts` — 删除重复的 running、backupPath、message key，backupPath→backupFilePath
- [x] 修复 `frontend/src/views/app/installed/index.vue` — 修正 TS 类型错误
- [x] 修复 `frontend/src/views/app/store/index.vue` — 修正 null 检查

### 关键决策
- helper 包采用简化版（不依赖 validator），与项目现有风格一致
- 保留所有业务逻辑不变，仅修复编译错误

### 下一步计划
- 实际部署测试应用中心功能
- 测试 1Panel 备份导入流程

---



### 完成内容

**1Panel 备份导入功能（完整实现）**：
- [x] `backend/app/service/app_import.go` — 完整的备份导入服务（已在 Session #78 实现）
  - 解压 tar.gz 备份包
  - 读取元数据（app.json 或从文件推断）
  - 环境变量转换（PANEL_ → XPANEL_）
  - 端口冲突自动分配
  - 数据文件复制
  - docker-compose.yml 处理
  - 创建安装记录并启动容器
- [x] `backend/app/api/v1/app.go` — ImportApp API Handler（已在 Session #78 实现）
- [x] `backend/router/router.go` — POST /apps/import 路由注册（已在 Session #78 实现）
- [x] `backend/constant/errs.go` + `backend/i18n/lang/zh.yaml` — 导入相关错误码和翻译（已在 Session #78 实现）
- [x] `frontend/src/api/modules/app.ts` — importApp API 调用（已在 Session #78 实现）
- [x] `frontend/src/views/app/store/index.vue` — 导入对话框 UI（已在 Session #78 实现）
- [x] `frontend/src/i18n/zh.ts` — **本次完成：添加导入功能的 15 条中文翻译**
  - importFromBackup: '从备份导入'
  - import: '导入'
  - backupPath: '备份文件路径'
  - backupPathHint: '服务器上的备份文件绝对路径'
  - backupPathDesc: '例如: /opt/xpanel/backup/1panel_wordpress_20240101.tar.gz'
  - backupPathRequired: '请输入备份文件路径'
  - appKey: '应用标识'
  - appKeyHint: '可选，用于匹配应用商店'
  - appKeyDesc: '如 wordpress、mysql 等，留空则从备份中自动识别'
  - versionHint: '可选，如 latest、1.0.0'
  - importWarning: '导入说明'
  - importWarningDesc: '1. 支持导入 1Panel 和 X-Panel 的备份文件<br/>2. 如果端口冲突，系统会自动分配新端口<br/>3. 导入过程可能需要几分钟，请耐心等待<br/>4. 导入后请检查应用配置是否正确'
  - importSuccess: '导入成功'
  - importFailed: '导入失败'

### 功能特性

**支持的备份格式**：
- 1Panel 官方备份格式（含 app.json 元数据）
- 简化备份格式（仅 docker-compose.yml + .env + data/）
- 自动从文件推断应用信息

**智能处理**：
- 环境变量自动转换（PANEL_ 前缀 → XPANEL_）
- 端口冲突自动检测并分配新端口
- 应用商店匹配（通过 appKey）
- 找不到应用时创建临时应用记录

**用户体验**：
- 应用商店页面新增「从备份导入」按钮
- 表单验证（必填项：安装名称、备份路径）
- 可选字段（appKey、version）
- 详细的导入说明和警告提示
- 导入成功后自动跳转到已安装应用页面

### 测试建议

1. **准备 1Panel 备份文件**：
   - 从 1Panel 导出应用备份（如 WordPress、MySQL）
   - 上传到 X-Panel 服务器的某个目录

2. **测试导入流程**：
   - 打开应用商店页面
   - 点击「从备份导入」按钮
   - 填写安装名称（如 `my-wordpress`）
   - 填写备份文件路径（如 `/tmp/1panel_wordpress_20240101.tar.gz`）
   - 可选填写 appKey（如 `wordpress`）和版本（如 `latest`）
   - 点击导入
   - 等待导入完成（可能需要几分钟）
   - 检查已安装应用列表

3. **验证结果**：
   - 应用是否成功启动
   - 端口是否正确分配
   - 数据是否完整迁移
   - Web UI 是否可访问

### 下一步

- 测试真实的 1Panel 备份文件导入
- 根据测试结果优化错误处理
- 考虑添加导入进度显示
- 考虑支持从 URL 下载备份文件

---

## 2026-04-17 — Session #78：应用中心日志查看 + 1Panel 备份导入设计

### 完成内容

**容器日志查看功能**：
- [x] `backend/app/service/app_install.go` — 新增 `GetLogs()` 方法
  - 使用 `docker logs --tail N` 获取容器日志
  - 支持指定行数（默认 100，最多 1000）
  - 自动处理 stdout 和 stderr
- [x] `backend/app/api/v1/app.go` — 新增 `GetAppLogs` API
  - GET `/apps/installed/:id/logs?lines=100`
  - 返回纯文本日志内容
- [x] `backend/router/router.go` — 注册日志查看路由
- [x] `frontend/src/api/modules/app.ts` — 新增 `getAppLogs` API 调用
- [x] `frontend/src/views/app/installed/index.vue` — 实现真实的日志查看
  - 日志对话框展示最近 500 行
  - 支持刷新
  - monospace 字体显示
- [x] 错误码和翻译：`ErrContainerNotFound`、`ErrGetContainerLogs`

**1Panel 备份导入设计**：
- [x] `backend/app/dto/app.go` — 新增 `AppImportReq` DTO
  - 支持指定安装名称、备份路径
  - 可选指定应用 key 和版本

### 关于 1Panel 备份导入

**当前状态**：
- DTO 已定义，但导入功能尚未完全实现
- 需要解析 1Panel 备份包的元数据和目录结构

**实现方案**：

1. **备份包结构分析**：
   ```
   1panel_backup_appname_20240101.tar.gz
   ├── app.json          # 应用元数据（名称、版本、参数等）
   ├── docker-compose.yml
   ├── .env              # 环境变量
   └── data/             # 应用数据目录
   ```

2. **导入流程**：
   - 解压备份包到临时目录
   - 读取 `app.json` 获取应用信息
   - 尝试从应用商店匹配应用（通过 key）
   - 如果匹配成功，使用应用商店的配置
   - 如果匹配失败，使用备份包中的配置
   - 转换环境变量（1Panel → X-Panel 格式）
   - 创建安装记录
   - 启动容器

3. **环境变量转换**：
   ```
   1Panel:  PANEL_APP_PORT_HTTP=8080
   X-Panel: XPANEL_APP_PORT_HTTP=8080
   ```

4. **兼容性考虑**：
   - 端口可能冲突，需要重新分配
   - 数据目录路径不同
   - 容器名称格式不同

**手动导入方法（临时方案）**：

如果你现在就需要导入 1Panel 的应用，可以手动操作：

1. **解压 1Panel 备份**：
   ```bash
   tar -xzf 1panel_backup_wordpress_20240101.tar.gz -C /tmp/import
   ```

2. **查看应用信息**：
   ```bash
   cat /tmp/import/app.json
   cat /tmp/import/.env
   ```

3. **在 X-Panel 中安装相同应用**：
   - 从应用商店安装同名应用
   - 记录安装目录

4. **停止新安装的应用**：
   ```bash
   docker-compose -f /opt/xpanel/apps/wordpress/mysite/docker-compose.yml down
   ```

5. **复制数据**：
   ```bash
   cp -r /tmp/import/data/* /opt/xpanel/apps/wordpress/mysite/
   ```

6. **启动应用**：
   - 在 X-Panel 界面点击启动

### 下一步

**优先级 1（已完成）**：
- ✅ 容器日志查看

**优先级 2（建议实现）**：
- [ ] 完整的 1Panel 备份导入功能
  - 备份包解析
  - 元数据提取
  - 环境变量转换
  - 自动端口分配
  - 数据迁移

**优先级 3（可选）**：
- [ ] 应用更新功能
- [ ] 应用导出功能（生成 X-Panel 格式备份）
- [ ] 批量操作（批量启停、批量备份）

### 测试建议

1. **测试日志查看**：
   - 安装一个应用（如 Nginx）
   - 点击"查看日志"按钮
   - 验证日志内容正确显示
   - 测试刷新功能

2. **测试完整流程**：
   - 同步应用商店
   - 安装 WordPress
   - 启停控制
   - 创建备份
   - 恢复备份
   - 查看日志

---

## 2026-04-17 — Session #77：应用中心前端实现

### 完成内容

**前端页面（Vue 3 + Element Plus）**：
- [x] `frontend/src/views/app/store/index.vue` — 应用商店页面
  - 应用列表网格展示（卡片式）
  - 搜索和筛选（名称、类型、标签）
  - 应用详情对话框
  - 安装对话框（版本选择、动态参数表单）
  - 同步应用商店功能
- [x] `frontend/src/views/app/installed/index.vue` — 已安装应用页面
  - 应用列表表格展示
  - 启动/停止/重启操作
  - 备份对话框
  - 卸载确认
  - 日志查看对话框（预留）
  - Web UI 快速访问
- [x] `frontend/src/views/app/backups/index.vue` — 应用备份页面
  - 备份列表表格展示
  - 备份详情对话框
  - 恢复功能
  - 删除备份

**路由和菜单**：
- [x] `frontend/src/routers/modules/app.ts` — 应用中心路由模块
- [x] `frontend/src/routers/index.ts` — 注册应用中心路由
- [x] `frontend/src/layout/components/Sidebar.vue` — 侧边栏新增「应用中心」菜单
  - 应用商店
  - 已安装
  - 应用备份

**API 客户端**：
- [x] `frontend/src/api/modules/app.ts` — 完整的 API 客户端（已在上一轮完成）

**国际化**：
- [x] `frontend/src/i18n/zh.ts` — 新增 60+ 应用中心相关翻译
  - 应用商店相关
  - 已安装应用相关
  - 备份相关

### 关键设计决策

**应用商店页面**：
- 网格布局：每个应用卡片展示图标、名称、描述、类型、安装次数
- 标签筛选：点击标签快速过滤应用
- 动态参数表单：根据应用版本的 params 定义自动生成表单字段（text/number/boolean/select）
- 安装名称：用户可自定义实例名称，默认使用应用 key

**已安装应用页面**：
- 表格布局：展示应用图标、名称、版本、状态、Web UI、安装时间
- 状态标签：运行中（绿色）、已停止（灰色）、安装中（黄色）、错误（红色）
- 操作按钮：启停控制、备份、更多（更新、日志、卸载）
- Web UI 链接：直接打开应用的 Web 界面

**备份页面**：
- 表格布局：展示应用名称、备份名称、类型、大小、状态、创建时间
- 恢复功能：带确认提示，警告数据将被覆盖
- 详情对话框：展示完整备份信息（路径、校验和等）

### UI/UX 特性

- 应用卡片 hover 效果：阴影和轻微上移
- 状态标签颜色编码：success/info/warning/danger
- 分页支持：应用商店 12/24/48，列表页 10/20/50
- 加载状态：所有异步操作都有 loading 状态
- 错误处理：统一的错误提示
- 确认对话框：危险操作（卸载、恢复）需要确认

### 遗留问题

- 应用更新功能：需要获取最新版本的 appDetailId，暂未实现
- 容器日志查看：API 未实现，前端预留了对话框
- 应用详情：可以扩展更多信息（系统要求、更新日志等）

### 下一步

- 测试完整流程：同步 → 安装 → 启停 → 备份 → 恢复
- 实现容器日志查看 API
- 完善应用更新功能
- 添加应用搜索建议
- 优化大量应用时的性能

---

## 2026-04-17 — Session #76：应用中心 Repository + Service + API 层实现

### 完成内容

**Repository 层（完整 CRUD + 查询选项）**：
- [x] `backend/app/repo/common.go` — 通用 DBOption 函数 + getDb/getTx 辅助函数
- [x] `backend/app/repo/app.go` — 应用商店应用 CRUD
- [x] `backend/app/repo/app_detail.go` — 应用版本详情 CRUD
- [x] `backend/app/repo/app_install.go` — 已安装应用 CRUD
- [x] `backend/app/repo/app_tag.go` — 应用标签关联 CRUD
- [x] `backend/app/repo/tag.go` — 标签 CRUD
- [x] `backend/app/repo/app_backup.go` — 应用备份记录 CRUD

**Service 层（核心业务逻辑）**：
- [x] `backend/app/service/app.go` — 应用商店服务
  - 从 1Panel 官方仓库同步应用列表和标签
  - 自动过滤 Runtime 应用（php/node/python/java/go）
  - 应用查询、标签管理、数据转换
- [x] `backend/app/service/app_install.go` — 应用安装服务
  - 端口自动分配（8000-9000 范围）
  - 环境变量生成、docker-compose 处理
  - 应用启停控制、异步安装
- [x] `backend/app/service/app_backup.go` — 应用备份服务
  - tar.gz 备份（排除日志文件）
  - 失败自动回滚的恢复机制
  - 备份管理

**API Handler 层**：
- [x] `backend/app/api/v1/app.go` — 完整的 HTTP 接口（15 个 API）
  - 应用商店：同步、分页查询、详情、标签
  - 应用安装：安装、已安装列表、详情
  - 应用操作：启动/停止/重启/卸载/更新
  - 应用备份：备份、恢复、备份列表、删除备份

**路由和配置**：
- [x] `backend/router/router.go` — 注册 15 条应用中心路由
- [x] `backend/app/api/v1/entry.go` — 添加 AppAPI 到 ApiGroup
- [x] `backend/init/migration/migration.go` — AutoMigrate 6 个新表
- [x] `backend/constant/errs.go` — 添加 20+ 应用中心错误码
- [x] `backend/i18n/lang/zh.yaml` — 添加 20+ 中文错误翻译

**工具函数**：
- [x] `backend/utils/docker/network.go` — Docker 网络管理

### 设计决策

**Repository 层**：
- DBOption 模式：函数式查询选项，灵活组合
- Context 事务支持：所有写操作支持事务传递
- Preload 关联：自动加载关联数据
- Omit Associations：避免级联写入

**Service 层**：
- **1Panel 兼容**：默认从 `https://resource.1panel.hk/appstore` 同步
- **Runtime 过滤**：自动跳过不支持的应用类型
- **端口管理**：智能分配，避免冲突
- **Docker 网络**：所有应用加入 `xpanel-network`
- **异步安装**：goroutine 执行，不阻塞 API
- **备份回滚**：恢复失败自动回滚

**API 层**：
- RESTful 风格：GET 查询、POST 操作
- 统一响应：helper.SuccessWithData / ErrorWithDetail
- 参数校验：CheckBindAndValidate 统一校验
- 错误处理：业务错误码 + i18n 翻译

### API 路由列表

```
POST   /api/v1/apps/sync                  # 同步应用商店
POST   /api/v1/apps/search                # 分页查询应用
GET    /api/v1/apps/tags                  # 获取标签
GET    /api/v1/apps/:key                  # 获取应用详情
GET    /api/v1/apps/detail                # 获取版本详情
POST   /api/v1/apps/install               # 安装应用
POST   /api/v1/apps/installed/search      # 已安装应用列表
GET    /api/v1/apps/installed/:id         # 已安装应用详情
POST   /api/v1/apps/operate               # 操作应用（启停重启）
POST   /api/v1/apps/uninstall             # 卸载应用
POST   /api/v1/apps/update                # 更新应用
POST   /api/v1/apps/backup                # 备份应用
POST   /api/v1/apps/restore               # 恢复应用
POST   /api/v1/apps/backups/search        # 备份列表
POST   /api/v1/apps/backups/del           # 删除备份
```

### 1Panel 兼容性

- ✅ 应用商店数据格式完全兼容
- ✅ 环境变量命名兼容（`PANEL_APP_PORT_HTTP` 等）
- ✅ docker-compose.yml 结构兼容
- ✅ 可以直接使用 1Panel 的应用（WordPress、Nextcloud、MySQL 等）
- ❌ 不支持 Runtime 管理（PHP/Node.js 等）
- ❌ 不支持应用商店管理后台

### 下一步

- 实现前端页面（应用商店、已安装应用、备份管理）
- 添加到侧边栏菜单
- 测试应用安装流程
- 完善错误处理和日志

---

## 2026-04-17 — Session #75：HAProxy 负载均衡可视化管理（apt 安装版）

### 完成内容

**设计文档**：
- [x] `docs/haproxy-design.md` — 从 HAProxy 用户视角出发的完整设计文档
  - 三大设计原则：场景化菜单、三段式安全变更、Runtime 黄金通道（admin socket）
  - 数据模型（LB/Backend/Server/ACL/ConfigVersion）与功能映射
  - 配置生成器与证书合并策略、apt 安装流程、实时统计流水线
  - 完整 API 设计与前端模块拆分

**后端（Go）**：
- [x] `backend/app/model/haproxy.go` — LB/Backend/Server/ACL/ConfigVersion 模型
- [x] `backend/app/dto/haproxy.go` — 全部 HAProxy 相关 DTO
- [x] `backend/utils/haproxy/socket.go` — admin socket 客户端（disable/enable/set-weight/show stat/show info/clear counters）
- [x] `backend/utils/haproxy/parser.go` — `show stat` CSV 解析 + 版本号提取
- [x] `backend/utils/haproxy/builder.go` — 基于数据库模型生成 `haproxy.cfg`（global/defaults/frontend/backend/listen stats），支持 HTTP/TCP 模式、ACL 路由、证书合并
- [x] `backend/app/repo/haproxy.go` — 五类模型的 GORM Repo + 共用 DBOption
- [x] `backend/app/service/haproxy_install.go` — apt 安装/升级/卸载，systemd 控制，初始化默认配置 + rsyslog
- [x] `backend/app/service/haproxy.go` — 业务核心：CRUD + **三段式 ApplyChange**（生成→`haproxy -c -f` 校验→备份→写入→reload→失败自动回滚→记录版本）
- [x] `backend/app/service/haproxy_runtime.go` — 运行时通道：实时上下线/调整权重，统计聚合带 2 秒缓存
- [x] `backend/app/service/haproxy.go` — 新增 `PreviewConfig()`（只生成不应用，用于前端向导预览）
- [x] `backend/app/api/v1/haproxy.go` — 全部 HTTP Handler + 统一响应 + `operator` 上下文提取
- [x] `backend/router/router.go` — 注册所有 `/haproxy/*` 路由
- [x] `backend/init/migration/migration.go` — AutoMigrate 5 个新模型
- [x] `backend/constant/errs.go` + `backend/i18n/lang/zh.yaml` — HAProxy 专属业务错误

**前端（Vue 3 + Element Plus）**：
- [x] `frontend/src/api/modules/haproxy.ts` — 全部 API 客户端封装
- [x] `frontend/src/routers/modules/haproxy.ts` + `routers/index.ts` — 7 个路由注册
- [x] `frontend/src/layout/components/Sidebar.vue` — 顶级菜单：概览/HTTP LB/TCP LB/后端池/实时监控/原始配置/配置历史
- [x] `frontend/src/views/haproxy/status/index.vue` — 概览页：安装/升级/启停/自启/Runtime 通道/Stats 端点展示
- [x] `frontend/src/views/haproxy/components/LBList.vue` — HTTP/TCP 共用的 LB 列表+表单（SSL 证书下拉、默认后端、超时、备注）
- [x] `frontend/src/views/haproxy/components/ACLDialog.vue` — HTTP LB 的 ACL 规则管理（host/path/header/src 7 种匹配类型）
- [x] `frontend/src/views/haproxy/http-lb/index.vue` + `tcp-lb/index.vue` — 复用 LBList 组件
- [x] `frontend/src/views/haproxy/backends/index.vue` — 后端池列表（可展开查看成员）+ 完整表单（负载算法/会话保持/健康检查 7 种类型）
- [x] `frontend/src/views/haproxy/components/BackendServers.vue` — 嵌入式成员表，权重调整/上下线都走 admin socket **不 reload**
- [x] `frontend/src/views/haproxy/stats/index.vue` — 实时监控：Frontends/Backends/Servers 三张表 + 自动刷新 + 关键指标卡片
- [x] `frontend/src/views/haproxy/config/index.vue` — 原始配置三模式切换（向导预览/当前激活/自定义编辑）+ `haproxy -c` 校验 + 保存并热重载
- [x] `frontend/src/views/haproxy/history/index.vue` — 配置历史 + 详情查看 + 一键回滚
- [x] `frontend/src/i18n/zh.ts` — 新增 `commons.online/offline/failed/view` 及 `haproxy.*` 约 90 条文案

### 关键决策

- **Nginx 并存**：HAProxy 走 apt 系统包（`/etc/haproxy/` + `systemctl`），与 X-Panel 原有 Nginx 并存，互不干扰
- **配置安全**：所有变更（CRUD + 原始编辑 + 回滚）统一走 `ApplyChange`，失败自动 `rollback` 回旧版本并重载
- **运行时操作不 reload**：上下线成员、调整权重通过 admin socket 立即生效，同时更新数据库；数据库成为"意图真理源"，下次 reload 自动保持
- **SSL 证书复用**：自动把 X-Panel 已签发的 `fullchain.pem` + `privkey.pem` 合并为 HAProxy 需要的单 PEM，写入 `/etc/haproxy/certs/cert-{id}.pem`
- **ACL 分层**：ACL 规则按 priority 排序生成，匹配失败走 LB 的 `defaultBackendID`
- **版本历史**：成功/失败/回滚都记录，最多保留 50 个版本，防止无限增长

### 验证

- [x] `go build ./...` 通过
- [x] `ReadLints` 对 haproxy 后端/前端文件无错误
- [ ] 待真实 Ubuntu 服务器上端到端验证 apt 安装/配置/运行时切换/SSL 证书合并流程

### 下一步

- 在真实 Ubuntu 24 服务器上验证完整流程（apt 安装 → 创建 LB → 成员上下线不断连 → SSL 跳转 → 日志查看）
- 可选扩展：配置历史 diff 对比、日志查看器、防火墙自动开端口、跨节点同步（Agent 模式）
- 发版：以 **GitHub 上当前最新 `v*` tag** 为基准递增 PATCH（截至本段编写时最新为 `v0.7.5`，则下一发为 `v0.7.6`），推送 tag 触发 CI；勿再使用已废弃的四段式 tag 格式

---

## 2026-04-15 — Session #74：系统代理功能重构

### 完成内容

- [x] `backend/init/migration/migration.go` — 新增 `ProxyType` 默认设置，支持 `mix/http/socks5`
- [x] `backend/app/dto/setting.go` — `SettingInfo` 增加 `ProxyType` 字段
- [x] `backend/app/service/setting.go` — 重写系统代理同步逻辑：同步执行、三模式分支、写入 shell `/etc/environment` `apt` Docker daemon 和面板自身环境变量，关闭时彻底清理
- [x] `backend/server/server.go` — 启动时恢复面板进程代理环境
- [x] `frontend/src/views/setting/index.vue` — 增加代理类型选择器、动态 placeholder、SOCKS5 警告、覆盖范围说明和重启提示
- [x] `frontend/src/i18n/zh.ts` — 补充系统代理相关文案
- [x] 验证 `go build ./...` 与 `npx vue-tsc --noEmit` 通过

### 关键决策

- 默认采用 `mix` 模式适配 Xray/Clash 常见中国服务器场景，用户只需填写 `host:port`
- `apt` 和 Docker 仅在存在 HTTP 能力时写入配置，纯 SOCKS5 模式自动跳过并在前端明确提示
- 面板进程通过 `os.Setenv` 在保存后立即生效，并在启动时自动恢复

### 遗留问题

- `go vet` 仍报告 `backend/app/service/host.go` 和 `backend/app/service/node.go` 的 IPv6 地址格式历史问题，非本次改动引入

### 下一步

- 在真实带代理的 Linux 服务器上验证 mix/http/socks5 三种模式的启停效果
- 验证 Docker daemon 和 `apt update` 在启停代理后的行为是否符合预期

---

## 2026-04-13 — Session #73：安装时预设管理员账户

### 完成内容

- [x] `backend/cmd/server/main.go` — 加入子命令分发，支持 `xpanel setup` 子命令
- [x] `backend/cmd/server/setup.go` — 新增 `setup` 子命令，接受 `--username` / `--password` 参数，初始化配置+数据库后写入 bcrypt 哈希密码
- [x] `scripts/install-online.sh` — 新增 `--username, -u` 和 `--password, -P` 可选参数，安装完成后调用 `xpanel setup` 完成账户预设；安装完成提示根据是否预设账户显示不同信息
- [x] `README.md` — 补充新参数说明和示例
- [x] `.gitignore` — 新增 `.cursor/`、`.kiro/`、`*.log` 排除规则，并从 git 追踪中移除已提交的 `.cursor` 和 `1.log`
- [x] 发布 `v0.6.0.2`

### 关键决策
- 密码哈希在 Go 二进制内完成（bcrypt），安装脚本不依赖外部加密工具
- `--username` / `--password` 完全可选，不传则保持原有首次打开面板初始化的流程不变

### 下一步
- 验证 `v0.6.0.2` 编译产物中 `setup` 子命令正常工作

---

## 2026-04-12 — Session #72：历史监控功能

### 完成内容

**后端 — 独立监控数据库**：
- [x] `global.MonitorDB` — 独立 `monitor.db`，避免与主业务库锁竞争
- [x] 数据模型：`MonitorBase`（CPU%/内存%/负载）、`MonitorIO`（磁盘读写速率）、`MonitorNetwork`（网络上下行速率）
- [x] `InitMonitorDB()` + AutoMigrate

**后端 — 采集 Service**：
- [x] `MonitorHistoryService.Run()` — gopsutil 采集 CPU/内存/负载
- [x] IO/网络差分采样 — channel 两次采样计算速率（参考 1Panel 模式）
- [x] cron 注册 — `@every {interval}s` 动态注册
- [x] 过期清理 — 每次采集结束按 `MonitorStoreDays` 删除过期数据
- [x] `StartMonitorCollector()` — 统一的启动/重启/停止逻辑

**后端 — API 路由**：
- [x] `POST /monitor/history` — 按时间范围查询历史数据
- [x] `GET /monitor/setting` — 获取监控设置
- [x] `POST /monitor/setting/update` — 更新设置（动态开关/间隔调整）
- [x] `POST /monitor/history/clean` — 清空监控数据
- [x] `GET /monitor/io-options` — 可选 IO 设备列表
- [x] `GET /monitor/network-options` — 可选网卡列表

**后端 — 默认设置**：
- [x] MonitorStatus=enable, MonitorInterval=300, MonitorStoreDays=7, DefaultNetwork=all, DefaultIO=all

**前端 — 历史图表页**：
- [x] 监控页新增 Tab 切换（实时 / 历史），保留原有实时概览
- [x] 时间范围选择器 + 快捷按钮（1h/6h/24h/7d）
- [x] 5 个 ECharts 折线图：负载（Load1/5/15 三线）、CPU、内存、磁盘IO（读/写）、网络（上/下行）
- [x] 所有图表支持 dataZoom（内置缩放 + 底部滑块）
- [x] IO/网络设备选择下拉

**前端 — 监控设置 UI**：
- [x] 齿轮按钮打开设置对话框
- [x] 启用/禁用监控采集 Switch
- [x] 采集间隔选择（60s/120s/300s/600s）
- [x] 保留天数选择（1/3/7/14/30 天）
- [x] 默认网卡/IO 设备选择
- [x] 清空监控数据按钮（带确认）

### 关键决策
- 独立 `monitor.db` 文件，与主库分离，参考 1Panel 验证过的方案
- IO/网络速率通过两次采样差分计算，确保数据准确性
- 默认 5 分钟间隔 × 7 天保留，数据量约 600KB，SQLite 无压力

### 下一步
- 部署验证历史数据采集和图表展示
- 观察长时间运行下的数据库大小和性能

---

## 2026-04-12 — Session #71：样式和功能优化合集

### 完成内容

**默认主题 + 侧边栏**：
- [x] `bgPreset` 默认值从 `abyss` 改为 `void`（纯黑）
- [x] `sidebarWidth` 默认值从 `default` 改为 `narrow`（只显示图标）

**隐藏多节点入口**：
- [x] 侧边栏注释掉节点管理菜单项（保留路由和后端 API 不动）
- [x] 顶栏注释掉节点切换下拉框

**监控页面重构**：
- [x] 完全重写 `host/monitor/index.vue`，参照首页三列布局风格
- [x] 三列：资源占用（CPU/内存/负载/磁盘进度条）| 网络（IP + 实时速率）| Top 进程表
- [x] 下方保留磁盘详情表（含 Inode 使用率）
- [x] 去掉原有仪表盘风格，改为紧凑进度条

**本地块设备挂载**：
- [x] 后端新增 `ListBlockDevices()` — 调用 `lsblk -Jb` 解析 JSON
- [x] 后端新增 `MountLocal()` / `UnmountLocal()` — mount/umount + 可选写 fstab
- [x] 新增 DTO：`BlockDevice`、`LocalMountRequest`、`LocalUnmountRequest`
- [x] 新增路由：`GET /disk/block-devices`、`POST /disk/local/mount`、`POST /disk/local/unmount`
- [x] 前端磁盘管理页新增块设备表格（树形缩进展示 disk → part）
- [x] 未挂载设备显示「挂载」按钮，弹窗选择挂载点和文件系统
- [x] 已挂载设备显示「卸载」按钮，带确认弹窗

### 涉及文件
- `frontend/src/store/modules/global.ts`（默认主题/侧边栏）
- `frontend/src/layout/components/Sidebar.vue`（隐藏节点菜单）
- `frontend/src/layout/components/Header.vue`（隐藏节点切换）
- `frontend/src/views/host/monitor/index.vue`（完全重写）
- `backend/app/dto/disk.go`（新增 BlockDevice/LocalMount DTOs）
- `backend/app/service/disk.go`（新增 ListBlockDevices/MountLocal/UnmountLocal）
- `backend/app/api/v1/disk.go`（新增 3 个 handler）
- `backend/router/router.go`（新增 3 条路由）
- `frontend/src/api/modules/disk.ts`（新增 3 个 API 调用）
- `frontend/src/views/host/disk/index.vue`（新增块设备 UI）
- `frontend/src/i18n/zh.ts`（块设备翻译）

### 关键决策
- 多节点功能仅前端隐藏，后端保留完整 API 和路由，便于后续恢复
- 监控页去掉系统信息卡片（首页已有），避免重复
- 块设备挂载仅支持已有文件系统的分区挂载，不支持格式化（避免误操作）

---

## 2026-04-12 — Session #70：上传修复 + 文件预览功能

### 完成内容

**上传 400 修复**：
- [x] 上一轮删除 `Content-Type: multipart/form-data` 后，Axios 全局默认 `application/json` 接管，Go 后端无法解析 FormData
- [x] 改为 `headers: { 'Content-Type': undefined }` 显式清除默认值，让浏览器自动设置含 boundary 的 multipart header

**文件预览功能**：
- [x] 新建 `FilePreview.vue` 通用预览弹窗组件
- [x] 图片预览：`<el-image>` 带缩放和预览列表
- [x] 视频预览：HTML5 `<video>` 原生播放器，支持 mp4/webm/ogg/mov/mkv
- [x] 音频预览：HTML5 `<audio>` 原生播放器，支持 mp3/wav/flac/aac/m4a
- [x] PDF 预览：`<iframe>` 内嵌浏览器 PDF 查看器
- [x] Excel/CSV 预览：安装 `xlsx` 库，fetch 文件 → 解析 → `<el-table>` 渲染，支持多 Sheet 切换
- [x] 操作列添加「预览」按钮（绿色，可预览文件才显示）
- [x] 右键菜单添加「预览」选项
- [x] 双击文件行为：可预览文件优先打开预览，其次打开编辑器

### 涉及文件
- `frontend/src/api/modules/file.ts`（Content-Type 修复）
- `frontend/src/views/host/file/FilePreview.vue`（新增）
- `frontend/src/views/host/file/index.vue`（集成预览入口）
- `frontend/src/i18n/zh.ts`（预览翻译）
- `frontend/package.json`（xlsx 依赖）

---

## 2026-04-12 — Session #69：首页重构与多项 Bug 修复

### 完成内容

**文件上传进度修复**：
- [x] 修复 Vue 响应式断裂 Bug：`doUploadFiles` 中 `items` 引用原始对象而非响应式代理，导致进度始终 0%
- [x] 改为从 `uploadQueue.value` 取响应式引用，进度实时更新
- [x] 删除手动设置的 `Content-Type: multipart/form-data`，让 Axios 自动处理 boundary
- [x] 上传全部完成 3 秒后自动清理进度浮窗

**浏览器标签页 title 动态化**：
- [x] `App.vue` 新增 `watchEffect` 动态设置 `document.title`
- [x] 格式：`页面名 - 面板名称`，解决多服务器标签页无法区分的问题

**首页三合一卡片重构**：
- [x] 原 Card1（系统信息+网络）和 Card2（资源+磁盘）合并为单卡片三等分
- [x] 布局：资源占用（左）| 网络（中）| 系统信息（右）
- [x] 资源占用 CPU/内存/Load/磁盘全部竖向排列，每项一行
- [x] 网络 IP 使用 `grid-template-columns: auto 1fr` 实现左对齐
- [x] 系统信息改为单列竖排，去除之前的 auto-fill grid
- [x] 响应式：<1200px 降为单列堆叠

**顶栏时钟兼容性修复**：
- [x] 后端 `getTimezone()` 增加 3 层 fallback：`/etc/localtime` 符号链接 → `timedatectl` → `TZ` 环境变量
- [x] 确保各种 Linux 发行版/容器都能返回有效的 IANA 时区名
- [x] 前端新增 `showServerClock` 开关（默认开启），纳入个性化设置

**仪表盘刷新间隔可配置**：
- [x] global store 新增 `dashboardRefreshInterval`（默认 5000ms）
- [x] 设置页新增下拉选择：2s / 5s / 10s / 30s / 不自动刷新
- [x] 首页 `watch` 该值动态重建 interval

**个性化设置同步完善**：
- [x] `showServerClock` 和 `dashboardRefreshInterval` 加入 `getAppearanceKeys`、`loadAppearanceFromBackend`、`persist.pick`
- [x] `App.vue` 的 appearance watch 列表同步更新
- [x] 确认所有 16 个外观字段在三处（同步/加载/持久化）完全一致

### 关键决策
- HTTPS 自动跳转需额外监听端口，用户确认暂不实施
- 面板名称默认值初始化时取自 hostname，之后为独立设置项（设计正确，无需修改）

### 涉及文件
- `frontend/src/views/home/index.vue`（完全重写）
- `frontend/src/views/host/file/index.vue`（上传进度修复）
- `frontend/src/api/modules/file.ts`（删除手动 Content-Type）
- `frontend/src/App.vue`（title 动态化 + 新字段 watch）
- `frontend/src/store/modules/global.ts`（新增 2 字段 + 同步）
- `frontend/src/views/setting/index.vue`（新增 2 个设置项）
- `frontend/src/layout/components/Header.vue`（时钟开关）
- `frontend/src/i18n/zh.ts`（新增翻译）
- `backend/app/service/monitor.go`（时区 fallback）

---

## 2026-04-09 — Session #68：多项体验增强

### 完成内容

**顶栏时间修复**：
- [x] 后端返回 `"America/Los_Angeles (PDT)"` 格式的 timezone，`Intl.DateTimeFormat` 需要纯 IANA 格式
- [x] 前端 `extractIANA()` 提取 IANA 部分，时间格式改为 `zh-CN`（年月日时分秒）

**Fail2ban 增强**：
- [x] 封禁列表新增「封禁时长」和「预计解封时间 (CST)」两列
- [x] 后端 `parseBanDuration()` 支持 `90d`、`10m`、`-1`（永久）等 fail2ban 格式
- [x] 封禁时间统一转换为 `Asia/Shanghai` 上海时区显示
- [x] SSH 端口默认改为空值（自动检测 sshd 实际监听端口），提示文字更新

**外观设置持久化到后端**：
- [x] 新增 `AppearanceConfig` Setting 键，存储 JSON 格式的外观配置
- [x] `global.ts` store 新增 `syncAppearanceToBackend()` 和 `loadAppearanceFromBackend()` 方法
- [x] 登录后自动从后端加载外观配置，修改时 1.5s 防抖自动同步
- [x] 解决了换浏览器外观设置丢失的问题

**Nginx 日志 IP 下钻**：
- [x] 后端 drilldown 新增 `ip` 过滤类型：按 IP 统计访问的 URL
- [x] Top IP 排行表格中 IP 可点击，弹窗展示该 IP 访问的所有 URL
- [x] 威胁 IP 排行同理，IP 可点击下钻

### 版本
- v0.5.68

---

## 2026-04-09 — Session #67：证书同步修复 + 路由加载条

### 完成内容

**证书同步 — 关键 Bug 修复**：
- [x] 修复 `fetchRemoteCerts` 中响应码判断错误：后端返回 `code: 0` 表示成功，但客户端判断 `code != 200` 导致所有同步都失败（报 "server returned code 0"）
- [x] 同步连接失败时也写入 `CertSyncLog`（domain 为 `*`），确保即使连接层出错也有日志可查

**前端 — 路由切换加载条**：
- [x] `App.vue` 添加顶部 accent 色加载条（2px，scaleX 动画）
- [x] `beforeEach` 开始加载条，`afterEach` 结束，8s 超时自动关闭
- [x] 配合之前的 `cancelAllPendingRequests` 实现完整的路由切换体验优化

### 版本
- v0.5.67

---

## 2026-04-09 — Session #65：证书同步功能

### 完成内容

**后端 — 数据模型**：
- [x] 新建 `CertSource` 模型：证书源配置（名称、地址、Token、同步间隔、冲突策略、同步后命令）
- [x] 新建 `CertSyncLog` 模型：同步日志（域名、状态、消息、关联证书 ID）
- [x] migration 自动迁移新表，初始化 `CertServerEnabled` / `CertServerToken` 设置项

**后端 — 证书服务端（被拉取方）**：
- [x] `CertServerAuth` 中间件：校验 `CertServerEnabled` 开关 + `X-Cert-Token` 令牌
- [x] `GET /api/v1/cert-server/certs`：暴露所有已签发证书（PEM + 私钥），供远程拉取
- [x] `CertServerService`：GetSetting / UpdateSetting（开关 + Token 管理）

**后端 — 证书源管理（拉取方）**：
- [x] `CertSourceRepo` + `CertSyncLogRepo`：完整 CRUD + 分页
- [x] `CertSourceService`：Create / Update / Delete / GetList / Sync / SyncAll / TestConnection
- [x] 同步逻辑：HTTPS 拉取远程证书列表 → 逐个对比本地 → 根据冲突策略（skip/overwrite）决定是否覆盖 → 保存到 DB + 磁盘
- [x] 同步后动作：优先执行用户自定义命令（如 `systemctl reload nginx`），否则自动 reload Nginx
- [x] 支持链式传播：B 从 A 同步的证书存入本地 Certificate 表，B 开启证书服务后 C 可以从 B 拉取

**后端 — 定时同步**：
- [x] Cron 每 10 分钟检查所有证书源，根据各源的 `syncInterval` 决定是否触发同步

**后端 — API & 路由**：
- [x] 证书源 CRUD：`GET/POST /cert-sources`、`/cert-sources/update|del|sync|test`
- [x] 同步日志：`POST /cert-sync/logs`
- [x] 证书服务设置：`GET/POST /cert-server/setting`

**前端**：
- [x] SSL 管理页新增「证书同步」Tab：证书源列表、CRUD 对话框、立即同步、查看同步日志
- [x] SSL 管理页新增「证书服务」Tab：启停开关、Token 管理、一键生成 Token
- [x] `cert-sync.ts` API 模块，`interface/index.ts` 新增类型
- [x] `zh.ts` 新增 30+ 证书同步相关 i18n 键

### 关键设计决策

- **Token 认证**：证书服务端不复用面板 JWT，使用独立 `X-Cert-Token` 头认证，避免暴露面板登录凭据
- **冲突策略**：skip 模式下，本地证书存在且到期时间 >= 远程时跳过；overwrite 模式始终以远程为准
- **链式传播**：同步来的证书以 `type=synced` 存入本地 Certificate 表，开启证书服务后自动对外提供，支持 A→B→C 链式传播
- **HTTPS 跳过验证**：面板间通信使用 `InsecureSkipVerify: true`，因为面板通常使用自签名证书
- **同步间隔灵活**：每个源可独立设置同步间隔（分钟），设为 0 仅手动同步

### 遗留/后续

- 可考虑添加「选择性同步」（仅同步指定域名的证书）
- 可考虑同步日志自动清理（保留最近 N 条）

---

## 2026-04-08 — Session #64：外观自定义系统 + UI 优化

### 完成内容

**外观自定义系统（完整主题引擎）**：
- [x] `global.ts` 新增 11 个外观持久化字段（bgPreset, uiFont, uiDensity, borderRadiusPreset, reduceMotion, termTheme, termFont, termFontSize, termBgOpacity, cardBorderStyle, sidebarWidth）
- [x] 新建 `appearance.ts` — 5 套背景预设（深渊/纯黑/微染/星空/暖夜）、4 种字体、3 档密度、3 档圆角、3 种卡片边框风格、3 档侧边栏宽度
- [x] `applyAppearance()` 统一通过 CSS 变量注入所有外观设置
- [x] `App.vue` 添加 watch 监听所有外观 store 字段，onMounted 统一应用

**终端自定义**：
- [x] `terminal-theme.ts` 扩展为 5 套终端配色（X-Panel/Dracula/One Dark/Solarized Dark/Monokai）
- [x] 5 种终端字体预设（JetBrains Mono/Fira Code/Cascadia Code/Consolas/系统等宽）
- [x] 终端字号从 localStorage 迁移到 Pinia store（设置页 slider 实时调节）
- [x] 终端背景透明度支持（applyBgOpacity 函数）
- [x] `terminal/index.vue` 和 `terminal-dialog.vue` 均读取 store 配置，watch 实时切换

**CSS 基础层调整**：
- [x] `_variables.scss` 默认色阶调整为方案 B（更深的 base #080a10、更亮的 surface #141c2b）
- [x] 新增 `--xp-bg-main-gradient` 变量供背景预设切换
- [x] 新增 `--xp-font-size`、`--xp-spacing`、`--xp-form-margin` 密度变量
- [x] `_components.scss` 支持 `data-card-border` 属性切换卡片边框风格（accent-left/full/shadow-only）
- [x] 新增全局 `.reduce-motion` class 禁用动画
- [x] `AppMain.vue` 背景改用 `var(--xp-bg-main-gradient)` CSS 变量

**设置页整合**：
- [x] 6 张卡片合并为 3 张：版本信息 / 外观与个性化 / 面板与安全
- [x] 面板与安全使用 `el-collapse` 折叠面板（面板设置/端口/Agent/账户密码）
- [x] 外观卡片包含完整自定义 UI：主题模式、强调色、背景预设色块、字体/密度/圆角/卡片边框/侧边栏宽度分段选择、减弱动画开关、终端配色预览卡片、终端字体/字号/透明度 slider

**侧边栏 accent 分割线**：
- [x] `Sidebar.vue` border-right 改为 `rgba(accent-rgb, 0.2)` accent 色分割线

**服务器时区时钟**：
- [x] `ServerInfo` 接口新增 `timezone` 字段
- [x] `Header.vue` 使用 `Intl.DateTimeFormat` + 1 秒 timer 显示实时时区时间
- [x] 格式 `HH:mm:ss (PDT)`，tooltip 显示完整时区名

**首页超宽屏适配**：
- [x] `.kv-grid` 改为 `minmax(200px, 1fr)` 自动填充列
- [x] `@media (min-width: 1600px)` 系统信息区域扩展为 3:1 比例

**i18n**：
- [x] zh.ts 新增 18 个外观相关翻译 key

### 关键决策
- 外观系统完全复用 accent 色已有模式（Store persist → App.vue watch → CSS 变量注入），零新架构成本
- 背景预设全部使用静态 CSS gradient，零持续 GPU 消耗
- 终端配色采用 xterm.js ITheme 接口标准，可自由扩展

### 下一步
- 推送到 GitHub 触发编译发布
- 根据用户反馈微调背景色阶和预设

---

## 2026-04-08 — Session #63：SPA 404 修复 + GPU 优化 + Fail2ban 修复 + 全局样式统一

### 完成内容

**SPA 404 修复**：
- [x] `setupFrontend` 的 NoRoute handler 使用 `c.Data(200, ...)` 替代 `c.Writer.Write()`，修复 Gin 默认 404 状态码问题

**GPU 占用优化**：
- [x] 移除首页卡片 `::after` 伪元素 radial-gradient（持续合成层）
- [x] 移除资源指示点 `box-shadow: 0 0 6px` 发光（持续 GPU 绘制）
- [x] 移除进度条 `.bar-fg::after` 高光伪元素
- [x] 移除快速入口 `::before` radial-gradient + `transform` + `box-shadow` 动效
- [x] Drawer/Dialog header 移除 `::after` 伪元素指示线，改用 `border-bottom` 实色线
- [x] Drawer/Dialog header 移除 `linear-gradient` 背景，用纯色背景

**全局卡片样式统一**：
- [x] `_components.scss` 全局 `.el-card` 添加左侧 accent 色边框（`border-left: 2px solid accent`）
- [x] hover 效果简化为 `border-color` 变化 + 简单 `box-shadow`，无 `transform`
- [x] 移除 `inset` 内阴影和 `transform: translateY` 浮动效果
- [x] 所有页面（文件管理、Nginx 管理等）自动继承美化效果

**Fail2ban unban 修复**：
- [x] `Unban()` 先调用 `fail2ban-client set <jail> unbanip <ip>` 解除 fail2ban 封禁
- [x] 再调用 `ensureNftBanInfra()` 确保 nftables 表存在后再删除元素
- [x] 忽略 "No such file or directory" 错误（IP 不在 nft set 中属正常）
- [x] 只有两种方式都失败才报错

### 关键设计决策

- **GPU 优化原则**：仅使用 `border-color`、`background-color`、`box-shadow` 等不需要创建合成层的属性做 hover 效果；避免 `::before`/`::after` 伪元素 + gradient + transition 组合
- **全局 vs 页面样式**：卡片基础美化放在全局 `_components.scss`，页面只做布局微调，确保全站一致性
- **Fail2ban unban 双通道**：IP 可能来自 fail2ban jail 或面板手动封禁，两个通道都尝试解封

---

## 2026-04-08 — Session #62：UI 深度美化 + 文件上传体验 + 默认值优化

### 完成内容

**文件上传体验优化**：
- [x] 新增浮动上传进度面板（右下角固定定位）
- [x] 支持多文件上传时显示每个文件的独立进度条
- [x] 上传 API 增加 `onUploadProgress` 回调支持
- [x] 上传完成后显示成功/部分失败通知
- [x] 拖拽上传同样显示进度

**ACME/SSL 密钥类型默认值优化**：
- [x] 证书申请默认密钥类型从 RSA 2048 改为 EC P256（现代设备最佳选择）
- [x] ACME 账户创建默认密钥类型从 RSA 2048 改为 EC P256

**自动升级默认开启**：
- [x] 修改 migration 默认值从 `disable` 改为 `enable`

**默认强调色更新**：
- [x] Neon 预设主色从 `#14FF4F` 改为 `#41FB44`
- [x] 同步更新 hover、muted、glow 及 Element Plus 主色阶梯

**首页卡片渐变效果修复**：
- [x] 将不可见的 `linear-gradient(0.04 opacity)` 替换为 `radial-gradient` 方案
- [x] 使用 `::after` 伪元素实现右上角弧形光晕效果，opacity 提升到 0.06
- [x] 添加 hover 时边框发光和阴影效果

**首页快速入口防火墙图标**：
- [x] 新建 `components/icons/ShieldIcon.vue` 自定义盾牌+勾 SVG 图标
- [x] 使用 `markRaw` 传递组件引用解决 Element Plus 无 Shield 图标问题

**UI 深度美化**：
- [x] 首页卡片 section header 添加强调色左边条 + 底部分隔线
- [x] 资源指示点添加发光效果（box-shadow）
- [x] 进度条末端添加高光渐变效果
- [x] 快速入口卡片添加悬浮光晕动效 + 更大图标区域
- [x] Drawer/Dialog 头部添加渐变背景 + 强调色底部指示线
- [x] Drawer/Dialog 底栏添加暗色背景增强层次感
- [x] 备份 Drawer 添加分组标题（基本信息 / 连接配置）
- [x] 备份对话框类型选择改为 Radio Button 组（含图标）
- [x] 备份类型选项添加图标（文件夹/云/连接/分享）
- [x] 计划任务 Drawer 添加分组标题（基本设置 / 执行计划 / 任务配置 / 高级选项）

### 关键设计决策

- **EC P256 作为默认密钥类型**：比 RSA 2048 更快、更安全、密钥更短，所有现代浏览器和客户端均支持
- **文件上传进度使用固定定位浮窗**：不阻塞文件管理操作，用户可继续浏览文件
- **卡片渐变使用 radial-gradient**：比 linear-gradient 更自然，集中在右上角形成视觉焦点
- **ShieldIcon 独立组件**：Element Plus 无 Shield 图标，通过 markRaw 传递组件实现混合渲染

### 遗留问题
- 无

---

## 2026-04-08 — Session #61：备份系统全面修复 + 压缩加密增强

### 完成内容

**备份系统 Bug 修复**：
- [x] 修复 `backupWebsite` 目录不存在时错误回退到 nginx conf.d 的问题 — 改为查询 Website 模型的 SiteDir
- [x] 修复 `backupDatabase` 多服务器时固定用 `servers[0]` 可能连错服务器 — 改为遍历查找数据库实例所在服务器
- [x] 修复 `Backup()` 失败时 `filepath.Base(errorMessage)` 写入垃圾 BackupRecord 数据
- [x] 修复计划任务 website/directory 无备份账号时直接失败 — 新增 `localBackupTar` 支持本地备份

**备份压缩格式选择**：
- [x] 新建 `utils/backup/archive.go` 统一处理打包/压缩/加密逻辑
- [x] 支持 gzip（默认）、zstd（更快更小）、xz（最高压缩率）三种格式
- [x] Cronjob 模型/DTO 新增 `CompressFormat` 字段
- [x] 前端计划任务表单新增压缩格式选择器

**备份加密压缩**：
- [x] 支持 openssl AES-256-CBC + PBKDF2 加密备份文件
- [x] Cronjob 模型/DTO 新增 `EncryptPassword` 字段
- [x] 前端计划任务表单新增加密密码输入框
- [x] 提供 `DecryptFile` 函数支持恢复时解密

**排除规则生效**：
- [x] 将 `ExclusionRules`（已有字段但未使用）转换为 `tar --exclude` 参数
- [x] 支持每行一条规则（如 `*.log`、`node_modules`、`.git`）
- [x] 前端计划任务表单新增排除规则文本框

### 关键设计决策

- **统一归档工具**：所有备份路径（计划任务本地备份、备份管理页）都通过 `utils/backup.CreateArchive` 统一处理，避免重复代码
- **扩展名自动调整**：根据压缩格式和是否加密自动生成正确扩展名（如 `.tar.zst.enc`）
- **加密方案选择 openssl**：几乎所有 Linux 系统预装，无需额外依赖
- **zstd 压缩**：Debian 12+ 内核自带 zstd 支持，tar 原生支持 `--zstd`

### 遗留问题

- RetainCopies 只清理数据库记录不清理磁盘备份文件
- 备份管理页手动备份暂未暴露压缩/加密选项（仅计划任务支持）

---

## 2026-04-08 — Session #60：数据库备份链路修复 + 面板增强

### 完成内容

**数据库备份链路修复（严重 Bug）**：
- [x] 修复计划任务 database/website/directory 类型备份空实现 — 原代码走 `default` 分支只写假成功日志，实际不执行任何备份
- [x] `execDatabaseBackup`：支持通过备份账号上传或本地直接备份两种路径
- [x] `execWebsiteBackup`、`execDirectoryBackup`：通过备份服务执行实际备份
- [x] 新增数据库恢复 API — `POST /databases/instances/restore`，暴露已有的 MySQL `mysql` / PostgreSQL `pg_restore` 恢复能力
- [x] 前端数据库实例列表新增「恢复」按钮和对话框（输入备份文件路径）
- [x] i18n 新增数据库恢复相关翻译

**面板名称默认 hostname**：
- [x] 初始化 migration 中 `PanelName` 默认值从硬编码 `"X-Panel"` 改为 `os.Hostname()` 获取系统主机名
- [x] 回退值仍为 `"X-Panel"`（获取主机名失败时）

**面板自动升级（含开关）**：
- [x] 新增 `AutoUpgrade` 设置项（enable/disable），默认关闭
- [x] 后端 Cron 每天凌晨 3:30 检查设置，若启用则自动检测并升级到最新版本
- [x] 前端设置页版本信息区新增自动升级开关，即时生效
- [x] SettingInfo DTO 和前端接口同步添加 `autoUpgrade` 字段

**首页显示系统运行时间**：
- [x] 首页系统信息卡片新增「运行时间」显示（X天 X时 X分格式）
- [x] 后端 `uptime` 字段已存在于 SystemStats，前端直接使用

### 关键设计决策

- **计划任务数据库备份双路径**：有备份账号时走 `BackupService.PerformBackup`（上传到云/本地账号路径），无备份账号（`targetAccountID=0`）时走 `DatabaseService.BackupInstance`（本地 `DataDir/backup/database/`）
- **自动升级使用 cron 而非 systemd timer**：复用已有的 Go cron 框架，设置通过数据库持久化，无需重启 cron 进程
- **恢复 API 接受文件绝对路径**：因备份文件在服务器本地，无需上传流程，直接传路径恢复

### 遗留问题

- 计划任务前端缺少备份账号选择器（`targetAccountID` 永远为 0），需要后续补充
- 数据库备份后没有 HTTP 下载接口，只返回服务器路径
- 备份管理页中 `backupDatabase` 使用 `servers[0]`，多服务器时可能连错

---

## 2026-04-08 — Session #59：UI 修复 + 进程管理网络视图 + SSH 公钥增强

### 完成内容

**UI 渐变 Bug 修复**：
- [x] 修复首页卡片渐变不生效问题 — `--xp-bg-card` CSS 变量未在 `_variables.scss` 中定义，导致 `linear-gradient` 声明无效回退为纯色
- [x] 在 `_variables.scss` 的 Backgrounds 区域补全 `--xp-bg-card: #111827`

**进程管理 — 网络监控增强**：
- [x] 新增「监听端口」视图 — 筛选 LISTEN 状态连接，展示端口/协议/监听地址/进程名/连接数
- [x] 新增「活跃连接」视图 — 按远程 IP 聚合 ESTABLISHED 连接，支持展开查看详情
- [x] 增强「全部连接」视图 — 新增状态/协议/地址多维筛选，顶部统计卡片（LISTEN/ESTABLISHED/TIME_WAIT 计数）
- [x] 进程列表 Tab 新增自动刷新（3s/5s/10s 可选）、输入防抖搜索
- [x] 网络视图共享自动刷新机制
- [x] i18n 新增 15+ 条进程/网络相关翻译

**SSH authorized_keys 多方式添加**：
- [x] 添加公钥对话框支持三种模式：粘贴文本、上传 `.pub` 文件、从面板已有密钥中选择
- [x] 私钥管理列表新增「部署」按钮 — 一键将公钥添加到 authorized_keys
- [x] i18n 新增 SSH 相关翻译

### 关键设计决策

- **网络视图拆分为三 Tab**：监听端口（最常用）、活跃连接（按 IP 聚合直观看谁在连）、全部连接（原始数据）
- **数据复用**：三个网络视图共用同一个 connections 数据源（一次 API 调用），前端 computed 过滤
- **公钥文件上传**：使用 FileReader 在前端读取文件内容，无需后端新增上传 API
- **密钥部署**：复用已有的 `addAuthorizedKey` API，前端串联密钥库数据实现闭环

### 需求分析（本次评估但未实施）

- 终端主机导出：可做但优先级不高，涉及密码导出安全问题
- 防火墙 iptables：当前仅支持 UFW，直接管理 iptables 需大量工作，建议短期引导安装 UFW
- Gost 配置路径：确认为 `/opt/xpanel/gost/gost.yaml`，合理无需改动
- ACME 账户预设：建议做引导式创建而非静默预设

---

## 2026-04-07 — Session #58：Nginx 日志分析功能

### 完成内容

**Nginx 日志分析（全局/按站点）**：
- [x] 后端：Nginx 配置文件解析器 — 自动检测所有 server 块的 `server_name`、`access_log`、`error_log`
- [x] 后端：支持 `conf.d/` 和 `sites-enabled/` 两种目录结构
- [x] 后端：access log 解析器（combined 格式）+ 统计聚合（请求趋势/状态码/Top IP/URL/UA）
- [x] 后端：大文件优化 — 从尾部反向读取，按时间范围截断
- [x] 后端：Top IP 自动附带 IP 归属地（复用 MMDB 数据库）
- [x] 后端 API：`GET /nginx/log/sites`、`POST /nginx/log/analyze`、`POST /nginx/log/tail`
- [x] 前端：Nginx 管理页新增「日志分析」Tab（lazy 加载）
- [x] 前端：站点选择器（"全部汇总" + 各检测到的站点）
- [x] 前端：时间范围选择器（1h / 6h / 24h / 7d / 30d）
- [x] 前端：统计概览 — 4 个汇总卡片 + ECharts 请求趋势图（柱状+流量折线）+ 状态码环形图
- [x] 前端：Top IP 表格（含归属地列）、Top URL 表格、Top User-Agent 表格
- [x] 前端：访问日志 / 错误日志实时查看器（tail 模式，支持行数选择）
- [x] i18n 完整翻译

### 关键设计决策

- **源码模式兼容**：不依赖数据库中的站点记录，直接从 Nginx 配置文件解析站点信息
- **集成在 Nginx 管理页**：不新增菜单入口，作为第三个 Tab 与"运行状态""配置编辑"并列
- **全部汇总模式**：合并多个站点的日志文件进行统一分析

### 版本

- **v0.5.55** — 已推送部署

---

## 2026-04-07 — Session #57：Fail2ban UI 优化 + 封禁问题修复

### 完成内容

**Fail2ban UI 优化**：
- [x] 检测窗口（findTime）和封禁时长（banTime）从文本输入改为下拉选择（支持自定义输入）
- [x] findTime 预设：1m / 5m / 10m / 30m / 1h / 6h / 12h / 1d
- [x] banTime 预设：10m / 1h / 6h / 12h / 1d / 7d / 30d / 90d / 365d / 永久
- [x] 默认封禁时长改为 90 天（原为 1 小时）
- [x] 改进 findTime 提示说明："在此时间内失败超过最大重试次数则封禁"
- [x] 编辑 Jail 对话框同步使用下拉选择

**Fail2ban 封禁不生效排查**：
- [x] 诊断发现：系统无 `/var/log/auth.log`（使用 systemd-journal），需显式设置 `backend = systemd`
- [x] 后端添加 `detectBackend()` 函数：自动检测应使用 auto 还是 systemd
- [x] `SetSSHJail` 和 `ensureJailLocal` 现在生成配置包含 `backend` 参数
- [x] 服务器 jail.local 已更新，确认封禁功能正常（当前已封禁 3 个攻击 IP）

**SSH MCP 配置**：
- [x] 修改 Cursor SSH MCP 配置为私钥认证（`--key` 参数）

### 版本

- **v0.5.54** — 已推送

---

## 2026-04-07 — Session #56：SSH 私钥管理 + 主机管理优化 + 用户管理 sudo + 文件管理默认显示隐藏

### 完成内容

**SSH 私钥管理**：
- [x] 后端：`ssh_key.go` — 私钥 CRUD + RSA 密钥对生成（2048/4096 位），存储在 `{dataDir}/ssh-keys/`
- [x] 支持导入已有私钥（自动解析提取公钥）
- [x] 查看/复制公钥、查看私钥、删除密钥对
- [x] 前端：SSH 管理新增"私钥管理" Tab，含生成、导入、列表、查看、删除功能
- [x] 修复 SSH 管理页面硬编码中文

**主机管理优化**：
- [x] 操作按钮排版修复：`width: 260px` + `flex-wrap: nowrap` 防止按钮换行
- [x] 私钥认证增加"预设私钥"下拉选择：从面板管理的 SSH 密钥中选取，自动填充私钥内容

**用户管理 sudo 权限**：
- [x] 移除"创建为系统用户"选项
- [x] 新增 sudo 权限开关：创建/编辑时可将用户加入/移出 sudo 组
- [x] 自动检测系统使用 `sudo` 还是 `wheel` 组
- [x] 列表显示 sudo 标签

**文件管理**：
- [x] 隐藏文件默认显示（`showHidden` 默认值改为 `true`）

### 版本
- Tag: `v0.5.53`

---

## 2026-04-07 — Session #55：文件管理模块深度审查与修复

### 完成内容

**ZIP 压缩修复（主要 Bug）**：
- [x] 添加 `exec.LookPath("zip")` 检查，未安装时返回明确错误提示而非"服务器内部错误"
- [x] 多路径跨目录压缩：当源文件不在同一父目录时，使用临时目录 + symlink 策略正确处理
- [x] 防止输出文件落在被压缩目录内部（检查 dst 是否为任一源的子路径）
- [x] 确保目标目录存在（`MkdirAll`），使用绝对路径避免歧义

**解压修复**：
- [x] 添加 `unzip` 命令存在性检查
- [x] 单独 `.gz`/`.bz2`/`.xz` 文件不再误判为 tar 归档，改用 `gunzip -c`/`bunzip2 -c`/`xz -dc` 正确解压
- [x] 解压错误信息使用专用业务错误码 `ErrFileDecompress`

**安全修复**：
- [x] **上传路径穿越**：使用 `filepath.Base(file.Filename)` 防止 `../` 注入覆盖系统文件
- [x] **重命名保护**：对受保护系统路径（`/etc`、`/usr` 等）禁止重命名操作
- [x] **搜索注入**：转义 `find -iname` 中的特殊通配符（`[`, `]`, `?`, `*`），并添加 `-maxdepth 10` 限制

**错误信息优化**：
- [x] 新增 `ErrFileCompress`/`ErrFileDecompress` 错误码，替换笼统的 `ErrInternalServer`
- [x] 压缩/解压失败时返回工具的实际 stderr 输出作为错误详情
- [x] `ErrCmdNotFound` 提示中附带安装命令（如 `apt install zip`）

### 版本
- Tag: `v0.5.52`

---

## 2026-04-07 — Session #54：Linux 用户管理 + 系统设置（主机名/时区/DNS/Swap）

### 完成内容

**Linux 用户管理 (CRUD)**：
- [x] 后端：`host_user.go` — 解析 `/etc/passwd` 列出用户，`useradd`/`usermod`/`userdel`/`chpasswd` 实现增删改
- [x] 支持系统用户/普通用户切换显示，root 用户禁止编辑/删除
- [x] 可用 Shell 列表从 `/etc/shells` 读取，支持下拉选择或自定义输入
- [x] 系统组列表从 `/etc/group` 读取
- [x] 前端页面：用户列表 + 创建/编辑对话框 + 删除确认（可选同时删除主目录）

**系统设置**：
- [x] **主机名管理**：`hostnamectl set-hostname`，即时生效
- [x] **时区管理**：`timedatectl set-timezone`，可搜索下拉选择时区
- [x] **DNS 配置**：读写 `/etc/resolv.conf`，保留非 nameserver 行，常用 DNS 快速添加（Google/Cloudflare/阿里/腾讯/114）
- [x] **Swap 管理**：创建/删除 `/swapfile`，启用/停用，自动写入/移除 `/etc/fstab`

**路由/菜单**：
- [x] 新增 `/host/users`（用户管理）和 `/host/system`（系统设置）路由
- [x] 侧栏「系统」菜单新增两个子项

### 版本
- Tag: `v0.5.51`

---

## 2026-04-07 — Session #53：NFS 修复 + 远程挂载网络优化 + SSL 通配符路径 + Nginx 双安装管理

### 完成内容

**NFS 修复**：
- [x] 修复 NFS 开机自启状态始终显示"已禁用"：`nfsServiceName()` 依次探测 `nfs-server` → `nfs-kernel-server`

**远程挂载网络优化**：
- [x] 新增四种网络预设：默认 / 不稳定网络 / 高速局域网 / 自定义
  - NFS 不稳定网络：`rw,soft,timeo=10,retrans=2,actimeo=60,noatime`
  - CIFS 不稳定网络：`rw,soft,echo_interval=5,actimeo=30,cache=loose,nobrl,noserverino`
- [x] fstab 持久化：挂载时可选写入 `/etc/fstab`，卸载时同步移除
- [x] 卸载 fallback 到 `umount -l`
- [x] 远程挂载列表新增持久化状态列

**SSL 通配符证书路径修复**：
- [x] `*.example.com` 目录改为 `_wildcard.example.com`，避免 shell 通配符冲突
- [x] `safeDomainDir()` 统一替换所有路径引用（ssl.go / nginx_config.go / gost.go）
- [x] 启动时自动迁移旧 `*` 目录到 `_wildcard`（migration.go）
- [x] 删除时兼容清理新旧两种路径

**Nginx 双安装快捷卸载**：
- [x] `NginxUninstallReq` 增加 `Mode` 字段：可指定 `system` / `prefix` / 空（当前活跃模式）
- [x] `uninstallSystemNginx()`：apt-get remove --purge + autoremove
- [x] `uninstallPrefixNginx()`：停止进程 + 清理 systemd 服务文件 + 删除安装目录
- [x] 前端双安装警告栏增加「卸载 xxx」快捷按钮
- [x] `NginxConfig` 暴露 `HasSystemInstalled()` / `HasPrefixInstalled()` 方法

### 版本

- v0.5.40 (NFS 修复)
- v0.5.41 (远程挂载增强)
- v0.5.42 (SSL 通配符路径 + Nginx 双安装管理)
- v0.5.43 (首页仪表盘布局优化)
- v0.5.44 (首页布局重构)
- v0.5.45 (Fail2ban 可视化管理)
- v0.5.46 (Fail2ban SSH 端口自动检测修复)
- v0.5.47 (IP 归属地数据库集成)
- v0.5.48 (Nginx 双安装检测修复)
- v0.5.49 (Systemd 服务管理器)

**Systemd 服务管理器**：
- [x] 完整的 systemd 服务管理：列表/详情/启停/重启/自启/日志
- [x] 可视化创建新服务：服务名、命令、工作目录、用户、重启策略、环境变量
- [x] 面板创建的服务使用 `xp-` 前缀标识，支持编辑和删除
- [x] 系统已有服务也可管理（启停/自启/查看日志/详情）
- [x] 搜索过滤 + 显示所有/仅系统服务筛选
- [x] 服务详情：PID、内存、CPU、配置文件内容
- [x] journalctl 日志查看器

**IP 归属地数据库**：
- [x] `utils/iplocation` 包：基于 MMDB 格式的 IP 地理查询服务（oschwald/geoip2-golang）
- [x] 支持自动下载 DB-IP City Lite 免费数据库（~70MB，每月更新）
- [x] Fail2ban 封禁列表自动附带国家/省份/城市信息（中文优先）
- [x] 批量 IP 查询 API + 单 IP 查询 API
- [x] IP 数据库管理 Tab：查看状态/一键下载更新
- [x] 面板启动时自动加载已有的 MMDB 文件

**Fail2ban SSH 端口自动检测**：
- [x] `detectSSHPort()` 解析 `ss -tlnp` 输出获取 sshd 实际监听端口
- [x] 修复 Fail2ban 封禁 22 端口而 SSH 运行在自定义端口时封禁无效的问题
- [x] 去掉硬编码 `logpath=/var/log/auth.log`，兼容 systemd journal

**Fail2ban 可视化管理**：
- [x] IFail2banService 完整实现：安装/卸载/启停/jail管理/封禁管理/日志查看
- [x] jail.local 解析与安全写入（备份→写入→reload→失败回滚）
- [x] fail2ban-client 命令调用：status、jail状态、封禁列表、解封
- [x] SSH 防护快捷配置：一键设置 maxretry/findtime/bantime
- [x] 前端四 Tab 页面：SSH 防护 / 封禁列表（搜索+解封）/ Jail 管理 / 日志查看器
- [x] 10 条 API 路由 + ToolboxAPI Handler
- [x] 侧栏工具箱菜单增加 Fail2ban 条目

---

## 2026-04-07 — Session #52：工具箱 - Samba 和 NFS 共享管理

### 完成内容

**Samba 共享管理**：
- [x] smb.conf 纯 Go 解析器 (`utils/samba/parser.go`)：支持 Section 增删改查，保留注释和空行
- [x] ISambaService 完整实现：服务管理(安装/卸载/启停)、共享 CRUD、用户管理(创建/删除/启禁用/改密)、全局配置、连接监控
- [x] 安全模式：修改配置 → testparm 校验 → 失败自动回滚 → 成功 reload
- [x] 前端 Samba 管理页：四 Tab 布局(共享目录/用户管理/连接监控/全局配置)
- [x] 共享创建/编辑弹窗，用户创建/改密弹窗
- [x] 共享创建时用户下拉选择 + hosts allow/deny 安全控制
- [x] 全局配置增加协议版本控制 (min/max protocol) + 公网安全提示
- [x] smbstatus 连接监控解析修复（不依赖 -j JSON 输出）

**NFS 共享管理**：
- [x] /etc/exports 解析器 (`utils/nfs/parser.go`)：支持多客户端、选项解析
- [x] INfsService 完整实现：服务管理、导出 CRUD、连接监控
- [x] 安全模式：修改配置 → exportfs -ra 应用 → 失败回滚
- [x] 前端 NFS 管理页：两 Tab 布局(导出目录/连接监控)
- [x] 导出创建/编辑弹窗，支持动态添加多个客户端
- [x] NFS 选项可视化选择器（checkbox + tooltip 替代纯文本输入）
- [x] 操作后立即刷新状态 (`await loadStatus()`)
- [x] 连接监控增强：showmount + /proc/fs/nfsd/clients/ 双源

**通用基础设施**：
- [x] ToolboxAPI 注册到 ApiGroup + 25 条 API 路由
- [x] ServiceStatus/ServiceOperate 通用 DTO
- [x] 前端工具箱路由模块 + 侧边栏菜单项
- [x] 前端 i18n (toolbox 命名空间) + 后端 i18n 错误键 (11 条)
- [x] 前后端编译通过，零 lint 错误

### 设计决策

- **无数据库表**：所有共享信息直接读写系统配置文件 (`/etc/samba/smb.conf`、`/etc/exports`)，面板仅作 UI 层
- **apt 系统包管理**：通过 `apt install/remove` 管理服务，卸载面板不影响 Samba/NFS 运行
- **工具箱模块归属**：Samba/NFS 放在「工具箱」子菜单下，与未来 FTP/Fail2ban 并列
- **配置安全机制**：与 Nginx 一致的 "写入→校验→失败回滚→成功 reload" 模式

### 变更文件

| 文件 | 变更 |
|---|---|
| `backend/utils/samba/parser.go` | 新增：smb.conf INI 解析器 |
| `backend/utils/nfs/parser.go` | 新增：/etc/exports 解析器 |
| `backend/app/dto/toolbox.go` | 新增：Samba + NFS 全部 DTO |
| `backend/app/service/toolbox_samba.go` | 新增：ISambaService 接口 + 实现 |
| `backend/app/service/toolbox_nfs.go` | 新增：INfsService 接口 + 实现 |
| `backend/app/api/v1/toolbox.go` | 新增：ToolboxAPI 全部 Handler |
| `backend/app/api/v1/entry.go` | 新增 ToolboxAPI 嵌入 |
| `backend/router/router.go` | 新增 25 条工具箱路由 |
| `backend/i18n/lang/zh.yaml` | 新增 Samba/NFS 错误键 |
| `frontend/src/api/modules/toolbox.ts` | 新增：工具箱 API 封装 |
| `frontend/src/routers/modules/toolbox.ts` | 新增：工具箱路由 |
| `frontend/src/routers/index.ts` | 导入 toolboxRoutes |
| `frontend/src/layout/components/Sidebar.vue` | 侧边栏增加工具箱菜单 |
| `frontend/src/views/toolbox/samba/index.vue` | 新增：Samba 管理页 |
| `frontend/src/views/toolbox/nfs/index.vue` | 新增：NFS 管理页 |
| `frontend/src/i18n/zh.ts` | 新增 toolbox 命名空间翻译 |

### 下一步计划

- [ ] FTP 管理 (vsftpd/Pure-FTPd)
- [ ] Fail2ban 管理
- [ ] Samba/NFS 操作日志描述映射

---

## 2026-04-06 — Session #51：GOST TLS 证书支持 + 检查更新/升级

### 完成内容

**TLS 证书支持**：
- [x] Model 新增 `CertificateID`、`CustomCertPath`、`CustomKeyPath` 字段（GostService）
- [x] DTO/API 全链路透传证书配置
- [x] GOST Client `ListenerConfig` 新增 `TLS` 结构体（certFile/keyFile）
- [x] `buildServiceConfig()` 自动解析证书路径：自定义路径优先 → 面板证书（从 SSL 目录读取）→ 不配置
- [x] 前端 Relay 页面：创建/编辑对话框支持三种证书模式（面板证书下拉选择 / 自定义路径输入 / 不配置）
- [x] 表格新增 TLS 证书列，显示证书域名或自定义标记

**检查更新 / 升级**：
- [x] 后端 `CheckUpdate()` 方法：对比本地版本与 GitHub Releases 最新版本
- [x] 后端 `Upgrade()` 方法：异步下载 → 停止 GOST → 备份旧二进制 → 替换 → 启动 → 同步配置；失败自动回滚
- [x] API 路由：`GET /gost/check-update`、`POST /gost/upgrade`
- [x] 前端状态页新增"检查更新"卡片：显示当前/最新版本、一键升级按钮、复用安装进度条

### 变更文件

| 文件 | 变更 |
|---|---|
| `backend/app/model/gost.go` | 新增 CertificateID/CustomCertPath/CustomKeyPath 字段 |
| `backend/app/dto/gost.go` | DTO 新增证书字段 + GostCheckUpdateResp/GostUpgradeReq |
| `backend/utils/gost/client.go` | ListenerConfig 新增 TLS 配置 |
| `backend/app/service/gost.go` | resolveServiceCert() + 证书字段透传 |
| `backend/app/service/gost_install.go` | CheckUpdate() + Upgrade() + doUpgrade() |
| `backend/app/api/v1/gost.go` | CheckGostUpdate + UpgradeGost handler |
| `backend/router/router.go` | 新增 check-update/upgrade 路由 |
| `frontend/src/api/modules/gost.ts` | checkGostUpdate + upgradeGost API |
| `frontend/src/api/interface/index.ts` | GostCheckUpdateResp + 证书字段 |
| `frontend/src/views/gost/relay/index.vue` | TLS 证书选择 UI（三模式切换） |
| `frontend/src/views/gost/status/index.vue` | 检查更新卡片 + 升级进度 |
| `frontend/src/i18n/zh.ts` | 新增证书/更新相关 i18n 文本 |

---

## 2026-04-06 — Session #50：集成 GOST 代理管理功能

### 完成内容

**GOST 进程管理**：
- [x] GOST 安装/卸载/升级 Service（从 GitHub Releases 下载二进制，创建 systemd 服务）
- [x] GOST 启停/重启操作（systemctl 管理 `xpanel-gost` 服务）
- [x] 安装时自动生成 API 认证凭证，写入 gost.yaml 配置文件
- [x] 安装进度轮询（与 Nginx 安装模式一致）
- [x] API 凭证存储在 Settings 表（GostAPIAddr/GostAPIUser/GostAPIPass）

**GOST Web API Client**：
- [x] `utils/gost/client.go`：封装 GOST REST API（Service/Chain CRUD、Config 保存）
- [x] 支持 Basic Auth 认证，所有配置变更即时推送到 GOST 进程

**端口转发功能**：
- [x] 后端：Model/DTO/Repo/Service/API 全套实现（TCP/UDP 转发规则 CRUD）
- [x] 前端：端口转发页面（表格列表、创建/编辑对话框、启用/禁用开关）
- [x] 支持关联转发链实现链式代理转发

**Relay 中继服务**：
- [x] 后端：relay_server 类型服务管理（支持 tcp/tls/ws/wss 传输协议、认证配置）
- [x] 前端：中继服务页面（CRUD + 一键查看客户端连接命令）

**转发链管理**：
- [x] 后端：GostChain 模型，Hops 以 JSON 存储，格式与 GOST 原生配置一致
- [x] 前端：可视化跳跃点编辑器（每个 Hop 配置节点地址、连接器类型、传输协议、认证）
- [x] 删除保护：被端口转发规则引用的转发链不可删除

**配置同步**：
- [x] SyncAll()：全量将 DB 中的 Chain 和 Service 推送到 GOST API
- [x] GOST 启动/重启后自动触发同步
- [x] X-Panel 启动时后台异步同步（如 GOST 已运行）
- [x] 每次配置变更后调用 GOST `POST /config` 持久化到 YAML 文件

**其他**：
- [x] 数据库迁移：新增 `gost_services` 和 `gost_chains` 表
- [x] 路由注册：`/api/v1/gost/*` 路由组
- [x] i18n：前端中文翻译（gost 命名空间）、后端错误码翻译
- [x] 侧边栏：新增「代理管理」菜单（GOST 状态/端口转发/中继服务/转发链）
- [x] 操作日志：authPass 字段加入敏感字段脱敏列表

### 关键决策

- GOST 通过 Web API 动态管理（即时生效）+ YAML 文件持久化（重启恢复），不需要重启进程
- 安装路径固定为 `/opt/xpanel/gost/`，与面板其他组件（Nginx、SSL）保持一致
- API 监听地址限定 `127.0.0.1:18080`（仅本地访问），凭证随机生成
- 转发链 Hops 使用 JSON 存储，保持与 GOST 原生 YAML 配置的结构一致性
- 端口转发和中继服务共用 `gost_services` 表，通过 `type` 字段区分

### 新增文件

- `backend/app/model/gost.go` — GostService、GostChain 模型
- `backend/app/dto/gost.go` — 请求/响应 DTO
- `backend/app/repo/gost.go` — 数据库 CRUD
- `backend/app/service/gost.go` — 业务逻辑 + GOST API 调用
- `backend/app/service/gost_install.go` — 安装/卸载/升级
- `backend/app/api/v1/gost.go` — HTTP Handler
- `backend/utils/gost/client.go` — GOST Web API 客户端
- `frontend/src/api/modules/gost.ts` — 前端 API 封装
- `frontend/src/routers/modules/gost.ts` — 路由模块
- `frontend/src/views/gost/status/index.vue` — GOST 状态页
- `frontend/src/views/gost/forward/index.vue` — 端口转发页
- `frontend/src/views/gost/relay/index.vue` — 中继服务页
- `frontend/src/views/gost/chain/index.vue` — 转发链页

### 修改文件

- `backend/app/api/v1/entry.go` — 注册 GostAPI
- `backend/router/router.go` — 注册 `/gost/*` 路由
- `backend/init/migration/migration.go` — 新增 GOST 模型迁移
- `backend/server/server.go` — 启动时异步同步 GOST 配置
- `backend/constant/errs.go` — 新增 GOST 错误码
- `backend/i18n/lang/zh.yaml` — 新增 GOST 错误翻译
- `backend/middleware/operation_log.go` — authPass 加入脱敏字段
- `backend/app/repo/setting.go` — 新增 CreateOrUpdate 方法
- `frontend/src/routers/index.ts` — 注册 gost 路由模块
- `frontend/src/layout/components/Sidebar.vue` — 新增代理管理菜单
- `frontend/src/i18n/zh.ts` — 新增 gost 翻译命名空间
- `frontend/src/api/interface/index.ts` — 新增 GOST 类型定义

### 下一步计划

- 考虑在 GOST 状态页显示各服务的实时流量统计（利用 GOST enableStats）
- 支持 GOST 版本升级（类似 Nginx 升级流程）
- 考虑添加 SOCKS5/HTTP 代理服务类型

---

## 2026-04-04 — Session #49：移除 Xray 功能 + SSL 证书存储独立化

### 完成内容

**移除 Xray 代理功能**：
- [x] 删除后端 Xray 全部代码：model、dto、repo、service、api/v1
- [x] 清理路由（router.go 中 `/xray/*` 路由组）
- [x] 清理 cron 定时任务（流量同步、过期检查、每日快照）
- [x] 清理数据库迁移（XrayNode、XrayUser、XrayTrafficDaily、XrayOutbound 四张表）
- [x] 清理默认设置（XrayLogLevel、XrayAccessLog、XrayErrorLog）
- [x] 清理错误常量（ErrXrayInvalidSettings 等）和 i18n 翻译
- [x] 删除前端 Xray 页面（views/xray/）、API 模块、路由模块
- [x] 清理侧边栏菜单、前端 i18n xray 命名空间
- [x] 清理操作日志中 Xray 相关的 API 路径映射和分组名
- [x] 删除根目录 `xray-install.sh` 脚本
- [x] 清理安装脚本（install.sh、install-online.sh）中的 Xray 安装步骤

**SSL 证书存储独立化**：
- [x] 新增 `ServerConfig.GetDefaultSSLDir()` 返回 `/opt/xpanel/ssl/`（基于 DataDir）
- [x] `NginxConfig.GetSSLDir()` 改为指向独立路径，不再绑定 Nginx 目录
- [x] `CertificateService.GetSSLDir()` 和 `NginxConfigGenerator.getSSLDir()` 同步更新
- [x] Nginx 安装时不再创建 `conf/ssl` 目录
- [x] 启动时自动创建 `/opt/xpanel/ssl/{certs,logs}` 目录结构
- [x] 自动迁移：首次启动将旧路径（`/etc/nginx/ssl/certs`、`{install_dir}/conf/ssl/certs`）下的证书复制到新位置

**证书权限修复**：
- [x] `saveCertFiles()` 写入后主动 `chmod` 确保权限链完整
- [x] 证书文件 0644、私钥 0600、目录链 0755
- [x] 错误信息增强：写入失败时返回具体路径和错误

**修复证书申请"第一次失败第二次成功"的问题**：
- [x] 根因：`DisableCompletePropagationRequirement()` 跳过 DNS 传播检查，TXT 记录未传播就请求 CA 验证
- [x] 移除 `DisableCompletePropagationRequirement()`，改用 `AddRecursiveNameservers` 指定公共 DNS（1.1.1.1/8.8.8.8）做传播检查
- [x] `ObtainCertificate` 增加自动重试：首次失败后等待 30 秒再重试一次，无需用户手动操作

### 关键决策

- SSL 证书默认路径从 Nginx 内部改为 `/opt/xpanel/ssl/`（跟随 `system.data_dir`），确保删除 Nginx 后证书不丢失
- 旧证书采用复制（非移动）方式迁移，原路径保留不删除，避免正在使用的 Nginx 配置引用失效
- Nginx master 进程以 root 运行，可直接读取 root:root 0600 的私钥文件，无需 chown

### 遗留问题

- 数据库中 `xray_*` 旧表需要手动清理（AutoMigrate 不会自动删除不再引用的表）
- 用户如果自定义了 `SSLDir` 设置为 Nginx 内部路径，需手动更新

### 下一步计划

- 考虑添加数据库迁移脚本清理 xray 旧表
- 前端 SSL 目录设置页面添加路径提示

---

## 2026-03-26 — Session #48：网站配置全面增强 + SSL 证书修复

### 完成内容

**SSL 证书申请修复**（v0.5.28 ~ v0.5.29）：
- [x] 修复 Cloudflare `AuthToken` / `AuthKey` 混用导致 DNS 验证失败
- [x] 支持 Global API Key 和 API Token 两种 Cloudflare 认证模式
- [x] 添加 `DisableCompletePropagationRequirement` 跳过 DNS 传播轮询
- [x] 缩短超时：propagation 3min、dns01 5min、polling 5s、TTL 600
- [x] lego 内部日志重定向到证书日志文件（实时可查）
- [x] 前端 DNS 账户增加 Cloudflare 认证模式说明

**Nginx SSL 配置去重**（v0.5.30）：
- [x] `customHasDirective()` 函数检测自定义配置中已有的指令，自动跳过面板生成
- [x] 修复用户自定义 SSL 指令与面板默认 SSL 配置冲突（duplicate 报错）

**网站配置全面增强**（v0.5.31）：
- [x] **Gzip 压缩**：新建网站默认开启，comp_level=6，覆盖 text/css/js/json/xml/font/svg 等
- [x] **安全响应头**：默认开启 X-Content-Type-Options、X-Frame-Options、Referrer-Policy、Permissions-Policy、server_tokens off
- [x] **静态资源缓存**：可选功能，图片/字体 30天、CSS/JS/WOFF 7天，含 Cache-Control 和 access_log off
- [x] **反向代理优化**：内置 proxy buffer（8x8k）、timeout（connect 60s / read 600s）、buffering on
- [x] **SSL 升级到 Mozilla Intermediate v5.7**：ssl_ecdh_curve X25519、ssl_prefer_server_ciphers off、HSTS max-age 63072000+preload
- [x] **前端「性能优化」Tab**：Gzip/安全头/静态缓存独立开关
- [x] **自定义配置改进**：placeholder 示例 + 冲突说明

### 关键决策

- 研究了 1Panel 的配置管理方式（include 子文件 + 解析器操作配置树），X-Panel 采用更轻量的策略：把常用优化内置为可开关的功能模块 + `customHasDirective` 智能去重
- `ssl_prefer_server_ciphers` 改为 `off` — 遵循 Mozilla Intermediate 指南，让客户端选择最优密码套件
- Gzip 和安全头对新旧网站都默认开启（通过一次性数据迁移）
- 静态缓存默认关闭 — 需要用户确认其部署策略支持 cache-busting
- DNS 传播检查完全跳过（`DisableCompletePropagationRequirement`）— 对 Cloudflare 等主流 DNS 提供商，传播是秒级完成的

### 涉及文件

**后端**：
- `backend/app/model/website.go` — 新增 GzipEnable、SecurityHeaders、StaticCacheEnable 字段
- `backend/app/dto/website.go` — DTO 同步更新
- `backend/app/service/nginx_config.go` — writeGzipBlock、writeSecurityHeaders、writeStaticCacheBlock、增强 writeSSLBlock 和 writeReverseProxy
- `backend/app/service/website.go` — Create/Update 处理新字段
- `backend/init/migration/migration.go` — 一次性迁移：已有网站开启 Gzip 和安全头
- `backend/utils/ssl/acme.go` — ObtainCertificate 增加 logWriter、DisableCompletePropagation
- `backend/utils/ssl/dns_provider.go` — Cloudflare AuthToken/AuthKey 修复、超时优化

**前端**：
- `frontend/src/views/website/website/config.vue` — 新增「性能优化」Tab、自定义配置改进
- `frontend/src/views/website/ssl/index.vue` — Cloudflare 认证模式说明
- `frontend/src/i18n/zh.ts` — 新增 i18n 条目

### 版本

`v0.5.28` ~ `v0.5.31`

---

## 2026-03-25 — Session #47：修复 Cloudflare DNS 验证卡住问题

### 完成内容

**Cloudflare 认证修复**：
- [x] 修复 `AuthToken` / `AuthKey` 混用导致 lego Cloudflare provider 认证失败
- [x] 支持 Global API Key 模式（email + key）和 API Token 模式（仅 token）两种方式
- [x] 后端根据 `email` 是否为空自动选择认证方式

**DNS 验证超时优化**：
- [x] `propagationTimeout` 从 30 分钟降到 10 分钟
- [x] `dns01.AddDNSTimeout` 从 30 分钟降到 10 分钟
- [x] 申请/续签时添加 DNS 验证阶段日志提示

**lego 日志重定向**：
- [x] `ObtainCertificate` / `RenewCertificate` 接受可选 `io.Writer` 参数
- [x] 将 Go 标准 `log` 包输出重定向到证书日志文件（同时保留 stderr 输出）
- [x] DNS 验证过程中 lego 内部日志（TXT 记录创建、轮询等）可在前端日志面板实时查看

**前端 UI 改进**：
- [x] DNS 账户创建对话框：Cloudflare 选项显示认证模式说明（API Token 推荐 / Global API Key）
- [x] DNS 字段添加友好标签和 placeholder 提示

**其他**：
- [x] 清理 ACME 调试日志（`fmt.Printf` 的 DEBUG 输出）

### 关键决策

- Cloudflare 认证判断方式：email 为空 → API Token 模式（`AuthToken`），email 非空 → Global API Key 模式（`AuthEmail` + `AuthKey`）
- 超时降到 10 分钟对 Cloudflare 足够（DNS 传播通常几秒），同时避免凭证错误时卡太久

### 涉及文件

- `backend/utils/ssl/acme.go` — ObtainCertificate/RenewCertificate 增加 logWriter 参数
- `backend/utils/ssl/dns_provider.go` — Cloudflare 认证逻辑修复 + 超时调整
- `backend/app/service/ssl.go` — Apply/Renew 传递 logWriter + DNS 阶段日志
- `frontend/src/views/website/ssl/index.vue` — Cloudflare 认证模式提示 + 字段友好名称

### 版本

`v0.5.28`

---

## 2026-03-25 — Session #46：SSL 证书体系完善 + 网站管理增强

### 完成内容

**SSL 证书体系**：
- [x] **证书路径统一**：`ssl.go` 和 `nginx_config.go` 的默认 SSLDir 统一使用 `global.CONF.Nginx.GetSSLDir()`（apt 模式为 `/etc/nginx/ssl`，prefix 模式为 `{installDir}/conf/ssl`），不再使用不可靠的 `os.Executable()` 路径
- [x] **证书自动续期**：新增 cron 定时任务（每天凌晨 2 点），自动检查 `autoRenew=true` 且即将 7 天内过期的证书，调用 ACME 续期
- [x] **续期后自动 reload nginx**：`Apply` 和 `Renew` 成功后自动调用 `reloadNginxGlobal()`，新证书即时生效，无需手动操作

**网站管理增强**：
- [x] **自定义 alias**：创建网站时可自定义标识名称（用于 nginx 配置文件名和目录名），留空自动从域名生成，后端检查唯一性
- [x] **Upstream 块支持**：托管模式下反向代理可配置 upstream 块（负载均衡），生成器在 server 块之前输出 upstream 定义
- [x] **前端 UI**：创建对话框新增 alias 输入；反向代理详情页新增 upstream textarea 编辑

### 关键决策

- SSLDir 默认值改为 `global.CONF.Nginx.GetSSLDir()`，保证 nginx 能直接读到证书文件，无路径不一致问题
- 自动续期采用 7 天过期阈值，cron `0 2 * * *` 每天执行一次，失败只打日志不影响其他证书
- `Apply` 也加了 reload — 新申请的证书如果已被网站引用，直接生效
- upstream 块是完全自由的文本输入，用户可写任意 upstream 指令，不做结构化校验

### 涉及文件

**后端**：
- `backend/app/service/ssl.go` — 证书路径统一、Renew/Apply 后 reload nginx、新增 `AutoRenewCerts()`/`reloadNginxGlobal()`
- `backend/app/service/nginx_config.go` — 证书路径统一、Generate 支持 upstream 块输出
- `backend/init/cron/cron.go` — 注册证书自动续期 cron 任务
- `backend/app/dto/website.go` — WebsiteCreate 新增 Alias、WebsiteUpdate/Detail 新增 Upstream
- `backend/app/model/website.go` — 新增 Upstream 字段
- `backend/app/service/website.go` — Create 支持自定义 alias + 唯一性检查、Update/Detail 支持 Upstream
- `backend/app/repo/website.go` — 新增 `WithByAlias`

**前端**：
- `frontend/src/views/website/website/index.vue` — 创建对话框新增 alias 输入
- `frontend/src/views/website/website/config.vue` — 反向代理详情新增 upstream 编辑
- `frontend/src/i18n/zh.ts` — alias/upstream/hint 翻译

---

## 2026-03-25 — Session #45：Nginx Bug 修复 + 版本检查更新功能

### 完成内容

**Bug 修复**：
- [x] **安装失败 UI 卡死**：error phase 时未重置 `installing` 状态，未弹错误提示 → 现在正确结束进度页并显示错误信息
- [x] **网站删除配置残留**：停用状态网站删除不清理 nginx 配置文件、运行中删除不清理 `sites-available` → 统一删除所有配置文件（enabled + available/conf.d + htpasswd）
- [x] **预编译安装后状态不同步**：`doInstall()` 结束后未调用 `DetectNginx()` → 安装完成后自动刷新检测
- [x] **卸载错误码不规范**：使用 `fmt.Errorf` 字符串拼接而非 `buserr` → 新增 `ErrNginxHasSites`/`ErrNginxAlreadyInstalled` 常量

**新功能 — 版本检查更新**：
- [x] **后端 `CheckUpdate()`**：apt 模式通过 `apt-cache policy nginx` 检查可用版本；预编译模式通过 GitHub Release 列表对比
- [x] **后端 `Upgrade()`**：apt 模式执行 `apt-get install --only-upgrade`；预编译模式先停止后覆盖安装
- [x] **前端版本卡片**：添加「检查更新」按钮，有更新时显示新版本标签 + 升级按钮
- [x] **进度复用**：升级过程复用安装进度轮询机制

### 关键决策

- apt 升级使用 `--only-upgrade` 而非 `dist-upgrade`，避免意外升级其他包
- apt 升级完成后自动 `systemctl reload nginx`
- 预编译升级先 `quit` 停止再覆盖安装（复用 `doInstall`）
- 网站删除时无论状态都清理所有配置文件，只有运行中才 reload nginx

### 涉及文件

**后端**：
- `backend/constant/errs.go` — 新增 `ErrNginxHasSites`、`ErrNginxAlreadyInstalled`
- `backend/app/dto/nginx.go` — 新增 `NginxUpdateInfo`、`NginxUpgradeReq`
- `backend/app/service/nginx_install.go` — 新增 `CheckUpdate()`/`Upgrade()`/`doUpgradeApt()`/`doUpgradePrecompiled()`/`checkAptUpdate()`，修复 `doInstall` 尾部、`Uninstall` 错误码
- `backend/app/service/website.go` — `Delete` 方法统一清理所有配置文件
- `backend/app/api/v1/nginx.go` — 新增 `CheckNginxUpdate`/`UpgradeNginx` handler
- `backend/router/router.go` — 新增更新检查和升级路由

**前端**：
- `frontend/src/api/modules/nginx.ts` — 新增 `checkNginxUpdate`/`upgradeNginx`
- `frontend/src/views/website/nginx/index.vue` — 安装错误处理修复 + 版本卡片更新 UI + 升级逻辑
- `frontend/src/i18n/zh.ts` — 新增版本更新相关翻译

---

## 2026-03-25 — Session #44：Nginx 安装默认改为 apt + 卸载安全检查

### 完成内容

- [x] **安装方式重构**：Nginx 安装默认使用 `apt-get install nginx`，保留预编译版本为备选
- [x] **后端 Install 方法**：`NginxInstallReq` 新增 `Method` 字段（`apt`/`precompiled`），默认 `apt`
- [x] **apt 安装流程**：`doInstallApt()` 执行 `apt-get update` + `apt-get install -y nginx` + `systemctl enable/start`
- [x] **卸载安全检查**：卸载前检查网站数量，有网站时拒绝卸载（需用户确认强制清理）
- [x] **强制卸载清理**：`forceCleanup=true` 时先清理所有网站 nginx 配置和数据库记录，再执行卸载
- [x] **前端卸载对话框**：有网站时显示醒目警告（含网站数量），需确认「强制卸载并清理」
- [x] **NginxStatus 增加 websiteCount**：前端获取状态时即可判断是否有网站存在
- [x] **前端安装对话框**：双模式 Radio 选择，apt 标记「推荐」，预编译才需选版本号
- [x] **i18n 新增**：安装方式、卸载警告、强制卸载等新文案

### 关键决策

- `apt` 安装完成后自动调用 `DetectNginx()` 刷新运行时状态
- 安装进度复用现有 `setProgress` + 前端轮询机制
- 卸载 apt nginx 执行 `apt-get remove`（非 purge），保留 `/etc/nginx` 配置文件
- 强制卸载清理逻辑：先遍历所有网站删除 nginx 配置文件（sites-enabled/sites-available/conf.d + htpasswd），再清空 DB 记录，最后执行卸载
- 网站数量在 NginxStatus 中返回，避免额外 API 请求

### 涉及文件

**后端**：
- `backend/app/dto/nginx.go` — `NginxInstallReq` 新增 `Method`，新增 `NginxUninstallReq`，`NginxStatus` 新增 `WebsiteCount`
- `backend/app/service/nginx_install.go` — 新增 `doInstallApt()`/`cleanupAllSites()`，重写 `Install()`/`Uninstall()`
- `backend/app/service/nginx.go` — `GetStatus` 返回网站数量
- `backend/app/api/v1/nginx.go` — `InstallNginx`/`UninstallNginx` handler 适配新参数
- `backend/app/repo/website.go` — 新增 `Count()` 方法

**前端**：
- `frontend/src/api/modules/nginx.ts` — `installNginx`/`uninstallNginx` 参数变更
- `frontend/src/api/interface/index.ts` — `NginxStatus` 新增 `websiteCount`
- `frontend/src/views/website/nginx/index.vue` — 安装对话框双模式 + 卸载安全检查
- `frontend/src/i18n/zh.ts` — 新增安装方式/卸载警告翻译

---

## 2026-03-25 — Session #43：容器功能增强 & Nginx 双模式重构

### 完成内容

- [x] **容器列表增强**：新增端口映射、资源使用率 (CPU%/内存)、IP 地址、运行时长列，对齐 1Panel 容器列表展示
- [x] **Docker 状态检测优化**：区分「未安装」和「已安装未启动」两种状态，分别给出不同引导
- [x] **Docker 一键安装**：新增安装按钮，使用 `get.docker.com` 官方脚本安装，支持安装日志实时查看
- [x] **容器资源统计**：通过 Docker Stats API (OneShot) 批量获取运行容器的 CPU/内存占用
- [x] **Nginx 双模式架构**：重构 `NginxConfig`，自动检测并支持两种模式：
  - 系统包模式 (apt)：`/etc/nginx/`、`sites-available`/`sites-enabled`、`systemctl`
  - 自包含模式 (prefix)：`/opt/xpanel/nginx/`、`nginx -p`、自定义 systemd 服务
- [x] **网站自动启用**：创建网站后自动生成配置并启用，无需手动启用
- [x] **源码模式配置修复**：正确读取磁盘上的实际配置文件（系统模式优先 sites-available）
- [x] **日志路径修复**：`nginx_log.go` 使用正确的日志路径 (`GetLogDir()/sites/`)
- [x] **i18n 补全**：容器模块 Docker 安装相关、Nginx 运行模式等新文案

### 关键决策

- Nginx 检测策略：启动时 `DetectNginx()` 优先检查自包含安装目录，若不存在则检测系统 `nginx` 二进制
- 系统模式下使用 `systemctl start/stop/reload nginx`，不加 `-p` 参数
- 网站启用：系统模式写入 `sites-available/{alias}.conf` + symlink 到 `sites-enabled/`
- 网站禁用：仅删除 `sites-enabled/` symlink，保留 `sites-available/` 文件
- Docker 安装使用官方脚本 `curl -fsSL https://get.docker.com | bash`，异步执行并轮询日志

### 涉及文件

**后端**：
- `backend/global/global.go` — NginxConfig 重构，添加 DetectNginx/IsSystemMode/各路径方法
- `backend/app/dto/container.go` — ContainerInfo 新增端口/资源/IP 字段，DockerStatusResp
- `backend/app/dto/nginx.go` — NginxStatus 新增 SystemMode 字段
- `backend/app/service/container.go` — 列表填充端口/IP/资源，DockerStatus 区分安装/运行
- `backend/app/service/docker_install.go` — 新增 Docker 安装服务
- `backend/app/service/nginx.go` — 双模式操作（systemctl vs nginx -p）
- `backend/app/service/website.go` — applyConfig 双模式、自动启用、源码模式修复
- `backend/app/service/nginx_config.go` — EnsureNginxInclude 双模式
- `backend/app/service/nginx_log.go` — 日志路径修复
- `backend/utils/docker/client.go` — 新增 IsDockerInstalled/GetDockerVersion
- `backend/app/api/v1/container.go` — 新增 InstallDocker/GetDockerInstallLog
- `backend/router/router.go` — 新增 Docker 安装路由
- `backend/server/server.go` — 启动时调用 DetectNginx

**前端**：
- `frontend/src/views/container/index.vue` — 全面重写，新增资源列/端口列/安装引导
- `frontend/src/api/modules/container.ts` — 新增安装相关 API
- `frontend/src/api/interface/index.ts` — Container/DockerStatus/NginxStatus 类型更新
- `frontend/src/views/website/nginx/index.vue` — 显示 Nginx 运行模式
- `frontend/src/i18n/zh.ts` — 新增容器/Nginx 模式相关翻译

### 遗留问题

- 容器创建表单尚未添加端口/卷映射字段（后端已支持，前端未暴露）
- Compose 功能在前端仍未接入标签页
- 系统模式下的 Nginx 安装按钮应隐藏或改为提示

### 下一步计划

- 完善容器创建表单（端口映射、卷映射、CPU/内存限制）
- 接入 Compose 标签页
- 考虑容器终端 WebSocket 功能
- 容器批量操作

---

## 2026-03-25 — Session #42：Xray 模块全面优化

### 完成内容

- [x] **流量实时化**：SyncTraffic cron 从 5 分钟缩短到 1 分钟；前端用户表格 30 秒自动轮询（可开关）
- [x] **流量限额功能**：Model/DTO/Service 全链路新增 TrafficLimit 字段；SyncTraffic 后自动检查超限用户并禁用
- [x] **修复 SS 节点编辑丢数据**：openNodeDrawer 回填 ssMethod/ssPassword/fallbacks/sniffMetadataOnly；XrayNode 类型补全
- [x] **错误处理完善**：SyncTraffic 日志从 Debug 改 Warn；toggleNode/generateRealityKeys 加错误提示；流量图表加 loading；停止/重启服务加二次确认
- [x] **UI/UX 优化**：用户表格加备注列、空状态、分页 sizes；流量列有限额时显示进度条；节点卡片显示 remark；流量图表 loading
- [x] **i18n 补全**：约 50+ 处硬编码中文改为 t() 调用，覆盖 SS 加密、监听地址、传输设置标题、TLS/Reality、日志级别、出站协议等
- [x] 修复 Nginx 反代配置生成按钮 SyntaxError（vue-i18n `{}` 转义问题）
- [x] 清理死代码：移除未使用的 getXrayShareLink 前端导入
- [x] onUnmounted 清理所有定时器（轮询/安装日志/图表）

### 关键决策

- 流量限额单位：后端字节，前端以 GB 输入并换算
- 自动刷新默认开启，30 秒间隔，用户可通过开关关闭
- SyncTraffic 后自动检查超限，无需额外 cron 任务
- Nginx 配置生成器中的中文注释保留不走 i18n（属于配置文件注释）

---

## 2026-03-24 — Session #41：6 项 UI/UX 修复 + SSH 管理重构

### 完成内容

- [x] **主题色选择器样式修复**
  - 根因：scoped 样式 + `:global()` 嵌套在 Teleport 场景下子选择器失效
  - 改为独立非 scoped `<style>` 块，网格改为 4 列（8 个预设均匀排列）
  - 色块尺寸从 26px 调整为 28px，间距优化

- [x] **终端字号持久化**
  - 字号保存到 `localStorage`，刷新页面不再重置为 14

- [x] **登录日志时间人性化**
  - 添加 `formatTime` 函数（今天/昨天/X月X日 + 时分秒）
  - 状态标签显示"成功/失败"而非原始英文

- [x] **文件管理 tab + 按钮位置**
  - 覆盖 Element Plus `el-tabs__header` 默认 `space-between` 为 `flex-start`
  - `nav-wrap` 改为 `flex: 0 1 auto`，+ 按钮紧跟最后一个标签

- [x] **SSH 管理全面修复**
  - 自启检测：支持 `enabled`/`static`/`indirect`/`alias` 多种状态
  - 服务名检测：Debian 优先 `ssh`，RHEL 优先 `sshd`
  - 配置修改后自动 `systemctl restart` 重载
  - sshd_config 编辑器保存后同样自动重载
  - 开关组件改用 `:model-value` + `@change` 事件驱动，消除只读 computed 的脆弱写法
  - 新增公钥管理功能：列表、添加、删除 authorized_keys

- [x] **容器 Docker 未安装友好引导**
  - 进入页面先调 `getDockerStatus` 检测
  - 未安装时显示 `el-empty` 引导页 + 重新检测按钮

### 关键决策

- 主题色选择器用独立非 scoped style 彻底解决 Teleport 样式问题
- SSH 配置修改后用 restart 而非 reload，确保端口等变更立即生效
- 公钥管理用 base64 前 16 字符作为指纹标识（简单可靠）

---

## 2026-03-24 — Session #40：安全修复与稳定性改进

### 完成内容

- [x] **MySQL Restore 命令注入修复（高危）**
  - 移除 `bash -c` + `fmt.Sprintf` 拼接命令的危险实现
  - 改为 `exec.Command("mysql", ...)` + `os.Open` 文件句柄传 stdin
  - 密码通过 `MYSQL_PWD` 环境变量传递，不暴露到命令行参数
  - 与 PostgreSQL Restore 实现模式对齐

- [x] **Xray 安装竞态修复**
  - 后端新增 `IsInstallRunning()` 方法，直接暴露 `installRunning` 状态
  - API `GetInstallLog` 改用服务状态判断 `running`，替代日志内容推断
  - 修复：安装初期日志为空时前端误判安装已结束的竞态问题
  - 修复：升级场景下日志结束标记不匹配的问题
  - 前端安装失败时增加 `ElMessage.error` 提示

- [x] **虚拟化检测语义优化**
  - 首页虚拟化字段：空值时显示 `-` 而非「物理机」
  - 避免 VPS 漏检时误导用户以为是物理机

### 关键决策

- MySQL Restore 使用 `MYSQL_PWD` 环境变量而非 `-p` 参数传密码，避免进程列表泄露
- Xray 安装状态用后端布尔值而非日志内容推断，彻底消除竞态

---

## 2026-03-24 — Session #39：前端 `any` 类型彻底清零

### 完成内容

- [x] `api/interface/index.ts`：新增 `SSHInfo`、`SSHLogEntry`、`DiskDetail`、`PartitionInfo`、`RemoteMountInfo` 5 个接口，与后端 DTO 对齐
- [x] `views/host/process/index.vue`：`processes` → `ProcessInfo[]`、`connections` → `NetworkConn[]`、`handleKill(row)` → `ProcessInfo`、`statusType` 返回值改为 Element Plus 联合类型
- [x] `views/website/nginx/index.vue`：`availableVersions` → `NginxVersion[]`、`confFiles` → `ConfFile[]`、`handleAutoStart(val)` → `boolean`
- [x] `views/host/disk/index.vue`：`partitions` → `DiskDetail[]`、`remoteMounts` → `RemoteMountInfo[]`
- [x] `views/host/ssh/index.vue`：`sshInfo` → `SSHInfo`、`sshLogs` → `SSHLogEntry[]`
- [x] `views/terminal/index.vue`：模板 ref 回调 `el: any` → `el: unknown`
- [x] `views/website/website/config.vue`：`handleModeSwitch(val)` → `string | number | boolean`
- [x] `views/host/monitor/index.vue`：`stats` → `SystemStats`

### 结果

- **`.vue` 文件 `any` 出现次数：14 → 0**
- **`.ts` 文件 `any` 出现次数：0（Session #38 已清零）**
- **前端 `any` 总数：0，问题彻底解决**
- Lint 检查通过，无新增错误

---

## 2026-03-24 — Session #38：前端 API 模块去除 `any`

### 完成内容

- [x] `api/modules`：`container`、`node`、`database`、`backup`、`cronjob`、`ssl` 请求体改用 `../interface` 或显式对象类型，移除 `any`
- [x] `api/interface/index.ts`：新增 `CronjobCreateForm`、`CronjobUpdateForm`（与后端 `dto.CronjobCreate` / `CronjobUpdate` 对齐）；`createCronjob`/`updateCronjob` 不用含 `status` 的 `Cronjob`，避免 `vue-tsc` 与表单不一致

---

## 2026-03-24 — Session #37：多项 UI/UX 优化与功能增强

### 完成内容

- [x] **虚拟化检测增强**
  - 后端 `monitor.go` 新增 `detectVirtualization()` 函数
  - gopsutil 结果为空时回退：`systemd-detect-virt` → DMI 产品名 → `/proc/cpuinfo` hypervisor 标记
  - 解决了部分 VPS 环境虚拟化类型显示为空的问题

- [x] **Xray 安装脚本路径动态化**
  - `xray.go` 新增 `getXrayInstallScript()` 动态获取安装脚本路径
  - 基于可执行文件位置向上搜索 `xray-install.sh`，不再硬编码 `/data/X-Panel/`
  - 解决了在非 `/data/X-Panel` 路径部署时安装脚本找不到的问题

- [x] **主题色选择器样式修复**
  - 修正 Header.vue popover 宽度和网格布局（6列代替8列）
  - 修复色块和自定义颜色区域的间距和层级问题

- [x] **流量统计 i18n 缺失修复**
  - 添加 `traffic.addConfig: '添加监控'` 翻译 key
  - 按钮不再显示 "traffic.addConfig" 原始 key

- [x] **操作日志优化**
  - 后端：`OperationLog` model 新增 `Latency` 字段，中间件独立记录格式化耗时
  - 前端：增加人性化操作描述（API 路径映射为中文描述）
  - 时间格式优化：今天/昨天/月日 + 时分秒格式
  - 状态显示：Success/Failed → 成功/失败

- [x] **磁盘管理远程挂载功能**
  - 后端：`disk.go` 新增 `MountRemote`/`UnmountRemote`/`ListRemoteMounts` 方法
  - 支持 NFS 和 SMB/CIFS 协议
  - 前端：磁盘管理页增加远程挂载列表 + 挂载对话框（协议/服务器/路径/认证/选项）
  - 新增 3 条 API 路由

- [x] **终端美化**
  - 添加字体大小调节控制（+/- 按钮，10-24px 范围）
  - 字体列表增加 Cascadia Code
  - 硬编码中文迁入 i18n（连接断开/连接错误/批量发送提示等）

- [x] **Xray 页面 i18n 补全**
  - 补充 11 个缺失 key：ssMethod/ssPassword/clientEncryption 等

### 关键决策

- 虚拟化检测采用多级回退策略，而非仅依赖 gopsutil
- 远程挂载功能直接调用系统 mount/umount 命令
- 操作日志描述采用前端映射而非后端翻译，保持后端无状态

### 遗留问题

- SSL 管理页面仍有大量硬编码中文
- MySQL Restore 存在命令注入风险需重构
- ~~前端大量 `any` 类型需要逐步替换为接口类型~~（Session #38 + #39 已彻底清零）
- 英文翻译文件 `en.ts` 尚未创建

---

## 2026-03-21 — Session #36：Xray 权限/日志/更新/出站代理/UI 全面升级

### 完成内容

- [x] **权限修复（nobody 兼容）**
  - `FixPermissions()` 方法：`MkdirAll` 创建 `/data/xray/log` 和 `/data/xray/etc`，然后 `chown -R nobody:nogroup`（Debian），失败时自动回退 `nobody:nobody`（RHEL）
  - 安装/升级完成后自动调用权限修复
  - 写入 `config.json` 后自动 `chmod 640`
  - 前端状态栏新增"修复权限"按钮（带 tooltip 说明）

- [x] **日志配置（可视化管理）**
  - 日志设置存入 `settings` 表（`XrayLogLevel` / `XrayAccessLog` / `XrayErrorLog`）
  - Migration 添加默认值（warning / /data/xray/log/access.log / /data/xray/log/error.log）
  - `GetLogSettings()` / `UpdateLogSettings()` 接口；修改后自动 reload config
  - `buildXrayConfig` 从 DB 读取日志设置，支持 `none` 和空值禁用日志
  - 前端"设置"抽屉 → 日志 tab：级别下拉、路径输入、logrotate 建议文案

- [x] **版本更新与升级**
  - `CheckUpdate()`：调 GitHub API `repos/XTLS/Xray-core/releases/latest`，比较版本号
  - `DoUpgrade()`：复用安装脚本，完成后自动修权 + reload
  - 前端"设置"抽屉 → 版本/更新 tab：显示当前/最新版本，"检查更新"按钮，有更新时出现"立即升级"按钮，升级时展示日志

- [x] **出站代理（全套实现）**
  - 新增 `XrayOutbound` 模型（name/tag/protocol/settings JSON/enabled/remark）
  - `OutboundTag` 字段加入 `XrayNode`（空 = direct 直连）
  - `buildXrayConfig` 中：加载所有启用的出站代理，按 `node.OutboundTag` 生成路由规则
  - 全套 CRUD API：`/xray/outbounds` GET/POST/POST/update/del
  - 前端"设置"抽屉 → 出站代理 tab：表格管理，编辑对话框（协议选择 + JSON settings 模板自动填充）
  - 节点编辑"高级设置"tab 新增出站路由下拉（可选 direct/blocked/自定义出站）
  - 更换协议时自动填充对应 settings JSON 模板

- [x] **UI 修复**
  - 状态栏标签加 `white-space: nowrap` 防换行，version-text/config-path 加 `max-width + text-overflow: ellipsis`
  - 新增"设置"按钮（齿轮图标），打开全局设置抽屉（日志/版本更新/出站代理三个 tab）

### 版本
- `v0.5.9` 已推送 GitHub，CI 自动构建中


### 完成内容

- [x] **后端：Xray 服务控制接口**
  - 新增 `POST /xray/service/control`，支持 `start/stop/restart/enable/disable`
  - `GetStatus()` 新增 `enabledOnBoot` 字段（`systemctl is-enabled xray`）
  - `ControlService()` 统一调用 `systemctl <action> xray`

- [x] **前端：状态栏服务控制按钮**
  - 启动 / 重启 / 停止 三按钮组（`el-button-group`），依状态自动 `disabled`
  - 开机自启切换按钮，状态来自 `enabledOnBoot`；操作后自动刷新状态

- [x] **前端：修复 nginx 配置对话框 SyntaxError**
  - 根本原因：`v-model="generatedNginxConfig"` 绑定到只读 `computed` ref，Vue 内部尝试写入触发解析错误
  - 修复：改为 `:model-value="generatedNginxConfig"`（单向绑定）
  - 同时移除 gRPC 模板注释中的 `⚠` emoji（避免部分解析器异常）

- [x] **前端：分享链接 TLS 覆盖选项**
  - 当节点 `security=none`（nginx 反代场景）时，对话框展示"客户端加密"区块
  - 支持选择：无加密（直连）/ TLS（via nginx）
  - TLS 模式下可配置：SNI、ALPN（h2/http1.1 多选）、uTLS 指纹
  - `buildShareLinkClient` 重构为接收 `override` 对象，覆盖 security/sni/alpn/fp
  - 默认值：打开分享链接时自动预填 security=tls、alpn=[h2, http/1.1]、fp=chrome

- [x] **package.json**：固化 `NODE_OPTIONS=--max-old-space-size=3072`

### 版本
- `v0.5.8` 已推送 GitHub，CI 自动构建中


### 完成内容

- [x] **彻底重写 Xray 节点模型**（`model/xray.go`）
  - 移除旧的扁平化字段（domain/tlsCert/path/serviceName 等硬编码字段）
  - 新增 `ListenAddr`（监听地址：0.0.0.0 或 127.0.0.1，适合 nginx 反代场景）
  - 新增 `NetworkSettings`（JSON 存储传输方式专属参数）
  - 新增 `SecuritySettings`（JSON 存储 TLS/Reality 专属参数）
  - 新增节点级 `Flow` 字段，用户可独立覆盖
  - 新增 `SniffEnabled` / `SniffDestOverride` 流量探测设置
  - 用户模型新增独立 `Flow` 字段（可覆盖节点默认 flow）

- [x] **完整 DTO 重设计**（`dto/xray.go`）
  - RAW(TCP)：`headerType`、`acceptProxyProtocol`
  - WebSocket：`path`、`host`、`acceptProxyProtocol`
  - gRPC：`serviceName`、`multiMode`、`idleTimeout`、`healthCheckTimeout`、`permitWithoutStream`
  - XHTTP：`path`、`host`、`mode`（auto/packet-up/stream-up/stream-one）、`xPaddingBytes`、`scStreamUpServerSecs`
  - HTTPUpgrade：`path`、`host`、`acceptProxyProtocol`
  - TLS：`serverName`、`certFile`、`keyFile`、`alpn`、`fingerprint`（uTLS）、`minVersion`、`rejectUnknownSni`
  - Reality：`privateKey`、`publicKey`、`shortIds[]`、`serverNames[]`、`dest`、`fingerprint`、`spiderX`、`xver`

- [x] **Service 层完整重写**（`service/xray.go`）
  - 支持所有传输方式的配置反序列化映射到 Xray JSON
  - TLS 证书使用文件路径（`certificateFile`/`keyFile`）
  - Reality 完整参数映射，`network: "raw"` 正确使用 `rawSettings` 键
  - 用户 flow 优先于节点默认 flow

- [x] **前端界面全面重设计**（`views/xray/index.vue`）
  - 节点使用 Drawer 抽屉（640px）替代对话框
  - 表单 4 个 Tab：基础设置 / 传输协议 / 安全加密 / 高级设置
  - 动态子表单：切换传输/安全类型后显示对应参数
  - Reality: serverNames/shortIds 支持 Tag 形式增删
  - VLESS Flow 含组合警告提示

- [x] **TypeScript 类型全面更新**（`api/modules/xray.ts`）

### 关键架构决策

- `NetworkSettings`/`SecuritySettings` 以 JSON 字符串存入 SQLite，避免频繁 Schema 变更
- `127.0.0.1` 监听 + `acceptProxyProtocol` 支持 nginx 透传场景
- 分享链接连接地址完全依赖节点配置，不做外网 IP 探测

---

## 2026-03-21 — Session #33：Xray 功能完善

### 完成内容

- [x] **Xray 安装引导**：进入页面时检测 `/data/xray/bin/xray` 是否存在，未安装则显示引导卡片；点击「一键安装」后台执行 `xray-install.sh install`，前端每 2 秒轮询日志流展示实时进度，安装成功后自动刷新状态
- [x] **修复 getServerIP()**：移除 `curl ifconfig.me` 外网依赖，改用 `ip route get 1.1.1.1` 从本机路由表获取主出口 IP，无网络依赖；备用方案为 `hostname -I`
- [x] **SyncTraffic 并发安全**：新增独立 `syncMu sync.Mutex`，`SyncTraffic` 使用 `TryLock` 防止 cron 积压导致并发执行
- [x] **节点快速启用/禁用**：节点列表每项新增 el-switch 开关，调用 `POST /xray/nodes/toggle`，无需打开编辑对话框即可切换；禁用节点在列表中半透明显示
- [x] **流量历史图表**：
  - 新增 `XrayTrafficDaily` 数据库模型（user_id + date 联合唯一索引）
  - cron 每日 00:01 调用 `SnapshotDailyTraffic()` 快照当前累计值
  - `GetTrafficHistory()` 将累计值转换为每日增量
  - 前端点击流量单元格弹出 ECharts 折线图（30 天上行/下行）
- [x] **XrayStatus 新增 installed 字段**：前端根据此字段区分「未安装」和「已安装未运行」两种状态
- [x] **新增 ToggleNode API**：`POST /xray/nodes/toggle`
- [x] **新增安装相关 API**：`POST /xray/install`、`GET /xray/install/log`
- [x] **新增流量历史 API**：`POST /xray/users/traffic-history`

### 版本
- 待发布

---



### 完成内容

- [x] **后端 Model**：`XrayNode`（节点/入站配置）、`XrayUser`（代理用户，含 UUID/Email/到期时间/流量统计）
- [x] **后端 DTO**：节点和用户的创建/更新/搜索/响应 DTO，含 `XrayStatusResponse`、`XrayGenerateKeyResponse` 等
- [x] **后端 Repo**：`IXrayNodeRepo` + `IXrayUserRepo`，含 `WithXrayNodeID` DBOption
- [x] **后端 Service**：`IXrayService` 完整实现
  - 节点 CRUD + Xray config.json 动态生成（VLESS/VMess/Trojan × TCP/WS/gRPC × none/TLS/Reality）
  - 用户 CRUD，UUID 自动生成，Email 用于流量统计 key
  - Stats API 流量同步（`xray api statsquery --reset true` → 原子累加到 SQLite）
  - 到期用户自动禁用并 reload Xray
  - Reality 密钥对生成（`xray x25519`）
  - 分享链接生成（VLESS/VMess/Trojan URI 格式）
- [x] **后端 API**：`XrayAPI` 12 个接口，注册进 `entry.go` 和 `router.go`
- [x] **DB 迁移**：`XrayNode`、`XrayUser` 表自动迁移
- [x] **Cron 任务**：每 5 分钟同步流量，每小时检查过期用户
- [x] **前端 API 模块**：`api/modules/xray.ts`，TS 接口定义 + 所有 API 调用封装
- [x] **前端页面**：`views/xray/index.vue` 左右布局
  - 左侧节点列表（协议/安全类型标签、端口、用户数统计）
  - 右侧用户表格（UUID、到期时间、上下行流量、状态）
  - 节点对话框（Reality 密钥自动生成、TLS 证书配置）
  - 用户对话框（到期时间、启用/禁用）
  - 分享链接复制弹窗
- [x] **路由注册**：`routers/modules/xray.ts` + 注入 `routers/index.ts`
- [x] **侧边栏菜单**：在「流量统计」下方添加「Xray 代理」菜单项
- [x] **i18n**：zh.ts 新增 `menu.xray` + 完整 `xray` 命名空间（50+ 条文本）

### 关键决策

- Xray 安装路径：`/data/xray/`，配置文件 `/data/xray/etc/config.json`
- 流量统计方案：使用 `xray api statsquery --reset true` CLI，避免引入 gRPC 依赖，每次调用返回增量并清零，累加存 DB
- 配置生成策略：以 SQLite 为单一数据源，每次 CRUD 后重新生成完整 config.json 并 `systemctl reload xray`
- Stats API 端口：固定 `127.0.0.1:10085`，作为 `dokodemo-door` inbound 注入每次生成的配置中

### 下一步计划

- 用户流量历史图表（按日/周统计折线图）
- 订阅链接（Base64 编码的多节点合并链接）
- 节点二维码生成
- 限速功能（通过 Xray Policy Level 实现）

---

## 2026-03-19 — Session #30：主题色自定义系统 + 全局视觉增强

### 完成内容

- [x] **主题色系统**：8 种预设色板（青蓝/靛蓝/翡翠/琥珀/玫红/天蓝/紫罗兰/橙色）+ 自定义拾色器
- [x] **动态 CSS 注入**：accent 颜色实时修改 `--xp-accent` 及 Element Plus `--el-color-primary` 等变量，无需刷新
- [x] **Header 色彩选择器**：Popover 面板内含预设色块网格 + HTML5 颜色输入
- [x] **设置页外观区块**：新增「外观设置」卡片，深浅模式切换 + 主题色预设 + 自定义
- [x] **硬编码颜色清理**：全面替换 `rgba(34, 211, 238, ...)` 为 `var(--xp-accent-muted)` 等动态变量
- [x] **组件视觉增强**：Card 悬停阴影、Dialog/Drawer 圆角和阴影、Dropdown 圆角、侧边栏装饰线、Header 模糊增强
- [x] **文件图标适配**：SVG 默认文件夹颜色跟随主题色变化
- [x] **ECharts/进度条适配**：动态读取 CSS 变量而非硬编码颜色值
- [x] **Pinia 持久化**：`accentKey` / `accentCustom` 保存在 localStorage

### 版本
- 发布 `v0.5.2`

---

## 2026-03-19 — Session #31：深度 UI 优化

### 完成内容

- [x] **全局组件重写** (_components.scss)：
  - 卡片：悬停升浮 + 渐变阴影 + 内发光边缘
  - 弹窗/抽屉：大圆角 + 深阴影 + 关闭按钮旋转动效 + header/footer 边框
  - 按钮：Default 悬停变色、Primary 发光阴影、link 按钮悬停底色
  - 输入框：聚焦微透明背景、textarea 焦点外发光
  - 下拉菜单/选择器：项目圆角 + 内边距 + 选中项背景高亮
  - 表格：行悬停 accent 高亮、圆角溢出隐藏
  - 分页/日期选择器/标签/加载遮罩等全面增强
  - 遮罩层增加 backdrop-filter 模糊

- [x] **侧边栏重构** (Sidebar.vue)：
  - Logo 图标改为渐变色背景 (accent→secondary)
  - 菜单项悬停图标微放大 scale(1.1)
  - 子菜单项悬停右移 padding 视觉反馈
  - 展开子菜单增加左侧连接线
  - 活跃子菜单箭头变色

- [x] **首页增强** (home/index.vue)：
  - 资源卡片悬停升起 translateY(-2px) + 阴影扩散
  - 进度条优化：6px 高度 + 3px 圆角 + 0.8s 缓动
  - 快捷入口悬停：升起 3px + 图标放大 + 标签变色 + 阴影
  - 磁盘卡片悬停统一 accent 边框

- [x] **终端页增强** (terminal/index.vue)：
  - 标签栏顶部 padding + 圆角标签
  - 终端容器内阴影增强深度感
  - 命令面板遮罩增加 blur

- [x] **计划任务增强** (cronjob/index.vue)：
  - Cron 预览区块增加背景色和边框

- [x] **全局工具类** (_utilities.scss)：
  - 工具栏增加背景容器化（背景+边框+圆角）
  - 右键菜单增加 backdrop-filter
  - 新增 `.status-dot` / `.hover-reveal` 等工具类

- [x] **全局基础** (index.scss)：
  - 页面切换增加 translateY 入场动效
  - 新增 slide-fade 过渡动画
  - focus-visible 聚焦轮廓环
  - 链接默认样式

### 版本
- 发布 `v0.5.3`

---

## 2026-03-19 — Session #28：流量统计功能

### 完成内容

- [x] **后端三层架构**：Model (`TrafficConfig` / `TrafficHourly` / `TrafficSnapshot`) → Repo → Service → API
- [x] **流量采集器**：基于 cron 每 5 分钟采样系统网卡计数器，计算增量写入 SQLite 小时记录，支持重启后计数器归零检测
- [x] **计费周期计算**：根据用户配置的 `ResetDay`（每月重置日 1-28）动态计算当前计费周期起止时间
- [x] **6 个 API 接口**：配置 CRUD、网卡列表、按时间范围查询流量统计、当前周期汇总
- [x] **前端顶级菜单页面**：概览卡片（环形进度条 + 用量详情）、ECharts 柱状图（上行/下行分色堆叠）、明细表格
- [x] **配置弹窗**：选择网卡、设置月配额（GB/TB 单位切换）、重置日
- [x] **数据清理**：cron 每月自动清理 12 个月前的旧记录
- [x] **i18n 支持**：完整中文翻译

### 关键决策
- 采用小时粒度存储（每月约 720 条/网卡），兼顾查询灵活性和存储效率
- 使用 gopsutil 读取 `/proc/net/dev` 计数器，通过快照差值法计算增量
- 计数器回退检测：当前值 < 上次值时视为重启，增量 = 当前值

### 涉及文件
- 后端新增：`model/traffic.go` `dto/traffic.go` `repo/traffic.go` `service/traffic.go` `api/v1/traffic.go`
- 后端修改：`api/v1/entry.go` `router/router.go` `init/migration/migration.go` `init/cron/cron.go`
- 前端新增：`views/traffic/index.vue` `views/traffic/config-dialog.vue` `api/modules/traffic.ts` `routers/modules/traffic.ts`
- 前端修改：`routers/index.ts` `layout/components/Sidebar.vue` `i18n/zh.ts`

---

## 2026-03-13 — Session #27：六大功能模块全量实现

### 完成内容

#### Phase 1：登录防暴力破解
- [x] `utils/captcha/captcha.go`：基于 base64Captcha 生成图片验证码
- [x] `init/auth/ip_tracker.go`：内存 IP 失败计数器，3 次阈值 + 30 分钟过期
- [x] `global.IPTracker` 全局实例
- [x] Auth API 增加验证码校验逻辑 + `GET /auth/captcha` 接口
- [x] 前端登录页动态显示验证码输入框

#### Phase 2：计划任务管理
- [x] `robfig/cron/v3` 集成，`global.CRON` 全局调度器
- [x] Cronjob + CronjobRecord 模型，标准四层 CRUD
- [x] 支持 shell / curl / website / database / directory 五种任务类型
- [x] 手动触发 (HandleOnce)、启停状态切换、执行记录查看
- [x] 前端 views/cronjob 完整管理界面

#### Phase 3：数据库管理
- [x] MySQL (`go-sql-driver/mysql`) + PostgreSQL (`lib/pq`) 驱动
- [x] DatabaseServer + DatabaseInstance 模型
- [x] utils/database/ 封装连接、CRUD、备份恢复 (mysqldump/pg_dump)
- [x] 同步远程数据库列表功能
- [x] 前端 views/database 服务器管理 + 库管理界面

#### Phase 4：容器管理
- [x] Docker SDK (`docker/docker`) 集成，支持容器/镜像/网络/卷完整 CRUD
- [x] Compose 管理（基于 docker compose CLI）
- [x] 容器启停重启、日志查看、镜像拉取删除
- [x] 前端 views/container Tab 式管理界面

#### Phase 5：备份系统
- [x] BackupAccount + BackupRecord 模型
- [x] utils/cloud_storage/ 统一接口：Local / S3 / SFTP / WebDAV 四种后端
- [x] 网站备份(tar.gz) / 数据库备份(mysqldump/pg_dump) / 目录备份(tar.gz)
- [x] 异步备份任务，自动写入备份记录
- [x] 前端 views/backup 账户管理 + 备份创建 + 记录查看

#### Phase 6：面板集群 (Agent 模式)
- [x] Node 模型 + CRUD + 连接测试
- [x] `middleware/node_proxy.go`：根据 X-Node-ID header 转发请求到 Agent
- [x] `middleware/agent_token.go`：Agent 端 Token 认证
- [x] 60 秒心跳定时检测节点在线状态
- [x] 前端全局 store 增加 currentNodeID，HTTP 拦截器自动附加
- [x] Header 节点切换下拉框 + views/node 节点管理页

### 关键决策
- 验证码阈值 3 次（而非 1Panel 的 1 次），平衡安全性和体验
- 容器管理不依赖 Model，数据直接来自 Docker API
- 备份系统通过统一 CloudStorageClient 接口解耦存储后端
- 集群通过 HTTP API Proxy 实现，Agent 复用完整 X-Panel 实例

### 新增依赖
- `github.com/mojocn/base64Captcha` (验证码)
- `github.com/robfig/cron/v3` (定时任务)
- `github.com/go-sql-driver/mysql` + `github.com/lib/pq` (数据库)
- `github.com/docker/docker` + `github.com/docker/go-connections` (容器)
- `github.com/aws/aws-sdk-go-v2` (S3) + `github.com/pkg/sftp` (SFTP) + `github.com/studio-b12/gowebdav` (WebDAV)

### 下一步计划
- 前端构建验证 + 集成测试
- 计划任务与备份系统联动（cronjob type=website/database/directory 触发备份）
- 容器终端 (docker exec WebSocket)
- 节点监控数据聚合展示

---

## 2026-03-09 — Session #26：终端 vim 修复 + 首页美化运维按钮 + SSH 配置编辑

### 完成内容

#### Nginx 默认自启
- 安装完成后自动执行 `systemctl enable xpanel-nginx`

#### 终端 vim 严重 bug 修复（根因：WebSocket 编码）
- **根因分析**：后端 PTY 输出作为 WebSocket TextMessage 发送，当 4096 字节缓冲区在多字节 UTF-8 字符中间截断时，浏览器因 UTF-8 校验失败丢弃/损坏数据
- **对比 1Panel**：1Panel 使用 JSON + Base64 编码绕开此问题
- **X-Panel 修复**：后端改用 `BinaryMessage` 发送 PTY/SSH 输出，前端设置 `ws.binaryType = 'arraybuffer'` 并用 `Uint8Array` 写入 xterm.js
- 本地终端和 SSH 终端两处同步修复
- 文件管理终端 (terminal-dialog.vue) 同步修复

#### 首页运维按钮
- 新增 3 个 API：重启服务器(`/settings/reboot`)、关机(`/settings/shutdown`)、重启面板(`/settings/restart-panel`)
- 首页顶部增加「重启面板」「重启服务器」按钮组，均有二次确认对话框

#### SSH 管理增加 sshd_config 编辑
- 新增后端 API：`GET/POST /ssh/sshd-config`
- 保存前自动执行 `sshd -t` 测试，不通过自动回滚
- 前端 SSH 管理新增「配置文件」tab，Monaco Editor 直接编辑

#### 首页布局美化
- 系统信息和网络信息合并为左右双列布局
- 运维按钮集成到 header 右侧
- 网络信息改为纵向列表（之前是网格，信息挤在一起）

### 关键决策
- 终端数据传输使用 BinaryMessage 而非 1Panel 的 JSON+Base64（更高效，零开销）
- 重启/关机使用 500ms 延迟确保 HTTP 响应先返回

---

## 2026-03-09 — Session #25：网站管理全面优化（HTTP/2 + 自启 + 双模式配置 + 日志分析）

### 完成内容

#### HTTP/2 开关
- `model.Website` 新增 `Http2Enable` 字段（默认 true）
- 配置生成器 `writeSSLBlock()` 增加 `http2 on;` 指令（Nginx 1.25.1+ 语法）
- 前端 HTTPS 设置 tab 增加 HTTP/2 开关

#### Nginx 开机自启
- 安装时自动创建 systemd service 文件 `/etc/systemd/system/xpanel-nginx.service`
- 新增 `SetAutoStart(enable)` / `isAutoStartEnabled()` 方法
- 前端 Nginx 管理页增加开机自启开关
- 新增 API: `POST /nginx/autostart`

#### 双模式配置管理（托管/源码）
- `model.Website` 新增 `ConfigMode` 字段（`managed`/`source`）
- **托管模式**：保持原有 DB→生成→覆写流程
- **源码模式**：Monaco Editor 直接编辑 conf 文件，保存前 `nginx -t` 验证，失败自动回滚
- 模式切换：托管→源码 (加载现有配置)，源码→托管 (警告后从 DB 重新生成)
- 新增 API: `POST /websites/conf-content`, `/conf-content/save`, `/config-mode`

#### 日志分析
- 新增 `nginx_log.go` 服务：解析 Nginx combined 格式 access log
- 支持按时间范围过滤（今日/7天/30天）
- 聚合统计：总请求、UV、流量、错误率、状态码分布、Top URL/IP/UA
- 时间序列：按小时/天的请求和流量趋势
- 前端新增「日志分析」tab：概览卡片 + ECharts 图表 + 排行表格
- 新增 API: `POST /websites/log-analysis`

### 关键决策
- HTTP/2 使用 `http2 on;` 而非 `listen 443 ssl http2;`，兼容 Nginx 1.25.1+
- 双模式配置解决了"手动修改被覆盖"的核心痛点，源码模式下 UI 表单操作不会覆写配置
- 日志分析采用纯 Go 流式解析，不依赖外部工具

### 遗留问题
- 日志文件较大时解析可能较慢，后续可考虑增量解析或 SQLite 缓存
- 源码模式暂不支持语法高亮（Nginx 语法），Monaco 使用 plaintext 模式

### 发布
- 版本 `v0.4.3`，已推送 tag 触发 GitHub Actions 自动构建

---

## 2026-03-09 — Session #24：终端焦点修复 + 版本号 bug + WebSocket 协议改进

### 完成内容

#### 终端 vim 焦点修复（根因定位）
- **核心问题**：终端创建后未调用 `terminal.focus()`，xterm.js 的内部 textarea 没有焦点，导致按键事件（i、o、: 等）无法被捕获传递给 vim
- **修复**：在 `createTerminal()` 首次 fit 后和 WebSocket 连接后都调用 `terminal.focus()`
- **点击聚焦**：为 `.terminal-container` 添加 `@click="focusActiveTerminal"` 处理器，点击任意位置都能恢复焦点
- **焦点恢复**：命令面板关闭后、视图切换回终端后自动 `focus()`
- **文件管理终端同步修复**：`terminal-dialog.vue` 同步添加焦点管理

#### 版本号缓存 bug 修复
- **根因**：Pinia store 使用 `persist: true`，Sidebar 仅在 `!globalStore.version` 时获取版本，缓存旧版本号后永远不会再请求后端
- **修复**：改为每次 `onMounted` 都从后端 API 获取真实版本号

#### 后端 WebSocket 协议改进
- **改进**：使用 `messageType`（TextMessage vs BinaryMessage）区分终端数据和控制帧（resize）
- **原因**：原实现通过 `msg[0] == 1` 检查内容首字节，理论上 Ctrl+A（ASCII 0x01）可能与 resize 控制帧冲突
- **影响**：本地终端和 SSH 终端两处 WebSocket 处理同步修改

### 发布
- 版本 `v0.4.2`，已推送 tag 触发 GitHub Actions 自动构建

---

## 2026-03-09 — Session #23：终端修复 + 首页信息增强 + 文件管理美化

### 完成内容

#### P0: 终端核心修复
- **vim 快捷键冲突修复**: 添加 `attachCustomKeyEventHandler` 自定义按键处理，仅将 Ctrl+Shift+C/V (复制粘贴)、F11/F12 交给浏览器，其他所有按键（Esc、Ctrl+C、方向键等）都传递给终端
- **最后一行显示截断修复**: 将 padding 从 `.terminal-instance` 内移到 `.terminal-container` 外层，确保 FitAddon 计算 rows/cols 时不受 padding 干扰；同时将 `window.resize` 替换为 `ResizeObserver` 精确监听容器尺寸变化
- **同步修复文件管理终端弹窗** (`terminal-dialog.vue`)

#### P1: 首页/监控信息增强
- **后端 `dto/monitor.go`**: SystemHostInfo 扩展新字段：PublicIPv4/IPv6、Interfaces (网卡IP/MAC/状态)、Timezone、Virtualization、DNSServers
- **后端 `service/monitor.go`**: 实现 `getNetworkInterfaces()`(net.Interfaces)、`getCachedPublicIP()`(ipify.org,缓存5分钟)、`getTimezone()`、`getDNSServers()`(/etc/resolv.conf)
- **前端首页**: 增加网络信息卡片（公网IP、各网卡IP、DNS），所有信息项增加悬浮显示的复制按钮
- **前端监控页**: 硬编码中文改为 i18n，增加公网IP/时区/虚拟化信息展示
- **i18n**: 新增 hostname/publicIPv4/publicIPv6/timezone/virtualization/physicalMachine 等 key

#### P2: 文件管理图标美化
- **新建 SVG 图标组件** `components/file-icons/FileIcon.vue`: 基于文件扩展名显示不同颜色的 SVG 图标，支持 50+ 文件类型（Go/JS/TS/Vue/Python/Rust/Shell/JSON/YAML/图片/视频/压缩包/证书等）
- **特殊目录图标**: .git(红色)、node_modules/vendor(绿色)、conf/config(蓝色)、log/logs(黄色)
- **目录大小计算**: 前端接入已有的 `getDirSize` API，目录大小列显示"计算"链接，点击异步计算并显示结果

#### P2: 终端快捷命令面板 + 批量输入增强
- **命令面板 (Ctrl+P)**: 类似 VSCode 的弹出面板，支持模糊搜索快速命令，上下键选择，回车执行到当前终端
- **批量输入增强**: 增加终端选择功能，可以勾选发送到哪些终端（全部/指定）

### 关键决策
- 公网IP获取使用 ipify.org API + 5分钟缓存，避免频繁外部请求
- 终端按键处理仅放行 Ctrl+Shift+C/V (运维常用复制粘贴) 和 F11/F12，其余全部交给终端
- 文件图标使用纯 SVG 组件而非引入图标字体库，保持零依赖

### 下一步计划
- 文件管理收藏夹功能（快速跳转常用路径）
- 历史监控趋势图表（CPU/内存 24h 趋势）
- 告警阈值配置
- 终端会话审计/回放

---

## 2026-02-09 — Session #22：文件管理功能增强

### 完成内容

#### 1. 文件搜索增加子目录递归搜索
- **后端 `backend/app/service/file.go`**: 新增 `searchRecursive` 函数，当 `ContainSub=true` 时使用 `find` 命令递归搜索子目录，结果限制 1000 条
- **后端 `backend/app/dto/file.go`**: `FileSearchReq` 已有 `ContainSub` 字段
- **前端工具栏**: 搜索框新增「子目录」复选框，勾选后搜索所有子目录

#### 2. 工具栏重新布局（参考 1Panel）
- **导航栏和工具栏分离**: 导航按钮（后退/前进/上级/刷新/路径输入）独立为 `.file-nav`
- **工具栏**: 创建按钮改为下拉菜单，新增远程下载按钮，批量操作（复制/剪切/压缩/权限/删除）改为按钮组
- **隐藏文件**: 改为圆形图标按钮（眼睛图标）
- **剪贴板**: 粘贴按钮改为按钮组（粘贴 + 取消）

#### 3. 压缩/解压缩功能增强
- **修复路径问题**: `Compress` 方法使用 `-C dir base` 模式（tar）和 `cmd.Dir`（zip），避免解压时出现绝对路径
- **支持更多格式**: 新增 `detectArchiveType` 函数，支持 `.7z`（需要 7z 命令）和 `.rar`（需要 unrar 命令）解压
- **多后缀识别**: 正确识别 `.tar.gz`、`.tar.bz2`、`.tar.xz`、`.tgz`、`.tbz2`、`.txz` 等复合后缀

#### 4. 远程下载功能
- **后端**: 新增 `FileService.Wget` 方法，使用 `wget -q -P` 下载文件到指定目录
- **后端 API**: 新增 `POST /files/wget` 路由和 `WgetFile` handler
- **前端**: 新增远程下载弹窗，输入 URL 即可下载到当前目录

#### 5. 其他改进
- **i18n**: 新增 `containSub`、`remoteDownload`、`remoteUrl`、`remoteUrlPlaceholder` 翻译
- **错误码**: 新增 `ErrCmdNotFound` 错误码，解压 7z/rar 时若命令不存在给出友好提示

### 遗留问题
- 批量权限修改当前仅打开第一个文件的权限弹窗，后续可扩展为真正的批量修改
- 远程下载暂不支持进度显示

### 下一步计划
- 可选：添加文件在线预览（图片/PDF/视频）
- 可选：优化大目录加载性能（分页/虚拟滚动）

---

## 2026-02-09 — Session #21：设置页完善 + 侧边栏版本号动态化 (v0.3.1)

### 完成内容

#### 1. 侧边栏版本号动态化
- **`frontend/src/layout/components/Sidebar.vue`**: 硬编码的 `v0.1.0` 改为从后端 API 动态获取
- **`frontend/src/store/modules/global.ts`**: 新增 `version` 字段和 `setVersion` action
- **`frontend/src/views/home/index.vue`**: 获取版本后同步到 global store

#### 2. 设置页新增端口/用户名/密码修改
- **`frontend/src/views/setting/index.vue`**: 新增"端口设置"、"用户名与密码"两个卡片
- **`backend/app/service/setting.go`**: 新增 `UpdatePort` 方法（写入 config.yaml），`UserName` 加入可更新 key
- **`backend/app/api/v1/setting.go`**: 新增 `UpdatePort` handler
- **`backend/app/dto/setting.go`**: 新增 `PortUpdate` DTO，`SettingInfo` 增加 `serverPort` 字段
- **`backend/router/router.go`**: 新增 `POST /settings/port/update` 路由
- **`frontend/src/api/modules/setting.ts`**: 新增 `updatePort` API
- **`frontend/src/layout/components/Header.vue`**: 修改密码按钮改为跳转到设置页

#### 3. 版本号规则写入项目规则
- **`.cursor/rules/x-panel.mdc`**: 新增"版本号规则"章节，明确语义化版本递增策略

#### 4. i18n 翻译补充
- **`frontend/src/i18n/zh.ts`**: 新增端口设置、用户名密码相关翻译 key

### 关键决策
- 端口修改写入 config.yaml，需要重启服务才能生效（前端提示用户）
- 用户名修改直接更新数据库 Setting 表
- 密码修改复用已有 `/auth/password` API
- 版本号采用 PATCH 递增策略，每次发布 +0.0.1

### 遗留问题
- 端口修改后需要手动重启服务（暂不做自动重启，避免意外）

### 下一步计划
- 测试 v0.3.1 自动更新功能

---

## 2026-02-09 — Session #20：概览页面（Dashboard）重做

### 完成内容

#### 1. 概览页面全面重构
- **`frontend/src/views/home/index.vue`**: 参考 1Panel 概览页面风格，完全重写首页
  - 顶部展示主机名、面板版本、系统标签、运行时间
  - 系统详情卡片：操作系统、内核、架构、CPU 型号、核心数、总内存
  - **资源占用使用进度条风格**（非圆形仪表盘）：CPU、内存、负载、网络
  - 进度条颜色根据使用率动态变化：正常（青色/紫色）→ 警告（黄色）→ 危险（红色）
  - 磁盘使用独立区域，每块磁盘一张卡片，含进度条 + inode 信息
  - 快速入口：文件管理、终端、Nginx、SSL、防火墙、进程管理、设置、日志
  - Top 进程表格（CPU 占用前 5）
  - 每 5 秒自动刷新数据

#### 2. i18n 翻译补充
- **`frontend/src/i18n/zh.ts`**: 新增概览页面所需的全部翻译 key

### 关键决策
- 资源占用使用横向进度条而非圆形仪表盘，更直观且节省空间
- 后端 API 已有 `/monitor/stats` 返回完整系统状态，无需新增后端代码
- 保持深色科技风主题一致性，使用 `--xp-*` CSS 变量

---

## 2026-02-09 — Session #19：安装脚本增强

### 完成内容

#### 1. 自定义安装路径
- **`scripts/install-online.sh`**: 新增 `--path <路径>` 参数，允许用户指定 X-Panel 安装路径（默认 `/opt/xpanel`）
- 所有路径引用（配置文件、数据目录、Nginx 目录、SSL 证书、systemd 服务）均基于自定义路径动态生成
- 卸载命令自动附带 `--path` 参数（当使用非默认路径时）

#### 2. SQLite3 依赖自动检测与安装
- 安装脚本在安装前自动检测 `sqlite3` 是否可用
- 支持 apt-get / yum / dnf / apk / pacman 多种包管理器自动安装
- 自动安装失败时交互式询问用户是否继续（`--yes` 模式自动跳过并继续）
- 无 sqlite3 时安全入口需在面板 Web 界面中手动配置

#### 3. 默认端口改为 7777
- **`scripts/install-online.sh`**: `DEFAULT_PORT` 从 `9999` 改为 `7777`
- **`backend/init/viper/viper.go`**: 默认端口改为 `7777`
- **`backend/configs/config.yaml`**: 开发配置端口改为 `7777`
- **`frontend/vite.config.ts`**: 代理目标端口改为 `7777`
- **`scripts/install.sh`**: 本地安装脚本端口改为 `7777`
- **`README.md`**: 更新所有端口引用和参数说明，新增 `--path` 参数文档
- **`docs/quick-start.md`**: 更新所有端口引用
- **`docs/development-guide.md`**: 更新配置示例端口

### 关键决策
- 默认端口选择 7777：避免与常见服务端口冲突，且易于记忆
- sqlite3 采用"尽力自动安装 + 优雅降级"策略：不阻塞面板安装，仅影响命令行配置安全入口

### 遗留问题
- 暂无

### 下一步计划
- 暂无

---

## 2026-02-09 — Session #18：Nginx 预编译仓库 + 下载安装模式

### 完成内容

#### 新建 nginx-build 预编译仓库 (`/data/nginx-build/`)
- [x] `.github/workflows/build.yml`：GitHub Actions 自动编译工作流
  - 支持 `v*` tag 触发和 `workflow_dispatch` 手动触发
  - 为 amd64 (原生编译) 和 arm64 (QEMU/Docker) 编译
  - 产物发布为 GitHub Release（tar.gz + sha256 校验）
- [x] `build.sh`：Nginx 编译脚本
  - 使用 `--prefix=/opt/xpanel/nginx` 作为编译前缀
  - 包含模块: http_ssl, http_v2, http_realip, http_gzip_static, http_stub_status, stream, stream_ssl, pcre
  - 使用 `DESTDIR` 分阶段安装，自动创建 conf.d/ssl/temp 等目录
- [x] `README.md`：仓库使用说明

#### 后端改造 (X-Panel)
- [x] `backend/app/service/nginx_install.go` — **完全重写**
  - `Install()`: 改为从 GitHub Release 下载预编译 Nginx
  - `doInstall()`: 下载 → SHA256 校验 → 解压安装 → 创建目录结构 → 更新配置
  - `ListVersions()`: 新增，从 GitHub API 获取可用版本列表
  - `Uninstall()`: 保留，调整为使用 `-p installDir` 停止 Nginx
  - 移除 `CheckDeps()`: 预编译模式不再需要编译依赖检查
- [x] `backend/app/service/nginx.go` — 所有 nginx 命令增加 `-p installDir` 参数
  - `start()`, `reload()`, `signal()`, `TestConfig()`, `GetStatus()` 全部传 `-p`
  - 确保预编译二进制在任何安装目录下都能正确找到配置和日志
- [x] `backend/app/dto/nginx.go` — 新增 `NginxVersionInfo` DTO（version, tag, publishedAt）
- [x] `backend/app/api/v1/nginx.go` — 新增 `ListNginxVersions` handler，移除 `CheckNginxDeps`
- [x] `backend/router/router.go` — 路由 `/nginx/deps` 替换为 `/nginx/versions`
- [x] `backend/global/global.go` — `NginxConfig` 新增 `BuildRepo` 字段
- [x] `backend/init/viper/viper.go` — 默认值 `nginx.build_repo: Anikato/nginx-build`
- [x] `backend/configs/config.yaml` — 新增 `build_repo` 配置

#### 前端改造
- [x] `frontend/src/views/website/nginx/index.vue` — 重写安装 UI
  - 移除"检查依赖"按钮和依赖检查结果展示
  - 安装对话框改为版本下拉选择（从后端获取可用版本列表）
  - 无可用版本时显示警告并允许手动输入
  - 进度状态移除 configure/compile，新增 verify（校验）
- [x] `frontend/src/api/modules/nginx.ts` — 移除 `checkNginxDeps`，新增 `listNginxVersions`
- [x] `frontend/src/i18n/zh.ts` — 更新翻译
  - 移除编译相关（checkDeps, depsOk, depsMissing, phaseConfigure, phaseCompile）
  - 新增预编译相关（selectVersion, noVersions, phaseVerify）
  - 修改安装确认文案

### 关键决策
1. **预编译仓库独立于 X-Panel**：`Anikato/nginx-build` 独立管理，tag = Nginx 版本号
2. **编译前缀固定为 `/opt/xpanel/nginx`**：匹配 X-Panel 默认安装目录
3. **运行时 `-p` 参数**：所有 nginx 命令传 `-p installDir`，确保在不同安装目录下也能工作
4. **arm64 使用 QEMU/Docker 编译**：GitHub Actions runner 原生 amd64，arm64 通过 Docker 交叉编译

### 遗留问题
- nginx-build 仓库需要用户在 GitHub 上创建并推送

### 下一步计划
- 创建 `Anikato/nginx-build` GitHub 仓库并推送编译配置
- 发布第一个 Nginx 预编译版本 (v1.26.2)

---

## 2026-02-08 — Session #17：HTTPS 默认启用 + 安全入口 + 自定义端口

### 完成内容

#### 后端：HTTPS 支持
- [x] `global.go` 新增 `SSLConfig` 结构体（enable, cert_path, key_path）
- [x] `server.go` 根据配置选择 `r.RunTLS()` 或 `r.Run()` 启动
- [x] `viper.go` 添加 SSL 默认配置
- [x] `config.yaml` 新增 `ssl` 配置节

#### 后端：安全入口中间件
- [x] 新增 `middleware/security_entrance.go`
  - 从数据库读取 `SecurityEntrance` 配置
  - 未配置时不生效，直接放行
  - 配置后：访问 `/{entrance}` 设置 cookie 并重定向到首页
  - 后续访问检查 cookie，无效则返回 404
  - API 路由 (`/api/*`) 不受限制（由 JWT 保护）
- [x] `router.go` 全局挂载安全入口中间件

#### 前端：设置页面
- [x] 面板设置增加安全入口输入框
- [x] 保存时同步更新 `SecurityEntrance` 配置
- [x] 新增 i18n 翻译

#### 安装脚本增强
- [x] `--port, -p <端口>` 自定义面板端口
- [x] `--entrance, -e <路径>` 配置安全入口
- [x] `--ssl` / `--no-ssl` 控制 HTTPS（默认启用）
- [x] 安装时自动生成 10 年有效期的自签名 SSL 证书
- [x] 安全入口通过 `sqlite3` 写入数据库
- [x] 完成信息显示正确的协议 / 端口 / 入口路径

#### 仓库公开化
- [x] GitHub 仓库设为公开
- [x] 移除代码中的 GitHub Token（Push Protection）
- [x] 简化 README 安装命令
- [x] 更新 cursor rules

### 关键决策
- SSL 采用自签名证书方案：安装简单、无需域名，用户可后续替换为正式证书
- 安全入口基于 cookie 机制：首次访问入口路径设 cookie，后续检查 cookie
- API 请求不受安全入口限制，避免影响 JWT 认证流程

### 新增文件
- `backend/middleware/security_entrance.go`

### 下一步计划
- 网站管理（Nginx 站点 CRUD）
- 数据库管理（MySQL/PostgreSQL）
- 面板 SSL 证书支持 Let's Encrypt 替换自签名

---

## 2026-02-08 — Session #16：GitHub CI/CD 自动构建 + 升级系统重写

### 完成内容

#### GitHub Actions 自动构建发布
- [x] 新增 `.github/workflows/release.yml`：
  - Tag 推送（`v*`）自动触发构建
  - 矩阵构建 linux/amd64 + linux/arm64 双架构
  - 自动构建前端（npm ci）→ 嵌入后端 → 交叉编译 Go
  - 生成 tar.gz 安装包 + SHA256 校验文件
  - 自动创建 GitHub Release 并上传产物
  - 支持 pre-release 标记（beta/rc/alpha 标签自动识别）

#### 后端：升级服务完全重写 (`service/upgrade.go`)
- [x] **GitHub Releases API 集成**：默认从 `Anikato/x-panel` 的 GitHub Releases 检查更新
  - 解析 GitHub Release 响应，自动匹配当前架构的下载文件
  - 兼容保留自建服务器 `version.json` 模式
  - 自动识别 GitHub URL vs 自定义 URL
- [x] **语义化版本比较** (`compareVersions`)：
  - 支持 `v` 前缀、三段版本号、pre-release 标识（beta/rc）
  - `dev` 版本视为最低版本
  - 不再使用简单 `!=` 比较，避免降级误判
- [x] **升级互斥锁**：`sync.Mutex` 防止并发升级
- [x] **原子二进制替换**：先 copy 到 `.new` 文件再 `os.Rename`，失败回退 `copyFile`
- [x] **SHA256 校验**：下载后验证文件完整性，防止篡改或传输损坏
- [x] 新增错误常量 `ErrUpgradeInProgress`

#### DTO 更新 (`dto/upgrade.go`)
- [x] 新增 `GitHubRelease` / `GitHubAsset` 结构体，对应 GitHub API 响应
- [x] `UpgradeInfo` 新增 `ChecksumURL` 字段
- [x] `UpgradeReq` 新增 `ChecksumURL` 字段
- [x] 保留 `RemoteVersionInfo` 兼容自建服务器

#### 数据迁移
- [x] `migration.go` 新增 `UpgradeURL` 默认设置项

#### 前端更新
- [x] `api/modules/upgrade.ts`：`doUpgrade` 新增 `checksumUrl` 参数
- [x] `views/setting/index.vue`：
  - 更新源输入框默认留空（自动使用 GitHub）
  - 添加提示说明文字
  - 升级请求传递 `checksumUrl`
  - 添加 `onUnmounted` 清理定时器
  - 修复模板中缺失的 `<el-alert>` 标签
- [x] `i18n/zh.ts`：新增 `upgradeUrlHint` 翻译

#### Makefile 更新
- [x] `package` 目标新增 SHA256 校验和生成

### 关键技术决策

1. **更新源选择 GitHub Releases**：
   - 无需自建更新服务器
   - GitHub Actions 推送 tag 自动构建发布
   - 用户只需 `git tag v1.0.0 && git push --tags` 即可发布新版
   - GitHub API 60次/小时免认证限额足够日常检查

2. **双模式兼容**：
   - 默认 GitHub Releases（留空或 GitHub URL）
   - 自定义 URL 走旧版 `version.json` 协议
   - 用户可在面板设置中覆盖更新源

3. **安全加固**：
   - SHA256 checksum 校验下载完整性
   - `os.Rename` 原子替换减少损坏窗口
   - 互斥锁防止并发升级

### 新增/修改文件
- `NEW` `.github/workflows/release.yml`
- `MOD` `backend/app/service/upgrade.go`（完全重写）
- `MOD` `backend/app/dto/upgrade.go`
- `MOD` `backend/constant/errs.go`
- `MOD` `backend/init/migration/migration.go`
- `MOD` `frontend/src/api/modules/upgrade.ts`
- `MOD` `frontend/src/views/setting/index.vue`
- `MOD` `frontend/src/i18n/zh.ts`
- `MOD` `Makefile`

### 发布流程（使用方法）
```bash
# 1. 确保代码已推送到 GitHub
git push origin main

# 2. 创建版本标签并推送
git tag v1.0.0
git push origin v1.0.0

# 3. GitHub Actions 自动执行：
#    构建前端 → 编译后端(amd64+arm64) → 打包 → 创建 Release

# 4. 面板自动从 GitHub Releases 检查更新
#    用户在设置页点击"检查更新" → 发现新版本 → 确认升级
```

#### 一键安装脚本 (`scripts/install-online.sh`)
- [x] 基于 GitHub Releases 的在线一键安装脚本
  - 自动检测系统架构（amd64/arm64）
  - 从 GitHub Releases 下载最新版本
  - SHA256 校验文件完整性
  - 自动生成配置文件（随机 JWT Secret、生产模式）
  - 自动配置 systemd 服务并启动
  - 支持 `--version` 指定版本安装
  - 支持 `--uninstall` 卸载
  - 自动检测升级模式（已安装时停止、备份、替换）
  - 安装完成显示访问地址和常用命令

### 遗留与下一步
- [ ] 添加下载进度反馈（百分比）
- [ ] 升级历史记录（数据库模型 + 回滚 API）
- [ ] 自动定期检查更新（后台 cron + 全局通知）
- [ ] GitHub Release 代理加速（国内用户场景）

---

## 2026-02-06 — Session #15：构建系统与自更新发布功能

### 完成内容

#### 架构梳理
- [x] 确认 X-Panel 当前为**单进程架构**（非 Core+Agent 分离），前端未嵌入后端

#### 后端：版本管理基础
- [x] 新增 `app/version/version.go`：版本信息（Version/CommitHash/BuildTime/GoVersion），通过 `-ldflags` 编译时注入
- [x] 新增 `cmd/server/web/embed.go`：使用 `go:embed` 将前端构建产物嵌入 Go 二进制
- [x] 修改 `router/router.go`：
  - 新增公开 `/api/v1/version` 端点
  - 新增 `setupFrontend()` 函数，生产模式下直接从嵌入 FS 提供 SPA 静态文件服务
  - NoRoute 处理器支持静态资源 + SPA 回退 index.html

#### 后端：自更新服务
- [x] 新增 `dto/upgrade.go`：VersionInfo、UpgradeCheckReq、RemoteVersionInfo、UpgradeInfo、UpgradeReq
- [x] 新增 `service/upgrade.go`（IUpgradeService 接口 + 4 个方法）：
  - `GetCurrentVersion`：返回编译时注入的版本信息
  - `CheckUpdate`：从远端 `{releaseURL}/version.json` 检查新版本，自动拼接平台下载 URL
  - `DoUpgrade`：后台异步执行升级（下载 → 解压 → 备份 → 替换 → systemctl 重启）
  - `GetUpgradeLog`：读取升级日志文件
- [x] 新增 `api/v1/upgrade.go`：4 个 API Handler
- [x] 新增路由：`GET /upgrade/current`、`POST /upgrade/check`、`POST /upgrade/do`、`GET /upgrade/log`
- [x] 注册 UpgradeAPI 到 entry.go

#### 构建与部署
- [x] 新增 `Makefile`（项目根目录）：
  - `make build`：构建前端 + 嵌入 + 构建后端（完整流程）
  - `make build_frontend` / `make build_backend`：分步构建
  - `make package`：打包为 `xpanel-{version}-{os}-{arch}.tar.gz`
  - `make build_linux_amd64` / `make build_linux_arm64`：交叉编译
  - 版本信息通过 ldflags 自动注入（git describe、commit hash、构建时间）
- [x] 新增 `scripts/xpanel.service`：Systemd 服务文件
- [x] 新增 `scripts/install.sh`：安装脚本（创建目录、复制二进制、生成配置、注册服务）
- [x] 新增 `scripts/gen-version-json.sh`：生成更新服务器所需的 version.json

#### 前端：版本与升级 UI
- [x] 新增 `api/modules/upgrade.ts`：4 个 API 函数（getCurrentVersion、checkUpdate、doUpgrade、getUpgradeLog）
- [x] 重写 `views/setting/index.vue`：
  - 新增「版本信息」卡片：显示版本号、构建时间、提交哈希、Go 版本
  - 自定义更新源输入框 + 检查更新按钮
  - 更新结果展示（无更新/有更新 + 更新说明 + 发布日期）
  - 一键升级按钮 + 升级确认弹框
  - 升级日志轮询与实时显示
  - 开发版本（dev）标识 + 提示
- [x] 更新 `views/home/index.vue`：首页版本号从 API 动态获取
- [x] 新增 i18n 翻译：~20 个升级相关 key

### 新增文件清单
| 文件 | 说明 |
|------|------|
| `backend/app/version/version.go` | 版本信息（ldflags 注入） |
| `backend/cmd/server/web/embed.go` | 前端嵌入（go:embed） |
| `backend/cmd/server/web/assets/index.html` | 开发模式占位文件 |
| `backend/app/dto/upgrade.go` | 升级相关 DTO |
| `backend/app/service/upgrade.go` | 升级服务 |
| `backend/app/api/v1/upgrade.go` | 升级 API Handler |
| `frontend/src/api/modules/upgrade.ts` | 前端升级 API |
| `Makefile` | 项目构建系统 |
| `scripts/xpanel.service` | Systemd 服务 |
| `scripts/install.sh` | 安装脚本 |
| `scripts/gen-version-json.sh` | 版本 JSON 生成 |

### 技术决策
1. **单二进制分发**：通过 `go:embed` 将前端 dist 嵌入 Go 二进制，生产部署只需一个文件
2. **版本注入方式**：使用 `go build -ldflags` 在编译时注入，不需要额外配置文件
3. **更新检查协议**：远端放置 `version.json`（版本号+说明+日期），下载包命名 `xpanel-{ver}-linux-{arch}.tar.gz`
4. **升级安全策略**：下载 → 备份当前二进制为 .bak → 替换 → systemctl restart，失败自动回滚
5. **SPA 服务**：NoRoute handler 先尝试 fs.Stat 静态文件，失败则回退 index.html

### 完整构建发布流程
```
make build    →  构建前端 + 嵌入 + 构建后端
make package  →  打包为 tar.gz（含二进制+配置模板+服务文件+安装脚本）
                 部署到服务器后执行 install.sh
make help     →  查看所有可用目标
```

### 遗留/后续
- [ ] GitHub Actions CI/CD 自动构建发布
- [ ] 升级过程中前端显示维护页面
- [ ] 版本对比逻辑（semver 比较而非简单字符串比较）
- [ ] 升级回滚功能（手动从 .bak 恢复）
- [ ] 多架构自动发布（amd64 + arm64）

---

## 2026-02-06 — Session #14：文件管理 Phase 1 实施（核心体验补齐）

### 完成内容

#### 后端：质量修复
- [x] **SaveContent 权限保持**：保存文件前读取原文件 `FileMode`，写入时使用原权限而非固定 `0644`
- [x] **路径安全增强**：新增 `isProtectedPath()`（20+ 系统关键目录）和 `isInvalidChar()`（空字符/换行/首尾空格），Create/Rename/Compress 均校验
- [x] **Create 权限继承**：支持 `mode` 参数指定权限，默认继承父目录权限
- [x] **Move 冲突处理**：支持 `cover` 参数覆盖同名文件，自动处理跨分区移动（cp+rm）
- [x] **Move 安全检查**：防止移动到自身内部，cp 使用 `-rp` 保留权限

#### 后端：新增接口（6 个）
- [x] `ChangeOwner`：chown 修改所有者，支持 `-R` 递归
- [x] `GetUsersAndGroups`：读取 /etc/passwd 和 /etc/group，返回可用用户和组列表
- [x] `ChangeMode` 增强：支持 `sub` 参数递归修改子目录权限（使用 `chmod -R`）
- [x] `GetFileTree`：目录树接口，浅层展开（只返回目录），用于路径选择器
- [x] `GetDirSize`：使用 `du -sb` 计算目录大小
- [x] `ListFiles` 增强：支持 `search` 参数后端过滤 + `sortBy`/`sortOrder` 排序

#### 后端：DTO 增强
- [x] `FileInfo` 新增字段：`modeNum`(八进制)、`isSymlink`、`linkPath`、`uid`、`gid`、`extension`
- [x] 新增 DTO：`FileChownReq`、`FileTreeReq`、`FileTreeNode`、`UserInfo`、`UserGroupResp`、`DirSizeReq/Resp`
- [x] `FileModeReq` 新增 `sub` 字段、`FileCreateReq` 新增 `mode` 字段、`FileMoveReq` 新增 `cover` 字段

#### 后端：路由新增
- [x] `POST /files/owner` — 修改所有者
- [x] `POST /files/tree` — 文件树
- [x] `POST /files/size` — 目录大小
- [x] `POST /files/user/group` — 用户和组列表

#### 前端：多 Tab 浏览系统
- [x] `el-tabs` 多标签页：每个 Tab 维护独立路径和历史栈
- [x] 新建标签默认继承当前标签路径
- [x] 标签可关闭（保留至少一个）、可切换
- [x] Tab 状态结构：`{ id, name, path, historyBack[], historyForward[] }`

#### 前端：导航前进/后退
- [x] 每个 Tab 独立维护 `historyBack` 和 `historyForward` 栈
- [x] 工具栏：后退/前进/上级目录三按钮，禁用状态联动
- [x] 新导航自动清空前进历史

#### 前端：文件搜索
- [x] 搜索框（工具栏右侧）：输入文件名实时过滤
- [x] 后端 `search` 参数：大小写不敏感包含匹配
- [x] 防抖 300ms 避免频繁请求

#### 前端：拖拽上传
- [x] 文件管理器整体响应 dragover/drop 事件
- [x] 拖拽时显示覆盖层提示："拖拽文件到此处上传"
- [x] 支持多文件同时拖入上传

#### 前端：新组件
- [x] `chown-dialog.vue`：修改所有者弹窗，自动加载系统用户/组列表，filterable 下拉选择，支持递归
- [x] `detail-drawer.vue`：文件详情面板（Drawer），显示名称/类型/路径/大小/权限/UID/GID/修改时间，目录大小可按需计算
- [x] 权限弹窗增强：新增"递归应用到子目录"选项

#### 前端：集成
- [x] 右键菜单新增：详情、修改所有者
- [x] 操作列新增：详情按钮
- [x] 下拉菜单新增：修改所有者
- [x] 所有者列可点击打开 chown 弹窗
- [x] API 模块新增：`changeFileOwner`、`getUsersAndGroups`、`getFileTree`、`getDirSize`

#### i18n 翻译
- [x] 新增 ~25 条中文翻译（多Tab/导航/搜索/详情/Chown/拖拽等）
- [x] 新增后端错误码翻译：`ErrFileInvalidChar`、`ErrFileChown`

### 技术决策
- 多 Tab 使用 `el-tabs` 原生 editable 模式，每个 Tab 是独立的 `TabState` 对象
- 导航历史栈使用数组 push/pop，前进历史在新导航时清空（标准浏览器行为）
- 文件搜索通过后端 `search` 参数实现，避免前端持有全部文件列表
- chown 使用 `exec.Command("chown")` 调用系统命令，支持递归
- 文件树接口采用懒加载：只返回第一层目录，检查是否有子目录标记可展开

### 文件变更
- `backend/app/dto/file.go` — 全面增强，新增 6 个 DTO 类型
- `backend/app/service/file.go` — 重写，新增 6 个方法 + 修复 3 个已有方法
- `backend/app/api/v1/file.go` — 新增 4 个 handler
- `backend/router/router.go` — 新增 4 条路由
- `backend/constant/errs.go` — 新增 2 个错误码
- `backend/i18n/lang/zh.yaml` — 新增 2 条错误翻译
- `frontend/src/api/modules/file.ts` — 新增 4 个 API 方法，更新参数类型
- `frontend/src/i18n/zh.ts` — 新增 ~25 条翻译
- `frontend/src/views/host/file/index.vue` — 重写（多Tab + 导航栈 + 搜索 + 拖拽 + 详情 + chown 集成）
- `frontend/src/views/host/file/chown-dialog.vue` — **新建**
- `frontend/src/views/host/file/detail-drawer.vue` — **新建**
- `frontend/src/views/host/file/permission-dialog.vue` — 增加递归选项

### 当前覆盖率提升
| 维度 | 之前 | 之后 | 变化 |
|------|------|------|------|
| 后端方法 | 11/29 (38%) | 17/29 (59%) | +21% |
| 后端路由 | 13/45 (29%) | 17/45 (38%) | +9% |
| 前端组件 | 5/26 (19%) | 7/26 (27%) | +8% |
| 功能覆盖 | Phase 0 | Phase 1 完成 | 核心体验补齐 |

### 下一步计划（Phase 2）
- 文件预览（图片/视频/音频）
- 回收站系统
- 分片上传 + 进度条
- Wget 远程下载
- 收藏夹

---

## 2026-02-06 — Session #13：文件管理系统深度差距分析

### 完成内容

#### 与 1Panel 文件管理全面对比
- [x] 系统性分析 1Panel 后端：3 个 Service（File + RecycleBin + Favorite）共 29+ 方法
- [x] 系统性分析 1Panel 前端：26 个组件，45+ API 端点
- [x] 逐项对比 X-Panel 现状：11 方法 / 13 端点 / 5 组件
- [x] 整体覆盖率：后端方法 ~38%，路由 ~29%，前端组件 ~19%

#### 已实现功能的质量差距识别
- [x] `SaveContent` 固定 0644 权限 → 1Panel 保留原文件权限模式
- [x] `GetContent` 无编码检测 → 1Panel 自动检测 GBK/GB2312 并转换
- [x] `ChangeMode` 不支持递归 → 1Panel `ChmodR` 递归修改子目录
- [x] `Create` 不支持链接 → 1Panel 支持软链接/硬链接创建
- [x] `Move` 无冲突处理 → 1Panel 支持覆盖/coverPaths
- [x] 压缩/解压无加密 → 1Panel 支持密码加密
- [x] 路径安全校验不足 → 1Panel 有 `IsInvalidChar` + `IsProtected` + 过滤路径

#### 完全缺失功能清单（18 项）
- [x] **P0（6项）**：ChangeOwner、文件树、多Tab浏览、分片上传、导航前进/后退、文件搜索
- [x] **P1（9项）**：回收站、收藏夹、Wget远程下载、文件预览、文件详情面板、目录大小计算、下载增强、拖拽上传、批量权限修改
- [x] **P2（3项）**：文件备注(xattr)、文件格式转换、挂载点信息

#### 实施路线图规划
- [x] Phase 1（核心体验）：chown + 权限保持 + 安全校验 + 多Tab + 导航栈 + 搜索
- [x] Phase 2（功能扩展）：详情面板 + 预览 + 拖拽上传 + 文件树 + 回收站
- [x] Phase 3（高级功能）：分片上传 + Wget + 收藏夹 + 批量权限 + 编码转换

### 关键发现
- 1Panel 的 `utils/files` 包封装了 `FileOp` 统一文件操作层，X-Panel 直接用 `os.*` 和 `exec.Command`
- 1Panel 回收站基于分区级 `.1panel_clash` 隐藏目录实现，跨分区感知
- 1Panel 收藏夹用 DB 存储（model.Favorite），文件备注用 xattr 存储
- 1Panel 前端文件管理主组件 ~2100 行，功能密度极高

### 下一步计划
- 实施 Phase 1：ChangeOwner + SaveContent 权限保持 + 路径安全增强
- 前端：多 Tab 浏览 + 导航前进/后退 + 文件搜索

---

## 2026-02-06 — Session #12：文件管理增强（Monaco Editor + 内嵌终端 + 全功能对接）

### 完成内容

#### 与 1Panel 文件管理对比分析
- [x] 系统性分析 1Panel 文件管理功能（26 个组件）vs X-Panel 现状
- [x] 按优先级分类：高/中/低，确定实施路线

#### Monaco Editor 代码编辑器
- [x] 安装 `monaco-editor` 依赖
- [x] 创建 `code-editor.vue`：从右侧 Drawer 打开，支持 ~20 种语言语法高亮
- [x] 按文件扩展名自动识别语言（js/ts/py/go/sh/json/yaml/html/css 等）
- [x] 支持主题切换（Dark / Light / High Contrast）
- [x] 内置 Ctrl+S 快捷键保存
- [x] 缩略图 (minimap)、自动换行、行号、折叠、括号颜色配对
- [x] 未保存提示：关闭时如有修改弹出确认

#### 文件管理内嵌终端
- [x] 创建 `terminal-dialog.vue`：从底部 Drawer 打开 xterm.js 终端
- [x] 自动 `cd` 到当前浏览目录
- [x] 复用已有 WebSocket 终端后端

#### 右键上下文菜单
- [x] 完整右键菜单：打开/编辑/下载/复制路径/复制到/移动到/重命名/权限/压缩/解压/删除
- [x] Teleport 定位，点击外部自动关闭

#### 对接已有后端
- [x] 压缩弹窗（`compress-dialog.vue`）：选择格式(tar.gz/zip)、目标路径
- [x] 解压弹窗：选择解压目标路径
- [x] 权限修改弹窗（`permission-dialog.vue`）：读/写/执行 checkbox + 八进制代码双向联动
- [x] 从 `-rwxr-xr-x` 格式自动解析权限初始值
- [x] 移动/复制：剪贴板模式（选择→复制/剪切→导航→粘贴），工具栏状态提示

#### 文件管理器 UI 全面升级
- [x] 文件类型图标区分：文件夹(cyan)/图片(pink)/视频(orange)/音频(purple)/压缩包(yellow)/代码(green)/配置(blue)
- [x] 操作列重构：编辑/下载 + "更多"下拉菜单（重命名/复制路径/复制/移动/权限/压缩/解压/删除）
- [x] 权限列可点击直接打开权限修改弹窗
- [x] 表格高度自适应窗口
- [x] 批量操作增强：选中项显示批量压缩和批量删除按钮

### 技术决策
- Monaco Editor 直接使用 Vite worker 导入模式，无需额外插件
- 终端弹窗复用现有 WebSocket 协议，连接后自动执行 `cd` 命令
- 权限修改使用 checkbox 和八进制代码双向绑定，解析 `ls -l` 格式字符串

### 文件变更
- `frontend/package.json` — 新增 `monaco-editor`
- `frontend/src/i18n/zh.ts` — 新增文件管理相关翻译 (~40 条)
- `frontend/src/views/host/file/index.vue` — 全面重写文件管理器
- `frontend/src/views/host/file/code-editor.vue` — 新建 Monaco Editor 组件
- `frontend/src/views/host/file/terminal-dialog.vue` — 新建内嵌终端组件
- `frontend/src/views/host/file/compress-dialog.vue` — 新建压缩/解压弹窗
- `frontend/src/views/host/file/permission-dialog.vue` — 新建权限修改弹窗

### 下一步计划
- 多 Tab 浏览支持
- 文件搜索（目录内搜索）
- 文件预览（图片/视频/音频）
- 拖拽上传
- 回收站
- 收藏夹

---

## 2026-02-06 — Session #11：SSL 证书申请日志系统

### 完成内容

#### 与 1Panel 证书系统对比分析
- [x] 对比 1Panel `WebsiteSSLService.ObtainSSL()` 的日志机制：每证书独立 `.log` 文件 + `log.Logger` 全程记录
- [x] 识别核心差距：申请日志、错误信息展示、日志查看 UI

#### 后端：证书日志文件系统
- [x] 新增 `getSSLLogDir()` / `getSSLLogPath()` / `openSSLLog()` 辅助函数
- [x] 日志路径规则：`{sslDir}/logs/{domain}-ssl-{id}.log`（参考 1Panel）
- [x] 改造 `Apply()` 方法：全程 logger 记录（开始 → ACME客户端 → DNS配置 → 申请 → 成功/失败 → 证书信息 → 文件保存）
- [x] 改造 `Renew()` 方法：同样增加全程日志
- [x] 新增 `GetLog(id)` 接口：读取日志文件返回内容

#### 后端 API
- [x] 新增 `POST /certificates/log` 路由 → `GetCertificateLog` handler

#### 前端：日志查看 + 错误提示
- [x] 证书列表新增「日志」列 + "查看"按钮
- [x] 「申请日志」弹窗：等宽字体 + 暗色代码块 + 刷新/关闭按钮
- [x] 错误状态 hover popover：鼠标悬浮"错误"标签显示具体 error message
- [x] 申请中状态：显示 Loading 图标
- [x] 申请中自动轮询：3 秒刷新日志内容 + 5 秒刷新证书列表状态
- [x] 引入 `getCertificateLog` API 方法

### 关键决策
- 日志采用文件级存储（非数据库），与 1Panel 一致，便于大日志文件和运维查看
- Apply/Renew 内同步写日志（非 goroutine），确保日志完整性
- 日志格式 `时间戳 [标签] 内容`，标签包括：开始/信息/成功/错误/警告/完成

### 涉及文件
| 文件 | 变更 |
|------|------|
| `backend/app/service/ssl.go` | Apply/Renew 增加 logger + GetLog + 辅助函数 |
| `backend/app/api/v1/ssl.go` | 新增 GetCertificateLog handler |
| `backend/router/router.go` | 新增 /certificates/log 路由 |
| `frontend/src/api/modules/ssl.ts` | 新增 getCertificateLog |
| `frontend/src/views/website/ssl/index.vue` | 日志弹窗 + 错误 popover + 申请中轮询 |

### 与 1Panel 剩余差距
| 功能 | 状态 | 优先级 |
|------|------|--------|
| 自签证书 (CA) | 未实现 | 中 |
| 推送到自定义目录 | 未实现 | 中 |
| 申请后执行脚本 | 未实现 | 中 |
| 证书下载 | 未实现 | 中 |
| 手动 DNS 验证 | 未实现 | 低 |
| IP 证书 | 未实现 | 低 |

### 下一步计划
- 网站管理（Nginx 站点 CRUD）
- 数据库管理模块

---

## 2026-02-06 — Session #10：防火墙友好提示 + 终端 PTY 修复 + 监控模块增强

### 完成内容

#### 防火墙模块 — ufw 未安装时友好提示
- [x] 后端：所有 firewall service 方法（ListPortRules、ListIPRules、Operate、CreatePortRule、DeletePortRule、CreateIPRule、DeleteIPRule）增加 `isUFWInstalled()` 前置检查
- [x] 后端：新增 `isUFWInstalled()` 辅助函数，用 `which ufw` 检测
- [x] 前端：`firewall/index.vue` 的 `onMounted` 改为先 `await loadBase()`，确认 `baseInfo.isExist` 后才加载规则，避免触发 500 错误
- [x] 效果：未安装 ufw 时显示空状态 "未安装 (ufw)"，不再报 "服务器内部错误"

#### 终端模块 — /dev/ptmx 问题修复
- [x] 诊断：VM 上 `/dev/ptmx` 存在且权限正常（0666），Python PTY 测试通过
- [x] 根因：后端进程在 Cursor IDE 沙箱内启动，沙箱限制了 `/dev/ptmx` 设备访问
- [x] 修复：后端改为在沙箱外（`required_permissions: all`）启动，PTY 正常工作
- [x] 代码改进：`terminal.go` 增加 `/dev/ptmx` 存在性预检，PTY 失败时返回带 ANSI 颜色的中文错误提示（原因分析 + 解决建议）

#### 监控模块增强
- [x] 后端 DTO 新增：`SystemHostInfo`（主机名/OS/平台/内核/架构）、`NetIOStats`（每网卡实时速率）、`ProcessBrief`（Top 进程）、磁盘 inode 信息
- [x] 后端 Service：
  - 新增 `hostUtil.Info()` 获取系统基本信息
  - 网络从 `IOCounters(false)` 改为 `IOCounters(true)` 按网卡统计，增加速率计算（基于上次采样的差值/时间）
  - 新增 `getTopProcesses(n)` 获取 CPU 占用 Top N 进程
  - 磁盘增加 inode 使用率统计
- [x] 前端页面全面重构：
  - 新增系统信息卡片（3 列 Grid：主机名、操作系统、内核版本、系统架构、运行时间、CPU 型号）
  - 网络卡片改为显示每网卡实时上下行速率 + 累计流量
  - 新增 Top 进程表格（PID、进程名、CPU%、内存）
  - 磁盘表格增加 Inode 使用率列
  - 布局改为 Top 进程（左 10 列）+ 磁盘（右 14 列）并排

### 关键决策
- 网络速率使用服务端差值计算（而非前端），因为前端轮询间隔不稳定
- Top 进程固定显示 5 个，按 CPU 占用排序
- 不过滤 docker/bridge 网卡，让用户看到完整的网络接口信息

### 涉及文件
| 文件 | 变更 |
|------|------|
| `backend/app/service/firewall.go` | 所有方法增加 isUFWInstalled 检查 |
| `backend/app/api/v1/terminal.go` | PTY 友好错误提示 |
| `backend/app/dto/monitor.go` | 新增 SystemHostInfo/NetIOStats/ProcessBrief/磁盘 inode |
| `backend/app/service/monitor.go` | 系统信息/网络速率/Top 进程/inode |
| `frontend/src/views/host/firewall/index.vue` | onMounted 逻辑调整 |
| `frontend/src/views/host/monitor/index.vue` | 全面重构 |

### 遗留问题
- 监控历史数据存储 + ECharts 时间线图表（后续实现）
- GPU 监控（需要 nvidia-smi 或 ROCm，视需求）

### 下一步计划
- 网站管理（Nginx 站点 CRUD）
- 数据库管理模块
- 监控历史数据 + ECharts 图表

---

## 2026-02-06 — Session #9：Nginx 管理模块（前端）+ 网站菜单重构

### 完成内容

#### 环境修复
- [x] 修复 `node_modules` 符号链接损坏问题（重新 `npm install`）
- [x] Go 后端编译验证通过（零错误）

#### Nginx 管理前端页面
- [x] 前端 API 封装（`api/modules/nginx.ts`）：7 个方法（状态/操作/配置测试/安装/进度/卸载/依赖检查）
- [x] Nginx 管理页面（`views/website/nginx/index.vue`）完整实现：
  - 未安装状态：安装引导 + 依赖检查结果展示
  - 安装进度：实时轮询（2 秒间隔）+ 进度条 + 阶段标签
  - 已安装状态：四宫格信息卡片（运行状态/版本/PID/配置状态）
  - 操作按钮：启动/停止/重载/重新打开日志/优雅退出/配置测试/卸载
  - 详情面板：安装目录/版本/启动时间/PID
  - 配置测试输出：等宽字体 + 代码块展示

#### 侧边栏菜单重构
- [x] SSL 从独立菜单项改为「网站」二级菜单的子项
- [x] 新增「网站」二级展开菜单：Nginx 管理 + 证书管理
- [x] 路由注册新增 Nginx 页面路由

#### i18n 更新
- [x] 新增 `nginx.*` 共 35+ 翻译键（状态/操作/安装/依赖/配置测试/进度阶段）
- [x] 新增 `menu.nginx` / `menu.ssl` 菜单翻译

### 构建结果
| 检查项 | 结果 |
|--------|------|
| Go 编译 | ✅ 零错误 |
| Vite 生产构建 | ✅ 成功 (9.55s) |
| Linter | ✅ 零错误 |

### 新增前端文件
```
frontend/src/
├── api/modules/nginx.ts               # Nginx API 封装
├── views/website/nginx/index.vue      # Nginx 管理页面
```

### 修改前端文件
```
frontend/src/
├── layout/components/Sidebar.vue      # 侧边栏菜单重构（网站二级菜单）
├── routers/modules/website.ts         # 新增 Nginx 路由
├── i18n/zh.ts                         # 新增 nginx.* 翻译
```

### 关键决策
- Nginx 后端（Session #8 末尾已完成）：状态查询/操作/源码编译安装/卸载/依赖检查
- 前端安装进度采用 2 秒轮询 `getInstallProgress` API
- 侧边栏「网站」菜单从独立 SSL 链接升级为二级展开菜单

### 下一步
- [ ] 网站管理（Nginx 站点 CRUD：反向代理/静态站点/重定向）
- [ ] Nginx 配置解析器（参考 1Panel `agent/utils/nginx/`）
- [ ] 证书绑定到站点
- [ ] 数据库管理模块（MySQL/PostgreSQL）
- [ ] 容器管理模块（Docker）

---

## 2026-02-06 — Session #8：系统模块开发（监控/防火墙/进程/SSH/磁盘）

### 完成内容

#### Bug 修复
- [x] 修复 WebSocket 终端并发写入 panic（`concurrent write to websocket connection`）
  - 引入 `safeConn` 结构体统一加锁，stdout/stderr/心跳共享同一把 mutex
  - 本地终端和 SSH 远程终端均已修复

#### 新增 utils/cmd 工具包
- [x] `ExecWithOutput` — 带 30s 超时的命令执行，返回标准输出
- [x] `ExecWithTimeoutAndOutput` — 自定义超时
- [x] `Exec` — 不关心输出的简单执行

#### 系统监控模块
- [x] 后端：`gopsutil/v4` 实时获取 CPU/内存/负载/磁盘/网络/运行时间
- [x] 前端：Dashboard 仪表盘（四宫格概览 + 磁盘使用表格），5 秒自动刷新

#### 进程管理模块
- [x] 后端：列出所有进程（PID/名称/用户/CPU%/内存%/状态/命令行），支持过滤和排序
- [x] 后端：停止进程（SIGTERM/SIGKILL/SIGSTOP）
- [x] 后端：列出网络连接（TCP/UDP，进程名解析）
- [x] 前端：进程列表 + 网络连接两个 Tab，支持搜索/过滤/终止操作

#### SSH 管理模块
- [x] 后端：读取 `/etc/ssh/sshd_config` 解析配置（端口/Root 登录/密码认证/公钥认证/DNS）
- [x] 后端：修改 SSH 配置（白名单校验 + sshd -t 测试 + 失败回滚）
- [x] 后端：systemctl 控制 sshd 服务（start/stop/restart/enable/disable）
- [x] 后端：解析 SSH 登录日志（journalctl / auth.log），支持成功/失败过滤和分页
- [x] 前端：配置面板（Switch/Select 修改配置 + 服务控制按钮）+ 登录日志 Tab

#### 防火墙模块（Debian/ufw）
- [x] 后端：检测 ufw 安装状态和版本
- [x] 后端：启用/禁用/重载防火墙
- [x] 后端：解析端口规则（`ufw status numbered`），支持搜索和分页
- [x] 后端：创建/删除端口规则和 IP 规则
- [x] 前端：端口规则 + IP 规则两个 Tab，支持添加/删除/搜索

#### 磁盘管理模块
- [x] 后端：列出分区信息（设备/挂载点/文件系统/容量/inode），过滤虚拟文件系统
- [x] 前端：分区卡片列表，进度条显示使用率

#### 前端框架更新
- [x] 侧边栏「系统」菜单改为二级展开：文件/监控/防火墙/进程管理/SSH 管理/磁盘管理
- [x] 路由注册 6 个新页面
- [x] i18n 新增 monitor/process/sshManage/firewall/disk 5 组翻译

### 关键决策
- 防火墙只支持 ufw（Debian 系），不做 firewalld（CentOS/RHEL）适配
- SSH 配置修改先 `sshd -t` 测试，失败自动回滚原配置
- 监控采用轮询模式（5 秒），暂不使用 WebSocket 推送
- gopsutil 在 macOS 上也可工作，便于开发调试

### 下一步计划
- [ ] 监控模块增加历史数据存储和 ECharts 图表
- [ ] 防火墙增加转发规则管理
- [ ] 磁盘管理增加挂载/卸载操作
- [ ] 系统信息页面（OS、内核版本、主机名等）
- [ ] 创建文档 `docs/quick-start.md`（已完成）

---

## 2026-02-06 — Session #7：SSL 证书管理（ACME + DNS 验证 + 账户导入导出）

### 完成内容

#### 后端数据模型
- [x] `AcmeAccount` 模型：ACME 账户（邮箱/类型/密钥类型/私钥/CA URL/EAB 凭证）
- [x] `DnsAccount` 模型：DNS 账户（名称/类型/认证参数 JSON）
- [x] `Certificate` 模型：SSL 证书（主域名/附加域名/提供商/PEM/私钥/状态/到期时间）
- [x] 数据库迁移自动创建三张新表 + `SSLDir` 设置项

#### ACME 证书签发服务（lego 集成）
- [x] 基于 `go-acme/lego/v4` 实现 ACME 客户端封装
- [x] 支持 5 种 CA：Let's Encrypt / ZeroSSL / Buypass / Google Trust / 自定义 CA URL
- [x] 自动注册 ACME 账户并持久化私钥
- [x] 支持 EC (P256/P384) 和 RSA (2048/3072/4096) 密钥类型

#### DNS 提供商支持（7 家）
- [x] Cloudflare / 阿里云 DNS / DNSPod / 腾讯云 DNS / 华为云 DNS / NameSilo / GoDaddy
- [x] 通用 `DNSParam` 结构 + `GetDNSProvider` 工厂函数，易于扩展
- [x] `SupportedDNSProviders()` 返回提供商列表及所需字段，前端动态渲染表单

#### 证书管理完整流程
- [x] 创建证书 → 可选"立即申请" → 异步调用 ACME 签发
- [x] 手动上传证书（粘贴 PEM）自动解析域名和有效期
- [x] 证书续签（重新申请模式）
- [x] 证书文件存储：`{SSLDir}/certs/{domain}/fullchain.pem` + `privkey.pem`
- [x] 证书路径用户可配置（默认安装目录/ssl，Setting 中持久化）
- [x] 删除证书时同步清理文件系统

#### 账户导入导出
- [x] 导出：一键生成 JSON 文件（含 ACME 账户私钥 + DNS 账户凭证）
- [x] 导入：上传 JSON 文件批量创建账户，跳过失败项
- [x] 用途：多服务器部署时无需重复填写账户信息

#### 后端 API（22 个新端点）
- [x] 证书：search / create / update / upload / del / detail / apply / renew
- [x] ACME：list / create / del
- [x] DNS：list / create / update / del
- [x] 导入导出：export / import
- [x] SSL 设置：get dir / update dir / dns-providers

#### 前端 SSL 管理页面
- [x] 三 Tab 布局：证书列表 / ACME 账户 / DNS 账户
- [x] 证书列表：域名、状态徽章、到期日（30 天预警红色）、自动续签标识
- [x] 申请证书对话框：域名/ACME/DNS 账户选择/密钥类型/立即申请开关
- [x] 上传证书对话框：粘贴 PEM + 私钥
- [x] 证书详情对话框：基本信息 + PEM 内容展示 + 文件路径
- [x] ACME 账户注册对话框：邮箱/CA 类型/密钥类型
- [x] DNS 账户对话框：根据选择的提供商动态渲染认证字段
- [x] SSL 路径设置对话框
- [x] 导出下载 JSON / 导入上传 JSON

### 构建结果
| 检查项 | 结果 |
|--------|------|
| Go 编译 | ✅ 零错误 |
| TypeScript 检查 | ✅ 零错误 |
| Vite 生产构建 | ✅ 成功 (9.95s) |

### 新增后端文件
```
backend/
├── app/model/ssl.go                 # AcmeAccount / DnsAccount / Certificate 模型
├── app/repo/ssl.go                  # 三个 Repo
├── app/dto/ssl.go                   # SSL 相关 DTO
├── app/service/ssl.go               # 证书管理 Service（签发/续签/文件存储）
├── app/service/acme_account.go      # ACME/DNS 账户 Service + 导入导出
├── app/api/v1/ssl.go                # SSL API Handler（22 个方法）
├── utils/ssl/acme.go                # ACME 客户端封装（lego）
├── utils/ssl/dns_provider.go        # DNS 提供商配置工厂（7 家）
```

### 新增前端文件
```
frontend/src/
├── api/modules/ssl.ts               # SSL API 封装
├── views/website/ssl/index.vue      # SSL 管理页面（三 Tab）
├── routers/modules/website.ts       # 网站模块路由
```

### 关键技术决策
- lego v4.31.0 作为 ACME 客户端核心
- DNS 验证参数存储为 JSON 字符串，前端根据提供商动态生成表单
- 证书文件路径统一存储在用户可配置的 `SSLDir` 下
- 账户导出含私钥，导入时直接使用，无需重新注册 CA

### 下一步
- 网站管理模块（Nginx 配置解析器 + 站点 CRUD + SSL 绑定）
- 系统监控面板（CPU/内存/磁盘/网络实时图表）
- 证书自动续签定时任务（cron）
- 更多 DNS 提供商支持

---

## 2026-02-06 — Session #6：完整 SSH 客户端 + 主机管理 + 快速命令

### 完成内容

#### 后端数据模型 + CRUD
- [x] `Host` 模型：SSH 主机（名称/地址/端口/用户/认证方式/密码/私钥/描述/分组）
- [x] `Command` 模型：快速命令（名称/命令内容/分组）
- [x] `Group` 模型：通用分组（支持 host/command 两种类型）
- [x] Repo 层：Host/Command/Group 完整 CRUD + 分页 + 条件查询
- [x] 数据库迁移自动创建三张新表

#### 后端 SSH 连接服务
- [x] `HostService`：CRUD + 树形列表 + 测试连接 + SSH 拨号
- [x] SSH 支持密码认证和密钥认证（含 passphrase）
- [x] `TestHostConn`：不保存即可测试连接（表单直测）
- [x] `CommandService`：CRUD + 树形分组列表
- [x] `GroupService`：CRUD + 按类型过滤

#### 后端 API + 路由
- [x] 主机 API：7 个端点（create/update/del/search/tree/test/test-conn）
- [x] 命令 API：5 个端点（create/update/del/search/tree）
- [x] 分组 API：4 个端点（create/update/del/list）
- [x] 路由注册：所有新端点挂载到 JWT 认证路由组

#### 终端 WebSocket 增强
- [x] WS Handler 重构：支持本地 PTY 和远程 SSH 两种模式
- [x] 通过 `?id=hostID` 参数区分连接目标
- [x] SSH 终端：创建 SSH 客户端 → 新建 Session → 请求 PTY → 启动 Shell
- [x] SSH 输出通过 `wsWriter` 适配器转发到 WebSocket
- [x] SSH resize 支持：`session.WindowChange()` 响应终端尺寸变化
- [x] SSH 连接错误以红色 ANSI 提示显示在终端内

#### 前端 API 模块
- [x] `host.ts`：主机/命令/分组完整 API 封装（18 个方法）
- [x] i18n 更新：新增 host/command/group 共 40+ 翻译键

#### 前端终端页面重构
- [x] 三视图切换：终端 / 主机管理 / 快速命令（Radio 按钮组）
- [x] 终端左侧边栏：本地终端入口 + 远程主机树 + 快速命令列表
- [x] 终端标签增强：显示 SSH badge、区分本地/远程图标颜色
- [x] 批量输入：弹窗输入命令，同时发送到所有打开的终端
- [x] 从侧边栏点击远程主机直接打开 SSH 终端标签
- [x] 快速命令一键执行到当前活跃终端

#### 主机管理页面（完整 CRUD）
- [x] 主机列表表格：名称/地址/用户/认证方式/分组/描述
- [x] 搜索 + 分组过滤 + 分页
- [x] 新增/编辑主机对话框：支持密码和密钥两种认证
- [x] 连接测试（保存前和列表内均可测试）
- [x] 分组管理弹窗（增删分组）
- [x] 列表内直接连接按钮 → 打开 SSH 终端

#### 快速命令管理页面（完整 CRUD）
- [x] 命令卡片网格布局：名称 + 命令代码块
- [x] 一键执行/复制/编辑/删除
- [x] 搜索 + 分组过滤 + 分页
- [x] 新增/编辑命令对话框
- [x] 分组管理弹窗

### 构建结果
| 检查项 | 结果 |
|--------|------|
| Go 编译 | ✅ 零错误 |
| TypeScript 检查 | ✅ 零错误 |
| Vite 生产构建 | ✅ 成功 (8.94s) |
| Linter | ✅ 零错误 |

### 新增后端文件
```
backend/
├── app/model/host.go               # Host/Command/Group 模型
├── app/repo/host.go                # Host/Command/Group Repo（含 WithByGroupID/WithByType）
├── app/dto/host.go                 # Host/Command/Group DTO（请求/响应/搜索/树形）
├── app/service/host.go             # Host Service（CRUD + SSH 连接）
├── app/service/command.go          # Command Service（CRUD + 树形）
├── app/service/group.go            # Group Service（CRUD）
├── app/api/v1/host.go              # Host API Handler
├── app/api/v1/command.go           # Command API Handler
├── app/api/v1/group.go             # Group API Handler
```

### 修改后端文件
```
backend/
├── app/api/v1/entry.go             # 新增 HostAPI/CommandAPI/GroupAPI
├── app/api/v1/terminal.go          # 重写：支持本地 PTY + 远程 SSH
├── router/router.go                # 新增主机/命令/分组路由
├── init/migration/migration.go     # 新增三张表迁移
```

### 新增前端文件
```
frontend/src/
├── api/modules/host.ts             # 主机/命令/分组 API
├── views/terminal/host/index.vue   # 主机管理页面
├── views/terminal/command/index.vue # 快速命令页面
```

### 修改前端文件
```
frontend/src/
├── views/terminal/index.vue        # 终端主页重构（三视图 + 侧边栏）
├── routers/modules/terminal.ts     # 新增子路由
├── i18n/zh.ts                      # 新增 40+ 翻译键
```

### 关键技术决策
- SSH 连接复用 `golang.org/x/crypto/ssh`（go.mod 已有依赖）
- 终端 WS 通过 `?id=hostID` 区分本地/远程，无需新增 WS 端点
- SSH 输出用 `wsWriter` 适配器实现 `io.Writer` 接口
- 前端三视图用 `v-show`/`v-if` 切换，终端实例不销毁

### 下一步
- 网站管理模块（Nginx 配置解析器 + 站点 CRUD）
- 系统监控面板（CPU/内存/磁盘/网络实时图表）
- SSL 证书管理（ACME 自动签发）
- 数据库管理（MySQL/PostgreSQL）

---

## 2026-02-06 — Session #5：暗色科技风 UI + 文件管理 + Web 终端

### 完成内容

#### 暗色主题全面重构
- [x] 创建 `dark-theme.scss`：Element Plus 暗色变量全量覆盖 + 自定义 CSS 变量体系
- [x] 配色方案 "Cyber Dark"：深黑底色 `#0b0e14` + 青色主调 `#22d3ee` + 靛蓝辅助 `#818cf8`
- [x] Element Plus 组件微调：Card/Table/Input/Tag/Breadcrumb/Pagination/Dropdown 全部适配暗色
- [x] 登录页重设计：暗色网格背景 + 径向渐变光晕 + 毛玻璃卡片 + 青色发光边框
- [x] 初始化页匹配暗色风格
- [x] 侧边栏重写：深黑底色、青色渐变活跃指示器 + 左侧高亮条
- [x] 顶栏重写：半透明背景 + 模糊效果 + 渐变头像
- [x] 首页/设置/日志页全部适配暗色
- [x] `index.html` 添加 `class="dark"` + `main.ts` 引入 Element Plus dark CSS

#### 文件管理（后端 + 前端）
- [x] 后端 DTO：`FileInfo` / `FileSearchReq` / `FileCreateReq` / `FileDeleteReq` / `FileRenameReq` / `FileMoveReq` / `FileContentReq` / `FileSaveReq` / `FileModeReq` / `FileCompressReq` / `FileDecompressReq`
- [x] 后端 Service：列目录、读写文件内容、创建/删除/重命名/移动/复制、权限修改、压缩/解压
- [x] 后端 API：13 个端点（search/create/del/batch-del/rename/move/content/save/mode/compress/decompress/upload/download）
- [x] 后端安全：路径清理、系统目录保护（`/`/`/root`/`/home`）、文件大小限制（10MB）
- [x] 前端文件浏览器：路径输入框 + 面包屑导航 + 文件表格（名称/大小/权限/所有者/修改时间/操作）
- [x] 文件操作：双击进入目录、新建文件/目录、上传、下载、重命名、删除、批量删除
- [x] 文件编辑器弹窗：暗色代码编辑器（等宽字体、语法高亮背景）
- [x] 隐藏文件开关

#### Web 终端（后端 + 前端）
- [x] 后端依赖：`gorilla/websocket` + `creack/pty`
- [x] WebSocket 终端处理器：PTY 分配 + 双向数据转发 + 心跳 + resize 支持
- [x] JWT 中间件增强：支持 query 参数 `?token=` 传递（WebSocket 兼容）
- [x] JWT 前缀校验修复：仅在 token 以 `Bearer ` 开头时才去除前缀
- [x] 前端 xterm.js 集成：`@xterm/xterm` + `@xterm/addon-fit`
- [x] 多标签终端：新建/切换/关闭标签、自动 fit、窗口 resize 响应
- [x] 终端主题：匹配 Cyber Dark 配色（黑底/青色光标/语法色彩）

#### 路由与导航
- [x] 新增路由模块：`host.ts`（文件管理）、`terminal.ts`（终端）
- [x] 侧边栏菜单更新：首页 → 文件管理 → 终端 → 日志审计 → 面板设置
- [x] i18n 更新：新增 `file.*`/`terminal.*` 翻译键
- [x] Vite 代理配置：添加 `ws: true` 支持 WebSocket 代理

### 测试结果
| 场景 | 结果 |
|------|------|
| 登录页暗色科技风 | ✅ 毛玻璃卡片 + 网格背景 + 青色渐变按钮 |
| 主布局暗色主题 | ✅ 侧边栏/顶栏/内容区统一暗色 |
| 文件管理 - 浏览目录 | ✅ 根目录 16 项正确列出 |
| 文件管理 - 新建/删除/重命名 | ✅ 工具栏和操作按钮就绪 |
| Web 终端 - 连接 | ✅ WebSocket 通过 Vite 代理成功连接 |
| Web 终端 - 命令执行 | ✅ `ls` 输出正确显示 |
| Web 终端 - 多标签 | ✅ 标签栏 + 新增按钮 |
| TypeScript 检查 | ✅ 零错误 |
| Go 编译 | ✅ 零错误 |
| Vite 生产构建 | ✅ 成功 |

### 修复记录
- JWT 中间件 Bearer 前缀剥离 bug：未检查前缀直接截取导致 query token 被破坏
- Vite WebSocket 代理：需显式 `ws: true`
- `ElMessageBox.prompt` TypeScript 类型：返回值需用 `any` 类型处理

### 关键决策
- 暗色主题使用 CSS 变量（`--xp-*`）+ Element Plus dark 覆盖，无运行时主题切换
- 终端 WebSocket 协议：原始文本 I/O + `\x01` 前缀 resize 消息
- 文件下载通过 query 参数传递 JWT（避免 `<a>` 标签无法设置 Header）

### 新增后端文件
```
backend/
├── app/dto/file.go                  # 文件管理 DTO
├── app/service/file.go              # 文件管理 Service（13 个方法）
├── app/api/v1/file.go               # 文件管理 API Handler
├── app/api/v1/terminal.go           # WebSocket 终端 Handler
```

### 新增前端文件
```
frontend/src/
├── assets/styles/dark-theme.scss    # 暗色科技风主题
├── api/modules/file.ts              # 文件管理 API
├── views/host/file/index.vue        # 文件管理页面
├── views/terminal/index.vue         # Web 终端页面
├── routers/modules/host.ts          # 文件管理路由
├── routers/modules/terminal.ts      # 终端路由
```

### 下一步
- 网站管理模块（Nginx 配置解析器 + 站点 CRUD）
- 文件管理增强：拖拽上传、右键上下文菜单、文件图标分类
- 终端增强：连接断开重连、终端标题显示当前目录
- 系统监控面板（CPU/内存/磁盘/网络实时图表）

---

## 2026-02-06 — Session #4：前后端联调（Sprint 2 完成）

### 完成内容
- [x] 后端配置适配本地开发：`config.yaml` 改为相对路径（`./data/`），支持 macOS 开发
- [x] 前后端同时启动：后端 Go `:9999` + 前端 Vite `:5173`（proxy `/api` → 后端）
- [x] **初始化流程联调**：访问首页 → 检测未初始化 → 跳转初始化页 → 设置管理员 → 成功跳转登录页
- [x] **登录流程联调**：输入用户名密码 → 后端 bcrypt 校验 → JWT 返回 → 前端存储 → 跳转首页
- [x] **主布局验证**：深色侧边栏（可折叠）+ 面包屑顶栏 + 用户下拉菜单 + 内容区过渡动画
- [x] **首页验证**：系统信息卡片 + 快速入口导航
- [x] **登录日志联调**：分页表格正确显示登录记录（IP、浏览器、状态、时间），清空按钮可用
- [x] **面板设置联调**：从后端加载 PanelName/SessionTimeout，表单编辑 + 保存
- [x] 侧边栏折叠功能：展开 220px / 折叠 64px，动画平滑
- [x] 项目根目录 `.gitignore` 添加（排除 node_modules、dist、backend/data 等）
- [x] 浏览器全流程截图验证 ✅

### 联调测试结果
| 场景 | 结果 |
|------|------|
| 首次访问 → 自动跳初始化页 | ✅ 通过 |
| 初始化管理员 → 跳登录页 | ✅ 通过 |
| 登录 → JWT → 跳首页 | ✅ 通过 |
| 未登录访问私有路由 → 跳登录 | ✅ 通过 |
| 侧边栏导航 + 面包屑 | ✅ 通过 |
| 登录日志分页查询 | ✅ 通过 |
| 面板设置加载 + 保存 | ✅ 通过 |
| 侧边栏折叠/展开 | ✅ 通过 |

### 修复记录
- 后端 `config.yaml`：绝对路径 `/opt/xpanel/` → 相对路径 `./data/`，适配本地开发

### 下一步
- Sprint 3：网站管理模块开发（Nginx 配置解析器 + 站点 CRUD）
- 时间戳格式化（当前显示 ISO 格式原始字符串）
- 修改密码弹窗组件

---

## 2026-02-06 — Session #3：前端项目骨架搭建（Sprint 2 前端部分完成）

### 完成内容
- [x] 前端项目初始化：Vue 3 + Vite 6 + TypeScript 5.7
- [x] 核心依赖集成：Element Plus + Pinia + Vue Router + Axios + vue-i18n v11
- [x] Vite 配置：`@` 别名、代理 `/api` → `localhost:9999`
- [x] Axios 封装（`api/http.ts`）：JWT 自动附加、响应拦截（code=0 成功）、401 跳登录、错误提示
- [x] API 模块：auth（登录/初始化/改密）、setting（获取/更新）、log（分页/清空）
- [x] Vue Router：路由守卫（未认证跳登录）、模块化路由（home/log/setting）
- [x] Pinia Store：global（面板状态、侧边栏折叠）、user（token/用户名，持久化）
- [x] i18n 国际化：仅中文，vue-i18n Composition API 模式，70+ 翻译键
- [x] 主布局：深色侧边栏（可折叠）+ 白色顶栏（面包屑+用户菜单）+ 内容区
- [x] 登录页：渐变背景、居中卡片、自动检测初始化状态
- [x] 初始化页：管理员账户设置表单（用户名+密码+确认密码+校验）
- [x] 首页：系统信息概览 + 快速入口
- [x] 面板设置页：面板名称 + 会话超时配置
- [x] 日志页×2：登录日志 / 操作日志（分页表格 + 清空功能）
- [x] 全局样式：滚动条美化、Element Plus 微调
- [x] TypeScript 类型检查通过 ✅
- [x] Vite 生产构建通过 ✅

### 前端项目文件清单（35 个源文件）
```
frontend/
├── package.json / vite.config.ts / tsconfig*.json / index.html
├── src/
│   ├── main.ts / App.vue / vite-env.d.ts
│   ├── api/
│   │   ├── http.ts                    # Axios 实例 + 拦截器
│   │   └── modules/{auth,setting,log}.ts
│   ├── routers/
│   │   ├── index.ts / guard.ts
│   │   └── modules/{home,log,setting}.ts
│   ├── store/
│   │   ├── index.ts
│   │   └── modules/{global,user}.ts
│   ├── i18n/
│   │   ├── index.ts                   # vue-i18n 初始化（仅中文）
│   │   └── zh.ts                      # 中文翻译（70+ key）
│   ├── layout/
│   │   ├── index.vue                  # 主布局
│   │   └── components/{Sidebar,Header,AppMain}.vue
│   ├── views/
│   │   ├── login/index.vue            # 登录页
│   │   ├── init/index.vue             # 初始化页
│   │   ├── home/index.vue             # 首页
│   │   ├── setting/index.vue          # 面板设置
│   │   └── log/{login,operation}/index.vue  # 日志页
│   └── assets/styles/{index,variables}.scss
```

### 关键决策
- **vue-i18n 升级到 v11**：v10 已弃用，v11 API 兼容 Composition API
- **pinia-plugin-persistedstate v3**：v4 需要 pinia 3.x，当前使用 pinia 2.x
- **后端 API 适配**：成功码为 `code: 0`（非 200），业务错误返回 HTTP 200 + `code: 500`
- **登录字段**：后端用 `name` 而非 `username`，前端已对齐
- **i18n 仅中文**：Element Plus 中文语言包 + vue-i18n 仅 zh locale
- **侧边栏菜单**：独立定义（非路由派生），当前仅展示已实现模块

### 修复记录
- `pinia-plugin-persistedstate` v3 使用 `paths` 而非 v4 的 `pick` → 修正 store 配置

### 下一步
- 前后端联调：启动后端 + 前端 dev server，测试完整登录流程
- 修改密码弹窗组件
- 后续模块开发（网站管理 / Nginx 配置解析器）

---

## 2026-02-06 — Session #2：后端骨架搭建（Sprint 1 完成）

### 完成内容
- [x] Go module 初始化（`xpanel`）+ 完整目录结构创建
- [x] 全局变量 + Viper 配置加载（`global/`、`init/viper/`、`configs/config.yaml`）
- [x] 日志模块（Logrus，文件+控制台双输出）（`init/log/`）
- [x] 数据库连接（GORM + glebarez/sqlite）（`init/db/`）+ `BaseModel`
- [x] 数据库迁移 + 默认设置初始化（`init/migration/`）
- [x] 统一响应结构 `dto.Response` / `dto.PageResult` + 分页 DTO
- [x] 业务错误包 `buserr`，支持 i18n 错误消息
- [x] i18n 国际化框架（go-i18n + embed，中/英双语）
- [x] 工具包：bcrypt 密码哈希（`utils/encrypt/`）、JWT 生成/解析（`utils/jwt/`）
- [x] Repo 层：`ISettingRepo`、`ILogRepo` + 通用 `DBOption` 模式
- [x] Service 层：`IAuthService`（登录/初始化/改密）、`ISettingService`（CRUD）、`ILogService`（分页/清空）
- [x] API 层：`AuthAPI`、`SettingAPI`、`LogAPI` + `helper` 统一响应工具
- [x] 中间件：CORS（gin-contrib/cors）、JWT 认证、操作日志（异步写入+脱敏）
- [x] 路由注册：公开路由（登录/初始化）+ 私有路由（JWT 保护）
- [x] 初始化链串联：Viper → Logger → DB → Migration → i18n → Router → HTTP Server
- [x] 全量编译通过：`go build ./...` ✅

### 项目文件清单（28 个 Go 源文件）
```
backend/
├── cmd/server/main.go          # 入口
├── server/server.go            # 初始化链 + HTTP 启动
├── global/global.go            # 全局变量 + 配置结构体
├── constant/{constant,errs}.go # 常量 + 错误码
├── configs/config.yaml         # 默认配置
├── init/{viper,log,db,migration}/ # 初始化模块
├── i18n/{i18n.go,lang/{zh,en}.yaml} # 国际化
├── buserr/errors.go            # 业务错误
├── utils/{encrypt,jwt}/        # 工具包
├── app/model/{base,setting,log}.go # 数据模型
├── app/dto/{common,auth,setting}.go # DTO
├── app/repo/{common,setting,log}.go # 仓库层
├── app/service/{auth,setting,log}.go # 服务层
├── app/api/v1/{entry,auth,setting,log}.go # API 处理器
├── app/api/v1/helper/helper.go  # 响应工具
├── middleware/{cors,jwt_auth,operation_log}.go # 中间件
└── router/router.go            # 路由注册
```

### API 路由表
| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/api/v1/auth/setting` | 获取登录页设置 | ✗ |
| GET | `/api/v1/auth/is-init` | 检查是否已初始化 | ✗ |
| POST | `/api/v1/auth/init` | 初始化用户 | ✗ |
| POST | `/api/v1/auth/login` | 用户登录 | ✗ |
| POST | `/api/v1/auth/logout` | 退出登录 | ✓ |
| POST | `/api/v1/auth/password` | 修改密码 | ✓ |
| GET | `/api/v1/settings` | 获取面板设置 | ✓ |
| POST | `/api/v1/settings/update` | 更新设置 | ✓ |
| POST | `/api/v1/logs/login` | 登录日志分页 | ✓ |
| POST | `/api/v1/logs/operation` | 操作日志分页 | ✓ |
| POST | `/api/v1/logs/login/clean` | 清空登录日志 | ✓ |
| POST | `/api/v1/logs/operation/clean` | 清空操作日志 | ✓ |

### 修复记录
- CORS 中间件：`AllowOrigins: ["*"]` + `AllowCredentials: true` 互斥 → 改为 `AllowAllOrigins: true` + `AllowCredentials: false`
- `service/setting.go` 重复导入 `repo` 包（`settingRepo` 已在 `auth.go` 中定义）→ 移除冗余导入
- 操作日志中间件 `maskSensitiveFields` JSON 值替换逻辑修正

### 决策记录
- 密码存储使用 bcrypt（非 RSA 加密），符合最佳安全实践
- Setting 采用 Key-Value 模式，比独立字段更灵活
- JWT 无状态设计，暂不实现 Token 黑名单
- 操作日志异步写入，不阻塞请求
- 仅记录写操作（POST/PUT/DELETE），GET 不记录

### 下一步
- Sprint 2：搭建前端项目骨架（Vue 3 + Vite + Element Plus + Pinia）
- 前后端联调：登录流程 → 面板设置页 → 日志查看

---

## 2026-02-06 — Session #1：项目规划

### 完成内容
- [x] 需求分析：对照 1Panel 源码，梳理 X-Panel 需要复刻的 15 个功能模块
- [x] 开发指导文档：编写 `docs/development-guide.md`（2378 行），涵盖架构设计、数据模型、API 规范、前端架构、开发计划
- [x] Cursor 规则配置：
  - `.cursor/rules/x-panel.mdc` — 全局规则（alwaysApply）
  - `.cursor/rules/backend.mdc` — Go 后端规则（backend/** 生效）
  - `.cursor/rules/frontend.mdc` — Vue 前端规则（frontend/** 生效）
- [x] 工作日志机制：建立 `docs/worklog.md`，规则中加入日志更新约束

### 决策记录
- Nginx 采用本地安装管理，不走 Docker
- 初期单体架构，后续拆分 Core + Agent
- 不实现应用商店、Runtime 管理、系统快照、备份账号管理
- 开发顺序：后端骨架 → 前端骨架 → 登录认证 → 联调

### 下一步
- Sprint 1：搭建后端项目骨架（Go module、目录结构、配置加载、DB 连接、统一响应）
