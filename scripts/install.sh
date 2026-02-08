#!/bin/bash
set -e

# X-Panel 安装脚本
INSTALL_DIR="/opt/xpanel"
SERVICE_NAME="xpanel"
CONFIG_FILE="$INSTALL_DIR/config.yaml"

echo "=============================="
echo "  X-Panel 安装脚本"
echo "=============================="

# 检查 root 权限
if [ "$(id -u)" -ne 0 ]; then
    echo "错误：请使用 root 用户运行此脚本"
    exit 1
fi

# 创建安装目录
echo ">>> 创建安装目录: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data/db"
mkdir -p "$INSTALL_DIR/data/log"

# 复制二进制
if [ -f "./xpanel" ]; then
    echo ">>> 安装主程序..."
    cp -f ./xpanel "$INSTALL_DIR/xpanel"
    chmod +x "$INSTALL_DIR/xpanel"
else
    echo "错误：当前目录下未找到 xpanel 二进制文件"
    exit 1
fi

# 复制配置文件（不覆盖已有配置）
if [ ! -f "$CONFIG_FILE" ]; then
    if [ -f "./config.yaml.example" ]; then
        echo ">>> 创建默认配置文件..."
        cp ./config.yaml.example "$CONFIG_FILE"
        # 生成随机 JWT Secret
        JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)
        sed -i "s/dev-secret-change-in-production/$JWT_SECRET/" "$CONFIG_FILE"
        # 设置生产模式
        sed -i 's/mode: "debug"/mode: "release"/' "$CONFIG_FILE"
        # 设置数据目录为绝对路径
        sed -i "s|data_dir: \"./data\"|data_dir: \"$INSTALL_DIR/data\"|" "$CONFIG_FILE"
    fi
else
    echo ">>> 配置文件已存在，跳过"
fi

# 安装 systemd 服务
if [ -f "./xpanel.service" ]; then
    echo ">>> 安装 systemd 服务..."
    cp -f ./xpanel.service /etc/systemd/system/xpanel.service
    systemctl daemon-reload
    systemctl enable xpanel
fi

echo ""
echo "=============================="
echo "  X-Panel 安装完成!"
echo "=============================="
echo ""
echo "安装目录: $INSTALL_DIR"
echo "配置文件: $CONFIG_FILE"
echo ""
echo "启动命令:"
echo "  systemctl start xpanel"
echo ""
echo "查看状态:"
echo "  systemctl status xpanel"
echo ""
echo "访问面板:"
echo "  http://<服务器IP>:9999"
echo ""
