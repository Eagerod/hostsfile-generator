// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HostsEntry struct {
	IP    string
	Hosts []string
}

func (he HostsEntry) String() string {
	return fmt.Sprintf(strings.Join(append([]string{he.IP}, he.Hosts...), "\t"))
}

type ConcurrentHostsFile struct {
	Lock    sync.RWMutex
	Entries map[string]HostsEntry
}

func (chf *ConcurrentHostsFile) AddHostname(ip string, hostname string) {
    entry, present := (*chf).Entries[ip]
    if !present {
        entry = HostsEntry{ip, []string{}}
        (*chf).Entries[ip] = entry
    }

    entry.Hosts = append(entry.Hosts, hostname)
}

func main() {
	ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")
	searchDomain := flag.String("search-domain", "", "Search domain to append to bare hostnames.")

	flag.Parse()

	if *ip == "" || *searchDomain == "" {
		flag.Usage()
		os.Exit(1)
	}

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Received", sig)
		done <- true
	}()

	hostsFile := ConcurrentHostsFile{sync.RWMutex{}, map[string]HostsEntry{}}

	go ManageIngressChanges(clientset, &hostsFile)
	go ManageServiceChanges(clientset, *searchDomain, &hostsFile)

	<-done
}

func ManageIngressChanges(clientset *kubernetes.Clientset, hosts *ConcurrentHostsFile) {
	api := clientset.ExtensionsV1beta1()
	listOptions := metav1.ListOptions{}
	watcher, err := api.Ingresses("").Watch(context.Background(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	ch := watcher.ResultChan()
	fmt.Println("Starting to monitor ingresses in all namespaces.")

	for event := range ch {
		ingress, ok := event.Object.(*extensionsv1beta1.Ingress)
		if !ok {
			log.Fatal("unexpected type")
		}

		ingressClass, ok := ingress.Annotations["kubernetes.io/ingress.class"]
		if !ok {
			continue
		}

		if ingressClass != "nginx" {
			continue
		}

		// If it's a redirect host, skip it.
		// Have to append these to the existing ones, not skip.
		_, isRedirect := ingress.Annotations["nginx.ingress.kubernetes.io/temporal-redirect"]
		if isRedirect {
			continue
		}

		hosts.Lock.Lock()
		hosts.Entries[ingress.Spec.Rules[0].Host] = HostsEntry{"192.168.200.128", []string{ingress.Spec.Rules[0].Host}}

		PrintHostsFile(hosts.Entries)
		fmt.Println("")
		hosts.Lock.Unlock()
	}
}

func ManageServiceChanges(clientset *kubernetes.Clientset, searchDomain string, hosts *ConcurrentHostsFile) {
	api := clientset.CoreV1()
	listOptions := metav1.ListOptions{}
	watcher, err := api.Services("").Watch(context.Background(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	ch := watcher.ResultChan()
	fmt.Println("Starting to monitor services in all namespaces.")

	for event := range ch {
		service, ok := event.Object.(*v1.Service)
		if !ok {
			log.Fatal("unexpected type")
		}

		if service.Spec.Type != "LoadBalancer" {
			continue
		}

		hosts.Lock.Lock()
		// Serivces don't include the full search domain, so append it.
		serviceName := service.ObjectMeta.Name
		serviceIP := service.Spec.LoadBalancerIP

		fqdn := serviceName + "." + searchDomain
		hosts.Entries[fqdn] = HostsEntry{serviceIP, []string{fqdn}}
		PrintHostsFile(hosts.Entries)
		fmt.Println("")
		hosts.Lock.Unlock()
	}
}

func PrintHostsFile(hostsEntries map[string]HostsEntry) {
	for _, he := range hostsEntries {
		fmt.Println(he)
	}
}
