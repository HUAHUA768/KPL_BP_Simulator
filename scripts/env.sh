#!/bin/bash
# KPL BP Simulator — Environment Setup Script
# Usage: source scripts/env.sh  (or . scripts/env.sh)

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Go
export PATH="$PROJECT_ROOT/go/bin:$PATH"
export GOPATH="$PROJECT_ROOT/backend/.gopath"
export GOCACHE="$PROJECT_ROOT/.gocache"
export GOMODCACHE="$PROJECT_ROOT/.gomodcache"
export GOPROXY="https://goproxy.cn,direct"
export GONOSUMDB="*"
export GONOSUMCHECK="*"
export GOFLAGS="-mod=mod"

# npm
export npm_config_cache="$PROJECT_ROOT/.npm-cache"

echo "✅ Environment configured for KPL BP Simulator"
echo "   Go:      $(go version 2>/dev/null || echo 'not found')"
echo "   Node:    $(node --version 2>/dev/null || echo 'not found')"
echo "   npm:     $(npm --version 2>/dev/null || echo 'not found')"
echo "   GOPATH:  $GOPATH"
echo "   Project: $PROJECT_ROOT"
