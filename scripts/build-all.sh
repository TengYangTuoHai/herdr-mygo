#!/bin/bash
# Build both desktop architectures for macOS, Windows and Linux.
set -euo pipefail

project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$project_root"

# MyGo includes native libraries and places each target in build/<os>-<arch>.
# Installers are optional so cross-compilation needs neither NSIS nor DMG tools.
exec go tool mygo build \
  -platform darwin/arm64,darwin/amd64,windows/arm64,windows/amd64,linux/arm64,linux/amd64 \
  -skip-dmg \
  -skip-installer \
  "$@"
