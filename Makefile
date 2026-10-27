MODULE   := github.com/bpauli/gccli
BINARY   := gccli
BIN_DIR  := ./bin
CMD_DIR  := ./cmd/gccli
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

require-tools = for t in $(1); do test -x "$(TOOLS_DIR)/$$t" || { echo "$(TOOLS_DIR)/$$t not found; run 'make tools'"; exit 1; }; done

.PHONY: fmt
fmt:
	@$(call require-tools,goimports gofumpt)
	$(TOOLS_DIR)/goimports -w -local $(LOCAL_PREFIX) .
	$(TOOLS_DIR)/gofumpt -w .

.PHONY: fmt-check
fmt-check:
	@$(call require-tools,goimports gofumpt)
	@out=$$($(TOOLS_DIR)/goimports -l -local $(LOCAL_PREFIX) .) || \
		{ echo "goimports failed"; exit 1; }; \
		test -z "$$out" || { echo "goimports check failed; run 'make fmt'"; exit 1; }
	@out=$$($(TOOLS_DIR)/gofumpt -l .) || \
		{ echo "gofumpt failed"; exit 1; }; \
		test -z "$$out" || { echo "gofumpt check failed; run 'make fmt'"; exit 1; }

.PHONY: lint
lint:
	@$(call require-tools,golangci-lint)
	$(TOOLS_DIR)/golangci-lint run ./...

# The internal/cmd tests set package variables, the environment and the
# working directory, so they cannot call t.Parallel. make test runs them in
# four processes at once instead, split by measured time.
CMD_TEST_SHARDS := golden-a-g golden-h-z write-account rest
CMD_TEST_RUN_golden-a-g    := -run '^TestGolden$$/^[a-g]'
CMD_TEST_RUN_golden-h-z    := -run '^TestGolden$$/^[^a-g]'
CMD_TEST_RUN_write-account := -run '^TestWriteCommand|^TestEveryLeafNeedsAccount$$'
CMD_TEST_RUN_rest          := -skip '^TestGolden$$|^TestWriteCommand|^TestEveryLeafNeedsAccount$$'

.PHONY: test
test:
	go test -race -run '^$$' ./internal/cmd
	@$(MAKE) --no-print-directory -j test-other-packages $(addprefix test-cmd-,$(CMD_TEST_SHARDS))

.PHONY: test-other-packages
test-other-packages:
	go test -race -timeout 20m $$(go list ./... | grep -v '/internal/cmd$$')

.PHONY: $(addprefix test-cmd-,$(CMD_TEST_SHARDS))
$(addprefix test-cmd-,$(CMD_TEST_SHARDS)): test-cmd-%:
	go test -race -timeout 20m $(CMD_TEST_RUN_$*) ./internal/cmd

.PHONY: test-e2e
test-e2e:
	go test -tags=e2e -v -count=1 ./internal/e2e/...

.PHONY: conformance
conformance:
	GCCLI_BIN=$(abspath $(GCCLI_BIN)) go test -tags=conformance -count=1 ./conformance/

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
