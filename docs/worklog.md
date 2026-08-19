# X-Panel 工作日志

## 2026-08-19

### 完成内容

- [x] 写完 Dashboard 只读网站/证书快照设计：`docs/superpowers/specs/2026-08-19-xpanel-sites-snapshot-design.md`
- [x] 写完实施计划：`docs/superpowers/plans/2026-08-19-xpanel-sites-snapshot.md`
- [x] X-Panel 增加 `xpanel invoke`，第一期只开放只读 `sites.snapshot`；未知子命令不再误入 `Start()`
- [x] Dashboard 新表 `xpanel_site_snapshots`、独立并发槽、API `GET/POST /api/v1/xpanel/sites*`、页面 `/dashboard/xpanel/sites`
- [x] 本机验证：X-Panel `go test ./cmd/server ./app/service`、Dashboard controller/model/singleton、admin-frontend 相关 Vitest 与 `tsc -b` 均通过

### 关键决策

- 第一期只读；主节点发证、其余同步，不做远程续期
- 按节点查看，按剩余天数排序；同一域名多节点多行
- 节点侧用稳定 `xpanel invoke` 能力信封，运输层沿用现有 Exec
- 与现网 Dashboard 隔离：不改 Agent/protobuf/`servers` 表/升级 API；快照用新表、新路由、独立并发槽
- 打开网站页只读缓存，刷新由管理员手动触发

### 遗留问题

- 旧节点没有 `invoke` 时会误走 `Start()`（常见端口占用），刷新显示为 failed，需先升级 X-Panel

### 下一步计划

- 走 `Build & Release` workflow_dispatch 正式发布；Dashboard 二进制部署到 Tencent-Swift
- 不发新 Agent

## 2026-08-18

### 完成内容

- [x] 容器编排：把一份 docker-compose 当项目管理（创建 YAML、挂载已有路径、整栈启停、更新镜像、编辑 YAML）
- [x] 计划任务新增 `compose` 类型，支持定时 pull / pull+up
- [x] 备份校验和、SFTP 校验、备份 hook、目录任务前后命令一并纳入

- [x] 发布 X-Panel `v0.7.87`（linux/amd64、linux/arm64，公网 SHA256 通过）

## 2026-08-17

### 完成内容

- [x] 通知降噪：去掉 logrus Error hook 和「任意写接口失败」通知
- [x] 事件名统一小写；计划任务失败能命中偏好；安静事件的 `show_badge=false` 能写入 SQLite
- [x] 默认规则允许全关，不再被零值重置
- [x] 登录失败、证书自动续签失败改为显式事件；成功类任务默认不打红点
- [x] 捆绑 Agent 安装跳过 tar 包根目录 `./`，避免 `invalid archive entry name`
- [x] 前端：xlsx 按需加载；监控/日志分析改 `echarts/core`；文件管理与终端 `keep-alive`
- [x] 侧栏不再重复拉版本号；HTTP 拦截器文案走 i18n
- [x] 首页 hero 改成仪表盘密度；默认字体改为系统栈，不再假装加载了 Inter
- [x] 发布 `agent-v2.3.1-xpanel.2`（握手带 `node_role`）
- [x] 绑定后发布 X-Panel `v0.7.86`

### 关键决策

- 先修站内 inbox，不加邮件/Webhook
- 旧的 `operation.failed` / `system.log.error` 记录仍可显示，但不再产生、也不再出现在默认偏好里
- 前端视觉走工业仪表盘，不换字体、不拆超大页面、不改全局图标注册

### 遗留问题

- 仍是 30 秒轮询，没有站外通道和保留策略
- 登录失败每次尝试都会写一条通知，暴力破解时可能刷屏
- 证书/网站配置/文件管理仍是超大 SFC；页面里还有硬编码中文 ElMessage

### 下一步计划

- 节点升级到 v0.7.86 并重连后，Dashboard X-Panel 页才会出现 `node_role=xpanel` 节点
- 需要时再加 WebSocket 推送和独立 Alert 通道

## 2026-08-13

### 完成内容

- [x] 对齐三角色设计：X-Panel 安装、配置保存和升级写入 `node_role: xpanel`
- [x] 事务升级只合并该字段，保留 UUID、密钥和未知字段；缺配置不造文件；损坏 YAML 不阻断升级
- [x] 安装脚本首次配置写入 `node_role`，已有 `config.yml` 升级时同样只合并该字段
- [x] 更新现行文档：`docs/dashboard-agent-xpanel.md`、`docs/nezha-agent.md`、`XPANEL_COLLABORATION.md`

### 关键决策

- 升级不整文件替换 `config.yml`，也不从发布包套用配置
- OpenWrt 打包与 `node_role: openwrt` 仍不在本产品路径处理
- 本次不发 Agent 标签、不发 X-Panel 版本

### 遗留问题

- 认识 `node_role` 的 Agent 仍在本地工作区，生产节点要等先发 Agent、再发 X-Panel 补丁后才会带上角色

### 下一步计划

- 按 `RELEASE.md` 先发带该字段的 Agent，再发绑定该 Agent 的 X-Panel 补丁
