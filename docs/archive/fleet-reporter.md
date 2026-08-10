# Fleet Reporter 说明

> **归档说明（2026-08-09）**：Fleet Reporter、Fleet v2 与 Fleet Center 已从 X-Panel 活跃产品路径退役。本文仅保留历史实现与运维背景，不再代表受支持功能，也不得作为新开发或部署依据。

## 定位

Fleet Reporter 是 X-Panel 内置的 Fleet Center 节点客户端。新版 Reporter 只支持 Fleet Center v2：配置 Bootstrap 后自动 Enrollment，取得证书后维持主动出站 WSS。当前 WSS 已承载身份认证和自动续签；实时状态、任务、通知和终端属于后续 capability，尚未接入远程通道。

> 2026-08-03 已完成 v2-only 收敛。v1 register、heartbeat、task poll、notification POST 和旧远程 Shell 网络路径已删除；安全迁移仍保留旧 setting 名称，只用于升级时清空历史凭据。正式规格见 `fleet-center/docs/superpowers/specs/2026-08-03-fleet-center-v2-bootstrap-cutover-design.md`。

默认配置写在 `settings` 表中：

| Key | 默认值 | 说明 |
|---|---|---|
| `FleetEnabled` | `enable` | 是否启用 Fleet Reporter |
| `FleetEndpoint` | 空 | 由升级参数、本地已有配置或更新清单下发，不编译写死 |
| `FleetInstanceID` | 空 | 首次启动自动生成，作为稳定 Node ID |
| `FleetEnrollmentToken` | 空 | Fleet Center 签发的受管注册密钥，注册成功后自动清除 |
| `FleetRecoveryToken` | 空 | 绑定 Node ID 的一次性恢复凭据，成功恢复后清除 |
| `FleetEnrollmentMode` | 空 | Bootstrap 临时模式：`automatic` 或显式 `force` |
| `FleetNodePrivateSeed` | 空 | Ed25519 节点 seed，由凭据管理器加密，不得人工复制 |
| `FleetPendingRecoverySeed` | 空 | 换钥恢复候选 seed，加密保存，只有恢复成功后转为 active |
| `FleetNodeCertificate` | 空 | Fleet Center Node CA 签发的应用层节点证书，加密保存 |
| `FleetNodeCAPublicKey` | 空 | 经公网 HTTPS Enrollment 响应取得并固定的 Node CA 公钥 |
| `FleetManualRecoveryRequired` | 空 | 服务端拒绝或本地身份损坏后的持久化人工恢复标记 |
| `FleetHeartbeatIntervalSeconds` | `300` | Reporter 低频生命周期检查周期；实时遥测后续走 WSS 独立节奏 |

生产安装的默认数据库路径：

```bash
/opt/xpanel/data/db/xpanel.db
```

查看当前 Fleet 配置：

```bash
sqlite3 /opt/xpanel/data/db/xpanel.db "select key,value from settings where key like 'Fleet%';"
```

不要在工单、日志或截图中展示任何 Token、节点 seed、证书全文或隐藏管理入口。Node CA 是 Fleet Center 的应用层 Ed25519 签名密钥，与网站公网 SSL 证书互相独立，不需要把 Node CA 安装到系统 TLS 信任库。旧 `FleetInstanceToken`、`FleetLegacyInstanceToken` 和 `FleetProtocolMode` 在 v2-only 收敛后必须由向前迁移清除或停止读取。

## v2 节点身份与注册

Fleet Reporter 继续默认启用。新节点获得 Fleet Center 的受管 Enrollment Token 后，会在本机生成 Ed25519 密钥，通过公网 HTTPS 调用 `/api/v2/fleet/enroll`，验证并保存 Node CA 签发的节点证书。Token 可以按管理员配置重复使用或永不过期，但它只负责 Bootstrap；当前节点注册成功后仍立即清除本地副本。

托管安装时，Endpoint 可使用参数或环境变量，Token 只允许通过环境变量传入。交互执行时先无回显读取，避免秘密进入命令历史和进程 argv：

```bash
read -rsp 'Fleet Enrollment Token: ' fleet_enrollment_token && echo
export XPANEL_FLEET_ENDPOINT='https://fleet-node.example.com'
export XPANEL_FLEET_ENROLLMENT_TOKEN="$fleet_enrollment_token"
sudo --preserve-env=XPANEL_FLEET_ENDPOINT,XPANEL_FLEET_ENROLLMENT_TOKEN \
  bash install-online.sh
unset fleet_enrollment_token XPANEL_FLEET_ENDPOINT XPANEL_FLEET_ENROLLMENT_TOKEN
```

自动化安装或升级同样使用这两个环境变量，不得把 Token 拼进 `--fleet-enrollment-token`（该参数不存在），也不得使用 `sudo env TOKEN=...` 把秘密变成 `sudo` 的 argv。安装器读取秘密后立即取消导出，只在执行 `xpanel fleet-enroll` 时向该子进程注入。Fleet Bootstrap 失败会单独报错并返回退出码 `2`，但新二进制和本地 X-Panel 服务仍会启动；Fleet 暂时离线不影响面板本地功能，之后可生成新 Token 按下方命令手动恢复。

向已安装但尚未注册的节点离线写入 Token：

```bash
read -rsp 'Fleet Enrollment Token: ' fleet_enrollment_token && echo
export XPANEL_FLEET_ENROLLMENT_TOKEN="$fleet_enrollment_token"
export XPANEL_FLEET_ENDPOINT='https://fleet-node.example.com'
sudo --preserve-env=XPANEL_FLEET_ENDPOINT,XPANEL_FLEET_ENROLLMENT_TOKEN \
  /opt/xpanel/xpanel fleet-enroll
unset fleet_enrollment_token XPANEL_FLEET_ENDPOINT XPANEL_FLEET_ENROLLMENT_TOKEN
sudo systemctl restart xpanel
```

安全行为：

- Token 不需要出现在 X-Panel 配置文件或服务启动参数中。
- v2 请求只允许 `https://` origin，拒绝 HTTP、URL 用户信息、查询参数、片段、额外路径和 HTTP 重定向。
- Enrollment 响应上限为 64 KiB，并严格拒绝未知字段、尾随对象、错误 Node ID、公钥、Node CA 签名和非规范时间。
- 本地私钥持有证明绑定协议版本、凭据类型、Node ID、公钥和凭据 SHA-256；请求和日志不输出私钥或 Token。
- Enrollment 成功后清除当前节点的 Token；Fleet Center 中可重复使用的注册密钥是否继续有效由管理员配置决定。
- 新版 Reporter 不使用 v1 Bearer Token，也不连接旧 Fleet Center。

### v2-only 生命周期

新版 Reporter 不再存在协议选择模式：

1. 没有 Endpoint 时进入 `not_configured`，不联网且不影响本地业务。
2. 有 Endpoint 和 Enrollment Token、没有证书时自动注册；失败保留加密 Token 并退避重试。
3. 有可用证书时只进入 v2 WSS 生命周期。
4. 证书剩余 30 天时通过已认证连接自动续签，不使用 Enrollment Token。
5. 证书损坏、过期、吊销或被服务端判为无效后进入 `manual_recovery_required`；seed 可用时执行 `fleet-enroll --force`，私钥损坏时执行管理员授权的 `fleet-recover --reset-key --yes`。该状态不会自动生成新 key 或清空现有身份。
6. 任何状态都不读取旧 Instance Token 作为认证，也不降级到旧 Fleet API。

Fleet Center 必须先部署，新版 X-Panel 再分批升级。未注册、断线或证书错误只影响 Fleet 能力，不阻止面板本地服务启动。

> 当前实现状态（2026-08-06）：v2 Enrollment、force 重签、私钥 Recovery、证书安全存储、WSS 挑战连接、节点自动重建、自动续签闭环、Endpoint 更新清单/安装脚本接入和 v1 Reporter 删除已经完成。已认证 WSS 上的 `node.telemetry` 实时样本已接入，采样间隔由 Fleet Center 按需下发（默认 5 秒，被观察时 1 秒）。任务、通知和终端仍待迁入已认证 WSS。

### 采样间隔协商（本地完成，尚未部署）

`node.hello` 携带 `capabilities: ["telemetry.interval"]`，向 Fleet Center 声明本节点支持运行时改频。旧版本不发送该字段，Fleet Center 会把它当作不支持并且永远不向它发送频率消息 —— 这是必须的，因为节点对无法识别的服务端消息会判定协议违规并结束当前连接。

Fleet Center 通过 `server.telemetry_config`（payload 为 `{"intervalMs": <整数毫秒>}`）下发目标间隔。节点行为：

- 把收到的值钳制到 1–60 秒后再应用，不信任任意值。
- payload 非法时忽略该消息并继续读取，不断开连接。频率是尽力而为的控制信号，不构成故障。
- 应用新间隔时只重建采样 ticker，不影响认证状态与证书续签。
- 其他未知消息类型仍然按既有策略视为协议违规。

认证完成后节点用 `fleetV2InstallKeepalive` 同时安装 Pong 和 Ping 处理器，两者都刷新读超时。Fleet Center 在没有事件时只发 Ping，因此只在 Pong 上刷新会让节点每个读超时周期就拆掉一条健康连接；Ping 处理器在刷新后仍按 gorilla 语义回 Pong，保证服务端不会反过来判定节点已死。

每次采样都读取 CPU、内存、交换、负载与网络速率；进程数和磁盘用量固定按 5 秒刷新并复用缓存值，避免 1 秒档在进程较多的主机上产生常驻的 `/proc` 扫描开销。CPU 占用由采样器自持的累计 CPU 时间增量计算，不再使用会与其他调用方共享全局基线的 `cpu.Percent(0, ...)`。

## v2 主动连接

证书可用后，X-Panel 从 `FleetEndpoint` 的 HTTPS origin 派生固定的 `wss://<origin>/api/v2/fleet/connect`，主动出站建立 `fleet.node.v1` 子协议连接。它不发送浏览器 `Origin`，也不需要节点暴露公网端口。

认证顺序固定为：

```text
X-Panel node.hello（Node ID + 证书）
Fleet Center server.challenge（32 字节随机挑战 + 30 秒期限）
X-Panel node.authenticate（节点私钥签名）
Fleet Center server.ready
```

安全和运行边界：

- 签名绑定 Node ID、证书序列号、连接 ID、挑战和到期 Unix 秒，不能复制到另一节点或连接重放。
- 服务端在第一次认证尝试前消费挑战；错误签名不能在同一连接重试。
- 同一 Node ID 的新认证连接会替换旧连接，避免同一节点产生两个活动命令通道。
- 断线从 1 秒开始指数退避，最大 1 分钟；Reporter 不会退回已清除的 v1 Bearer Token。
- 续签候选必须通过 Node ID、公钥、固定 Node CA、规范时间和 365 天有效期校验；客户端先原子保存证书，再经单 writer 回复 `node.certificate_applied`。非法候选属于协议错误，不能把仍有效的本机身份误标为需要人工恢复。
- 本地保存失败时不回复 applied，先在当前连接内按 1、2、4 秒重试并保留旧证书；持续失败才让 Supervisor 重连，避免一次瞬时故障永久错过续签。
- applied 丢失后，下一次重连重新加载已保存的新证书，服务端可据 pending 候选完成激活；成功 applied 后当前连接也立即重新加载新身份。
- 每个认证 WSS 会话最长 24 小时；即使底层网络常年不断，也会定期重新认证并让服务端重新判断 30 天续签窗口。
- `CERTIFICATE_RENEWAL_FAILED` 是非致命错误，不主动断开节点；普通网络错误自动退避重连。
- 本地时间偏差和协议错误会结束当前 Supervisor；证书过期、吊销或无效会进入 `manual_recovery_required`，并输出不含证书或签名的稳定错误码。
- 当前长连接只完成可信生命周期和自动重建；实时资源、网速、任务和终端消息将在后续工作流迁移到该通道，不能把尚未存在的消息当作已实现。

## 遥测负载（后续 WSS capability）

Reporter 保留本机清单/资源采集代码，但 v2-only 版本当前不会通过旧注册或心跳接口发送这些数据。后续实时遥测 capability 计划上报：

- 面板版本、commit、构建时间、Go 版本。
- 主机名、系统发行版、内核、架构、时区、虚拟化类型、TCP 拥塞控制。
- CPU 型号、物理核、逻辑核。
- 内存、交换分区、根分区容量与使用量。
- CPU 使用率、负载、TCP/UDP 连接数、进程数。
- 网络速率和累计流量。
- 网卡地址列表。

后续 Fleet Center 将使用这些数据更新实例快照并记录监控采样；当前尚未远程发送。

## 网卡地址遥测规划

Reporter 会从系统网卡中采集 IPv4 / IPv6 地址。待实时遥测 capability 落地后，`host.networkInterfaces` 可用于在 Fleet Center 详情页看到 NetBird、NAT 内网、VPC 内网或公网地址；当前不会远程发送该字段。

上报示例：

```json
{
  "host": {
    "hostname": "Pathfinder",
    "networkInterfaces": [
      {
        "name": "eth0",
        "ips": ["74.88.96.104"]
      },
      {
        "name": "wt0",
        "ips": ["100.80.12.34", "fd7a:115c:a1e0::1234"]
      }
    ]
  }
}
```

过滤规则：

- 跳过未启用网卡。
- 跳过 loopback。
- 跳过 link-local 地址。
- 跳过 Docker / 容器 / CNI 常见网卡前缀：
  - `docker*`
  - `br-*`
  - `veth*`
  - `cni*`
  - `flannel*`
  - `kube*`
  - `tunl*`
  - `ipvs*`
  - `nerdctl*`

NetBird、Tailscale、自建 WireGuard、普通物理网卡、云厂商内网网卡不会被默认排除。

## 任务通道

Fleet Center 不直接连接节点。v2 任务将通过节点已经建立的主动出站 WSS 下发和回传，不继续使用旧 `/api/v1/fleet/tasks/*` Bearer Token 轮询。可靠任务状态机属于后续工作流。

本地仍保留以下任务执行 helper，供后续经过 capability/RBAC 认证的 WSS 任务状态机复用；当前没有远程下发入口，不能视为已支持的 Fleet 任务：

| 类型 | 行为 |
|---|---|
| `tail_panel_log` | 拉取 X-Panel 系统日志 |
| `run_command` | 使用 `/bin/bash` 或 `/bin/sh` 执行命令 |
| `open_shell` | 反连 Fleet Center，建立远程终端 |
| `panel_check_update` | 调用 X-Panel 升级服务检查更新 |
| `panel_upgrade` | 调用 X-Panel 升级服务执行升级 |

节点不需要暴露公网 API，只需要能主动访问 `FleetEndpoint`。

## 通知上报

X-Panel 本地消息中心当前只在本地保存通知，不启动远程 goroutine。后续将通过已认证 v2 WSS 异步上报节点事件，不使用旧 Instance Token；计划 payload 保留以下业务字段：

```json
{
  "instanceId": "xp_...",
  "notificationId": 123,
  "type": "success",
  "title": "数据库「app」备份完成",
  "content": "备份文件已保存到：/opt/xpanel/data/backup/database/app_20260509120000.sql",
  "source": "database",
  "targetUrl": "/database",
  "createdAt": "2026-05-09T12:00:00+08:00"
}
```

上报是尽力而为：Fleet Center 不可达或暂未支持对应消息时，不影响本地通知创建和业务流程。

## 常见排查

确认 Fleet Center 基础入口可达：

```bash
curl -v https://fleet-node.example.com/api/v1/health
```

查看 Reporter 日志：

```bash
grep -i "fleet reporter" /opt/xpanel/data/log/xpanel.log | tail -50
```

常见日志含义：

| 日志 | 含义 |
|---|---|
| `ENROLLMENT_INVALID` | 注册密钥无效、已撤销、过期或达到使用上限 |
| `PUBLIC_KEY_MISMATCH` | 现有 Node ID 的公钥不同，必须使用绑定 Recovery Token |
| `CERTIFICATE_EXPIRED` | 自动续签未完成，使用相同私钥显式重签 |
| `CERTIFICATE_REVOKED` | 证书已撤销，确认是否需要恢复或重新注册 |
| `RECOVERY_INVALID` | Recovery Token 无效、过期、已使用或绑定的 Node ID 不一致 |
| `CLOCK_SKEW` | 节点系统时间偏差过大，先校时再重试 |

如果重装面板后 Fleet Center 出现两个同名节点，通常是新的安装生成了新的 `FleetInstanceID`。保留最新在线实例，删除旧的离线实例即可。

如证书损坏但节点 seed 仍在，创建或选用有效注册密钥后执行：

```bash
read -rsp 'Fleet Enrollment Token: ' fleet_enrollment_token && echo
export XPANEL_FLEET_ENROLLMENT_TOKEN="$fleet_enrollment_token"
export XPANEL_FLEET_ENDPOINT='https://fleet-node.example.com'
sudo --preserve-env=XPANEL_FLEET_ENDPOINT,XPANEL_FLEET_ENROLLMENT_TOKEN \
  /opt/xpanel/xpanel fleet-enroll --force
unset fleet_enrollment_token XPANEL_FLEET_ENDPOINT XPANEL_FLEET_ENROLLMENT_TOKEN
sudo systemctl restart xpanel
```

如节点 seed 也丢失，必须在 Fleet Center 节点详情生成一次性 Recovery Token，再执行：

```bash
read -rsp 'Fleet Recovery Token: ' fleet_recovery_token && echo
export XPANEL_FLEET_ENDPOINT='https://fleet-node.example.com'
export XPANEL_FLEET_RECOVERY_TOKEN="$fleet_recovery_token"
sudo --preserve-env=XPANEL_FLEET_ENDPOINT,XPANEL_FLEET_RECOVERY_TOKEN \
  /opt/xpanel/xpanel fleet-recover --reset-key --yes
unset fleet_recovery_token XPANEL_FLEET_ENDPOINT XPANEL_FLEET_RECOVERY_TOKEN
sudo systemctl restart xpanel
```

不要通过 SQLite 手工复制、删除或替换 seed、证书和 Node ID；普通注册密钥不能改变已有 Node ID 的公钥。Fleet Center 的 Node CA 丢失时，节点级 Recovery Token 同样无效，只能恢复成套备份，或重建信任根后让全部节点重新 Enrollment。
