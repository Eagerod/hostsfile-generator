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

type DaemonResourceMonitor interface {
	Name() string

	Informer(sif informers.SharedInformerFactory) cache.SharedInformer

	ValidateResource(obj interface{}) (string, error)
	GetResourceHostsEntry(obj interface{}) hostsfile.HostsEntry
}

type HostsFileDaemon struct {
	config         DaemonConfig
	hostsfile      hostsfile.IHostsFile
	updatesChannel chan bool
}

type IHostsFileDaemon interface {
	Run()

	Monitor(drm DaemonResourceMonitor)
}

func NewHostsFileDaemon(config DaemonConfig) *HostsFileDaemon {
	hfd := HostsFileDaemon{
		config,
		hostsfile.NewConcurrentHostsFile(),
		make(chan bool, 100),
	}
	return &hfd
}

func (hfd *HostsFileDaemon) Run() {
	defer close(hfd.updatesChannel)

	go hfd.performUpdates()
	go hfd.Monitor(&DaemonIngressMonitor{hfd.config.IngressIp, hfd.config.SearchDomain})
	go hfd.Monitor(&DaemonServiceMonitor{hfd.config.SearchDomain})
	go hfd.updateAfterInterval(time.Second*60)

	interrupt.WaitForAnySignal(syscall.SIGINT, syscall.SIGTERM)
}

func (hfd *HostsFileDaemon) Monitor(drm DaemonResourceMonitor) {
	// Resync every minute, just in case something somehow gets missed.
	informerFactory := informers.NewSharedInformerFactory(hfd.config.KubernetesClientSet, time.Minute)

	drm.Informer(informerFactory).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				objectId, err := drm.ValidateResource(obj)
				if err != nil {
					return
				}

				if hfd.hostsfile.SetHostsEntry(objectId, drm.GetResourceHostsEntry(obj)) {
					log.Printf("Creating entry for %s: %s\n", drm.Name(), objectId)
					hfd.updatesChannel <- true
				}
			},
			DeleteFunc: func(obj interface{}) {
				objectId, err := drm.ValidateResource(obj)
				if err != nil {
					return
				}

				if hfd.hostsfile.RemoveHostsEntry(objectId) {
					log.Printf("Remove entry for %s: %s\n", drm.Name(), objectId)
					hfd.updatesChannel <- true
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				objectId, err := drm.ValidateResource(newObj)
				if err != nil {
					if objectId != "" &&  hfd.hostsfile.RemoveHostsEntry(objectId) {
						log.Printf("Removing outdated entry %s: %s\n", drm.Name(), objectId)
						hfd.updatesChannel <- true
					}
					return
				}

				if hfd.hostsfile.SetHostsEntry(objectId, drm.GetResourceHostsEntry(newObj)) {
					log.Printf("Updating entry for %s: %s\n", drm.Name(), objectId)
					hfd.updatesChannel <- true
				}
			},
		},
	)

	stop := make(chan struct{})
	informerFactory.Start(stop)
	informerFactory.WaitForCacheSync(stop)
}

func (hfd *HostsFileDaemon) performUpdates() {
	lastUpdate := time.Now()
	for _ = range hfd.updatesChannel {
		// Check the length of the channel before doing anything.
		// If there are more items in it, just let the next iteration
		//    handle the update.
		if len(hfd.updatesChannel) >= 1 {
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
		if len(hfd.updatesChannel) >= 1 {
			log.Println("Aborting hostsfile update. Newer hostsfile is pending.")
			continue
		}

		err := WriteHostsFileAndRestartPihole(&hfd.config, hostsfile)
		if err != nil {
			log.Fatal(err)
		}
		lastUpdate = time.Now()
	}
}

func (hfd *HostsFileDaemon) updateAfterInterval(delay time.Duration) {
	time.Sleep(delay)
	log.Println("Forcing update to ensure consistency")
	hfd.updatesChannel <- true
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
