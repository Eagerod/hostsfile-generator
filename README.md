# Hostsfile generator daemon
This tool is designed to support updating hostsfiles in a Kubernetes-hosted Pihole deployment.
The applications runs continuously, and monitors Kuberetes Services and Ingresses and pushes new hostsfiles to Pihole containers.
It allows for local network DNS to remain up-to-date, even when new services as being added constantly.

Although this is called a daemon, it's really just a foreground app to run as a sidecar.
I'm not really sure what to call a complete anciliary application like this.

## Usage:

    hostsfile-daemon --ingress-ip 192.168.200.128 --search-domain internal.aleemhaji.com

Does require some values to be given as env vars in the event the application is being run outside a Kubernetes pod.

    export SERVER_IP=<Kubernetes API Server Hostname>
    export SERVICE_ACCOUNT_TOKEN=<Secret>
    export PIHOLE_POD_NAME=<Some pod name>

