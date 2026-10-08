package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("profiling listener: %v", err)
		}
	}()
}
