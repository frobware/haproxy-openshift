#! /usr/bin/env bash

set -eu

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

if ! type -P mkcert >/dev/null 2>&1; then
    echo "No mkcert utilility."
    echo "Install as: "
    echo "go install filippo.io/mkcert@latest"
    exit 1
fi

certdir="$thisdir/certs"
regenerate=0
PARAMS=""

while (( "$#" )); do
    case "$1" in
	-r|--regenerate)
	    regenerate=1; shift
	    ;;
	*) # preserve positional arguments
	    PARAMS="$PARAMS $1"
	    shift
	    ;;
    esac
done

# reset positional arguments
eval set -- "$PARAMS"

if [[ -f "$certdir/full-chain.pem" ]] && [[ $regenerate -eq 0 ]]; then
    exit 0
fi

export CAROOT="$certdir"

mkdir -p "$certdir"
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

cat "$certdir/rootCA-key.pem" "$certdir/rootCA.pem" "${certdir}/tls.key" "${certdir}/tls.crt" > "$certdir/full-chain.pem"
ls -lR $certdir
