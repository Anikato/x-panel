# Dashboard、Agent 与 X-Panel 的关系

本文说明三个组件在当前产品中的职责、通信和发布关系。它描述的是已经实现并需要长期保持的架构边界；正式发布操作以根目录的 [`RELEASE.md`](../RELEASE.md) 为唯一手册。

## 组件职责

| 组件 | 主要职责 | 不负责 |
| --- | --- | --- |
| X-Panel | 安装和配置 Agent、管理 `xpanel-nezha-agent.service`、展示本机状态、随 X-Panel 发布包升级绑定 Agent | 节点监控协议、中心任务调度、第二套云控通道 |
| Agent | 独立采集节点状态、连接 Dashboard、执行经允许的远程任务、保存 UUID 和运行配置 | X-Panel 业务功能和集中管理界面 |
| Dashboard | 节点监控、远程命令、批量升级 X-Panel、任务状态和版本展示 | 直接覆盖节点上的 X-Panel 或 Agent 文件 |

三者的运行关系如下：

```text
Dashboard  <──── Agent 协议 ────>  xpanel-nezha-agent.service
                                             │
                                             │ 读取面板名称、执行固定命令
                                             ▼
                                      /opt/xpanel/xpanel
```

Agent 是独立的 systemd 服务，不嵌入 X-Panel 主进程。X-Panel 重启不应终止已经运行的 Agent；Dashboard 是唯一的中心控制面。

## 配置与节点身份

- X-Panel 把 Dashboard 地址、AgentSecret、面板名称和节点角色写入 `/opt/xpanel/nezha-agent/config.yml`。
- `config.yml` 是 Agent 运行配置的事实来源。X-Panel 修改配置时必须保留 UUID、未知字段和 Dashboard 写入的新字段。
- X-Panel 安装包与面板保存配置时写入 `node_role: xpanel`。事务升级只合并这一字段，不整文件替换，也不从发布包套用 `config.yml`。未配置过 Agent 的节点不会因此生成配置文件。
- Agent 建立连接时通过元数据上报 `node_role` 和 X-Panel 面板名称。Dashboard 只把 `node_role == "xpanel"` 的节点列入 X-Panel 页并允许批量升级；普通节点省略该字段。`openwrt` 由后续独立打包写入，不在本产品路径处理。
- Dashboard 仅在自动创建新节点时把面板名称用作节点名称；已有节点不会被自动重命名。
- 新自动注册节点默认设置为对游客隐藏。
- AgentSecret 不得写入命令参数、systemd unit 或普通日志。X-Panel 的管理接口不回显明文。

## X-Panel 批量升级

Dashboard 不传入任意升级脚本，而是让选中节点执行固定入口：

```bash
/opt/xpanel/xpanel update --latest
```

实际执行由 `systemd-run --no-block` 启动为一次性后台服务。因此，Dashboard 到 Agent 的初次调用超时不会限制完整升级只能运行 30 秒。Dashboard 会继续查询 systemd 任务状态并刷新：

- 已接受
- 升级中
- 成功
- 失败
- 离线
- 待确认

自动跟踪目前最长约 10 分钟；超过时间仍无法确认时标记为“待确认”，可人工重新检查。命令已下发不等于升级成功，最终状态以任务结果和 `/opt/xpanel/xpanel --version` 为准。

当前不会持续自动轮询所有节点的 X-Panel 版本。管理员在 Dashboard 中使用“刷新版本”后，Agent 执行 `xpanel --version` 并上报结果。

## 登录会话

Dashboard 的默认登录有效期为 720 小时。会话保存在服务端，并校验浏览器 User-Agent：

- 同一浏览器和 User-Agent 下更换 IP，不要求重新登录；服务端只更新会话记录中的 IP。
- 更换浏览器、User-Agent 发生变化、退出登录、会话被撤销或到期后，需要重新登录。

这就是当前“记住我”的简化实现，不额外保存长期密码或独立刷新令牌。

## Agent 与 X-Panel 的发布关系

Agent 源码和 X-Panel 源码可以分别开发，但节点只从 `https://xpanel.qm.mk` 获取 X-Panel 更新。正式流程为：

1. Agent 有改动时，先在 GitHub 发布一个不可变、可追溯的 Agent 版本，同时提供 amd64、arm64 和 `checksums.txt`。
2. X-Panel 发布工作流用 `CUSTOM_AGENT_REPO` 与 `CUSTOM_AGENT_VERSION` 下载并校验该版本。
3. 两个架构的 Agent 被打入对应的 X-Panel 发布包。
4. X-Panel 发布工作流将包和 `releases/latest.json` 发布到 `xpanel.qm.mk`。
5. 节点通过 `xpanel update --latest` 同时获得经过绑定验证的 X-Panel 和 Agent。

即使只有 Agent 发生变化，也必须发布一个新的 X-Panel 补丁版本。当前没有独立的 Agent 在线更新清单，也不允许 Dashboard 绕过 X-Panel 的事务升级直接覆盖 Agent。

具体变量、Secret、失败重试和验收步骤见 [`RELEASE.md`](../RELEASE.md)。

## 当前实现范围

| 需求 | 当前状态 |
| --- | --- |
| 可见界面移除“哪吒/Nezha”品牌 | 已实现；源码协议名、目录名和技术文档可保留兼容标识 |
| amd64 与 arm64 定制 Agent | 已实现 |
| 新节点默认不向游客展示 | 已实现 |
| 新节点默认采用 X-Panel 面板名称 | 已实现，仅影响首次自动注册 |
| Agent 声明 `node_role: xpanel` | 已实现；安装、配置保存和升级只合并该字段 |
| Dashboard 按节点角色过滤 X-Panel 页 | 已实现于 Dashboard / 管理后台；X-Panel 只负责写入角色 |
| Dashboard 批量触发 X-Panel 更新 | 已实现，固定命令并记录任务状态 |
| Dashboard 查看 X-Panel 版本 | 已实现，当前需要手动刷新 |
| 长登录会话与换 IP 保持登录 | 已实现 |
| Dashboard 查看 X-Panel 网站和证书剩余时间 | 已实现（只读快照，手动刷新）；见 [`docs/superpowers/specs/2026-08-19-xpanel-sites-snapshot-design.md`](superpowers/specs/2026-08-19-xpanel-sites-snapshot-design.md) |
| Dashboard 独立升级 Agent | 不实现；使用新的 X-Panel 补丁版本统一升级 |

## 修改边界

- 不恢复 Fleet Center、Fleet Reporter 或 Fleet v2。
- 不在 X-Panel 内实现 Agent 协议客户端、采集器或第二套中心控制面。
- 不恢复 Agent 自动更新或 Dashboard 强制覆盖 Agent。
- 协议或配置字段变更必须说明旧 Dashboard、旧 Agent、旧 X-Panel 的兼容或明确拒绝行为。
- `node_role`：旧 Agent 忽略未知字段；旧 X-Panel 不写入该字段。Dashboard 在握手未声明时保留库中角色。过滤打开后，尚未带上 `node_role: xpanel` 的节点会暂时离开 X-Panel 页，直到升级或保存配置后重连。
- 跨仓库开发前先阅读 X-Panel 根目录的 `AGENTS.md`、`RELEASE.md` 和本文。

相关实现入口：

- X-Panel Agent 管理：`backend/app/service/nezha_agent.go`
- X-Panel Agent 配置合并：`backend/app/service/nezha_agent_config.go`
- X-Panel 事务升级：`backend/app/service/component_upgrade.go`
- X-Panel 只读能力入口：`backend/cmd/server/invoke.go`（`sites.snapshot`）
- Dashboard X-Panel 升级任务：`Nezha-Server/nezha/cmd/dashboard/controller/xpanel.go`
- Dashboard 网站/证书快照：`Nezha-Server/nezha/cmd/dashboard/controller/xpanel_inventory.go`、`Nezha-Server/admin-frontend/src/routes/xpanel-sites.tsx`
- Dashboard 新节点默认策略与角色：`Nezha-Server/nezha/service/rpc/auth.go`
- Agent 握手元数据（`xpanel_name`、`node_role`）：`Nezha-Server/agent/model/auth.go`
