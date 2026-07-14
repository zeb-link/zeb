BINARY := zeb
BUILD_DIR ?= bin
INSTALL_DIR ?= $(HOME)/.local/bin
VERSION ?= $(shell node -p "require('./npm/package.json').version")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build release-build npm-build npm-publish release-check install-local uninstall-local test fmt vet spec-sync clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/zeb

release-build:
	VERSION="$(VERSION)" scripts/build-release.sh

npm-build: release-build
	VERSION="$(VERSION)" scripts/build-npm.sh

npm-publish:
	scripts/publish-npm.sh

release-check: test npm-build
	scripts/publish-npm.sh

install-local:
	ZEB_INSTALL_DIR="$(INSTALL_DIR)" scripts/install-local.sh

uninstall-local:
	ZEB_INSTALL_DIR="$(INSTALL_DIR)" scripts/uninstall-local.sh

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

spec-sync:
	go run ./cmd/zeb spec sync

clean:
	rm -rf $(BUILD_DIR) dist npm/packages
