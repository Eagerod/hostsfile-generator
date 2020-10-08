// This project isn't set up to be a complete application.
// It's designed to be a super trivial script.
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    // "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

type HostsEntry struct {
    IP string
    Hosts []string
}

func (he HostsEntry) String() string {
    return fmt.Sprintf(strings.Join(append([]string{he.IP}, he.Hosts...), "\t"))
}

func main() {
    ip := flag.String("ingress-ip", "", "IP address of the NGINX Ingress Controller.")

    flag.Parse()

    if *ip == "" {
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

    api := clientset.ExtensionsV1beta1()
    listOptions := metav1.ListOptions{}
    watcher, err := api.Ingresses("").Watch(context.Background(), listOptions)

    if err != nil {
      log.Fatal(err)
    }

    ch := watcher.ResultChan()
    hostsEntries := map[string]HostsEntry{}

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
        _, isRedirect := ingress.Annotations["nginx.ingress.kubernetes.io/temporal-redirect"]
        if isRedirect {
            continue
        }

        hostsEntries[ingress.Spec.Rules[0].Host] = HostsEntry{"192.168.200.128", []string{ingress.Spec.Rules[0].Host}}

        PrintHostsFile(hostsEntries)
        fmt.Println("")
    }
}

func PrintHostsFile(hostsEntries map[string]HostsEntry) {
    for _, he := range hostsEntries {
        fmt.Println(he)
    }
}