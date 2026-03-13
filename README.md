# X-Panel

> 一个现代化的 Linux 服务器管理面板，参考 1Panel 重新实现

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Vue Version](https://img.shields.io/badge/vue-3.4+-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![Build](https://github.com/Anikato/x-panel/actions/workflows/release.yml/badge.svg)](https://github.com/Anikato/x-panel/actions)

## ✨ 特性

- 🚀 **现代化技术栈**：Go 1.24+ / Gin / GORM / SQLite + Vue 3 / TypeScript / Element Plus
- 🌐 **网站管理**：Nginx 站点管理（反向代理/静态站点/重定向）
- 🔒 **SSL 证书**：ACME 自动申请（Let's Encrypt/ZeroSSL 等），支持 7+ DNS 提供商
- 📁 **文件管理**：多标签浏览、代码编辑、权限管理
- 💻 **Web 终端**：本地 PTY + SSH 远程终端
- 📊 **系统监控**：CPU/内存/磁盘/网络实时监控
- 🔥 **防火墙管理**：ufw 端口/IP 规则管理
- 🐳 **容器管理**：Docker 容器/镜像/网络/存储卷/Compose 编排
- 🗄️ **数据库管理**：MySQL/MariaDB、PostgreSQL 数据库管理
- ⏱️ **计划任务**：Shell 脚本、URL 请求、网站/数据库/目录定时备份
- 💾 **备份系统**：支持本地、S3、SFTP、WebDAV 多种备份存储
- 🖥️ **面板集群**：一个主面板管理多台服务器（Agent 模式）
- 🛡️ **安全防护**：登录失败自动触发验证码，防暴力破解
- 🛠️ **系统工具**：SSH 配置、进程管理、磁盘管理
- 🔄 **自动更新**：GitHub Releases 自动检查更新，一键升级
- 🎨 **暗色主题**：科技风 UI 设计

## 🚀 一键安装

在 Linux 服务器上以 root 用户执行：

```bash
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash
```

默认启用 HTTPS（自签名证书），安装后访问 `https://服务器IP:7777`。

### 自定义安装

```bash
# 自定义端口 + 安全入口（推荐）
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --port 8443 --entrance mySecret123

# 自定义安装路径
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --path /usr/local/xpanel

# 禁用 HTTPS（使用 HTTP）
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --no-ssl

# 安装指定版本
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --version v1.0.0

# 作为 Agent 节点安装（被主面板管理）
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --agent-token YOUR_SECRET_TOKEN
```

| 参数 | 说明 |
|------|------|
| `--port, -p <端口>` | 自定义面板端口（默认 7777） |
| `--path <路径>` | 自定义安装路径（默认 /opt/xpanel） |
| `--entrance, -e <路径>` | 安全入口路径，防止面板被扫描到 |
| `--ssl` / `--no-ssl` | 启用/禁用 HTTPS（默认启用） |
| `--version, -v <版本>` | 安装指定版本 |
| `--agent-token <密钥>` | 设置 Agent Token，用于被主面板管理 |

### 卸载

```bash
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash -s -- --uninstall --yes
```

> **系统要求**：Linux (amd64 / arm64)
> 
> **安全入口**：设置后面板只能通过 `https://IP:端口/入口路径` 访问，直接访问根路径返回 404

## 🖥️ 面板集群（多服务器管理）

X-Panel 支持 **Agent 模式**，一台主面板可以管理多台远程服务器。

### 架构说明

```
┌─────────────────┐
│   主面板 (Hub)    │── 管理所有节点
│   服务器 A        │
└────┬────┬────┬──┘
     │    │    │    API Proxy (X-Node-ID / X-Agent-Token)
     ▼    ▼    ▼
┌────┴┐ ┌┴───┐ ┌┴───┐
│ 节点B │ │节点C│ │节点D│  ← 每台都运行 X-Panel (Agent)
└─────┘ └────┘ └────┘
```

- **主面板**：正常安装的 X-Panel，通过「节点管理」添加和管理远程节点
- **Agent 节点**：安装了 X-Panel 并配置了 Agent Token 的远程服务器
- 主面板通过 API 代理与 Agent 通信，在顶部导航栏切换节点后，所有操作都会代理到对应节点

### 部署步骤

#### 步骤 1：部署主面板

在主服务器上正常安装 X-Panel：

```bash
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash
```

安装完成后通过浏览器访问，完成初始化设置。

#### 步骤 2：部署 Agent 节点

在每台需要被管理的远程服务器上安装 X-Panel，并设置 Agent Token：

```bash
# 方式一：安装时指定 Agent Token（推荐）
curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh \
  | bash -s -- --agent-token YOUR_SECRET_TOKEN

# 方式二：先正常安装，再通过面板设置 Agent Token
# 安装后进入 设置 → Agent 节点设置 → 填写通信密钥
```

> **注意**：Agent Token 是主面板与 Agent 节点之间的通信凭证，请使用足够复杂的密钥。多个节点可以使用相同或不同的 Token。

#### 步骤 3：在主面板添加节点

1. 登录主面板
2. 进入左侧菜单「节点管理」
3. 点击「添加节点」
4. 填写：
   - **名称**：自定义节点名（如「美国 VPS」）
   - **地址**：Agent 节点的 `IP:端口`（如 `203.0.113.50:7777`）
   - **通信密钥**：与 Agent 节点设置的 Agent Token 一致
5. 点击「测试连接」验证连通性
6. 保存

#### 步骤 4：使用

添加节点后，在顶部导航栏的节点下拉菜单中切换目标节点。切换后，面板中的所有操作（网站管理、终端、文件管理、容器管理等）都会作用于选中的远程节点。

选择「本机」则回到管理主面板本机。

## 🏗️ 架构

```
┌──────────────────────────────────────────┐
│                 Browser                   │
└────────────────┬─────────────────────────┘
                 │ HTTPS
┌────────────────▼─────────────────────────┐
│            X-Panel Server                  │
│  ┌─────────────────────────────────────┐ │
│  │          API Gateway (Gin)           │ │
│  │  ├── JWT 认证中间件                  │ │
│  │  ├── 节点代理中间件 (Agent Proxy)    │ │
│  │  ├── 操作日志中间件                  │ │
│  │  └── CORS / Rate Limit             │ │
│  └──────────────┬──────────────────────┘ │
│  ┌──────────────▼──────────────────────┐ │
│  │         Service Layer               │ │
│  │  ├── WebsiteService                 │ │
│  │  ├── SSLService                     │ │
│  │  ├── ContainerService               │ │
│  │  ├── DatabaseService                │ │
│  │  ├── CronjobService                 │ │
│  │  ├── BackupService                  │ │
│  │  ├── NodeService                    │ │
│  │  └── ...                            │ │
│  └──────────────┬──────────────────────┘ │
│  ┌──────────────▼──────────────────────┐ │
│  │        Repository Layer             │ │
│  │  └── GORM + SQLite                 │ │
│  └─────────────────────────────────────┘ │
└──────────────────────────────────────────┘
```

## 🔧 常用命令

```bash
systemctl start xpanel       # 启动
systemctl stop xpanel        # 停止
systemctl restart xpanel     # 重启
systemctl status xpanel      # 查看状态
journalctl -u xpanel -f      # 查看日志
```

## 📦 从源码构建

### 前置要求

- Go 1.24+
- Node.js 18+
- Linux 系统（推荐 Debian/Ubuntu）

### 构建

```bash
# 克隆仓库
git clone https://github.com/Anikato/x-panel.git
cd x-panel

# 完整构建（前端 + 后端）
make build

# 打包发布
make package

# 安装
cd build && sudo tar -xzf xpanel-*.tar.gz && sudo bash install.sh
```

### 开发模式

```bash
# 后端（端口 7777）
cd backend
go run cmd/server/main.go

# 前端（端口 5173）
cd frontend
npm install
npm run dev
```

## 📚 文档

- [开发指南](docs/development-guide.md) - 详细的架构设计和开发规范
- [工作日志](docs/worklog.md) - 开发进度记录
- [进度分析](docs/progress-analysis.md) - 项目完成度分析

## 🎯 功能模块

### ✅ 已完成

- [x] 用户认证（登录/JWT/初始化/验证码防暴力破解）
- [x] SSL 证书管理（ACME + DNS 验证）
- [x] 文件管理（多标签/导航/搜索/编辑/权限）
- [x] Web 终端（本地 + SSH）
- [x] 系统监控（CPU/内存/磁盘/网络）
- [x] 防火墙管理（ufw）
- [x] SSH 管理
- [x] 进程/磁盘管理
- [x] Nginx 管理（安装/状态/操作）
- [x] 主机管理 + 快速命令
- [x] 构建系统 + 自更新（GitHub Releases）
- [x] 一键安装脚本
- [x] 计划任务（Shell/URL/网站备份/数据库备份/目录备份）
- [x] 数据库管理（MySQL/MariaDB + PostgreSQL）
- [x] 容器管理（Docker 容器/镜像/网络/存储卷/Compose）
- [x] 备份系统（本地/S3/SFTP/WebDAV）
- [x] 面板集群（Agent 模式，多服务器管理）

### 🚧 开发中

- [ ] 网站管理（Nginx 站点 CRUD）
- [ ] 工具箱（FTP/Fail2ban/ClamAV）

## 🔑 与 1Panel 的关键差异

| 方面 | 1Panel | X-Panel |
|------|--------|---------|
| Nginx 部署 | Docker 容器 | 本地安装管理 |
| 架构 | Core + Agent（双进程） | 单进程（Hub + Agent 模式） |
| 应用商店 | ✅ | ❌ |
| 集群管理 | ✅（多 Agent） | ✅（API Proxy 模式） |
| 备份存储 | 本地/S3/SFTP/等 | 本地/S3/SFTP/WebDAV |

## 🛠️ 技术栈

### 后端
- **语言**：Go 1.24+
- **框架**：Gin
- **ORM**：GORM
- **数据库**：SQLite (glebarez/sqlite)
- **配置**：Viper
- **日志**：Logrus
- **SSL**：go-acme/lego/v4
- **系统监控**：shirou/gopsutil/v4
- **容器**：Docker SDK (docker/docker)
- **定时任务**：robfig/cron/v3
- **验证码**：mojocn/base64Captcha

### 前端
- **框架**：Vue 3 + TypeScript
- **UI 库**：Element Plus
- **状态管理**：Pinia
- **路由**：Vue Router 4
- **构建工具**：Vite 7
- **终端**：xterm.js 5
- **代码编辑**：Monaco Editor

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

本项目参考了 [1Panel](https://github.com/1Panel-dev/1Panel) 的设计和实现，在此表示感谢。

---

**注意**：本项目仍在积极开发中，API 和功能可能会有变化。
