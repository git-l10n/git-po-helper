PKG := github.com/git-l10n/git-po-helper
TARGET := git-po-helper
VENDOR_EXISTS=$(shell test -d vendor && echo 1 || echo 0)
ifeq ($(VENDOR_EXISTS), 1)
    GOBUILD := GO111MODULE=on go build -mod=vendor
    GOTEST := GO111MODULE=on go test -mod=vendor
else
    GOBUILD := GO111MODULE=on go build
    GOTEST := GO111MODULE=on go test
endif

ifeq ($(shell uname), Darwin)
    CC := clang
endif

## Exhaustive lists of our source files, either dynamically generated,
## or hardcoded.
SOURCES_CMD = ( \
        git ls-files \
                '*.go' \
                ':!test' \
                ':!contrib' \
                2>/dev/null || \
        $(FIND) . \
                \( -name .git -type d -prune \) \
                -o \( -name test -type d -prune \) \
                -o \( -name contrib -type d -prune \) \
                -o \( -name '*.go' -type f -print \) \
                | sed -e 's|^\./||' \
        )
FOUND_SOURCE_FILES := $(shell $(SOURCES_CMD))

# Returns a list of all non-vendored (local packages)
LOCAL_PACKAGES = $(shell go list ./... | grep -v -e '^$(PKG)/vendor/')

define message
	@echo "### $(1)"
endef

all: $(TARGET)

VERSION-FILE: FORCE
	$(call message,Generate version file)
	@/bin/sh ./VERSION-GEN
-include VERSION-FILE

# Define LDFLAGS after include of REPO-VERSION-FILE
LDFLAGS := -ldflags "-X $(PKG)/version.Version=$(VERSION)"

git-po-helper: $(FOUND_SOURCE_FILES) data/iso-3166.go data/iso-639.go | VERSION-FILE
	$(call message,Building $@)
	$(GOBUILD) $(LDFLAGS) -o $@

data/iso-3166.go: data/iso-3166.csv data/iso-3166.t
	$(call message,Generate code for iso-3166 and iso-639)
	go generate github.com/git-l10n/git-po-helper/data/...

data/iso-639.go: data/iso-639.csv data/iso-639.t
	$(call message,Generate code for iso-639 and iso-3166)
	go generate github.com/git-l10n/git-po-helper/data/...

.PHONY: golint lint
golint: lint
lint:
	$(call message,Testing using static analysis tools for better coding style)
	go vet ./...
	staticcheck -checks all ./...

test: $(TARGET) lint ut it

ut: $(TARGET)
	$(call message,Testing git-po-helper for unit tests)
	$(GOTEST) $(PKG)/...

it: $(TARGET)
	$(call message,Testing git-po-helper for integration tests)
	@make -C test

clean:
	$(call message,Cleaning $(TARGET))
	@rm -f $(TARGET)
	@rm -f VERSION-FILE

.PHONY: test clean
.PHONY: go-gen
.PHONY: FORCE
.PHONY: ut it
