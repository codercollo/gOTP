# GoTP — Makefile

.DEFAULT_GOAL := help
.PHONY: help run/api build/api audit test tidy

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  make /'

## run/api: run the API server
run/api:
	-go run ./cmd/api

## build/api: build a static binary into ./bin/api
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

# Catch-all: unknown target -> error + help
%:
	@echo "make: unknown command '$@'"
	@echo
	@$(MAKE) --no-print-directory help
	@exit 1