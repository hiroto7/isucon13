package main

import "sync"

// Stream rows and tags never change after reservation. Do not cache the owner
// DTO here: its icon hash can change. SQL remains the persistent source of truth.
type streamMetadata struct {
	model LivestreamModel
	tags  []Tag
}

var streamCache = struct {
	sync.RWMutex
	generation uint64
	entries    map[int64]streamMetadata
}{entries: make(map[int64]streamMetadata)}

func cloneStreamMetadata(entry streamMetadata) streamMetadata {
	entry.tags = append([]Tag{}, entry.tags...)
	return entry
}
func currentStreamGeneration() uint64 {
	streamCache.RLock()
	defer streamCache.RUnlock()
	return streamCache.generation
}
func cachedStreamMetadata(id int64, generation uint64) (streamMetadata, bool) {
	streamCache.RLock()
	defer streamCache.RUnlock()
	if generation != streamCache.generation {
		return streamMetadata{}, false
	}
	entry, ok := streamCache.entries[id]
	return cloneStreamMetadata(entry), ok
}
func publishStreamMetadata(entries map[int64]streamMetadata, generation uint64) {
	streamCache.Lock()
	defer streamCache.Unlock()
	if generation != streamCache.generation {
		return
	}
	for id, entry := range entries {
		if len(streamCache.entries) >= 65536 {
			break
		}
		streamCache.entries[id] = cloneStreamMetadata(entry)
	}
}
func clearStreamMetadata() {
	streamCache.Lock()
	defer streamCache.Unlock()
	streamCache.generation++
	streamCache.entries = make(map[int64]streamMetadata)
}
