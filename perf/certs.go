package main

import (
	"log"
	"os/exec"
	"strings"
)

func (c *GenCertsCmd) Run(p *ProgramCtx) error {
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
		s := strings.TrimSuffix(string(out), "\n")
		if s != "" {
			log.Println(s)
		}
	}
	return err
}
