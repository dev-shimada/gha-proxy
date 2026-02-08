#!/bin/bash
set -e

echo "Starting gha-proxy server in background..."
export PORT=8080
export IP_WHITELIST=127.0.0.1,::1
export AUDIENCE=https://localhost:8080
export GOPROXY_URL=https://proxy.golang.org

go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 2

echo ""
echo "Testing proxy from whitelisted IP (localhost)..."
GOPROXY=http://localhost:8080 go list -m -versions github.com/golang-jwt/jwt/v5

echo ""
echo "✅ Test passed! Proxy is working for whitelisted IPs"
echo ""
echo "To test with OIDC tokens, you need to:"
echo "1. Deploy the proxy to a public URL"
echo "2. Use it from a GitHub Actions workflow with proper OIDC token"
echo ""

# Clean up
kill $SERVER_PID
echo "Server stopped"
