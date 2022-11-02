#! /usr/bin/env bash

iptables -F
iptables -t nat -F
iptables -t mangle -F

# delete all chains
iptables -X
