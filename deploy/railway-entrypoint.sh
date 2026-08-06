#!/bin/sh
set -eu

service_name="${SERVICE_NAME:-${RAILWAY_SERVICE_NAME:-}}"
service_name=$(printf '%s' "$service_name" | tr '[:upper:]' '[:lower:]')

case "$service_name" in
  gate|gateserver|*gate*)
    binary="gateserver"
    default_port="8004"
    ;;
  http|httpserver|api|*http*|*api*)
    binary="httpserver"
    default_port="8088"
    ;;
  login|loginserver|*login*)
    binary="loginserver"
    default_port="8003"
    ;;
  chat|chatserver|*chat*)
    binary="chatserver"
    default_port="8002"
    ;;
  slg|slgserver|game|*slg*|*game*)
    binary="slgserver"
    default_port="8001"
    ;;
  *)
    echo "SERVICE_NAME không hợp lệ: '$service_name'."
    echo "Giá trị hợp lệ: gate, http, login, chat hoặc slg."
    exit 1
    ;;
esac

export PORT="${PORT:-$default_port}"
echo "Khởi động $binary trên cổng $PORT"
exec "/app/$binary"
