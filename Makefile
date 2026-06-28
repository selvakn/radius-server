BINARY       := radius-server
CMD          := ./cmd/server
BIN_DIR      := bin
BUILD_FLAGS  := -ldflags="-s -w" -trimpath
DOCKER_IMAGE ?= ghcr.io/selvakn/radius-server

export GOCACHE    ?= /tmp/gocache
export GOPATH     ?= /tmp/gopath
export GOMODCACHE ?= /tmp/gomodcache

.PHONY: build build-linux test lint gosec run clean check docker-build docker-push

build:
	mkdir -p $(BIN_DIR)
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY) $(CMD)

build-linux:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-linux-amd64 $(CMD)

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

gosec:
	golangci-lint run --enable-only gosec ./...

run:
	go run $(CMD) --config config.yaml

clean:
	rm -rf $(BIN_DIR) coverage.out

check: lint test

docker-build:
	docker build -t $(DOCKER_IMAGE):latest .

docker-push:
	docker push $(DOCKER_IMAGE):latest
