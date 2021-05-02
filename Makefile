TARGET := git-po-helper
PKG := github.com/git-l10n/git-po-helper
VENDOR_EXISTS=$(shell test -d vendor && echo 1 || echo 0)
ifeq ($(VENDOR_EXISTS), 1)
    GOBUILD := GO111MODULE=on CGO_ENABLED=0 go build -mod=vendor
    GOTEST := GO111MODULE=on go test -mod=vendor
else
    GOBUILD := GO111MODULE=on CGO_ENABLED=0 go build
    GOTEST := GO111MODULE=on go test
endif

ifeq ($(shell uname), Darwin)
    TAR=gtar
else
    TAR=tar
endif

GOBUILD_LINUX_64 := env GOOS=linux GOARCH=amd64 $(GOBUILD)
GOBUILD_LINUX_32 := env GOOS=linux GOARCH=386 $(GOBUILD)
GOBUILD_WINDOWS_64 := env GOOS=windows GOARCH=amd64 $(GOBUILD)
GOBUILD_WINDOWS_32 := env GOOS=windows GOARCH=386 $(GOBUILD)
GOBUILD_MAC_64 := env GOOS=darwin GOARCH=amd64 $(GOBUILD)
GOBUILD_MAC_32 := env GOOS=darwin GOARCH=386 $(GOBUILD)

BUILD_RELEASE_FLAG=-ldflags "-s -w"

SHA256SUM=shasum -a 256
GPGSIGN=gpg -sba -u Alibaba
# Returns a list of all non-vendored (local packages)
LOCAL_PACKAGES = $(shell go list ./... | grep -v -e '^$(PKG)/vendor/')
LOCAL_GO_FILES = $(shell find -L $BUILD_DIR  -name "*.go" -not -path "$(PKG_BUILD_DIR)/vendor/*" -not -path "$(PKG_BUILD_DIR)/_build/*")

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
RELEASE_LDFLAGS := -ldflags "-X $(PKG)/version.Version=$(VERSION) -s -w"

go-gen:
	$(call message,Generate code for iso-639 and iso-3166)
	go generate github.com/git-l10n/git-po-helper/data/...

git-po-helper: $(shell find . -name '*.go') | VERSION-FILE go-gen
	$(call message,Building $@)
	$(GOBUILD) $(LDFLAGS) -o $@

golint:
	$(call message,Testing git-po-helper using golint for coding style)
	@golint $(LOCAL_PACKAGES)

test: golint ut it

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
