# GitHub 仓库设置指南

本文档指导如何将 X-Panel 项目提交到 GitHub。

## 📋 前置准备

1. **GitHub 账号**：确保已有 GitHub 账号
2. **Git 配置**：配置用户名和邮箱（如果还没配置）

```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

## 🚀 步骤 1：初始化 Git 仓库

如果还没有初始化，执行：

```bash
cd /data/X-Panel

# 初始化仓库
git init

# 将默认分支改为 main（推荐）
git branch -M main
```

## 📝 步骤 2：添加文件并创建初始提交

```bash
# 添加所有文件（.gitignore 会自动排除不需要的文件）
git add .

# 查看将要提交的文件
git status

# 创建初始提交
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

## 🌐 步骤 3：在 GitHub 创建仓库

1. 登录 GitHub
2. 点击右上角 **+** → **New repository**
3. 填写仓库信息：
   - **Repository name**: `x-panel`（或你喜欢的名字）
   - **Description**: `现代化的 Linux 服务器管理面板`
   - **Visibility**: 选择 Public 或 Private
   - **不要**勾选 "Initialize this repository with a README"（因为本地已有）
4. 点击 **Create repository**

## 🔗 步骤 4：连接远程仓库并推送

GitHub 创建仓库后，会显示推送命令。执行：

```bash
# 添加远程仓库（替换 YOUR_USERNAME 为你的 GitHub 用户名）
git remote add origin https://github.com/YOUR_USERNAME/x-panel.git

# 或者使用 SSH（如果已配置 SSH 密钥）
# git remote add origin git@github.com:YOUR_USERNAME/x-panel.git

# 推送代码到 GitHub
git push -u origin main
```

如果使用 HTTPS，GitHub 会要求输入用户名和 Personal Access Token（不是密码）。

### 创建 Personal Access Token

如果还没有 Token：

1. GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. 点击 **Generate new token (classic)**
3. 设置权限：
   - ✅ `repo`（完整仓库访问权限）
4. 生成后**复制 Token**（只显示一次）
5. 推送时，用户名输入 GitHub 用户名，密码输入 Token

## ✅ 步骤 5：验证

推送成功后，访问 `https://github.com/YOUR_USERNAME/x-panel` 查看仓库。

## 📌 后续提交

日常开发后提交代码：

```bash
# 查看修改
git status

# 添加修改的文件
git add .

# 或者添加特定文件
git add backend/app/service/website.go

# 提交
git commit -m "feat: 添加网站管理功能"

# 推送到 GitHub
git push
```

## 🔄 分支管理建议

### 主分支策略

- `main`：稳定版本，用于生产环境
- `develop`：开发分支（可选）
- `feature/*`：功能分支

### 创建功能分支

```bash
# 创建并切换到新分支
git checkout -b feature/website-management

# 开发完成后合并到 main
git checkout main
git merge feature/website-management
git push
```

## 📋 提交信息规范

建议使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）**：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具变更

**示例**：
```bash
git commit -m "feat(website): 实现 Nginx 站点创建功能"
git commit -m "fix(ssl): 修复证书续签失败问题"
git commit -m "docs: 更新开发指南"
```

## 🛡️ .gitignore 说明

项目已配置 `.gitignore`，会自动排除：

- `node_modules/` - 前端依赖
- `frontend/dist/` - 前端构建产物
- `backend/data/` - 本地开发数据
- `backend/xpanel` - 编译后的二进制
- `1Panel/` - 参考代码目录
- IDE 配置文件

## 🚨 常见问题

### 1. 推送被拒绝

如果远程仓库有 README 等文件，需要先拉取：

```bash
git pull origin main --allow-unrelated-histories
# 解决冲突后
git push -u origin main
```

### 2. 忘记添加 .gitignore

如果已经提交了不应该提交的文件：

```bash
# 从 Git 中删除但保留本地文件
git rm --cached -r node_modules/
git commit -m "chore: 更新 .gitignore"
git push
```

### 3. 撤销最后一次提交

```bash
# 保留修改
git reset --soft HEAD~1

# 丢弃修改（谨慎使用）
git reset --hard HEAD~1
```

## 📚 更多资源

- [Git 官方文档](https://git-scm.com/doc)
- [GitHub 文档](https://docs.github.com/)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

**提示**：首次推送可能需要几分钟，取决于文件大小和网络速度。
