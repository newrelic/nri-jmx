WORKDIR     := $(shell pwd)
TARGET      := target
TARGET_DIR   = $(WORKDIR)/$(TARGET)
NATIVEOS    := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH  := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION := jmx
BINARY_NAME  = nri-$(INTEGRATION)
GO_PKGS     := $(shell go list ./... | grep -v "/vendor/")
GO_FILES    := ./src/
GO_VERSION 	  ?= $(shell grep '^go ' go.mod | awk '{print $$2}')
BUILDER_IMAGE ?= "ghcr.io/newrelic/coreint-automation:latest-go$(GO_VERSION)-ubuntu16.04"
GOFLAGS			= -mod=readonly
GOLANGCI_LINT	= github.com/golangci/golangci-lint/cmd/golangci-lint

all: build

build: clean validate test compile

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)

validate:
	@printf "=== $(INTEGRATION) === [ validate ]: running golangci-lint & semgrep... "
	@go run  $(GOFLAGS) $(GOLANGCI_LINT) run --verbose
	@[ -f .semgrep.yml ] && semgrep_config=".semgrep.yml" || semgrep_config="p/golang" ; \
	docker run --rm -v "${PWD}:/src:ro" --workdir /src returntocorp/semgrep -c "$$semgrep_config"

compile:
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

test:
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@go test -race ./... -count=1

integration-test:
	@echo "=== $(INTEGRATION) === [ test ]: running integration tests..."
	@if [ "$(NRJMX_VERSION)" = "" ]; then \
	    echo "Error: missing required env-var: NRJMX_VERSION\n" ;\
        exit 1 ;\
	fi
	@docker compose -f test/integration/docker-compose.yml up -d --build
	@go test -v -tags=integration ./test/integration/. -count=1 ; (ret=$$?; docker compose -f test/integration/docker-compose.yml down && exit $$ret)

# rt-update-changelog runs the release-toolkit run.sh script by piping it into bash to update the CHANGELOG.md.
# It also passes down to the script all the flags added to the make target. To check all the accepted flags,
# see: https://github.com/newrelic/release-toolkit/blob/main/contrib/ohi-release-notes/run.sh
#  e.g. `make rt-update-changelog -- -v`
rt-update-changelog:
	curl "https://raw.githubusercontent.com/newrelic/release-toolkit/v1/contrib/ohi-release-notes/run.sh" | bash -s -- $(filter-out $@,$(MAKECMDGOALS))

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean validate compile test integration-test check-version
