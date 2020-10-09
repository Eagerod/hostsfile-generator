package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Eagerod/hosts-file-daemon/pkg/hostsfile"
	"github.com/Eagerod/hosts-file-daemon/pkg/interrupt"
)

func Run(ingressIp string, searchDomain string) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	hostsFile := hostsfile.NewConcurrentHostsFile()

	go ManageIngressChanges(clientset, ingressIp, hostsFile)
	go ManageServiceChanges(clientset, searchDomain, hostsFile)

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func ManageIngressChanges(clientset *kubernetes.Clientset, ingressIp string, hosts *hostsfile.ConcurrentHostsFile) {
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

		// For each host found, add a record to the hosts file.
		// If this is an fqdn already, add it with a ., else add it as-is
		objectId := ingress.ObjectMeta.Namespace + "/" + ingress.ObjectMeta.Name
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			hostnames := []string{}
			for _, rule := range ingress.Spec.Rules {
				if strings.HasSuffix(rule.Host, ingressIp) {
					hostnames = append(hostnames, rule.Host+".")
				} else {
					hostnames = append(hostnames, rule.Host)
				}
			}
			hosts.SetHostnames(objectId, ingressIp, hostnames)
		} else if event.Type == "DELETED" {
			hosts.RemoveHostnames(objectId)
		}

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
		serviceIp := service.Spec.LoadBalancerIP

		fqdn := serviceName + "." + searchDomain + "."
		objectId := service.ObjectMeta.Namespace + "/" + service.ObjectMeta.Name
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			hosts.SetHostnames(objectId, serviceIp, []string{fqdn})
		} else if event.Type == "DELETED" {
			hosts.RemoveHostnames(objectId)
		}

		fmt.Println(hosts)
	}
}
