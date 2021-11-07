package hostsfile

import (
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestHostsFileSetHostsEntry(t *testing.T) {
	hf := NewHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	assert.True(t, hf.SetHostsEntry("abc", he1))
	assert.False(t, hf.SetHostsEntry("abc", he1))
	assert.True(t, hf.SetHostsEntry("xyz", he1))
	assert.True(t, hf.SetHostsEntry("abc", he2))
}

func TestHostsFileRemoveHostsEntry(t *testing.T) {
	hf := NewHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	hf.SetHostsEntry("abc", he1)
	hf.SetHostsEntry("xyz", he2)

	assert.False(t, hf.RemoveHostsEntry("123"))
	assert.True(t, hf.RemoveHostsEntry("abc"))
	assert.False(t, hf.RemoveHostsEntry("abc"))

	_, ok := hf.entries["xyz"]
	assert.True(t, ok)
}

func TestHostsFileString(t *testing.T) {
	hf := NewHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	hf.SetHostsEntry("abc", he1)

	assert.True(t, strings.Contains(hf.String(), he1.String()))

	hf.SetHostsEntry("xyz", he2)

	assert.True(t, strings.Contains(hf.String(), he1.String()))
	assert.True(t, strings.Contains(hf.String(), he2.String()))
}
