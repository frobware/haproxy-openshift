package main

import (
	"log"
	"os/exec"
)

func (c *MakeCertsCmd) Run(p *ProgramCtx) error {
	return generateCerts(p, c.Regenerate)
}

func generateCerts(p *ProgramCtx, regenerate bool) error {
	log.Printf("generating certs using %q\n", p.MkCert)
	args := "no-op"
	if regenerate {
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
