// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
	"flag"
	"os"

	daemon "github.com/Eagerod/hosts-file-daemon/cmd/hosts-file-daemon"
)

func main() {
	ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")
	searchDomain := flag.String("search-domain", "", "Search domain to append to bare hostnames.")

	flag.Parse()

	if *ip == "" || *searchDomain == "" {
		flag.Usage()
		os.Exit(1)
	}

	daemon.Run(*ip, *searchDomain)
}
