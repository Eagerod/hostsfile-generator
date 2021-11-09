package daemon

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
	"github.com/Eagerod/hostsfile-generator/pkg/interrupt"
)

// Basically a container for informers
type DaemonResourceMonitor interface {
	Informer(sif informers.SharedInformerFactory) cache.SharedInformer

	ValidateResource(obj interface{}) (string, error)
	HandleNewResource(objectId string, obj interface{}) bool
	HandleDeletedResource(objectId string, obj interface{}) bool
	HandleUpdatedResource(objectId string, obj interface{}) bool
}

type HostsFileDaemon struct {
	config    DaemonConfig
	hostsfile hostsfile.IHostsFile
}

type IHostsFileDaemon interface {
	Run()

	Monitor(c chan<- bool, drm DaemonResourceMonitor)
}

func NewHostsFileDaemon(config DaemonConfig) *HostsFileDaemon {
	hfd := HostsFileDaemon{config, hostsfile.NewConcurrentHostsFile()}
	return &hfd
}

func (hfd *HostsFileDaemon) Run() {
	updatesChannel := make(chan bool, 100)
	defer close(updatesChannel)

	go func() {
		lastUpdate := time.Now()
		for _ = range updatesChannel {
			// Check the length of the channel before doing anything.
			// If there are more items in it, just let the next iteration
			//    handle the update.
			if len(updatesChannel) >= 1 {
				continue
			}

			hostsfile := hfd.hostsfile.String()
			// If the last update was more than 60 seconds ago, write this one
			//   immediately
			if time.Now().Sub(lastUpdate).Minutes() >= 1 {
				log.Println("Last update was more than 1 minute ago. Updating immediately.")
				err := WriteHostsFileAndRestartPihole(&hfd.config, hostsfile)
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

			err := WriteHostsFileAndRestartPihole(&hfd.config, hostsfile)
			if err != nil {
				log.Fatal(err)
			}
			lastUpdate = time.Now()
		}
	}()

	go hfd.Monitor(updatesChannel, &DaemonIngressMonitor{hfd})
	go hfd.Monitor(updatesChannel, &DaemonServiceMonitor{hfd})

	go func() {
		time.Sleep(time.Second * 60)
		log.Println("Forcing update of hostsfile to ensure initial launch configurations persist")
		updatesChannel <- true
	}()

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func WriteHostsFileAndRestartPihole(daemonConfig *DaemonConfig, hostsfile string) error {
	log.Println("Updating kube.list in pod:", daemonConfig.PiholePodName)
	fmt.Println(hostsfile)
	// if err := CopyFileToPod(daemonConfig, "/etc/pihole/kube.list", hostsfile); err != nil {
	// 	return err
	// }

	// log.Println("Restarting DNS service in pod:", daemonConfig.PiholePodName)
	// if err := ExecInPod(daemonConfig, []string{"pihole", "restartdns"}); err != nil {
	// 	return err
	// }

	// log.Println("Successfully restarted DNS service in pod:", daemonConfig.PiholePodName)
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

func (hfd *HostsFileDaemon) Monitor(c chan<- bool, drm DaemonResourceMonitor) {
	// Resync every minute, just in case something somehow gets missed.
	informerFactory := informers.NewSharedInformerFactory(hfd.config.KubernetesClientSet, time.Minute)

	drm.Informer(informerFactory).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				objectId, err := drm.ValidateResource(obj)
				if err != nil {
					return
				}

				if drm.HandleNewResource(objectId, obj) {
					c <- true
				}
			},
			DeleteFunc: func(obj interface{}) {
				objectId, err := drm.ValidateResource(obj)
				if err != nil {
					return
				}

				if drm.HandleDeletedResource(objectId, obj) {
					c <- true
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				objectId, err := drm.ValidateResource(newObj)
				if err != nil {
					return
				}

				if drm.HandleUpdatedResource(objectId, newObj) {
					c <- true
				}
			},
		},
	)

	stop := make(chan struct{})
	informerFactory.Start(stop)
	informerFactory.WaitForCacheSync(stop)
}
