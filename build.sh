#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

echo ""
echo "  VAULT  build script"
echo "  ================================"
echo ""

if ! command -v go &> /dev/null; then
  echo "  [!] go not found"
  echo "      install from https://go.dev then run again"
  exit 1
fi

OS=$(uname -s)
case "$OS" in
  MINGW*|MSYS*|CYGWIN*)
    echo "  building vault.exe (windows)..."
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o vault.exe .
    echo "  done. output: vault.exe"
    ;;
  Darwin)
    echo "  building vault (macos)..."
    go build -ldflags="-s -w" -o vault .
    echo "  done. output: vault"
    ;;
  Linux)
    echo "  building vault (linux)..."
    go build -ldflags="-s -w" -o vault .
    echo "  done. output: vault"
    ;;
  *)
    echo "  building vault..."
    go build -ldflags="-s -w" -o vault .
    echo "  done. output: vault"
    ;;
esac

echo ""
