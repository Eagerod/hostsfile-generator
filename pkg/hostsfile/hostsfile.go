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

func (he HostsEntry) String() string {
	return fmt.Sprintf(strings.Join(append([]string{he.ip}, he.hosts...), "\t"))
}

type ConcurrentHostsFile struct {
	lock    *sync.RWMutex
	entries map[string]*HostsEntry
}

func NewConcurrentHostsFile() *ConcurrentHostsFile {
	mutex := sync.RWMutex{}
	chf := ConcurrentHostsFile{&mutex, map[string]*HostsEntry{}}
	return &chf
}

func (chfptr *ConcurrentHostsFile) SetHostnames(objectId string, ip string, hostnames []string) {
	chf := *chfptr
	chf.lock.Lock()
	he := HostsEntry{ip, hostnames}
	chf.entries[objectId] = &he
	chf.lock.Unlock()
}

func (chfptr *ConcurrentHostsFile) RemoveHostnames(objectId string) {
	chf := *chfptr
	chf.lock.Lock()
	delete(chf.entries, objectId)
	chf.lock.Unlock()
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
