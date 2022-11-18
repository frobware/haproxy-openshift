#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BASE_DIR=$(dirname $SCRIPT_DIR)
IMG="docker.io/envoyproxy/envoy"
VER="v1.24-latest"

podman run --name=envoy --rm --network=host -it -v ${SCRIPT_DIR}/envoy-dynamic.yaml:/envoy-dynamic.yaml:z  -v ${BASE_DIR}/testrun/:${BASE_DIR}/testrun/:Z,ro ${IMG}:${VER} -c /envoy-dynamic.yaml
