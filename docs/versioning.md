# 版本号与发版（与 CI 一致）

本文档为仓库内**可跟踪**的版本说明；本地 Cursor 规则中的版本段落应与本文件保持一致。

## 当前约定

- **Git tag 格式**：`vMAJOR.MINOR.PATCH`（**三段**），例如 `v0.7.5`。与 GitHub Release、安装包文件名、二进制内 `version.Version` 一致。
- **已废弃**：历史上曾使用四段式 tag（如 `v0.6.0.2`），**新发布请勿再使用四段式**。
- **下一版本号**：以 [GitHub Releases](https://github.com/Anikato/x-panel/releases) 或 `git tag -l 'v*' | sort -V | tail -1` 的**当前最新 tag** 为基准，常规迭代递增 **PATCH**（例如 `v0.7.5` → `v0.7.6`）。

## 触发自动构建与发布

工作流：`.github/workflows/release.yml`，在推送匹配 `v*` 的 tag 时构建并创建 Release。

### 方式一：`gh`（本机已安装时推荐）

```bash
# 1. 默认分支已包含待发改动
git push origin main

# 2. 在最新提交上创建 Release（会创建并推送同名 tag，触发 CI）
gh release create v0.7.6 --generate-notes --title "v0.7.6"
```

将 `v0.7.6` 换成你实际要发的、且比当前最新 tag 递增后的版本号。

### 方式二：纯 Git

```bash
git push origin main
git tag v0.7.6
git push origin v0.7.6
```

## 安装脚本

`scripts/install-online.sh` 的 `--version` 必须为 Releases 上**已存在**的 tag，例如 `--version v0.7.6`。
