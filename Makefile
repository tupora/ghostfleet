SHELL := /bin/sh

.PHONY: build check dev-down dev-up fmt test vet

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
	go vet ./...
	go test -race ./...
	go build ./...

dev-up:
	docker compose up -d --wait

dev-down:
	docker compose down

