#!/bin/bash
# Builds the adsm CLI.
set -e

VERSION="${VERSION:-dev}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
PKG="github.com/afterdarksys/adsm/internal/cmd"

echo "Building adsm ${VERSION} (${COMMIT})..."

go mod tidy
go build -ldflags "-X ${PKG}.version=${VERSION} -X ${PKG}.commit=${COMMIT}" \
    -o adsm ./cmd/adsm

chmod +x adsm

echo "SUCCESS! Binary ready at ./adsm"
echo "Try: ./adsm --help"
