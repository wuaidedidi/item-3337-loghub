#!/bin/sh
set -e

echo "=== LogHub Application Log System ==="
echo "Starting initialization..."

# Generate certificates if they don't exist
if [ ! -f /app/certs/server.crt ] || [ ! -f /app/certs/server.key ]; then
    echo "Generating TLS certificates..."
    /app/certgen -out /app/certs -org "LogHub" -cn "LogHub WSS Server" -days 365 -hosts "localhost,127.0.0.1,backend"
    echo "Certificates generated successfully."
else
    echo "TLS certificates already exist, skipping generation."
fi

# Seed demo data only on first run (marker file check)
SEED_FLAG=""
if [ ! -f /app/data/logs/.seeded ]; then
    echo "First run detected, seeding demo data..."
    SEED_FLAG="-seed"
    touch /app/data/logs/.seeded
else
    echo "Demo data already seeded, skipping."
fi

echo "Starting LogHub server..."
exec /app/loghub -config /app/config.yaml $SEED_FLAG
