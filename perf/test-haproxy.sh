#!/usr/bin/env bash

set -eu

trap "echo interrupt; exit 1" INT TERM

: "${PROXY_HOST:?not set}"
: "${DURATION:=60}"
: "${MB:=$HOME/src/github.com/jmencak/mb/mb}"
: "${TRAFFIC_TYPES:="edge http reencrypt passthrough"}"
: "${SAMPLES:=8}"
: "${GATHER_METADATA:=1}"
: "${GATHER_METADATA_HAPROXY:=1}"

date="$(date +%Y%m%d-%H%M%S)"
top_level_results_dir="RESULTS/$date"
metadata_dir="${top_level_results_dir}/$PROXY_HOST/.metadata"

mkdir -p "${metadata_dir}"

if [[ $GATHER_METADATA -eq 1 ]]; then
    ssh "$PROXY_HOST" sudo sysctl -a > "$metadata_dir/sysctl"
    ssh "$PROXY_HOST" cat /etc/os-release > "$metadata_dir/os-release"
    $MB version --version > "$metadata_dir/mb"
fi

if [[ $GATHER_METADATA_HAPROXY -eq 1 ]]; then
    ssh "$PROXY_HOST" rpm -qa haproxy26 > "$metadata_dir/rpm"
    ssh "$PROXY_HOST" haproxy -vv > "$metadata_dir/haproxy"
    cp ./testrun/haproxy/haproxy.config "$metadata_dir/haproxy.config"
fi

for traffic_type in ${TRAFFIC_TYPES}; do
    test_output_dir="${top_level_results_dir}/$PROXY_HOST/$traffic_type"
    mkdir -p "${test_output_dir}"
    for i in $(seq 1 $SAMPLES); do
	echo "${i}/$SAMPLES $test_output_dir"
	stdout="${test_output_dir}/${i}-${traffic_type}-${PROXY_HOST}.stdout"
	stderr="${test_output_dir}/${i}-${traffic_type}-${PROXY_HOST}.stderr"
	time_wait=0
	while [[ $(ss -a | grep -c TIME_WAIT) -gt 100 ]]; do
	    time_wait=1
	    echo -n "TIME_WAIT..."
	    sleep 1
	done
	[[ $time_wait -gt 0 ]] && echo
	echo "$MB --duration ${DURATION} --request-file ./testrun/requests/haproxy/traffic-${traffic_type}-backends-100-clients-100-keepalives-0.json > $stdout 2> $stderr"
	$MB --duration ${DURATION} --request-file "./testrun/requests/haproxy/traffic-${traffic_type}-backends-100-clients-100-keepalives-0.json" > "$stdout" 2> "$stderr"
    done
    chmod -R u-w,g-w "${test_output_dir}"
done

pushd RESULTS
rm -f latest
ln -sf "$date" latest

