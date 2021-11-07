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
//   have the same interface + behaviour?
func TestConcurrentHostsFileSetHostnames(t *testing.T) {
	hf := NewConcurrentHostsFile()

	assert.True(t, hf.SetHostnames("abc", "192.168.1.2", []string{"google.com"}))
	assert.False(t, hf.SetHostnames("abc", "192.168.1.2", []string{"google.com"}))
	assert.True(t, hf.SetHostnames("xyz", "192.168.1.2", []string{"google.com"}))
	assert.True(t, hf.SetHostnames("abc", "192.168.1.2", []string{"google.com", "www.google.com"}))
}

func TestConcurrentHostsFileRemoveHostnames(t *testing.T) {
	hf := NewConcurrentHostsFile()

	hf.SetHostnames("abc", "192.168.1.2", []string{"google.com"})
	hf.SetHostnames("xyz", "192.168.1.2", []string{"google.com", "www.google.com"})

	assert.False(t, hf.RemoveHostnames("123"))
	assert.True(t, hf.RemoveHostnames("abc"))
	assert.False(t, hf.RemoveHostnames("abc"))

	_, ok := hf.entries["xyz"]
	assert.True(t, ok)
}

func TestConcurrentHostsFileString(t *testing.T) {
	hf := NewConcurrentHostsFile()

	he1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	he2 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}

	hf.SetHostnames("abc", "192.168.1.2", []string{"google.com"})

	assert.True(t, strings.Contains(hf.String(), he1.String()))

	hf.SetHostnames("xyz", "192.168.1.2", []string{"google.com", "www.google.com"})

	assert.True(t, strings.Contains(hf.String(), he1.String()))
	assert.True(t, strings.Contains(hf.String(), he2.String()))
}
