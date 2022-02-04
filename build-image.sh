#!/usr/bin/env bash

set -u
set -o pipefail
set -o errexit

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

: "${REGISTRY:=quay.io}"
: "${REGISTRY_USERNAME:=amcdermo}"
: "${IMAGENAME:=openshift-router-bz2044682}"

push_image=1
dry_run=""
build_container=
build_script=
containerfile=Dockerfile-rev

: ${TAGNAME="haproxy-$(git describe --tags --abbrev=0)-debug"}

$dry_run podman build -t "${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}" -f "$containerfile" .

if [[ $push_image -eq 1 ]]; then
    $dry_run podman tag "${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}" "${REGISTRY}/${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}"
    $dry_run podman push "${REGISTRY}/${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}"
fi
