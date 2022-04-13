package daemon

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

func TestInformerAddFunc(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}
	f := hfd.InformerAddFunc(&dsm)

	i := validTestIngress()
	f(i)

	assert.Equal(t, 1, len(hfd.updatesChannel))
}

func TestInformerAddFuncWrongType(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}
	f := hfd.InformerAddFunc(&dsm)

	i := validTestService()
	f(i)

	assert.Equal(t, 0, len(hfd.updatesChannel))
}

func TestInformerDeleteFunc(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}

	i := validTestIngress()

	objectId, err := dsm.ValidateResource(i)
	assert.NoError(t, err)
	he := dsm.GetResourceHostsEntry(i)
	hfd.hostsfile.SetHostsEntry(objectId, he)

	f := hfd.InformerDeleteFunc(&dsm)
	f(i)

	assert.Equal(t, 1, len(hfd.updatesChannel))
}

func TestInformerDeleteFuncWrongType(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}

	i := validTestIngress()

	objectId, err := dsm.ValidateResource(i)
	assert.NoError(t, err)
	he := dsm.GetResourceHostsEntry(i)
	hfd.hostsfile.SetHostsEntry(objectId, he)

	ii := validTestService()
	f := hfd.InformerDeleteFunc(&dsm)
	f(ii)

	assert.Equal(t, 0, len(hfd.updatesChannel))
}

func TestInformerUpdateFunc(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}

	i := validTestIngress()

	objectId, err := dsm.ValidateResource(i)
	assert.NoError(t, err)
	he := dsm.GetResourceHostsEntry(i)
	hfd.hostsfile.SetHostsEntry(objectId, he)

	ii := *i
	ii.Spec.Rules = []extensionsv1beta1.IngressRule{
		extensionsv1beta1.IngressRule{
			Host: "another-ingress.internal.aleemhaji.com",
		},
	}

	f := hfd.InformerUpdateFunc(&dsm)
	f(i, &ii)

	assert.Equal(t, 1, len(hfd.updatesChannel))
}

func TestInformerUpdateFuncWrongType(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}

	i := validTestIngress()

	objectId, err := dsm.ValidateResource(i)
	assert.NoError(t, err)
	he := dsm.GetResourceHostsEntry(i)
	hfd.hostsfile.SetHostsEntry(objectId, he)

	ii := validTestService()
	f := hfd.InformerUpdateFunc(&dsm)
	f(i, &ii)

	assert.Equal(t, 0, len(hfd.updatesChannel))
}

func TestInformerUpdateFuncInvalidated(t *testing.T) {
	dc, err := NewDaemonConfig("1", "2", "3", "4", "5")
	assert.Nil(t, err)

	hfd := NewHostsFileDaemon(*dc)
	dsm := DaemonIngressMonitor{"1", "2"}

	i := validTestIngress()

	objectId, err := dsm.ValidateResource(i)
	assert.NoError(t, err)
	he := dsm.GetResourceHostsEntry(i)
	hfd.hostsfile.SetHostsEntry(objectId, he)

	ii := *i
	ii.Annotations["kubernetes.io/ingress.class"] = "nginx-external"

	f := hfd.InformerUpdateFunc(&dsm)
	f(i, &ii)

	assert.Equal(t, 1, len(hfd.updatesChannel))

	// Assert that the entry has been removed from the update.
	assert.False(t, hfd.hostsfile.RemoveHostsEntry(objectId))
}
