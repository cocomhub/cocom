SHELL := /bin/bash
GO = go

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
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
GOCACHE=$(abspath .gocache)
GOBUILD="GOCACHE=$(GOCACHE)" CGO_ENABLED=0 installsuffix=cgo $(GO) build -trimpath
GOTEST="GOCACHE=$(GOCACHE)" $(GO) test -v $(RACE)
GOFMT=gofmt
GOFUMPT=gofumpt
FMT_LOG=.fmt.log
IMPORT_LOG=.import.log

GIT_SHA=$(shell git rev-parse HEAD)
GIT_CLOSEST_TAG=$(shell git describe --abbrev=0 --tags)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
ifneq ($(GIT_CLOSEST_TAG),$(shell echo ${GIT_CLOSEST_TAG} | grep -E "$(semver_regex)"))
	$(warning GIT_CLOSEST_TAG=$(GIT_CLOSEST_TAG) is not in the semver format $(semver_regex))
endif
TZ=UTC-8
DATE=$(shell TZ=${TZ} git show --quiet --date='format-local:%Y-%m-%dT%H:%M:%SZ' --format="%cd")
BUILD_AT=$(shell TZ=${TZ} date +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TIME=$(shell TZ=${TZ} date +"%Y%m%d%H%M%S")
BUILD_INFO_IMPORT_PATH=github.com/suixibing/cocom/pkg/version
BUILD_INFO=-ldflags '-X $(BUILD_INFO_IMPORT_PATH).CommitSHA=$(GIT_SHA) \
 -X $(BUILD_INFO_IMPORT_PATH).LatestVersion=$(GIT_CLOSEST_TAG) \
 -X $(BUILD_INFO_IMPORT_PATH).Branch=$(GIT_BRANCH) \
 -X $(BUILD_INFO_IMPORT_PATH).Date=$(DATE) \
 -X $(BUILD_INFO_IMPORT_PATH).BuildAt=$(BUILD_AT)'

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

.DEFAULT_GOAL := test-and-lint

.PHONY: test-and-lint
test-and-lint: test fmt lint

# TODO: no files actually use this right now
.PHONY: go-gen
go-gen:
	@echo skipping go generate ./...

.PHONY: clean
clean:
	rm -rf cover.out .cover/ cover.html
    GOCACHE=$(GOCACHE) go clean -cache -testcache

.PHONY: build
build: go-gen
	$(GO) build $(BUILD_INFO) -o cocom

install: build
	cp cocom ~/bin
	cocom completion zsh > ~/.cocom/zsh_completion
	source ~/.cocom/zsh_completion

.PHONY: build-image
build-image:
	docker build . -t suixibing/cocom-server:$(BUILD_TIME)

.PHONY: build-image-tag
build-image-tag:
	docker build . -t suixibing/cocom-server:$(GIT_CLOSEST_TAG)

.PHONY: test
test: go-gen

echo-all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort

echo-all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

.PHONY: cover
cover: nocover
	$(GOTEST) -tags=memory_storage_integration -timeout 5m -coverprofile cover.out ./...
	grep -E -v 'model.pb.*.go' cover.out > cover-nogen.out
	mv cover-nogen.out cover.out
	go tool cover -html=cover.out -o cover.html

.PHONY: nocover
nocover:
	@echo Verifying that all packages have test files to count in coverage
	@scripts/check-test-files.sh $(ALL_PKGS)

.PHONY: fmt
fmt: install-tools
	#./scripts/import-order-cleanup.sh inplace
	@echo Running gofmt on ALL_SRC ...
	@$(GOFMT) -e -s -l -w $(ALL_SRC)
	@echo Running gofumpt on ALL_SRC ...
	@$(GOFUMPT) -e -l -w $(ALL_SRC)
	./scripts/updateLicenses.sh

.PHONY: lint
lint:
	golangci-lint -v run

.PHONY: changelog
changelog:
	./scripts/release-notes.py --exclude-dependabot

.PHONY: draft-release
draft-release:
	./scripts/draft-release.py

.PHONY: install-tools
install-tools:
	#$(GO) install github.com/vektra/mockery/v2@v2.14.0
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.1
	$(GO) install mvdan.cc/gofumpt@latest

.PHONY: install-ci
install-ci: install-tools

.PHONY: echo-version
echo-version:
	@echo $(GIT_CLOSEST_TAG)

.PHONY: certs
certs:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh

.PHONY: certs-dryrun
certs-dryrun:
	cd pkg/config/tlscfg/testdata && ./gen-certs.sh -d