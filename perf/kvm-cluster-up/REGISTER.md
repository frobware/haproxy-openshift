Install the machines.

Login to any machine and register because once done we can enumerate available "Employee SKUs":

	$ subscription-manager register
	username: amcdermo@redhat.com
	password: "$(pass -c rhat/access.redhat.com)"

Find a suitable/appropriate Employee SKU.

	# subscription-manager list --available --matches "Employee SKU"

Run a specific playbook

	$ ansible-playbook -u root -i ./hl-perf-inventory.yaml ./playbooks/sysctl.yaml
