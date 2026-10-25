#!/usr/bin/env sh
for patch_file in /build/opentelemetry-c-wrapper/scripts/build/000*-opentelemetry-cpp-*.patch; do
  patch -p1 <"$patch_file"
done &&
  patch -p1 </tmp/opentelemetry-cpp-cstdint.patch
