# X-Panel 工作日志

## 2026-09-24 晚

### 完成内容

- [x] 源码准备 X-Panel `v0.8.3-20260924.1`：修复申请证书时确认阿里云域名触发的空指针崩溃
- [ ] GitHub `Build & Release` 打出 `v0.8.3-20260924.1` 正式包

### 关键决策

- 阿里云列域名必须传入 `RuntimeOptions`，不能传空指针
- 申请或续签协程里的 panic 只把该证书标为失败，不再打垮面板进程

## 2026-09-24

### 完成内容

- [x] 源码准备 X-Panel `v0.8.3-20260924`：阿里云申请先确认域名归属，公共 DNS 改用国内可通的解析，进程中断后不再停在申请中
- [ ] GitHub `Build & Release` 打出 `v0.8.3-20260924` 正式包（linux/amd64、linux/arm64）

### 关键决策

- 递归 DNS 使用 `223.5.5.5`、`119.29.29.29`，`8.8.8.8` 放在后面；单次查询超时 3 秒
- 只有旧 TXT 未清理干净时才等待后重试，其他失败直接结束
- Agent 仍沿用已发布的 `agent-v2.3.4-xpanel.1`

### 下一步计划

- 构建完成后在阿里云机器执行 `xpanel update --latest`，再申请 `*.hiny.cn`

## 2026-09-23

### 完成内容

- [x] 源码准备 X-Panel `v0.8.3-20260923`：证书使用者与失效替换、网站多域名、日志分析、防火墙编号、网卡协商速率、任务速率与通知清除
- [ ] GitHub `Build & Release` 打出 `v0.8.3-20260923` 正式包（linux/amd64、linux/arm64）

### 关键决策

- 版本号用 `v0.8.3-20260923`，工作流只允许后缀一段字母数字或点，不用 `v0.8.3-2026-09-23`
- 虚拟网卡没有协商速率时标明「不提供」，不用「未协商」冒充掉线
- Agent 仍沿用已发布的 `agent-v2.3.4-xpanel.1`

### 下一步计划

- `Build & Release` 成功后，节点执行 `xpanel update --latest`

## 2026-09-19

### 完成内容

- [x] 源码打上 X-Panel `v0.8.3`：主机网卡页、服务 Unit 预览、侧栏与主题、列表页表面统一
- [ ] GitHub Release / `Build & Release` 尚未跑（linux/amd64、linux/arm64）

### 关键决策

- 列表页禁止卡套表，摘要用 deck，与首页同一套表面；设置页分类卡片保留
- 内置主题拉开色相与结构（轨式侧栏、纸纹底、展示级标题），不靠换强调色凑数

### 下一步计划

- 走 `Build & Release` 的 `workflow_dispatch` 打出正式包后，节点 `xpanel update --latest`

## 2026-09-12

### 完成内容

- [x] 发布 X-Panel `v0.8.2`（主题包导入导出、四种表面配方、自定义字体、亮暗色编辑器工作区；linux/amd64、linux/arm64，公网 SHA256 通过）
- [x] Agent 仍为 `agent-v2.3.4-xpanel.1`，未换制品

### 关键决策

- 正式发布只走 `Build & Release` 的 `workflow_dispatch`，源码先推到 `3c20ccb`
- 公开主题 schema 仍未冻结；本版是产品补丁，不是契约冻结

### 下一步计划

- 节点执行 `xpanel update --latest` 升级到 `v0.8.2`

## 2026-09-11

### 完成内容

- [x] 发布 Agent `agent-v2.3.4-xpanel.1`（对齐官方 v2.3.4，保留 `xpanel_name` / `node_role` 握手）
- [x] 源码推到 `Anikato/nezha-agent` `012570b`，制品在 `Anikato/x-panel` Release
- [x] 绑定仓库变量 `CUSTOM_AGENT_VERSION=agent-v2.3.4-xpanel.1`
- [x] 发布 X-Panel `v0.8.0`（linux/amd64、linux/arm64，公网 SHA256 通过）
- [x] 发布 X-Panel `v0.8.1`（首页仪表与外观默认、样式导入导出；linux/amd64、linux/arm64，公网 SHA256 通过）

### 关键决策

- Agent 必须先有不可变 GitHub Release，X-Panel 工作流才能打包；本地二进制不能直接进正式包
- 版本号用 `v0.8.0`，不跳到 `v8.0.0`

### 下一步计划

- 节点执行 `xpanel update --latest` 升级到 `v0.8.1`，Agent 仍为 v2.3.4-xpanel.1

## 2026-08-19

### 完成内容

- [x] 写完 Dashboard 只读网站/证书快照设计：`docs/superpowers/specs/2026-08-19-xpanel-sites-snapshot-design.md`
- [x] 写完实施计划：`docs/superpowers/plans/2026-08-19-xpanel-sites-snapshot.md`
- [x] X-Panel 增加 `xpanel invoke`，第一期只开放只读 `sites.snapshot`；未知子命令不再误入 `Start()`
- [x] Dashboard 新表 `xpanel_site_snapshots`、独立并发槽、API `GET/POST /api/v1/xpanel/sites*`、页面 `/dashboard/xpanel/sites`
- [x] 本机验证：X-Panel `go test ./cmd/server ./app/service`、Dashboard controller/model/singleton、admin-frontend 相关 Vitest 与 `tsc -b` 均通过
- [x] 发布 X-Panel `v0.7.88`（linux/amd64、linux/arm64，公网 SHA256 通过）
- [x] Dashboard `v2.3.2-xpanel.2` 二进制已部署到 Tencent-Swift `/data/nezha-dashboard`

### 关键决策

- 第一期只读；主节点发证、其余同步，不做远程续期
- 按节点查看，按剩余天数排序；同一域名多节点多行
- 节点侧用稳定 `xpanel invoke` 能力信封，运输层沿用现有 Exec
- 与现网 Dashboard 隔离：不改 Agent/protobuf/`servers` 表/升级 API；快照用新表、新路由、独立并发槽
- 打开网站页只读缓存，刷新由管理员手动触发

### 遗留问题

- 旧节点没有 `invoke` 时会误走 `Start()`（常见端口占用），刷新显示为 failed，需先升级 X-Panel

### 下一步计划

- 找机器升级到 `v0.7.88` 后，在 Dashboard「网站与证书」页手动刷新验证
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
