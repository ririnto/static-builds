#!/bin/sh
set -eu
cmake -S "/build/quickjs-${QUICKJS_VERSION}" -B /build/quickjs-build \
  -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release
cmake --build /build/quickjs-build "-j$(nproc)"
cmake --install /build/quickjs-build
