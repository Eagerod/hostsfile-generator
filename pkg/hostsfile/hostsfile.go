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

func (chfptr *ConcurrentHostsFile) AddHostname(ip string, hostname string) {
	chf := *chfptr
	chf.lock.Lock()
	entry, present := chf.entries[ip]
	if !present {
		fmt.Println("Didn't find", ip, hostname)
		he := HostsEntry{ip, []string{}}
		entry = &he
		chf.entries[ip] = entry
	}

	entry.hosts = append(entry.hosts, hostname)
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
