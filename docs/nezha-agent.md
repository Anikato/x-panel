# X-Panel 捆绑哪吒 Agent 运维说明

X-Panel 发布包捆绑固定版本的官方 Nezha Agent，并以独立的 `xpanel-nezha-agent.service` 运行。Agent 不是 X-Panel 主进程的一部分；X-Panel 只负责安全配置、systemd 生命周期和随面板版本统一升级。

## 安全边界

- Dashboard 地址必须是无路径、查询参数、片段或用户信息的 HTTPS origin，例如 `https://dashboard.example.com`。
- AgentSecret 只写入 `/opt/xpanel/nezha-agent/config.yml`，面板和 API 只显示“已配置”，不会回显明文。
- 开启远程运维后，Dashboard 管理员可以通过 Agent 执行命令、终端、文件操作、配置下发和密钥轮换，等同于获得节点上的 root 级运维能力。只连接受信任并妥善保护的 Dashboard。
- 默认禁止 Agent 自更新和 Dashboard 强制更新。Agent 版本只能随经过校验的 X-Panel 发布包升级。
- X-Panel 不接管外部安装的哪吒 Agent。检测到外部 unit、进程或常见安装目录时，捆绑 Agent 的启动、重启和启用操作会被阻止；外部实例不会被停止、覆盖或导入。

## 安装时配置

交互安装时先无回显读取 AgentSecret，再通过环境变量交给安装器。不要把秘密写入命令参数、行内环境变量、Shell 历史或工单。

```bash
read -rsp 'Nezha AgentSecret: ' xpanel_nezha_secret && printf '\n'
export XPANEL_NEZHA_DASHBOARD_URL='https://dashboard.example.com'
export XPANEL_NEZHA_AGENT_SECRET="$xpanel_nezha_secret"

curl -fsSL https://xpanel.qm.mk/install-online.sh -o /tmp/xpanel-install-online.sh
sudo --preserve-env=XPANEL_NEZHA_DASHBOARD_URL,XPANEL_NEZHA_AGENT_SECRET \
  bash /tmp/xpanel-install-online.sh

unset xpanel_nezha_secret XPANEL_NEZHA_DASHBOARD_URL XPANEL_NEZHA_AGENT_SECRET
rm -f /tmp/xpanel-install-online.sh
```

安装器读取环境变量后会立即取消导出并清空原变量。非交互部署只应通过 CI、配置管理或秘密注入设施设置 `XPANEL_NEZHA_AGENT_SECRET`；不要使用 `sudo env SECRET=...`，因为这会让秘密进入进程参数。

也可以只安装 X-Panel，不提供 Dashboard 或 AgentSecret。此时 Agent 二进制和无凭据的 systemd unit 会安装，但服务保持禁用、停止且不联网。之后在面板左侧进入“哪吒 Agent”完成首次配置。

## 面板配置与生命周期

首次配置需要同时填写 Dashboard HTTPS 地址和 AgentSecret。修改已有配置时，AgentSecret 留空表示保留当前值。

管理页操作语义：

| 操作 | 行为 |
| --- | --- |
| 配置 | 原子更新 `config.yml`；运行中的服务会安全恢复运行 |
| 启动 | 只启动本次运行，不改变开机自启期望 |
| 停止 | 只停止当前进程，不关闭开机自启 |
| 重启 | 重新加载磁盘配置并重启服务 |
| 启用并启动 | 执行 systemd enable-and-start，并记录期望状态 |
| 完全禁用 | 执行 disable-and-stop，但保留二进制、配置和 UUID |

关闭“远程运维”后，监控连接仍可工作，但 Dashboard 下发的命令、终端、文件操作、ApplyConfig 和密钥轮换都会停止，而且只能从节点本机的 X-Panel 管理页重新开启。

## 文件与服务

| 项目 | 路径或名称 |
| --- | --- |
| 组件目录 | `/opt/xpanel/nezha-agent`，权限 `0700` |
| Agent 二进制 | `/opt/xpanel/nezha-agent/nezha-agent`，权限 `0755` |
| Agent 配置 | `/opt/xpanel/nezha-agent/config.yml`，权限 `0600` |
| systemd unit | `xpanel-nezha-agent.service` |

常用只读检查：

```bash
sudo systemctl status xpanel-nezha-agent
sudo systemctl is-enabled xpanel-nezha-agent
sudo journalctl -u xpanel-nezha-agent -n 100 --no-pager
```

优先通过 X-Panel 管理页执行生命周期操作，以保持面板期望状态与 systemd 一致。紧急情况下可直接运行：

```bash
sudo systemctl restart xpanel-nezha-agent
sudo systemctl stop xpanel-nezha-agent
```

## 配置同步与 UUID

健康的 `config.yml` 是 Agent 配置的事实来源。Dashboard 通过 ApplyConfig 或密钥轮换修改文件后，X-Panel 会从磁盘同步必要字段，不会用数据库旧值覆盖文件，也会保留未知 YAML 字段。

修改 Dashboard 地址、停止服务、完全禁用、重新启用或升级 X-Panel 都不会主动更换 UUID。不要删除 `config.yml` 或手工改写 UUID；切换 Dashboard 后，原 UUID 会作为同一节点身份连接新的 Dashboard。

## 冲突与故障排查

管理页显示“外部冲突”时，检查是否存在官方默认 unit、实例化 unit、外部进程或常见安装目录：

```bash
systemctl list-unit-files 'nezha-agent*.service'
systemctl list-units --type=service --all 'nezha-agent*.service'
pgrep -a nezha-agent
```

X-Panel 不会替你删除或停用外部实例。确认归属并自行处理冲突后，再回到管理页刷新状态并启动捆绑 Agent。

若页面显示配置损坏或权限警告，先检查文件类型和权限，不要在日志或截图中展示 AgentSecret：

```bash
sudo stat -c '%F %a %n' \
  /opt/xpanel/nezha-agent \
  /opt/xpanel/nezha-agent/nezha-agent \
  /opt/xpanel/nezha-agent/config.yml
```

管理页返回的 journal 会对已知 AgentSecret、认证元数据和 Authorization 值再次脱敏，但仍应按敏感运维日志管理。

## 升级行为

X-Panel 发布包同时包含面板和与之绑定的官方 Agent 资产。升级过程先校验 checksum，再按 Agent → X-Panel 顺序替换；任何阶段失败都会回滚对应二进制并恢复原服务状态。已有 `config.yml`、UUID 和启用状态不会被安装器覆盖。

不要单独替换 Agent 二进制、开启 Agent 自更新或从 Dashboard 强制升级，否则会破坏 X-Panel 发布版本的一致性与回滚保证。
