package hostsfile

import (
	"strings"
	"sync"
)

type ConcurrentHostsFile struct {
	lock    *sync.RWMutex
	entries map[string]*HostsEntry
}

func NewConcurrentHostsFile() *ConcurrentHostsFile {
	mutex := sync.RWMutex{}
	chf := ConcurrentHostsFile{&mutex, map[string]*HostsEntry{}}
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
