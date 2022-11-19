# !/usr/bin/env bash

MB=~/src/github.com/mb/mb

for i in http edge reencrypt passthrough; do
    echo "Waiting for TIME_WAIT Connection to be under 100"
    while [[ $(netstat | grep TIME_WAIT | wc -l) -gt 100 ]]; do
      echo -n "."
      sleep 1
    done
    echo
    echo "## Testing traffic type: $i";
    REQUEST_FILE=./testrun/requests/haproxy/traffic-${i}-backends-100-clients-50-keepalives-0.json
    out=$(LD_PRELOAD=./dns/libmydns.so $MB --duration 60 --request-file $REQUEST_FILE | tee /dev/tty)
    report="${report}\n## Traffic Type: $i\nRequest File: ${REQUEST_FILE}\n${out}"
    sleep 5
done
echo "############################################################"
echo -e "$report"
