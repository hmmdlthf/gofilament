#!/usr/bin/env sh
# Build the website into site/public: docs for every release tag, then Hugo.
#   ./site/build.sh            production build
#   ./site/build.sh server     live preview at http://localhost:1313
set -eu
cd "$(dirname "$0")"
go run ./tools/gendocs
if [ "${1:-}" = "server" ]; then
  exec hugo server --disableFastRender
fi
hugo --minify --gc --cleanDestinationDir
