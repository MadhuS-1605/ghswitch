#!/usr/bin/env bash
# Installs the latest ghswitch release binary for this OS/arch.
# Usage: curl -fsSL https://raw.githubusercontent.com/MadhuS-1605/ghswitch/main/scripts/install.sh | bash
set -euo pipefail

REPO="MadhuS-1605/ghswitch"
INSTALL_DIR="${GHSWITCH_INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "Unsupported arch: $arch" >&2; exit 1 ;;
esac

version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$version" ]; then
  echo "Could not determine latest version." >&2
  exit 1
fi

archive="ghswitch_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${archive}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "Downloading ${url}"
curl -fsSL "$url" -o "${tmp}/${archive}"
tar -xzf "${tmp}/${archive}" -C "$tmp" ghswitch

if [ -w "$INSTALL_DIR" ]; then
  mv "${tmp}/ghswitch" "${INSTALL_DIR}/ghswitch"
else
  sudo mv "${tmp}/ghswitch" "${INSTALL_DIR}/ghswitch"
fi
chmod +x "${INSTALL_DIR}/ghswitch"
echo "Installed ghswitch ${version} to ${INSTALL_DIR}/ghswitch"
echo 'Add: eval "$(ghswitch shellinit)" to your shell rc for per-terminal identity isolation.'
