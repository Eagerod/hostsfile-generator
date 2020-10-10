// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	cmd "github.com/Eagerod/hosts-file-daemon/cmd/hosts-file-daemon"
)

func main() {
	ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")
	searchDomain := flag.String("search-domain", "", "Search domain to append to bare hostnames.")
	version := flag.Bool("v", false, "Print the version and exit.")

	flag.Parse()

	if *version == true {
		fmt.Println(cmd.VersionBuild)
		return
	}

	if *ip == "" || *searchDomain == "" {
		flag.Usage()
		os.Exit(1)
	}

	daemonConfig, err := cmd.NewDaemonConfigInCluster(*ip, *searchDomain)
	if err != nil {
		serverIp := os.Getenv("SERVER_IP")
		sat := os.Getenv("SERVICE_ACCOUNT_TOKEN")
		piholePodName := os.Getenv("PIHOLE_POD_NAME")

		daemonConfig, err = cmd.NewDaemonConfig(*ip, *searchDomain, serverIp, sat, piholePodName)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(2)
		}
	}

	fmt.Fprintf(os.Stderr, "Waiting 60 seconds to allow pihole to start before monitoring resources...\n")
	time.Sleep(time.Second * 60)

	cmd.Run(daemonConfig)
}
