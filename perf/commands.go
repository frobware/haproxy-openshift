package main

type GenProxyConfigCmd struct {
	HTTPPort  int  `default:"8080"`
	HTTPSPort int  `default:"8443"`
	Maxconn   int  `default:"0"`
	Nthreads  int  `default:"4"`
	StatsPort int  `default:"1936"`
	TLSReuse  bool `default:"true"`
}

type GenCertsCmd struct {
	Regenerate bool `default:"false" short:"r"`
}

type GenHostsCmd struct {
	Domain string `short:"d" default:"localdomain"`
}

type ServeBackendsCmd struct{}

type ServeBackendCmd struct {
	Args []string `arg:""`
}

type VersionCmd struct{}
