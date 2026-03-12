# Maintainer: Limehawk <128890849+limehawk@users.noreply.github.com>
pkgname=lazyreno
pkgver=0.1.0
pkgrel=2
pkgdesc="TUI dashboard for self-hosted Renovate CE"
arch=('x86_64' 'aarch64')
url="https://github.com/limehawk/lazyreno"
license=('MIT')
makedepends=('cargo')

build() {
    cd "$startdir"
    export RUSTFLAGS="-C strip=symbols"
    cargo build --release --locked
}

package() {
    cd "$startdir"
    install -Dm755 "target/release/$pkgname" "$pkgdir/usr/bin/$pkgname"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
