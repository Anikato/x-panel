# X-Panel 捆绑 Nezha Agent 与 Fleet 退役设计

- **日期：** 2026-08-09
- **状态：** 已批准并实施；本地自动化验证完成，Linux/systemd/Dashboard 实机验收待部署环境执行
- **适用范围：** `x-panel` 及其安装、升级、发布流程
- **参考实现：** `/Users/kevin/Data/Project/Nezha-Server/nezha`、`/Users/kevin/Data/Project/Nezha-Server/agent`

## 1. 背景

X-Panel 当前在主进程内运行 Fleet Reporter，并围绕 Fleet Center/Fleet Center v2 实现了注册、节点身份、遥测、任务、终端和升级联动。该路线不再继续。

新的产品方向是：X-Panel 安装包自带固定版本的官方 Nezha Agent，由独立 systemd 服务运行并连接现有 Nezha Dashboard。节点监控、远程命令、终端和以后新增的面板专属能力统一放在 Dashboard 侧，不再维护第二套控制面。

本机 Nezha 源码已经具备本设计依赖的基础能力：

- Agent 在首次读取配置时本机生成 UUID，并写回 `config.yml`；
- Dashboard 使用全局或用户 AgentSecret 认证；
- Dashboard 收到合法但未知的 UUID 时自动创建节点；
- Agent 支持 ReportConfig、ApplyConfig 和 Server Transfer 密钥轮换；
- Agent 保存配置时将权限收紧为 `0600`。

因此，本期不需要修改 Dashboard 的基础注册协议，也不需要在 X-Panel 中实现 Nezha gRPC 客户端。

## 2. 目标

1. X-Panel 发布包为 `linux/amd64` 和 `linux/arm64` 捆绑固定版本的官方 Nezha Agent。
2. Agent 以独立 systemd 服务运行，不将 Agent 协议或采集逻辑揉进 X-Panel 主进程。
3. 新装时可一次完成 X-Panel、Agent 配置和自动注册。
4. 未提供 Dashboard 地址和 AgentSecret 时，Agent 不启动、不联网。
5. 历史 X-Panel 节点升级后可在面板中配置并启用 Agent。
6. 面板可管理 Agent 配置、启动、停止、重启、开机自启和完全禁用。
7. Dashboard ApplyConfig 或密钥轮换后的文件配置不能被 DB 旧值覆盖。
8. Agent 版本跟随 X-Panel 发布，默认禁止 Agent 自动更新和 Dashboard 强制更新。
9. 从 X-Panel 活跃代码、数据库、安装器、发布和文档中完全剥离 Fleet。

## 3. 非目标

- 本期不开发 Fleet Center 或 Fleet Center v2。
- 不把 Fleet 节点身份或 Fleet Instance ID 迁移为 Nezha UUID。
- 不在 X-Panel 中实现 Nezha Agent 协议、Dashboard API 客户端或在线状态代理。
- 不在 X-Panel 中展示 Dashboard 的节点列表、监控图表、任务或终端。
- 不提供 Agent 独立选版、在线检查更新或绕过 X-Panel 的 Agent 升级按钮。
- 不自动接管节点上已经由其他方式安装的 Nezha Agent。
- 不在本期实现面板版本态势、网站证书态势或 Dashboard 侧的一键升级扩展。
- 不删除 X-Panel 原有 Hub/Agent 节点管理、AgentToken、本地终端、证书或升级能力。

## 4. 总体架构

采用 X-Panel 管理的独立 systemd 服务：

```text
X-Panel 安装器 / 管理页
          │
          ├── 配置与凭证管理
          ├── systemd 生命周期管理
          └── X-Panel 版本绑定升级
                    │
                    ▼
       xpanel-nezha-agent.service
                    │
                    ▼
          官方 nezha-agent 二进制
                    │ gRPC/TLS
                    ▼
           现有 Nezha Dashboard
```

组件边界：

- X-Panel 只管理文件、凭证、服务和版本；
- 官方 Agent 负责 UUID、注册、采集、重连、任务、终端和远程配置；
- Dashboard 是唯一中心控制面；
- X-Panel 主进程退出或重启不应终止已运行的 Agent。

## 5. 文件与服务布局

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

权限要求：

| 路径 | 权限 | 所有者 | 说明 |
| --- | --- | --- | --- |
| `/opt/xpanel/nezha-agent` | `0700` | `root:root` | 仅 root 访问组件目录 |
| `nezha-agent` | `0755` | `root:root` | 官方二进制 |
| `config.yml` | `0600` | `root:root` | 包含 AgentSecret |
| systemd unit | `0644` | `root:root` | 不包含任何凭证 |

Agent 需要采集主机状态并执行高权限远程操作，因此本期服务以 root 运行。Dashboard 被攻破可能导致节点被远程控制，这属于明确接受的安全边界。

unit 的关键约束：

```ini
[Unit]
Description=Nezha Agent managed by X-Panel
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/xpanel/nezha-agent
ExecStart=/opt/xpanel/nezha-agent/nezha-agent -c /opt/xpanel/nezha-agent/config.yml
Restart=always
RestartSec=10
UMask=0077

[Install]
WantedBy=multi-user.target
```

unit 不使用 shell 包装，也不通过 argv 或环境变量传递 AgentSecret。

## 6. 配置模型与权威关系

### 6.1 分域权威

| 数据 | 权威来源 | DB 作用 |
| --- | --- | --- |
| 哪吒功能总开关 | X-Panel DB | 表达面板期望的开机自启状态 |
| systemd 当前启停状态 | systemd | 面板只读取和操作，不伪造 |
| `server`、`client_secret` | `config.yml` | 加密镜像，供面板管理 |
| `uuid` | `config.yml` | 只读展示，不写入 DB |
| 远程运维开关 | `config.yml` | 从 `disable_command_execute` 读取 |
| 其他 Agent 字段 | `config.yml` | X-Panel 不管理、不覆盖 |

一旦 `config.yml` 存在，它就是 Agent 运行配置的唯一真相。DB 中的地址和密钥不能在启动或后台同步时反向覆盖文件。

`NezhaEnabled` 表示 X-Panel 的期望状态，systemd 的 `is-enabled` 与 `is-active` 表示操作系统实际状态。启用或完全禁用成功后才更新 DB；如果用户在面板外执行 `systemctl` 造成漂移，状态接口同时返回期望值和实际值并提示不一致，不静默改 DB，也不自动启停服务。升级流程以 systemd 实际状态为准。

### 6.2 DB 设置

使用现有 Setting 与凭证加密机制保存：

- `NezhaEnabled`：是否启用 Agent 开机自启；
- `NezhaServer`：从文件同步的规范化 `host:port`；
- `NezhaClientSecret`：从文件同步并加密保存，加入敏感字段注册表。

API 不返回 AgentSecret 明文，只返回 `secretConfigured: true/false`。UUID 和远程运维状态直接来自文件。

### 6.3 初始配置

首次显式配置时写入：

```yaml
server: dashboard.example.com:443
client_secret: "<AgentSecret>"
tls: true
insecure_tls: false
disable_auto_update: true
disable_force_update: true
disable_command_execute: false
```

初始文件不写 UUID，由官方 Agent 首次启动时在本机生成并保存。

`disable_command_execute: false` 表示默认允许 Dashboard 执行命令、终端、文件操作、ReportConfig、ApplyConfig 和 Server Transfer 密钥轮换。面板允许用户改为 `true`，但必须提示：关闭后 Dashboard 不能远程重新开启，也不能继续远程配置或轮换密钥，只能在本机 X-Panel 中恢复。

`disable_auto_update` 与 `disable_force_update` 默认同时为 `true`，确保 Agent 只能跟随 X-Panel 升级。它们是初始默认值，不在以后每次面板保存时强制覆盖 Dashboard 已写入的文件值。

### 6.4 Dashboard 地址

面板和安装器接受 HTTPS origin：

```text
https://dashboard.example.com
https://dashboard.example.com:8443
```

要求：

- 必须使用 `https`；
- 不允许 userinfo、路径、查询参数或 fragment；
- 未指定端口时使用 `443`；
- 写入 Agent 文件时转换为 `host:port` 并设置 `tls: true`、`insecure_tls: false`。

本期不提供明文连接或跳过证书校验的 UI。

如果 Dashboard ApplyConfig 把既有文件改成非 TLS 或 `insecure_tls: true`，文件仍然优先。管理页按实际值展示安全告警，不用 DB 或 UI 默认值覆盖；用户通过面板重新提交 Dashboard 地址时，才恢复为 `tls: true`、`insecure_tls: false`。

## 7. 配置同步与写入

### 7.1 文件到 DB

以下时机读取 `config.yml` 并刷新 DB 镜像：

1. X-Panel 启动时；
2. 打开 Nezha Agent 管理页时；
3. 本机修改配置前；
4. 首次启动 Agent 成功后。

本期不增加后台文件 watcher。Dashboard 修改配置后，用户下次打开管理页即可看到新状态；本机保存前也会强制读取最新文件。

### 7.2 面板写配置

面板修改地址、密钥或远程运维开关时：

1. 在停止服务前完成请求校验；
2. 记录 Agent 当前 `is-active` 与 `is-enabled` 状态；
3. 如果 Agent 正在运行，停止并等待其完全退出；
4. 读取停止后的最新 `config.yml`；
5. 解析为通用 YAML 映射，只合并用户明确修改的字段；
6. 保留 UUID、未知字段和 Dashboard 写入的其他字段；
7. 在同目录创建 `0600` 临时文件，写入并同步后原子 rename；
8. 拒绝符号链接和非普通目标文件；
9. 重新读取最终文件，刷新 DB 加密镜像；
10. 原来运行则重新启动，原来停止则保持停止。

YAML 字段语义必须保留；注释和字段顺序不作保证，因为官方 Agent 自身保存配置时也会重新序列化。

密钥输入为空或未提交表示“不修改”。代码不得从 DB 读取旧 `NezhaClientSecret` 补回文件。

文件替换成功但服务恢复失败时，不回滚用户已提交的新配置；接口返回明确的服务启动错误，页面进入“服务故障”状态并保留日志入口。

### 7.3 Dashboard ApplyConfig 与密钥轮换

```text
Dashboard ApplyConfig / Server Transfer
                │
                ▼
官方 Agent 合并并写 config.yml
                │
                ▼
X-Panel 页面读取或本机保存前同步
                │
                ▼
DB 更新加密镜像
```

如果 Server Transfer 将 `client_secret` 轮换为专用密钥，X-Panel 必须接受文件中的新值。以后只修改 Dashboard 地址或远程运维开关时，仍需保留该新密钥。

### 7.4 缺失或损坏配置

- 文件缺失：显示“配置缺失”，不使用 DB 自动重建，避免生成新 UUID 后出现重复节点；
- YAML 损坏：显示解析错误，不覆盖原文件；
- 权限过宽：读取时报警，本机成功保存时修复为 `0600`；
- 保存失败：原文件保持不变，并恢复 Agent 原运行状态。

## 8. 服务管理语义

管理页必须区分“当前运行状态”和“开机自启状态”。

| 操作 | systemd 行为 | 是否修改开机自启 |
| --- | --- | --- |
| 启动 | `systemctl start` | 否 |
| 停止 | `systemctl stop` | 否 |
| 重启 | `systemctl restart` | 否 |
| 启用哪吒 | `systemctl enable --now` | 是，成功后写 `NezhaEnabled=true` |
| 完全禁用 | `systemctl disable --now` | 是，成功后写 `NezhaEnabled=false` |

完全禁用后不运行进程、不联网，但保留二进制、配置和 UUID。重新启用时仍作为原节点连接。

## 9. 管理页设计

增加单页“哪吒 Agent”组件管理入口，复用 GOST 状态页的交互模式，但不创建新的节点控制面或多级功能菜单。

### 9.1 页面状态

| 状态 | 页面表现 |
| --- | --- |
| 待配置 | Dashboard 地址、AgentSecret、远程运维开关、“配置并启动” |
| 已配置、已停止 | 显示启动、启用、修改配置 |
| 正在运行 | 状态、版本、UUID、Dashboard、权限、停止、重启、禁用 |
| 配置损坏 | 显示解析错误，不提供 DB 覆盖按钮 |
| 外部 Agent 冲突 | 显示冲突服务或路径，禁止启动捆绑 Agent |
| 服务故障 | 显示 systemd 状态并提供日志入口 |

### 9.2 展示字段

- 组件是否可用；
- 是否已配置；
- systemd 是否 enabled；
- systemd 是否 active；
- 捆绑 Agent 版本；
- UUID；
- Dashboard 连接地址与 TLS 状态；
- AgentSecret 是否已配置，不显示明文；
- 是否允许 Dashboard 远程运维；
- 配置健康状态和冲突信息。

### 9.3 操作

- 配置并启动；
- 修改 Dashboard 地址；
- 更新 AgentSecret；
- 开启或关闭 Dashboard 远程运维；
- 启动、停止、重启；
- 启用开机自启、完全禁用；
- 查看 Agent 日志。

日志复用现有 systemd 日志能力，不新增 Agent 日志协议。

本期不提供删除二进制的“卸载”按钮。日常退出能力由“完全禁用”覆盖，保留 UUID 可防止误操作产生新节点。

### 9.4 后端接口

建议最小接口集合：

- `GET /api/v1/nezha-agent/status`：同步文件并返回组件、配置和服务状态；
- `PUT /api/v1/nezha-agent/config`：首次配置或合并更新配置，可选择配置后启用并启动；
- `POST /api/v1/nezha-agent/operate`：`start`、`stop`、`restart`、`enable`、`disable`。

日志读取复用通用 systemd 服务日志 API。所有修改接口使用现有登录、CSRF、操作日志和敏感字段脱敏机制。

## 10. 新装流程

安装器新增：

```text
--nezha-dashboard https://dashboard.example.com
```

也支持环境变量：

- `XPANEL_NEZHA_DASHBOARD_URL`；
- `XPANEL_NEZHA_AGENT_SECRET`。

Dashboard 地址不是秘密，可以进入安装参数。AgentSecret 不接受命令行参数。

输入规则：

| 输入 | 行为 |
| --- | --- |
| 地址与 AgentSecret 均存在 | 配置、enable 并启动 Agent |
| 两者均不存在 | 安装捆绑组件但保持未配置、未启动 |
| 只有地址且可交互 | 隐藏提示输入 AgentSecret |
| 只有一项且不可交互 | 预检失败，不执行半套配置 |
| 检测到外部 Agent | X-Panel 安装继续，捆绑 Agent 不启动并给出警告 |

脚本读取 AgentSecret 后立即清空并 `unset` 对应环境变量。不得将密钥写入日志、argv、安装摘要或临时文件。交互输入不回显。

面向人工安装的文档默认引导隐藏式交互输入，不给出把密钥直接写在命令行或行内环境变量赋值中的示例。非交互安装只允许由 CI、配置管理或其他秘密注入机制设置 `XPANEL_NEZHA_AGENT_SECRET`。

首次完整安装顺序：

1. 校验 Dashboard HTTPS origin 和输入配对；
2. 检测外部 Agent 冲突；
3. 安装 X-Panel、捆绑 Agent 和 unit；
4. 写初始 `config.yml`；
5. 启动 X-Panel；
6. 有完整凭证时 enable 并启动 Agent；
7. Agent 生成 UUID 并自动连接 Dashboard；
8. X-Panel 从最终文件刷新 DB。

安装器只能确认本地服务状态。它没有 Dashboard 管理 API 凭证，不应在本地伪造“Dashboard 已上线”；节点可见性由端到端验收验证。

## 11. 历史节点升级

历史 X-Panel 节点升级到包含本功能的版本时：

1. 升级包把 Agent 二进制放入目标目录；
2. 不存在配置时不启动 Agent；
3. 管理页显示“待配置”；
4. 用户输入 Dashboard 地址和 AgentSecret；
5. 点击“配置并启动”后写文件、安装/刷新 unit 并 enable/start；
6. Agent 本机生成 UUID 并在 Dashboard 自动注册。

页面安装不从 GitHub 下载“最新 Agent”，也不让用户选择独立版本；它只配置当前 X-Panel 发布包提供的 Agent。

## 12. 发布与升级

### 12.1 供应链

发布 CI 针对 `amd64`、`arm64`：

1. 固定 `NEZHA_AGENT_VERSION`；
2. 下载该版本官方 Release 资产与 `checksums.txt`；
3. 校验 SHA256；
4. 将官方二进制原样加入 X-Panel 发布包；
5. 加入 Agent 许可证和第三方声明；
6. 生成并校验 X-Panel 发布包自身的 SHA256。

建议发布包结构：

```text
xpanel
config.yaml.example
xpctl
install.sh
xpanel.service
nezha-agent/
├── nezha-agent
└── LICENSE
```

### 12.2 面板内自升级

当前 X-Panel 自升级只复制新的 `xpanel` 二进制，必须扩展为组件包升级：

1. 下载并校验完整发布包；
2. 确认新 X-Panel 与 Agent 文件都存在且架构正确；
3. 记录 Agent active/enabled 状态；
4. 如 Agent 正在运行，先停止；
5. 分别备份 X-Panel 与 Agent 二进制；
6. 替换 Agent，永不覆盖 `config.yml`；
7. 替换 X-Panel；
8. 任一步失败则回滚已替换的二进制；
9. 原来运行则启动 Agent，原来停止或禁用则保持原状；
10. 重启 X-Panel。

发布包缺失 Agent 资产或校验失败时，升级必须在替换任何运行文件前终止。

## 13. 外部 Agent 冲突检测

默认假设新节点未安装 Nezha Agent，但安装器和管理页仍需检查：

- `nezha-agent.service` 及常见实例化 unit；
- 正在运行且可执行路径不是 `/opt/xpanel/nezha-agent/nezha-agent` 的 Agent 进程；
- `/opt/nezha/agent` 等常见外部安装目录。

检测到强冲突时：

- 不停止、不覆盖、不接管外部 Agent；
- 不启动第二份捆绑 Agent；
- X-Panel 本体仍可安装或升级；
- 管理页显示冲突信息和人工处理建议。

本期不实现自动迁移、保留外部 UUID 或服务接管。

## 14. Fleet 退役

### 14.1 从 X-Panel 活跃代码移除

- Fleet Reporter 启动入口与所有采集逻辑；
- Fleet v1/v2 Enrollment、Recovery、身份、证书和连接；
- Fleet 任务、远程命令、终端和升级联动；
- `fleet-enroll`、`fleet-recover` CLI；
- 安装脚本的 Fleet Endpoint 与 Enrollment Token；
- 发布清单中的 Fleet Endpoint；
- `FleetAutoUpgrade` 及前端提示；
- Fleet DTO、API、设置、敏感字段注册、测试和只服务 Fleet 的依赖；
- 当前架构文档中的 Fleet 产品入口。

### 14.2 数据库迁移

增加幂等迁移，删除 Settings 中全部 `Fleet*` 键及其废弃密钥材料，包括：

- 开关、Endpoint、心跳间隔；
- Instance ID/Node ID；
- Enrollment、Recovery、Instance Token；
- 私钥、证书、CA 公钥；
- 自动升级、Enrollment 模式和手动恢复状态。

迁移完成后再从敏感字段注册表移除 Fleet 字段。不得复用 Fleet Instance ID 作为 Nezha UUID。

### 14.3 归档

独立 `fleet-center/` 源码和历史设计记录可以保留，但必须标记：

- 已废弃；
- 不再构建、发布或部署；
- 禁止继续接入 X-Panel；
- 仅供历史追溯。

除归档目录和 Git 历史外，活跃 X-Panel 代码不应继续出现 `fleet-center`、`FleetReporter`、`FleetEnrollment` 等运行引用。

当前 X-Panel 与 Fleet Center 工作树存在大量未提交改动。实施时不得 reset 或批量覆盖，必须逐文件辨认并保留与 Fleet 无关的用户修改。

## 15. 错误处理

| 场景 | 行为 |
| --- | --- |
| Dashboard URL 无效 | 写文件前拒绝请求 |
| AgentSecret 缺失 | 首次配置拒绝；更新时未提交表示保留现值 |
| Dashboard 不可达 | Agent 保持运行并自动重连，X-Panel 不受影响 |
| AgentSecret 错误 | Agent 重试；日志和 API 不输出密钥 |
| 配置文件缺失 | 显示故障，不用 DB 重建 |
| YAML 损坏 | 显示解析错误，不覆盖文件 |
| 配置保存失败 | 保留原文件并恢复原服务状态 |
| Agent 启动失败 | X-Panel 正常运行，页面显示 systemd 故障 |
| 外部 Agent 冲突 | 不启动捆绑 Agent，不修改外部安装 |
| Agent 资产校验失败 | 安装或升级在替换前终止 |
| X-Panel/Agent 升级中途失败 | 回滚已替换的二进制并恢复服务状态 |

## 16. 安全要求

1. 生产只接受 TLS Dashboard 地址，默认验证证书。
2. AgentSecret 不进入 argv、URL、日志、操作日志、前端响应或安装摘要。
3. `NezhaClientSecret` 使用现有凭证加密机制保存。
4. 配置 API 只返回密钥是否存在，不返回明文。
5. `config.yml` 固定 `0600`，写入前拒绝符号链接和非普通文件。
6. 配置写入使用同目录临时文件与原子 rename。
7. systemd unit 不包含凭证，不使用 shell。
8. Agent 版本资产必须校验官方 checksum，X-Panel 包再做二次校验。
9. 默认远程运维意味着 Dashboard 管理员可获得节点 root 级能力；UI 和文档必须明确提示。
10. 修改配置、启停、禁用等操作进入现有操作日志，但敏感字段必须脱敏。
11. 日志 API 返回 Agent journal 时，对已知 AgentSecret、认证元数据和可能的授权头做二次脱敏。

## 17. 测试与验收

### 17.1 单元测试

- HTTPS origin 规范化与非法输入；
- YAML 合并保留 UUID、未知字段和远程轮换后的密钥；
- 空密钥输入不修改文件；
- 文件到 DB 的加密同步；
- 缺失、损坏、符号链接和错误权限处理；
- systemd active/enabled 状态映射；
- start/stop/restart/enable/disable 操作语义；
- 配置修改保持原运行状态；
- 外部 Agent 冲突分类；
- AgentSecret API/日志/操作日志脱敏；
- Fleet 设置幂等删除迁移。

### 17.2 安装与升级测试

- 无凭证新装：Agent 不启动；
- 有效凭证新装：自动生成 UUID 并注册；
- 非交互环境缺一项凭证：预检失败；
- AgentSecret 不出现在 shell 参数捕获和安装输出；
- 历史 X-Panel 升级：Agent 资产落盘但保持待配置；
- Agent 已运行升级：配置不变、版本更新、恢复运行；
- Agent 已停止或禁用升级：升级后保持原状态；
- 发布包缺少 Agent 或 checksum 错误：不替换任何二进制；
- 外部 Agent 存在：不启动第二份服务。

### 17.3 前端测试

- 待配置、运行、停止、损坏、冲突和故障状态；
- AgentSecret 始终为空白/掩码，不回填明文；
- 修改 Dashboard 地址、远程运维和服务操作；
- 关闭远程运维前展示能力影响提示；
- 完全禁用与临时停止的文案和状态不同；
- UUID、版本、Dashboard 与自启/运行状态展示。

### 17.4 端到端验收

使用 `/Users/kevin/Data/Project/Nezha-Server` 的 Dashboard 与 Agent 兼容测试环境验证：

| 场景 | 通过标准 |
| --- | --- |
| 有效凭证首次安装 | Agent 启动后 30 秒内 Dashboard 出现节点 |
| 无凭证安装 | 无 Agent 进程、无网络连接、Dashboard 无新增节点 |
| Dashboard ApplyConfig | 文件更新，面板下次读取反映新值 |
| Dashboard 密钥轮换 | Agent 使用新密钥重连，DB 镜像更新 |
| 面板修改非密钥字段 | UUID、轮换后密钥和未知字段保持不变 |
| 修改 Dashboard 地址 | UUID 保持不变，新 Dashboard 自动注册 |
| 关闭远程运维 | 监控继续，命令、终端和远程配置被拒绝 |
| 启动/停止 | 只改变运行状态，不改变开机自启 |
| 完全禁用/重新启用 | 无后台进程；重新启用仍使用原 UUID |
| X-Panel 升级 | Agent 与 X-Panel 版本绑定，配置和服务状态保持 |
| Fleet 退役 | X-Panel 启动不再运行 Reporter，活跃代码无 Fleet 路径 |

## 18. 实施顺序建议

1. 保护并盘点当前未提交 Fleet 相关改动；
2. 增加 Nezha Agent 发布资产与校验；
3. 实现 Agent 配置、systemd 和状态服务；
4. 实现安装器和面板内升级对 Agent 资产的处理；
5. 实现管理 API 与前端单页；
6. 完成 DB 敏感字段与配置同步；
7. 剥离 Fleet 活跃代码并增加清理迁移；
8. 更新当前架构、安装和运维文档；
9. 执行单元、安装、升级及本机 Nezha 端到端验收。

## 19. 完成定义

同时满足以下条件才算完成：

- 新装和历史升级节点均能通过 X-Panel 配置并运行官方 Nezha Agent；
- 有效凭证下节点无需预建即可出现在 Dashboard；
- Dashboard 与 X-Panel 双方修改配置时不丢 UUID、不回滚密钥、不覆盖未知字段；
- Agent 可启动、停止、重启、启用和完全禁用；
- Agent 版本只能随 X-Panel 发布升级；
- AgentSecret 全链路不泄露；
- Fleet Reporter 不再启动，Fleet Center 不再出现在活跃产品路径；
- 所有验收测试通过，且现有非 Fleet 用户改动得到保留。
