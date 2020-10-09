// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
	"flag"
	"os"

	cmd "github.com/Eagerod/hosts-file-daemon/cmd/hosts-file-daemon"
)

func main() {
	ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")
	searchDomain := flag.String("search-domain", "", "Search domain to append to bare hostnames.")

	flag.Parse()

	if *ip == "" || *searchDomain == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Try to pull a couple values out of the environment.
	// If they're there, cool; if not, let downstream errors report it.
	serverIp := os.Getenv("SERVER_IP")
	sat := os.Getenv("SERVICE_ACCOUNT_TOKEN")

	cmd.Run(serverIp, sat, *ip, *searchDomain)
}
