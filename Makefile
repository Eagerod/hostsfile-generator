GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hostsfile-daemon
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hostsfile-generator
PKG_PACKAGE_DIR := ./pkg/*
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

AUTOGEN_VERSION_FILENAME=$(CMD_PACKAGE_DIR)/version-temp.go

SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(AUTOGEN_VERSION_FILENAME)

# Publish targets are treated as phony to force rebuilds.
PUBLISH_DIR=publish
PUBLISH := \
	$(PUBLISH_DIR)/linux-amd64 \
	$(PUBLISH_DIR)/darwin-amd64 \
	$(PUBLISH_DIR)/darwin-arm64

.PHONY: $(PUBLISH)

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BIN_NAME) $(MAIN_FILE)


.PHONY: publish
publish: $(PUBLISH)

$(PUBLISH):
	rm -f $(BIN_NAME)
	GOOS_GOARCH="$$(basename $@)" \
	GOOS="$$(cut -d '-' -f 1 <<< "$$GOOS_GOARCH")" \
	GOARCH="$$(cut -d '-' -f 2 <<< "$$GOOS_GOARCH")" \
		$(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
	mv $(BIN_NAME) $@

.PHONY: install isntall
install isntall: $(INSTALLED_NAME)

$(INSTALLED_NAME): $(BIN_NAME)
	cp $(BIN_NAME) $(INSTALLED_NAME)

.PHONY: test
test: $(SRC)
	@if [ -z $$T ]; then \
		$(GO) test -v $(PACKAGE_PATHS); \
	else \
		$(GO) test -v $(PACKAGE_PATHS) -run $$T; \
	fi

.PHONY: system-test
system-test: $(BIN_NAME)
	@if [ -z $$T ]; then \
		$(GO) test -v main_test.go; \
	else \
		$(GO) test -v main_test.go -run $$T; \
	fi

.PHONY: test-cover
test-cover: $(SRC)
	$(GO) test -v --coverprofile=coverage.out $(PACKAGE_PATHS)

.PHONY: coverage
coverage: test-cover
	$(GO) tool cover -func=coverage.out

.INTERMEDIATE: $(AUTOGEN_VERSION_FILENAME)
$(AUTOGEN_VERSION_FILENAME):
	@version="$${VERSION:-$$(git describe --dirty)}"; \
	printf "package cmd\n\nconst VersionBuild = \"%s\"" $$version > $@

.PHONY: pretty-coverage
pretty-coverage: test-cover
	$(GO) tool cover -html=coverage.out

.PHONY: fmt
fmt:
	@$(GO) fmt ./...

.PHONY: fmt-check
fmt-check:
	@if [ ! -z "$$($(MAKE) -s fmt)" ]; then \
		exit 1; \
	fi

.PHONY: clean
clean:
	rm -rf coverage.out $(BUILD_DIR)

.PHONY: container
container: $(BIN_NAME)
	@version="$$(git describe --dirty | sed 's/^v//')"; \
	docker build . --build-arg VERSION="$$version" -t "registry.internal.aleemhaji.com/hostsfile-daemon:$$version"
