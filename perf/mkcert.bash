#! /usr/bin/env bash

set -eu

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

if ! type -P mkcert; then
    echo "No mkcert utilility."
    echo "Install as: "
    echo "go install filippo.io/mkcert@latest"
    exit 1
fi

certdir="$thisdir/certs"
mkdir -p "$certdir"

export CAROOT="$certdir"

mkcert \
    -client \
    -cert-file "$certdir/tls.crt" \
    -key-file  "$certdir/tls.key" \
    "$(hostname)" \
    "$(hostname -s).localdomain" \
    "$(hostname -f)" \
    localhost \
    127.0.0.1 \
    ::1

if ! [[ -L tls.key ]]; then
    echo "expected tls.key to be a symlink; not removing"
    exit 1
fi

if ! [[ -L tls.crt ]]; then
    echo "expected tls.crt to be a symlink; not removing"
    exit 1
fi

rm -f tls.key tls.crt
ln -sf "$certdir/tls.crt" tls.crt
ln -sf "$certdir/tls.key" tls.key

# Sanity check; will exit with an error if they don't resolve.
ls -lL tls.crt
ls -lL tls.key
ls -lR "$certdir"
