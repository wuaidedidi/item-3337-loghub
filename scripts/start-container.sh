#!/bin/sh
set -e

echo "Starting LogHub backend on ports 8000 and 8443..."
/app/backend/entrypoint.sh &
backend_pid=$!

echo "Starting Nginx frontend on port 3000..."
nginx -g "daemon off;" &
nginx_pid=$!

trap 'kill $backend_pid $nginx_pid 2>/dev/null || true; wait' INT TERM
wait -n $backend_pid $nginx_pid
