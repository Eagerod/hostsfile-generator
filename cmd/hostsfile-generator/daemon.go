package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Eagerod/hostsfile-generator/pkg/daemon"
)

var VersionBuild string = "unstable-dev"

func Run() error {
	ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")
	searchDomain := flag.String("search-domain", "", "Search domain to append to bare hostnames.")
	version := flag.Bool("v", false, "Print the version and exit.")

	flag.Parse()

	if *version == true {
		fmt.Println(VersionBuild)
		return nil
	}

	if *ip == "" || *searchDomain == "" {
		flag.Usage()
		return errors.New("Invalid configuration")
	}

	// If running in the cluster, pull the service account token, else, pull
	//   the values from an environment variable.
	daemonConfig, err := daemon.NewDaemonConfigInCluster(*ip, *searchDomain)
	if err != nil {
		serverIp := os.Getenv("SERVER_IP")
		sat := os.Getenv("SERVICE_ACCOUNT_TOKEN")
		piholePodName := os.Getenv("PIHOLE_POD_NAME")

		daemonConfig, err = daemon.NewDaemonConfig(*ip, *searchDomain, serverIp, sat, piholePodName)
		if err != nil {
			return err
		}
	}

	d := daemon.NewHostsFileDaemon(*daemonConfig)
	d.Run()
	return nil
}
