# CurlDrop Makefile
# 用法: make [target]

APP_NAME    := curldrop
BUILD_DIR   := build
IMAGE_NAME  := ghcr.io/gaoxinlxl/curldrop
IMAGE_TAG   := latest
LDFLAGS     := -s -w
GOFLAGS     := CGO_ENABLED=0

# 默认目标
.PHONY: all
all: build

# ====== 编译 ======

# 编译当前平台
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	$(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "编译完成: $(BUILD_DIR)/$(APP_NAME)"

# 编译所有平台
.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "全平台编译完成: $(BUILD_DIR)/"

.PHONY: build-linux-amd64
build-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

.PHONY: build-linux-arm64
build-linux-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

.PHONY: build-windows-amd64
build-windows-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOFLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

# ====== Docker ======

# 构建 Docker 镜像
.PHONY: docker
docker:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "镜像构建完成: $(IMAGE_NAME):$(IMAGE_TAG)"

# 推送 Docker 镜像
.PHONY: docker-push
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "镜像推送完成: $(IMAGE_NAME):$(IMAGE_TAG)"

# 构建并推送
.PHONY: docker-release
docker-release: docker docker-push

# ====== 清理 ======

# 清理编译产物
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	@echo "清理完成"

# ====== 运行 ======

# 本地运行
.PHONY: run
run: build
	$(BUILD_DIR)/$(APP_NAME)

# ====== 帮助 ======

.PHONY: help
help:
	@echo "CurlDrop Makefile"
	@echo ""
	@echo "编译:"
	@echo "  make build              编译当前平台二进制 -> $(BUILD_DIR)/"
	@echo "  make build-all          编译所有平台二进制 -> $(BUILD_DIR)/"
	@echo "  make build-linux-amd64  编译 Linux amd64"
	@echo "  make build-linux-arm64  编译 Linux arm64"
	@echo "  make build-darwin-amd64 编译 macOS amd64"
	@echo "  make build-darwin-arm64 编译 macOS arm64 (Apple Silicon)"
	@echo "  make build-windows-amd64 编译 Windows amd64"
	@echo ""
	@echo "Docker:"
	@echo "  make docker             构建 Docker 镜像"
	@echo "  make docker-push        推送 Docker 镜像"
	@echo "  make docker-release     构建并推送 Docker 镜像"
	@echo ""
	@echo "其他:"
	@echo "  make run                编译并运行"
	@echo "  make clean              清理编译产物"
	@echo "  make help               显示此帮助"
	@echo ""
	@echo "变量:"
	@echo "  IMAGE_NAME=$(IMAGE_NAME)"
	@echo "  IMAGE_TAG=$(IMAGE_TAG)"
	@echo "  例: make docker IMAGE_TAG=v1.0.0"
