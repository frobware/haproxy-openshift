# !/usr/bin/env bash

for i in edge http reencrypt passthrough; do
    echo "## Testing traffic type: $i";
    ~/mb/mb --ramp-up 5 --duration 60 --request-file ./testrun/requests/direct/traffic-${i}-backends-100-clients-100-keepalives-0.json
    sleep 180
done
