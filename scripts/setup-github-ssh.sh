#!/bin/bash
# GitHub SSH 配置脚本

set -e

echo "🔧 配置 GitHub SSH 连接..."

# 1. 创建 .ssh 目录
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# 2. 复制私钥到 ~/.ssh/id_rsa
if [ -f "main_id_rsa" ]; then
    cp main_id_rsa ~/.ssh/id_rsa
    chmod 600 ~/.ssh/id_rsa
    echo "✅ SSH 私钥已配置"
else
    echo "❌ 错误: 找不到 main_id_rsa 文件"
    exit 1
fi

# 3. 配置 SSH known_hosts
if [ ! -f ~/.ssh/known_hosts ] || ! grep -q "github.com" ~/.ssh/known_hosts 2>/dev/null; then
    ssh-keyscan github.com >> ~/.ssh/known_hosts 2>/dev/null || {
        echo "Host github.com" >> ~/.ssh/config
        echo "  StrictHostKeyChecking no" >> ~/.ssh/config
        chmod 600 ~/.ssh/config
    }
    echo "✅ GitHub SSH 主机密钥已配置"
fi

# 4. 测试 SSH 连接
echo "🔍 测试 SSH 连接..."
if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
    echo "✅ SSH 连接测试成功！"
else
    echo "⚠️  SSH 连接测试失败，请检查："
    echo "   1. 公钥是否已添加到 GitHub (Settings → SSH and GPG keys)"
    echo "   2. 私钥文件权限是否正确 (chmod 600 ~/.ssh/id_rsa)"
    exit 1
fi

echo ""
echo "🎉 GitHub SSH 配置完成！"
echo ""
echo "现在可以执行："
echo "  git remote add origin git@github.com:Anikato/x-panel.git"
echo "  git push -u origin main"
