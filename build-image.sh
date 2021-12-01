#!/usr/bin/env bash

set -u
set -o pipefail
set -o errexit

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

: "${REGISTRY:=quay.io}"
: "${REGISTRY_USERNAME:=amcdermo}"
: "${IMAGENAME:=openshift-router}"

push_image=0
dry_run=""
build_container=
build_script=
containerfile=

PARAMS=""
while (( "$#" )); do
    case "$1" in
	-b|--build-container)
	    if [ -n "$2" ] && [ "${2:0:1}" != "-" ]; then
		build_container=$2
		shift 2
	    else
		echo "error: Argument for $1 is missing" >&2
		exit 1
	    fi
	    ;;
	-s|--build-script)
	    if [ -n "$2" ] && [ "${2:0:1}" != "-" ]; then
		build_script=$2
		shift 2
	    else
		echo "error: Argument for $1 is missing" >&2
		exit 1
	    fi
	    ;;
	-f|--containerfile)
	    if [ -n "$2" ] && [ "${2:0:1}" != "-" ]; then
		containerfile=$2
		shift 2
	    else
		echo "error: Argument for $1 is missing" >&2
		exit 1
	    fi
	    ;;
	-p|--push-image)
	    push_image=1
	    shift
	    ;;
	-n|--dry-run)
	    dry_run="echo"
	    shift
	    ;;
	-.|--.=)
	echo "error: Unsupported flag $1" >&2
	exit 1
	;;
	*) # preserve positional arguments
	    PARAMS="$PARAMS $1"
	    shift
	    ;;
    esac
done

if [[ -z "${build_container}" ]]; then
    echo "no build container specified (e.g., --build-container haproxy-builder-ubi8)."
    exit 1
fi

if [[ -z "${containerfile}" ]]; then
    echo "no containerfile specified (e.g., --containerfile Dockerfile.4.10)."
    exit 1
fi

if [[ -z "${build_script}" ]]; then
    echo "no haproxy build script specified (e.g., --build-script build-haproxy-1.8.sh)."
    exit 1
fi

if [[ ! -f "${build_script}" ]]; then
    echo "build script ${build_script} does not exist."
    exit 1
fi

ocpver=$(grep -oP 'router:\K\d+.\d+$' "$containerfile" || true)

if [[ -z "${ocpver}" ]]; then
    # try again but for OpenShift v3
    ocpver=$(grep -oP 'router:v\K\d+.\d+' "$containerfile" || true)
    if [[ -z "${ocpver}" ]]; then
	echo "no OCP version discovered in $containerfile."
	exit 1
    fi
fi

: "${TAGNAME:=ocp-${ocpver}-haproxy-$(git describe --tags --abbrev=0)}"

# reset positional arguments
eval set -- "$PARAMS"

$dry_run toolbox run --container "${build_container}" "${build_script}"
$dry_run podman build -t "${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}" -f "$containerfile" .

if [[ $push_image -eq 1 ]]; then
    $dry_run podman tag "${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}" "${REGISTRY}/${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}"
    $dry_run podman push "${REGISTRY}/${REGISTRY_USERNAME}/${IMAGENAME}:${TAGNAME}"
fi
