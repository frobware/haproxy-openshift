#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BASE_DIR=$(dirname $SCRIPT_DIR)
IMG="docker.io/envoyproxy/envoy"
VER="v1.24-latest"
PERF=${BASE_DIR}/perf-test-hydra

if [[ ! -f "$PERF" ]]; then
  echo "ERRROR: $PERF doesn't not exist"
  exit 1
fi

podman rm envoy &>/dev/null
podman run --name=envoy --network=host -it -v ${SCRIPT_DIR}/envoy-static-example-cookies.yaml:/envoy-static.yaml:z  -v ${BASE_DIR}/testrun/:${BASE_DIR}/testrun/:Z,ro ${IMG}:${VER} -c /envoy-static.yaml

#${BASE_DIR}/perf-test-hydra sync-envoy-config --enable-logging=false

#podman attach envoy
