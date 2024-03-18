package hostsfile

import (
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// This file is basically a copy paste of `hostsfile_tests`.
// Not sure if there's a more sensible technique of testing things that should
// have the same interface + behaviour?
func TestConcurrentHostsFileSetHostnames(t *testing.T) {
	hf := NewConcurrentHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	assert.True(t, hf.SetHostsEntry("abc", he1))
	assert.False(t, hf.SetHostsEntry("abc", he1))
	assert.True(t, hf.SetHostsEntry("xyz", he1))
	assert.True(t, hf.SetHostsEntry("abc", he2))
}

func TestConcurrentHostsFileRemoveHostnames(t *testing.T) {
	hf := NewConcurrentHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	hf.SetHostsEntry("abc", he1)
	hf.SetHostsEntry("xyz", he2)

	assert.False(t, hf.RemoveHostsEntry("123"))
	assert.True(t, hf.RemoveHostsEntry("abc"))
	assert.False(t, hf.RemoveHostsEntry("abc"))

	_, ok := hf.hf.entries["xyz"]
	assert.True(t, ok)
}

func TestConcurrentHostsFileString(t *testing.T) {
	hf := NewConcurrentHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	hf.SetHostsEntry("abc", he1)

	assert.True(t, strings.Contains(hf.String(), he1.String()))

	hf.SetHostsEntry("xyz", he2)

	assert.True(t, strings.Contains(hf.String(), he1.String()))
	assert.True(t, strings.Contains(hf.String(), he2.String()))
}
