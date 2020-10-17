package hostsfile

import (
	"fmt"
	"strings"
	"sync"
)

type HostsEntry struct {
	ip    string
	hosts []string
}

func (he *HostsEntry) String() string {
	return fmt.Sprintf(strings.Join(append([]string{he.ip}, he.hosts...), "\t"))
}

func (thisp *HostsEntry) Equals(otherp *HostsEntry) bool {
	this := *thisp
	other := *otherp

	if this.ip != other.ip {
		return false
	}

	if len(this.hosts) != len(other.hosts) {
		return false
	}

	for i, entry := range this.hosts {
		if entry != other.hosts[i] {
			return false
		}
	}

	return true
}

type ConcurrentHostsFile struct {
	lock    *sync.RWMutex
	entries map[string]*HostsEntry
	dirty   bool
}

func NewConcurrentHostsFile() *ConcurrentHostsFile {
	mutex := sync.RWMutex{}
	chf := ConcurrentHostsFile{&mutex, map[string]*HostsEntry{}, false}
	return &chf
}

func (chfptr *ConcurrentHostsFile) Lock() {
	(*chfptr).lock.Lock()
}

func (chfptr *ConcurrentHostsFile) Unlock() {
	(*chfptr).lock.Unlock()
}

func (chfptr *ConcurrentHostsFile) SetHostnames(objectId string, ip string, hostnames []string) bool {
	chf := *chfptr
	updated := false
	chfptr.Lock()
	he := HostsEntry{ip, hostnames}
	if existing, ok := chf.entries[objectId]; !ok || !existing.Equals(&he) {
		updated = true
		chf.entries[objectId] = &he
	}
	chfptr.Unlock()
	return updated
}

func (chfptr *ConcurrentHostsFile) RemoveHostnames(objectId string) bool {
	chf := *chfptr
	updated := false
	chfptr.Lock()
	if _, ok := chf.entries[objectId]; ok {
		updated = true
		delete(chf.entries, objectId)
	}
	chfptr.Unlock()
	return updated
}

func (chfptr *ConcurrentHostsFile) String() string {
	chf := *chfptr
	chf.lock.RLock()
	var sb strings.Builder
	for _, hostEntry := range chf.entries {
		sb.WriteString(hostEntry.String())
		sb.WriteString("\n")
	}
	chf.lock.RUnlock()
	return sb.String()
}
