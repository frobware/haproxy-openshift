package main

import (
	"fmt"
)

func (c *GenHostsCmd) Run(p *ProgramCtx) error {
	addr := mustResolveHostIP()
	if c.IPAddress != "" {
		addr = c.IPAddress
	}
	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			hostname := fmt.Sprintf("%v-%v-%v", p.HostPrefix, t, i)
			fmt.Println(addr, hostname)
		}
	}
	return nil
}
