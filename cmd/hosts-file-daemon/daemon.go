package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/tools/cache"

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

	go func() {
		time.Sleep(time.Second * 60)
		log.Println("Forcing update of hostsfile to ensure initial launch configurations persist")
		hostfileContents := hostsFile.String()
		err := WriteHostsFileAndRestartPihole(daemonConfig, hostfileContents)
		if err != nil {
			log.Fatal(err)
		}
	}()

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func GetNginxIngress(obj interface{}) (*extensionsv1beta1.Ingress, *string, error) {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		return nil, nil, errors.New(fmt.Sprintf("Failed to get ingress from provided object."))
	}

	objectId := ingress.ObjectMeta.Namespace + "/" + ingress.ObjectMeta.Name

	ingressClass, ok := ingress.Annotations["kubernetes.io/ingress.class"]
	if !ok {
		return nil, &objectId, errors.New(fmt.Sprintf("Skipping ingress (%s) because it doesn't have an ingress class\n", objectId))
	}

	if ingressClass != "nginx" {
		return nil, &objectId, errors.New(fmt.Sprintf("Skipping ingress (%s) because it doesn't belong to NGINX Ingress Controller\n", objectId))
	}

	return ingress, &objectId, nil
}

func UpdateHostsFromIngress(hosts *hostsfile.ConcurrentHostsFile, ingress *extensionsv1beta1.Ingress, objectId string, ingressIp string) bool {
	hostnames := []string{}
	for _, rule := range ingress.Spec.Rules {
		if strings.HasSuffix(rule.Host, ingressIp) {
			hostnames = append(hostnames, rule.Host+".")
		} else {
			hostnames = append(hostnames, rule.Host)
		}
	}
	return hosts.SetHostnames(objectId, ingressIp, hostnames)
}

func ManageIngressChanges(daemonConfig *DaemonConfig, updatesChannel chan *string, hosts *hostsfile.ConcurrentHostsFile) {
	// Resync every minute, just in case something somehow gets missed.
	informerFactory := informers.NewSharedInformerFactory(daemonConfig.KubernetesClientSet, time.Minute)

	ingressInformer := informerFactory.Extensions().V1beta1().Ingresses()
	
	ingressInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
            AddFunc: func(obj interface{}) {
				ingress, objectId, err := GetNginxIngress(obj)
				if err != nil {
					return
				}

				if UpdateHostsFromIngress(hosts, ingress, *objectId, daemonConfig.IngressIp) {
					log.Println("Updating hostsfile from discovered ingress", *objectId)
					hostsfile := hosts.String()
					updatesChannel <- &hostsfile
				}
            },
            DeleteFunc: func(obj interface{}) {
				_, objectId, err := GetNginxIngress(obj)
				if err != nil {
					return
				}

				if hosts.RemoveHostnames(*objectId) {
					log.Println("Updating hostsfile from removed ingress", *objectId)
					hostsFile := hosts.String()
					updatesChannel <- &hostsFile
				}
            },
            UpdateFunc:func(oldObj, newObj interface{}) {
				ingress, objectId, err := GetNginxIngress(newObj)
				if err != nil {
					return
				}

				if UpdateHostsFromIngress(hosts, ingress, *objectId, daemonConfig.IngressIp) {
					log.Println("Updating hostsfile from updated ingress", *objectId)
					hostsFile := hosts.String()
					updatesChannel <- &hostsFile
				}
            },
        },
	)
		
    stop := make(chan struct{})
	informerFactory.Start(stop)
	informerFactory.WaitForCacheSync(stop)
}

func GetLoadBalancerService(obj interface{}) (*v1.Service, *string, error) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return nil, nil, errors.New(fmt.Sprintf("Failed to get service from provided object."))
	}

	objectId := service.ObjectMeta.Namespace + "/" + service.ObjectMeta.Name

	if service.Spec.Type != "LoadBalancer" {
		return nil, &objectId, errors.New(fmt.Sprintf("Skipping service (%s) because it isn't of type LoadBalancer\n", objectId))
	}

	return service, &objectId, nil
}

func UpdateHostsFromService(hosts *hostsfile.ConcurrentHostsFile, service *v1.Service, objectId string, searchDomain string) bool {
	// Serivces don't include the full search domain, so append it.
	serviceName := service.ObjectMeta.Name
	serviceIp := service.Spec.LoadBalancerIP

	fqdn := serviceName + "." + searchDomain + "."
	return hosts.SetHostnames(objectId, serviceIp, []string{fqdn})
}

func ManageServiceChanges(daemonConfig *DaemonConfig, updatesChannel chan *string, hosts *hostsfile.ConcurrentHostsFile) {
	// Resync every minute, just in case something somehow gets missed.
	informerFactory := informers.NewSharedInformerFactory(daemonConfig.KubernetesClientSet, time.Minute)

	serviceInformer := informerFactory.Core().V1().Services()

	serviceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				service, objectId, err := GetLoadBalancerService(obj)
				if err != nil {
					return
				}

				if UpdateHostsFromService(hosts, service, *objectId, daemonConfig.SearchDomain) {
					log.Println("Updating hostsfile from discovered service", *objectId)
					hostsfile := hosts.String()
					updatesChannel <- &hostsfile
				}
			},
			DeleteFunc: func(obj interface{}) {
				_, objectId, err := GetLoadBalancerService(obj)
				if err != nil {
					return
				}

				if hosts.RemoveHostnames(*objectId) {
					log.Println("Updating hostsfile from removed service", *objectId)
					hostsfile := hosts.String()
					updatesChannel <- &hostsfile
				}
			},
			UpdateFunc:func(oldObj, newObj interface{}) {
				service, objectId, err := GetLoadBalancerService(newObj)
				if err != nil {
					return
				}

				if UpdateHostsFromService(hosts, service, *objectId, daemonConfig.SearchDomain) {
					log.Println("Updating hostsfile from updated service", *objectId)
					hostsfile := hosts.String()
					updatesChannel <- &hostsfile
				}
			},
		},
	)
		
	stop := make(chan struct{})
	informerFactory.Start(stop)
	informerFactory.WaitForCacheSync(stop)
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
