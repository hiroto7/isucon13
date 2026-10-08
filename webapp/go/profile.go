package main

import (
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	http.HandleFunc("/debug/dnsstats", func(w http.ResponseWriter, r *http.Request) {
		authority := dnsAuthorityService.Load()
		if authority == nil {
			json.NewEncoder(w).Encode(map[string]string{"backend": "pdns"})
			return
		}
		names := 0
		if zone := authority.zone.Load(); zone != nil {
			names = len(zone.names)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backend": "go", "names": names, "positive": authority.positive.Load(),
			"negative": authority.negative.Load(), "dropped": authority.dropped.Load(),
		})
	})
	http.HandleFunc("/debug/dbstats", func(w http.ResponseWriter, r *http.Request) {
		if dbConn == nil {
			http.Error(w, "DB unavailable", 503)
			return
		}
		json.NewEncoder(w).Encode(dbConn.Stats())
	})
	go func() {
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("profiling listener: %v", err)
		}
	}()
}
