# !/usr/bin/env bash

PERF_HYDRA=./perf-test-hydra

for i in http edge reencrypt passthrough; do
    echo "Waiting for TIME_WAIT Connection to be under 100"
    while [[ $(netstat | grep TIME_WAIT | wc -l) -gt 100 ]]; do
      echo -n "."
      sleep 1
    done
    echo
    echo "## Testing traffic type: $i";
    REQUEST_FILE=./testrun/requests/haproxy/traffic-${i}-backends-100-clients-50-keepalives-0.json
    out=$(LD_PRELOAD=./dns/libmydns.so $PERF_HYDRA test --duration 1s --request-file $REQUEST_FILE 2>&1 | tee /dev/tty)
    report="${report}\n## Traffic Type: $i\nRequest File: ${REQUEST_FILE}\n$(echo "$out" | grep -i 'request')\n"
    sleep 5
done
echo "############################################################"
echo -e "$report"
