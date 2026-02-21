TARGETS := nginx haproxy apache-httpd coredns dnsmasq vector
BUILD_SCRIPT := $(CURDIR)/build.sh
BUILD_TARGET := $(word 2,$(MAKECMDGOALS))

.PHONY: help list-targets check-target download build

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


%:
	@:
