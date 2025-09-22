# Detect platform for sed compatibility
SED := $(shell if [ "$(shell uname)" = "Darwin" ]; then echo gsed; else echo sed; fi)

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.5.0
# renovate: datasource=npm depName=bootstrap
BOOTSTRAP_VERSION ?= 5.3.8
# renovate: datasource=github-releases depName=tristen/tablesort
TABLESORT_VERSION ?= 5.6.0

.PHONY: test cover clean update patch minor major tag

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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

# Find the latest tag (with prefix filter if defined, default to 0.0.0 if none found)
# Lazy evaluation ensures fresh values on every run
VERSION_PREFIX ?= v
LATEST_TAG = $(shell git tag --list "$(VERSION_PREFIX)*" --sort=-v:refname | head -n 1)
VERSION = $(shell [ -n "$(LATEST_TAG)" ] && echo $(LATEST_TAG) | sed "s/^$(VERSION_PREFIX)//" || echo "0.0.0")

patch: ## Create a new patch release (x.y.Z+1)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

minor: ## Create a new minor release (x.Y+1.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.0", $$1, $$2+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

major: ## Create a new major release (X+1.0.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.0.0", $$1+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

tag: ## Show latest tag
	@echo "Latest version: $(LATEST_TAG)"

push: ## Push tags to remote
	git push --tags

##@ Dependencies

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

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

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

