package hostsfile

import (
// 	"fmt"
// 	"strings"
// 	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestHostsEntryString(t *testing.T) {
	var tests = []struct {
		name string
		ip  string
		hosts  []string
		rv  string
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
