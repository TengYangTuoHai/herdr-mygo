#!/bin/bash
# Build macOS and Linux for arm64 and amd64, and Windows for amd64, with available installers.
set -euo pipefail

project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$project_root"

# MyGo includes native libraries and places each target in build/<os>-<arch>.
# MyGo creates DMGs on macOS and Windows installers when NSIS is available.
# Unsupported packaging steps are skipped with a warning; Linux packages need no extra tools.
exec go tool mygo build \
  -platform darwin/arm64,darwin/amd64,windows/amd64,linux/arm64,linux/amd64 \
  "$@"
