#! /usr/bin/env bash

# Usage:

# cd ~/git.haproxy.org
# % ls -1
# haproxy-1.8
# haproxy-1.9
# haproxy-2.0
# haproxy-2.1
# haproxy-2.2
# haproxy-2.3
# haproxy-2.4
# haproxy-2.5
# haproxy-2.6
# haproxy-2.7

# % cd haproxy-2.2
# % /u/aim/src/github.com/frobware/haproxy-openshift/build-all.sh

set -eu

thisdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
series=$(basename $PWD)
series="${series#haproxy-}"

git remote update

for i in $(git tag -l --sort=v:refname | grep "v${series}"); do
    git checkout -f $i
    git clean -f -d -x
    patch -p1 < "$thisdir/mold.patch"
    ${thisdir}/build-haproxy.sh
    mv haproxy /tmp/haproxy-${i#v}
done
