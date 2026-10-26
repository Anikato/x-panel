> 本文件与开发目录上级的 `README.md` 相同。四个仓库的现行地址以 [repository-map.md](repository-map.md) 为准。

# X-Panel 开发四件套

这是一份**拷贝**，用于带到另一台机器继续开发。原项目仍在原路径，没有移动或删除。

## 目录

| 目录 | 角色 | 原路径 |
|---|---|---|
| `x-panel/` | 面板本体 | `/Users/kevin/Data/Project/panel/x-panel` |
| `agent/` | 定制 Nezha Agent | `/Users/kevin/Data/Project/Nezha-Server/agent` |
| `nezha/` | 定制 Dashboard 后端 | `/Users/kevin/Data/Project/Nezha-Server/nezha` |
| `admin-frontend/` | 定制管理后台前端 | `/Users/kevin/Data/Project/Nezha-Server/admin-frontend` |

`XPANEL_COLLABORATION.md` 已改成相对本目录的路径。各仓库内部的 `docs/superpowers/` 仍是历史实施记录，里面的 `/Users/kevin/Data/Project/...` 不要当现行命令执行。

## 打包时故意没带

- `node_modules`、`1Panel/`、各仓库 `artifacts/`、`x-panel/frontend/dist`
- `panel/fleet-center`（已退役）
- 哪吒用户主题源码（`aobobo` / `nazhua` 等）
- 已编译的 `nezha/cmd/dashboard/admin-dist` 和 `user-dist`。管理后台源码在 `admin-frontend/`，构建后再放进这两个目录

## 现行仓库

四个目录都在私有 Forgejo 上，对照表在 `x-panel/docs/repository-map.md`。

| 目录 | 仓库 | 分支 |
| --- | --- | --- |
| `x-panel/` | https://git.qm.mk/kevin/x-panel | `main` |
| `agent/` | https://git.qm.mk/kevin/agent | `main` |
| `nezha/` | https://git.qm.mk/kevin/nezha | `master` |
| `admin-frontend/` | https://git.qm.mk/kevin/admin-frontend | `main` |

GitHub 上的 `origin` 仍是上游或公开仓库。推这套私有副本用各仓库的 `forgejo` 远程。哪吒的 IP 国家库在 `nezha/pkg/geoip/geoip.db`，克隆后编译会嵌进 Dashboard。

## 新机器上手

需要 Go 1.24+、Node 18+、npm。

```bash
# X-Panel
cd x-panel/backend && go run cmd/server/main.go
cd x-panel/frontend && npm ci && npm run dev

# Dashboard 后端
cd nezha && go run ./cmd/dashboard

# 管理后台前端
cd admin-frontend && npm ci && npm run dev
```

完整联调（Agent systemd、批量升级）需要 Linux。

## 文档怎么用

现行只看：

- `XPANEL_COLLABORATION.md`
- `x-panel/AGENTS.md`、`x-panel/RELEASE.md`
- `x-panel/docs/dashboard-agent-xpanel.md`、`x-panel/docs/nezha-agent.md`
- `x-panel/docs/ui-design-system.md`、`x-panel/docs/theme-field-contract.md`
- `x-panel/docs/worklog.md`、`x-panel/docs/quick-start.md`

其余 `docs/superpowers/`、handoff、旧设计稿当历史，不当规范。

打包日期：2026-09-19
