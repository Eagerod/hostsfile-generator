package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/Eagerod/hosts-file-daemon/pkg/hostsfile"
	"github.com/Eagerod/hosts-file-daemon/pkg/interrupt"
)

func Run(daemonConfig *DaemonConfig) {
	hostsFile := hostsfile.NewConcurrentHostsFile()

	updatesChannel := make(chan *string, 100)
	defer close(updatesChannel)

	go func() {
		lastUpdate := time.Now()
		for hostsfile := range updatesChannel {
			// Check the length of the channel before doing anything.
			// If there are more items in it, just let the next iteration
			//    handle the update.
			if len(updatesChannel) >= 1 {
				continue
			}

			// If the last update was more than 60 seconds ago, write this one
			//   immediately
			if time.Now().Sub(lastUpdate).Minutes() >= 1 {
				log.Println("Last update was more than 1 minute ago. Updating immediately.")
				err := WriteHostsFileAndRestartPihole(daemonConfig, *hostsfile)
				if err != nil {
					log.Fatal(err)
				}
				lastUpdate = time.Now()
				continue
			}

			// Wait 3 seconds.
			log.Println("Waiting 1 seconds before attempting hostsfile update.")
			time.Sleep(time.Second * 1)
			if len(updatesChannel) >= 1 {
				log.Println("Aborting hostsfile update. Newer hostsfile is pending.")
				continue
			}

			err := WriteHostsFileAndRestartPihole(daemonConfig, *hostsfile)
			if err != nil {
				log.Fatal(err)
			}
			lastUpdate = time.Now()
		}
	}()

	go ManageIngressChanges(daemonConfig, updatesChannel, hostsFile)
	go ManageServiceChanges(daemonConfig, updatesChannel, hostsFile)

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func ManageIngressChanges(daemonConfig *DaemonConfig, updatesChannel chan *string, hosts *hostsfile.ConcurrentHostsFile) {
	api := daemonConfig.KubernetesClientSet.ExtensionsV1beta1()
	listOptions := metav1.ListOptions{}
	watcher, err := api.Ingresses("").Watch(context.Background(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	ch := watcher.ResultChan()
	log.Println("Starting to monitor ingresses in all namespaces.")

	for event := range ch {
		ingress, ok := event.Object.(*extensionsv1beta1.Ingress)
		if !ok {
			log.Fatal("unexpected type")
		}

		objectId := ingress.ObjectMeta.Namespace + "/" + ingress.ObjectMeta.Name

		ingressClass, ok := ingress.Annotations["kubernetes.io/ingress.class"]
		if !ok {
			log.Printf("Skipping ingress (%s) because it doesn't have an ingress class\n", objectId)
			continue
		}

		if ingressClass != "nginx" {
			log.Printf("Skipping ingress (%s) because it doesn't belong to NGINX Ingress Controller\n", objectId)
			continue
		}

		// For each host found, add a record to the hosts file.
		// If this is an fqdn already, add it with a ., else add it as-is
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			hostnames := []string{}
			for _, rule := range ingress.Spec.Rules {
				if strings.HasSuffix(rule.Host, daemonConfig.IngressIp) {
					hostnames = append(hostnames, rule.Host+".")
				} else {
					hostnames = append(hostnames, rule.Host)
				}
			}
			hosts.SetHostnames(objectId, daemonConfig.IngressIp, hostnames)
		} else if event.Type == "DELETED" {
			hosts.RemoveHostnames(objectId)
		}

		hostsFile := hosts.String()
		updatesChannel <- &hostsFile
	}
}

func ManageServiceChanges(daemonConfig *DaemonConfig, updatesChannel chan *string, hosts *hostsfile.ConcurrentHostsFile) {
	api := daemonConfig.KubernetesClientSet.CoreV1()
	listOptions := metav1.ListOptions{}
	watcher, err := api.Services("").Watch(context.Background(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	ch := watcher.ResultChan()
	log.Printf("Starting to monitor services in all namespaces.")

	for event := range ch {
		service, ok := event.Object.(*v1.Service)
		if !ok {
			log.Fatal("unexpected type")
		}

		objectId := service.ObjectMeta.Namespace + "/" + service.ObjectMeta.Name

		if service.Spec.Type != "LoadBalancer" {
			log.Printf("Skipping service (%s) because it isn't of type LoadBalancer\n", objectId)
			continue
		}

		// Serivces don't include the full search domain, so append it.
		serviceName := service.ObjectMeta.Name
		serviceIp := service.Spec.LoadBalancerIP

		fqdn := serviceName + "." + daemonConfig.SearchDomain + "."
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			hosts.SetHostnames(objectId, serviceIp, []string{fqdn})
		} else if event.Type == "DELETED" {
			hosts.RemoveHostnames(objectId)
		}

		hostsFile := hosts.String()
		updatesChannel <- &hostsFile
	}
}

func WriteHostsFileAndRestartPihole(daemonConfig *DaemonConfig, hostsfile string) error {
	log.Println("Updating kube.list in pod:", daemonConfig.PiholePodName)
	if err := CopyFileToPod(daemonConfig, "/etc/pihole/kube.list", hostsfile); err != nil {
		return err
	}

	log.Println("Restarting DNS service in pod:", daemonConfig.PiholePodName)
	if err := ExecInPod(daemonConfig, []string{"pihole", "restartdns"}); err != nil {
		return err
	}

	log.Println("Successfully restarted DNS service in pod:", daemonConfig.PiholePodName)
	return nil
}

func CopyFileToPod(daemonConfig *DaemonConfig, filepath string, contents string) error {
	// There's certainly a more correct way of doing this, but that's a lot of
	//   extra code.
	script := fmt.Sprintf("cat <<EOF > %s\n%s\nEOF", filepath, contents)
	return ExecInPod(daemonConfig, []string{"sh", "-c", script})
}

func ExecInPod(daemonConfig *DaemonConfig, command []string) error {
	api := daemonConfig.KubernetesClientSet.CoreV1()

	execResource := api.RESTClient().Post().Resource("pods").Name(daemonConfig.PiholePodName).
		Namespace("default").SubResource("exec").Param("container", "pihole")

	podExecOptions := &v1.PodExecOptions{
		Command: command,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}

	execResource.VersionedParams(
		podExecOptions,
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(daemonConfig.RestConfig, "POST", execResource.URL())
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
