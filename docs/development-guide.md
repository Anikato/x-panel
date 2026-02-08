# X-Panel 开发指导文档

> 基于 1Panel 源码分析的服务器管理面板开发指南
> 
> 版本：v0.1 | 日期：2026-02-06

---

## 目录

1. [项目概述](#1-项目概述)
2. [架构设计](#2-架构设计)
3. [技术栈选型](#3-技术栈选型)
4. [数据模型设计](#4-数据模型设计)
5. [功能模块详细设计](#5-功能模块详细设计)
6. [API 设计规范](#6-api-设计规范)
7. [前端架构设计](#7-前端架构设计)
8. [关键实现参考](#8-关键实现参考)
9. [开发计划与优先级](#9-开发计划与优先级)
10. [与 1Panel 的关键差异](#10-与-1panel-的关键差异)

---

## 1. 项目概述

### 1.1 项目定位

X-Panel 是一个现代化的 Linux 服务器管理面板，参考 1Panel 的功能设计，但在以下方面做出差异化：

- **Nginx 管理**：面板自包含安装 Nginx（源码编译/预编译二进制），所有文件集中在面板安装目录下，不依赖系统包管理器
- **架构简化**：初期可采用单体架构，后续扩展为 Core + Agent 模式支持多机管理
- **面向运维**：强调实用性，保留核心服务器管理能力

### 1.2 核心功能清单

| 序号 | 模块 | 子功能 | 优先级 |
|------|------|--------|--------|
| 1 | 网站管理 | Nginx 站点管理、反向代理、重定向、静态站点 | P0 |
| 2 | 证书管理 | SSL 证书申请/续签/管理、ACME、DNS验证、私有CA | P0 |
| 3 | 数据库管理 | MySQL/PostgreSQL/Redis 实例管理、库管理、远程连接 | P0 |
| 4 | 容器管理 | 容器、镜像、网络、存储卷、Compose、镜像仓库 | P1 |
| 5 | 文件管理 | 文件浏览、上传下载、编辑、压缩解压、权限管理 | P0 |
| 6 | 监控中心 | CPU/内存/磁盘/网络/GPU 监控 | P1 |
| 7 | 进程管理 | 进程列表、终止进程 | P2 |
| 8 | SSH 管理 | SSH 配置、密钥管理 | P1 |
| 9 | 防火墙 | 端口管理、IP 黑白名单、防火墙规则 | P1 |
| 10 | 终端 | Web Terminal、SSH 连接 | P0 |
| 11 | 计划任务 | 定时脚本、数据库备份、日志清理 | P1 |
| 12 | 工具箱 | FTP 管理、Fail2ban、ClamAV 病毒扫描 | P2 |
| 13 | 多机管理 | Agent 注册、节点管理、文件对传 | P2 |
| 14 | 日志审计 | 登录日志、操作日志 | P1 |
| 15 | 面板设置 | 密码修改、端口设置、SSL、安全策略、外观设置 | P0 |

---

## 2. 架构设计

### 2.1 1Panel 架构参考

1Panel 采用 **Core + Agent** 双进程架构：

```
┌─────────────────────────────────────────────┐
│                   Browser                    │
└─────────────────┬───────────────────────────┘
                  │ HTTPS
┌─────────────────▼───────────────────────────┐
│              Core (核心服务)                   │
│  ├── 用户认证 (JWT + Session)                 │
│  ├── 面板设置                                 │
│  ├── 日志审计                                 │
│  ├── 多机管理 (节点注册/调度)                   │
│  └── 代理转发请求到 Agent                      │
└─────────────────┬───────────────────────────┘
                  │ Internal HTTPS (证书双向认证)
┌─────────────────▼───────────────────────────┐
│             Agent (代理服务)                   │
│  ├── 网站管理 (Nginx)                         │
│  ├── 数据库管理                               │
│  ├── 容器管理                                 │
│  ├── 文件管理                                 │
│  ├── 系统管理 (防火墙/SSH/监控/进程)            │
│  ├── 计划任务                                 │
│  └── 工具箱                                   │
└─────────────────────────────────────────────┘
```

**核心通信机制**：
- Core 与 Agent 之间通过 **HTTPS + 证书双向认证** 通信
- Agent 的 middleware 会验证来自 Core 的请求证书
- Core 启动时通过 `cmux` 复用端口，同时提供 HTTP 服务和 gRPC（可选）

### 2.2 X-Panel 推荐架构

**第一阶段（单机模式）**：单进程，所有功能集成在一起

```
┌──────────────────────────────────────────┐
│                 Browser                   │
└────────────────┬─────────────────────────┘
                 │ HTTPS
┌────────────────▼─────────────────────────┐
│            X-Panel Server                 │
│  ┌─────────────────────────────────────┐ │
│  │          API Gateway (Gin)          │ │
│  │  ├── JWT 认证中间件                  │ │
│  │  ├── 操作日志中间件                  │ │
│  │  └── CORS / Rate Limit             │ │
│  └──────────────┬──────────────────────┘ │
│  ┌──────────────▼──────────────────────┐ │
│  │         Service Layer               │ │
│  │  ├── WebsiteService                 │ │
│  │  ├── NginxService                   │ │
│  │  ├── SSLService                     │ │
│  │  ├── DatabaseService                │ │
│  │  ├── ContainerService               │ │
│  │  ├── FileService                    │ │
│  │  ├── MonitorService                 │ │
│  │  ├── FirewallService                │ │
│  │  ├── CronjobService                 │ │
│  │  ├── TerminalService                │ │
│  │  └── SettingService                 │ │
│  └──────────────┬──────────────────────┘ │
│  ┌──────────────▼──────────────────────┐ │
│  │        Repository Layer             │ │
│  │  └── GORM + SQLite                 │ │
│  └─────────────────────────────────────┘ │
└──────────────────────────────────────────┘
```

**第二阶段（多机模式）**：拆分为 Core + Agent

```
Core (管理节点)：用户认证、面板设置、日志审计、节点调度
Agent (被管节点)：实际执行所有服务器管理操作
通信方式：HTTPS + 证书双向认证
```

### 2.3 项目目录结构（推荐）

```
x-panel/
├── backend/                    # Go 后端
│   ├── cmd/                    # 入口点
│   │   └── server/
│   │       └── main.go
│   ├── server/                 # 服务启动
│   │   ├── server.go           # HTTP 服务器启动
│   │   └── init.go             # 初始化流程
│   ├── app/
│   │   ├── api/                # API 控制器层 (Handler)
│   │   │   └── v1/
│   │   │       ├── website.go
│   │   │       ├── nginx.go
│   │   │       ├── ssl.go
│   │   │       ├── database.go
│   │   │       ├── container.go
│   │   │       ├── file.go
│   │   │       ├── monitor.go
│   │   │       ├── firewall.go
│   │   │       ├── ssh.go
│   │   │       ├── cronjob.go
│   │   │       ├── terminal.go
│   │   │       ├── toolbox.go
│   │   │       ├── log.go
│   │   │       └── setting.go
│   │   ├── dto/                # 数据传输对象
│   │   │   ├── request/        # 请求 DTO
│   │   │   └── response/       # 响应 DTO
│   │   ├── model/              # 数据库模型
│   │   │   ├── base.go
│   │   │   ├── website.go
│   │   │   ├── ssl.go
│   │   │   ├── database.go
│   │   │   ├── container.go
│   │   │   ├── cronjob.go
│   │   │   ├── firewall.go
│   │   │   ├── monitor.go
│   │   │   ├── log.go
│   │   │   └── setting.go
│   │   ├── repo/               # 数据访问层
│   │   └── service/            # 业务逻辑层
│   ├── router/                 # 路由注册
│   ├── middleware/              # 中间件
│   ├── global/                 # 全局变量
│   ├── constant/               # 常量定义
│   ├── utils/                  # 工具函数
│   │   ├── nginx/              # Nginx 配置解析器
│   │   │   ├── parser/         # 解析器
│   │   │   ├── components/     # Nginx 配置组件模型
│   │   │   └── dumper.go       # 配置输出
│   │   ├── ssl/                # SSL/ACME 工具
│   │   ├── cmd/                # 命令行工具
│   │   ├── docker/             # Docker SDK 封装
│   │   ├── firewall/           # 防火墙工具
│   │   └── encrypt/            # 加密工具
│   ├── i18n/                   # 国际化
│   ├── init/                   # 初始化模块
│   │   ├── db/                 # 数据库初始化
│   │   ├── migration/          # 数据库迁移
│   │   ├── cache/              # 缓存初始化
│   │   ├── viper/              # 配置加载
│   │   └── log/                # 日志初始化
│   ├── cron/                   # 定时任务管理
│   ├── log/                    # 日志模块
│   ├── go.mod
│   └── go.sum
├── frontend/                   # Vue3 前端
│   ├── src/
│   │   ├── api/                # API 请求封装
│   │   ├── assets/             # 静态资源
│   │   ├── components/         # 公共组件
│   │   ├── hooks/              # 组合式函数
│   │   ├── i18n/               # 国际化
│   │   ├── layout/             # 布局组件
│   │   ├── routers/            # 路由配置
│   │   ├── store/              # 状态管理 (Pinia)
│   │   ├── styles/             # 全局样式
│   │   ├── utils/              # 工具函数
│   │   └── views/              # 页面视图
│   │       ├── website/        # 网站管理
│   │       ├── database/       # 数据库管理
│   │       ├── container/      # 容器管理
│   │       ├── host/           # 系统管理
│   │       ├── terminal/       # 终端
│   │       ├── cronjob/        # 计划任务
│   │       ├── toolbox/        # 工具箱
│   │       ├── log/            # 日志审计
│   │       └── setting/        # 面板设置
│   ├── package.json
│   └── vite.config.ts
├── docs/                       # 文档
└── Makefile                    # 构建脚本
```

---

## 3. 技术栈选型

### 3.1 后端技术栈

| 组件 | 1Panel 选型 | X-Panel 推荐 | 说明 |
|------|------------|-------------|------|
| 语言 | Go 1.24 | Go 1.24+ | 高性能、跨平台编译 |
| Web 框架 | Gin | Gin | 成熟稳定、性能优秀 |
| ORM | GORM | GORM | 功能全面、社区活跃 |
| 数据库 | SQLite (glebarez/sqlite) | SQLite | 轻量、免运维、适合面板场景 |
| 配置管理 | Viper | Viper | 支持多种配置格式 |
| 日志 | Logrus | Logrus / Zap | Logrus 简单易用，Zap 性能更好 |
| 定时任务 | robfig/cron/v3 | robfig/cron/v3 | 标准 Go cron 库 |
| WebSocket | gorilla/websocket | gorilla/websocket | 用于终端、日志实时推送 |
| Docker SDK | docker/docker client | docker/docker client | 官方 SDK |
| ACME 客户端 | go-acme/lego/v4 | go-acme/lego/v4 | Let's Encrypt 自动化证书 |
| SSH 客户端 | golang.org/x/crypto/ssh | golang.org/x/crypto/ssh | SSH 连接管理 |
| 缓存 | patrickmn/go-cache | patrickmn/go-cache | 进程内缓存 |
| 数据库迁移 | go-gormigrate/gormigrate | go-gormigrate/gormigrate | 数据库版本控制 |
| 系统指标 | shirou/gopsutil/v4 | shirou/gopsutil/v4 | 系统信息采集 |
| 国际化 | go-i18n/v2 | go-i18n/v2 | 多语言支持 |
| 对象存储 | aliyun-oss/aws-sdk/minio | 按需选择 | 备份目标存储 |
| SFTP | pkg/sftp | pkg/sftp | 文件传输 |
| 数据库驱动 | go-sql-driver/mysql, jackc/pgx | 相同 | MySQL/PostgreSQL 连接 |
| Redis 客户端 | go-redis/redis | go-redis/redis | Redis 管理 |

### 3.2 前端技术栈

| 组件 | 1Panel 选型 | X-Panel 推荐 | 说明 |
|------|------------|-------------|------|
| 框架 | Vue 3.4 | Vue 3.4+ | 组合式 API |
| UI 库 | Element Plus 2.11 | Element Plus | 组件丰富、中文友好 |
| 状态管理 | Pinia | Pinia | 轻量、TypeScript 友好 |
| 路由 | Vue Router 4 | Vue Router 4 | 标准路由 |
| 构建工具 | Vite 7 | Vite 7 | 快速构建 |
| HTTP 客户端 | Axios | Axios | 请求拦截、响应处理 |
| 图表 | ECharts 5 | ECharts 5 | 监控图表 |
| 终端 | xterm.js 5 | xterm.js 5 | Web Terminal |
| 代码编辑 | Monaco Editor + CodeMirror | Monaco Editor | Nginx 配置编辑 |
| 国际化 | vue-i18n 10 | vue-i18n 10 | 多语言 |
| CSS | Tailwind CSS 3 + SCSS | Tailwind CSS + SCSS | 样式方案 |
| Markdown | md-editor-v3 | md-editor-v3 | Markdown 编辑/预览 |

---

## 4. 数据模型设计

### 4.1 基础模型

所有模型继承自 BaseModel：

```go
// model/base.go
type BaseModel struct {
    ID        uint      `gorm:"primarykey;AUTO_INCREMENT" json:"id"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

### 4.2 网站管理模型

```go
// model/website.go
type Website struct {
    BaseModel
    Protocol      string `gorm:"not null" json:"protocol"`          // HTTP/HTTPS
    PrimaryDomain string `gorm:"not null" json:"primaryDomain"`     // 主域名
    Type          string `gorm:"not null" json:"type"`              // 站点类型: proxy/static/redirect
    Alias         string `gorm:"not null" json:"alias"`             // 别名（唯一标识）
    Remark        string `json:"remark"`
    Status        string `gorm:"not null" json:"status"`            // Running/Stopped
    HttpConfig    string `gorm:"not null" json:"httpConfig"`        // HTTP 配置策略
    ExpireDate    time.Time `json:"expireDate"`
    SiteDir       string `json:"siteDir"`                           // 站点根目录
    ErrorLog      bool   `json:"errorLog"`
    AccessLog     bool   `json:"accessLog"`
    DefaultServer bool   `json:"defaultServer"`                     // 是否默认站点
    IPV6          bool   `json:"IPV6"`
    Rewrite       string `json:"rewrite"`                           // URL 重写规则
    WebsiteSSLID  uint   `json:"websiteSSLID"`                      // 关联的 SSL 证书
    ProxyType     string `json:"proxyType"`                         // 反向代理类型
    
    // 关联
    Domains  []WebsiteDomain `json:"domains" gorm:"foreignKey:WebsiteID"`
    WebsiteSSL WebsiteSSL    `json:"websiteSSL" gorm:"foreignKey:WebsiteSSLID"`
}

// model/website_domain.go
type WebsiteDomain struct {
    BaseModel
    WebsiteID uint   `gorm:"not null" json:"websiteID"`
    Domain    string `gorm:"not null" json:"domain"`
    Port      int    `gorm:"not null" json:"port"`
}
```

### 4.3 SSL 证书模型

```go
// model/ssl.go
type WebsiteSSL struct {
    BaseModel
    PrimaryDomain  string    `gorm:"not null" json:"primaryDomain"`  // 主域名
    PrivateKey     string    `json:"privateKey"`
    Pem            string    `json:"pem"`
    Domains        string    `json:"domains"`                         // JSON: 多域名列表
    CertURL        string    `json:"certURL"`
    Type           string    `json:"type"`                            // autoRenew/manual/import
    Provider       string    `json:"provider"`                        // letsencrypt/zerossl 等
    Organization   string    `json:"organization"`
    Status         string    `json:"status"`                          // ready/applying/error
    ExpireDate     time.Time `json:"expireDate"`
    StartDate      time.Time `json:"startDate"`
    AutoRenew      bool      `json:"autoRenew"`
    Message        string    `json:"message"`                         // 错误信息
    KeyType        string    `json:"keyType"`                         // P256/P384/2048/4096/8192
    PushDir        bool      `json:"pushDir"`
    Dir            string    `json:"dir"`                             // 证书推送目录
    Description    string    `json:"description"`
    SkipDNS        bool      `json:"skipDNS"`                         // 跳过 DNS 检查
    Nameserver1    string    `json:"nameserver1"`
    Nameserver2    string    `json:"nameserver2"`
    DisableCNAME   bool      `json:"disableCNAME"`
    ExecShell      bool      `json:"execShell"`
    Shell          string    `json:"shell"`                           // 申请后执行脚本
    
    // 关联
    AcmeAccountID  uint `json:"acmeAccountID"`
    DNSAccountID   uint `json:"dnsAccountID"`
    CAID           uint `json:"caID"`
}

// model/ssl_acme_account.go
type WebsiteAcmeAccount struct {
    BaseModel
    Email      string `gorm:"not null" json:"email"`
    URL        string `gorm:"not null" json:"url"`
    PrivateKey string `json:"privateKey"`
    Type       string `gorm:"not null" json:"type"`                // letsencrypt/zerossl/buypass/google/sslcom
    KeyType    string `gorm:"not null" json:"keyType"`
    EabKid     string `json:"eabKid"`
    EabHmacKey string `json:"eabHmacKey"`
}

// model/ssl_dns_account.go
type WebsiteDnsAccount struct {
    BaseModel
    Name          string `gorm:"not null" json:"name"`
    Type          string `gorm:"not null" json:"type"`              // dnspod/cloudflare/aliyun 等
    Authorization string `json:"authorization"`                      // JSON: DNS API 凭证
}

// model/ssl_ca.go  (私有 CA)
type WebsiteCA struct {
    BaseModel
    CSR            string `json:"csr"`
    Name           string `gorm:"not null" json:"name"`
    PrivateKey     string `json:"privateKey"`
    KeyType        string `gorm:"not null" json:"keyType"`
    CommonName     string `gorm:"not null" json:"commonName"`
    Country        string `json:"country"`
    Organization   string `json:"organization"`
    OrganizationUP string `json:"organizationUP"`
    Province       string `json:"province"`
    City           string `json:"city"`
}
```

### 4.4 数据库管理模型

```go
// model/database.go
type Database struct {
    BaseModel
    Name     string `gorm:"not null" json:"name"`
    Type     string `gorm:"not null" json:"type"`      // mysql/postgresql/redis
    Version  string `json:"version"`
    From     string `gorm:"not null" json:"from"`      // local/remote
    Address  string `gorm:"not null" json:"address"`
    Port     uint   `gorm:"not null" json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
    IsDelete bool   `json:"isDelete"`
    Description string `json:"description"`
}

// model/database_mysql.go
type DatabaseMysql struct {
    BaseModel
    DatabaseID  uint   `json:"databaseID"`
    Name        string `gorm:"not null" json:"name"`
    Format      string `gorm:"not null" json:"format"`     // 字符集 utf8mb4 等
    Username    string `gorm:"not null" json:"username"`
    Password    string `json:"password"`
    Permission  string `json:"permission"`                  // 访问权限：% / localhost / IP
    IsDelete    bool   `json:"isDelete"`
    Description string `json:"description"`
}

// model/database_postgresql.go
type DatabasePostgresql struct {
    BaseModel
    DatabaseID  uint   `json:"databaseID"`
    Name        string `gorm:"not null" json:"name"`
    Format      string `json:"format"`
    Username    string `gorm:"not null" json:"username"`
    Password    string `json:"password"`
    SuperUser   bool   `json:"superUser"`
    IsDelete    bool   `json:"isDelete"`
    Description string `json:"description"`
}
```

### 4.5 监控模型

```go
// model/monitor.go
type MonitorBase struct {
    BaseModel
    Cpu         float64 `json:"cpu"`
    TopCPU      string  `json:"topCPU"`         // JSON: Top CPU 进程
    Memory      float64 `json:"memory"`
    TopMem      string  `json:"topMem"`          // JSON: Top 内存进程
    LoadUsage   float64 `json:"loadUsage"`
    CpuLoad1    float64 `json:"cpuLoad1"`
    CpuLoad5    float64 `json:"cpuLoad5"`
    CpuLoad15   float64 `json:"cpuLoad15"`
}

type MonitorIO struct {
    BaseModel
    Name  string `json:"name"`
    Read  uint64 `json:"read"`
    Write uint64 `json:"write"`
    Count uint64 `json:"count"`
    Time  uint64 `json:"time"`
}

type MonitorNetwork struct {
    BaseModel
    Name string  `json:"name"`
    Up   float64 `json:"up"`
    Down float64 `json:"down"`
}

type MonitorGPU struct {
    BaseModel
    ProductName   string  `json:"productName"`
    GPUUtil       float64 `json:"gpuUtil"`
    Temperature   float64 `json:"temperature"`
    PowerDraw     float64 `json:"powerDraw"`
    MaxPowerLimit float64 `json:"maxPowerLimit"`
    MemUsed       float64 `json:"memUsed"`
    MemTotal      float64 `json:"memTotal"`
    FanSpeed      int     `json:"fanSpeed"`
    Processes     string  `json:"processes"`
}
```

### 4.6 计划任务模型

```go
// model/cronjob.go
type Cronjob struct {
    BaseModel
    Name       string `gorm:"not null" json:"name"`
    Type       string `gorm:"not null" json:"type"`     // shell/website/database/directory/curl/log
    GroupID    uint   `json:"groupID"`
    SpecCustom bool   `json:"specCustom"`
    Spec       string `gorm:"not null" json:"spec"`      // cron 表达式
    
    // 执行配置
    Executor      string `json:"executor"`                // bash/python 等
    Command       string `json:"command"`
    ContainerName string `json:"containerName"`
    ScriptMode    string `json:"scriptMode"`
    Script        string `json:"script"`
    User          string `json:"user"`
    
    // 备份配置
    Website        string `json:"website"`
    DBType         string `json:"dbType"`
    DBName         string `json:"dbName"`
    URL            string `json:"url"`
    SourceDir      string `json:"sourceDir"`
    ExclusionRules string `json:"exclusionRules"`
    RetainCopies   uint64 `json:"retainCopies"`
    
    // 运行状态
    IsExecuting bool   `json:"isExecuting"`
    Status      string `json:"status"`
    EntryIDs    string `json:"entryIDs"`
}

type JobRecords struct {
    BaseModel
    CronjobID uint      `json:"cronjobID"`
    TaskID    string    `json:"taskID"`
    StartTime time.Time `json:"startTime"`
    Interval  float64   `json:"interval"`       // 执行耗时(秒)
    Records   string    `json:"records"`
    File      string    `json:"file"`            // 备份文件路径
    Status    string    `json:"status"`
    Message   string    `json:"message"`
}
```

### 4.7 防火墙模型

```go
// model/firewall.go
type Firewall struct {
    BaseModel
    Type        string `json:"type"`           // port/address
    Chain       string `json:"chain"`          // INPUT/OUTPUT/FORWARD
    Protocol    string `json:"protocol"`       // tcp/udp/tcp/udp
    SrcIP       string `json:"srcIP"`
    SrcPort     string `json:"srcPort"`
    DstIP       string `json:"dstIP"`
    DstPort     string `json:"dstPort"`
    Strategy    string `gorm:"not null" json:"strategy"`  // accept/drop
    Description string `json:"description"`
}
```

### 4.8 日志审计模型

```go
// model/log.go
type OperationLog struct {
    BaseModel
    Source    string        `json:"source"`       // 操作来源
    IP       string        `json:"ip"`
    Node     string        `json:"node"`          // 节点标识
    Path     string        `json:"path"`          // API 路径
    Method   string        `json:"method"`        // HTTP 方法
    UserAgent string       `json:"userAgent"`
    Latency  time.Duration `json:"latency"`       // 请求耗时
    Status   string        `json:"status"`
    Message  string        `json:"message"`
    DetailZH string        `json:"detailZH"`      // 中文操作详情
    DetailEN string        `json:"detailEN"`      // 英文操作详情
}

type LoginLog struct {
    BaseModel
    IP      string `json:"ip"`
    Address string `json:"address"`               // IP 地理位置
    Agent   string `json:"agent"`                  // User-Agent
    Status  string `json:"status"`                 // Success/Failed
    Message string `json:"message"`
}
```

### 4.9 面板设置模型

```go
// model/setting.go  — Key-Value 模式
type Setting struct {
    BaseModel
    Key   string `json:"key" gorm:"not null;uniqueIndex"`
    Value string `json:"value"`
    About string `json:"about"`
}

// 常用 Key 列表:
// UserName, Password, Email, SystemIP, SystemVersion
// SessionTimeout, LocalTime, PanelName, Theme, Language
// Port, BindAddress, SSL, SSLType, AutoRecover
// SecurityEntrance, ExpirationDays, ComplexityVerification
// MFAStatus, MFASecret, MFAInterval
// MonitorStatus, MonitorInterval, MonitorStoreDays
// AllowIPs, BindDomain
// ApiInterfaceStatus, ApiKey
```

### 4.10 主机管理模型（多机管理）

```go
// model/host.go
type Host struct {
    BaseModel
    GroupID     uint   `json:"groupID"`
    Name        string `json:"name"`
    Addr        string `gorm:"not null" json:"addr"`       // SSH 地址
    Port        int    `gorm:"not null" json:"port"`       // SSH 端口
    User        string `gorm:"not null" json:"user"`
    AuthMode    string `gorm:"not null" json:"authMode"`   // password/key
    Password    string `json:"password"`
    PrivateKey  string `json:"privateKey"`
    PassPhrase  string `json:"passPhrase"`
    RememberPassword bool `json:"rememberPassword"`
    Description string `json:"description"`
}
```

---

## 5. 功能模块详细设计

### 5.1 网站管理模块（基于本地 Nginx）

#### 5.1.1 核心差异：面板自包含 Nginx vs Docker Nginx

1Panel 的 Nginx 运行在 Docker 容器中，通过 Docker SDK 管理。X-Panel 采用**面板自包含安装**策略——Nginx 的二进制、配置、日志全部集中在面板安装目录下：

| 操作 | 1Panel (Docker) | X-Panel (自包含) |
|------|-----------------|-----------------|
| 安装 | Docker 拉取镜像 | 源码编译 / 下载预编译二进制 |
| 安装目录 | Docker 容器内 | `{install_dir}/nginx/` |
| 启停 | docker start/stop | `{nginx_dir}/sbin/nginx` / `-s quit` |
| 配置路径 | 容器内 /etc/nginx | `{nginx_dir}/conf/` |
| 重载 | docker exec nginx -s reload | `{nginx_dir}/sbin/nginx -s reload` |
| 日志路径 | 容器日志 | `{nginx_dir}/logs/` |
| 站点配置 | 容器卷映射 | `{nginx_dir}/conf/conf.d/` |

**自包含安装优势**：
- 不污染系统环境，不与已有 Nginx 冲突
- 版本完全可控，面板管理升级
- 干净卸载——删除目录即可
- 不依赖系统包管理器（apt/yum）

**安装策略（两阶段）**：
1. **开发阶段**：从 Nginx 官方源码编译，`--prefix` 指定安装到面板目录
2. **生产阶段**：预编译好 x86_64 二进制，托管在 Web 服务器供下载，面板安装脚本直接下载解压

**编译安装命令参考**：
```bash
NGINX_VERSION="1.26.2"
INSTALL_DIR="/opt/x-panel/nginx"

wget http://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz
tar -xzf nginx-${NGINX_VERSION}.tar.gz && cd nginx-${NGINX_VERSION}

./configure \
  --prefix=${INSTALL_DIR} \
  --sbin-path=${INSTALL_DIR}/sbin/nginx \
  --conf-path=${INSTALL_DIR}/conf/nginx.conf \
  --pid-path=${INSTALL_DIR}/logs/nginx.pid \
  --error-log-path=${INSTALL_DIR}/logs/error.log \
  --http-log-path=${INSTALL_DIR}/logs/access.log \
  --with-http_ssl_module \
  --with-http_v2_module \
  --with-http_realip_module \
  --with-http_gzip_static_module \
  --with-http_stub_status_module \
  --with-stream \
  --with-stream_ssl_module \
  --with-pcre

make && make install
```

**Nginx 自包含目录结构**：
```
{install_dir}/nginx/
├── sbin/
│   └── nginx                 # Nginx 二进制
├── conf/                     # 配置目录
│   ├── nginx.conf            # 主配置（面板生成）
│   ├── conf.d/               # 站点配置（面板管理）
│   ├── ssl/                  # SSL 证书
│   └── mime.types            # MIME 类型
├── logs/                     # 日志
│   ├── access.log
│   ├── error.log
│   └── nginx.pid
├── temp/                     # 临时文件
│   ├── client_body/
│   ├── proxy/
│   └── fastcgi/
└── html/                     # 默认静态文件
```

#### 5.1.2 Nginx 配置解析器

1Panel 实现了一个完整的 Nginx 配置解析器，位于 `agent/utils/nginx/`，这是一个**核心组件**需要复用或重新实现：

**解析器架构**：
```
utils/nginx/
├── parser/
│   ├── lexer.go              # 词法分析器 - 将配置文本分词
│   ├── parser.go             # 语法分析器 - 构建 AST
│   └── flag/
│       └── flag.go           # 解析标志位
├── components/               # Nginx 配置 AST 节点
│   ├── config.go             # 顶层配置 (nginx.conf)
│   ├── block.go              # 通用块 (http{}, events{} 等)
│   ├── server.go             # server{} 块
│   ├── location.go           # location{} 块
│   ├── upstream.go           # upstream{} 块
│   ├── upstream_server.go    # upstream 中的 server 条目
│   ├── server_listen.go      # listen 指令解析
│   ├── directive.go          # 通用指令 (key value;)
│   ├── comment.go            # 注释
│   ├── http.go               # http{} 块
│   ├── lua_block.go          # Lua 代码块
│   └── statement.go          # 语句接口
└── dumper.go                 # 将 AST 序列化回配置文本
```

**关键接口**：

```go
// components/statement.go - 所有 Nginx 配置节点的接口
type IDirective interface {
    GetName() string
    GetParameters() []string
    GetBlock() IBlock
    GetComment() string
}

type IBlock interface {
    GetDirectives() []IDirective
    FindDirectives(directiveName string) []IDirective
    UpdateDirective(directiveName string, params []string)
    RemoveDirective(directiveName string, params []string)
}

// components/config.go - 顶层配置
type Config struct {
    FilePath   string
    Block      IBlock        // 顶层块
    Upstreams  []*Upstream   // upstream 列表
}

// components/server.go - server 块
type Server struct {
    Block      IBlock
    Listens    []*ServerListen    // 监听配置
    Locations  []*Location        // location 列表
    Comment    string
}
```

**解析器使用方式**：

```go
// 解析 Nginx 配置文件
p, err := parser.NewParser(configFilePath)
config, err := p.Parse()

// 获取所有 server 块
servers := config.FindServers()

// 修改 server 配置
server.UpdateDirective("server_name", []string{"example.com", "www.example.com"})
server.UpdateDirective("root", []string{"/var/www/html"})

// 添加 location 块
location := components.NewLocation("/api/", ...)
server.AddLocation(location)

// 输出修改后的配置
output := dumper.DumpConfig(config)
os.WriteFile(configFilePath, []byte(output), 0644)
```

#### 5.1.3 网站服务核心接口

```go
type IWebsiteService interface {
    // 站点 CRUD
    CreateWebsite(req request.WebsiteCreate) error
    GetWebsite(id uint) (Website, error)
    PageWebsite(req request.WebsiteSearch) (int64, []Website, error)
    UpdateWebsite(req request.WebsiteUpdate) error
    DeleteWebsite(req request.WebsiteDelete) error
    
    // Nginx 配置
    GetNginxConfigByScope(req request.NginxScopeReq) ([]response.NginxParam, error)
    UpdateNginxConfigByScope(req request.NginxConfigUpdate) error
    GetSiteDomain(websiteId uint) ([]WebsiteDomain, error)
    
    // 站点操作
    OpWebsite(req request.WebsiteOp) error           // 启动/停止
    CreateRedirect(req request.NginxRedirectReq) error
    UpdateRedirect(req request.NginxRedirectUpdate) error
    GetRedirect(websiteId uint) ([]response.NginxRedirectConfig, error)
    CreateProxy(req request.WebsiteProxyConfig) error
    UpdateProxy(req request.WebsiteProxyConfig) error
    GetProxies(websiteId uint) ([]response.WebsiteProxyConfig, error)
    
    // 高级功能
    GetAntiLeech(websiteId uint) (response.NginxAntiLeechRes, error)
    UpdateAntiLeech(req request.NginxAntiLeechUpdate) error
    GetAuthBasics(req request.NginxAuthReq) ([]response.NginxAuthRes, error)
    UpdateAuthBasic(req request.NginxAuthUpdate) error
    GetRewriteConfig(req request.NginxRewriteReq) (response.NginxRewriteRes, error)
    UpdateRewriteConfig(req request.NginxRewriteUpdate) error
}
```

#### 5.1.4 站点类型支持

| 站点类型 | 说明 | Nginx 配置模板 |
|---------|------|--------------|
| `proxy` | 反向代理 | proxy_pass + upstream |
| `static` | 静态站点 | root + index |
| `redirect` | 重定向 | return 301 |

#### 5.1.5 Nginx 操作辅助函数

```go
// service/nginx.go - Nginx 服务接口
type INginxService interface {
    GetNginxConfig() (*response.NginxFile, error)     // 获取主配置
    GetStatus() (response.NginxStatus, error)          // 获取运行状态
    UpdateConfigFile(req request.NginxConfigFileUpdate) error  // 更新配置文件
}

// service/nginx_utils.go - 关键辅助函数
func getNginxFull() (NginxFull, error)                 // 获取 Nginx 完整配置
func getNginxConfig(configPath string) (NginxConfig, error)
func createPemFile(website Website, websiteSSL WebsiteSSL) error  // 创建证书文件
func applySSL(website Website, websiteSSL WebsiteSSL, ...) error  // 应用 SSL 配置
func removeSSL(website Website) error
func changeIPV6(website Website, enable bool) error
func changeHSTS(website Website, enable bool, ...) error
```

**X-Panel Nginx 路径（从配置读取，禁止硬编码）**：

```go
// 所有路径通过 global.CONF.Nginx 读取
// 默认值基于面板安装目录 {install_dir}/nginx/
type NginxConfig struct {
    InstallDir string   // 安装根目录: {install_dir}/nginx
    Binary     string   // 二进制路径: {install_dir}/nginx/sbin/nginx
    MainConf   string   // 主配置:     {install_dir}/nginx/conf/nginx.conf
    SitesDir   string   // 站点配置:   {install_dir}/nginx/conf/conf.d
    SSLDir     string   // SSL 证书:   {install_dir}/nginx/conf/ssl
    LogDir     string   // 日志目录:   {install_dir}/nginx/logs
}
```

**进程管理方式（直接信号控制，不使用 systemctl）**：

```go
// 启动
cmd.Exec(nginxBin)
// 配置测试
cmd.Exec(nginxBin, "-t")
// 优雅重载
cmd.Exec(nginxBin, "-s", "reload")
// 优雅停止
cmd.Exec(nginxBin, "-s", "quit")
// 强制停止
cmd.Exec(nginxBin, "-s", "stop")
```

### 5.2 SSL 证书管理模块

#### 5.2.1 证书管理功能矩阵

| 功能 | 说明 |
|------|------|
| ACME 自动申请 | Let's Encrypt / ZeroSSL / Buypass / Google CA / SSL.com |
| DNS 验证 | 支持 30+ DNS 提供商 API（阿里云、腾讯云、Cloudflare 等） |
| HTTP 验证 | 通过 Nginx 配置 .well-known 路径 |
| 手动上传 | 导入已有证书（PEM + Key） |
| 自动续签 | 到期前自动续签，通过定时任务 |
| 私有 CA | 自建 CA 颁发证书 |
| 证书推送 | 将证书推送到指定目录 |
| 执行脚本 | 证书申请/续签后执行自定义脚本 |
| 多域名 | 一张证书支持多个域名（SAN） |

#### 5.2.2 ACME 客户端实现

1Panel 基于 `go-acme/lego/v4` 实现 ACME 协议：

```go
// utils/ssl/client.go - ACME 客户端创建
func NewAcmeClient(account *WebsiteAcmeAccount) (*AcmeClient, error) {
    // 1. 根据 account.Type 确定 ACME 服务器 URL
    // 2. 创建 lego.Config
    // 3. 配置 KeyType (P256/P384/2048/4096)
    // 4. 注册账户（如未注册）
    // 5. 返回客户端实例
}

// utils/ssl/acme.go - 证书申请
func (c *AcmeClient) ObtainSSL(req SSLObtain) error {
    // 1. 设置 DNS 或 HTTP challenge provider
    // 2. 调用 client.Certificate.Obtain()
    // 3. 保存证书和私钥
}

func (c *AcmeClient) RenewSSL(req SSLRenew) error {
    // 1. 加载现有证书
    // 2. 调用 client.Certificate.RenewWithOptions()
    // 3. 更新数据库记录
    // 4. 重载 Nginx
}
```

#### 5.2.3 DNS 提供商支持

```go
// utils/ssl/dns_provider.go - 支持的 DNS 提供商列表
var DNSProviders = map[string]func(authorization string) (challenge.Provider, error){
    "alidns":      newAliDNSProvider,      // 阿里云 DNS
    "dnspod":      newDnspodProvider,      // 腾讯云 DNSPod
    "cloudflare":  newCloudflareProvider,  // Cloudflare
    "namesilo":    newNamesiloProvider,    // NameSilo
    "godaddy":     newGodaddyProvider,     // GoDaddy
    "namecom":     newNamecomProvider,     // Name.com
    "huaweicloud": newHuaweiProvider,      // 华为云
    // ... 30+ 提供商
}
```

#### 5.2.4 SSL 服务接口

```go
type IWebsiteSSLService interface {
    // 证书管理
    Page(search request.WebsiteSSLSearch) (int64, []WebsiteSSL, error)
    Create(req request.WebsiteSSLCreate) (request.WebsiteSSLCreate, error)
    Renew(sslId uint) error
    Delete(req request.WebsiteBatchDelReq) error
    Update(req request.WebsiteSSLUpdate) error
    Upload(req request.WebsiteSSLUpload) error
    
    // ACME 账户管理
    GetACMEAccount(id uint) (*WebsiteAcmeAccount, error)
    CreateACMEAccount(req request.WebsiteAcmeAccountCreate) (*WebsiteAcmeAccount, error)
    DeleteACMEAccount(id uint) error
    
    // DNS 账户管理
    GetDnsAccount(id uint) (*WebsiteDnsAccount, error)
    CreateDnsAccount(req request.WebsiteDnsAccountCreate) (*WebsiteDnsAccount, error)
    UpdateDnsAccount(req request.WebsiteDnsAccountUpdate) error
    DeleteDnsAccount(id uint) error
    
    // 私有 CA
    GetCA(id uint) (*WebsiteCA, error)
    CreateCA(req request.WebsiteCACreate) (*WebsiteCA, error)
    ObtainSSLByCA(req request.WebsiteCAObtain) error
    
    // 证书下载
    DownloadFile(id uint) (*os.File, error)
    
    // 自动续签
    GetSSLByDomains(domains []string) (WebsiteSSL, error)
}
```

### 5.3 数据库管理模块

#### 5.3.1 数据库服务接口

**MySQL 管理**：
```go
type IMysqlService interface {
    // 数据库实例管理
    Create(req dto.MysqlDBCreate) (*model.DatabaseMysql, error)
    Delete(req dto.MysqlDBDelete) error
    ChangeAccess(req dto.ChangeDBInfo) error
    ChangePassword(req dto.ChangeDBInfo) error
    
    // 数据库信息
    SearchWithPage(search dto.MysqlDBSearch) (int64, interface{}, error)
    LoadStatus(req dto.OperateByType) (*dto.MysqlStatus, error)
    LoadVariables(req dto.OperateByType) (*dto.MysqlVariables, error)
    LoadBaseInfo(req dto.OperateByType) (*dto.DBBaseInfo, error)
    
    // 远程连接管理
    LoadRemoteAccess(req dto.OperateByType) (bool, error)
}
```

**PostgreSQL 管理**：
```go
type IPostgresqlService interface {
    Create(req dto.PostgresqlDBCreate) (*model.DatabasePostgresql, error)
    Delete(req dto.PostgresqlDBDelete) error
    ChangePassword(req dto.ChangeDBInfo) error
    ChangePrivileges(req dto.PostgresqlPrivileges) error
    SearchWithPage(search dto.PostgresqlDBSearch) (int64, interface{}, error)
    LoadStatus(req dto.OperateByType) (*dto.PgStatus, error)
    LoadBaseInfo(req dto.OperateByType) (*dto.DBBaseInfo, error)
}
```

**Redis 管理**：
```go
type IRedisService interface {
    LoadStatus(req dto.OperateByType) (*dto.RedisStatus, error)
    LoadConf(req dto.OperateByType) (*dto.RedisConf, error)
    LoadPersistenceConf(req dto.OperateByType) (*dto.RedisPersistence, error)
    UpdateConf(req dto.RedisConfUpdate) error
    ChangePassword(req dto.ChangeRedisPass) error
}
```

#### 5.3.2 数据库连接管理

X-Panel 需要管理本地和远程数据库连接：

```go
// 数据库连接工厂
func NewMySQLClient(info Database) (*sql.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", info.Username, info.Password, info.Address, info.Port)
    return sql.Open("mysql", dsn)
}

func NewPostgreSQLClient(info Database) (*pgx.Conn, error) {
    connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
        info.Address, info.Port, info.Username, info.Password)
    return pgx.Connect(context.Background(), connStr)
}

func NewRedisClient(info Database) (*redis.Client, error) {
    return redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", info.Address, info.Port),
        Password: info.Password,
    }), nil
}
```

### 5.4 容器管理模块

#### 5.4.1 容器服务接口

```go
type IContainerService interface {
    // 容器操作
    Page(req dto.PageContainer) (int64, interface{}, error)
    Inspect(req dto.InspectReq) (string, error)
    ContainerCreate(req dto.ContainerOperate) error
    ContainerOperation(req dto.ContainerOperation) error     // start/stop/restart/kill/pause/unpause
    ContainerLogs(wsConn *websocket.Conn, req dto.ContainerLog) error  // 实时日志
    ContainerStats(id string) (dto.ContainerStats, error)
    
    // 镜像管理
    PageImage(req dto.SearchWithPage) (int64, interface{}, error)
    ImagePull(req dto.ImagePull) error
    ImagePush(req dto.ImagePush) error
    ImageBuild(req dto.ImageBuild) error
    ImageRemove(req dto.BatchDelete) error
    ImageTag(req dto.ImageTag) error
    ImageLoad(req dto.ImageLoad) error
    ImageSave(req dto.ImageSave) error
    
    // 网络管理
    PageNetwork(req dto.SearchWithPage) (int64, interface{}, error)
    CreateNetwork(req dto.NetworkCreate) error
    DeleteNetwork(req dto.BatchDelete) error
    
    // 存储卷管理
    PageVolume(req dto.SearchWithPage) (int64, interface{}, error)
    CreateVolume(req dto.VolumeCreate) error
    DeleteVolume(req dto.BatchDelete) error
    
    // Compose 编排
    PageCompose(req dto.SearchWithPage) (int64, interface{}, error)
    CreateCompose(req dto.ComposeCreate) error
    ComposeOperation(req dto.ComposeOperation) error
    ComposeUpdate(req dto.ComposeUpdate) error
    
    // 镜像仓库
    PageRegistry(req dto.SearchWithPage) (int64, interface{}, error)
    CreateRegistry(req dto.RegistryCreate) error
    DeleteRegistry(req dto.BatchDelete) error
}
```

#### 5.4.2 Docker SDK 封装

```go
// utils/docker/docker.go
import (
    "github.com/docker/docker/client"
)

func NewDockerClient() (*client.Client, error) {
    return client.NewClientWithOpts(
        client.FromEnv,
        client.WithAPIVersionNegotiation(),
    )
}
```

### 5.5 系统管理模块

#### 5.5.1 文件管理

```go
type IFileService interface {
    GetFileList(req request.FileOption) ([]response.FileInfo, error)
    GetFileTree(req request.FileOption) ([]response.FileTree, error)
    GetContent(req request.FileContentReq) (response.FileInfo, error)
    SaveContent(req request.FileEdit) error
    
    Create(req request.FileCreate) error
    Delete(req request.FileDelete) error
    Rename(req request.FileRename) error
    Move(req request.FileMove) error
    Copy(req request.FileCopy) error
    
    Compress(req request.FileCompress) error
    DeCompress(req request.FileDeCompress) error
    
    Upload(req request.FileUpload) error
    Download(req request.FileDownload) (*os.File, error)
    
    ChangeMode(req request.FileChmod) error
    ChangeOwner(req request.FileChown) error
    
    BatchChangeModeAndOwner(req request.FileChownAndChmod) error
    GetRecycleList(req request.RecycleBinSearch) ([]response.RecycleBinDTO, error)
}
```

#### 5.5.2 监控中心

```go
type IMonitorService interface {
    // 实时数据
    LoadCurrentInfo() (*dto.DashboardCurrent, error)
    
    // 历史数据查询
    Search(req dto.MonitorSearch) ([]dto.MonitorData, error)
    
    // 清理历史数据
    Clean(req dto.MonitorClean) error
}

// 定时采集（通过 cron job）
// 默认每 5 分钟采集一次，保存到 SQLite
// 采集指标：CPU、内存、负载、磁盘IO、网络IO、GPU
```

#### 5.5.3 SSH 管理

```go
type ISSHService interface {
    GetSSHInfo() (*dto.SSHInfo, error)
    OperateSSH(req dto.SSHOperate) error           // start/stop/restart
    Update(req dto.SSHUpdate) error                 // 修改 sshd_config
    UpdateByFile(req dto.SSHUpdateByFile) error     // 直接编辑配置文件
    GenerateSSHKey(req dto.GenerateSSHKey) error    // 生成密钥对
    GetLogList(req dto.SSHLogSearch) (int64, []dto.SSHLog, error)  // SSH 登录日志
}
```

#### 5.5.4 防火墙管理

```go
type IFirewallService interface {
    LoadBaseInfo() (dto.FirewallBaseInfo, error)        // 防火墙状态信息
    SearchWithPage(req dto.RuleSearch) (int64, interface{}, error)
    OperateFirewall(req dto.FirewallOperation) error    // 启用/禁用防火墙
    OperatePortRule(req dto.PortRuleOperate) error      // 端口规则
    OperateAddressRule(req dto.AddrRuleOperate) error   // IP 规则
    BatchOperateRule(req dto.BatchRuleOperate) error    // 批量操作
    UpdatePortRule(req dto.PortRuleUpdate) error
    UpdateAddrRule(req dto.AddrRuleUpdate) error
}

// 底层支持 firewalld 和 ufw 两种防火墙后端
// utils/firewall/ 中封装了统一接口
```

#### 5.5.5 进程管理

```go
type IProcessService interface {
    List(req dto.ProcessSearch) ([]dto.ProcessInfo, error)
    Stop(pid int) error
}
```

### 5.6 终端模块

#### 5.6.1 Web Terminal 实现

基于 WebSocket + xterm.js 的 Web 终端：

```go
// 后端：WebSocket Handler
func (api *TerminalAPI) WsSsh(c *gin.Context) {
    // 1. 升级 HTTP 连接为 WebSocket
    conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
    
    // 2. 建立 SSH 连接（本地或远程主机）
    sshClient, err := ssh.NewClient(connInfo)
    session, err := sshClient.NewSession()
    
    // 3. 获取 PTY
    modes := ssh.TerminalModes{...}
    session.RequestPty("xterm-256color", rows, cols, modes)
    
    // 4. 启动 Shell
    session.Shell()
    
    // 5. 双向数据转发
    //    WebSocket → SSH stdin
    //    SSH stdout → WebSocket
    go io.Copy(stdinPipe, wsReader)
    go io.Copy(wsWriter, stdoutPipe)
}

// 本地终端使用 creack/pty 包
func (api *TerminalAPI) WsLocal(c *gin.Context) {
    cmd := exec.Command("/bin/bash")
    ptmx, err := pty.Start(cmd)
    // 双向数据转发
}
```

**前端**：
```typescript
// 使用 xterm.js
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'

const term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    theme: { background: '#1e1e1e' }
})
const fitAddon = new FitAddon()
term.loadAddon(fitAddon)

// WebSocket 连接
const ws = new WebSocket(`wss://${host}/api/v1/terminal/ws`)
ws.onmessage = (event) => term.write(event.data)
term.onData((data) => ws.send(data))
```

### 5.7 计划任务模块

#### 5.7.1 任务类型

| 类型 | 说明 | 执行方式 |
|------|------|---------|
| `shell` | Shell 脚本 | 执行 bash/sh 脚本 |
| `website` | 网站备份 | 备份站点目录 + 数据库 |
| `database` | 数据库备份 | mysqldump / pg_dump |
| `directory` | 目录备份 | tar 打包压缩 |
| `curl` | URL 请求 | 定时请求指定 URL |
| `log` | 日志清理 | 清理过期日志文件 |
| `cutWebsiteLog` | 网站日志切割 | 切割 Nginx 日志 |

#### 5.7.2 计划任务服务

```go
type ICronjobService interface {
    Create(req dto.CronjobCreate) error
    Update(req dto.CronjobUpdate) error
    Delete(req dto.CronjobBatchDelete) error
    
    SearchWithPage(search dto.CronjobSearch) (int64, interface{}, error)
    SearchRecords(search dto.CronjobRecordSearch) (int64, interface{}, error)
    
    HandleOnce(id uint) error                      // 手动执行一次
    UpdateStatus(id uint, status string) error     // 启用/禁用
    
    CleanRecord(req dto.CronjobClean) error
    Download(req dto.CronjobDownload) (string, error)
}

// 底层使用 robfig/cron/v3
var Cron *cron.Cron

func init() {
    Cron = cron.New(cron.WithSeconds())
    Cron.Start()
}

func (s *CronjobService) Create(req dto.CronjobCreate) error {
    // 1. 保存到数据库
    // 2. 注册到 cron
    entryID, _ := Cron.AddFunc(req.Spec, func() {
        s.executeCronjob(cronjob)
    })
    // 3. 保存 entryID
}
```

### 5.8 工具箱模块

#### 5.8.1 FTP 管理

```go
type IFtpService interface {
    SearchWithPage(req dto.SearchWithPage) (int64, interface{}, error)
    Create(req dto.FtpCreate) error
    Delete(req dto.BatchDeleteReq) error
    Update(req dto.FtpUpdate) error
    Sync() error                   // 同步系统 FTP 用户
    GetStatus() (dto.FtpBaseInfo, error)
    Operate(operation string) error // start/stop/restart
}
// 支持 Pure-FTPd 和 vsftpd
```

#### 5.8.2 Fail2ban 管理

```go
type IFail2BanService interface {
    GetFail2BanBaseInfo() (dto.Fail2BanBaseInfo, error)
    SearchWithPage(req dto.Fail2BanSearch) (int64, interface{}, error)
    Operate(operation string) error
    OperateSSHD(req dto.Fail2BanSet) error    // SSH 防护配置
    UpdateConf(req dto.Fail2BanUpdate) error
}
```

#### 5.8.3 ClamAV 病毒扫描

```go
type IClamService interface {
    SearchWithPage(req dto.SearchClamWithPage) (int64, interface{}, error)
    Create(req dto.ClamCreate) error           // 创建扫描任务
    Delete(req dto.ClamDelete) error
    Update(req dto.ClamUpdate) error
    HandleOnce(req dto.OperateByID) error      // 手动扫描
    LoadFile(req dto.ClamFileReq) (string, error)  // 查看扫描报告
    UpdateFile(req dto.UpdateByFile) error
    GetBaseInfo() (dto.ClamBaseInfo, error)
    Operate(operation string) error             // 启动/停止 ClamAV
}
```

### 5.9 多机管理模块

#### 5.9.1 架构设计

```
┌──────────────────────────────────────────────┐
│               Core (管理节点)                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Host 管理 │ │ 节点监控  │ │ 文件对传  │     │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘     │
│       └────────────┼────────────┘            │
│                    │ HTTPS + Cert Auth        │
└────────────────────┼─────────────────────────┘
        ┌────────────┼────────────┐
        ▼            ▼            ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ Agent 1  │ │ Agent 2  │ │ Agent 3  │
   │ (节点A)  │ │ (节点B)  │ │ (节点C)  │
   └─────────┘ └─────────┘ └─────────┘
```

#### 5.9.2 节点管理接口

```go
// Core 端
type IHostService interface {
    TestLocalConn(id uint) bool
    TestByInfo(req dto.HostConnTest) bool
    GetHostByID(id uint) (*dto.HostInfo, error)
    SearchForTree(search dto.SearchForTree) ([]dto.HostTree, error)
    SearchWithPage(search dto.SearchPageWithGroup) (int64, interface{}, error)
    Create(req dto.HostOperate) (*dto.HostInfo, error)
    Update(id uint, upMap map[string]interface{}) (*dto.HostInfo, error)
    Delete(id []uint) error
}

// 文件对传 - 通过 SFTP 实现
func TransferFile(srcHost, dstHost Host, srcPath, dstPath string) error {
    // 1. 连接源主机 SFTP
    // 2. 连接目标主机 SFTP
    // 3. 流式传输文件
}
```

### 5.10 日志审计模块

#### 5.10.1 功能列表

| 功能 | 说明 |
|------|------|
| 登录日志 | 记录所有登录尝试（成功/失败）、IP、地理位置 |
| 操作日志 | 记录所有 API 操作、请求详情、耗时 |
| 日志清理 | 支持手动清理和定期清理 |
| IP 地理位置 | 使用 MaxMind GeoIP 数据库解析 IP 位置 |

#### 5.10.2 操作日志中间件

```go
// middleware/operation_log.go
func OperationLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 记录请求开始时间
        start := time.Now()
        
        // 处理请求
        c.Next()
        
        // 构建操作日志
        log := model.OperationLog{
            Source:    getSource(c),
            IP:       c.ClientIP(),
            Path:     c.Request.URL.Path,
            Method:   c.Request.Method,
            UserAgent: c.Request.UserAgent(),
            Latency:  time.Since(start),
            Status:   getStatus(c),
        }
        
        // 异步保存日志
        go logService.CreateOperationLog(&log)
    }
}
```

### 5.11 面板设置模块

#### 5.11.1 设置项一览

```go
type ISettingService interface {
    // 基本信息
    GetSettingInfo() (*dto.SettingInfo, error)
    Update(key, value string) error
    
    // 安全设置
    UpdatePassword(c *gin.Context, old, new string) error
    UpdatePort(port uint) error
    UpdateBindInfo(req dto.BindInfo) error
    HandlePasswordExpired(c *gin.Context, old, new string) error
    
    // SSL 设置
    UpdateSSL(c *gin.Context, req dto.SSLUpdate) error
    LoadFromCert() (*dto.SSLInfo, error)
    
    // API 密钥
    GenerateApiKey() (string, error)
    UpdateApiConfig(req dto.ApiInterfaceConfig) error
    
    // 代理设置
    UpdateProxy(req dto.ProxyUpdate) error
    
    // 终端设置
    GetTerminalInfo() (*dto.TerminalInfo, error)
    UpdateTerminal(req dto.TerminalInfo) error
    
    // 登录安全
    GetLoginSetting() (*dto.SystemSetting, error)
    
    // MFA 双因子认证
    GenerateRSAKey() error
    
    // 界面设置
    DefaultMenu() error
    
    // 备忘录
    GetMemo() (string, error)
    UpdateMemo(content string) error
}
```

#### 5.11.2 关键设置项

| 分类 | 设置项 | 说明 |
|------|--------|------|
| **安全** | SecurityEntrance | 安全入口路径（如 /panel-login） |
| | ExpirationDays | 密码过期天数 |
| | ComplexityVerification | 密码复杂度要求 |
| | AllowIPs | IP 白名单 |
| | BindDomain | 绑定域名 |
| **认证** | MFAStatus | MFA 开关 |
| | MFASecret | TOTP 密钥 |
| | SessionTimeout | 会话超时时间 |
| **面板** | PanelName | 面板名称 |
| | Theme | 主题（light/dark/auto） |
| | Language | 界面语言 |
| | Port | 面板端口 |
| | SSL | 是否启用 HTTPS |
| **监控** | MonitorStatus | 监控开关 |
| | MonitorInterval | 采集间隔 |
| | MonitorStoreDays | 数据保存天数 |
| **API** | ApiInterfaceStatus | API 接口开关 |
| | ApiKey | API 密钥 |

---

## 6. API 设计规范

### 6.1 路由结构

参考 1Panel 的路由设计，采用 RESTful 风格：

```go
// router/entry.go
func Routers() *gin.Engine {
    router := gin.Default()
    
    // 公共中间件
    router.Use(middleware.CORS())
    router.Use(middleware.RateLimiter())
    
    // 无需认证的路由
    publicGroup := router.Group("/api/v1")
    {
        publicGroup.POST("/auth/login", authAPI.Login)
        publicGroup.POST("/auth/mfa-login", authAPI.MFALogin)
        publicGroup.GET("/auth/captcha", authAPI.Captcha)
    }
    
    // 需要认证的路由
    privateGroup := router.Group("/api/v1")
    privateGroup.Use(middleware.JWTAuth())
    privateGroup.Use(middleware.OperationLog())
    {
        // 网站管理
        websiteRouter := privateGroup.Group("/websites")
        {
            websiteRouter.POST("/search", websiteAPI.Page)
            websiteRouter.POST("", websiteAPI.Create)
            websiteRouter.GET("/:id", websiteAPI.Get)
            websiteRouter.PUT("/:id", websiteAPI.Update)
            websiteRouter.DELETE("", websiteAPI.Delete)
            websiteRouter.POST("/:id/operate", websiteAPI.Operate)
        }
        
        // SSL 证书
        sslRouter := privateGroup.Group("/websites/ssl")
        {
            sslRouter.POST("/search", sslAPI.Page)
            sslRouter.POST("", sslAPI.Create)
            sslRouter.POST("/renew", sslAPI.Renew)
            sslRouter.POST("/upload", sslAPI.Upload)
            sslRouter.DELETE("", sslAPI.Delete)
        }
        
        // 数据库
        databaseRouter := privateGroup.Group("/databases")
        {
            databaseRouter.POST("/search", dbAPI.Page)
            databaseRouter.POST("", dbAPI.Create)
            databaseRouter.POST("/mysql", dbMysqlAPI.Create)
            databaseRouter.POST("/pg", dbPgAPI.Create)
            databaseRouter.POST("/redis", dbRedisAPI.UpdateConf)
        }
        
        // 容器
        containerRouter := privateGroup.Group("/containers")
        {
            containerRouter.POST("/search", containerAPI.Page)
            containerRouter.POST("/operate", containerAPI.Operate)
            containerRouter.GET("/:id/inspect", containerAPI.Inspect)
            containerRouter.POST("/image/search", containerAPI.PageImage)
            containerRouter.POST("/image/pull", containerAPI.ImagePull)
            containerRouter.POST("/network/search", containerAPI.PageNetwork)
            containerRouter.POST("/volume/search", containerAPI.PageVolume)
            containerRouter.POST("/compose/search", containerAPI.PageCompose)
        }
        
        // 系统
        hostRouter := privateGroup.Group("/host")
        {
            hostRouter.POST("/files/search", fileAPI.List)
            hostRouter.POST("/files", fileAPI.Create)
            hostRouter.POST("/files/upload", fileAPI.Upload)
            hostRouter.POST("/monitor/search", monitorAPI.Search)
            hostRouter.POST("/firewall/search", firewallAPI.Page)
            hostRouter.POST("/ssh/info", sshAPI.GetInfo)
            hostRouter.POST("/process/search", processAPI.List)
        }
        
        // 计划任务
        cronjobRouter := privateGroup.Group("/cronjobs")
        {
            cronjobRouter.POST("/search", cronjobAPI.Page)
            cronjobRouter.POST("", cronjobAPI.Create)
            cronjobRouter.PUT("/:id", cronjobAPI.Update)
            cronjobRouter.POST("/handle", cronjobAPI.HandleOnce)
            cronjobRouter.POST("/records", cronjobAPI.Records)
        }
        
        // 工具箱
        toolboxRouter := privateGroup.Group("/toolbox")
        {
            toolboxRouter.POST("/ftp/search", ftpAPI.Page)
            toolboxRouter.POST("/fail2ban/search", fail2banAPI.Page)
            toolboxRouter.POST("/clam/search", clamAPI.Page)
        }
        
        // 日志审计
        logRouter := privateGroup.Group("/logs")
        {
            logRouter.POST("/login", logAPI.PageLoginLog)
            logRouter.POST("/operation", logAPI.PageOperationLog)
            logRouter.POST("/clean", logAPI.Clean)
        }
        
        // 面板设置
        settingRouter := privateGroup.Group("/settings")
        {
            settingRouter.GET("", settingAPI.GetInfo)
            settingRouter.PUT("", settingAPI.Update)
            settingRouter.POST("/password", settingAPI.UpdatePassword)
            settingRouter.POST("/port", settingAPI.UpdatePort)
            settingRouter.POST("/ssl", settingAPI.UpdateSSL)
            settingRouter.POST("/mfa", settingAPI.UpdateMFA)
        }
        
        // WebSocket 路由
        wsRouter := privateGroup.Group("/ws")
        {
            wsRouter.GET("/terminal", terminalAPI.WsLocal)
            wsRouter.GET("/ssh", terminalAPI.WsSsh)
            wsRouter.GET("/container/logs", containerAPI.WsLogs)
        }
    }
    
    // 前端静态文件
    router.StaticFS("/public", http.FS(publicFS))
    router.NoRoute(func(c *gin.Context) {
        c.File("./web/index.html")
    })
    
    return router
}
```

### 6.2 通用响应格式

```go
// dto/response.go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type PageResult struct {
    Total int64       `json:"total"`
    Items interface{} `json:"items"`
}

// 使用示例
func (api *WebsiteAPI) Page(c *gin.Context) {
    var req request.WebsiteSearch
    if err := helper.CheckBindAndValidate(&req, c); err != nil {
        helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
        return
    }
    total, websites, err := websiteService.PageWebsite(req)
    if err != nil {
        helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
        return
    }
    helper.SuccessWithData(c, dto.PageResult{
        Total: total,
        Items: websites,
    })
}
```

### 6.3 通用搜索 DTO

```go
// dto/common.go
type PageInfo struct {
    Page     int `json:"page" validate:"required,number"`
    PageSize int `json:"pageSize" validate:"required,number"`
}

type OrderInfo struct {
    OrderBy string `json:"orderBy"`
    Order   string `json:"order" validate:"oneof=null ascending descending"`
}

type SearchWithPage struct {
    PageInfo
    Info string `json:"info"`
}
```

---

## 7. 前端架构设计

### 7.1 路由设计

参考 1Panel 的路由模块化设计：

```typescript
// routers/modules/website.ts
export default {
    path: '/websites',
    component: Layout,
    redirect: '/websites/list',
    meta: { title: '网站管理', icon: 'Website' },
    children: [
        {
            path: 'list',
            name: 'WebsiteList',
            component: () => import('@/views/website/list/index.vue'),
            meta: { title: '网站列表' }
        },
        {
            path: 'ssl',
            name: 'WebsiteSSL',
            component: () => import('@/views/website/ssl/index.vue'),
            meta: { title: 'SSL 证书' }
        },
        {
            path: 'acme-account',
            name: 'AcmeAccount',
            component: () => import('@/views/website/acme/index.vue'),
            meta: { title: 'ACME 账户' }
        },
        {
            path: 'dns-account',
            name: 'DnsAccount',
            component: () => import('@/views/website/dns/index.vue'),
            meta: { title: 'DNS 账户' }
        },
        {
            path: 'ca',
            name: 'WebsiteCA',
            component: () => import('@/views/website/ca/index.vue'),
            meta: { title: '证书颁发机构' }
        }
    ]
}

// routers/modules/database.ts
export default {
    path: '/databases',
    component: Layout,
    redirect: '/databases/mysql',
    meta: { title: '数据库', icon: 'Database' },
    children: [
        { path: 'mysql', name: 'MySQL', component: () => import('@/views/database/mysql/index.vue') },
        { path: 'postgresql', name: 'PostgreSQL', component: () => import('@/views/database/postgresql/index.vue') },
        { path: 'redis', name: 'Redis', component: () => import('@/views/database/redis/index.vue') }
    ]
}

// routers/modules/container.ts
export default {
    path: '/containers',
    component: Layout,
    meta: { title: '容器', icon: 'Container' },
    children: [
        { path: 'list', name: 'ContainerList', ... },
        { path: 'image', name: 'ImageList', ... },
        { path: 'network', name: 'NetworkList', ... },
        { path: 'volume', name: 'VolumeList', ... },
        { path: 'compose', name: 'ComposeList', ... },
        { path: 'registry', name: 'RegistryList', ... }
    ]
}

// routers/modules/host.ts
export default {
    path: '/host',
    component: Layout,
    meta: { title: '系统', icon: 'System' },
    children: [
        { path: 'files', name: 'FileManager', ... },
        { path: 'monitor', name: 'Monitor', ... },
        { path: 'process', name: 'Process', ... },
        { path: 'ssh', name: 'SSHManager', ... },
        { path: 'firewall', name: 'Firewall', ... }
    ]
}

// routers/modules/terminal.ts
export default {
    path: '/terminal',
    component: Layout,
    meta: { title: '终端', icon: 'Terminal' },
    children: [
        { path: '', name: 'Terminal', component: () => import('@/views/terminal/index.vue') }
    ]
}

// routers/modules/cronjob.ts
export default {
    path: '/cronjobs',
    component: Layout,
    meta: { title: '计划任务', icon: 'Cronjob' },
    children: [
        { path: '', name: 'CronjobList', component: () => import('@/views/cronjob/index.vue') }
    ]
}

// routers/modules/toolbox.ts
export default {
    path: '/toolbox',
    component: Layout,
    meta: { title: '工具箱', icon: 'Toolbox' },
    children: [
        { path: 'ftp', name: 'FTP', ... },
        { path: 'fail2ban', name: 'Fail2ban', ... },
        { path: 'clam', name: 'ClamAV', ... }
    ]
}

// routers/modules/log.ts
export default {
    path: '/logs',
    component: Layout,
    meta: { title: '日志审计', icon: 'Log' },
    children: [
        { path: 'login', name: 'LoginLog', ... },
        { path: 'operation', name: 'OperationLog', ... },
        { path: 'system', name: 'SystemLog', ... }
    ]
}

// routers/modules/setting.ts
export default {
    path: '/settings',
    component: Layout,
    meta: { title: '面板设置', icon: 'Setting' },
    children: [
        { path: 'panel', name: 'PanelSetting', ... },
        { path: 'safety', name: 'SafetySetting', ... },
        { path: 'about', name: 'About', ... }
    ]
}
```

### 7.2 状态管理

```typescript
// store/modules/global.ts
import { defineStore } from 'pinia'

export const useGlobalStore = defineStore('global', {
    state: () => ({
        isLogin: false,
        language: 'zh',
        theme: 'light',
        panelName: 'X-Panel',
        isFullScreen: false,
        currentNode: null,     // 当前选中的节点（多机管理）
        menuCollapse: false,
    }),
    actions: {
        setLogin(status: boolean) { this.isLogin = status },
        setLanguage(lang: string) { this.language = lang },
        setTheme(theme: string) { this.theme = theme },
    },
    persist: true,
})
```

### 7.3 API 请求封装

```typescript
// api/http.ts
import axios from 'axios'

const http = axios.create({
    baseURL: '/api/v1',
    timeout: 60000,
})

// 请求拦截
http.interceptors.request.use(config => {
    const token = sessionStorage.getItem('token')
    if (token) {
        config.headers['Authorization'] = `Bearer ${token}`
    }
    return config
})

// 响应拦截
http.interceptors.response.use(
    response => response.data,
    error => {
        if (error.response?.status === 401) {
            router.push('/login')
        }
        return Promise.reject(error)
    }
)

export default http
```

---

## 8. 关键实现参考

### 8.1 初始化流程

参考 1Panel 的初始化流程：

```go
// server/init.go
func Init() {
    // 1. 加载配置文件 (Viper)
    initViper()
    
    // 2. 初始化日志
    initLogger()
    
    // 3. 初始化数据库 (SQLite + GORM)
    initDB()
    
    // 4. 运行数据库迁移
    runMigrations()
    
    // 5. 初始化缓存
    initCache()
    
    // 6. 初始化国际化
    initI18n()
    
    // 7. 初始化定时任务
    initCronJobs()
    
    // 8. 初始化防火墙
    initFirewall()
    
    // 9. 初始化路由
    initRouter()
}

// server/server.go
func Start() {
    Init()
    
    router := Routers()
    
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", global.CONF.System.Port),
        Handler: router,
    }
    
    // 优雅关闭
    go func() {
        if err := server.ListenAndServe(); err != nil {
            log.Fatal(err)
        }
    }()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}
```

### 8.2 Nginx 配置管理流程

```go
// 创建网站的核心流程
func (w *WebsiteService) CreateWebsite(req request.WebsiteCreate) error {
    // 1. 验证域名可用性
    if err := checkDomainAvailable(req.PrimaryDomain); err != nil {
        return err
    }
    
    // 2. 创建站点目录
    siteDir := path.Join("/var/www", req.Alias)
    os.MkdirAll(siteDir, 0755)
    
    // 3. 生成 Nginx 配置
    config := generateNginxConfig(req)
    
    // 4. 写入配置文件
    configPath := path.Join(NginxSitesDir, req.Alias+".conf")
    os.WriteFile(configPath, []byte(config), 0644)
    
    // 5. 检测配置合法性
    if err := testNginxConfig(); err != nil {
        os.Remove(configPath)  // 回滚
        return fmt.Errorf("Nginx config test failed: %v", err)
    }
    
    // 6. 保存到数据库
    website := model.Website{...}
    db.Create(&website)
    
    // 7. 重载 Nginx
    reloadNginx()
    
    return nil
}

// Nginx 配置测试
func testNginxConfig() error {
    cmd := exec.Command("nginx", "-t")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("%s", string(output))
    }
    return nil
}

// Nginx 重载
func reloadNginx() error {
    cmd := exec.Command("nginx", "-s", "reload")
    return cmd.Run()
    // 或者使用 systemctl
    // cmd := exec.Command("systemctl", "reload", "nginx")
}
```

### 8.3 1Panel 中的常量定义（Nginx 相关）

```go
// constant/nginx.go
const (
    NginxScopeHTTP   = "http"
    NginxScopeServer = "server"
    NginxScopeOut    = "out"
    
    // 常见指令
    Listen          = "listen"
    ServerName      = "server_name"
    Root            = "root"
    Index           = "index"
    ProxyPass       = "proxy_pass"
    SSLCertificate  = "ssl_certificate"
    SSLCertificateKey = "ssl_certificate_key"
    AccessLog       = "access_log"
    ErrorLog        = "error_log"
    Return          = "return"
    Rewrite         = "rewrite"
    
    // 配置文件名
    IndexConfig     = "index"
    LimitConfig     = "limit"
    CacheConfig     = "cache"
    SSLConfig       = "ssl"
    HttpConfig      = "http_config"
    ProxyConfig     = "proxy"
)
```

### 8.4 认证与安全

```go
// 登录流程
func (a *AuthService) Login(req dto.Login) (*dto.UserLoginInfo, error) {
    // 1. 检查安全入口 (SecurityEntrance)
    // 2. 验证验证码 (base64Captcha)
    // 3. 检查 IP 白名单
    // 4. 验证用户名密码
    // 5. 检查 MFA (如已启用)
    // 6. 生成 JWT Token
    // 7. 记录登录日志
}

// JWT 中间件
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        claims, err := jwt.ParseToken(token)
        if err != nil {
            c.JSON(401, gin.H{"code": 401, "message": "unauthorized"})
            c.Abort()
            return
        }
        c.Set("claims", claims)
        c.Next()
    }
}
```

### 8.5 错误处理

```go
// buserr/errors.go - 业务错误
type BusinessError struct {
    Msg    string
    Detail interface{}
    Err    error
    Map    map[string]interface{}
}

func New(msg string) BusinessError {
    return BusinessError{Msg: msg}
}

func WithDetail(msg string, detail interface{}, err error) BusinessError {
    return BusinessError{Msg: msg, Detail: detail, Err: err}
}

// 使用示例
if website.ID == 0 {
    return buserr.New("ErrWebsiteNotFound")
}
```

---

## 9. 开发计划与优先级

### 9.1 阶段规划

#### 第一阶段：基础框架 + 核心功能（预计 4-6 周）

```
Sprint 1 (Week 1-2): 项目骨架
├── 搭建后端框架 (Gin + GORM + SQLite)
├── 搭建前端框架 (Vue3 + Element Plus + Pinia)
├── 实现用户认证 (登录/JWT/Session)
├── 实现基础面板设置
└── 实现操作日志中间件

Sprint 2 (Week 3-4): 网站管理
├── 实现 Nginx 配置解析器（移植/重写）
├── 实现网站 CRUD（创建/配置/删除）
├── 实现反向代理管理
├── 实现静态站点管理
└── 实现 Nginx 状态监控

Sprint 3 (Week 5-6): 证书管理
├── 实现 ACME 客户端集成
├── 实现 DNS 验证（主流提供商）
├── 实现证书自动续签
├── 实现手动证书上传
└── 实现证书与网站关联
```

#### 第二阶段：系统管理 + 数据库（预计 4-6 周）

```
Sprint 4 (Week 7-8): 系统管理
├── 实现文件管理器
├── 实现 Web Terminal
├── 实现 SSH 管理
├── 实现防火墙管理
└── 实现进程管理

Sprint 5 (Week 9-10): 数据库管理
├── 实现 MySQL 管理
├── 实现 PostgreSQL 管理
├── 实现 Redis 管理
└── 实现数据库远程连接

Sprint 6 (Week 11-12): 监控 + 计划任务
├── 实现监控数据采集
├── 实现监控面板 (ECharts)
├── 实现计划任务管理
└── 实现日志审计完整功能
```

#### 第三阶段：容器 + 工具箱 + 高级功能（预计 4-6 周）

```
Sprint 7 (Week 13-14): 容器管理
├── 实现容器列表/操作
├── 实现镜像管理
├── 实现 Compose 编排
├── 实现网络/存储卷管理
└── 实现镜像仓库管理

Sprint 8 (Week 15-16): 工具箱
├── 实现 FTP 管理
├── 实现 Fail2ban 管理
├── 实现 ClamAV 扫描
└── 实现其他系统工具

Sprint 9 (Week 17-18): 多机管理 + 优化
├── 实现 Core + Agent 架构拆分
├── 实现节点注册/管理
├── 实现文件对传
├── 实现界面设置（主题/语言）
└── 全面测试与优化
```

### 9.2 每个模块的预估工作量

| 模块 | 后端 (人天) | 前端 (人天) | 总计 |
|------|------------|------------|------|
| 项目框架搭建 | 3 | 3 | 6 |
| 认证与安全 | 3 | 2 | 5 |
| 网站管理 (Nginx) | 8 | 6 | 14 |
| SSL 证书管理 | 6 | 4 | 10 |
| MySQL 管理 | 4 | 3 | 7 |
| PostgreSQL 管理 | 3 | 3 | 6 |
| Redis 管理 | 2 | 2 | 4 |
| 容器管理 | 6 | 5 | 11 |
| 文件管理 | 5 | 5 | 10 |
| 监控中心 | 3 | 4 | 7 |
| 进程管理 | 1 | 1 | 2 |
| SSH 管理 | 2 | 2 | 4 |
| 防火墙 | 3 | 2 | 5 |
| Web Terminal | 3 | 3 | 6 |
| 计划任务 | 4 | 3 | 7 |
| 工具箱 (FTP/Fail2ban/ClamAV) | 4 | 3 | 7 |
| 日志审计 | 2 | 2 | 4 |
| 面板设置 | 3 | 3 | 6 |
| 多机管理 | 6 | 4 | 10 |
| **合计** | **71** | **60** | **131 人天** |

---

## 10. 与 1Panel 的关键差异

### 10.1 技术架构差异

| 方面 | 1Panel | X-Panel |
|------|--------|---------|
| Nginx 部署 | Docker 容器内 | 面板自包含安装（{install_dir}/nginx/） |
| Nginx 安装方式 | Docker 拉取镜像 | 源码编译 / 下载预编译二进制 |
| Nginx 配置路径 | Docker volume 映射 | {install_dir}/nginx/conf/ |
| Nginx 操作方式 | Docker exec | 直接执行 nginx 二进制信号控制 |
| 应用安装 | 通过 App Store (Docker) | 仅管理已安装的服务 |
| 数据库部署 | Docker 容器 | 本地安装或远程连接 |
| 初期架构 | Core + Agent 双进程 | 单进程（后续拆分） |

### 10.2 需要重点重写的模块

1. **Nginx 管理层**：去除所有 Docker 依赖，改为面板自包含 Nginx
   - `getNginxFull()` 需重写，读取 `{install_dir}/nginx/conf/nginx.conf`
   - `createPemFile()` 证书文件放到 `{install_dir}/nginx/conf/ssl/`
   - 站点配置写入 `{install_dir}/nginx/conf/conf.d/`
   - 直接执行 `{install_dir}/nginx/sbin/nginx -s reload` 管理进程（不使用 systemctl）
   - 新增 Nginx 安装服务：源码编译安装 / 下载预编译二进制

2. **数据库管理层**：去除 Docker 依赖
   - MySQL/PostgreSQL/Redis 直接通过 socket 或 TCP 连接
   - 配置文件直接操作 `/etc/mysql/`, `/etc/postgresql/`, `/etc/redis/`

3. **可直接复用的模块**：
   - Nginx 配置解析器（`utils/nginx/`）—— 纯 Go 实现，与 Docker 无关
   - SSL/ACME 工具（`utils/ssl/`）—— 纯 Go 实现
   - 防火墙工具
   - 文件管理
   - 监控采集（gopsutil）
   - 计划任务（robfig/cron）
   - 日志审计

### 10.3 不需要实现的功能

1. **App Store（应用商店）**：1Panel 的核心功能之一，但 X-Panel 不做应用商店
2. **Runtime 管理**：1Panel 管理 PHP/Node.js 等运行时环境，X-Panel 暂不涉及
3. **快照管理**：1Panel 的系统快照功能，X-Panel 初期不需要
4. **备份账号管理**：1Panel 支持备份到 S3/OSS 等，X-Panel 初期可简化

---

## 附录 A：1Panel 关键源码文件索引

### 后端 - Agent

| 文件路径 | 说明 |
|---------|------|
| `agent/server/server.go` | Agent HTTP 服务启动 |
| `agent/server/init.go` | Agent 初始化流程 |
| `agent/global/global.go` | 全局变量定义 (DB, LOG, CONF, CACHE) |
| `agent/router/entry.go` | 路由注册入口 |
| `agent/app/api/v2/entry.go` | API Handler 注册 |
| `agent/app/service/nginx.go` | Nginx 服务层 |
| `agent/app/service/nginx_utils.go` | Nginx 辅助函数 |
| `agent/app/service/website.go` | 网站管理服务（2391行） |
| `agent/app/service/website_ssl.go` | SSL 证书服务 |
| `agent/app/service/database_mysql.go` | MySQL 管理服务 |
| `agent/app/service/database_postgresql.go` | PostgreSQL 管理服务 |
| `agent/app/service/database_redis.go` | Redis 管理服务 |
| `agent/app/service/container.go` | 容器管理服务（1823行） |
| `agent/app/service/file.go` | 文件管理服务 |
| `agent/app/service/monitor.go` | 监控服务 |
| `agent/app/service/firewall.go` | 防火墙服务 |
| `agent/app/service/ssh.go` | SSH 管理服务 |
| `agent/app/service/cronjob.go` | 计划任务服务 |
| `agent/app/service/setting.go` | Agent 设置服务 |
| `agent/app/service/clam.go` | ClamAV 服务 |
| `agent/app/service/fail2ban.go` | Fail2ban 服务 |
| `agent/app/service/ftp.go` | FTP 服务 |
| `agent/utils/nginx/` | Nginx 配置解析器（核心工具） |
| `agent/utils/ssl/` | SSL/ACME 工具 |
| `agent/constant/nginx.go` | Nginx 相关常量 |

### 后端 - Core

| 文件路径 | 说明 |
|---------|------|
| `core/server/server.go` | Core HTTP 服务启动 |
| `core/app/service/host.go` | 主机管理（SSH 连接） |
| `core/app/service/logs.go` | 日志审计服务 |
| `core/app/service/setting.go` | 面板设置服务 |
| `core/app/model/logs.go` | 日志模型 (OperationLog, LoginLog) |
| `core/app/model/setting.go` | 设置模型 (Key-Value) |
| `core/app/model/host.go` | 主机模型 |
| `core/middleware/` | 中间件（JWT、日志、CORS等） |

### 前端

| 路径 | 说明 |
|------|------|
| `frontend/src/routers/modules/website.ts` | 网站管理路由 |
| `frontend/src/routers/modules/database.ts` | 数据库路由 |
| `frontend/src/routers/modules/container.ts` | 容器路由 |
| `frontend/src/routers/modules/host.ts` | 系统管理路由 |
| `frontend/src/routers/modules/terminal.ts` | 终端路由 |
| `frontend/src/routers/modules/cronjob.ts` | 计划任务路由 |
| `frontend/src/routers/modules/toolbox.ts` | 工具箱路由 |
| `frontend/src/routers/modules/log.ts` | 日志审计路由 |
| `frontend/src/routers/modules/setting.ts` | 面板设置路由 |
| `frontend/src/views/` | 页面组件目录 |

---

## 附录 B：关键第三方库列表

### 后端核心依赖

```go
// Web & API
github.com/gin-gonic/gin             // HTTP 框架
gorm.io/gorm                         // ORM
github.com/glebarez/sqlite            // SQLite 驱动（纯 Go）

// 安全 & 认证
golang.org/x/crypto                   // SSH、加密
github.com/gorilla/websocket          // WebSocket
github.com/gorilla/sessions           // Session

// Docker
github.com/docker/docker              // Docker SDK
github.com/docker/compose/v2          // Docker Compose
github.com/compose-spec/compose-go/v2 // Compose 规范

// SSL & ACME
github.com/go-acme/lego/v4           // ACME 客户端（Let's Encrypt）

// 数据库客户端
github.com/go-sql-driver/mysql        // MySQL 驱动
github.com/jackc/pgx/v5              // PostgreSQL 驱动
github.com/go-redis/redis            // Redis 客户端

// 系统信息
github.com/shirou/gopsutil/v4         // 系统指标采集

// 配置 & 日志
github.com/spf13/viper               // 配置管理
github.com/sirupsen/logrus           // 日志

// 终端
github.com/creack/pty                // PTY（本地终端）
github.com/pkg/sftp                  // SFTP 客户端

// 定时任务
github.com/robfig/cron/v3            // Cron 任务

// 工具
github.com/jinzhu/copier             // 对象复制
github.com/nicksnyder/go-i18n/v2     // 国际化
github.com/mholt/archiver/v4         // 压缩解压
github.com/google/uuid               // UUID
```

### 前端核心依赖

```json
{
    "vue": "^3.4",
    "element-plus": "^2.11",
    "pinia": "^2.1",
    "vue-router": "^4.5",
    "axios": "^1.7",
    "echarts": "^5.5",
    "@xterm/xterm": "^5.5",
    "monaco-editor": "^0.53",
    "vue-i18n": "^10",
    "crypto-js": "^4.2",
    "codemirror": "^6"
}
```

---

## 附录 C：配置文件模板

### 后端配置 (config.yml)

```yaml
system:
  port: 7777
  mode: release          # debug/release
  data_dir: /opt/x-panel/data
  cache: /opt/x-panel/cache
  tmp_dir: /opt/x-panel/tmp
  log_dir: /opt/x-panel/log

db:
  path: /opt/x-panel/data/x-panel.db

log:
  level: info
  max_size: 200          # MB
  max_backups: 5
  max_age: 30            # days

nginx:
  install_dir: /opt/x-panel/nginx        # Nginx 自包含安装目录
  version: "1.26.2"                       # 当前安装的 Nginx 版本

monitor:
  enabled: true
  interval: 300          # seconds
  store_days: 30

session:
  timeout: 86400         # seconds
```

---

> **文档维护说明**：本文档随项目进展持续更新，记录架构决策和实现细节的变更。
