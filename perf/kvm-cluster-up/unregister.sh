#! /usr/bin/env bash

export ANSIBLE_NOCOWS=1

set -eux

ansible-playbook -i ./hl-perf-inventory.yaml ./playbooks/rhel-unsubscribe.yaml
