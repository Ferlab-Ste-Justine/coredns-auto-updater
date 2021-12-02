package main

import (
	"fmt"
	"os"

	"ferlab/coredns-auto-updater-integration-tests/etcdserver"
	"ferlab/coredns-auto-updater-integration-tests/autoupdater"
)



func main() {
	s, err := etcdserver.NewEtcdServer("/home/eric/bin/terraform", "provider.tf", "terraform-scripts")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	stop := make(chan struct {})
	
	caCert, caCertErr := s.GetCaCert()
	if caCertErr != nil {
		fmt.Println(caCertErr.Error())
		os.Exit(1)
	}

	rootCert, rootCertErr := s.GetRootCert()
	if rootCertErr != nil {
		fmt.Println(rootCertErr.Error())
		os.Exit(1)
	}

	rootKey, rootKeyErr := s.GetRootKey()
	if rootKeyErr != nil {
		fmt.Println(rootKeyErr.Error())
		os.Exit(1)
	}

	go autoupdater.LaunchDaemon("../coredns-auto-updater", ".", caCert, rootCert, rootKey, stop)
	close(stop)

	err = s.Close()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	
}