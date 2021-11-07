package hostsfile

import (
	"strings"
)

type HostsEntry struct {
	ip    string
	hosts []string
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
