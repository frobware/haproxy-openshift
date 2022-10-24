package main

import (
	"fmt"
	"net"
)

type PrintHostsCmd struct {
	Domain string `short:"d" default:"localdomain"`
}

func (c *PrintHostsCmd) Run(p *ProgramCtx) error {
	for _, t := range etcHosts(p, getOutboundIPAddr(), c.Domain) {
		fmt.Println(t)
	}

	return nil
}

func etcHosts(p *ProgramCtx, preferredIPAddr net.IP, domain string) []string {
	var names []string

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			hostname := fmt.Sprintf("%v-%v-%v", p.HostPrefix, t, i)
			hostnameFQ := hostname + "." + domain
			names = append(names, fmt.Sprintf("%v %v %v", preferredIPAddr, hostnameFQ, hostname))
		}
	}

	return names
}
