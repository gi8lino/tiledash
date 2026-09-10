# Detect platform for sed compatibility
SED := $(shell if [ "$(shell uname)" = "Darwin" ]; then echo gsed; else echo sed; fi)

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin

## Tool Versions
# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.5.0

## Tool Binaries
DEV_TOOL_NAMES := dev-port open-browser dev-tag make-help go-install-tool
DEV_TOOL_TARGETS := $(addprefix $(LOCALBIN)/,$(DEV_TOOL_NAMES))
DEV_TOOL_VERSIONED := $(addsuffix -$(DEV_TOOLS_VERSION),$(DEV_TOOL_TARGETS))

DEV_PORT := $(LOCALBIN)/dev-port
OPEN_BROWSER := $(LOCALBIN)/open-browser
DEV_TAG := $(LOCALBIN)/dev-tag
MAKE_HELP := $(LOCALBIN)/make-help
GO_INSTALL_TOOL := $(LOCALBIN)/go-install-tool

# Run a local tool while displaying only its executable name.
define run-tool
@printf '%s\n' '$(notdir $(1)) $(2)'
@$(1) $(2)
endef

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2
# renovate: datasource=npm depName=bootstrap
BOOTSTRAP_VERSION ?= 5.3.8
# renovate: datasource=github-releases depName=tristen/tablesort
TABLESORT_VERSION ?= 5.7.1

.PHONY: test cover clean update patch minor major tag

##@ General

.PHONY: help
help: $(MAKE_HELP) ## Display this help.
	@$(MAKE_HELP) $(MAKEFILE_LIST)

##@ Development

.PHONY: run
run: ## Run the server for local testing
	go run main.go --template-dir examples/templates --config examples/config.yaml

.PHONY: download
download: ## Download go packages
	go mod download

.PHONY: update-packages
update-packages: ## Update all Go packages to their latest versions
	go get -u ./...
	go mod tidy

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit tests.
	go test -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...

.PHONY: cover
cover: ## Display test coverage
	go test -coverprofile=coverage.out -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean up generated files
	rm -f coverage.out coverage.html

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

##@ Mock Server

.PHONY: run-mock
run-mock: ## Run the mock server for local testing
	go run ./tests/main.go --config=./tests/config.yaml

##@ Tagging

VERSION_PREFIX ?= v

.PHONY: current
current: $(DEV_TAG) ## Show the current semantic version tag.
	$(call run-tool,$(DEV_TAG),--prefix "$(VERSION_PREFIX)" current)

.PHONY: patch
patch: $(DEV_TAG) ## Create a new patch release (x.y.Z+1).
	$(call run-tool,$(DEV_TAG),--prefix "$(VERSION_PREFIX)" patch)

.PHONY: minor
minor: $(DEV_TAG) ## Create a new minor release (x.Y+1.0).
	$(call run-tool,$(DEV_TAG),--prefix "$(VERSION_PREFIX)" minor)

.PHONY: major
major: $(DEV_TAG) ## Create a new major release (X+1.0.0).
	$(call run-tool,$(DEV_TAG),--prefix "$(VERSION_PREFIX)" major)

.PHONY: tag
tag: current

.PHONY: push
push: ## Push tags to the configured remote.
	git push --tags

##@ Dependencies


.PHONY: update-bootstrap update-tablesort update-assets

update-bootstrap: ## Download Bootstrap CSS and JS locally
	@set -euo pipefail; \
	curl -sSL -o web/static/css/bootstrap.min.css \
	  "https://cdn.jsdelivr.net/npm/bootstrap@$(BOOTSTRAP_VERSION)/dist/css/bootstrap.min.css"
	@set -euo pipefail; \
	curl -sSL -o web/static/css/bootstrap.min.css.map \
	  "https://cdn.jsdelivr.net/npm/bootstrap@$(BOOTSTRAP_VERSION)/dist/css/bootstrap.min.css.map"
	@set -euo pipefail; \
	curl -sSL -o web/static/js/bootstrap.bundle.min.js \
	  "https://cdn.jsdelivr.net/npm/bootstrap@$(BOOTSTRAP_VERSION)/dist/js/bootstrap.bundle.min.js"

update-tablesort: ## Download tablesort.min.js locally
	@set -euo pipefail; \
	curl -sSL -o web/static/js/tablesort.min.js \
	  "https://cdn.jsdelivr.net/npm/tablesort@$(TABLESORT_VERSION)/dist/tablesort.min.js"

update-assets: update-bootstrap update-tablesort ## Download all frontend assets

.PHONY: verify-assets
verify-assets: ## Check that all frontend assets exist
	@test -f web/static/css/bootstrap.min.css
	@test -f web/static/css/bootstrap.min.css.map
	@test -f web/static/js/bootstrap.bundle.min.js
	@test -f web/static/js/tablesort.min.js

##@ Development tools

.PHONY: dev-tools
dev-tools: $(DEV_TOOL_TARGETS) ## Download the pinned development tools.

$(DEV_TOOL_TARGETS): $(LOCALBIN)/%: $(LOCALBIN)/%-$(DEV_TOOLS_VERSION)
	@ln -sf "$(notdir $<)" "$@"

$(DEV_TOOL_VERSIONED): $(LOCALBIN)/%-$(DEV_TOOLS_VERSION): | $(LOCALBIN)
	$(call download-dev-tool,$*,$@)

# download-dev-tool downloads a versioned tool from gi8lino/dev-tools.
# $1 - release asset name
# $2 - versioned destination path
define download-dev-tool
	@set -eu; \
	tmp="$(2).tmp"; \
	trap 'rm -f "$$tmp"' EXIT INT TERM; \
	echo "Downloading gi8lino/dev-tools $(DEV_TOOLS_VERSION) $(1)"; \
	curl --fail --silent --show-error --location \
		"https://github.com/gi8lino/dev-tools/releases/download/$(DEV_TOOLS_VERSION)/$(1)" \
		-o "$$tmp"; \
	chmod +x "$$tmp"; \
	mv "$$tmp" "$(2)"; \
	trap - EXIT INT TERM
endef

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) ## Download golangci-lint locally if necessary.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"
