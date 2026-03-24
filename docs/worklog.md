# X-Panel 工作日志

> 记录每次开发会话的工作内容，便于追踪项目进展和上下文衔接。

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
