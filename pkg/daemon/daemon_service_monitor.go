package daemon

import (
	"errors"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

type DaemonServiceMonitor struct {
	searchDomain string
}

func (d *DaemonServiceMonitor) Name() string {
	return "service"
}

func (d *DaemonServiceMonitor) Informer(sif informers.SharedInformerFactory) cache.SharedInformer {
	return sif.Core().V1().Services().Informer()
}

func (d *DaemonServiceMonitor) ValidateResource(obj interface{}) (string, error) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return "", errors.New("failed to get service from provided object")
	}

	objectId := fmt.Sprintf("v1.service/%s/%s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)

	if service.Spec.Type != "LoadBalancer" {
		return objectId, fmt.Errorf("skipping service (%s) because it isn't of type LoadBalancer", objectId)
	}

	return objectId, nil
}

func (d *DaemonServiceMonitor) GetResourceHostsEntry(obj interface{}) hostsfile.HostsEntry {
	service, ok := obj.(*v1.Service)
	if !ok {
		panic("Failed to get service from pre-validated object.")
	}

	fqdn := fmt.Sprintf("%s.%s.", service.ObjectMeta.Name, d.searchDomain)
	he := hostsfile.NewHostsEntry(service.Spec.LoadBalancerIP, []string{fqdn})
	return *he
}
