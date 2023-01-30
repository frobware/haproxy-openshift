# !/usr/bin/env bash

set -eu

trap "echo interrupt; exit 1" INT TERM

: ${PROXY_HOST:?not set}
: ${DURATION:=60}

s1=1
s2=10

date="$(date +%Y%m%d-%H%M%S)"
top_level_results_dir="RESULTS/$date"
mkdir -p "${top_level_results_dir}/$PROXY_HOST"

for traffic_type in edge http reencrypt passthrough; do
    test_output_dir="${top_level_results_dir}/$PROXY_HOST/$traffic_type"
    mkdir -p "${test_output_dir}"
    for i in $(seq $s1 $s2); do
	echo "${i}/$s2 $test_output_dir"
	stdout="${test_output_dir}/${i}-${traffic_type}-${PROXY_HOST}.stdout"
	stderr="${test_output_dir}/${i}-${traffic_type}-${PROXY_HOST}.stderr"
	time_wait=0
	while [[ $(ss -a | grep TIME_WAIT | wc -l) -gt 100 ]]; do
	    time_wait=1
	    echo -n "TIME_WAIT..."
	    sleep 1
	done
	[[ $time_wait -gt 0 ]] && echo
	echo "~/src/github.com/jmencak/mb/mb --duration ${DURATION} --request-file ./testrun/requests/haproxy/traffic-${traffic_type}-backends-100-clients-100-keepalives-0.json > "$stdout" 2>"$stderr""
	~/src/github.com/jmencak/mb/mb --duration ${DURATION} --request-file ./testrun/requests/haproxy/traffic-${traffic_type}-backends-100-clients-100-keepalives-0.json > "$stdout" 2>"$stderr"
    done
    chmod -R u-w,g-w "${test_output_dir}"
    sleep 30
done

pushd RESULTS
rm -f latest
ln -sf "$date" latest

