# Makefile for managing the Go microservices project

ifeq ($(OS),Windows_NT)
    IS_WINDOWS := 1
endif

# load environment variables from .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

CURRENT_DIR	:= $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
ROOT_DIR	:= $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

SRCS_MK		:= $(foreach dir, app, $(wildcard $(dir)/*/*/Makefile))

.PHONY: help wire gen ent build api openapi init all vendor dep test cover vet lint docker \
		install-dev install-prod docker-up docker-down docker-libs pm2-deploy

# show environment variables
env:
	@echo "CURRENT_DIR: $(CURRENT_DIR)"
	@echo "ROOT_DIR: $(ROOT_DIR)"
	@echo "SRCS_MK: $(SRCS_MK)"

# initialize develop environment
init: plugin cli

# install protoc plugin
plugin:
	# go
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	@go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	@go install github.com/envoyproxy/protoc-gen-validate@latest
	@go install github.com/menta2k/protoc-gen-redact/v3@latest
	@go install github.com/go-kratos/protoc-gen-typescript-http@latest

# install cli tools
cli:
	@go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	@go install github.com/google/gnostic@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install entgo.io/ent/cmd/ent@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@go install github.com/tx7do/kratos-cli/config-exporter/cmd/cfgexp@latest
	@go install github.com/tx7do/kratos-cli/sql-orm/cmd/sql2orm@latest
	@go install github.com/tx7do/kratos-cli/sql-proto/cmd/sql2proto@latest
	@go install github.com/tx7do/kratos-cli/sql-kratos/cmd/sql2kratos@latest
	@go install github.com/tx7do/kratos-cli/gowind/cmd/gow@latest

# download dependencies of module
dep:
	@go mod download

# create vendor
vendor:
	@go mod vendor

# run tests
test:
	@go test ./...

# run coverage tests
cover:
	@go test -v ./... -coverprofile=coverage.out

# run static analysis
vet:
	@go vet

# run lint
lint:
	@golangci-lint run

# generate wire code
wire:
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make wire;\
    )

# generate ent code
ent:
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make ent;\
    )

# generate code
gen: ent wire api openapi

# generate protobuf api go code
api:
	@cd api && \
	buf generate

# generate protobuf api OpenAPI v3 docs.
openapi:
	@cd api && \
	buf generate --template buf.admin.openapi.gen.yaml

# generate protobuf api Typescript code
ts:
	@cd api && \
	buf generate --template buf.vue.admin.typescript.gen.yaml

# build all service applications
build: api openapi
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make build;\
    )

# only build all service applications without generating api and openapi
build_only:
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make build_only;\
    )

# export configuration to etcd
export:
	@cfgexp \
		--type=etcd \
		--addr=localhost:2379 \
		--proj=$(PROJECT_NAME)

# generate & build all service applications
all:
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make app;\
    )

# use docker compose to run backend services and all its dependency services like redis, mysql, etc.
compose-up:
	@docker compose up -d --force-recreate

# use docker compose to restart backend services and all its dependency services like redis, mysql, etc.
compose-restart:
	@docker compose restart

# use docker compose to down backend services and all its dependency services like redis, mysql, etc.
compose-down:
	@docker compose down

# use docker compose to run only dependency services like redis, mysql, etc. without backend services.
compose-up-without-service:
	@docker compose -f `docker-compose-without-services.yaml` up -d

# build docker image
docker:
	$(foreach dir, $(dir $(realpath $(SRCS_MK))),\
      cd $(dir);\
      make docker;\
    )

# ============================================================================
# Script Commands - 脚本命令
# ============================================================================

# install development environment (Unix/Linux/macOS)
install-dev:
	@echo "Installing development environment..."
	@bash scripts/env/install_unix_dev.sh

# install production environment (Unix/Linux/macOS)
install-prod:
	@echo "Installing production environment..."
	@bash scripts/env/install_unix_prod.sh

# install golang only
install-golang:
	@echo "Installing Golang..."
	@bash scripts/env/install_golang.sh

# start all services with docker compose (application + dependencies)
docker-up:
	@echo "Starting all services (application + dependencies)..."
ifdef IS_WINDOWS
	@powershell -ExecutionPolicy Bypass -File scripts/docker/full_deploy.ps1
else
	@bash scripts/docker/full_deploy.sh
endif

# start only dependency services with docker compose (without application)
docker-libs:
	@echo "Starting dependency services only..."
ifdef IS_WINDOWS
	@powershell -ExecutionPolicy Bypass -File scripts/docker/libs_only.ps1
else
	@bash scripts/docker/libs_only.sh
endif

# stop all docker compose services
docker-down:
	@echo "Stopping all services..."
	@docker compose down

# deploy services with PM2
pm2-deploy:
	@echo "Deploying services with PM2..."
	@bash scripts/deploy/pm2_service.sh

# show help
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
