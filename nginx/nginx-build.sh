#!/bin/sh
set -eu
make "-j$(nproc)"
make install
