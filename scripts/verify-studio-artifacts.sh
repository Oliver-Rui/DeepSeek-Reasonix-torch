#!/usr/bin/env bash
# usage: verify-studio-artifacts.sh DIRECTORY [PLATFORM...]
#
# The artifacts each platform a release claims must carry, per architecture.
# Naming no platform means every platform the line claims.
set -euo pipefail
dir=${1:?usage: verify-studio-artifacts.sh DIRECTORY [PLATFORM...]}
shift
platforms=("$@")
if [ "$#" -eq 0 ]; then
  platforms=(linux windows darwin)
fi
for arch in amd64 arm64; do
  for platform in "${platforms[@]}"; do
    case "$platform" in
      linux)
        names=("ReasonixStudio-linux-$arch.deb" "ReasonixStudio-linux-$arch.rpm" "ReasonixStudio-linux-$arch.AppImage")
        ;;
      windows)
        names=("ReasonixStudio-windows-$arch-installer.exe" "ReasonixStudio-windows-$arch.zip")
        ;;
      darwin)
        names=("ReasonixStudio-darwin-$arch.dmg" "ReasonixStudio-darwin-$arch.zip")
        ;;
      *)
        echo "unknown Studio platform: $platform" >&2
        exit 2
        ;;
    esac
    for name in "${names[@]}"; do
      test -s "$dir/$name" || { echo "missing or empty Studio artifact: $name" >&2; exit 1; }
    done
  done
done
