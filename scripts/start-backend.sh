#!/bin/bash
# Start KPL BP Simulator backend server

cd "$(dirname "$0")/.."
export DB_PASSWORD="83827644"

echo "Starting backend server on port 8080..."
./backend/server
