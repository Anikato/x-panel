#!/bin/bash
# 修复并推送脚本

set -e

echo "🔧 修复 Git 状态并推送到 GitHub..."

cd /data/X-Panel

# 1. 添加所有文件
echo "📝 添加文件..."
git add .

# 2. 创建初始提交
echo "💾 创建初始提交..."
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

# 3. 将分支重命名为 main（如果当前是 master）
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "master" ]; then
    echo "🔄 将分支从 master 重命名为 main..."
    git branch -M main
fi

# 4. 检查远程仓库是否已配置
if ! git remote get-url origin >/dev/null 2>&1; then
    echo "🔗 添加远程仓库..."
    git remote add origin git@github.com:Anikato/x-panel.git
else
    echo "✅ 远程仓库已配置"
fi

# 5. 推送到 GitHub
echo "⬆️  推送到 GitHub..."
git push -u origin main

echo ""
echo "✅ 推送完成！"
echo "🌐 查看仓库: https://github.com/Anikato/x-panel"
