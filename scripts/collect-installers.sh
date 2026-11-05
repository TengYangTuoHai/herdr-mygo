#!/bin/bash
# Collect only the native installers produced by this runner's MyGo build.
set -euo pipefail

target_os=${1:?Usage: collect-installers.sh <os> <version>}
version=${2:?Usage: collect-installers.sh <os> <version>}

case "$target_os" in
  darwin)
    files=("darwin-arm64/herdr-$version-arm64.dmg" "darwin-amd64/herdr-$version-amd64.dmg")
    ;;
  windows)
    files=("windows-amd64/herdr-setup-$version-amd64.exe")
    ;;
  linux)
    # Match MyGo's Debian version: prereleases use ~ and replace later - with .
    deb_version=$version
    if [[ "$version" == *-* ]]; then
      prerelease=${version#*-}
      deb_version="${version%%-*}~${prerelease//-/.}"
    fi
    files=("linux-arm64/herdr_${deb_version}_arm64.deb" "linux-amd64/herdr_${deb_version}_amd64.deb")
    ;;
  *)
    echo "Unsupported installer platform: $target_os" >&2
    exit 1
    ;;
esac

mkdir -p artifacts
for file in "${files[@]}"; do
  if [[ ! -s "build/$file" ]]; then
    echo "Missing or empty installer: build/$file" >&2
    exit 1
  fi
  cp "build/$file" artifacts/
done
