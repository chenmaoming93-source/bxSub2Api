#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
exec env -u SERVER_HOST -u SERVER_PORT -u DATA_DIR go run ./cmd/server
