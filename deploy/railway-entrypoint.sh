#!/bin/sh
set -eu

export PORT="${PORT:-8080}"
export STARTUP_TIMEOUT="${STARTUP_TIMEOUT:-180s}"

echo "Khởi động backend hợp nhất trên cổng public $PORT"
exec /app/allserver
