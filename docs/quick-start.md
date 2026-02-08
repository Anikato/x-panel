# X-Panel 快速启动指南

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.24+ | 后端编译运行 |
| Node.js | 18+ | 前端开发服务 |
| npm | 9+ | 前端包管理 |

## 项目结构

```
X-Panel/
├── backend/          # Go 后端
│   ├── cmd/server/   # 入口 main.go
│   ├── configs/      # 配置文件 config.yaml
│   └── data/         # 运行时数据（自动生成）
│       ├── db/       #   SQLite 数据库
│       └── log/      #   日志文件
├── frontend/         # Vue 3 前端
└── docs/             # 文档
```

## 一、启动后端

```bash
# 进入后端目录
cd backend

# 首次运行需要下载依赖
go mod download

# 启动后端服务
go run cmd/server/main.go
```

启动成功后会看到：

```
Using config file: .../backend/configs/config.yaml
time="..." level=info msg="Logger initialized"
time="..." level=info msg="Database initialized"
time="..." level=info msg="Database migration completed"
time="..." level=info msg="i18n initialized (zh)"
time="..." level=info msg="X-Panel server starting on :7777"
[GIN-debug] Listening and serving HTTP on :7777
```

后端默认监听 **http://localhost:7777**。

### 后端配置

配置文件位于 `backend/configs/config.yaml`：

```yaml
system:
  port: "7777"          # 服务端口
  mode: "debug"         # debug（开发） / release（生产）
  data_dir: "./data"    # 数据目录
  db_path: "db/xpanel.db"
  jwt_secret: "dev-secret-change-in-production"

log:
  level: "debug"
  path: "log"           # 相对于 data_dir
```

## 二、启动前端

```bash
# 进入前端目录
cd frontend

# 首次运行需要安装依赖
npm install

# 启动开发服务器
npm run dev
```

启动成功后会看到：

```
VITE v6.x.x  ready in xxx ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: use --host to expose
```

前端默认运行在 **http://localhost:5173**，自动将 `/api` 请求代理到后端 `http://localhost:7777`。

## 三、访问面板

1. 浏览器打开 **http://localhost:5173**
2. 首次使用会进入初始化页面，设置管理员密码
3. 使用用户名 `admin` + 设置的密码登录

## 日常开发流程

**需要同时开两个终端窗口：**

| 终端 | 命令 | 说明 |
|------|------|------|
| 终端 1 | `cd backend && go run cmd/server/main.go` | 后端服务 |
| 终端 2 | `cd frontend && npm run dev` | 前端开发服务 |

### 后端修改后需要重启

Go 没有内置热重载，修改后端代码后需要手动重启：

```bash
# 终端 1 中按 Ctrl+C 停止，然后重新运行
go run cmd/server/main.go
```

### 前端修改自动热更新

Vite 支持 HMR，修改前端代码后浏览器会自动刷新，无需手动操作。

## 常见问题

### 端口被占用

```bash
# 查看占用 7777 端口的进程
lsof -i :7777

# 强制结束占用进程
kill $(lsof -t -i:7777)
```

### 前端显示"服务器错误"

确认后端服务正在运行。前端所有 `/api` 请求都代理到后端，后端未启动则全部报错。

### 重置数据库

删除 `backend/data/db/xpanel.db` 文件后重新启动后端，会自动重建数据库。

### 构建生产版本

```bash
# 构建前端
cd frontend
npm run build
# 产物在 frontend/dist/

# 构建后端
cd backend
go build -o xpanel cmd/server/main.go
```
