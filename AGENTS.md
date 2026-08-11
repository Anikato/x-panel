# X-Panel 项目规则

## 必读文档

- 发布或修改 `.github/workflows/release.yml` 前，必须阅读 `RELEASE.md`。
- 修改 X-Panel、Dashboard 或定制 Agent 的职责边界前，必须阅读 `docs/dashboard-agent-xpanel.md`。
- `RELEASE.md` 是正式发布操作的唯一权威来源；`docs/superpowers/specs` 和 `docs/superpowers/plans` 只记录历史设计与实施背景。

## 不可违反的约束

- 正式发布只允许通过 `Build & Release` 的 `workflow_dispatch` 入口执行。
- 不得手动创建正式版本标签、覆盖同版本 Release，或直接修改线上 `releases/latest.json`。
- Agent 发生变化时，先发布可追溯的 Agent Release，再发布新的 X-Panel 补丁版本。
- 发布必须同时验证 linux/amd64、linux/arm64、SHA256 和 `xpanel.qm.mk` 公网下载。
- 保护用户现有修改；未经明确授权，不执行 commit、push、tag、Release 或生产部署。
