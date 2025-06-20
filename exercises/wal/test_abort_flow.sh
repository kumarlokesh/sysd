#!/bin/bash
set -e

# Clean up any existing state
rm -rf /tmp/wal
rm -rf data/wal

# Build the CLI
echo "Building CLI..."
go build -o wald ./cmd/wald

# Test transaction abort flow
echo -e "\n=== Testing transaction abort flow ==="

# Begin transaction
echo -e "\n1. Beginning transaction..."
./wald begin-tx

# Write a record in the transaction
echo -e "\n2. Writing record in transaction..."
./wald tx-write -key test -value value1

# Abort the transaction
echo -e "\n3. Aborting transaction..."
./wald abort

# Read all records
echo -e "\n4. Reading all records (should be empty)..."
./wald read

echo -e "\n=== Transaction abort flow test complete ==="
