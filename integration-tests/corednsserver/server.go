package corednsserver

import {
	_ "embed"
}

//go:embed Corefile
var corefile string

//https://github.com/coredns/coredns/releases/download/v1.8.6/coredns_1.8.6_linux_amd64.tgz