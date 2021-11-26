#!/usr/bin/env bash

set -eu

: "${TAG:=latest}"

# HAProxy build prerequisites.
packages="gcc make openssl-devel pcre-devel zlib-devel diffutils sudo less vim wget"

prepare_haproxy_build_container() {
    ubi_version="$1"
    container="$(buildah from registry.access.redhat.com/${ubi_version}/ubi:${TAG})"
    tempfile="$(mktemp)"

    buildah run "${container}" yum -y --disableplugin=subscription-manager install $packages

    echo 'alias __vte_prompt_command=/bin/true' > "${tempfile}"
    buildah copy "${container}" "${tempfile}" '/etc/profile.d/vte.sh'
    buildah run "${container}" chmod 755 /etc/profile.d/vte.sh
    buildah commit "${container}" haproxy-builder-${ubi_version}

    rm "${tempfile}"
}

prepare_haproxy_build_container ubi7
prepare_haproxy_build_container ubi8

toolbox create --container haproxy-builder-ubi7 --image localhost/haproxy-builder-ubi7
toolbox create --container haproxy-builder-ubi8 --image localhost/haproxy-builder-ubi8
