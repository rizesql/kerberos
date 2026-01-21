# Optional colors to beautify output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

# LaTeX configuration
MAIN_TEX := main.tex
LATEXMK  := latexmk

.PHONY: all build run clean help

## Build
all: ## builds the documentation
	make build

build: ## compiles the latex document once
	$(LATEXMK) $(MAIN_TEX)
	@echo "${GREEN}Build passed${RESET}"

run: ## watches for changes and recompiles (live preview)
	$(LATEXMK) -pvc $(MAIN_TEX)

## Maintenance
clean: ## cleans latex auxiliary files
	$(LATEXMK) -c
	rm -f *.pdf
	@echo "${GREEN}Cleaned up${RESET}"

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
