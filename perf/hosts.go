package main

import (
	"fmt"
)

func (c *GenHostsCmd) Run(p *ProgramCtx) error {
	addr := mustResolveHostIP()
	if c.IPAddress != "" {
		addr = c.IPAddress
	}
	for _, t := range etcHosts(p, addr) {
		fmt.Println(t)
	}

	return nil
}

func etcHosts(p *ProgramCtx, ipAddr string) []string {
	var names []string

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			hostname := fmt.Sprintf("%v-%v-%v", p.HostPrefix, t, i)
			names = append(names, fmt.Sprintf("%v %v", ipAddr, hostname))
		}
	}

	return names
}
