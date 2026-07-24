# X-Panel 安全审计与系统性整改计划

> 本文档用于记录已确认的安全风险、功能缺陷和工程债务，并作为整改排期、验收和风险接受的唯一跟踪入口。

## 1. 文档信息

| 项目 | 内容 |
|---|---|
| 项目 | X-Panel |
| 审计日期 | 2026-07-24 |
| 审计基线 | Git commit `f58f42d` |
| 审计范围 | `backend/`、`frontend/`、`scripts/`、安装和运行配置；Fleet Center 服务端安全由其独立整改文档跟踪 |
| 审计方式 | 源码审查、数据流分析、构建、测试、竞态检查、静态检查、依赖审计 |
| 当前结论 | 存在 P0/P1 安全阻断项，修复并验收前不建议直接暴露到公网 |
| 文档状态 | Active |

行号链接以审计基线为准。后续代码变更可能导致行号漂移，应以文件名、函数名和风险编号为稳定定位依据。

## 2. 使用方式

### 2.1 责任分工

每个风险必须明确以下字段：

- `Owner`：唯一负责人，不能只写团队名称。
- `Target`：计划完成日期。
- `Status`：当前处理状态。
- `PR/Commit`：关联修复变更。
- `Verifier`：不能与 Owner 为同一人，P0/P1 至少需要一名独立复核者。
- `Evidence`：测试结果、截图、日志或验证命令。

### 2.2 状态定义

| 状态 | 含义 |
|---|---|
| `Open` | 已确认，尚未制定或开始修复 |
| `Planned` | 已明确 Owner、方案和 Target |
| `In Progress` | 正在实施 |
| `Blocked` | 存在外部阻塞，必须记录阻塞原因和解除条件 |
| `Fixed` | 已提交修复，但尚未完成独立验收 |
| `Verified` | 已按本文档验收标准独立验证 |
| `By Design` | 能力符合产品设计，不作为漏洞关闭；相关安全基线必须由独立风险项持续跟踪 |
| `Accepted` | 明确接受风险，必须记录批准人、理由和失效日期 |
| `Won't Fix` | 不修复，必须按风险接受流程审批 |

只有 `Verified`、未过期的 `Accepted` 或经审批的 `Won't Fix` 才视为风险关闭。`By Design` 只说明能力本身不是漏洞，不代表相关安全控制已经验收。

### 2.3 严重级别

| 级别 | 判定标准 | 默认处理时限 |
|---|---|---|
| Tier-0 / Trust Boundary | 具备跨实例 root 管理能力的最高信任级控制面；这是架构属性，不是漏洞等级 | 安全基线必须按 P0/P1 标准设计和验收 |
| P0 / Critical | 可导致远程 root 接管、批量实例失陷或核心信任边界完全失效 | 立即止血，24 小时内给出修复方案 |
| P1 / High | 可导致凭据、私钥泄露，MITM，或重要安全边界绕过 | 7 天内完成修复和验收 |
| P2 / Medium | 需要前置条件的攻击、DoS、重要功能错误或纵深防御缺失 | 30 天内完成 |
| P3 / Low | 可维护性、可观测性、兼容性或低影响缺陷 | 纳入常规迭代 |

## 3. 执行摘要

X-Panel 具备基础的 API、Service、Repo 分层，也能通过现有编译和测试。Fleet Reporter 默认启用、接受 Fleet Center 命令和建立远程终端是既定产品能力，不作为漏洞处理；Fleet Center 因此被定义为 Tier-0 最高权限控制面，其服务端安全由独立整改文档跟踪。

当前 X-Panel 侧安全阻断和待收尾项包括：

1. 标准在线安装的初始化抢占窗口已关闭，但手工启动空数据库、JWT 会话撤销和历史 Token 失效仍未完成。
2. 证书同步关闭 TLS 证书校验；当前按产品决定仅限可信内网使用并延期处理，风险仍保持开放。
3. 操作日志新增秘密写入已通过路由省略和递归脱敏阻断，但历史日志调查与相关凭据轮换尚未执行。
4. 外部系统凭据明文存储和 X-Panel 自有文件权限问题已完成代码整改，待真实旧版本数据库升级与测试机权限验收。

当前建议生产就绪度为 **3/10**。P0、P1 风险全部达到 `Verified` 前，不应将管理端直接暴露到不可信网络。

## 4. 风险总表

| ID | 级别 | 类型 | 标题 | 当前状态 | Owner | Target | 验收状态 |
|---|---|---|---|---|---|---|---|
| TRUST-001 | Tier-0 | 信任边界 | Fleet Center 是具备 root 运维能力的最高权限控制面 | By Design | TBD | TBD | Center 安全基线待验收 |
| SEC-002 | P0 | 认证 | 安装初始化窗口可被抢占并保留有效 JWT | In Progress | TBD | TBD | 在线安装路径已修，服务内会话撤销待处理 |
| SEC-003 | P1 | 传输安全 | 证书同步关闭 TLS 证书验证 | Open | TBD | TBD | 已延期；仅限可信内网，不视为修复 |
| SEC-004 | P1 | 数据泄露 | 操作日志记录凭据、Token 和私钥 | Fixed | TBD | TBD | 自动化回归通过；历史日志处置待完成 |
| SEC-005 | P1 | 密钥管理 | 敏感凭据明文存储且文件权限未显式收紧 | Fixed | TBD | TBD | 自动化验证通过，测试机升级待验收 |
| BUG-001 | P1 | 功能/安全 | 多节点代理认证和协议转发不完整 | Open | TBD | TBD | 未验收 |
| SEC-006 | P2 | 认证防护 | 登录限速可被代理头绕过且计数仅在内存 | Open | TBD | TBD | 未验收 |
| SEC-007 | P2 | WebSocket | Origin 不校验且 JWT 通过 URL 传递 | Open | TBD | TBD | 未验收 |
| SEC-008 | P2 | 可用性 | HTTP/WebSocket 缺少必要超时和资源限制 | Open | TBD | TBD | 未验收 |
| DEP-001 | P1 | 供应链 | 前端生产依赖包含多个 High 漏洞 | Open | TBD | TBD | 未验收 |
| BUG-002 | P2 | 兼容性 | SSH 地址拼接不支持 IPv6 | Fixed | TBD | TBD | `go vet ./...` 通过，待测试机连接验收 |
| BUG-003 | P2 | 功能 | MFA 只有状态分支，没有完整验证流程 | Open | TBD | TBD | 未验收 |
| BUG-004 | P3 | API 语义 | 业务错误普遍通过 HTTP 200 返回 | Open | TBD | TBD | 未验收 |
| DEBT-001 | P2 | 测试债务 | 认证和安全关键包测试覆盖极低 | Open | TBD | TBD | 未验收 |
| DEBT-002 | P3 | 架构债务 | 大文件、全局状态和形式化接口过多 | Open | TBD | TBD | 未验收 |

## 5. 高优先级安全与信任边界整改卡

### TRUST-001：Fleet Center 是具备 root 运维能力的最高权限控制面

**级别：Tier-0 / Trust Boundary**
**状态：By Design**

#### 已确认事实

- 新安装默认写入 `FleetEnabled=enable` 和 `FleetEndpoint=https://fcapi.qm.mk`：
  - [`backend/init/migration/migration.go`](../backend/init/migration/migration.go#L317)
- 面板启动时自动启动 Reporter：
  - [`backend/server/server.go`](../backend/server/server.go#L70)
- Reporter 自动注册、心跳、轮询中央任务：
  - [`backend/app/service/fleet_reporter.go`](../backend/app/service/fleet_reporter.go#L354)
- 支持 `run_command`、`open_shell`、升级、Cron 和进程任务：
  - [`backend/app/service/fleet_reporter.go`](../backend/app/service/fleet_reporter.go#L515)
- 命令通过 `/bin/bash -lc` 或 shell 执行：
  - [`backend/app/service/fleet_reporter.go`](../backend/app/service/fleet_reporter.go#L590)
- 远程终端会启动交互式登录 Shell：
  - [`backend/app/service/fleet_shell.go`](../backend/app/service/fleet_shell.go#L83)
- systemd 服务没有设置 `User=`，系统服务默认以 root 身份运行：
  - [`scripts/install-online.sh`](../scripts/install-online.sh#L699)
- Reporter 上报主机、网络、进程、流量和证书域名等资产信息：
  - [`backend/app/service/fleet_reporter.go`](../backend/app/service/fleet_reporter.go#L973)
- Reporter 使用 Go 标准 HTTPS/WSS 客户端校验 Fleet Center TLS，并在注册后使用 Instance Bearer Token 认证。
- Fleet Reporter 默认启用、自动注册、远程命令和终端均为产品明确需要的能力。

#### 定性

`Fleet Center 管理员 → 节点 root` 是预期授权关系，不是 X-Panel 漏洞。安全风险来自未经授权的攻击者取得 Fleet Center 管理权限、节点身份或任务通道凭据后，可以继承这一授权能力。

因此：

- 保留 `FleetEnabled=enable`。
- 保留默认 Fleet Endpoint、自动上报、命令执行和远程终端。
- 不把默认启用或 root 任务能力作为漏洞整改。
- 将 Fleet Center 定义为 Tier-0 控制面，对其管理员认证、实例注册、任务授权、会话和审计采用最高安全等级。
- Fleet Center 的具体服务端问题在其仓库 `docs/security-hardening-plan-2026-07-24.md` 中独立跟踪。

#### X-Panel 侧安全要求

- 继续使用标准 TLS 主机名和证书链校验，禁止为 Fleet 通道引入 `InsecureSkipVerify`。
- 支持安全的首次注册凭据，例如安装器传入的一次性 Enrollment Token；默认启用不等于无认证注册。
- Instance Token 必须支持轮换、撤销和安全迁移。
- 任务必须包含唯一 ID、过期时间和明确类型，客户端拒绝未知、过期和已完成任务。
- 本地记录任务 ID、类型、领取时间、完成状态和 Center 请求标识；避免记录任务中可能包含的明文秘密。
- Reporter 应能展示当前 Center、注册状态、最近心跳和最近任务，便于运维判断信任关系。
- 是否进一步拆分命令、终端和升级权限属于纵深防御选项，不作为保留 Fleet 功能的前置条件。

#### 验收标准

- 全新数据库初始化后 `FleetEnabled=enable`，并能够按产品设计连接配置的 Fleet Center。
- 正确 Center 证书、注册凭据和 Token 下，注册、心跳、任务和终端链路正常。
- 无效注册凭据、无效 Instance Token、错误 TLS 证书和错误主机名均被拒绝。
- Token 轮换后旧 Token 立即失效，节点能够安全切换到新 Token。
- 未知、过期和重复任务不会被执行。
- Fleet Center 仓库中的注册、管理员认证、RBAC、会话和审计安全项全部独立验收。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | X-Panel / Fleet Center 联合负责人（TBD） |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | TBD |

### SEC-002：安装初始化窗口可被抢占并保留有效 JWT

**级别：P0 / Critical**
**状态：In Progress**

#### 已确认事实

- `/api/v1/auth/init` 位于公开路由：
  - [`backend/router/router.go`](../backend/router/router.go#L28)
- HTTP 服务监听所有接口：
  - [`backend/server/server.go`](../backend/server/server.go#L87)
- 安装器先启动服务，再等待数据库并执行管理员设置：
  - [`scripts/install-online.sh`](../scripts/install-online.sh#L724)
- 初始化采用“查询密码为空后再分别更新”的非原子流程：
  - [`backend/app/service/auth.go`](../backend/app/service/auth.go#L83)
- CLI setup 会覆盖用户名和密码，但不会撤销此前签发的 JWT：
  - [`backend/cmd/server/setup.go`](../backend/cmd/server/setup.go#L19)
- Logout 不维护服务端撤销状态：
  - [`backend/app/api/v1/auth.go`](../backend/app/api/v1/auth.go#L117)
- JWT 解析不校验用户当前状态、密码版本或会话版本：
  - [`backend/utils/jwt/jwt.go`](../backend/utils/jwt/jwt.go#L45)

#### 攻击链

```text
安装器启动公网 HTTP 服务
        ↓
攻击者抢先调用 /api/v1/auth/init
        ↓
攻击者登录并获得 JWT
        ↓
安装器覆盖管理员密码
        ↓
旧 JWT 仍被 JWTAuth 接受
        ↓
攻击者访问文件、终端、证书等管理员能力
```

若安装时未提供预设账号，攻击者初始化的账户会直接成为最终管理员账户。

#### 修复结果（2026-07-24）

- 在线安装器现在始终在 `systemctl start` 前执行离线 `xpanel setup`。
- 未显式提供账号密码时自动生成 128 bit 随机初始密码，并在安装完成时只展示一次。
- 只提供用户名或只提供密码会中止安装，不允许退回半初始化状态。
- 账号初始化失败会中止安装，服务不会带着空密码启动。
- 这关闭了标准在线安装路径的公网抢占窗口；直接手工启动未初始化数据库、初始化原子性和 JWT 服务端撤销仍需继续整改，因此本项暂不标记 `Fixed`。

#### 立即止血

- 安装阶段不要在管理员初始化完成前监听非 loopback 地址。
- 对现有部署轮换 JWT Secret，使历史 Token 失效。
- 部署侧通过安全组或防火墙限制管理端口来源。

#### 正式整改

- 在 HTTP 启动前离线生成管理员账号。
- 如果必须通过 Web 初始化，使用一次性高熵 Bootstrap Token。
- Bootstrap Token 只能从本地文件或控制台读取，不能使用固定默认值。
- 初始化写入必须使用事务和原子条件更新，确保只能成功一次。
- JWT 增加 `sessionVersion` 或 `credentialVersion`。
- 修改密码、修改用户名、退出登录和管理员重置时使现有会话失效。
- 长期登录使用可撤销 Refresh Token，不使用 30 天不可撤销访问 Token。

#### 验收标准

- 安装过程中，初始化完成前外部网络无法访问管理 API。
- 20 个并发初始化请求只能有一个成功。
- 初始化、修改密码和管理员重置后，旧 JWT 立即返回 401。
- Logout 后同一 Token 立即返回 401。
- 新增初始化竞态、安装顺序、Token 撤销和用户名变更测试。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | 安装脚本中离线 `setup` 位于 `systemctl start` 前；`bash -n scripts/install-online.sh` 通过；会话撤销验收尚未完成 |

### SEC-003：证书同步关闭 TLS 证书验证

**级别：P1 / High**
**状态：Open**

#### 已确认事实

- TLS 客户端设置 `InsecureSkipVerify: true`：
  - [`backend/app/service/cert_sync_tls.go`](../backend/app/service/cert_sync_tls.go#L19)
- `TLSFingerprint` 被保存但未用于连接校验。
- 同步响应包含证书和私钥：
  - [`backend/app/service/cert_sync.go`](../backend/app/service/cert_sync.go#L599)
- 当前测试把接受不受信任自签名证书作为预期行为。

#### 风险

中间人可以冒充证书源，获取同步 Token，并窃取或替换证书私钥。

#### 处理决定（2026-07-24）

- 证书源地址由管理员显式配置，当前业务只在受控内网使用，因此本轮暂缓 TLS 信任模型改造。
- 补偿控制是限制证书源和 X-Panel 所在网络的访问范围，并避免在不可信网络使用该同步通道。
- `InsecureSkipVerify` 仍不能抵抗内网中间人；本项保持 `Open`，不得标记为 `Fixed` 或 `Verified`。

#### 正式整改

- 默认使用系统 CA 和标准主机名校验。
- 自签名场景支持导入专用 CA，而不是关闭全部校验。
- 如果保留 `TLSFingerprint`，实现 SHA-256 证书或 SPKI 指纹固定。
- 指纹不匹配必须硬失败，禁止自动回退到不校验模式。
- Token 不得出现在 URL 中，并应支持轮换和过期。

#### 验收标准

- 不受信任证书、主机名不匹配和错误指纹均连接失败。
- 正确系统 CA、自定义 CA 和正确固定指纹分别有测试。
- 中间人测试无法读取 Token、证书或私钥。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | 产品决定暂缓；补偿控制为管理员显式配置地址并仅限可信内网，未完成密码学验收 |

### SEC-004：操作日志记录凭据、Token 和私钥

**级别：P1 / High**
**状态：Fixed**

#### 修复前事实

- 中间件读取并存储绝大多数非 GET 请求体：
  - [`backend/middleware/operation_log.go`](../backend/middleware/operation_log.go#L33)
- 脱敏采用少量大小写敏感字段匹配：
  - [`backend/middleware/operation_log.go`](../backend/middleware/operation_log.go#L143)
- 未覆盖 `sshPassword`、`agentToken`、`privateKey`、`passPhrase`、`accessKey`、`credential`、`authorization` 等字段。
- 请求体超过 1 MiB 时，中间件可能把截断后的 Body 放回后续处理链。

#### 修复结果（2026-07-24）

- 密码、Token、私钥、数据库、备份、证书、主机、节点及原始配置等敏感路由不再读取或保存请求原文，统一记录 `[sensitive body omitted]`。
- 普通 JSON 先完整解析，再按规范化字段名递归脱敏；混合大小写、下划线、嵌套对象和数组均受保护。
- 非 JSON、URL 编码、multipart、非法 JSON 和超大 Body 只记录固定占位信息，不保存原始内容。
- 超过 1 MiB 的 JSON 使用“已读前缀 + 原始剩余流”重放，下游读取的字节序列保持完整。
- 错误消息中的 Bearer Token 和可识别秘密赋值会被替换为 `***`。
- 修复只阻止新增泄露；历史 `operation_logs` 调查、清理和已暴露凭据轮换仍需单独执行。

#### 立即止血

- 对认证、主机、节点、备份、数据库、DNS、证书和证书同步接口停止记录原始 Body。
- 检查并清理历史操作日志。
- 轮换可能已经进入日志的 SSH、数据库、DNS、对象存储、Agent 和证书同步凭据。

#### 已实施整改

- 日志记录采用敏感路由省略策略，不再依赖全局字符串替换黑名单。
- 记录普通 JSON 时先完整解析，再递归规范化字段名并脱敏。
- 对二进制、URL 编码、multipart、证书、密钥和超大 Body 只记录固定占位信息。
- 日志中禁止出现 Authorization、Cookie、Token、密码、私钥和连接字符串。
- Body 采集不得改变下游请求语义。

#### 验收标准

- 为每类敏感 DTO 建立日志测试，断言明文秘密不会出现。
- 混合大小写、嵌套对象、数组、URL 编码和 multipart 均被正确处理。
- 大于 1 MiB 的合法请求经过日志中间件后内容保持完整。
- 建立自动化“秘密字符串哨兵”测试：请求中放置唯一秘密，随后搜索数据库和日志必须无命中。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | `go test ./middleware -v`、`go test -race ./middleware`、`go test ./...`、`go test -race ./...` 通过；秘密哨兵、嵌套 JSON、URL 编码、multipart、大 Body 完整性、Bearer/赋值/带引号 JSON 错误消息测试通过；待独立复核与历史日志处置 |

### SEC-005：敏感凭据明文存储且文件权限未显式收紧

**级别：P1 / High**
**状态：Fixed**

#### 已确认事实

明文保存的内容包括：

- 主机密码、SSH 私钥和私钥口令：
  - [`backend/app/model/host.go`](../backend/app/model/host.go#L12)
- 节点 Token 和 SSH 密码：
  - [`backend/app/model/node.go`](../backend/app/model/node.go#L7)
- 对象存储 AccessKey 和 Credential：
  - [`backend/app/model/backup.go`](../backend/app/model/backup.go#L8)
- 数据库密码：
  - [`backend/app/model/database.go`](../backend/app/model/database.go#L11)
- ACME、DNS 授权和证书私钥：
  - [`backend/app/model/ssl.go`](../backend/app/model/ssl.go#L10)

数据库目录使用 `0755` 创建，安装脚本没有统一设置 `umask 077` 或逐项 `chmod`：

- [`backend/init/db/db.go`](../backend/init/db/db.go#L37)
- [`scripts/install-online.sh`](../scripts/install-online.sh#L604)

#### 修复结果（2026-07-24）

- 使用独立 `0600` JSON 密钥环和 AES-256-GCM 信封加密保护已登记的密码、Token、私钥及认证配置，字段作用域作为 AAD。
- 仓储 Create、Save、Update、Get、List 和 Page 统一保护/解密敏感字段；加解密或数据库写入错误不再被忽略。
- 首次启动自动识别全明文、混合、全密文、密钥缺失和密文损坏状态；迁移、校验和标记具备失败回滚及重试语义。
- SQLite 完成 WAL checkpoint 和 VACUUM 后才写迁移完成标记；失败时保留恢复快照，重试成功后清理。
- 密钥轮换保留旧 KEK，启动时检测非活动 KEK 密文并完成中断轮换。
- API 不再回显已登记的密码、Token、私钥和认证正文；空值默认保留旧秘密，明确清空使用独立语义。
- GOST 节点结构变化时不再按数组索引误复用密码；DNS 服务商切换时要求完整的新授权。
- 安装器、systemd 和启动路径统一收紧 X-Panel 自有配置、数据库、日志、备份和密钥权限，并拒绝路径越界、符号链接逃逸和非预期文件类型。
- `xpctl` 恢复在替换数据库前验证密钥环匹配，安装器保留其所需的 `sqlite3` 依赖。

当前状态为 `Fixed`，不是 `Verified`。仍需在测试机使用真实升级前版本数据库执行升级，并对 SQLite/WAL/SHM、日志及真实备份恢复产物完成一体化秘密哨兵扫描。

#### 正式整改

- 安装器最早阶段执行 `umask 077`。
- 数据、配置、日志和备份目录按最小权限分类。
- 配置文件、SQLite 数据库和主密钥文件至少为 `0600`。
- 启动时检测并修复过宽权限，无法修复时拒绝启动并输出明确错误。
- 对必须可恢复的密码、Token 和私钥实施信封加密。
- 数据加密密钥不得与密文保存在同一个 SQLite 数据库。
- 优先使用操作系统 Keyring、TPM、KMS 或独立 root-only 密钥文件。
- 为密钥增加版本字段，支持在线轮换和逐条重新加密。

#### 验收标准

- 全新安装后，非服务账号无法读取配置、数据库、密钥和备份。
- 复制 SQLite 文件本身不能直接恢复外部系统密码和私钥。
- 主密钥轮换不会造成服务中断或历史数据不可读。
- 权限回归测试覆盖新安装、升级安装、备份和恢复。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | 本地修复提交（以本次 `git log` 为准） |
| Verifier | 独立代码复审已完成；测试机升级验收待用户执行 |
| Evidence | `go test ./...`、`go test -race ./...`、`go vet ./...`、`pnpm run build`、Shell 语法检查、`scripts/test-xpctl.sh`、`scripts/test-install-xpctl.sh` 均通过；最终复审 Critical 0、Important 0 |

### BUG-001：多节点代理认证和协议转发不完整

**级别：P1 / High**
**状态：Open**

#### 已确认事实

- NodeProxy 向 Agent 发送 `X-Agent-Token`：
  - [`backend/app/service/node.go`](../backend/app/service/node.go#L198)
- Agent 私有 API 仍由 JWTAuth 保护，`AgentTokenAuth` 未挂载到对应路由。
- 代理没有完整保留 Query、Header、Content-Type、状态码和流式语义：
  - [`backend/middleware/node_proxy.go`](../backend/middleware/node_proxy.go#L29)
- Token 使用时间信息生成，不是密码学安全随机数：
  - [`backend/app/service/node.go`](../backend/app/service/node.go#L281)
- SSH 主机密钥验证被禁用：
  - [`backend/app/service/node.go`](../backend/app/service/node.go#L259)

#### 处理决策

在以下两个方案中必须明确选择一个：

1. **暂时下线**：隐藏入口、移除功能宣传、拒绝节点配置，直到协议完成。
2. **完整实现**：定义 Agent 专用 API 和认证边界，不复用浏览器 JWT 路由。

不建议继续维护“部分接口可用”的中间状态。

#### 正式整改

- Agent 使用独立路由组和独立认证中间件。
- Agent Token 使用 `crypto/rand` 生成，至少 256 bit 熵，只存储哈希或受保护密文。
- 使用标准反向代理或完整复制 Method、Path、RawQuery、Header、Body、Status、Response Header 和流。
- 明确定义上传、下载、WebSocket、SSE 和大文件传输行为。
- SSH 使用 known_hosts、TOFU 或管理员确认的主机指纹。
- 地址拼接统一使用 `net.JoinHostPort`。

#### 验收标准

- Agent Token 不能访问中心面板浏览器 API，JWT 不能替代 Agent Token。
- 覆盖 JSON、Query、multipart、文件下载、错误响应、流式响应和超时测试。
- 错误 Token、吊销 Token、过期 Token 和重放请求被拒绝。
- SSH 主机指纹变化时连接硬失败并产生可操作告警。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | TBD |

### DEP-001：前端生产依赖包含多个 High 漏洞

**级别：P1 / High**
**状态：Open**

#### 已确认事实

`npm audit --omit=dev` 在审计基线报告 6 个 High、6 个 Moderate。重点依赖：

| 依赖 | 审计版本 | 主要风险 | 可达性判断 |
|---|---:|---|---|
| `xlsx` | 0.18.5 | 原型污染、ReDoS | 可达；文件预览会解析用户选择的工作簿 |
| `echarts` | 6.0.0 | XSS | 可达；监控和图表页面使用 |
| `monaco-editor` / `dompurify` | 间接依赖 | 净化绕过/XSS | 需要结合编辑器输入继续验证 |
| `axios` | 1.13.4 | 多个公告 | 部分只影响 Node Adapter，不能全部按浏览器可利用处理 |
| `uuid`、`lodash`、`postcss`、`ua-parser-js` | 多版本 | 各类已知公告 | 逐项验证调用路径 |

`xlsx` 的直接调用位置：

- [`frontend/src/views/host/file/FilePreview.vue`](../frontend/src/views/host/file/FilePreview.vue#L137)

#### 正式整改

- 优先替换或升级 `xlsx`，不接受“npm 无修复版本”作为长期风险接受理由。
- 升级 ECharts 到修复版本，并验证 Tooltip、Label 和富文本渲染输入。
- 升级 Monaco 及其 DOMPurify 依赖。
- 对 Axios 公告按浏览器/Node 调用路径做可达性分类，记录排除理由。
- 引入 Lockfile 审核和定期依赖升级流程。

#### 验收标准

- `npm audit --omit=dev` 不存在未接受的 High/Critical。
- 恶意 XLSX 回归样本不能造成主线程长时间阻塞或对象原型污染。
- 图表和编辑器对不可信 HTML 有自动化 XSS 测试。
- 所有风险接受项记录影响范围、补偿措施、批准人和失效日期。

#### 跟踪

| 字段 | 内容 |
|---|---|
| Owner | TBD |
| Target | TBD |
| PR/Commit | TBD |
| Verifier | TBD |
| Evidence | TBD |

## 6. P2/P3 风险处理要求

### SEC-006：登录限速和可信代理

#### 问题

- 登录失败统计依赖 `c.ClientIP()`。
- Gin 没有显式配置 Trusted Proxies。
- 计数器仅在内存中保存，重启后清空。
- 三次失败显示验证码不是速率限制。

#### 整改

- 无反向代理时执行 `SetTrustedProxies(nil)`。
- 有反向代理时只配置明确的代理 CIDR。
- 使用 IP + 用户名双维度 Token Bucket 或指数退避。
- 失败计数进入共享或持久存储，并设置合理 TTL。
- 初始化、登录和验证码接口分别设置独立限制。

#### 验收

- 伪造 `X-Forwarded-For` 不能绕过限制。
- 并发、多进程和服务重启后限制仍有效。
- 不能通过大量不同用户名轻易绕过单 IP 限制。

### SEC-007：WebSocket Origin 和 URL Token

#### 问题

- Terminal WebSocket 的 `CheckOrigin` 永远返回 true：
  - [`backend/app/api/v1/terminal.go`](../backend/app/api/v1/terminal.go#L28)
- JWTAuth 接受 `?token=`：
  - [`backend/middleware/jwt_auth.go`](../backend/middleware/jwt_auth.go#L17)
- 前端把 JWT 保存到 localStorage/sessionStorage：
  - [`frontend/src/utils/auth.ts`](../frontend/src/utils/auth.ts#L1)

#### 整改

- WebSocket 只允许配置的同源 Origin。
- 使用 HttpOnly、Secure、SameSite Cookie，或短时单次 WebSocket Ticket。
- 下载、终端和证书接口不得使用长期 JWT 查询参数。
- Ticket 必须绑定用户、目的、资源、过期时间和一次性随机数。

#### 验收

- 非授权 Origin 的握手失败。
- URL、访问日志和浏览器历史中不出现长期 JWT。
- Ticket 过期、重放或跨资源使用均失败。

### SEC-008：HTTP 和 WebSocket 资源限制

#### 问题

- HTTP Server 未设置请求头、读取、写入和空闲超时：
  - [`backend/server/server.go`](../backend/server/server.go#L93)
- WebSocket 未统一设置 ReadLimit、读写 Deadline 和 PongHandler。

#### 整改

- 配置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 和 `MaxHeaderBytes`。
- 为普通 JSON API 设置 Body 大小上限。
- 上传接口采用单独限额和流式处理。
- WebSocket 设置消息大小、空闲时间、Ping/Pong 和断线清理。

#### 验收

- 慢请求、超大 Header、超大 Body 和空闲连接在预期时间内被释放。
- 正常大文件上传、下载和终端会话不受错误超时影响。

### BUG-002：IPv6 地址拼接错误

**状态：Fixed**

#### 问题

`go vet` 报告两处 `fmt.Sprintf("%s:%d")` 生成地址，不支持 IPv6：

- [`backend/app/service/host.go`](../backend/app/service/host.go#L235)
- [`backend/app/service/node.go`](../backend/app/service/node.go#L268)

#### 整改与验收

- 统一使用 `net.JoinHostPort(host, strconv.Itoa(port))`。
- 增加 IPv4、IPv6、域名和带 zone IPv6 的单元测试。
- `go vet ./...` 必须零错误。

主机和节点 SSH 连接地址现已统一使用 `net.JoinHostPort`，`go vet ./...` 已零错误；真实 IPv6/zone 地址连接仍待部署环境验收。

### BUG-003：MFA 流程不完整

#### 问题

当前仅存在 MFA 状态字段和登录分支，没有完整的绑定、挑战、验证、恢复码和禁用流程。开启状态后，登录可能返回无 Token 的不完整响应。

#### 处理决策

- 若当前版本不计划交付 MFA：隐藏设置、删除不可达状态和宣传。
- 若交付 MFA：先定义完整状态机和恢复流程，再实现 UI 和 API。

#### 验收

- 未配置、绑定中、已启用、挑战、恢复和禁用状态均有端到端测试。
- MFA 挑战不能重复使用，恢复码只能使用一次。
- 服务器时间漂移、暴力尝试和账号恢复有明确策略。

### BUG-004：HTTP 状态码语义错误

#### 问题

业务错误经常返回 HTTP 200，仅在 JSON 中写 `code=500`：

- [`backend/app/api/v1/helper/helper.go`](../backend/app/api/v1/helper/helper.go#L48)

#### 整改

- 认证失败使用 401，权限失败使用 403，参数错误使用 400/422。
- 资源不存在使用 404，冲突使用 409，限速使用 429。
- 未预期内部错误使用 500，且不向客户端暴露内部路径和原始错误。
- 建立统一错误码与 HTTP 状态映射表。

#### 验收

- API、反向代理、APM 和前端均按 HTTP 状态正确识别失败。
- 错误响应不包含数据库、命令、文件系统和堆栈细节。

## 7. 工程债务治理

### DEBT-001：测试覆盖不足

审计时观察到：

- Service 包覆盖率约 2.1%。
- API 包覆盖率约 0.4%。
- Migration 包覆盖率约 9.5%。
- Router、Middleware、Repo、JWT、Auth 等关键区域接近 0%。
- 约 205 个 Go 文件只有约 15 个测试文件。
- 约 132 个前端 TS/Vue 文件只有 2 个测试文件。

#### 覆盖目标

| 区域 | 最低分支覆盖目标 |
|---|---:|
| Auth、JWT、初始化、权限中间件 | 85% |
| Fleet 任务验证和权限 | 85% |
| 证书同步、密钥和日志脱敏 | 85% |
| 节点代理、上传下载 | 80% |
| 其他 Service/API | 60% |

覆盖率不是唯一质量门槛；关键攻击路径必须使用针对性负向测试，而不是仅追求行覆盖数字。

### DEBT-002：大文件、全局状态和无效抽象

审计时观察到：

- 41 个文件超过 500 行。
- 17 个文件超过 800 行。
- 9 个文件超过 1,000 行。
- 约 203 处显式忽略 Go 错误。
- 前端约 143 处 `any`。
- 约 39 个空 `catch`。
- 46 个 Service 接口对应约 45 个实现，存在大量“一接口一实现”。
- DB、日志、配置、Cron、模板和认证追踪器通过全局变量共享：
  - [`backend/global/global.go`](../backend/global/global.go#L1)

#### 治理原则

- 优先拆分安全风险高、修改频繁的文件，不做全仓一次性重写。
- 仅在存在多实现、稳定替换边界或测试替身价值时保留接口。
- 新代码禁止无说明忽略错误；允许忽略时必须写出原因。
- 新增 TypeScript 业务类型不得使用裸 `any`。
- 空 `catch` 必须删除或记录、转换、重新抛出。
- 逐步通过构造函数注入 DB、日志和外部客户端，减少全局状态。

#### 第一批拆分对象

1. `backend/app/service/fleet_reporter.go`
2. `backend/app/service/file.go`
3. `backend/app/service/website.go`
4. `backend/app/service/ssl.go`
5. `frontend/src/views/host/file/index.vue`
6. `frontend/src/views/website/config/index.vue`
7. `frontend/src/views/ssl/index.vue`

每次重构必须有行为测试保护，不得与大规模功能开发混在同一个变更中。

## 8. 分阶段整改路线图

### M0：立即止血

目标：阻断 X-Panel 侧未授权接管和秘密持续泄露；Fleet 默认启用作为产品能力保持不变。

- [x] SEC-002：标准在线安装改为服务启动前离线初始化。
- [ ] SEC-002：制定现有实例 JWT Secret 轮换方案。
- [ ] SEC-003：证书同步在校验修复前默认禁用。
- [x] SEC-004：敏感路由停止记录原始 Body。
- [ ] SEC-004：完成历史日志调查和凭据轮换清单。
- [x] TRUST-001：确认 Fleet Center 为 Tier-0 控制面，并关联其独立安全整改台账。

**M0 退出条件：** 不存在公开初始化窗口或持续写入日志的明文秘密；Fleet Center 的高优先级控制面风险已明确 Owner 和 Target。

### M1：安全边界重构

目标：完成所有 P1 风险修复。

- [ ] X-Panel 支持 Fleet Center 的可信注册、Token 轮换和任务过期/防重复协议，同时保持默认启用。
- [ ] JWT 会话版本和服务端撤销。
- [ ] 证书同步恢复完整 TLS 信任链。
- [x] 敏感字段信封加密和密钥轮换（代码完成，待测试机验收）。
- [x] 安装、升级、备份和恢复的权限模型（代码完成，待测试机验收）。
- [ ] 节点代理下线或完成独立 Agent 协议。
- [ ] 处理所有未接受的前端 High/Critical 依赖漏洞。

**M1 退出条件：** 所有 P0/P1 项达到 `Verified`，或存在明确批准且未过期的风险接受记录。

### M2：纵深防御和正确性

目标：完成 P2 风险和关键业务测试。

- [ ] 登录限速和可信代理。
- [ ] WebSocket Origin 和单次 Ticket。
- [ ] HTTP/WebSocket 超时及资源限制。
- [x] IPv6 地址兼容（代码完成，待真实连接验收）。
- [ ] MFA 明确下线或完整交付。
- [ ] 安全关键区域达到覆盖目标。

### M3：持续工程治理

目标：降低回归率和长期维护成本。

- [ ] 拆分第一批大文件。
- [ ] 减少无价值接口和全局状态。
- [ ] 建立错误处理、TypeScript 类型和空 catch 的静态门槛。
- [ ] 建立周期性依赖更新和安全扫描。
- [ ] 每个版本发布前执行本文档第 9 节验收。

## 9. 统一验证门槛

### 9.1 后端

在 `backend/` 目录执行：

```bash
go test ./...
go test -race ./...
go vet ./...
govulncheck ./...
```

要求：

- 所有命令成功退出。
- 不允许通过跳过包、删除测试或忽略安全检查来“修复”CI。
- `govulncheck` 尚未集成时，应先加入固定版本的 CI 工具链。

### 9.2 前端

在 `frontend/` 目录执行：

```bash
npm ci
npx vue-tsc --noEmit -p tsconfig.app.json
npx vitest run
npm run build
npm audit --omit=dev
```

要求：

- 类型检查、测试和构建全部通过。
- 不存在未审批的 High/Critical 生产依赖漏洞。
- 构建产物的大块告警必须记录原因和优化计划。

### 9.3 安全回归

至少覆盖：

- [ ] 并发初始化只能成功一次。
- [ ] 修改密码和 Logout 后旧 Token 失效。
- [ ] 全新安装默认启用 Fleet，并能使用有效注册凭据完成注册、心跳、任务和终端链路。
- [ ] 无效注册凭据、错误 TLS 证书、无效 Token、过期和重复 Fleet 任务均被拒绝。
- [ ] Fleet Instance Token 轮换后旧 Token 立即失效。
- [ ] 证书错误、主机名错误和指纹错误全部失败。
- [ ] 操作日志中不存在秘密哨兵字符串。
- [ ] 伪造代理头不能绕过登录限速。
- [ ] 非授权 Origin 不能建立 Terminal WebSocket。
- [ ] URL、浏览器历史和访问日志中不出现长期 JWT。
- [ ] SSH 主机指纹变化时连接失败。
- [ ] 超大和慢速请求按限制断开。

## 10. 风险接受模板

P0 默认不得接受。P1 风险接受必须有项目安全负责人和业务负责人共同批准。

```markdown
### Risk Acceptance: <风险 ID>

- 批准人：
- 批准日期：
- 失效日期：
- 无法立即修复的原因：
- 实际暴露范围：
- 补偿控制：
- 监控和告警：
- 重新评估条件：
- 关联 Issue/PR：
```

风险接受必须有失效日期；到期后自动恢复为 `Open`。

## 11. 单项更新模板

```markdown
### <风险 ID> 更新：YYYY-MM-DD

- Status：
- Owner：
- Target：
- 本次完成：
- 剩余工作：
- 阻塞项：
- PR/Commit：
- 验证命令：
- 验证结果：
- Verifier：
```

## 12. 审计基线结果

### 通过

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- Vue TypeScript 类型检查
- 前端现有 5 个单元测试
- Vite 生产构建
- Shell 语法检查、`xpctl` 恢复测试和独立安装测试

### 未通过或存在告警

- `npm audit --omit=dev`：6 High、6 Moderate。
- Vite 产生多个 1–7 MiB 大块资源。
- 安全关键区域测试覆盖不足。
- SEC-005 尚缺真实旧版本数据库 fixture 升级和 WAL/SHM/日志/真实备份恢复一体化哨兵验收。

### 审计限制

- 本次未执行真实生产环境渗透测试。
- 审计环境未安装 `govulncheck`、`gosec`、Semgrep 等工具，因此 Go 依赖漏洞和完整污点分析需要补充。
- 依赖漏洞结果是 2026-07-24 的快照，后续必须重新扫描。
- 本文档记录的是源码基线事实；实际部署还需要单独核查防火墙、反向代理、文件权限、systemd 沙箱和历史日志。

## 13. 变更记录

| 日期 | 变更 | 作者 |
|---|---|---|
| 2026-07-24 | 创建审计基线、风险台账、整改路线图和验收标准 | Codex |
| 2026-07-24 | 保留 Fleet Reporter 默认启用，接入一次性 Enrollment Token，并关闭标准在线安装的初始化抢占窗口 | Codex |
| 2026-07-24 | SEC-004 阻断操作日志新增秘密泄露并修复超大请求截断；SEC-003 按可信内网补偿控制延期 | Codex |
| 2026-07-24 | SEC-005 完成凭据加密、迁移、轮换、脱敏、权限和恢复门禁实现；BUG-002 完成 IPv6 地址拼接修复 | Codex |
