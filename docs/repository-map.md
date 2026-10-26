# X-Panel 四件套

四个目录是一套，分别放在私有 Forgejo 仓库里。未登录打不开。

| 目录 | 仓库 | 分支 | 角色 |
| --- | --- | --- | --- |
| `x-panel/` | https://git.qm.mk/kevin/x-panel | `main` | 面板本体。安装升级、网站、证书，以及 Agent 的配置入口和 systemd 生命周期 |
| `agent/` | https://git.qm.mk/kevin/agent | `main` | 定制 Nezha Agent。独立进程，负责节点采集和与 Dashboard 的协议 |
| `nezha/` | https://git.qm.mk/kevin/nezha | `master` | Nezha Dashboard。节点监控、命令、终端和远程配置的中心控制面 |
| `admin-frontend/` | https://git.qm.mk/kevin/admin-frontend | `main` | Dashboard 管理后台前端。构建产物放到 `nezha/cmd/dashboard/admin-dist` |

X-Panel 管理组件，Agent 负责节点能力，Dashboard 负责中心控制。协作规则、配置契约和发布顺序见 [xpanel-collaboration.md](xpanel-collaboration.md)。组件边界的现行说明见 [dashboard-agent-xpanel.md](dashboard-agent-xpanel.md)。

各仓库的本地远程名是 `forgejo`。GitHub 上的 `origin` 仍指向原来的上游或公开仓库，推这套私有副本时用 `git push forgejo`。

## 换机器时会一起带走的内容

- 哪吒的 IP 国家库在 `nezha/pkg/geoip/geoip.db`。这是 IPinfo Lite 的位置库，约 24MB。`pkg/geoip/geoip.go` 用 `go:embed` 在编译 Dashboard 时把它打进二进制。克隆 `nezha` 就带有这个文件，不需要再下载。
- 已编译的 `nezha/cmd/dashboard/admin-dist` 和 `user-dist` 不在 Git 里。管理后台源码在 `admin-frontend`，构建后再放进哪吒的对应目录。

## 开发目录里的两份说明

和四个仓库并列的 `README.md`、`XPANEL_COLLABORATION.md` 已收入本仓库：

- [dev-suite-readme.md](dev-suite-readme.md)
- [xpanel-collaboration.md](xpanel-collaboration.md)
