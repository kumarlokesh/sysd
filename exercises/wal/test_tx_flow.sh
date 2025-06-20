#!/bin/bash
set -e

# Clean up any existing state
rm -rf /tmp/wal
rm -rf data/wal

# Build the CLI
echo "Building CLI..."
go build -o wald ./cmd/wald

# Test transaction flow
echo -e "\n=== Testing transaction flow ==="

# Begin transaction
echo -e "\n1. Beginning transaction..."
./wald begin-tx

# Write a record in the transaction
echo -e "\n2. Writing record in transaction..."
./wald tx-write -key test -value value1

# Commit the transaction
echo -e "\n3. Committing transaction..."
./wald commit

# Read all records
echo -e "\n4. Reading all records..."
./wald read

echo -e "\n=== Transaction flow test complete ==="
