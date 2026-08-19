SHELL := /bin/sh

.PHONY: api-breaking api-check api-generate api-lint api-tools build check dev-down dev-up fmt test vet

BUF := go run github.com/bufbuild/buf/cmd/buf@v1.50.0
API_TOOLS := $(CURDIR)/.tools/bin
API_REMOTE ?= $(or $(shell git config --get remote.origin.url),https://github.com/tupora/ghostfleet.git)

api-tools:
	mkdir -p $(API_TOOLS)
	GOBIN=$(API_TOOLS) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	GOBIN=$(API_TOOLS) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

api-generate: api-tools
	PATH="$(API_TOOLS):$$PATH" $(BUF) generate

api-lint:
	$(BUF) lint

api-breaking:
	$(BUF) breaking --against "$(API_REMOTE)#branch=main,subdir=api/proto"

api-check: api-lint api-generate
	@git diff --exit-code -- api/gen || \
		(echo 'Generated API files are stale; run make api-generate' && exit 1)
	$(MAKE) api-breaking

build:
	mkdir -p bin
	go build -o bin/ghostfleet-controller ./cmd/ghostfleet-controller
	go build -o bin/ghostfleet-worker ./cmd/ghostfleet-worker
	go build -o bin/ghostcli ./cmd/ghostcli

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './api/gen/*')

vet:
	go vet ./...

test:
	go test -race -coverprofile=coverage.out ./...

check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './api/gen/*'))" || \
		(echo 'Go files need formatting; run make fmt' && exit 1)
	$(MAKE) api-check
	go vet ./...
	go test -race ./...
	go build ./...

dev-up:
	docker compose up -d --wait

dev-down:
	docker compose down
