.PHONY: help

SHELL := /bin/bash
VERSION=$(shell git describe --tags)
NOW=$(shell date +"%Y-%m-%d_%H:%M:%S")
COMMIT_REF=$(shell git rev-parse --short HEAD)
BUILD_ARGS=-ldflags "-X github.com/timo-reymann/ContainerHive/internal/buildinfo.GitSha=$(COMMIT_REF) -X github.com/timo-reymann/ContainerHive/internal/buildinfo.Version=$(VERSION) -X github.com/timo-reymann/ContainerHive/internal/buildinfo.BuildTime=$(NOW)" -tags prod
BIN_PREFIX="dist/"
CMD_CH_CLI = "./cmd/ch"

clean: ## Cleanup artifacts
	@rm -rf dist/

help: ## Display this help page
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-30s\033[0m %s\n", $$1, $$2}'

coverage: ## Run tests and measure coverage
	@go test -covermode=count -coverprofile=/tmp/count.out -v ./...

test-coverage-report: coverage ## Run test and display coverage report in browser
	@go tool cover -html=/tmp/count.out

save-coverage-report: coverage ## Save coverage report to coverage.html
	@go tool cover -html=/tmp/count.out -o coverage.html

create-dist: ## Create dist folder if not already existent
	@mkdir -p dist/
	@mkdir -p dist/linux-amd64
	@mkdir -p dist/linux-arm64
	@mkdir -p dist/darwin-arm64
	@mkdir -p dist/darwin-amd64

build-linux: create-dist ## Build binaries for linux
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BIN_PREFIX)/linux-amd64/ch $(BUILD_ARGS) $(CMD_CH_CLI)
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BIN_PREFIX)linux-arm64/ch $(BUILD_ARGS) $(CMD_CH_CLI)

build-darwin: create-dist  ## Build binaries for macOS
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BIN_PREFIX)darwin-amd64/ch $(BUILD_ARGS) $(CMD_CH_CLI)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BIN_PREFIX)darwin-arm64/ch $(BUILD_ARGS) $(CMD_CH_CLI)

build-image-container-hive: ## Build the container hive image
	@docker buildx build . \
		-t timoreymann/container-hive:${VERSION} \
		-t timoreymann/container-hive:latest \
		-f Dockerfile \
		--platform linux/amd64,linux/arm64 \
		--push

create-checksums: ## Create checksums for binaries
	@find ./dist/*/* -type f -exec sh -c 'sha256sum {} | cut -d " " -f 1 > {}.sha256' {} \;

build-image: build-image-container-hive ## Build all images

build: create-dist build-linux build-darwin ## Build binaries for all platform

bundle:
	@cd dist/ && find * -type d -exec sh -c 'cd {} && cp ../../LICENSE LICENSE.txt && cp ../../NOTICE . && tar -cf - . | zstd -9 -o ../{}.tar.zst' \;
	@find ./dist/*.tar.zst -type f -exec sh -c 'sha256sum {} | cut -d " " -f 1 > {}.sha256' {} \;

pack: create-checksums bundle ## Create checksums and pack archives for delivery

generate-json-schema: ## Generate the json schemas
	@go run tools/generate-image-schema.go
	@go run tools/generate-project-schema.go

build-docker: ## Build docker image based on the built linux builds in the dist folder
	@docker buildx build --tag timoreymann/containerhive:latest \
		--platform linux/amd64,linux/arm/v7,linux/arm64 \
		--build-arg BUILD_TIME="$(NOW)" \
		--build-arg BUILD_VERSION="$(VERSION)" \
		--build-arg BUILD_COMMIT_REF="$(COMMIT_REF)" \
		--push .
	@docker buildx build --tag timoreymann/containerhive:$(VERSION) \
		--platform linux/amd64,linux/arm/v7,linux/arm64 \
		--build-arg BUILD_TIME="$(NOW)" \
		--build-arg BUILD_VERSION="$(VERSION)" \
		--build-arg BUILD_COMMIT_REF="$(COMMIT_REF)" \
		--push .