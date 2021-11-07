package hostsfile

import (
	"strings"
)

type IHostsFile interface {
	SetHostsEntry(objectId string, entry HostsEntry) bool
	RemoveHostsEntry(objectId string) bool

	String() string
}

type HostsFile struct {
	entries    map[string]*HostsEntry
}

func NewHostsFile() *HostsFile {
	hf := HostsFile{
		map[string]*HostsEntry{},
	}

	return &hf
}

func (hf *HostsFile) SetHostsEntry(objectId string, entry HostsEntry) bool {
	updated := false

	if existing, ok := hf.entries[objectId]; !ok || !existing.Equals(&entry) {
		updated = true
		hf.entries[objectId] = &entry
	}

	return updated
}

func (hf *HostsFile) RemoveHostsEntry(objectId string) bool {
	updated := false

	if _, ok := hf.entries[objectId]; ok {
		updated = true
		delete(hf.entries, objectId)
	}

	return updated
}

func (hf *HostsFile) String() string {
	var sb strings.Builder

	for _, hostEntry := range hf.entries {
		sb.WriteString(hostEntry.String())
		sb.WriteString("\n")
	}

	return sb.String()
}
