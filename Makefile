# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

PROJECT_NAME := cocom
SHELL := /bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# STANDARD VARIABLES — 所有项目一致
# ═══════════════════════════════════════════════════════════════════════════════
BUILD_DIR       ?= build
BIN_DIR         ?= $(BUILD_DIR)/bin
RAW_GO          ?= go
GOOS            ?= $(shell $(RAW_GO) env GOOS)
GOARCH          ?= $(shell $(RAW_GO) env GOARCH)
HOST_GOARCH     ?= $(shell $(RAW_GO) env GOHOSTARCH)
EXE             :=
GO              := GOOS=$(GOOS) GOARCH=$(GOARCH) $(RAW_GO)
GORACE          := -race
GOTEST_COUNT    ?= -count=1
GOTEST_TIMEOUT  ?= -timeout=5m
NOTEST_IGNORE   := .notestignore
SUB_MODULE_DIRS := $(shell find . -name 'go.mod' \
  -not -path './$(BUILD_DIR)/*' \
  -not -path './.claude/*' \
  -not -path './vendor/*' \
  -exec dirname {} \; | sort -u | grep -v '^\.$$')

# ═══════════════════════════════════════════════════════════════════════════════
# CUSTOM VARIABLES — 本项目按需配置
# ═══════════════════════════════════════════════════════════════════════════════
COVER_THRESHOLD ?= 20
SONAR_PROJECT_KEY ?= cocomhub_cocom
SKIP_VERSION    ?= false
VERSION_DIR     ?= pkg/version/build
GOTAGS          ?= -tags=memory_storage_integration
GOBUILD_EXTRA   ?= -trimpath -v
GO_LDFLAGS      := -ldflags "\
  -X github.com/cocomhub/cocom/pkg/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
  -X github.com/cocomhub/cocom/pkg/version.BuiltAt=$(shell date +"%Y-%m-%dT%H:%M:%SZ") \
  -X github.com/cocomhub/cocom/pkg/version.CommitID=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) \
  -X github.com/cocomhub/cocom/pkg/version.Branch=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown) \
  -X github.com/cocomhub/cocom/pkg/version.ReleaseURL=https://github.com/cocomhub/cocom/releases/tag/$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)"

# ═══════════════════════════════════════════════════════════════════════════════
# OTHER VARIABLES — 原有变量，保留不动
# ═══════════════════════════════════════════════════════════════════════════════
VersionImportPath := pkg/version
SUB_TOOL_DIRS := $(shell ls -d tools/*/main.go 2>/dev/null | xargs -I{} dirname {} | sort -u)
SUB_TOOL_NAMES := $(notdir $(SUB_TOOL_DIRS))
UNAME := $(shell uname -m)
ifeq ($(UNAME), s390x)
GORACE :=
endif
GOFMT := gofmt
ALL_SRC := $(shell find . -name '*.go' \
  -not -name 'doc.go' \
  -not -name '_*' \
  -not -name '.*' \
  -not -name 'mocks*' \
  -not -name 'model.pb.go' \
  -not -name 'model_test.pb.go' \
  -not -name 'storage_test.pb.go' \
  -not -path './$(BUILD_DIR)/*' \
  -not -path './examples/*' \
  -not -path './vendor/*' \
  -not -path './node_modules/*' \
  -not -path './.claude/*' \
  -not -path './.trae/*' \
  -not -path '*/mocks/*' \
  -not -path '*/*-gen/*' \
  -type f | sort)
ALL_PKGS := $(sort $(dir $(ALL_SRC)))

.DEFAULT_GOAL := help

# ═══════════════════════════════════════════════════════════════════════════════
# STANDARD TARGETS — 所有项目一致
# ═══════════════════════════════════════════════════════════════════════════════

.PHONY: prepare
prepare:
	@mkdir -p $(BUILD_DIR) $(BIN_DIR)
ifneq ($(SKIP_VERSION), true)
	@mkdir -p $(VERSION_DIR)
	@if ! git diff --quiet HEAD 2>/dev/null; then \
		git diff HEAD > $(VERSION_DIR)/dirty_info.txt 2>/dev/null; \
		echo "[prepare] dirty_info.txt updated ($(VERSION_DIR)/dirty_info.txt)"; \
	else \
		printf '' > $(VERSION_DIR)/dirty_info.txt; \
	fi
endif

.PHONY: build
build: fmt
	$(GO) build $(GOBUILD_EXTRA) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME) .

.PHONY: build-ci
build-ci: prepare
	$(GO) build $(GOBUILD_EXTRA) $(GO_LDFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME) .

.PHONY: test
test: prepare
	$(GO) test $(GORACE) $(GOTEST_COUNT) $(GOTEST_TIMEOUT) $(GOTAGS) ./...

.PHONY: test-ci test-cover
test-ci test-cover: prepare
	$(GO) test $(GORACE) $(GOTEST_COUNT) $(GOTEST_TIMEOUT) $(GOTAGS) -coverprofile=$(BUILD_DIR)/cover.out ./...

.PHONY: notest
notest:
	@scripts/check-test-files.sh $(ALL_PKGS)

.PHONY: cover-check
cover-check: test-cover
	@total=$$(go tool cover -func=$(BUILD_DIR)/cover.out | tail -1 | awk '{print $$NF}' | sed 's/%//'); \
	if [ -z "$$total" ]; then \
		echo "FAIL: could not compute coverage"; \
		exit 1; \
	fi; \
	if (( $$(echo "$$total < $(COVER_THRESHOLD)" | bc -l) )); then \
		echo "FAIL: coverage $$total% < threshold $(COVER_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "PASS: coverage $$total% >= threshold $(COVER_THRESHOLD)%"

.PHONY: vet
vet:
	$(RAW_GO) vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: bench
bench: prepare
	@mkdir -p $(BUILD_DIR)/bench
	$(GO) test -bench=. -benchmem -count=5 $(GOTAGS) ./... 2>&1 | tee $(BUILD_DIR)/bench/output.txt

.PHONY: bench-cpu
bench-cpu: prepare
	@mkdir -p $(BUILD_DIR)/bench
	$(GO) test -bench=. -cpuprofile -count=5 -memprofile $(BUILD_DIR)/bench/mem.prof ./... 2>&1 | tee $(BUILD_DIR)/bench/cpu.prof

.PHONY: bench-compare
bench-compare:
	@which benchstat > /dev/null 2>&1 || go install golang.org/x/perf/cmd/benchstat@latest
	@if [ -f $(BUILD_DIR)/bench/output.txt ] && [ -f $(BUILD_DIR)/bench/baseline.txt ]; then \
		benchstat $(BUILD_DIR)/bench/baseline.txt $(BUILD_DIR)/bench/output.txt; \
	else \
		echo "Need both output.txt and baseline.txt to compare"; \
		exit 1; \
	fi

.PHONY: check-loopback
check-loopback:
	@if grep -rn '0\.0\.0\.0' --include='*.go' . \
		| grep -v '_test.go' \
		| grep -v 'vendor/' \
		| grep -v 'testdata/' \
		| grep -v 'fixtures/' \
		| grep -v '\.pb\.go' \
		| grep -v 'docs/' \
		| grep -v '\.claude/' \
		| grep -v 'internal/config/' \
		| grep '.' > /dev/null 2>&1; then \
		echo "FAIL: found potential unsafe listen addresses (0.0.0.0)"; \
		grep -rn '0\.0\.0\.0' --include='*.go' . \
			| grep -v '_test.go' \
			| grep -v 'vendor/' \
			| grep -v 'testdata/' \
			| grep -v 'fixtures/' \
			| grep -v '\.pb\.go' \
			| grep -v 'docs/' \
			| grep -v '\.claude/' \
			| grep -v 'internal/config/'; \
		exit 1; \
	else \
		echo "OK: no unsafe loopback addresses found"; \
	fi

.PHONY: gofix
gofix:
	$(RAW_GO) fix ./...

.PHONY: addlicense
addlicense:
	addlicense -c "The Cocomhub Authors. All rights reserved." -s=only -ignore ".claude/**" -ignore ".trae/**" -ignore ".cursor/**" .

.PHONY: fmt
fmt: gofix addlicense
	@echo "Running gofmt on ALL_SRC ..."
	@$(GOFMT) -e -s -l -w $(ALL_SRC)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) $(VERSION_DIR)

.PHONY: test-all
test-all:
	@for dir in $(SUB_MODULE_DIRS); do \
		echo "=== Testing $$dir ==="; \
		cd $$dir && $(RAW_GO) test $(GORACE) $(GOTEST_COUNT) $(GOTEST_TIMEOUT) ./... || exit 1; \
		cd $(CURDIR); \
	done

.PHONY: build-all
build-all:
	@for dir in $(SUB_MODULE_DIRS); do \
		echo "=== Building $$dir ==="; \
		cd $$dir && $(RAW_GO) build ./... || exit 1; \
		cd $(CURDIR); \
	done

.PHONY: check-ci
check-ci: vet lint check-loopback notest build-ci test-cover cover-check test-all build-all

.PHONY: sonar-analyze
sonar-analyze:
	@if [ ! -f sonar-project.properties ]; then \
		echo "missing sonar-project.properties"; exit 1; \
	fi
	sonar-scanner

.PHONY: sonar-remediate
sonar-remediate:
	@if [ ! -f sonar-project.properties ]; then \
		echo "missing sonar-project.properties"; exit 1; \
	fi
	sonar-scanner -Dsonar.remediation.projectKey=$(SONAR_PROJECT_KEY)

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Standard targets:"
	@echo "  build           Build the project binary (depends on fmt)"
	@echo "  build-ci        Build without fmt, for CI"
	@echo "  test            Run tests (no coverage)"
	@echo "  test-ci         Run tests with coverage (alias: test-cover)"
	@echo "  test-cover      Run tests with coverage"
	@echo "  cover-check     Check coverage meets threshold"
	@echo "  notest          Verify all packages have test files"
	@echo "  vet             Run go vet"
	@echo "  lint            Run golangci-lint"
	@echo "  bench           Run benchmarks"
	@echo "  bench-cpu       Run benchmarks cpu"
	@echo "  check-loopback  Check for unsafe listen addresses"
	@echo "  gofix           Run go fix"
	@echo "  addlicense      Add license headers"
	@echo "  fmt             Format code (gofix + addlicense + gofmt)"
	@echo "  clean           Clean build artifacts"
	@echo "  test-all        Test all sub-modules"
	@echo "  build-all       Build all sub-modules"
	@echo "  check-ci        Full CI pipeline"
	@echo "  sonar-analyze   Run SonarQube Cloud analysis"
	@echo "  sonar-remediate Run SonarQube Cloud remediation"
	@echo ""
	@echo "Custom targets:"
	@echo "  build-sub-tools Build sub-tools"
	@echo "  cover-html      Generate coverage HTML report"
	@echo "  install         Install binary to ~/bin w/ completions"
	@echo "  run-server      Build and run the server"
	@echo "  test-e2e        Run E2E tests"
	@echo "  release         Release with goreleaser"
	@echo "  release-snapshot Snapshot release"
	@echo "  fmt-web         Format web code"
	@echo "  config-doc      Generate config reference"
	@echo "  changelog       Generate changelog"
	@echo "  certs           Generate test certificates"
	@echo "  build-image     Docker image build"
	@echo "  echo-all-pkgs   Print all packages"
	@echo "  echo-all-srcs   Print all source files"
	@echo "  list-sub-tools  List available sub-tools"

# ═══════════════════════════════════════════════════════════════════════════════
# CUSTOM TARGETS — 本项目特有
# ═══════════════════════════════════════════════════════════════════════════════

.PHONY: go-gen
go-gen:
	@echo "go-gen: no-op"

.PHONY: build-sub-tools
build-sub-tools: fmt
	@for dir in $(SUB_TOOL_DIRS); do \
		name=$$(basename $$dir); \
		echo "Building $$name..."; \
		$(GO) build $(GOBUILD_EXTRA) $(GO_LDFLAGS) -o $(BUILD_DIR)/$$name ./$$dir; \
	done

# 格式化 Web 代码
.PHONY: fmt-web
fmt-web:
	@echo Running fmt-web ...
	@npm run fmt-web

.PHONY: cover-html
cover-html: test-cover
	@go tool cover -html=$(BUILD_DIR)/cover.out -o $(BUILD_DIR)/cover.html
	@echo "Coverage report: $(BUILD_DIR)/cover.html"

# 安装目标
.PHONY: install
install: build
	@echo "Installing $(PROJECT_NAME)..."
	cp $(BUILD_DIR)/$(PROJECT_NAME) ~/bin/
	@mkdir -p ~/.$(PROJECT_NAME)
	@-./$(BUILD_DIR)/$(PROJECT_NAME) completion zsh > ~/.config/$(PROJECT_NAME)/zsh_completion 2>/dev/null || true
	@echo "Run 'source ~/.config/$(PROJECT_NAME)/zsh_completion' to enable zsh completion in your current shell."
	@echo "Installation completed."

.PHONY: run-server
run-server: build
	./$(BUILD_DIR)/$(PROJECT_NAME) server --config ./$(BUILD_DIR)/conf/cocom.yaml

# playwright E2E 端到端浏览器测试（独立 module，需 playwright + Chromium 环境）
.PHONY: test-e2e
test-e2e:
	cd tests/e2e && CGO_ENABLED=1 $(RAW_GO) test -count=1 -v -timeout 120s ./...

# playwright E2E 安装（首次运行前执行）
.PHONY: test-e2e-install
test-e2e-install:
	cd tests/e2e && $(RAW_GO) mod tidy
	cd tests/e2e && $(RAW_GO) run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

.PHONY: release
release:
	goreleaser check
	goreleaser release --clean

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

# ═══════════════════════════════════════════════════════════════════════════════
# OTHER TARGETS — 保留旧版特有功能以避免破坏性变更
# ═══════════════════════════════════════════════════════════════════════════════

# 构建 docker 镜像目标# 构建 docker 镜像目标
.PHONY: build-image
build-image:
	docker build . -t $(DOCKER_NAMESPACE)/$(PROJECT_NAME):$(shell git describe --tags --always --dirty || echo dev)

# 安装工具目标
.PHONY: install-tools
install-tools: install-ci-tools
	#$(GO) install github.com/vektra/mockery/v2@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 安装 Web 工具
.PHONY: install-web-tools
install-web-tools:
	@echo "Installing Web tools..."
	@npm install -g prettier
	@npm install -D prettier @trivago/prettier-plugin-sort-imports
	@npm install --save-dev stylelint stylelint-order

# 安装CI工具目标
.PHONY: install-ci-tools
install-ci-tools:
	$(RAW_GO) install mvdan.cc/gofumpt@latest
	$(RAW_GO) install github.com/google/addlicense@latest

# 通过 uname -s 检测系统类型（兼容 Linux/macOS/Windows）
UNAME_OS := $(shell uname -s)
# WebP 工具安装命令
.PHONY: install-webp-tools
install-webp-tools:
ifeq ($(UNAME_OS),Darwin)
	@echo "Installing WebP tools on macOS..."
	@if ! command -v brew >/dev/null 2>&1; then \
		echo "Homebrew not found. Please install it first: https://brew.sh/"; \
		exit 1; \
	fi
	@brew install webp
else ifeq ($(UNAME_OS),Linux)
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
# 保留 $(OS) 检测 Windows（GNU Make 在 Windows 上通过环境变量获取）
	@echo "Installing WebP tools on Windows..."
	@if ! command -v choco >/dev/null 2>&1; then \
		echo "Chocolatey not found. Please install it first: https://chocolatey.org/"; \
		echo "Or download WebP tools manually from: https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html"; \
		exit 1; \
	fi
	@choco install webp
else
	@echo "Unsupported operating system: $(UNAME_OS) (OS=$(OS))"
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

.PHONY: certs
certs:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh

.PHONY: certs-dryrun
certs-dryrun:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh -d

# 生成配置文档（扫描 Viper 键并生成 docs/config-reference.md）
.PHONY: config-doc
config-doc:
	@go generate ./tools/config-doc-gen/
	@echo "Config reference doc generated at docs/config-reference.md"

.PHONY: changelog
changelog:
	./scripts/release-notes.py --exclude-dependabot

# 打印所有包目标
.PHONY: echo-all-pkgs
echo-all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

.PHONY: echo-all-srcs
echo-all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

# 列出可用子工具（调试用）
.PHONY: list-sub-tools
list-sub-tools:
	@echo "Available tools:"
	@for tool in $(SUB_TOOL_NAMES); do \
		echo "  - $$tool"; \
	done
