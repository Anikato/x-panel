# Phase 1：健康快照契约与威胁模型

> 状态：实现契约
> Schema：`health.snapshot` v1
> 更新日期：2026-10-30

本文定义 [`product-roadmap.md`](product-roadmap.md) Phase 1 的 X-Panel 节点健康快照。Dashboard 只能通过 Agent 执行固定命令 `/opt/xpanel/xpanel invoke` 获取该快照，不得拼接其他命令或读取 X-Panel 数据库文件。

## 调用契约

标准输入：

```json
{"capability":"health.snapshot","payload":{}}
```

标准输出继续使用 `xpanel invoke` 信封：

```json
{
  "ok": true,
  "capability": "health.snapshot",
  "generated_at": "2026-10-29T08:00:00Z",
  "data": {
    "schema_version": 1,
    "panel": {},
    "services": [],
    "websites": {},
    "certificates": {},
    "backups": {},
    "cronjobs": {},
    "storage": [],
    "security": {}
  }
}
```

`payload` 省略或为 `{}` 时有效；任何其他字段均返回 `invalid_request`。旧版本不支持该能力时返回 `unknown_capability`，Dashboard 必须显示“不支持”，不能显示为节点故障。

## 字段语义

### `panel`

- `version`、`commit_hash`、`build_time`：当前二进制构建信息。
- `started_at`：从固定 `xpanel.service` 的 systemd 激活时间转换得到的 UTC 时间；无法读取时省略。
- `database_readable`：快照数据库已成功以只读模式打开。

### `services[]`

首版只探测固定 unit：`xpanel.service`、`xpanel-nezha-agent.service`、`nginx.service`、`docker.service`。

- `name`：稳定逻辑名称，不接受调用方输入。
- `unit`：固定 systemd unit。
- `active_state`：systemd `ActiveState`，探测失败为 `unknown`。
- `unit_file_state`：systemd `UnitFileState`，探测失败为 `unknown`。

服务探测失败不能阻断数据库快照生成。

### `websites`

- 网站总数、运行数量、停止数量和启用 SSL 的数量。
- 不返回站点目录、反向代理地址、Basic Auth 用户或密码、自定义 Nginx 内容。

### `certificates`

- 证书总数、已过期数量、7 天内过期数量、15 天内过期数量和错误状态数量。
- 7 天内过期同时包含于 15 天内过期统计。
- 零值过期时间不计入过期风险。
- 不返回 PEM、私钥、ACME 私钥、DNS 授权信息、序列号或完整证书对象。

### `backups`

- 最近成功和失败时间、最近 48 小时成功/失败数量。
- 最多返回 20 条最近失败对象，只包含类型、名称和时间。
- 不返回错误正文、备份账户、Bucket、访问密钥、远程路径或加密密码。

### `cronjobs`

- 任务总数、启用数量和最近 24 小时失败次数。
- 最多扫描 1000 条最近执行记录，最多返回 20 个连续失败任务。
- 连续失败项只包含任务 ID、名称、连续失败次数和最后失败时间。
- 不返回脚本、URL、前后置命令、环境变量、执行输出或错误正文。

### `storage[]`

- 首版只采集根文件系统和 X-Panel 数据目录所在文件系统。
- 返回逻辑标签、容量、可用空间、使用率和 Inode 使用率。
- 不返回自定义安装路径或目录内容。

### `security`

- 仅返回数据库文件与数据目录权限是否安全，以及稳定问题码列表。
- 数据库文件存在组/其他用户写权限时报告 `database_permissions`。
- 数据目录存在其他用户写权限时报告 `data_dir_permissions`。
- 不返回真实路径、所有者名称、配置内容或任何凭据。

## 时间与限制

- 所有时间输出为 UTC RFC3339。
- 最近失败列表最多 20 条。
- 计划任务记录最多扫描 1000 条，并通过 `records_truncated` 声明可能截断。
- 快照应在 Agent 的 30 秒任务超时内完成。
- Dashboard 必须对响应设置 1 MiB 上限，并拒绝包含 PEM 标记的响应。

## 威胁模型

### 数据库被并发写入

`xpanel invoke` 使用 SQLite 只读连接、`PRAGMA query_only = ON` 和有限 `busy_timeout`。数据库忙返回稳定错误，不能切换为读写模式重试。

### 敏感信息外泄

实现使用专用输出 DTO 和明确列选择，不序列化业务模型。测试必须把凭据、私钥和错误正文写入测试数据库，并确认 JSON 中不存在这些哨兵值及以下标记：

- `BEGIN CERTIFICATE`
- `PRIVATE KEY`
- `agent_secret`
- `credential`
- `encrypt_password`

### 命令注入

调用方不能传 service 名称、路径或命令参数。systemd 探测列表写死在二进制中，并使用 `exec.CommandContext` 参数数组调用。

### 资源耗尽

所有明细列表和记录扫描均有上限；不读取日志文件、备份文件或证书内容；服务探测具有短超时。

### 旧版本兼容

Dashboard 按 `schema_version` 解析。未知高版本必须显示“不支持的快照版本”，不得尝试猜测字段。旧 X-Panel 返回 `unknown_capability` 时不得触发故障通知。

## Dashboard 评估边界

X-Panel 只提供事实数据，不决定最终告警等级。证书、备份、任务、磁盘和数据过期阈值由 Dashboard 统一评估，以便全局配置、去重、静默和恢复通知。
