BINARY     := aide
CLI_DIR    := cli
BIN_DIR    := bin
SANDBOX    := $(HOME)/.aide-sandbox
SDK_PATH   := $(CURDIR)/sdk/python
SDK_DIR    := sdk/python
FRONTEND_DIR := cli/internal/agent/frontend
PLUGINS_SRC := $(CURDIR)/../aide-plugins/plugins

.PHONY: build dev dev-plugins clean verify go-lint go-test go-vuln py-lint py-type py-test fe-lint fmt

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

dev-plugins: dev
	@echo "  Syncing plugin source files into sandbox..."
	@if [ ! -d "$(PLUGINS_SRC)" ]; then \
		echo "  [!] PLUGINS_SRC not found: $(PLUGINS_SRC)"; exit 1; \
	fi
	@for plugin_src in $(PLUGINS_SRC)/*/; do \
		name=$$(basename "$$plugin_src"); \
		dest=$(SANDBOX)/plugins/$$name; \
		if [ ! -f "$$dest/plugin.yaml" ]; then \
			echo "  [+] installing $$name (first time or broken)..."; \
			AIDE_HOME=$(SANDBOX) AIDE_SDK_PATH=$(SDK_PATH) $(SANDBOX)/bin/$(BINARY) plugin install --local "$$plugin_src" --yes || \
				echo "  [!] install failed for $$name — check output above"; \
		else \
			echo "  [+] syncing $$name source files..."; \
			rsync -a --exclude='.venv/' --exclude='__pycache__/' --exclude='*.pyc' "$$plugin_src" "$$dest/"; \
		fi; \
	done
	@echo "  Done."
	@echo "  SDK changes take effect automatically (AIDE_SDK_PATH injected at runtime)."
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
