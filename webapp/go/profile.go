package main

import (
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	http.HandleFunc("/debug/write-batches", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]int64{"batches": writeBatches.Load(), "committed_rows": writeRows.Load(), "max_batch": writeMaxBatch.Load()})
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
