#!/bin/bash
# 菜单重构验证脚本

set -e

echo "=========================================="
echo "  X-Panel v0.3.2 菜单重构验证"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_file() {
    local file=$1
    local desc=$2
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $desc: $file"
        return 0
    else
        echo -e "${RED}✗${NC} $desc: $file (文件不存在)"
        return 1
    fi
}

check_content() {
    local file=$1
    local pattern=$2
    local desc=$3
    
    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $desc"
        return 0
    else
        echo -e "${RED}✗${NC} $desc (未找到)"
        return 1
    fi
}

echo "1. 检查前端文件..."
echo "-------------------"
check_file "frontend/src/layout/components/Sidebar.vue" "侧边栏组件"
check_file "frontend/src/i18n/zh.ts" "国际化文件"
check_file "frontend/src/api/modules/haproxy.ts" "HAProxy API"
check_file "frontend/src/views/haproxy/stats/index.vue" "HAProxy 统计页面"
echo ""

echo "2. 检查菜单结构..."
echo "-------------------"
check_content "frontend/src/layout/components/Sidebar.vue" "menu.network" "网络服务菜单"
check_content "frontend/src/layout/components/Sidebar.vue" "/haproxy/status" "HAProxy 路由保持不变"
check_content "frontend/src/layout/components/Sidebar.vue" "/gost/status" "GOST 路由保持不变"
check_content "frontend/src/layout/components/Sidebar.vue" "/traffic" "流量统计路由保持不变"
echo ""

echo "3. 检查国际化..."
echo "-------------------"
check_content "frontend/src/i18n/zh.ts" "network: '网络服务'" "网络服务翻译"
check_content "frontend/src/i18n/zh.ts" "pleaseInstallFirst" "HAProxy 未安装提示"
check_content "frontend/src/i18n/zh.ts" "goToInstall" "前往安装按钮"
echo ""

echo "4. 检查 Bug 修复..."
echo "-------------------"
check_content "frontend/src/api/modules/haproxy.ts" "pageSize: 100" "证书分页参数修复"
check_content "frontend/src/views/haproxy/stats/index.vue" "notInstalledError" "HAProxy 未安装检测"
check_content "backend/app/service/haproxy_runtime.go" "isHAProxyInstalled()" "后端安装状态检查"
echo ""

echo "5. 检查后端文件..."
echo "-------------------"
check_file "backend/app/service/haproxy_runtime.go" "HAProxy Runtime 服务"
check_file "backend/i18n/lang/zh.yaml" "后端国际化"
echo ""

echo "6. 检查文档..."
echo "-------------------"
check_file "docs/menu-refactor-v0.3.2.md" "菜单重构说明"
check_file "docs/CHANGELOG-v0.3.2.md" "更新日志"
echo ""

echo "=========================================="
echo -e "${GREEN}验证完成！${NC}"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 编译前端: cd frontend && npm run build"
echo "2. 编译后端: cd backend && go build -o ../xpanel cmd/server/main.go"
echo "3. 重启服务: systemctl restart xpanel"
echo "4. 访问面板验证菜单结构"
echo ""
