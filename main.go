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
	"syscall"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    
    "github.com/Eagerod/hosts-file-daemon/pkg/hostsfile"
)

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

	hostsFile := hostsfile.NewConcurrentHostsFile()

	go ManageIngressChanges(clientset, *ip, hostsFile)
	go ManageServiceChanges(clientset, *searchDomain, hostsFile)

	<-done
}

func ManageIngressChanges(clientset *kubernetes.Clientset, ingressIP string, hosts *hostsfile.ConcurrentHostsFile) {
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

        hosts.AddHostname(ingressIP, ingress.Spec.Rules[0].Host)
        fmt.Println(hosts)
	}
}

func ManageServiceChanges(clientset *kubernetes.Clientset, searchDomain string, hosts *hostsfile.ConcurrentHostsFile) {
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

		// Serivces don't include the full search domain, so append it.
		serviceName := service.ObjectMeta.Name
		serviceIP := service.Spec.LoadBalancerIP

        fqdn := serviceName + "." + searchDomain
        hosts.AddHostname(serviceIP, fqdn)
        fmt.Println(hosts)
	}
}
