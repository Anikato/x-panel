# X-Panel 正式发布手册

本文档是 X-Panel 正式发布操作的唯一权威来源。架构和组件职责参见 [`docs/dashboard-agent-xpanel.md`](docs/dashboard-agent-xpanel.md)。历史 spec、plan 和聊天记录只能用于背景，不能替代本手册。

## 发布模型

- GitHub Actions 从已经推送到 GitHub 的源码构建和验证制品。
- GitHub Release 保存不可变的发布记录和安装包。
- 节点只通过 `https://xpanel.qm.mk` 检查和下载更新，不直接依赖 GitHub Release。
- 每个 X-Panel 安装包都包含该版本固定的定制兼容 Agent。
- X-Panel 或 Agent 任一二进制发生变化，都必须发布新的 X-Panel 版本。

正式发布的唯一入口是 GitHub Actions 中的 `Build & Release` 工作流（`workflow_dispatch`）。工作流根据输入版本创建标签；不要提前手动创建标签。

## 首次配置

在 GitHub 仓库的 Settings → Secrets and variables → Actions 中配置：

| 类型 | 名称 | 说明 |
| --- | --- | --- |
| Repository variable | `CUSTOM_AGENT_REPO` | 保存定制 Agent Release 的仓库，例如 `Anikato/x-panel` |
| Repository variable | `CUSTOM_AGENT_VERSION` | 固定的 Agent Release 标签，例如 `agent-v2.3.1-xpanel.1` |
| Repository variable | `UPDATE_SERVER_HOST` | 更新服务器 SSH 地址 |
| Repository variable | `UPDATE_SERVER_PORT` | SSH 端口，未设置时为 `22` |
| Repository variable | `UPDATE_SERVER_USER` | SSH 用户，未设置时为 `root` |
| Repository variable | `UPDATE_SERVER_PATH` | 发布目录，未设置时为 `/var/www/xpanel` |
| Repository secret | `UPDATE_SERVER_SSH_KEY` | Actions 部署更新服务器使用的私钥 |

Repository secret 是只写配置，AI、GitHub API 和普通仓库读取操作都不能取回其明文；工作流运行时可以使用它。

正式更新源固定为 `https://xpanel.qm.mk`，不是可配置 Repository variable。工作流会拒绝清单中不属于该域名和当前版本目录的包 URL。

## 源码仓库边界

当前本地 Agent 和 Dashboard 仓库的 `origin` 仍可能指向 `nezhahq` 官方仓库。发布定制源码前必须先确认远端属于用户控制的 fork，禁止尝试把定制提交推送到官方上游。

`CUSTOM_AGENT_REPO` 指向的是 Agent 制品所在的 GitHub Release 仓库，可以继续使用 `Anikato/x-panel`，不要求立即建立独立 Agent 仓库。若以后从 Agent 源码全自动发布，先配置用户控制的 Agent fork，再复用其 GitHub Actions。

## 发布普通 X-Panel 版本

1. 确认需要发布的源码已经进入目标 GitHub 仓库的正确提交。GitHub Actions 看不到只存在于本地工作树的修改。
2. 确认 `CUSTOM_AGENT_REPO` 和 `CUSTOM_AGENT_VERSION` 指向经过测试的 Agent 制品。
3. 打开 GitHub 仓库的 Actions 页面，选择 `Build & Release`。
4. 选择 Run workflow，输入新的版本号，例如 `v0.7.83`。
5. 等待工作流完成，不要同时运行第二个正式发布。

工作流依次执行：输入和资产预检、Shell/Go/前端测试、amd64/arm64 构建、Draft Release、更新服务器暂存、SHA256 校验、`latest.json` 激活、公网下载校验、正式发布 GitHub Release。公网校验或正式发布失败时会恢复旧更新清单，保留 Draft 供同版本重试。

## 只修改 Agent 时发布

Agent 没有独立的节点更新清单，也不允许绕过 X-Panel 自更新。只修改 Agent 时：

1. 从用户控制的 Agent 源码构建 `linux/amd64` 和 `linux/arm64`。
2. 创建新的 Agent Release，至少包含：

   ```text
   nezha-agent_linux_amd64.zip
   nezha-agent_linux_arm64.zip
   checksums.txt
   ```

3. 把 X-Panel 仓库变量 `CUSTOM_AGENT_VERSION` 更新为新 Agent 标签；仓库发生变化时同时更新 `CUSTOM_AGENT_REPO`。
4. 即使 X-Panel 业务代码没有变化，也发布新的 X-Panel 补丁版本。
5. 节点通过 `xpanel.qm.mk` 下载新的完整 X-Panel 包，并由 X-Panel 升级事务同时替换 Agent 和面板二进制。

不要重新上传或覆盖旧 Agent/X-Panel 版本；内容变化必须使用新版本号。

## 发布成功检查

工作流成功后检查：

```bash
curl -fsSL https://xpanel.qm.mk/releases/latest.json
```

清单必须满足：

- `version` 等于本次输入版本；
- 同时存在 `linux-amd64` 和 `linux-arm64`；
- 每个架构都有 `url`、`checksumUrl`、`sha256` 和 `size`；
- URL 指向 `https://xpanel.qm.mk/releases/<version>/`。

工作流会从公网重新下载两个架构的包和 checksum 并执行 `sha256sum -c`。还应确认 GitHub Release 已从 Draft 变为正式状态，资产名称和更新服务器一致。

## 失败与重试

| 失败位置 | 在线节点状态 | 处理方式 |
| --- | --- | --- |
| 输入、测试或构建 | 更新源不变；没有正式 Release | 修复源码或配置后重新运行 |
| Draft 或资产上传 | 更新源不变；可能保留 Draft | 从原工作流运行点 Re-run；保持同一 commit、Agent 版本和制品字节 |
| SSH 上传或远端校验 | `latest.json` 保持旧版本 | 修复 SSH/磁盘/权限后重新运行 |
| 激活或公网校验 | 工作流恢复旧清单 | 检查服务器或网络后使用同版本 Draft 重试 |
| 正式发布 GitHub Release | 工作流恢复旧清单；Draft 保留 | 修复 GitHub 权限后使用同版本重试 |
| 正式发布成功后的暂存清理 | 新版本仍然有效 | 按工作流警告清理对应 `.staging` 目录 |

已经正式发布的版本不得覆盖。同版本只有仍为 Draft、制品未公开且仍使用同一 commit、同一 Agent 版本和相同制品字节时才允许 Re-run；不要从更新后的分支重新输入旧版本号。其他情况必须发布新补丁版本。

## 禁止操作

- 禁止手动创建正式版本标签来触发发布。
- 禁止覆盖同版本 GitHub Release 或 `/releases/<version>/` 制品。
- 禁止直接编辑生产服务器的 `releases/latest.json`。
- 禁止让节点在正常更新路径中直接下载 Agent GitHub Release。
- 禁止使用 `latest` 解析 Agent 版本；必须固定 `CUSTOM_AGENT_VERSION`。
- 禁止向 `nezhahq` 官方仓库推送本项目的定制源码。

运行工作流、commit、push、tag、Release、修改仓库变量和生产部署都是外部状态变更，必须获得用户明确授权。完成本地代码和文档不等于已经完成生产发布。
