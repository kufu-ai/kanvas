GOLANGCI_LINT_VERSION=1.55.1
TAG  ?= $(shell git describe --tags --abbrev=0 HEAD)
DATE_FMT = +"%Y-%m-%dT%H:%M:%S%z"
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)")
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

GO_BUILD_VERSION_LDFLAGS=\
  -X go.szostok.io/version.version=$(TAG) \
  -X go.szostok.io/version.buildDate=$(BUILD_DATE) \
  -X go.szostok.io/version.commit=$(shell git rev-parse --short HEAD) \
  -X go.szostok.io/version.commitDate=$(shell git log -1 --date=format:"%Y-%m-%dT%H:%M:%S%z" --format=%cd) \
  -X go.szostok.io/version.dirtyBuild=false

build:
	go build -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" -o dist/kanvas ./cmd/kanvas
.PHONY: build

lint:
	docker run --rm -v $(shell pwd):/app -v ~/.cache/golangci-lint/v$(GOLANGCI_LINT_VERSION):/root/.cache -w /app golangci/golangci-lint:v$(GOLANGCI_LINT_VERSION) golangci-lint run -v
.PHONY: lint

test:
	go test -v ./...
.PHONY: test

# find or download yq
# download yq if necessary
# Use always go-version to get consistent line wraps etc.
docgen:
ifeq (, $(wildcard $(GOBIN)/docgen))
	echo "Downloading yq"
	@{ \
	set -e ;\
	DG_TMP_DIR=$$(mktemp -d) ;\
	cd $$DG_TMP_DIR ;\
	go mod init tmp ;\
	go install github.com/projectdiscovery/yamldoc-go/cmd/docgen@latest ;\
	rm -rf $$DG_TMP_DIR ;\
	}
endif
DOCGEN=$(GOBIN)/docgen
