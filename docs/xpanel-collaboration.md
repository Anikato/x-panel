> 本文件与开发目录上级的 `XPANEL_COLLABORATION.md` 相同。文中的 `x-panel/`、`agent/`、`nezha/`、`admin-frontend/` 是并列目录。现行地址见 [repository-map.md](repository-map.md)。

# Nezha 与 X-Panel 协同开发说明

- 更新日期：2026-10-26
- 适用范围：本目录中的 `x-panel/`、`agent/`、`nezha/`、`admin-frontend/`
- 目标读者：后续参与二次开发的 AI 和开发者
- 现行仓库地址见 `x-panel/docs/repository-map.md`

## 1. 文档目的

本目录后续对 Nezha Dashboard 和 Nezha Agent 的二次开发，主要目标是与 X-Panel 协同，而不是建设一套独立于 X-Panel 的 Fleet 控制体系。

开始开发前，必须先阅读 X-Panel 的当前项目规则、发布手册和组件关系说明：

- `x-panel/AGENTS.md`
- `x-panel/RELEASE.md`
- `x-panel/docs/dashboard-agent-xpanel.md`

`docs/superpowers/` 下的旧设计和计划只作为历史记录。如果本说明与上述当前文档或实现冲突，以当前文档和经过验证的最新实现为准，并在修改前明确指出冲突。

## 2. 仓库角色

| 目录 | 角色 |
| --- | --- |
| `x-panel/` | X-Panel 产品、安装升级、Agent 配置入口和 systemd 生命周期管理 |
| `agent/` | 独立运行的定制兼容 Agent 及后续协议扩展 |
| `nezha/` | Nezha Dashboard，唯一的节点中心控制面 |
| `admin-frontend/` | Dashboard 管理后台前端（含 X-Panel 页）；构建产物同步到 `nezha/cmd/dashboard/admin-dist` |

三者的关系是：X-Panel 管理组件，Agent 负责节点能力，Dashboard 负责中心控制。

## 3. 不可破坏的架构边界

1. Nezha Agent 必须作为独立 systemd 服务运行，不得把 Agent 的采集、通信或任务逻辑揉进 X-Panel 主进程。
2. Dashboard 是节点监控、远程命令、终端、任务和远程配置的唯一中心控制面。
3. X-Panel 只负责 Agent 的安装、配置入口、凭证处理、服务生命周期、状态展示和随 X-Panel 绑定升级。
4. 不恢复 Fleet Center、Fleet Reporter 或 Fleet v2，也不新增功能相同的第二套控制协议。
5. X-Panel 主进程退出或重启时，已运行的 Agent 不应被连带终止。
6. Agent 的自动更新和 Dashboard 强制更新默认关闭；Agent 版本由 X-Panel 发布包固定并随 X-Panel 升级。

## 4. 配置与服务契约

默认部署布局：

```text
/opt/xpanel/
├── xpanel
└── nezha-agent/
    ├── nezha-agent
    └── config.yml

/etc/systemd/system/
├── xpanel.service
└── xpanel-nezha-agent.service
```

必须保持以下契约：

- `config.yml` 是 Agent 运行配置的权威来源。
- Agent UUID 由 Agent 生成并保存在 `config.yml`，不得由 X-Panel 重建或替换。
- Dashboard 通过 ApplyConfig 或 Server Transfer 写入的新配置和轮换后的密钥必须被保留。
- X-Panel 修改配置时只合并用户明确修改的字段，保留 UUID、未知字段和 Dashboard 写入的其他字段。
- X-Panel 安装包、配置保存和升级写入 `node_role: xpanel`。升级只合并这一字段，不得整文件替换 `config.yml`，也不得从发布包套用配置。未配置的节点不得因此生成配置文件。
- 节点角色由 Agent 配置显式声明，Dashboard 不根据操作系统或 `xpanel_version` 猜测。普通节点省略该字段；`openwrt` 由后续独立打包写入。
- DB 中的地址或密钥只能作为加密镜像，不得在启动或同步时覆盖较新的文件配置。
- AgentSecret 不得出现在命令行参数、systemd unit 或普通日志中。
- `config.yml` 应保持 `0600`，Agent 目录应保持 `0700`。
- 未配置 Dashboard 地址和 AgentSecret 时，Agent 不启动、不联网。

## 5. 二次开发原则

### 5.1 修改 Agent 时

- 优先保持与上游 Dashboard 的协议兼容，避免产生只能连接 X-Panel 的封闭 Agent。
- 新能力如果需要 Dashboard 配合，必须同时说明协议、版本兼容和降级行为。
- 新增配置字段必须有安全默认值，并确保旧配置仍可读取。
- 不得破坏 UUID、自动注册、ReportConfig、ApplyConfig、Server Transfer 和密钥轮换流程。
- 不得绕过 X-Panel 的绑定升级策略自行恢复 Agent 自动更新。

### 5.2 修改 Dashboard 时

- 节点级远程能力和集中管理能力优先放在 Dashboard，而不是复制到 X-Panel。
- 新接口必须维持现有 Agent 兼容性，并明确鉴权、权限和审计边界。
- 面板专属扩展应尽量采用可选能力或兼容扩展，不应阻断普通兼容 Agent 接入。

### 5.3 修改 X-Panel 时

- 不在 X-Panel 中实现 Nezha gRPC 客户端、采集器或第二套节点控制面。
- X-Panel 只管理配置文件、凭证、systemd 服务和绑定版本。
- 必须区分 systemd 的当前运行状态与开机自启状态。
- 安装和升级必须同时维护 `linux/amd64` 与 `linux/arm64` 发布包。
- 升级失败必须保持旧 X-Panel、旧 Agent 和原配置可用，不留下半升级状态。

## 6. 跨仓库变更流程

后续 AI 开始任务时，应按以下顺序工作：

1. 阅读本说明以及 X-Panel 当前的 `AGENTS.md`、`RELEASE.md` 和组件关系文档。
2. 分别检查 `x-panel/`、`agent/`、`nezha/`、`admin-frontend/` 的工作树；所有既有未提交修改默认属于用户，禁止覆盖、丢弃或 reset。
3. 明确本次需求属于 X-Panel 管理层、Agent 节点能力还是 Dashboard 控制层。
4. 如果涉及多个仓库，先写清协议、配置、版本和发布顺序，再开始实现。
5. 按测试驱动方式修改，至少覆盖兼容性、失败回滚和敏感信息保护。
6. 先完成本地验证，再进行 Linux/systemd/Dashboard 实机验证。
7. 未经用户明确要求，不创建分支，不 commit，不打 tag，不 push，不发布。

## 7. 发布协同

- X-Panel 发布工作流负责从 `CUSTOM_AGENT_REPO` 下载并校验固定版本的定制兼容 Agent，再将二进制、许可证和 systemd unit 打入发布包。
- 更新源为 `https://xpanel.qm.mk`，版本清单位于 `/releases/latest.json`。
- 修改 Agent 后，必须先形成可追溯的 Agent 版本和校验值，再更新 X-Panel 的固定版本引用并重新发布 X-Panel。
- 修改配置协议时，发布顺序必须保证旧 Dashboard、旧 Agent 和旧 X-Panel 至少具有明确的兼容或拒绝行为，不能静默损坏配置。
- 发布前必须验证安装、旧版本升级、服务启停、开机自启、配置保留、密钥轮换和卸载行为。

## 8. Fleet 退役约束

Fleet Center 源码只可作为历史归档或迁移参考，不再继续开发。禁止在 X-Panel 活跃产品路径中重新加入以下内容：

- Fleet Reporter；
- Fleet Center / Fleet Center v2 注册与遥测；
- Fleet 节点身份或 Instance ID；
- 与 Nezha Dashboard 并行的任务、终端、升级或控制通道。

如果新需求看起来需要恢复 Fleet，应先重新评估能否由 Nezha Agent + Dashboard 完成；只有用户明确改变产品方向后才能突破此约束。

## 9. 完成标准

一次协同开发只有同时满足以下条件才算完成：

- 功能位于正确的组件边界内；
- 上游协议和旧配置兼容性已说明并验证；
- X-Panel 不接管 Agent 内部职责；
- Dashboard 仍是唯一中心控制面；
- 敏感信息未泄露，配置文件权威关系未被破坏；
- amd64、arm64 及升级/回滚路径已验证；
- 没有重新引入任何 Fleet 活跃依赖。
