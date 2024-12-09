package main

import (
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
	"log"
	"net/http"
	"time"
)

func main() {

	machineServer := sse.New()
	machineServer.CreateStream("machine")

	syncTriggerServer := sse.New()
	syncStatusServer := sse.New()

	syncStatusServer.EventTTL = time.Second
	syncTriggerServer.EventTTL = time.Second

	m := mux.NewRouter()

	ms := MachinesStore{
		Store:  &[]string{},
		Server: machineServer,
	}

	syncTriggerHandler := SyncTriggerHandler{
		Server: syncTriggerServer,
		M:      ms,
	}
	syncStatusHandler := SyncStatusHandler{
		Server: syncStatusServer,
		M:      ms,
	}

	// sync trigger handlers
	m.Methods("POST").Path("/sync-trigger/{machine-id}/notify").HandlerFunc(syncTriggerHandler.SyncNotify)
	m.Methods("GET").Path("/sync-trigger").HandlerFunc(syncTriggerHandler.Status)

	// sync trigger handlers
	m.Methods("POST").Path("/sync-status/{machine-id}/notify").HandlerFunc(syncStatusHandler.SyncStatusNotify)
	m.Methods("GET").Path("/sync-status").HandlerFunc(syncStatusHandler.SyncStatus)

	m.Path("/machines").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		machineServer.ServeHTTP(w, r)
	})

	c := cors.Default().Handler(m)
	log.Fatal(http.ListenAndServe(":8080", c))
}
