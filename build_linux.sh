#!/usr/bin/env bash
set -euo pipefail

# Build the custom ENet sources with the host Linux compiler.  Windows keeps
# using enet/lib/libenet.a through cgo_windows.go and is not modified here.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENET_SRC_DIR="$SCRIPT_DIR/enet_linux/enet"
ENET_INCLUDE_DIR="$ENET_SRC_DIR/include"
ENET_LIB_DIR="$SCRIPT_DIR/enet/lib"
ENET_ARCHIVE="$ENET_LIB_DIR/libenet_linux.a"
BUILD_DIR="${TMPDIR:-/tmp}/gtps-enet-build-$$"

cleanup() { rm -rf "$BUILD_DIR"; }
trap cleanup EXIT

command -v gcc >/dev/null || { echo "ERROR: gcc is required" >&2; exit 1; }
command -v ar >/dev/null || { echo "ERROR: binutils/ar is required" >&2; exit 1; }
command -v go >/dev/null || { echo "ERROR: Go is required" >&2; exit 1; }

if [ ! -f "$ENET_SRC_DIR/include/enet/enet.h" ]; then
    echo "ERROR: custom ENet source not found: $ENET_SRC_DIR" >&2
    exit 1
fi

mkdir -p "$BUILD_DIR" "$ENET_LIB_DIR"

# Keep this list explicit so platform-specific unix.c is used and no Windows
# objects or the Windows-built libenet.a can accidentally enter the Linux link.
ENET_SOURCES=(
    address.c callbacks.c compress.c host.c list.c packet.c peer.c protocol.c unix.c
)
OBJECTS=()
for source in "${ENET_SOURCES[@]}"; do
    object="$BUILD_DIR/${source%.c}.o"
    gcc -std=c11 -O2 -fPIC -I"$ENET_INCLUDE_DIR" -c "$ENET_SRC_DIR/$source" -o "$object"
    OBJECTS+=("$object")
done

ar rcs "$ENET_ARCHIVE" "${OBJECTS[@]}"
echo "Created Linux ENet archive: $ENET_ARCHIVE"

export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
# This workspace is intentionally usable outside a Git checkout.  Disable Go's
# optional VCS stamp so `go build` does not fail after querying git status.
go build -buildvcs=false -v -o "$SCRIPT_DIR/gtps-vallen" "$SCRIPT_DIR"

echo "BUILD SUCCESS: $SCRIPT_DIR/gtps-vallen"
