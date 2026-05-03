# Fleet Reporter 说明

## 定位

Fleet Reporter 是 X-Panel 内置的 Fleet Center 上报协程。面板启动后默认启用，定期向 Fleet Center 注册、心跳、拉取任务并回传结果。

默认配置写在 `settings` 表中：

| Key | 默认值 | 说明 |
|---|---|---|
| `FleetEnabled` | `enable` | 是否启用 Fleet Reporter |
| `FleetEndpoint` | `https://fcapi.qm.mk` | Fleet Center 节点 API 域名 |
| `FleetInstanceID` | 空 | 首次启动自动生成，作为实例身份 |
| `FleetInstanceToken` | 空 | 注册后由 Fleet Center 下发 |
| `FleetHeartbeatIntervalSeconds` | `300` | 心跳间隔，服务端可下发覆盖 |
| `FleetTaskPollIntervalSeconds` | `10` | 任务轮询间隔 |

生产安装的默认数据库路径：

```bash
/opt/xpanel/data/db/xpanel.db
```

查看当前 Fleet 配置：

```bash
sqlite3 /opt/xpanel/data/db/xpanel.db "select key,value from settings where key like 'Fleet%';"
```

## 上报内容

每次注册和心跳会上报：

- 面板版本、commit、构建时间、Go 版本。
- 主机名、系统发行版、内核、架构、时区、虚拟化类型、TCP 拥塞控制。
- CPU 型号、物理核、逻辑核。
- 内存、交换分区、根分区容量与使用量。
- CPU 使用率、负载、TCP/UDP 连接数、进程数。
- 网络速率和累计流量。
- 网卡地址列表。

Fleet Center 使用这些数据更新实例快照，并记录监控采样。

## 网卡地址上报

Reporter 会从系统网卡中采集 IPv4 / IPv6 地址，随 `host.networkInterfaces` 字段上报给 Fleet Center。该字段用于在 Fleet Center 详情页直接看到 NetBird、NAT 内网、VPC 内网或公网地址，方便从内网访问机器。

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

Fleet Center 不直接连接节点，所有任务都由 X-Panel 主动拉取：

```text
X-Panel -> POST /api/v1/fleet/tasks/poll
X-Panel -> POST /api/v1/fleet/tasks/report
```

当前支持的任务类型：

| 类型 | 行为 |
|---|---|
| `tail_panel_log` | 拉取 X-Panel 系统日志 |
| `run_command` | 使用 `/bin/bash` 或 `/bin/sh` 执行命令 |
| `open_shell` | 反连 Fleet Center，建立远程终端 |
| `panel_check_update` | 调用 X-Panel 升级服务检查更新 |
| `panel_upgrade` | 调用 X-Panel 升级服务执行升级 |

节点不需要暴露公网 API，只需要能主动访问 `FleetEndpoint`。

## 常见排查

确认 Fleet Center 基础入口可达：

```bash
curl -v https://fcapi.qm.mk/api/v1/health
```

查看 Reporter 日志：

```bash
grep -i "fleet reporter" /opt/xpanel/data/log/xpanel.log | tail -50
```

常见日志含义：

| 日志 | 含义 |
|---|---|
| `register failed` | 首次注册失败，检查 `FleetEndpoint`、DNS、TLS、Nginx 反代 |
| `heartbeat failed` | 心跳失败，检查 token、实例是否存在、服务端状态 |
| `poll tasks failed` | 任务轮询失败，检查 `/api/v1/fleet/tasks/poll` 是否能返回 |
| `invalid instance token` | 本地 `FleetInstanceToken` 与 Fleet Center 不一致 |
| `instance not registered` | Fleet Center 没有对应 `FleetInstanceID` |

如果重装面板后 Fleet Center 出现两个同名节点，通常是新的安装生成了新的 `FleetInstanceID`。保留最新在线实例，删除旧的离线实例即可。

如需强制重新注册：

```bash
sqlite3 /opt/xpanel/data/db/xpanel.db "update settings set value='' where key='FleetInstanceToken';"
systemctl restart xpanel
```

谨慎修改 `FleetInstanceID`。该字段代表节点身份，改动后 Fleet Center 会认为这是新实例。
