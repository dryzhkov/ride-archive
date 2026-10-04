#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

install_with_brew() {
  local package="$1"
  if ! command -v brew >/dev/null 2>&1; then
    echo "Missing required tool '$package'. Install Homebrew, then rerun ./setup.sh." >&2
    exit 1
  fi
  echo "Installing $package with Homebrew..."
  brew install "$package"
}

if ! command -v go >/dev/null 2>&1; then
  install_with_brew go
fi
if ! command -v npm >/dev/null 2>&1; then
  install_with_brew node
fi

go_version="$(go env GOVERSION)"
if [[ ! "$go_version" =~ ^go1\.(27|[3-9][0-9])([.]|$) ]] && [[ ! "$go_version" =~ ^go([2-9][0-9]*)\. ]]; then
  echo "Ride Archive requires Go 1.27 or newer; found $go_version." >&2
  exit 1
fi

echo "Installing frontend dependencies..."
npm ci --prefix web

echo "Checking and building the frontend..."
npm run check --prefix web
npm run build --prefix web

echo "Downloading Go modules and verifying the backend..."
go mod download
go test ./...
go vet ./...

echo "Building Ride Archive..."
mkdir -p bin
go build -trimpath -o bin/ride-archive ./cmd/api

echo "Starting Ride Archive at http://127.0.0.1:8080 (Ctrl-C to stop)."
exec "$ROOT/bin/ride-archive"
