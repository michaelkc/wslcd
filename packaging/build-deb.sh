#!/usr/bin/env bash
# Builds a .deb package for wslcd using only dpkg-deb (no external tooling).
#
# Usage: ./packaging/build-deb.sh <version> <deb-arch> <binary-path> [output-dir]
#   e.g. ./packaging/build-deb.sh 1.0.0 amd64 ./wslcd dist
set -euo pipefail

VERSION="${1:?usage: build-deb.sh <version> <deb-arch> <binary> [output-dir]}"
ARCH="${2:?usage: build-deb.sh <version> <deb-arch> <binary> [output-dir]}"
BIN="${3:?usage: build-deb.sh <version> <deb-arch> <binary> [output-dir]}"
OUTDIR="${4:-dist}"

[ -x "$BIN" ] || { echo "error: binary not found or not executable: $BIN" >&2; exit 1; }

PKGDIR="$(mktemp -d)"
trap 'rm -rf "$PKGDIR"' EXIT

install -D -m 0755 "$BIN"                       "$PKGDIR/usr/bin/wslcd"
install -D -m 0644 packaging/wslcd.sh           "$PKGDIR/etc/profile.d/wslcd.sh"
install -D -m 0644 README.md                    "$PKGDIR/usr/share/doc/wslcd/README.md"
mkdir -p "$PKGDIR/DEBIAN"

cat > "$PKGDIR/usr/share/doc/wslcd/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: wslcd
Source: https://github.com/michaelkc/wslcd

Files: *
Copyright: 2026 Michael Christensen
License: MIT
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
EOF

cat > "$PKGDIR/DEBIAN/control" <<EOF
Package: wslcd
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: michaelkc <michaelkc@users.noreply.github.com>
Description: Resolve Windows paths to /mnt paths for cd-ing in WSL
 wslcd maps a Windows path (e.g. C:\Users\me) to its /mnt/c equivalent,
 resolving path segments case-insensitively. Includes a shell wrapper in
 /etc/profile.d that makes 'wslcd' change the shell's directory directly.
EOF

mkdir -p "$OUTDIR"
DEB="$OUTDIR/wslcd_${VERSION}_${ARCH}.deb"
dpkg-deb --root-owner-group --build "$PKGDIR" "$DEB"
echo "Built $DEB"
