BINARY     := aide
CLI_DIR    := cli
BIN_DIR    := bin
SANDBOX    := $(HOME)/.aide-sandbox
SDK_PATH   := $(CURDIR)/sdk/python
SDK_DIR    := sdk/python
FRONTEND_DIR := cli/internal/agent/frontend

.PHONY: build dev clean verify go-lint go-test go-vuln py-lint py-type py-test fe-lint fmt

build:
	cd $(CLI_DIR) && go build -ldflags="-s -w" -o ../$(BIN_DIR)/$(BINARY) ./cmd/aide

dev: build
	@mkdir -p $(SANDBOX)/bin $(SANDBOX)/plugins $(SANDBOX)/data
	@cp $(BIN_DIR)/$(BINARY) $(SANDBOX)/bin/$(BINARY)
	@printf '#!/bin/sh\nexport AIDE_HOME="$(SANDBOX)"\nexport AIDE_SDK_PATH="$(SDK_PATH)"\nexec "$(SANDBOX)/bin/$(BINARY)" "$$@"\n' > $(SANDBOX)/bin/aide-dev
	@chmod +x $(SANDBOX)/bin/aide-dev
	@if [ ! -f $(SANDBOX)/config.yaml ]; then \
		AIDE_HOME=$(SANDBOX) $(SANDBOX)/bin/$(BINARY) init 2>/dev/null || true; \
	fi
	@echo ""
	@echo "  Sandbox ready: $(SANDBOX)"
	@echo "  Run:  AIDE_HOME=$(SANDBOX) AIDE_SDK_PATH=$(SDK_PATH) $(SANDBOX)/bin/$(BINARY) <command>"
	@echo "  Or:   $(SANDBOX)/bin/aide-dev <command>   (env pre-set)"
	@echo ""

clean:
	rm -f $(BIN_DIR)/$(BINARY)

fmt:
	cd $(CLI_DIR) && gofumpt -w .
	cd $(SDK_DIR) && ruff format .
	cd $(FRONTEND_DIR) && npx prettier --write src

go-lint:
	cd $(CLI_DIR) && golangci-lint run ./...

go-test:
	cd $(CLI_DIR) && go test -race ./...

go-vuln:
	cd $(CLI_DIR) && govulncheck ./...

py-lint:
	cd $(SDK_DIR) && ruff check .

py-type:
	cd $(SDK_DIR) && mypy aide_sdk

py-test:
	cd $(SDK_DIR) && .venv/bin/pytest

fe-lint:
	cd $(FRONTEND_DIR) && npm run typecheck
	cd $(FRONTEND_DIR) && npm run lint

verify: go-lint go-test py-lint py-type fe-lint
	@echo "verify passed"
