#!/usr/bin/env bash

set -eu

# HAProxy build prerequisites.
packages="install make gcc gdb make openssl-devel pcre-devel zlib-devel diffutils sudo less vim wget strace lsof curl rsyslog procps-ng util-linux socat valgrind wireshark-cli libasan git clang"

# use: yum debuginfo-install glibc-2.28-164.el8.x86_64 libgcc-8.5.0-4.el8_5.x86_64 libxcrypt-4.1.1-6.el8.x86_64 openssl-libs-1.1.1k-5.el8_5.x86_64 pcre-8.42-6.el8.x86_64 zlib-1.2.11-17.el8.x86_64

prepare_haproxy_build_container() {
    base_image="$1"
    local_image="$2"
    container="$(buildah from $base_image)"

#   buildah run "$container" yum repolist "--disablerepo=*"
    buildah run "$container" yum -y update
    buildah run "$container" yum -y $packages
#   buildah run "$container" yum clean all

    buildah commit "${container}" $local_image
}

prepare_haproxy_build_container quay.io/centos/centos:stream8 haproxy-builder-centos-bz2044682
toolbox create --container haproxy-builder-centos-bz2044682 --image localhost/haproxy-builder-centos-bz2044682
