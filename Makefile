TARGETS := nginx haproxy apache-httpd coredns dnsmasq vector monit
BUILD_SCRIPT := $(CURDIR)/build.sh
BUILD_TARGET := $(word 2,$(MAKECMDGOALS))

.PHONY: help list-targets check-target download build lint validate-tag

help:
	@printf '%s\n' "Usage:"
	@printf '%s\n' "  make build <target>"
	@printf '%s\n' "  make download <target>"
	@printf '%s\n' "  Output: <target>/"
	@printf '%s\n' ""
	@printf '%s\n' "Targets: $(TARGETS)"

list-targets:
	@printf '%s\n' "$(TARGETS)"

check-target:
	@if [ -z "$(BUILD_TARGET)" ]; then \
		printf '%s\n' "Error: target is required"; \
		printf '%s\n' "Example: make build nginx"; \
		exit 1; \
	fi; \
	case " $(TARGETS) " in \
	*" $(BUILD_TARGET) "*) ;; \
	*) \
		printf '%s\n' "Error: invalid target '$(BUILD_TARGET)'"; \
		printf '%s\n' "Allowed: $(TARGETS)"; \
		exit 1; \
	;; \
	esac

build: check-target
	@$(MAKE) --no-print-directory download "$(BUILD_TARGET)"
	"$(BUILD_SCRIPT)" "$(BUILD_TARGET)"

download: check-target
	@printf '%s\n' "Downloading source files for $(BUILD_TARGET)..."; \
	"$(CURDIR)/download.sh" "$(BUILD_TARGET)"

# Developer convenience targets

lint:
	@printf '%s\n' "Running shellcheck on project shell scripts..."
	@for file in .github/scripts/*.sh */download.sh; do \
		cat "$$file" | docker run --rm -i koalaman/shellcheck:stable --severity=error /dev/stdin || exit 1; \
	done
	@printf '%s\n' "✓ Shellcheck passed"

validate-tag:
	@if [ -z "$(word 2,$(MAKECMDGOALS))" ] || [ -z "$(word 3,$(MAKECMDGOALS))" ]; then \
		printf '%s\n' "Error: validate-tag requires two arguments"; \
		printf '%s\n' "Usage: make validate-tag <target> <tag>"; \
		printf '%s\n' "Example: make validate-tag nginx nginx-1.28.2.0"; \
		exit 1; \
	fi
	@"$(CURDIR)/.github/scripts/release-guard.sh" \
		"$(word 2,$(MAKECMDGOALS))" "$(word 3,$(MAKECMDGOALS))"

# Prevent make from treating validate-tag arguments as targets
%:
	@:
