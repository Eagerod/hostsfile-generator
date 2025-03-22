package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

var DefaultIngressClass string = "nginx"
var InvalidIngressClass string = "nginx-external"
var EmptyIngressClass string = ""

func validTestIngress() *networkingv1.Ingress {
	ingress := networkingv1.Ingress{}
	ingress.ObjectMeta.Namespace = "default"
	ingress.ObjectMeta.Name = "some-ingress"
	ingress.Annotations = make(map[string]string)
	ingress.Spec.IngressClassName = &DefaultIngressClass

	ingress.Spec.Rules = []networkingv1.IngressRule{
		networkingv1.IngressRule{
			Host: "some-ingress.internal.aleemhaji.com",
		},
	}

	return &ingress
}

func TestDaemonIngressMonitorName(t *testing.T) {
	drm := DaemonIngressMonitor{}

	assert.Equal(t, "ingress", drm.Name())
}

func TestDaemonIngressMonitorValidateResource(t *testing.T) {
	drm := DaemonIngressMonitor{}

	ingress := validTestIngress()

	objectId, err := drm.ValidateResource(ingress)
	assert.Nil(t, err)
	assert.Equal(t, "networkingv1.ingress/default/some-ingress", objectId)
}

func TestDaemonIngressMonitorValidateResourceLegacy(t *testing.T) {
	drm := DaemonIngressMonitor{}

	ingress := validTestIngress()
	ingress.Annotations["kubernetes.io/ingress.class"] = "nginx"

	objectId, err := drm.ValidateResource(ingress)
	assert.Nil(t, err)
	assert.Equal(t, "networkingv1.ingress/default/some-ingress", objectId)
}

func TestDaemonIngressMonitorValidateResourceNotIngress(t *testing.T) {
	drm := DaemonIngressMonitor{}

	objectId, err := drm.ValidateResource(&drm)
	assert.Equal(t, "failed to get ingress from provided object", err.Error())
	assert.Equal(t, "", objectId)
}

func TestDaemonIngressMonitorValidateResourceNoIngressClass(t *testing.T) {
	drm := DaemonIngressMonitor{}

	ingress := validTestIngress()
	ingress.Spec.IngressClassName = &EmptyIngressClass

	objectId, err := drm.ValidateResource(ingress)
	assert.Equal(t, "skipping ingress (networkingv1.ingress/default/some-ingress) because it doesn't have an ingress class", err.Error())
	assert.Equal(t, "networkingv1.ingress/default/some-ingress", objectId)
}

func TestDaemonIngressMonitorValidateResourceNotNginxIngress(t *testing.T) {
	drm := DaemonIngressMonitor{}

	ingress := validTestIngress()
	ingress.Spec.IngressClassName = &InvalidIngressClass

	objectId, err := drm.ValidateResource(ingress)
	assert.Equal(t, "skipping ingress (networkingv1.ingress/default/some-ingress) because it doesn't belong to NGINX Ingress Controller", err.Error())
	assert.Equal(t, "networkingv1.ingress/default/some-ingress", objectId)
}

func TestDaemonIngressMonitorGetResourceHostsEntry(t *testing.T) {
	drm := DaemonIngressMonitor{"192.168.1.1", "internal.aleemhaji.com"}

	ingress := validTestIngress()

	e := hostsfile.NewHostsEntry("192.168.1.1", []string{"some-ingress.internal.aleemhaji.com."})
	he := drm.GetResourceHostsEntry(ingress)
	assert.Equal(t, *e, he)

	ingress.Spec.Rules = []networkingv1.IngressRule{
		networkingv1.IngressRule{
			Host: "some-ingress",
		},
	}

	e = hostsfile.NewHostsEntry("192.168.1.1", []string{"some-ingress"})
	he = drm.GetResourceHostsEntry(ingress)
	assert.Equal(t, *e, he)
}
