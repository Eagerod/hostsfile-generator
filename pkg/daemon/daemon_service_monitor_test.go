package daemon

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	"k8s.io/api/core/v1"
)

import (
	"github.com/Eagerod/hostsfile-generator/pkg/hostsfile"
)

func validTestService() *v1.Service {
	service := v1.Service{}
	service.ObjectMeta.Namespace = "default"
	service.ObjectMeta.Name = "some-service"
	service.Spec.Type = "LoadBalancer"
	service.Spec.LoadBalancerIP = "192.168.1.2"

	return &service
}

func TestDaemonServiceMonitorName(t *testing.T) {
	drm := DaemonServiceMonitor{}

	assert.Equal(t, "service", drm.Name())
}

func TestDaemonServiceMonitorValidateResource(t *testing.T) {
	drm := DaemonServiceMonitor{}

	service := validTestService()

	objectId, err := drm.ValidateResource(service)
	assert.Nil(t, err)
	assert.Equal(t, "default/some-service", objectId)
}

func TestDaemonServiceMonitorValidateResourceNotService(t *testing.T) {
	drm := DaemonServiceMonitor{}

	objectId, err := drm.ValidateResource(&drm)
	assert.Equal(t, "Failed to get service from provided object.", err.Error())
	assert.Equal(t, "", objectId)
}

func TestDaemonServiceMonitorValidateResourceNotLoadBalancer(t *testing.T) {
	drm := DaemonServiceMonitor{}

	service := validTestService()
	service.Spec.Type = "NodePort"

	objectId, err := drm.ValidateResource(service)
	assert.Equal(t, "Skipping service (default/some-service) because it isn't of type LoadBalancer.", err.Error())
	assert.Equal(t, "default/some-service", objectId)
}

func TestDaemonServiceMonitorGetResourceHostsEntry(t *testing.T) {
	drm := DaemonServiceMonitor{"internal.aleemhaji.com"}

	service := validTestService()

	e := hostsfile.NewHostsEntry("192.168.1.2", []string{"some-service.internal.aleemhaji.com."})
	he := drm.GetResourceHostsEntry(service)

	assert.Equal(t, *e, he)
}
