package main

import (
	"fmt"
	"net"
)

func (c *GenHostsCmd) Run(p *ProgramCtx) error {
	for _, t := range etcHosts(p, HostIPAddress(), c.Domain) {
		fmt.Println(t)
	}

	return nil
}

func etcHosts(p *ProgramCtx, ipAddr net.IP, domain string) []string {
	var names []string

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			hostname := fmt.Sprintf("%v-%v-%v", p.HostPrefix, t, i)
			hostnameFQ := hostname + "." + domain
			names = append(names, fmt.Sprintf("%v %v %v", ipAddr, hostnameFQ, hostname))
		}
	}

	return names
}
