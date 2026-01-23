export GO111MODULE=on
APP=kerberos
APP_EXECUTABLE="./out/$(APP)"
ALL_PACKAGES=$(shell go list ./... | grep -v /vendor)
SHELL := /bin/bash # Use bash syntax

# Optional colors to beautify output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

generate:
	sqlc generate

## Quality
check-quality: ## runs code quality checks
	make lint
	make fmt
	make vet

# Append || true below if blocking local developement
lint: ## go linting. Update and use specific lint tool and options
	golangci-lint run

vet: ## go vet
	go vet ./...

fmt: ## runs go formatter
	go fmt ./...

tidy: ## runs tidy to fix go.mod dependencies
	go mod tidy

vendor: ## runs go mod vendor
	go mod vendor

## Test
test: ## runs tests and create generates coverage report
	make tidy
	make vendor
	gotest -v -timeout 10m ./... -coverprofile=coverage.out -json > report.json

coverage: ## displays test coverage report in html mode
	make test
	go tool cover -html=coverage.out

## Build
build: ## build the go application
	mkdir -p out/
	go build -o $(APP_EXECUTABLE) ./cmd/kdc
	@echo "Build passed"

run: ## runs the go binary. usage: make run [args...]
	make build
	chmod +x $(APP_EXECUTABLE)
	$(APP_EXECUTABLE) $(filter-out $@,$(MAKECMDGOALS)) $(ARGS)

docs: ## builds the documentation
	$(MAKE) -C docs

clean: ## cleans binary and other generated files
	go clean
	rm -rf out/
	rm -f coverage*.out

.PHONY: all test build vendor docs
## All
all: ## runs setup, quality checks and builds
	make check-quality
	make test
	make build

.PHONY: help
## Help
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

ifneq ($(filter run,$(MAKECMDGOALS)),)
%:
		@:
endif
