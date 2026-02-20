TARGETS := nginx haproxy apache-httpd coredns dnsmasq vector
BUILD_SCRIPT := $(CURDIR)/build.sh

.PHONY: help list-targets check-target build build-plain build-no-cache build-plain-no-cache build-all build-all-no-cache $(TARGETS)

help:
	@printf '%s\n' "Usage:"
	@printf '%s\n' "  make build TARGET=<target>"
	@printf '%s\n' "  make build-plain TARGET=<target>"
	@printf '%s\n' "  make build-no-cache TARGET=<target>"
	@printf '%s\n' "  make build-plain-no-cache TARGET=<target>"
	@printf '%s\n' "  make build-all"
	@printf '%s\n' "  make build-all-no-cache"
	@printf '%s\n' "  make <target>"
	@printf '%s\n' "  Output: <target>/output/"
	@printf '%s\n' ""
	@printf '%s\n' "Targets: $(TARGETS)"

list-targets:
	@printf '%s\n' "$(TARGETS)"

check-target:
	@if [ -z "$(TARGET)" ]; then \
		printf '%s\n' "Error: TARGET is required"; \
		printf '%s\n' "Example: make build TARGET=nginx"; \
		exit 1; \
	fi; \
	case " $(TARGETS) " in \
	*" $(TARGET) "*) ;; \
	*) \
		printf '%s\n' "Error: Invalid TARGET '$(TARGET)'"; \
		printf '%s\n' "Allowed: $(TARGETS)"; \
		exit 1; \
	;; \
	esac

build: check-target
	"$(BUILD_SCRIPT)" "$(TARGET)"

build-plain: check-target
	"$(BUILD_SCRIPT)" --plain "$(TARGET)"

build-no-cache: check-target
	"$(BUILD_SCRIPT)" --no-cache "$(TARGET)"

build-plain-no-cache: check-target
	"$(BUILD_SCRIPT)" --plain --no-cache "$(TARGET)"

build-all:
	@set -e; \
	for target in $(TARGETS); do \
		printf '%s\n' "=== Building $$target ==="; \
		"$(BUILD_SCRIPT)" "$$target"; \
	done

build-all-no-cache:
	@set -e; \
	for target in $(TARGETS); do \
		printf '%s\n' "=== Building $$target (no cache) ==="; \
		"$(BUILD_SCRIPT)" --no-cache "$$target"; \
	done

nginx:
	"$(BUILD_SCRIPT)" nginx

haproxy:
	"$(BUILD_SCRIPT)" haproxy

apache-httpd:
	"$(BUILD_SCRIPT)" apache-httpd

coredns:
	"$(BUILD_SCRIPT)" coredns

dnsmasq:
	"$(BUILD_SCRIPT)" dnsmasq

vector:
	"$(BUILD_SCRIPT)" vector
