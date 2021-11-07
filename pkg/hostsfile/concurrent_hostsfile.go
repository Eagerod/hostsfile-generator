package hostsfile

import (
	"sync"
)

type ConcurrentHostsFile struct {
	lock    *sync.RWMutex
	hf 		*HostsFile
}

func NewConcurrentHostsFile() *ConcurrentHostsFile {
	mutex := sync.RWMutex{}
	chf := ConcurrentHostsFile{&mutex, NewHostsFile()}
	return &chf
}

func (chfptr *ConcurrentHostsFile) Lock() {
	chfptr.lock.Lock()
}

func (chfptr *ConcurrentHostsFile) Unlock() {
	chfptr.lock.Unlock()
}

func (chfptr *ConcurrentHostsFile) SetHostsEntry(objectId string, he HostsEntry) bool {
	chfptr.Lock()
	rv := chfptr.hf.SetHostsEntry(objectId, he)
	chfptr.Unlock()
	return rv
}

func (chfptr *ConcurrentHostsFile) RemoveHostsEntry(objectId string) bool {
	chfptr.Lock()
	rv := chfptr.hf.RemoveHostsEntry(objectId)
	chfptr.Unlock()
	return rv
}

func (chfptr *ConcurrentHostsFile) String() string {
	chfptr.lock.RLock()
	rv := chfptr.hf.String()
	chfptr.lock.RUnlock()
	return rv
}
