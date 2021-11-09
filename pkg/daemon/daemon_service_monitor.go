package daemon

import (
	"errors"
	"fmt"
	"log"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

type DaemonServiceMonitor struct {
	hfd *HostsFileDaemon
}

func (d *DaemonServiceMonitor) Informer(sif informers.SharedInformerFactory) cache.SharedInformer {
	return sif.Core().V1().Services().Informer()
}

func (d *DaemonServiceMonitor) ValidateResource(obj interface{}) (string, error) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return "", errors.New("Failed to get service from provided object.")
	}

	objectId := fmt.Sprintf("%s/%s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)

	if service.Spec.Type != "LoadBalancer" {
		return objectId, errors.New(fmt.Sprintf("Skipping service (%s) because it isn't of type LoadBalancer\n", objectId))
	}

	return objectId, nil
}

func (d *DaemonServiceMonitor) HandleNewResource(objectId string, obj interface{}) bool {
	service, ok := obj.(*v1.Service)
	if !ok {
		return false
	}

	if d.setHostsEntry(objectId, service) {
		log.Println("Updating hostsfile from discovered service", objectId)
		return true
	}

	return false
}

func (d *DaemonServiceMonitor) HandleDeletedResource(objectId string, obj interface{}) bool {
	_, ok := obj.(*v1.Service)
	if !ok {
		return false
	}

	if d.hfd.hostsfile.RemoveHostsEntry(objectId) {
		log.Println("Updating hostsfile from removed service", objectId)
		return true
	}

	return false
}

func (d *DaemonServiceMonitor) HandleUpdatedResource(objectId string, obj interface{}) bool {
	service, ok := obj.(*v1.Service)
	if !ok {
		return false
	}

	if d.setHostsEntry(objectId, service) {
		log.Println("Updating hostsfile from updated service", objectId)
		return true
	}

	return false
}

func (d *DaemonServiceMonitor) setHostsEntry(objectId string, service *v1.Service) bool {
	fqdn := fmt.Sprintf("%s.%s.", service.ObjectMeta.Name, d.hfd.config.SearchDomain)
	he := hostsfile.NewHostsEntry(service.Spec.LoadBalancerIP, []string{fqdn})
	return d.hfd.hostsfile.SetHostsEntry(objectId, *he)
}
