# Apache HTTPd rotatelogs Packaging Decision

## Context

Apache HTTPd provides the `rotatelogs` utility for piped log rotation.
Use this utility in containerized deployments to rotate log files without external log management infrastructure.

## Decision

Apache HTTPd releases for this project include both `bin/httpd` and `bin/rotatelogs` in the release artifacts.

## Rationale

Including `rotatelogs` provides users with a built-in solution for log rotation without requiring additional dependencies or external tooling.
For containerized deployments:

- Users can keep external dependencies to a minimum.
- Users can manage self-contained log rotation.
- Apache HTTPd includes the utility in its source.

## Implementation Evidence

The `apache-httpd/Dockerfile` enables a static rotatelogs build:

```text
--enable-static-rotatelogs \
```

The final artifact copies the entire `${TARGET_PREFIX}` directory, which includes `bin/rotatelogs` along with `bin/httpd`.

## Alternatives Considered

1. External rotatelogs from system packages adds deployment complexity and distribution-specific behavior.
2. logrotate requires external daemon and configuration management.
3. Container-native logging drivers (Docker json-file, fluentd, etc.) require different logging configuration.

## References

- [README utility inclusion](README.md#build-options-explicit) documents the rotatelogs build option.
- [Dockerfile](Dockerfile) contains the `--enable-static-rotatelogs` configure option.
