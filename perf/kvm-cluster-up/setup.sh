#! /usr/bin/env bash

export ANSIBLE_NOCOWS=1
ansible-playbook -u root --become-user=root -i hl-perf-inventory.yaml ./initial-setup.yaml -e "foo@bar.com" -e "password=$(pass rhat/access.redhat.com)"
