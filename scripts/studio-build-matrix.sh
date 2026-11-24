#!/usr/bin/env bash
# usage: studio-build-matrix.sh all|windows
#
# The build legs a release claims, as the matrix the build job expands. The six
# legs live here rather than in the workflow because a narrower release has to
# name the same runners, architectures and artifact names, and a table written
# twice is one that drifts. ebarch is electron-builder's spelling of the
# architecture, stated beside the release one rather than mapped in a step; a
# missing one means the runner's own. artifact is the name publish downloads
# this leg's packages by; goarch and unpacked belong to the Windows legs.
set -euo pipefail
platforms="${1:-all}"
case "$platforms" in
  all | windows) ;;
  *) echo "usage: studio-build-matrix.sh all|windows" >&2; exit 2 ;;
esac
legs='[
  {"platform":"windows/amd64","runner":"windows-latest","goarch":"amd64","unpacked":"win-unpacked","artifact":"studio-build-windows-amd64"},
  {"platform":"windows/arm64","runner":"windows-11-arm","ebarch":"arm64","goarch":"arm64","unpacked":"win-arm64-unpacked","artifact":"studio-build-windows-arm64"},
  {"platform":"darwin/amd64","runner":"macos-latest","ebarch":"x64","artifact":"studio-build-darwin-amd64"},
  {"platform":"darwin/arm64","runner":"macos-latest","ebarch":"arm64","artifact":"studio-build-darwin-arm64"},
  {"platform":"linux/amd64","runner":"ubuntu-22.04","artifact":"studio-build-linux-amd64"},
  {"platform":"linux/arm64","runner":"ubuntu-22.04-arm","artifact":"studio-build-linux-arm64"}
]'
if [ "$platforms" = windows ]; then
  jq -c '{"include":[.[] | select(.platform | startswith("windows/"))]}' <<<"$legs"
else
  jq -c '{"include":.}' <<<"$legs"
fi
