export GO111MODULE=on
APP=kerberos
KDC_EXECUTABLE="./out/kdc"
CLIENT_EXECUTABLE="./out/client"
API_EXECUTABLE="./out/api"
KADMIN_EXECUTABLE="./out/kadmin"
ALL_PACKAGES=$(shell go list ./... | grep -v /vendor)
SHELL := /bin/bash # Use bash syntax

# Optional colors to beautify output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build vendor docs

## All
all: ## runs setup, quality checks and builds
	@$(MAKE) check-quality
	@$(MAKE) test
	@$(MAKE) build

generate:
	sqlc generate

## Quality
check-quality: ## runs code quality checks
	@$(MAKE) lint
	@$(MAKE) fmt
	@$(MAKE) vet

lint: ## go linting
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
	@$(MAKE) tidy
	@$(MAKE) vendor
	gotest -v -timeout 10m ./... -coverprofile=coverage.out -json > report.json

coverage: ## displays test coverage report in html mode
	@$(MAKE) test
	go tool cover -html=coverage.out

## Build
build: ## build all the go applications
	@mkdir -p out/
	@for dir in cmd/*; do \
		if [ -d "$$dir" ]; then \
			name=$$(basename $$dir); \
			echo "Building $$name..."; \
			go build -o out/$$name ./$$dir; \
		fi \
	done
	@echo "${GREEN}Build passed${RESET}"

## Run
kdc:
	@$(MAKE) build
	$(KDC_EXECUTABLE) $(filter-out $@,$(MAKECMDGOALS)) $(ARGS)

api: ## runs the kdc binary
	@$(MAKE) build
	$(API_EXECUTABLE) $(filter-out $@,$(MAKECMDGOALS)) $(ARGS)

client:
	@$(MAKE) build
	$(CLIENT_EXECUTABLE) $(filter-out $@,$(MAKECMDGOALS)) $(ARGS)

kadmin:
	@$(MAKE) build
	$(KADMIN_EXECUTABLE) $(filter-out $@,$(MAKECMDGOALS)) $(ARGS)

docs: ## builds the documentation
	$(MAKE) -C docs

clean: ## cleans binary and other generated files
	go clean
	rm -rf out/
	rm -f coverage*.out
	@$(MAKE) -C docs clean

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
