GOCMD      = go
GOBUILD    = $(GOCMD) build
GOCLEAN    = $(GOCMD) clean
GOARCH     = $(shell go env GOARCH)
GOOS       = $(shell go env GOOS)

# 路径
BASE_PATH    = $(shell pwd)
BUILD_PATH   = $(BASE_PATH)/build
BACKEND_PATH = $(BASE_PATH)/backend
FRONTEND_PATH= $(BASE_PATH)/frontend
ASSETS_PATH  = $(BACKEND_PATH)/cmd/server/web/assets

# 产物名称
APP_NAME     = xpanel

# 版本信息（可通过环境变量覆盖）
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION  ?= $(shell go version | awk '{print $$3}')

# ldflags 注入版本信息
LDFLAGS = -s -w \
	-X 'xpanel/app/version.Version=$(VERSION)' \
	-X 'xpanel/app/version.CommitHash=$(COMMIT_HASH)' \
	-X 'xpanel/app/version.BuildTime=$(BUILD_TIME)' \
	-X 'xpanel/app/version.GoVersion=$(GO_VERSION)'

.PHONY: all clean build_frontend build_backend build package help

# 默认目标
all: build

help:
	@echo "X-Panel 构建系统"
	@echo ""
	@echo "用法:"
	@echo "  make build          - 构建前端 + 后端（完整构建）"
	@echo "  make build_frontend - 仅构建前端"
	@echo "  make build_backend  - 仅构建后端（使用现有前端资源）"
	@echo "  make package        - 构建并打包为 tar.gz"
	@echo "  make clean          - 清理构建产物"
	@echo "  make dev_backend    - 开发模式启动后端"
	@echo ""
	@echo "变量:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT_HASH=$(COMMIT_HASH)"
	@echo "  BUILD_TIME=$(BUILD_TIME)"
	@echo "  GOARCH=$(GOARCH)  GOOS=$(GOOS)"

# 清理
clean:
	rm -rf $(BUILD_PATH)
	rm -rf $(ASSETS_PATH)
	mkdir -p $(ASSETS_PATH)
	echo '<!-- placeholder -->' > $(ASSETS_PATH)/index.html

# 构建前端
build_frontend:
	@echo ">>> 构建前端..."
	cd $(FRONTEND_PATH) && npm install && npm run build
	@echo ">>> 复制前端产物到嵌入目录..."
	rm -rf $(ASSETS_PATH)
	cp -r $(FRONTEND_PATH)/dist $(ASSETS_PATH)
	@echo ">>> 前端构建完成"

# 构建后端
build_backend:
	@echo ">>> 构建后端 $(APP_NAME) ($(GOOS)/$(GOARCH))..."
	@echo ">>> 版本: $(VERSION)  提交: $(COMMIT_HASH)  时间: $(BUILD_TIME)"
	mkdir -p $(BUILD_PATH)
	cd $(BACKEND_PATH) && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GOBUILD) -trimpath -ldflags "$(LDFLAGS)" \
		-o $(BUILD_PATH)/$(APP_NAME) ./cmd/server/
	@echo ">>> 后端构建完成: $(BUILD_PATH)/$(APP_NAME)"

# 完整构建
build: build_frontend build_backend
	@echo ">>> 完整构建完成!"
	@echo ">>> 产物: $(BUILD_PATH)/$(APP_NAME)"
	@ls -lh $(BUILD_PATH)/$(APP_NAME)

# 打包发布
package: build
	@echo ">>> 打包发布..."
	mkdir -p $(BUILD_PATH)/release
	# 复制二进制
	cp $(BUILD_PATH)/$(APP_NAME) $(BUILD_PATH)/release/
	# 复制配置文件模板
	cp $(BACKEND_PATH)/configs/config.yaml $(BUILD_PATH)/release/config.yaml.example
	# 复制安装脚本和服务文件
	@if [ -f $(BASE_PATH)/scripts/install.sh ]; then \
		cp $(BASE_PATH)/scripts/install.sh $(BUILD_PATH)/release/; \
	fi
	@if [ -f $(BASE_PATH)/scripts/xpanel.service ]; then \
		cp $(BASE_PATH)/scripts/xpanel.service $(BUILD_PATH)/release/; \
	fi
	# 打包
	cd $(BUILD_PATH) && tar -czf xpanel-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C release .
	# 生成 SHA256 校验和
	cd $(BUILD_PATH) && sha256sum xpanel-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz > xpanel-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz.sha256
	rm -rf $(BUILD_PATH)/release
	@echo ">>> 打包完成: $(BUILD_PATH)/xpanel-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	@ls -lh $(BUILD_PATH)/xpanel-*.tar.gz

# 开发模式启动后端
dev_backend:
	cd $(BACKEND_PATH) && $(GOCMD) run ./cmd/server/

# 交叉编译：构建 linux/amd64
build_linux_amd64: build_frontend
	@echo ">>> 交叉编译 linux/amd64..."
	mkdir -p $(BUILD_PATH)
	cd $(BACKEND_PATH) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GOBUILD) -trimpath -ldflags "$(LDFLAGS)" \
		-o $(BUILD_PATH)/$(APP_NAME)-linux-amd64 ./cmd/server/

# 交叉编译：构建 linux/arm64
build_linux_arm64: build_frontend
	@echo ">>> 交叉编译 linux/arm64..."
	mkdir -p $(BUILD_PATH)
	cd $(BACKEND_PATH) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		$(GOBUILD) -trimpath -ldflags "$(LDFLAGS)" \
		-o $(BUILD_PATH)/$(APP_NAME)-linux-arm64 ./cmd/server/
