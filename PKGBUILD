# Maintainer: Limehawk <limehawk@users.noreply.github.com>
pkgname=lazyreno
pkgver=0.1.0
pkgrel=1
pkgdesc="TUI dashboard for self-hosted Renovate CE"
arch=('x86_64' 'aarch64')
url="https://github.com/limehawk/lazyreno"
license=('MIT')
makedepends=('go')

build() {
    cd "$startdir"
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$srcdir/$pkgname" ./cmd/lazyreno
}

package() {
    cd "$srcdir"
    install -Dm755 "$pkgname" "$pkgdir/usr/bin/$pkgname"
    install -Dm644 "$startdir/LICENSE" "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
