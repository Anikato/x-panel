# X-Panel 工作日志

> 记录每次开发会话的工作内容，便于追踪项目进展和上下文衔接。

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
