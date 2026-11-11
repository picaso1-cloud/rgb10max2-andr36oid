#!/bin/sh
# build.sh — cross-compile the flasher for Linux, macOS and Windows.
set -e
cd "$(dirname "$0")"
mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    os=${target%/*}
    arch=${target#*/}
    out="dist/rgb10max2-flasher-$os-$arch"
    [ "$os" = "windows" ] && out="$out.exe"
    echo "building $out"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$out" .
done
ls -lh dist
