package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

func validTestBetaIngress() *extensionsv1beta1.Ingress {
	ingress := extensionsv1beta1.Ingress{}
	ingress.ObjectMeta.Namespace = "default"
	ingress.ObjectMeta.Name = "some-ingress"
	ingress.Annotations = make(map[string]string)
	ingress.Annotations["kubernetes.io/ingress.class"] = "nginx"

	ingress.Spec.Rules = []extensionsv1beta1.IngressRule{
		extensionsv1beta1.IngressRule{
			Host: "some-ingress.internal.aleemhaji.com",
		},
	}

	return &ingress
}

func TestDaemonBetaIngressMonitorName(t *testing.T) {
	drm := DaemonBetaIngressMonitor{}

	assert.Equal(t, "ingress", drm.Name())
}

func TestDaemonBetaIngressMonitorValidateResource(t *testing.T) {
	drm := DaemonBetaIngressMonitor{}

	ingress := validTestBetaIngress()

	objectId, err := drm.ValidateResource(ingress)
	assert.Nil(t, err)
	assert.Equal(t, "default/some-ingress", objectId)
}

func TestDaemonBetaIngressMonitorValidateResourceNotIngress(t *testing.T) {
	drm := DaemonBetaIngressMonitor{}

	objectId, err := drm.ValidateResource(&drm)
	assert.Equal(t, "failed to get ingress from provided object", err.Error())
	assert.Equal(t, "", objectId)
}

func TestDaemonBetaIngressMonitorValidateResourceNoIngressClass(t *testing.T) {
	drm := DaemonBetaIngressMonitor{}

	ingress := validTestBetaIngress()
	delete(ingress.Annotations, "kubernetes.io/ingress.class")

	objectId, err := drm.ValidateResource(ingress)
	assert.Equal(t, "skipping ingress (default/some-ingress) because it doesn't have an ingress class", err.Error())
	assert.Equal(t, "default/some-ingress", objectId)
}

func TestDaemonBetaIngressMonitorValidateResourceNotNginxIngress(t *testing.T) {
	drm := DaemonBetaIngressMonitor{}

	ingress := validTestBetaIngress()
	ingress.Annotations["kubernetes.io/ingress.class"] = "nginx-external"

	objectId, err := drm.ValidateResource(ingress)
	assert.Equal(t, "skipping ingress (default/some-ingress) because it doesn't belong to NGINX Ingress Controller", err.Error())
	assert.Equal(t, "default/some-ingress", objectId)
}

func TestDaemonBetaIngressMonitorGetResourceHostsEntry(t *testing.T) {
	drm := DaemonBetaIngressMonitor{"192.168.1.1", "internal.aleemhaji.com"}

	ingress := validTestBetaIngress()

	e := hostsfile.NewHostsEntry("192.168.1.1", []string{"some-ingress.internal.aleemhaji.com."})
	he := drm.GetResourceHostsEntry(ingress)
	assert.Equal(t, *e, he)

	ingress.Spec.Rules = []extensionsv1beta1.IngressRule{
		extensionsv1beta1.IngressRule{
			Host: "some-ingress",
		},
	}

	e = hostsfile.NewHostsEntry("192.168.1.1", []string{"some-ingress"})
	he = drm.GetResourceHostsEntry(ingress)
	assert.Equal(t, *e, he)
}
