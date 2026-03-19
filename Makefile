DOWNLOAD_SCRIPT := $(CURDIR)/scripts/download.sh
METADATA_SCRIPT := $(CURDIR)/scripts/metadata.sh

ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(ARGS):;@:)

BUILDKIT_PLATFORM ?= linux/amd64
BUILDKIT_CACHE_BACKEND ?= local
BUILDKIT_NETWORK ?= default

.PHONY: help build download list-targets lint shfmt-check shfmt-write

help:
	@echo "Usage:"
	@echo "  make build <target>       Download sources and build"
	@echo "  make download <target>    Download sources only"
	@echo "  make list-targets         List available targets"
	@echo "  make lint                 Run shellcheck on all scripts"
	@echo "  make shfmt-check          Check shell formatting"
	@echo "  make shfmt-write          Fix shell formatting"

list-targets:
	@python3 -c "import json;[print(k) for k in json.load(open('metadata.json'))]"

build: download
	$(eval TARGET := $(word 1,$(ARGS)))
	$(eval BUILD_OUTPUT_DEST ?= $(CURDIR)/.out/$(TARGET))
	$(eval CACHE_DIR := $(CURDIR)/.cache/$(TARGET))
	@mkdir -p "$(BUILD_OUTPUT_DEST)" "$(CACHE_DIR)"
	docker buildx build \
		"--platform=$(BUILDKIT_PLATFORM)" \
		"--network=$(BUILDKIT_NETWORK)" \
		"--output=type=local,dest=$(BUILD_OUTPUT_DEST)" \
		"--cache-from=type=local,src=$(CACHE_DIR)" \
		"--cache-to=type=local,dest=$(CACHE_DIR),mode=max" \
		$$(sh "$(METADATA_SCRIPT)" get-env "$(TARGET)" | while IFS='=' read -r k v; do \
			case "$$k" in \
			(*[!A-Z0-9_]*) \
				continue \
				;; \
			esac; \
			if [ -n "$$v" ]; then \
				printf ' --build-arg=%s=%s' "$$k" "$$v"; \
			fi; \
		done) \
		"--file=$(CURDIR)/$(TARGET)/Dockerfile" \
		"$(CURDIR)"

download:
	@if [ -z "$(word 1,$(ARGS))" ]; then \
		echo "Error: target required" >&2; \
		exit 1; \
	fi
	sh "$(DOWNLOAD_SCRIPT)" "$(word 1,$(ARGS))"

lint:
	@find scripts .github/scripts .gitlab/scripts -name '*.sh' 2>/dev/null | while read -r file; do \
		echo "shellcheck $$file"; \
		if ! shellcheck "$$file"; then \
			exit 1; \
		fi; \
	done

shfmt-check:
	@find scripts .github/scripts .gitlab/scripts -name '*.sh' 2>/dev/null \
		| xargs shfmt -d -i 2 -ci

shfmt-write:
	@find scripts .github/scripts .gitlab/scripts -name '*.sh' 2>/dev/null \
		| xargs shfmt -w -i 2 -ci
