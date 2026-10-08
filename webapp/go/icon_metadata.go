package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"sync"
)

// The fallback is an immutable deployed asset, independent of initialized DB data.
var fallbackHashOnce sync.Once
var fallbackHashValue string
var fallbackHashError error

func defaultIconHash() (string, error) {
	fallbackHashOnce.Do(func() {
		var image []byte
		image, fallbackHashError = os.ReadFile(fallbackImage)
		if fallbackHashError == nil {
			hash := sha256.Sum256(image)
			fallbackHashValue = fmt.Sprintf("%x", hash)
		}
	})
	return fallbackHashValue, fallbackHashError
}

func iconETagMatches(header, hash string) bool {
	for _, tag := range strings.Split(header, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "*" || strings.Trim(strings.TrimPrefix(tag, "W/"), "\"") == hash {
			return true
		}
	}
	return false
}
