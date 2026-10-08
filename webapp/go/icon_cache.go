package main

import "sync"

// Only public metadata is cached. Image bodies always come from the DB snapshot.
// Writes invalidate after commit, before returning success. Generation checks
// prevent an older transaction from repopulating metadata after invalidation.
var iconCache = struct {
	sync.RWMutex
	generation uint64
	hashes     map[string]string
	users      map[int64]User
}{hashes: make(map[string]string), users: make(map[int64]User)}

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
		iconCache.users = make(map[int64]User)
	} else {
		delete(iconCache.hashes, name)
		for id, user := range iconCache.users {
			if user.Name == name {
				delete(iconCache.users, id)
				break
			}
		}
	}
}
func clearIconHashes() { invalidateIconHash("") }

// Users and themes are immutable after registration. The only mutable DTO field
// is the icon hash, invalidated alongside the icon endpoint cache.
func cachedUserMetadata(id int64) (User, bool) {
	iconCache.RLock()
	defer iconCache.RUnlock()
	user, ok := iconCache.users[id]
	return user, ok
}
func rememberUserMetadata(user User, generation uint64) {
	iconCache.Lock()
	defer iconCache.Unlock()
	if generation == iconCache.generation {
		iconCache.users[user.ID] = user
		iconCache.hashes[user.Name] = user.IconHash
	}
}
