package hostsfile

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestNewHostsEntry(t *testing.T) {
	he := NewHostsEntry("192.168.1.2", []string{"google.com"})
	assert.Equal(t, he.ip, "192.168.1.2")
	assert.Equal(t, he.hosts, []string{"google.com"})
}

func TestHostsEntryString(t *testing.T) {
	var tests = []struct {
		name  string
		ip    string
		hosts []string
		rv    string
	}{
		{"One Domain", "192.168.1.2", []string{"google.com"}, "192.168.1.2	google.com"},
		{"Multiple Domains", "192.168.1.2", []string{"google.com", "www.google.com"}, "192.168.1.2	google.com	www.google.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			he := HostsEntry{tt.ip, tt.hosts}
			assert.Equal(t, he.String(), tt.rv)
		})
	}
}

func TestHostsEntryEqual(t *testing.T) {
	h1 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	h2 := HostsEntry{"192.168.1.2", []string{"google.com"}}
	h3 := HostsEntry{"192.168.1.2", []string{"google.com", "www.google.com"}}
	h4 := HostsEntry{"192.168.1.1", []string{"google.com"}}
	h5 := HostsEntry{"192.168.1.2", []string{"www.google.com"}}

	assert.True(t, h1.Equals(&h1))
	assert.True(t, h1.Equals(&h2))
	assert.False(t, h1.Equals(&h3))
	assert.False(t, h1.Equals(&h4))
	assert.False(t, h1.Equals(&h5))
}
