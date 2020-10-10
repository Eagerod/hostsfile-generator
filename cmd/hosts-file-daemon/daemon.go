package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/Eagerod/hosts-file-daemon/pkg/hostsfile"
	"github.com/Eagerod/hosts-file-daemon/pkg/interrupt"
)

func GetKubernetesConfig(apiServerUrl string, authToken string) (*rest.Config, error) {
	// If running in the cluster, pull the service account token, else, pull
	//   the token from an environment variable.
	var config *rest.Config
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); !os.IsNotExist(err) {
		// path/to/whatever does not exist
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config = &rest.Config{}
		err := rest.SetKubernetesDefaults(config)
		if err != nil {
			return nil, err
		}

		groupVersion := schema.GroupVersion{}
		url, str, err := rest.DefaultServerURL(apiServerUrl, "v1", groupVersion, true)
		if err != nil {
			return nil, err
		}

		config.Host = url.String()
		config.APIPath = str
		config.BearerToken = authToken
		config.TLSClientConfig.Insecure = true
	}

	return config, nil
}

func Run(clusterIp string, bearerToken string, ingressIp string, piholePodName string, searchDomain string) {
	config, err := GetKubernetesConfig(clusterIp, bearerToken)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	hostsFile := hostsfile.NewConcurrentHostsFile()

	go ManageIngressChanges(clientset, config, piholePodName, ingressIp, hostsFile)
	go ManageServiceChanges(clientset, config, piholePodName, searchDomain, hostsFile)

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func ManageIngressChanges(clientset *kubernetes.Clientset, config *rest.Config, piholePodName string, ingressIp string, hosts *hostsfile.ConcurrentHostsFile) {
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

		objectId := ingress.ObjectMeta.Namespace + "/" + ingress.ObjectMeta.Name

		ingressClass, ok := ingress.Annotations["kubernetes.io/ingress.class"]
		if !ok {
			fmt.Fprintf(os.Stderr, "Skipping ingress (%s) because it doesn't have an ingress class\n", objectId)
			continue
		}

		if ingressClass != "nginx" {
			fmt.Fprintf(os.Stderr, "Skipping ingress (%s) because it doesn't belong to NGINX Ingress Controller\n", objectId)
			continue
		}

		// For each host found, add a record to the hosts file.
		// If this is an fqdn already, add it with a ., else add it as-is
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

		err := CopyFileToPod(clientset, config, piholePodName, "/etc/pihole/kube.list", hosts.String())
		if err != nil {
			log.Fatal(err)
		}

		err = ExecInPod(clientset, config, piholePodName, []string{"pihole", "restartdns"})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ManageServiceChanges(clientset *kubernetes.Clientset, config *rest.Config, piholePodName string, searchDomain string, hosts *hostsfile.ConcurrentHostsFile) {
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

		objectId := service.ObjectMeta.Namespace + "/" + service.ObjectMeta.Name

		if service.Spec.Type != "LoadBalancer" {
			fmt.Fprintf(os.Stderr, "Skipping service (%s) because it isn't of type LoadBalancer\n", objectId)
			continue
		}

		// Serivces don't include the full search domain, so append it.
		serviceName := service.ObjectMeta.Name
		serviceIp := service.Spec.LoadBalancerIP

		fqdn := serviceName + "." + searchDomain + "."
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			hosts.SetHostnames(objectId, serviceIp, []string{fqdn})
		} else if event.Type == "DELETED" {
			hosts.RemoveHostnames(objectId)
		}

		err := CopyFileToPod(clientset, config, piholePodName, "/etc/pihole/kube.list", hosts.String())
		if err != nil {
			log.Fatal(err)
		}

		err = ExecInPod(clientset, config, piholePodName, []string{"pihole", "restartdns"})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func CopyFileToPod(clientset *kubernetes.Clientset, config *rest.Config, podName string, filepath string, contents string) error {
	// There's certainly a more correct way of doing this, but that's a lot of
	//   extra code.
	script := fmt.Sprintf("cat <<EOF > %s\n%s\nEOF", filepath, contents)
	return ExecInPod(clientset, config, podName, []string{"sh", "-c", script})
}

func ReloadPiholeInPod(clientset *kubernetes.Clientset, config *rest.Config, podName string) error {
	return ExecInPod(clientset, config, podName, []string{"pihole", "restartdns"})
}

func ExecInPod(clientset *kubernetes.Clientset, config *rest.Config, podName string, command []string) error {
	api := clientset.CoreV1()

	execResource := api.RESTClient().Post().Resource("pods").Name(podName).
		Namespace("default").SubResource("exec")

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

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execResource.URL())
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
