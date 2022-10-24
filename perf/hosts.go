package main

import "fmt"

type PrintHostsCmd struct{}

func (c *PrintHostsCmd) Run(p *ProgramCtx) error {
	ipAddr := getOutboundIPAddr()
	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Backends; i++ {
			fmt.Printf("%v %v-%v-%v\n", ipAddr, p.HostnamePrefix, t, i)
		}
	}

	return nil
}
