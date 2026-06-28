BINARY := zeb
BUILD_DIR ?= bin
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build install-local uninstall-local test fmt vet spec-sync clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/zeb

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
	rm -rf $(BUILD_DIR)
