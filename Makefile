MODULE   := github.com/bpauli/gccli
BINARY   := gc
BIN_DIR  := ./bin
CMD_DIR  := ./cmd/gc
TOOLS_DIR := .tools

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS  := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

LOCAL_PREFIX := github.com/bpauli/gccli

# ── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

.PHONY: run
run: build
	$(BIN_DIR)/$(BINARY) $(filter-out $@,$(MAKECMDGOALS))

# ── Quality ──────────────────────────────────────────────────────────────────

.PHONY: fmt
fmt:
	$(TOOLS_DIR)/goimports -w -local $(LOCAL_PREFIX) .
	$(TOOLS_DIR)/gofumpt -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$($(TOOLS_DIR)/goimports -l -local $(LOCAL_PREFIX) .)" || \
		{ echo "goimports check failed; run 'make fmt'"; exit 1; }
	@test -z "$$($(TOOLS_DIR)/gofumpt -l .)" || \
		{ echo "gofumpt check failed; run 'make fmt'"; exit 1; }

.PHONY: lint
lint:
	$(TOOLS_DIR)/golangci-lint run ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: test-e2e
test-e2e:
	go test -tags=e2e -v -count=1 ./internal/e2e/...

.PHONY: ci
ci: fmt-check lint test

# ── Tools ────────────────────────────────────────────────────────────────────

GOLANGCI_LINT_VERSION := v2.1.6
GOFUMPT_VERSION       := v0.7.0
GOIMPORTS_VERSION     := v0.31.0

.PHONY: tools
tools: $(TOOLS_DIR)/golangci-lint $(TOOLS_DIR)/gofumpt $(TOOLS_DIR)/goimports

$(TOOLS_DIR)/golangci-lint:
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(abspath $(TOOLS_DIR)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(TOOLS_DIR)/gofumpt:
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(abspath $(TOOLS_DIR)) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

$(TOOLS_DIR)/goimports:
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(abspath $(TOOLS_DIR)) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

# ── Helpers ──────────────────────────────────────────────────────────────────

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(TOOLS_DIR)

# catch-all for run target arguments
%:
	@:
