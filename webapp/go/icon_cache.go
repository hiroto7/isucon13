package main

import "sync"

// Only public metadata is cached. Image bodies always come from the DB snapshot.
// Writes invalidate after commit, before returning success. Generation checks
// prevent an older transaction from repopulating metadata after invalidation.
var iconCache = struct {
	sync.RWMutex
	generation uint64
	hashes     map[string]string
}{hashes: make(map[string]string)}

func cachedIconHash(name string) (string, bool, uint64) {
	iconCache.RLock()
	defer iconCache.RUnlock()
	hash, ok := iconCache.hashes[name]
	return hash, ok, iconCache.generation
}
func rememberIconHash(name, hash string, generation uint64) {
	iconCache.Lock()
	defer iconCache.Unlock()
	if generation == iconCache.generation {
		iconCache.hashes[name] = hash
	}
}
func invalidateIconHash(name string) {
	iconCache.Lock()
	defer iconCache.Unlock()
	iconCache.generation++
	if name == "" {
		iconCache.hashes = make(map[string]string)
	} else {
		delete(iconCache.hashes, name)
	}
}
func clearIconHashes() { invalidateIconHash("") }
