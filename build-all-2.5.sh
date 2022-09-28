#! /usr/bin/env bash

set -eu

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

git remote update

for i in $(git tag -l | grep v2.5 | grep -v dev | sort -n); do
    git checkout -f $i
    git clean -f -d -x
    echo  $i
    ${thisdir}/../haproxy-openshift/build-haproxy-2.2.sh clean all
    mv haproxy /tmp/haproxy-${i#v}
done
