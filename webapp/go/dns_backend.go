package main

import (
	"os"
	"os/exec"
	"sync"
)

// Initialization cannot truncate users while a registration is publishing DNS.
var dnsLifecycle sync.RWMutex

func addDNSRecord(name string) *exec.Cmd {
	if os.Getenv("ISUCON13_POWERDNS_BACKEND") == "bind" {
		return exec.Command("python3", "../pdns/publish_zone.py", "add", name, powerDNSSubdomainAddress)
	}
	return exec.Command("pdnsutil", "add-record", "u.isucon.dev", name, "A", "0", powerDNSSubdomainAddress)
}
