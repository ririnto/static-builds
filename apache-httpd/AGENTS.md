# Apache HTTPd rotatelogs Packaging Decision

## Context

Apache HTTPd provides the `rotatelogs` utility for piped log rotation. This utility is commonly used in containerized deployments where log files need to be rotated without external log management infrastructure.

## Decision

Apache HTTPd releases for this project include both `bin/httpd` and `bin/rotatelogs` in the release artifacts.

## Rationale

Including `rotatelogs` provides users with a built-in solution for log rotation without requiring additional dependencies or external tooling. This is particularly valuable for containerized deployments where:

- Minimal external dependencies are preferred
- Self-contained log rotation is simpler to manage
- The utility is already available as part of the Apache HTTPd source

## Implementation Evidence

The `apache-httpd/Dockerfile` explicitly enables static rotatelogs build:

```
--enable-static-rotatelogs \
```

The final artifact copies the entire `${TARGET_PREFIX}` directory, which includes `bin/rotatelogs` along with `bin/httpd`.

## Alternatives Considered

1. External rotatelogs from system packages - Adds deployment complexity and distribution-specific behavior
2. logrotate - Requires external daemon and configuration management
3. Container-native logging drivers (Docker json-file, fluentd, etc.) - Good alternative but requires different logging configuration

## References

- README.md "Logging Strategy" section where rotatelogs is mentioned
- `apache-httpd/Dockerfile` line containing `--enable-static-rotatelogs` configure option
