#! /usr/bin/env bash

set -eu

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

git remote update

for i in $(git tag -l --sort=v:refname | grep v2.4); do
    git checkout -f $i
    git clean -f -d -x
    echo  $i
    ${thisdir}/../haproxy-openshift/build-haproxy.sh clean all
    mv haproxy /tmp/haproxy-${i#v}
done
