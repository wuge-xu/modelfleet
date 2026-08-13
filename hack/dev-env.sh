#!/usr/bin/env bash

GO126_ROOT="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64"

if [ ! -x "$GO126_ROOT/bin/go" ]; then
    echo "ModelFleet Go 1.26 toolchain not found:"
    echo "$GO126_ROOT"
    return 1 2>/dev/null || exit 1
fi

export GOROOT="$GO126_ROOT"
export PATH="$GOROOT/bin:$PATH"
export GOTOOLCHAIN=local
unset GOTOOLDIR

export GOCACHE="$HOME/.cache/go-build-1.26-modelfleet"
mkdir -p "$GOCACHE"

hash -r

echo "ModelFleet Go environment ready:"
go version
go tool compile -V=full