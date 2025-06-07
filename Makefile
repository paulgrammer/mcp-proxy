PROXY_CMD_PATH = "./cmd/proxy"
CGO_ENABLED = 0
export CGO_ENABLED

# Set up OS specific bits
ifeq ($(OS),Windows_NT)
	CMD_SUFFIX = .exe
	NULL_FILE = nul
else
	CMD_SUFFIX =
	NULL_FILE = /dev/null
endif

# Define build number as last commit hash
ifndef BUILD_NUMBER
	BUILD_NUMBER = $(shell git rev-parse --short HEAD)
endif

LDFLAGS = -X main.Build=$(BUILD_NUMBER)

ALL_LINUX = linux-amd64 \
	linux-arm64 \
	linux-386

ALL = $(ALL_LINUX) \
	darwin-amd64 \
	darwin-arm64 \
	windows-amd64

all: $(ALL:%=build/%/proxy)

BUILD_ARGS += -trimpath

bin:
	go build $(BUILD_ARGS) -ldflags "$(LDFLAGS)" -o ./proxy${CMD_SUFFIX} ${PROXY_CMD_PATH}

install:
	go install $(BUILD_ARGS) -ldflags "$(LDFLAGS)" ${PROXY_CMD_PATH}

build/%/proxy: .FORCE
	GOOS=$(firstword $(subst -, , $*)) \
		GOARCH=$(word 2, $(subst -, ,$*)) \
		go build $(BUILD_ARGS) -o $@ -ldflags "$(LDFLAGS)" ${PROXY_CMD_PATH}

build/%/proxy.exe: build/%/proxy
	mv $< $@

build/proxy-%.tar.gz: build/%/proxy
	tar -zcv -C build/$* -f $@ proxy

test:
	go test -v ./... -race -cover

dev:
	go run github.com/air-verse/air@latest \
	--build.cmd "CGO_ENABLED=1 go build -o ./tmp/main cmd/proxy/main.go" \
	--build.bin "./tmp/main -config ./config.yml" \
	--build.exclude_dir "web,build"

dev-web:
	cd web && bun run dev

generate:
	go generate ./...

.FORCE:

.PHONY: dev bin test dev-web build-web generate

.DEFAULT_GOAL := dev
