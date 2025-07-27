# 项目名称
PROJECT_NAME := cocom

SHELL := /bin/bash
GO = go
#GOARCH = amd64
#GOARCH = arm64

# 检测操作系统类型
OS := $(shell uname -s)
ARCH := $(shell uname -m)

BuildDir := build

VersionImportPath := pkg/version
VersionBuildDir := $(VersionImportPath)/$(BuildDir)

# all .go files that are not auto-generated and should be auto-formatted and linted.
ALL_SRC = $(shell find . -name '*.go' \
				   -not -name 'doc.go' \
				   -not -name '_*' \
				   -not -name '.*' \
				   -not -name 'mocks*' \
				   -not -name 'model.pb.go' \
				   -not -name 'model_test.pb.go' \
				   -not -name 'storage_test.pb.go' \
				   -not -path './examples/*' \
				   -not -path './vendor/*' \
				   -not -path '*/mocks/*' \
				   -not -path '*/*-gen/*' \
				   -type f | \
				sort)

# ALL_PKGS is used with 'nocover'
ALL_PKGS = $(shell echo $(dir $(ALL_SRC)) | tr ' ' '\n' | sort -u)

UNAME := $(shell uname -m)
#Race flag is not supported on s390x architecture
ifeq ($(UNAME), s390x)
	RACE=
else
	RACE=-race
endif
GOMOD := $(shell $(GO) list)
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
GOBUILD=CGO_ENABLED=0 installsuffix=cgo $(GO) build -trimpath
GOTEST=$(GO) test -v $(RACE)
GOFMT=gofmt
GOFUMPT=gofumpt
FMT_LOG=.fmt.log
IMPORT_LOG=.import.log

COMMIT_ID=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags --always --dirty)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
TZ=Asia/Shanghai
BUILD_AT=$(shell TZ=${TZ} date +"%Y-%m-%dT%H:%M:%SZ")

GOLDFLAGS := -ldflags "\
	 -X '$(GOMOD)/$(VersionImportPath).Version=$(VERSION)' \
	 -X '$(GOMOD)/$(VersionImportPath).BuiltAt=$(BUILD_AT)' \
	 -X '$(GOMOD)/$(VersionImportPath).CommitID=$(COMMIT_ID)' \
	 -X '$(GOMOD)/$(VersionImportPath).Branch=$(GIT_BRANCH)'"

SED=sed

SWAGGER_VER=0.27.0
SWAGGER_IMAGE=quay.io/goswagger/swagger:v$(SWAGGER_VER)
SWAGGER=docker run --rm -it -u ${shell id -u} -v "${PWD}:/go/src/" -w /go/src/ $(SWAGGER_IMAGE)
SWAGGER_GEN_DIR=swagger-gen

COLOR_PASS=$(shell printf "\033[32mPASS\033[0m")
COLOR_FAIL=$(shell printf "\033[31mFAIL\033[0m")
COLORIZE ?=$(SED) ''/PASS/s//$(COLOR_PASS)/'' | $(SED) ''/FAIL/s//$(COLOR_FAIL)/''
DOCKER_NAMESPACE?=suixibing
DOCKER_TAG?=latest

.DEFAULT_GOAL := help

# 准备目标
.PHONY: prepare
prepare: go-gen
	@mkdir -p $(BuildDir) $(VersionBuildDir)
	@echo "Generating dirty info..."
	@if git diff HEAD --quiet; then \
		rm $(VersionBuildDir)/dirty_info.txt; \
		touch $(VersionBuildDir)/dirty_info.txt; \
	else \
		git diff HEAD > $(VersionBuildDir)/dirty_info.txt; \
	fi

# TODO: no files actually use this right now
.PHONY: go-gen
go-gen:
	@echo skipping go generate ./...

# 构建目标
.PHONY: build
build: fmt
	GOARCH=$(GOARCH) $(GO) build $(GOLDFLAGS) -o $(BuildDir)/$(PROJECT_NAME)

# 构建 docker 镜像目标
.PHONY: build-image
build-image:
	docker build . -t $(DOCKER_NAMESPACE)/$(PROJECT_NAME):$(VERSION)

.PHONY: run-server
run-server: build
	./$(BuildDir)/$(PROJECT_NAME) server --config ./$(BuildDir)/conf/cocom.yaml

# 安装目标
.PHONY: install
install: build
	@echo "Installing $(PROJECT_NAME)..."
	cp $(BuildDir)/$(PROJECT_NAME) ~/bin
	$(PROJECT_NAME) completion zsh > ~/.$(PROJECT_NAME)/zsh_completion
	source ~/.$(PROJECT_NAME)/zsh_completion
	@echo "Installation completed."

# 安装工具目标
.PHONY: install-tools
install-tools: install-webp-tools
	#$(GO) install github.com/vektra/mockery/v2@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest

# WebP 工具安装命令
.PHONY: install-webp-tools
install-webp-tools:
ifeq ($(OS),Darwin)
	@echo "Installing WebP tools on macOS..."
	@if ! command -v brew >/dev/null 2>&1; then \
		echo "Homebrew not found. Please install it first: https://brew.sh/"; \
		exit 1; \
	fi
	@brew install webp
else ifeq ($(OS),Linux)
	@echo "Installing WebP tools on Linux..."
	@if command -v apt-get >/dev/null 2>&1; then \
		sudo apt-get update && sudo apt-get install -y webp; \
	elif command -v yum >/dev/null 2>&1; then \
		sudo yum install -y libwebp-tools; \
	elif command -v dnf >/dev/null 2>&1; then \
		sudo dnf install -y libwebp-tools; \
	else \
		echo "Unsupported package manager. Please install WebP tools manually."; \
		exit 1; \
	fi
else ifeq ($(OS),Windows_NT)
	@echo "Installing WebP tools on Windows..."
	@if ! command -v choco >/dev/null 2>&1; then \
		echo "Chocolatey not found. Please install it first: https://chocolatey.org/"; \
		echo "Or download WebP tools manually from: https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html"; \
		exit 1; \
	fi
	@choco install webp
else
	@echo "Unsupported operating system: $(OS)"
	@echo "Please install WebP tools manually:"
	@echo "- macOS: brew install webp"
	@echo "- Linux: apt-get install webp / yum install libwebp-tools"
	@echo "- Windows: Download from https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html"
	@exit 1
endif
	@echo "WebP tools installation completed."
	@echo "Testing WebP tools..."
	@if command -v cwebp >/dev/null 2>&1; then \
		cwebp -version; \
	else \
		echo "WebP tools installation failed or not in PATH"; \
		exit 1; \
	fi

# 测试目标
.PHONY: test
test: fmt
	$(GOTEST) -tags=memory_storage_integration -timeout 5m -coverprofile $(BuildDir)/cover.out ./...

# 格式化目标
.PHONY: fmt
fmt: prepare
	#./scripts/import-order-cleanup.sh inplace
	@echo Running gofmt on ALL_SRC ...
	@$(GOFMT) -e -s -l -w $(ALL_SRC)
	@echo Running gofumpt on ALL_SRC ...
	@$(GOFUMPT) -e -l -w $(ALL_SRC)

# 代码检查目标
.PHONY: lint
lint:
	golangci-lint -v run

# 代码检查目标
vet: fmt
	@echo "Running go vet..."
	$(GO) vet ./...

# 清理项目
.PHONY: clean
clean:
	$(GO) clean -cache -testcache
	rm -f $(BuildDir) $(VersionBuildDir)

.PHONY: help
# 显示帮助信息
help:
	@echo "Makefile commands:"
	@echo "  build              构建目标"
	@echo "  build-image        构建 docker 镜像"
	@echo "  install            安装项目"
	@echo "  install-tools      安装工具"
	@echo "  install-webp-tools 安装 WebP 工具 (支持webp格式)"
	@echo "  fmt                格式化 Go 代码"
	@echo "  lint               运行 golangci-lint"
	@echo "  vet                运行 go vet"
	@echo "  clean              清理项目"
	@echo "  help               显示帮助信息"

# 打印所有包目标
echo-all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

echo-all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

.PHONY: cover
cover: nocover
	$(GOTEST) -tags=memory_storage_integration -timeout 5m -coverprofile $(BuildDir)/cover.out ./...
	grep -E -v 'model.pb.*.go' $(BuildDir)/cover.out > $(BuildDir)/cover-nogen.out
	mv $(BuildDir)/cover-nogen.out $(BuildDir)/cover.out
	go tool cover -html=$(BuildDir)/cover.out -o $(BuildDir)/cover.html

.PHONY: nocover
nocover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(ALL_PKGS)

# 生成 changelog 目标
.PHONY: changelog
changelog:
	./scripts/release-notes.py --exclude-dependabot

.PHONY: draft-release
draft-release:
	./scripts/draft-release.py

.PHONY: certs
certs:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh

.PHONY: certs-dryrun
certs-dryrun:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh -d
