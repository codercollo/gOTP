# GoTP — Makefile

.PHONY: help run/api build/api audit test tidy

help:
	@echo 'Usage:'
	@echo '  make run/api      run the API server'
	@echo '  make build/api    build a static binary into ./bin/api'
	@echo '  make test         run tests'
	@echo '  make audit        vet + race tests'
	@echo '  make tidy         format and tidy modules'

## run/api: run the API server
run/api:
	go run ./cmd/api

## build/api: build a static binary
build/api:
	CGO_ENABLED=0 go build -ldflags='-s -w' -o ./bin/api ./cmd/api

## test: run all tests
test:
	go test -v ./...

## audit: vet and race tests
audit:
	go vet ./...
	go test -race ./...

## tidy: format code and tidy modules
tidy:
	go fmt ./...
	go mod tidy
