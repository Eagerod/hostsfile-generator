package daemon

import (
	"errors"
	"fmt"
	"strings"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

type DaemonIngressMonitor struct {
	ingressIp string
}

func (d *DaemonIngressMonitor) Name() string {
	return "ingress"
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

func (d *DaemonIngressMonitor) GetResourceHostsEntry(obj interface{}) hostsfile.HostsEntry {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)
	if !ok {
		panic("Failed to get Ingress from pre-validated type.")
	}

	hostnames := []string{}

	for _, rule := range ingress.Spec.Rules {
		if strings.HasSuffix(rule.Host, d.ingressIp) {
			hostnames = append(hostnames, rule.Host+".")
		} else {
			hostnames = append(hostnames, rule.Host)
		}
	}

	he := hostsfile.NewHostsEntry(d.ingressIp, hostnames)
	return *he
}
