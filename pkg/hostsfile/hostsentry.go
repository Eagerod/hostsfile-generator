package hostsfile

import (
	"strings"
)

type HostsEntry struct {
	ip    string
	hosts []string
}

func NewHostsEntry(ip string, hosts []string) *HostsEntry {
	he := HostsEntry{ip, hosts}
	return &he
}

func (he *HostsEntry) String() string {
	return strings.Join(append([]string{he.ip}, he.hosts...), "\t")
}

func (he *HostsEntry) Equals(other *HostsEntry) bool {
	if he.ip != other.ip {
		return false
	}

	if len(he.hosts) != len(other.hosts) {
		return false
	}

	for i, entry := range he.hosts {
		if entry != other.hosts[i] {
			return false
		}
	}

	return true
}
