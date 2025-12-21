#!/bin/bash
set -e

echo "Installing CI prerequisites..."

# Install golangci-lint
echo "Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Install goimports
echo "Installing goimports..."
go install golang.org/x/tools/cmd/goimports@latest

echo "CI prerequisites installed successfully."
