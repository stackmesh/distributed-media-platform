#!/bin/sh
set -eu
dir="${BLOB_DIR:-/var/lib/stackmesh/blobs}"
mkdir -p "$dir"
chown -R 65532:65532 "$dir"
exec su-exec 65532:65532 /usr/local/bin/gateway
