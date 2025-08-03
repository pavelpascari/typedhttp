#!/bin/bash

echo "🚀 Starting TypedHTTP Quickstart Demo"
echo "======================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | grep -o '[0-9]\+\.[0-9]\+')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
    echo "✅ Go $GO_VERSION detected (>= 1.21 required)"
else
    echo "❌ Go $GO_VERSION detected, but 1.21+ is required"
    echo "   Please upgrade Go: https://golang.org/doc/install"
    exit 1
fi

echo ""
echo "📦 Installing dependencies..."
go mod tidy

echo ""
echo "🔥 Starting server on http://localhost:8080"
echo ""
echo "Try these commands in another terminal:"
echo "  curl http://localhost:8080/users/world"
echo "  curl http://localhost:8080/users/alice"
echo "  curl http://localhost:8080/users/developer"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the server
go run main.go