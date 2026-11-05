#!/bin/bash
# Build both desktop architectures and available installers for macOS, Windows and Linux.
set -euo pipefail

project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$project_root"

# MyGo includes native libraries and places each target in build/<os>-<arch>.
# MyGo creates DMGs on macOS and Windows installers when NSIS is available.
# Unsupported packaging steps are skipped with a warning; Linux packages need no extra tools.
exec go tool mygo build \
  -platform darwin/arm64,darwin/amd64,windows/arm64,windows/amd64,linux/arm64,linux/amd64 \
  "$@"
