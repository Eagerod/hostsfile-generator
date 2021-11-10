package daemon

import (
	"errors"
	"fmt"
	"log"
	"strings"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

type DaemonIngressMonitor struct {
	hostsfile hostsfile.IHostsFile
	ingressIp string
}

func (d *DaemonIngressMonitor) Informer(sif informers.SharedInformerFactory) cache.SharedInformer {
	return sif.Extensions().V1beta1().Ingresses().Informer()
}

func (d *DaemonIngressMonitor) ValidateResource(obj interface{}) (string, error) {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		return "", errors.New("Failed to get ingress from provided object.")
	}

	objectId := fmt.Sprintf("%s/%s", ingress.ObjectMeta.Namespace, ingress.ObjectMeta.Name)

	ingressClass, ok := ingress.Annotations["kubernetes.io/ingress.class"]
	if !ok {
		return objectId, fmt.Errorf("Skipping ingress (%s) because it doesn't have an ingress class.", objectId)
	}

	if ingressClass != "nginx" {
		return objectId, fmt.Errorf("Skipping ingress (%s) because it doesn't belong to NGINX Ingress Controller.", objectId)
	}

	return objectId, nil
}

func (d *DaemonIngressMonitor) HandleNewResource(objectId string, obj interface{}) bool {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		return false
	}

	if d.setHostsEntry(objectId, ingress) {
		log.Println("Updating hostsfile from discovered ingress", objectId)
		return true
	}

	return false
}

func (d *DaemonIngressMonitor) HandleDeletedResource(objectId string, obj interface{}) bool {
	_, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		return false
	}

	if d.hostsfile.RemoveHostsEntry(objectId) {
		log.Println("Updating hostsfile from removed ingress", objectId)
		return true
	}

	return false
}

func (d *DaemonIngressMonitor) HandleUpdatedResource(objectId string, obj interface{}) bool {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		return false
	}

	if d.setHostsEntry(objectId, ingress) {
		log.Println("Updating hostsfile from updated ingress", objectId)
		return true
	}

	return false
}

func (d *DaemonIngressMonitor) setHostsEntry(objectId string, ingress *extensionsv1beta1.Ingress) bool {
	hostnames := []string{}

	for _, rule := range ingress.Spec.Rules {
		if strings.HasSuffix(rule.Host, d.ingressIp) {
			hostnames = append(hostnames, rule.Host+".")
		} else {
			hostnames = append(hostnames, rule.Host)
		}
	}

	he := hostsfile.NewHostsEntry(d.ingressIp, hostnames)
	return d.hostsfile.SetHostsEntry(objectId, *he)
}
