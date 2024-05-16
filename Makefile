VERSION := $(shell git describe --tags 2>/dev/null)
GIT_HASH := $(shell git rev-parse --short HEAD)

GOOS :=
GOARCH :=
ENV := GOOS=${GOOS} GOARCH=${GOARCH}

LDFLAGS := \
	-X main.version=${VERSION} \
	-X main.gitHash=${GIT_HASH}
FLAGS := -ldflags='${LDFLAGS}'

.PHONY: build
build: modular-indexer-committee

modular-indexer-committee:
	env ${ENV} go build ${FLAGS} -o $@

config.json: config.example.json
	cp config.example.json config.json

.PHONY: ci
ci: config.json
	go run github.com/RiemaLabs/nubit-ci/cmd/nubitci-lint@latest

.PHONY: ci-fix
ci-fix: config.json
	go run github.com/RiemaLabs/nubit-ci/cmd/nubitci-lint@latest -w

.PHONY: clean
clean:
	rm -rf ./modular-indexer-committee
