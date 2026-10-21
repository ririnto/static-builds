# Apache HTTPd packaging rules

Include both `bin/httpd` and `bin/rotatelogs` in release artifacts selected by [metadata.json](../metadata.json).
Preserve `--enable-static-rotatelogs` in [Dockerfile](Dockerfile) and its documented [build option](README.md#build-options-explicit).
