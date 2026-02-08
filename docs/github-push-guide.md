# GitHub 推送指南 - Anikato/x-panel

> 针对你的仓库的快速推送指南

## 🎯 快速开始（推荐方式）

### 方式一：使用一键脚本（最简单）

```bash
cd /data/X-Panel

# 直接运行推送脚本（会自动处理所有步骤）
./scripts/push-to-github.sh
```

脚本会自动：
- ✅ 初始化 Git 仓库（如果还没初始化）
- ✅ 添加所有文件
- ✅ 创建初始提交
- ✅ 配置远程仓库
- ✅ 推送到 GitHub

### 方式二：手动执行（分步操作）

#### 步骤 1：配置 SSH（如果使用 SSH 方式）

```bash
cd /data/X-Panel

# 运行 SSH 配置脚本
./scripts/setup-github-ssh.sh
```

**重要**：如果使用 SSH，需要确保：
1. 公钥已添加到 GitHub：
   - 访问 https://github.com/settings/keys
   - 点击 "New SSH key"
   - 从 `main_id_rsa.pub` 复制公钥内容（如果没有，需要生成：`ssh-keygen -y -f main_id_rsa > main_id_rsa.pub`）

#### 步骤 2：初始化并提交

```bash
cd /data/X-Panel

# 初始化 Git（如果还没初始化）
git init
git branch -M main

# 配置 Git 用户信息（如果还没配置）
git config user.name "Anikato"
git config user.email "your.email@example.com"  # 替换为你的邮箱

# 添加文件
git add .

# 创建提交
git commit -m "feat: 初始提交 - X-Panel 服务器管理面板

- 后端框架：Go + Gin + GORM + SQLite
- 前端框架：Vue 3 + TypeScript + Element Plus
- 已完成功能：
  * 用户认证系统
  * SSL 证书管理（ACME + DNS 验证）
  * 文件管理（多标签/导航/搜索/编辑）
  * Web 终端（本地 + SSH）
  * 系统监控
  * 防火墙/SSH/进程管理
  * Nginx 管理基础功能
  * 构建系统 + 自更新"
```

#### 步骤 3：连接远程仓库并推送

**使用 SSH（推荐）**：
```bash
# 添加远程仓库
git remote add origin git@github.com:Anikato/x-panel.git

# 推送到 GitHub
git push -u origin main
```

**使用 HTTPS**：
```bash
# 添加远程仓库
git remote add origin https://github.com/Anikato/x-panel.git

# 推送到 GitHub（会要求输入用户名和 Personal Access Token）
git push -u origin main
```

## 🔐 SSH vs HTTPS

### SSH 方式（推荐）

**优点**：
- ✅ 无需每次输入密码
- ✅ 更安全
- ✅ 适合频繁推送

**配置步骤**：
1. 运行 `./scripts/setup-github-ssh.sh`
2. 确保公钥已添加到 GitHub
3. 使用 `git@github.com:Anikato/x-panel.git` 作为远程地址

### HTTPS 方式

**优点**：
- ✅ 配置简单
- ✅ 适合一次性操作

**缺点**：
- ❌ 每次推送需要输入 Personal Access Token

**配置步骤**：
1. 在 GitHub 创建 Personal Access Token：
   - Settings → Developer settings → Personal access tokens → Tokens (classic)
   - 生成新 Token，权限选择 `repo`
2. 推送时，用户名输入 `Anikato`，密码输入 Token

## ✅ 验证推送

推送成功后，访问：
**https://github.com/Anikato/x-panel**

你应该能看到：
- ✅ README.md
- ✅ 所有源代码文件
- ✅ 文档目录
- ✅ LICENSE 文件

## 🔄 后续推送

日常开发后推送代码：

```bash
# 查看修改
git status

# 添加修改
git add .

# 提交
git commit -m "feat: 添加新功能"

# 推送
git push
```

## 🚨 常见问题

### 1. SSH 连接失败

```bash
# 测试 SSH 连接
ssh -T git@github.com

# 如果失败，检查：
# 1. 公钥是否已添加到 GitHub
# 2. 私钥权限是否正确 (chmod 600 ~/.ssh/id_rsa)
```

### 2. 推送被拒绝

如果远程仓库已有内容（如 README），需要先拉取：

```bash
git pull origin main --allow-unrelated-histories
# 解决冲突后
git push -u origin main
```

### 3. 忘记添加 .gitignore

如果已经提交了不应该提交的文件：

```bash
# 从 Git 中删除但保留本地文件
git rm --cached -r node_modules/
git commit -m "chore: 更新 .gitignore"
git push
```

### 4. 私钥文件被提交

**重要**：`main_id_rsa` 文件已在 `.gitignore` 中，不会被提交。

如果意外提交了，立即删除：

```bash
# 从 Git 历史中删除
git rm --cached main_id_rsa
git commit -m "chore: 移除私钥文件"
git push

# 如果已推送到 GitHub，需要：
# 1. 在 GitHub 上删除该文件
# 2. 考虑重新生成 SSH 密钥对（因为私钥已泄露）
```

## 📝 提交信息规范

建议使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```bash
# 新功能
git commit -m "feat(website): 实现 Nginx 站点创建功能"

# 修复 bug
git commit -m "fix(ssl): 修复证书续签失败问题"

# 文档更新
git commit -m "docs: 更新开发指南"

# 重构
git commit -m "refactor(file): 重构文件服务层"
```

## 🎉 完成！

推送成功后，你的项目就正式在 GitHub 上了！

**仓库地址**：https://github.com/Anikato/x-panel

---

**提示**：
- 首次推送可能需要几分钟，取决于文件大小
- 确保 `.gitignore` 正确配置，避免提交敏感信息
- 定期推送代码，保持仓库同步
