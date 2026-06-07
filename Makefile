.PHONY: setup build web install run test-go test-py clean copy-scrapers deploy

VERSION := $(shell cat VERSION)
PYTHON_VENV := scrapers/.venv
PYTHON_BIN := $(PYTHON_VENV)/bin/python
BIN_DIR := bin
FRONTEND_DIR := cli/internal/agent/frontend
INSTALL_DIR := $(HOME)/.local/bin
SCRAPERS_EMBED := cli/internal/scrapers/embedded
LDFLAGS := -X main.version=$(VERSION) -X aide/cli/internal/agent.Version=$(VERSION)

NEXUS_URL ?= https://nexus.sharedservices.local/repository/aide

setup: setup-go setup-web

setup-go:
	cd cli && go mod download

setup-web:
	cd $(FRONTEND_DIR) && npm install

copy-scrapers:
	rm -rf $(SCRAPERS_EMBED)
	mkdir -p $(SCRAPERS_EMBED)
	rsync -a --exclude='.venv' --exclude='.sessions' --exclude='__pycache__' scrapers/ $(SCRAPERS_EMBED)/
	cp registry.yaml $(SCRAPERS_EMBED)/registry.yaml

build: copy-scrapers
	cd $(FRONTEND_DIR) && npm run build
	cd cli && go build -ldflags "$(LDFLAGS)" -o ../$(BIN_DIR)/aide ./cmd/aide

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BIN_DIR)/aide $(INSTALL_DIR)/aide
	@echo "Installed aide to $(INSTALL_DIR)/aide"
	@echo "Run 'aide init' to setup ~/.aide/ directory structure"

run: build
	./$(BIN_DIR)/aide run

serve: build
	./$(BIN_DIR)/aide agent start

report: build
	./$(BIN_DIR)/aide report

sources: build
	./$(BIN_DIR)/aide sources

test-go:
	cd cli && go test ./...

test-py:
	cd scrapers && .venv/bin/python -m pytest

deploy: copy-scrapers
	cd $(FRONTEND_DIR) && npm run build
	cd cli && GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ../$(BIN_DIR)/aide-darwin-arm64 ./cmd/aide
	cd cli && GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../$(BIN_DIR)/aide-darwin-amd64 ./cmd/aide
	@NUSER=$(NEXUS_USER); NPASS=$(NEXUS_PASSWORD); \
	if [ -z "$$NUSER" ]; then printf "Nexus user: "; read NUSER; fi; \
	if [ -z "$$NPASS" ]; then printf "Nexus password: "; stty -echo; read NPASS; stty echo; echo; fi; \
	curl -u "$$NUSER:$$NPASS" --upload-file $(BIN_DIR)/aide-darwin-arm64 $(NEXUS_URL)/$(VERSION)/aide-darwin-arm64 && \
	curl -u "$$NUSER:$$NPASS" --upload-file $(BIN_DIR)/aide-darwin-amd64 $(NEXUS_URL)/$(VERSION)/aide-darwin-amd64 && \
	curl -u "$$NUSER:$$NPASS" --upload-file registry.yaml $(NEXUS_URL)/$(VERSION)/registry.yaml && \
	curl -u "$$NUSER:$$NPASS" --upload-file VERSION $(NEXUS_URL)/VERSION && \
	curl -u "$$NUSER:$$NPASS" --upload-file install.sh $(NEXUS_URL)/install.sh && \
	echo "Deployed aide $(VERSION) to Nexus"

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(SCRAPERS_EMBED)
	rm -f data/aide.db
