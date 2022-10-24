package main

import (
	"log"
	"os/exec"
)

type MakeCertsCmd struct {
	Regenerate bool `default:"false" short:"r"`
}

func (c *MakeCertsCmd) Run(p *ProgramCtx) error {
	log.Printf("generating certs using %q\n", p.MkCert)
	args := "no-op"
	if c.Regenerate {
		args = "-r"
	}
	out, err := exec.Command(p.MkCert, args).CombinedOutput()
	if err != nil {
		log.Println(string(out), err)
	} else {
		log.Println(string(out))
	}
	return err
}
